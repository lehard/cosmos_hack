package onec_test

import (
	"context"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	app "ant/internal/application/erp"
	appingest "ant/internal/application/ingest"
	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	"ant/internal/contracts/schemas"
	dom "ant/internal/domain/erp"
	"ant/internal/infrastructure/integration/erp/onec"
	"ant/internal/infrastructure/integration/erp/onec/stand"
	"ant/internal/infrastructure/integration/ingest/stands"
)

// Контрактные тесты адаптера 1С (AD-20, AD-35): эталонные сообщения
// contracts/integrations/erp/1c/examples — по схемам qc.v1; адаптер и stand
// говорят одним протоколом; классы ответа, сбои, идемпотентность, сверка
// $metadata, чтение OData.

func TestExamplesMatchSchemas(t *testing.T) {
	cases := map[string]string{
		"posting.moved_to_defect.json":      "qc.v1/posting.schema.json",
		"posting.warehouse_transfer.json":   "qc.v1/posting.schema.json",
		"posting.released.json":             "qc.v1/posting.schema.json",
		"posting.returned_to_supplier.json": "qc.v1/posting.schema.json",
		"inspection-result.json":            "qc.v1/inspection-result.schema.json",
		"receipt.json":                      "qc.v1/receipt.schema.json",
		"error.json":                        "qc.v1/error.schema.json",
		"about.json":                        "qc.v1/about.schema.json",
	}
	for ex, sch := range cases {
		b, err := fs.ReadFile(schemas.FS, "contracts/integrations/erp/1c/examples/"+ex)
		if err != nil {
			t.Fatal(err)
		}
		if err := onec.Validate("integrations/erp/1c/"+sch, b); err != nil {
			t.Errorf("%s не по схеме %s: %v", ex, sch, err)
		}
	}
}

func TestMetadataRoundTrip(t *testing.T) {
	m, err := onec.LoadManifest()
	if err != nil {
		t.Fatal(err)
	}
	if diff, err := onec.CompareMetadata(m, onec.BuildMetadata(m, nil)); err != nil || len(diff) != 0 {
		t.Fatalf("свой $metadata не совпал с манифестом: %v %v", diff, err)
	}
	diff, _ := onec.CompareMetadata(m, onec.BuildMetadata(m, map[string]string{"Статус": "СтатусЭтапа"}))
	if len(diff) == 0 || !strings.Contains(strings.Join(diff, ";"), "Статус") {
		t.Fatalf("переименованный реквизит не замечен: %v", diff)
	}
}

type env struct {
	srv    *httptest.Server
	stand  *stand.Stand
	client *onec.Client
	reg    *stands.Registry
}

func setup(t *testing.T) *env {
	t.Helper()
	st, err := stand.New(stand.Options{})
	if err != nil {
		t.Fatal(err)
	}
	reg := stands.NewRegistry(st)
	srv := httptest.NewServer(reg.Handler())
	t.Cleanup(srv.Close)
	c, err := onec.New(onec.Config{BaseURL: srv.URL + "/stand/1c/erp", Stand: true, Timeout: 3 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	return &env{srv: srv, stand: st, client: c, reg: reg}
}

func (e *env) fault(t *testing.T, f appingest.Fault) {
	t.Helper()
	if err := e.reg.SetFault(context.Background(), stand.Name, f); err != nil {
		t.Fatal(err)
	}
}

func (e *env) clear() { _ = e.reg.ClearFaults(context.Background(), stand.Name) }

func out(action dom.Action, item, key string, v int) app.Outgoing {
	it := ev.ItemID(item)
	r := ev.ErpPostingRequestedV1{BusinessKey: key, ExternalSystem: "onec", Action: ev.ErpPostingRequestedV1Action(action), MessageVersion: v, ItemID: &it}
	from, to := ev.ObjectID("WH-AC"), ev.ObjectID("WH-FG")
	r.FromWarehouseID, r.ToWarehouseID = &from, &to
	return app.Outgoing{MessageID: dom.MessageID(key, v), Version: v, Request: r, OccurredAt: time.Date(2026, 9, 23, 7, 30, 0, 0, time.UTC), Attempt: 1}
}

func TestCheckAndCorruptMetadata(t *testing.T) {
	e := setup(t)
	if _, err := e.client.Check(context.Background()); err != nil {
		t.Fatalf("сверка со stand-ом: %v", err)
	}
	e.fault(t, appingest.Fault{Kind: appingest.FaultCorrupt})
	_, err := e.client.Check(context.Background())
	if ce, ok := app.AsContract(err); !ok || !strings.Contains(ce.Detail, "Статус") {
		t.Fatalf("несовместимые метаданные — ContractError: %v", err)
	}
	e.clear()
	e.fault(t, appingest.Fault{Kind: appingest.FaultOffline})
	if _, err := e.client.Check(context.Background()); err == nil {
		t.Fatal("недоступность")
	} else if _, ok := app.AsTransport(err); !ok {
		t.Fatalf("недоступность — транспорт: %v", err)
	}
}

// Квитанция, повтор с тем же номером — та же квитанция, один документ (AD-7).
func TestPostIdempotent(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	m := out(dom.Release, "ENT01:F-001", "ENT01:F-001/release/final.erp_release", 1)
	r1, err := e.client.Post(ctx, m)
	if err != nil || r1.Outcome != ev.ErpPostingRespondedV1OutcomeAccepted || r1.Receipt == "" {
		t.Fatalf("первая отправка: %+v %v", r1, err)
	}
	r2, err := e.client.Post(ctx, m)
	if err != nil || r2.Outcome != ev.ErpPostingRespondedV1OutcomeDuplicate || r2.Receipt != r1.Receipt || r2.DocumentRef != r1.DocumentRef {
		t.Fatalf("повтор: %+v %v", r2, err)
	}
	snap, _ := e.stand.Snapshot(ctx)
	if len(snap.Documents) != 1 || len(snap.Messages) != 1 || snap.Messages[0].Deliveries != 2 {
		t.Fatalf("двойной учёт: документов %d, сообщений %d", len(snap.Documents), len(snap.Messages))
	}
	if snap.Documents[0].Type != "Document_ПередачаПродукцииИзПроизводства" || snap.Documents[0].WarehouseTo != "WH-FG" {
		t.Fatalf("документ выпуска: %+v", snap.Documents[0])
	}
	// Страница «глазами 1С»: журнал обмена, документ, склад изделия, этап производства.
	resp, err := http.Get(e.srv.URL + "/stand/1c/")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	for _, want := range []string{"Журнал обмена", r1.Receipt, "Передача продукции из производства", "WH-FG", "ЭП00-000917", "повтор — та же квитанция"} {
		if !strings.Contains(string(body), want) {
			t.Errorf("на странице stand-а нет «%s»", want)
		}
	}
}

// Сбой по образцу: 503 на выпуск F-001 — транспорт; прочие проходят; 422 на
// возврат поставщику LOT-R-117 — ошибка данных с кодом договора.
func TestMatchFaults(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	e.fault(t, appingest.Fault{Kind: appingest.FaultError, Param: 503, Match: "release_good:F-001", Detail: "503 Service Unavailable"})
	_, err := e.client.Post(ctx, out(dom.Release, "ENT01:R7-F-001", "ENT01:R7-F-001/release/final.erp_release", 1))
	if _, ok := app.AsTransport(err); !ok {
		t.Fatalf("503 по образцу — транспорт: %v", err)
	}
	if r, err := e.client.Post(ctx, out(dom.Release, "ENT01:F-002", "ENT01:F-002/release/final.erp_release", 1)); err != nil || r.Outcome != "accepted" {
		t.Fatalf("другое изделие проходит: %+v %v", r, err)
	}
	e.clear()
	if r, err := e.client.Post(ctx, out(dom.Release, "ENT01:R7-F-001", "ENT01:R7-F-001/release/final.erp_release", 1)); err != nil || r.Outcome != "accepted" {
		t.Fatalf("после снятия сбоя — квитанция: %+v %v", r, err)
	}

	lot := ev.ObjectID("LOT-R-117")
	cb := "поры в теле колец"
	ret := app.Outgoing{MessageID: dom.MessageID("LOT-R-117/return_to_supplier/ZT-1", 1), Version: 1, OccurredAt: time.Now(),
		Request: ev.ErpPostingRequestedV1{BusinessKey: "LOT-R-117/return_to_supplier/ZT-1", ExternalSystem: "onec", Action: "return_to_supplier",
			MessageVersion: 1, LotID: &lot, ClaimBasis: &cb}}
	e.fault(t, appingest.Fault{Kind: appingest.FaultError, Param: 422, Match: "return_to_supplier:LOT-R-117", Detail: "422 не найден договор с контрагентом ctr-0b19-0003"})
	r, err := e.client.Post(ctx, ret)
	if err != nil || r.Outcome != ev.ErpPostingRespondedV1OutcomeRejected || r.ErrorCode != "erp.contract_not_found" || r.HTTPStatus != 422 {
		t.Fatalf("422 по образцу: %+v %v", r, err)
	}
	e.clear()
	if r, err := e.client.Post(ctx, ret); err != nil || r.Outcome != "accepted" {
		t.Fatalf("переотправка после исправления: %+v %v", r, err)
	}
}

// Дубль: документ проведён, ответ потерян — повтор получает ту же квитанцию,
// второго документа нет.
func TestDuplicateLostResponse(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	e.fault(t, appingest.Fault{Kind: appingest.FaultDuplicate})
	m := out(dom.WarehouseTransfer, "ENT01:F-003", "ENT01:F-003/warehouse_transfer/welding.erp_transfer_in", 1)
	if _, err := e.client.Post(ctx, m); err == nil {
		t.Fatal("ответ потерян — должна быть транспортная ошибка")
	} else if _, ok := app.AsTransport(err); !ok {
		t.Fatalf("потерянный ответ — транспорт: %v", err)
	}
	r, err := e.client.Post(ctx, m)
	if err != nil || r.Outcome != ev.ErpPostingRespondedV1OutcomeDuplicate {
		t.Fatalf("повтор после потери ответа: %+v %v", r, err)
	}
	snap, _ := e.stand.Snapshot(ctx)
	if len(snap.Documents) != 1 {
		t.Fatalf("документов %d — дубль задвоил учёт", len(snap.Documents))
	}
}

// Ошибка интеграции ловится до отправки (FR-111): сообщение не по схеме.
func TestLocalContractError(t *testing.T) {
	e := setup(t)
	m := out(dom.Release, "ENT01:F-004", "ENT01:F-004/release/final.erp_release", 1)
	bad := ev.ObjectID("склад с пробелом")
	m.Request.ToWarehouseID = &bad
	_, err := e.client.Post(context.Background(), m)
	if ce, ok := app.AsContract(err); !ok || !ce.Local {
		t.Fatalf("не по схеме — ContractError{Local}: %v", err)
	}
	snap, _ := e.stand.Snapshot(context.Background())
	if len(snap.Messages) != 0 {
		t.Fatal("в 1С ничего не должно уйти")
	}
}

func TestOfflineAndStatus(t *testing.T) {
	e := setup(t)
	e.fault(t, appingest.Fault{Kind: appingest.FaultOffline})
	_, err := e.client.Post(context.Background(), out(dom.Release, "ENT01:F-005", "ENT01:F-005/release/x", 1))
	if _, ok := app.AsTransport(err); !ok {
		t.Fatalf("недоступность — транспорт: %v", err)
	}
	e.clear()
	resp, err := http.Get(e.srv.URL + "/stand/1c/")
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("страница stand-а: %v %v", resp, err)
	}
	_ = resp.Body.Close()
}

// Чтение OData: номенклатура, задание, партии и соответствия ID; дубль строк
// не даёт второго факта; новый этап со страницы stand-а — новое задание.
func TestPull(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	facts, err := e.client.Pull(ctx)
	if err != nil {
		t.Fatal(err)
	}
	count := func(fs []app.Inbound) map[catalog.Type]int {
		n := map[catalog.Type]int{}
		for _, f := range fs {
			n[f.Type]++
		}
		return n
	}
	n := count(facts)
	if n[catalog.ErpNomenclatureSynced] != 1 || n[catalog.ErpOrderReceived] != 1 || n[catalog.ErpLotReceived] != 7 {
		t.Fatalf("факты: %v", n)
	}
	var order ev.ErpOrderReceivedV1
	var ringLot bool
	for _, f := range facts {
		switch d := f.Data.(type) {
		case ev.ErpOrderReceivedV1:
			order = d
		case ev.ErpLotReceivedV1:
			ringLot = ringLot || (d.LotID == "LOT-R-117" && d.SupplierID == "SUP-3" && d.Quantity == 10)
		}
	}
	if order.OrderID != "ORD-0917" || order.ItemTypeID != "FL-100.00.000" || order.Quantity != 40 || order.DueDate == nil || *order.DueDate != "2026-10-09" {
		t.Fatalf("задание: %+v", order)
	}
	if !ringLot {
		t.Fatal("партия колец LOT-R-117")
	}
	e.fault(t, appingest.Fault{Kind: appingest.FaultDuplicate})
	again, err := e.client.Pull(ctx)
	if err != nil || len(again) != len(facts) || again[0].EventID != facts[0].EventID {
		t.Fatalf("дубль строк OData: %d против %d, %v", len(again), len(facts), err)
	}
	e.clear()
	resp, err := http.PostForm(e.srv.URL+"/stand/1c/stages", url.Values{"quantity": {"6"}, "due": {"2026-10-20"}})
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	more, _ := e.client.Pull(ctx)
	if count(more)[catalog.ErpOrderReceived] != 2 {
		t.Fatalf("новое задание со страницы stand-а: %v", count(more))
	}
}

func TestIDRules(t *testing.T) {
	for in, want := range map[string]string{"ЭП00-000917": "ORD-0917", "ЗП-12345": "ORD-12345", "ЭП00-000001": "ORD-0001"} {
		if got := onec.OrderID(in); got != want {
			t.Errorf("OrderID(%s) = %s, ожидалось %s", in, got, want)
		}
	}
	if got := onec.ItemTypeID("ФЛ-100.00.000 СБ"); got != "FL-100.00.000" {
		t.Errorf("ItemTypeID = %s", got)
	}
	if got := onec.LotID("П-117"); got != "LOT-P-117" {
		t.Errorf("LotID = %s", got)
	}
}
