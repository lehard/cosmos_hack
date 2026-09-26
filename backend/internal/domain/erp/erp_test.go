package erp

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
)

// Цеха фланца (normative/process/flange-process.bpmn): дорожки в порядке laneSet.
var flange = Env{Topology: Topology{Lanes: []Lane{
	{ID: "Lane_SK", Workshop: "WS-SK", Warehouse: "WH-SK", Steps: []string{"incoming.erp_accept_blank"}},
	{ID: "Lane_MC", Workshop: "WS-MC", Warehouse: "WH-MC", Steps: []string{"machining.op"}},
	{ID: "Lane_WC", Workshop: "WS-WC", Warehouse: "WH-WC", Steps: []string{"welding.erp_transfer_in", "nc.erp_scrap_rework", "welding.erp_return_from_defect"}},
	{ID: "Lane_AC", Workshop: "WS-AC", Warehouse: "WH-AC", Steps: []string{"assembly.erp_transfer_in"}},
	{ID: "Lane_QA", Workshop: "WS-QA", Warehouse: "WH-FG", Steps: []string{"final.erp_release"}},
}}}

const item = "ENT01:F-001"

var seq int64

func rec(t catalog.Type, item string, data any) kernel.Record {
	seq++
	b, _ := json.Marshal(data)
	return kernel.Record{Seq: seq, EventID: kernel.UUIDv5("6ba7b810-9dad-11d1-80b4-00c04fd430c8", string(rune(seq))), Type: t,
		ItemID: item, OccurredAt: time.Date(2026, 9, 23, 7, 0, int(seq), 0, time.UTC), Data: b}
}

func thrown(step string, a Action) kernel.Record {
	return rec(catalog.OperationMessageThrown, item, ev.OperationMessageThrownV1{StepKey: ev.StepKey(step), MessageRef: "Msg", ErpAction: ev.OperationMessageThrownV1ErpAction(a), ClosingBasis: []ev.UUID{}})
}

// plan — шаг с проверкой числа сообщений.
func plan(t *testing.T, b Books, r kernel.Record, slot string, want int) ([]Draft, Books) {
	t.Helper()
	ds, nb, err := Plan(flange, b, Trigger{Record: r, Slot: slot})
	if err != nil {
		t.Fatal(err)
	}
	if len(ds) != want {
		t.Fatalf("%s: сообщений %d, ожидалось %d: %+v", r.Type, len(ds), want, ds)
	}
	return ds, nb
}

func wh(p *ev.ObjectID) string {
	if p == nil {
		return ""
	}
	return string(*p)
}

// Маршрут фланца: принято в работу → смена склада МЦ → СЦ → брак (переделка),
// повторное решение без нового движения → возврат из брака (одно сообщение на
// цикл от двух триггеров) → смена склада СЦ → СИЦ → выпуск после переделки.
func TestFlangeRoute(t *testing.T) {
	b := Books{Subject: item}
	_, b = plan(t, b, rec(catalog.ItemItemRegistered, item, ev.ItemItemRegisteredV1{ItemID: item, ItemTypeID: "FL-100", OrderID: oid("ORD-0917")}), "", 0)

	ds, b := plan(t, b, thrown("incoming.erp_accept_blank", AcceptIntoWork), "s1", 1)
	if d := ds[0]; wh(d.Data.FromWarehouseID) != "WH-SK" || wh(d.Data.ToWarehouseID) != "WH-MC" || d.Key != item+"/accept_into_work/incoming.erp_accept_blank" || wh(d.Data.OrderID) != "ORD-0917" {
		t.Fatalf("принято в работу: %+v", d)
	}
	ds, b = plan(t, b, thrown("welding.erp_transfer_in", WarehouseTransfer), "s2", 1)
	if d := ds[0]; wh(d.Data.FromWarehouseID) != "WH-MC" || wh(d.Data.ToWarehouseID) != "WH-WC" {
		t.Fatalf("смена склада МЦ → СЦ: %+v", d.Data)
	}
	_, b = plan(t, b, rec(catalog.DecisionDispositionSet, item, ev.DecisionDispositionSetV1{NcID: "NC-01", Disposition: "rework", Reason: ev.Reason{Text: "подрез"}}), "", 0)
	ds, b = plan(t, b, thrown("nc.erp_scrap_rework", ScrapRework), "s3", 1)
	if d := ds[0]; d.Key != item+"/scrap_transfer_rework/defect.1" || wh(d.Data.FromWarehouseID) != "WH-WC" || len(d.Data.NcIds) != 1 {
		t.Fatalf("перевод в брак: %+v", d)
	}
	// Повторное решение о переделке, пока изделие в браке, — нового движения нет.
	plan(t, b, thrown("nc.erp_scrap_rework", ScrapRework), "s3b", 0)
	ds, b = plan(t, b, rec(catalog.DecisionDispositionVerified, item, ev.DecisionDispositionVerifiedV1{NcID: "NC-01", RecheckEventIds: []ev.UUID{}}), "", 1)
	if d := ds[0]; d.Key != item+"/return_from_defect/defect.1" || wh(d.Data.ToWarehouseID) != "WH-WC" {
		t.Fatalf("возврат из брака: %+v", d)
	}
	// Событие-сообщение процесса «возврат из брака» того же цикла — ничего нового.
	plan(t, b, thrown("welding.erp_return_from_defect", ReturnFromDefect), "s4", 0)
	ds, b = plan(t, b, thrown("assembly.erp_transfer_in", WarehouseTransfer), "s5", 1)
	if d := ds[0]; wh(d.Data.FromWarehouseID) != "WH-WC" || wh(d.Data.ToWarehouseID) != "WH-AC" {
		t.Fatalf("смена склада СЦ → СИЦ: %+v", d.Data)
	}
	ds, _ = plan(t, b, thrown("final.erp_release", Release), "s6", 1)
	if d := ds[0]; wh(d.Data.FromWarehouseID) != "WH-AC" || wh(d.Data.ToWarehouseID) != "WH-FG" || d.Data.AfterRework == nil || !*d.Data.AfterRework {
		t.Fatalf("выпуск после переделки: %+v", d.Data)
	}
}

// Новая версия реакции процесса (тот же слот) — тот же бизнес-ключ; повторный
// проход шага (другой слот) — новый ключ.
func TestOccurrenceKeys(t *testing.T) {
	b := Books{Subject: item}
	d1, b := plan(t, b, thrown("welding.erp_transfer_in", WarehouseTransfer), "slot-a", 1)
	d2, b := plan(t, b, thrown("welding.erp_transfer_in", WarehouseTransfer), "slot-a", 1)
	d3, _ := plan(t, b, thrown("welding.erp_transfer_in", WarehouseTransfer), "slot-b", 1)
	if d1[0].Key != d2[0].Key || d3[0].Key == d1[0].Key || !strings.HasSuffix(d3[0].Key, ".r2") {
		t.Fatalf("ключи: %s %s %s", d1[0].Key, d2[0].Key, d3[0].Key)
	}
}

// Версии по бизнес-ключу (AD-7): то же содержимое — ничего; другое — версия 2
// с supersedes; номер сообщения — от ключа и версии.
func TestVersions(t *testing.T) {
	b := Books{Subject: item}
	ds, _ := plan(t, b, thrown("final.erp_release", Release), "s", 1)
	s1, emit := Version(nil, ds[0])
	if !emit || s1.Version != 1 {
		t.Fatal("первая версия")
	}
	again := ds[0]
	again.Data.BasisEventIds = []ev.UUID{"0192ad10-4c7e-7f21-9b3a-5c6d7e8f9a0b"}
	if _, emit := Version(&s1, again); emit {
		t.Fatal("то же учётное содержимое не даёт новой версии")
	}
	changed := ds[0]
	c := ev.ObjectID("CONC-7")
	changed.Data.ConcessionID = &c
	s2, emit := Version(&s1, changed)
	if !emit || s2.Version != 2 {
		t.Fatal("другое содержимое — версия 2")
	}
	rx, err := Reaction(changed, s2)
	if err != nil {
		t.Fatal(err)
	}
	d := rx.Data.(ev.ErpPostingRequestedV1)
	if d.MessageVersion != 2 || d.MessageID == nil || string(*d.MessageID) != MessageID(changed.Key, 2) || rx.Slot.Subject != Stream(changed.Key) {
		t.Fatalf("реакция: %+v", rx)
	}
	if MessageID(changed.Key, 1) == MessageID(changed.Key, 2) {
		t.Fatal("номер сообщения зависит от версии")
	}
}

// Партия: брак на входном контроле → результат контроля и возврат остатка
// поставщику с основанием претензии.
func TestLotReturn(t *testing.T) {
	b := Books{Subject: "LOT-R-117"}
	_, b = plan(t, b, rec(catalog.ErpLotReceived, "", ev.ErpLotReceivedV1{LotID: "LOT-R-117", ExternalSystem: "onec", ExternalNumber: "П-117", SupplierID: "SUP-3", ItemTypeID: "RING", Quantity: 10}), "", 0)
	_, b = plan(t, b, rec(catalog.GenealogyLotIssued, "", ev.GenealogyLotIssuedV1{LotID: "LOT-R-117", Quantity: 3, IssuedBy: "WH-01"}), "", 0)
	ds, _ := plan(t, b, rec(catalog.DecisionLotResolved, "", ev.DecisionLotResolvedV1{LotID: "LOT-R-117", Resolution: "reject", MethodEventIds: []ev.UUID{},
		Reason: &ev.Reason{Text: "Входной брак партии П-117"}}), "", 2)
	ret := ds[1]
	if ret.Action != ReturnToSupplier || ret.Key != "LOT-R-117/return_to_supplier/ZT-1" || ret.Data.Quantity == nil || *ret.Data.Quantity != 7 ||
		ret.Data.ClaimBasis == nil || wh(ret.Data.LotID) != "LOT-R-117" || wh(ret.Data.FromWarehouseID) != "WH-SK" {
		t.Fatalf("возврат поставщику: %+v", ret.Data)
	}
	if ds[0].Action != InspectionResult || ds[0].Data.Resolution == nil || *ds[0].Data.Resolution != "reject" {
		t.Fatalf("результат контроля партии: %+v", ds[0].Data)
	}
}

// Результат контроля по решению на закрывающей точке; исправление решения —
// тот же бизнес-ключ.
func TestInspectionResult(t *testing.T) {
	b := Books{Subject: item}
	r := rec(catalog.DecisionPresentationResolved, item, ev.DecisionPresentationResolvedV1{StepKey: "final.zt6", ClosingPoint: "ZT-6", Resolution: "accept_with_concession",
		PresentationNo: 1, ConcessionID: oid("CONC-1"), MethodEventIds: []ev.UUID{}})
	ds, b := plan(t, b, r, "", 1)
	if ds[0].Key != item+"/inspection_result/ZT-6.p1" || b.ConcessionID != "CONC-1" {
		t.Fatalf("результат контроля: %s %+v", ds[0].Key, b)
	}
	fix := rec(catalog.DecisionPresentationResolved, item, ev.DecisionPresentationResolvedV1{StepKey: "final.zt6", ClosingPoint: "ZT-6", Resolution: "reject",
		PresentationNo: 2, MethodEventIds: []ev.UUID{}})
	fix.Corrects = r.EventID
	ds2, _ := plan(t, b, fix, "", 1)
	if ds2[0].Key != ds[0].Key {
		t.Fatalf("исправление решения — тот же ключ: %s", ds2[0].Key)
	}
}

// Д-81: отзыв приёмки — новая версия результата контроля той же точки
// («мало данных»); «оставить в силе» учёт не меняет.
func TestReviewedPresentation(t *testing.T) {
	b := Books{Subject: item}
	r := rec(catalog.DecisionPresentationResolved, item, ev.DecisionPresentationResolvedV1{StepKey: "welding.zt3_acceptance", ClosingPoint: "ZT-3", Resolution: "accept",
		PresentationNo: 1, MethodEventIds: []ev.UUID{}})
	ds, b := plan(t, b, r, "", 1)
	up := rec(catalog.DecisionPresentationReviewed, item, ev.DecisionPresentationReviewedV1{ReviewedEventID: ev.UUID(r.EventID), StepKey: "welding.zt3_acceptance",
		ClosingPoint: "ZT-3", PresentationNo: 1, Outcome: ev.DecisionPresentationReviewedV1OutcomeUpheld, NewFactIds: []ev.UUID{}, Reason: ev.Reason{Text: "в допуске"}})
	plan(t, b, up, "", 0)
	rv := rec(catalog.DecisionPresentationReviewed, item, ev.DecisionPresentationReviewedV1{ReviewedEventID: ev.UUID(r.EventID), StepKey: "welding.zt3_acceptance",
		ClosingPoint: "ZT-3", PresentationNo: 1, Outcome: ev.DecisionPresentationReviewedV1OutcomeRevoked, NewFactIds: []ev.UUID{}, Reason: ev.Reason{Text: "ток вне уставки"}})
	ds2, _ := plan(t, b, rv, "", 1)
	if ds2[0].Key != ds[0].Key || ds2[0].Data.Resolution == nil || *ds2[0].Data.Resolution != ev.ErpPostingRequestedV1ResolutionInsufficientData {
		t.Fatalf("отзыв — исправление того же сообщения: %s %+v", ds2[0].Key, ds2[0].Data)
	}
}

// Ось «учёт в 1С» меняется только квитанцией; журнал обмена — по записям.
func TestViews(t *testing.T) {
	b := Books{Subject: item}
	ds, _ := plan(t, b, thrown("final.erp_release", Release), "s", 1)
	s1, _ := Version(nil, ds[0])
	rx, _ := Reaction(ds[0], s1)
	req := rec(catalog.ErpPostingRequested, "", rx.Data)
	var v View
	var a ItemAccounting
	var err error
	if v, err = v.Apply(req); err != nil || v.Status != StatusQueued {
		t.Fatal(v, err)
	}
	a, _ = a.Apply(req)
	if a.State != "not_sent" || len(a.Pending) != 1 {
		t.Fatalf("до квитанции ось не меняется: %+v", a)
	}
	resp := rec(catalog.ErpPostingResponded, "", ev.ErpPostingRespondedV1{BusinessKey: ds[0].Key, RequestEventID: ev.UUID(req.EventID), Outcome: "accepted"})
	v, _ = v.Apply(resp)
	a, _ = a.Apply(resp)
	if v.Status != StatusAcknowledged || v.Accounting != "released" || a.State != "released" || a.Warehouse != "WH-FG" {
		t.Fatalf("после квитанции: %+v %+v", v, a)
	}
}

func TestBusinessKeyLength(t *testing.T) {
	long := "ENT01:" + strings.Repeat("x", 110)
	k := BusinessKey(long, WarehouseTransfer, "welding.erp_transfer_in")
	if len(k) > 128 || !strings.Contains(k, "/warehouse_transfer/") {
		t.Fatalf("ключ %d: %s", len(k), k)
	}
	if BusinessKey(long, WarehouseTransfer, "welding.erp_transfer_in") != k {
		t.Fatal("ключ детерминирован")
	}
}
