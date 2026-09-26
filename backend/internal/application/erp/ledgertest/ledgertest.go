// Пакет ledgertest — общий контрактный тест порта учёта application/erp.Ledger
// для всех адаптеров (AD-35: «у каждого порта ключ конфигурации и общий
// контрактный тест для всех адаптеров»). Один и тот же внутренний сигнал —
// все учётные действия порта (Signals) — должен одинаково приниматься 1С,
// Галактикой и любым следующим получателем без изменения ядра (FR-92, FR-114).
//
// Слой: application (тестовая опора; без драйверов и сети). В сборку ролей
// не входит; вызывается из контрактных тестов адаптеров
// infrastructure/integration/erp/‹система›.
//
// Владелец: эпик 31.
package ledgertest

import (
	"context"
	"slices"
	"testing"
	"time"

	app "ant/internal/application/erp"
	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	dom "ant/internal/domain/erp"
)

// T0 — время учётных действий эталонного сигнала.
var T0 = time.Date(2026, 9, 25, 10, 42, 17, 305_000_000, time.UTC)

func oid(s string) *ev.ObjectID { x := ev.ObjectID(s); return &x }
func sp(s string) *string       { return &s }

// Signal — исходящее сообщение порта учёта с бизнес-ключом ‹субъект›/‹действие›/‹точка›.
func Signal(action dom.Action, item, lot, point string, fill func(*ev.ErpPostingRequestedV1)) app.Outgoing {
	subject := item
	if lot != "" {
		subject = lot
	}
	key := dom.BusinessKey(subject, action, point)
	r := ev.ErpPostingRequestedV1{BusinessKey: key, ExternalSystem: ev.ErpPostingRequestedV1ExternalSystemOnec,
		Action: ev.ErpPostingRequestedV1Action(action), MessageVersion: 1, ClosingPoint: sp(point)}
	if item != "" {
		it := ev.ItemID(item)
		r.ItemID = &it
	}
	if lot != "" {
		r.LotID = oid(lot)
	}
	if fill != nil {
		fill(&r)
	}
	mid := ev.UUID(dom.MessageID(key, 1))
	r.MessageID = &mid
	return app.Outgoing{MessageID: string(mid), Version: 1, Request: r, OccurredAt: T0, Attempt: 1}
}

// Signals — эталонный внутренний сигнал: по сообщению на каждое учётное
// действие порта (наш язык, AD-18). Адаптер переводит его в формат своей
// системы; система в самом сигнале (external_system) адаптер не читает.
func Signals() []app.Outgoing {
	return []app.Outgoing{
		Signal(dom.InspectionResult, "", "LOT-B-0915", "ZT-1", func(r *ev.ErpPostingRequestedV1) {
			res := ev.ErpPostingRequestedV1ResolutionReject
			r.Resolution, r.Quantity, r.OrderID = &res, ip(12), oid("ORD-0812")
			r.NcIds = []ev.ObjectID{"NC-2026-0031", "NC-2026-0032"}
			n := 1
			r.PresentationNo = &n
		}),
		Signal(dom.AcceptIntoWork, "ENT01:F-001", "", "incoming.erp_accept_blank", func(r *ev.ErpPostingRequestedV1) {
			r.FromWarehouseID, r.ToWarehouseID, r.OrderID = oid("WH-SK"), oid("WH-MC"), oid("ORD-0812")
		}),
		Signal(dom.WarehouseTransfer, "ENT01:F-001", "", "welding.erp_transfer_in", func(r *ev.ErpPostingRequestedV1) {
			r.FromWarehouseID, r.ToWarehouseID = oid("WH-MC"), oid("WH-WC")
		}),
		Signal(dom.ScrapRework, "ENT01:F-002", "", "defect.1", func(r *ev.ErpPostingRequestedV1) {
			r.FromWarehouseID, r.NcIds = oid("WH-WC"), []ev.ObjectID{"NC-2026-0040"}
		}),
		Signal(dom.ScrapWriteoff, "ENT01:F-003", "", "defect.1", func(r *ev.ErpPostingRequestedV1) {
			r.FromWarehouseID, r.NcIds = oid("WH-AC"), []ev.ObjectID{"NC-2026-0041"}
		}),
		Signal(dom.ScrapReprocess, "ENT01:F-004", "", "defect.1", func(r *ev.ErpPostingRequestedV1) {
			r.FromWarehouseID, r.NcIds = oid("WH-AC"), []ev.ObjectID{"NC-2026-0042"}
		}),
		Signal(dom.ReturnFromDefect, "ENT01:F-002", "", "defect.1", func(r *ev.ErpPostingRequestedV1) {
			r.ToWarehouseID, r.NcIds = oid("WH-WC"), []ev.ObjectID{"NC-2026-0040"}
		}),
		Signal(dom.ReturnToSupplier, "", "LOT-R-117", "ZT-1", func(r *ev.ErpPostingRequestedV1) {
			r.FromWarehouseID, r.Quantity, r.ClaimBasis = oid("WH-SK"), ip(10), sp("поры в теле колец")
		}),
		Signal(dom.Release, "ENT01:F-002", "", "final.erp_release", func(r *ev.ErpPostingRequestedV1) {
			t := true
			r.FromWarehouseID, r.ToWarehouseID, r.AfterRework, r.ConcessionID = oid("WH-AC"), oid("WH-FG"), &t, oid("RD-2026-0007")
		}),
	}
}

func ip(n int) *int { return &n }

// Inbound — типы входящих фактов порта учёта.
var Inbound = []catalog.Type{catalog.ErpOrderReceived, catalog.ErpNomenclatureSynced, catalog.ErpLotReceived, catalog.ReferenceExternalIdMapped}

// Options — что проверить сверх общего.
type Options struct {
	// System — ожидаемая система канала (onec, galaktika).
	System string
	// Envelope — проверка входящего факта схемами приёма (конверт шлюза,
	// application/erp.Envelope → ingest.ValidateEnvelope); nil — без неё.
	Envelope func(f app.Inbound) error
	// WantInbound — сколько входящих фактов ожидается (0 — хотя бы один).
	WantInbound int
	// Settle — асинхронный канал (каталог обмена): ответная сторона
	// обрабатывает отправленное; после транспортной ошибки «квитанции ещё
	// нет» тест вызывает Settle и повторяет тем же номером, как роль outbox.
	Settle func(t *testing.T)
}

// Run — общий контрактный тест порта учёта на готовом адаптере с ответной
// стороной (stand или эталонный фасад): сверка контракта, все учётные
// действия эталонного сигнала — квитанция, повтор того же номера — та же
// квитанция без второго документа (AD-7), входящие — типы порта учёта,
// детерминированные event_id при повторном чтении.
func Run(t *testing.T, l app.Ledger, o Options) {
	t.Helper()
	ctx := context.Background()
	info := l.Info()
	if info.System != o.System || info.ContractVersion == "" || info.Endpoint == "" {
		t.Fatalf("канал: %+v", info)
	}
	if _, err := l.Check(ctx); err != nil {
		t.Fatalf("сверка ответной стороны: %v", err)
	}
	for _, m := range Signals() {
		r1, err := l.Post(ctx, m)
		if _, transport := app.AsTransport(err); transport && o.Settle != nil {
			o.Settle(t)
			r1, err = l.Post(ctx, m)
		}
		if err != nil || r1.Outcome != ev.ErpPostingRespondedV1OutcomeAccepted || r1.Receipt == "" {
			t.Fatalf("%s: квитанция: %+v %v", m.Request.Action, r1, err)
		}
		r2, err := l.Post(ctx, m)
		if err != nil || r2.Receipt != r1.Receipt || r2.DocumentRef != r1.DocumentRef ||
			(r2.Outcome != ev.ErpPostingRespondedV1OutcomeDuplicate && r2.Outcome != ev.ErpPostingRespondedV1OutcomeAccepted) {
			t.Fatalf("%s: повтор с тем же номером — та же квитанция: %+v / %+v %v", m.Request.Action, r1, r2, err)
		}
	}
	a, err := l.Pull(ctx)
	if err != nil {
		t.Fatalf("входящие: %v", err)
	}
	b, err := l.Pull(ctx)
	if err != nil {
		t.Fatalf("входящие повторно: %v", err)
	}
	if len(a) == 0 || (o.WantInbound > 0 && len(a) != o.WantInbound) {
		t.Fatalf("входящих фактов %d, ожидалось %d", len(a), o.WantInbound)
	}
	ids := func(xs []app.Inbound) []string {
		var out []string
		for _, x := range xs {
			out = append(out, x.EventID)
		}
		slices.Sort(out)
		return out
	}
	if !slices.Equal(ids(a), ids(b)) {
		t.Fatal("повторное чтение входящих дало другие event_id (AD-7)")
	}
	for _, f := range a {
		if !slices.Contains(Inbound, f.Type) || f.EventID == "" || f.OccurredAt.IsZero() {
			t.Fatalf("входящий факт не порта учёта: %+v", f)
		}
		if o.Envelope != nil {
			if err := o.Envelope(f); err != nil {
				t.Fatalf("%s не проходит схемы приёма: %v", f.Type, err)
			}
		}
	}
}
