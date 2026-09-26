package nonconformity_test

import (
	"bytes"
	"context"
	"slices"
	"testing"
	"time"

	app "ant/internal/application/nonconformity"
	"ant/internal/application/nonconformity/nctest"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
)

func codeOf(err error) errcodes.Code {
	if e, ok := platform.AsError(err); ok {
		return e.Code
	}
	return ""
}

func hdr(basis int64) platform.CommandHeader {
	return platform.CommandHeader{CommandID: nctest.ID(), BasisSeq: basis}
}

func reason(s string) app.NCReason { return app.NCReason{Text: s} }

// signalNC — изделие с признаком дефекта после сварки → черновик несоответствия.
func signalNC(t *testing.T, w *nctest.World, svc *app.Service, item string) (string, app.NCCard) {
	t.Helper()
	w.Add(nctest.Run(item, nctest.T0, "RUN-"+item, "op-7"), nctest.Inspection(item, nctest.T0.Add(30*time.Minute), "Z-1"))
	w.Settle()
	l, err := svc.List(context.Background(), app.NCFilter{ItemID: item}, platform.Moment{}, platform.Page{})
	if err != nil || len(l.Items) != 1 {
		t.Fatalf("несоответствия %s: %+v %v", item, l, err)
	}
	c, err := svc.Card(context.Background(), l.Items[0].NCID, platform.Moment{})
	if err != nil {
		t.Fatal(err)
	}
	return c.NCID, c
}

// FR-51, FR-52: отклонение сигнала не меняет исходный сигнал — решение
// отдельной записью; без причины — отказ; карточка держит слои раздельно.
func TestRejectSignalIsSeparateRecord(t *testing.T) {
	w := nctest.NewWorld(t)
	svc := w.Service(app.DemoRoutes{})
	ncID, c := signalNC(t, w, svc, "FL:0001")
	if c.Status != "draft" || len(c.Evidence.Signals) != 1 || len(c.SystemAnalysis.Versions) == 0 {
		t.Fatalf("черновик: %+v", c)
	}
	if !slices.Contains(c.ToDecide.Decisions, "nonconformity.signal.reject") || !slices.Contains(c.ToDecide.Decisions, "nonconformity.nonconformity.confirm") {
		t.Fatalf("решения: %v", c.ToDecide.Decisions)
	}
	q, err := svc.Queue(context.Background(), app.QueueFilter{}, platform.Moment{}, platform.Page{})
	if err != nil || len(q.Items) != 1 || q.Items[0].Kind != "signal" || q.Items[0].NCID == nil || *q.Items[0].NCID != ncID {
		t.Fatalf("очередь: %+v %v", q, err)
	}
	before := slices.Clone(w.J.Entries())
	sig := c.Evidence.Signals[0].SignalID
	ctx := nctest.As("qc-1", "quality_inspector")
	_, err = svc.RejectSignal(ctx, "FL:0001", app.RejectSignal{CommandHeader: hdr(c.BasisSeq), SignalIDs: []string{sig}, Reason: reason("   ")})
	if codeOf(err) != errcodes.NonconformityRejectReasonRequired {
		t.Fatalf("без причины: %v", err)
	}
	r, err := svc.RejectSignal(ctx, "FL:0001", app.RejectSignal{CommandHeader: hdr(c.BasisSeq), SignalIDs: []string{sig}, Reason: reason("Блик на шве — не дефект")})
	if err != nil || r.Seq == 0 {
		t.Fatalf("отклонение: %+v %v", r, err)
	}
	w.Settle()
	after := w.J.Entries()
	for i, e := range before {
		ea, eb := after[i], e
		env1, _ := w.J.Open(context.Background(), eb)
		env2, _ := w.J.Open(context.Background(), ea)
		if ea.EventID != eb.EventID || !bytes.Equal(env1.Raw, env2.Raw) {
			t.Fatalf("запись seq %d изменилась", e.Seq)
		}
	}
	var rej []string
	for _, e := range after {
		if e.EventType == string(catalog.DecisionSignalRejected) {
			rej = append(rej, e.EventID)
		}
	}
	if len(rej) != 1 || rej[0] != r.EventIDs[0] {
		t.Fatalf("решение — отдельная запись: %v", rej)
	}
	c2, err := svc.Card(context.Background(), ncID, platform.Moment{})
	if err != nil {
		t.Fatal(err)
	}
	if c2.Status != "closed" || c2.Resolution == nil || *c2.Resolution != "signal_rejected" {
		t.Fatalf("карточка после отклонения: %s %v", c2.Status, c2.Resolution)
	}
	if len(c2.Evidence.Signals) != 1 || c2.Evidence.Signals[0].SignalID != sig || len(c2.HumanDecisions) != 1 || *c2.HumanDecisions[0].Author != "qc-1" {
		t.Fatalf("слои карточки: %+v %+v", c2.Evidence.Signals, c2.HumanDecisions)
	}
	// Карточка на момент до отклонения — черновик (AD-22).
	at := nctest.T0.Add(time.Hour)
	old, err := svc.Card(context.Background(), ncID, platform.Moment{Axis: platform.AxisOccurred, AsOf: &at})
	if err != nil || old.Status != "draft" {
		t.Fatalf("на момент: %s %v", old.Status, err)
	}
}

// FR-53, FR-54: «как есть» без действующего разрешения не подписывается;
// с разрешением — исполняется (демо-заглушка подписей помечена), «годно по
// разрешению на отклонение» ≠ «годно»; режим 4 без подписей не исполняется.
func TestUseAsIsConcessionAndApprovals(t *testing.T) {
	w := nctest.NewWorld(t)
	svc := w.Service(app.DemoRoutes{})
	ctx := nctest.As("qc-1", "quality_inspector")
	ncID, c := signalNC(t, w, svc, "FL:0002")
	if _, err := svc.Confirm(ctx, ncID, app.ConfirmNonconformity{CommandHeader: hdr(c.BasisSeq), SignalIDs: []string{c.Evidence.Signals[0].SignalID},
		Severity: "major", Reason: reason("Прожог подтверждён")}); err != nil {
		t.Fatal(err)
	}
	w.Settle()
	c, _ = svc.Card(ctx, ncID, platform.Moment{})
	if c.Status != "confirmed" || !c.ToDecide.ConcessionRequired || !slices.Contains(c.ToDecide.Decisions, "nonconformity.disposition.set") {
		t.Fatalf("подтверждено: %+v", c.ToDecide)
	}
	chief := nctest.As("chief-1", "head_of_qc")
	_, err := svc.SetDisposition(chief, ncID, app.SetDisposition{CommandHeader: hdr(c.BasisSeq), Disposition: "use_as_is", Reason: reason("в допуске")})
	if codeOf(err) != errcodes.NonconformityConcessionRequired {
		t.Fatalf("«как есть» без разрешения: %v", err)
	}
	_, err = svc.SetDisposition(chief, ncID, app.SetDisposition{CommandHeader: hdr(c.BasisSeq), Disposition: "use_as_is", ConcessionID: "CON-X", Reason: reason("в допуске")})
	if codeOf(err) != errcodes.NonconformityConcessionNotApplicable {
		t.Fatalf("несуществующее разрешение: %v", err)
	}
	if _, err := svc.GrantConcession(chief, app.GrantConcession{CommandHeader: hdr(0), ConcessionID: "CON-1", Title: "Отклонение катета шва",
		Kind: "use_as_is", ScopeItemIDs: []string{"FL:0002"}, Limit: 1, ValidUntil: "2026-12-31T00:00:00Z", Reason: reason("ТУ п. 4.2")}); err != nil {
		t.Fatal(err)
	}
	cl, err := svc.Concessions(ctx, "FL:0002", platform.Moment{})
	if err != nil || len(cl.Items) != 1 || cl.Items[0].Status != "active" {
		t.Fatalf("разрешения изделия: %+v %v", cl, err)
	}
	if other, _ := svc.Concessions(ctx, "FL:0099", platform.Moment{}); len(other.Items) != 0 {
		t.Fatal("разрешение вне области показано")
	}
	if _, err := svc.SetDisposition(chief, ncID, app.SetDisposition{CommandHeader: hdr(c.BasisSeq), Disposition: "use_as_is", ConcessionID: "CON-1",
		Reason: reason("в допуске по разрешению")}); err != nil {
		t.Fatal(err)
	}
	w.Settle()
	c, _ = svc.Card(ctx, ncID, platform.Moment{})
	if c.Axes.Disposition != "use_as_is" || c.Axes.Quality != "accepted_with_concession" || c.ApprovalsStatus == nil || *c.ApprovalsStatus != "demo_stub" {
		t.Fatalf("исполнено: %+v %v", c.Axes, c.ApprovalsStatus)
	}
	if cl, _ := svc.Concessions(ctx, "FL:0002", platform.Moment{}); cl.Items[0].Used != 1 || cl.Items[0].Status != "exhausted" {
		t.Fatalf("расход: %+v", cl.Items[0])
	}

	// Режим 4: подписи маршрута не собраны — решение записано, не исполняется.
	strict := w.Service(app.PendingRoutes{})
	nc3, c3 := signalNC(t, w, strict, "FL:0003")
	if _, err := strict.Confirm(ctx, nc3, app.ConfirmNonconformity{CommandHeader: hdr(c3.BasisSeq), SignalIDs: []string{c3.Evidence.Signals[0].SignalID},
		Severity: "major", Reason: reason("да")}); err != nil {
		t.Fatal(err)
	}
	w.Settle()
	if _, err := strict.SetDisposition(chief, nc3, app.SetDisposition{CommandHeader: hdr(0), Disposition: "scrap", ScrapKind: "writeoff", Reason: reason("списать")}); err != nil {
		t.Fatal(err)
	}
	w.Settle()
	c3, _ = strict.Card(ctx, nc3, platform.Moment{})
	if c3.Axes.Disposition != "none" || c3.ApprovalsStatus == nil || *c3.ApprovalsStatus != "pending" || slices.Contains(c3.ToDecide.Decisions, "nonconformity.nonconformity.close") {
		t.Fatalf("режим 4 без подписей: %+v %v %v", c3.Axes, c3.ApprovalsStatus, c3.ToDecide.Decisions)
	}
}

// FR-55, FR-56: изоляция со сроком по календарю и расхождение «физически не
// перемещено»; участник изготовления не принимает на точке предъявления.
func TestIsolationAndSeparationOfDuties(t *testing.T) {
	w := nctest.NewWorld(t)
	svc := w.Service(app.DemoRoutes{})
	ctx := nctest.As("qc-1", "quality_inspector")
	ncID, c := signalNC(t, w, svc, "FL:0004")
	w.Now = time.Date(2026, 10, 2, 10, 0, 0, 0, time.UTC) // пятница
	if _, err := svc.Isolate(ctx, "FL:0004", app.IsolateItem{CommandHeader: hdr(c.BasisSeq), Reason: reason("до решения")}); err != nil {
		t.Fatal(err)
	}
	w.Settle()
	c, _ = svc.Card(ctx, ncID, platform.Moment{})
	want := time.Date(2026, 10, 7, 10, 0, 0, 0, time.UTC) // 3 рабочих дня: пн, вт, ср
	if c.Isolation == nil || c.Isolation.DecisionDueAt == nil || !c.Isolation.DecisionDueAt.Equal(want) || !c.PhysicallyNotMoved || c.Axes.Position != "isolated" {
		t.Fatalf("изоляция: %+v not_moved=%v %s", c.Isolation, c.PhysicallyNotMoved, c.Axes.Position)
	}
	w.Now = want.Add(time.Hour)
	q, _ := svc.Queue(ctx, app.QueueFilter{}, platform.Moment{}, platform.Page{})
	if len(q.Items) == 0 || !q.Items[0].Overdue {
		t.Fatalf("просрочка в очереди: %+v", q.Items)
	}
	w.Add(nctest.Record(catalog.OperationMovementReceived, "FL:0004", time.Date(2026, 10, 2, 11, 0, 0, 0, time.UTC),
		map[string]any{"to_location_id": "ISO-1", "destination_kind": "isolator", "inspection_on_receipt": "not_inspected", "received_by": "st-1"}))
	w.Settle()
	c, _ = svc.Card(ctx, ncID, platform.Moment{})
	if c.PhysicallyNotMoved || !c.Isolation.PhysicallyMoved {
		t.Fatal("приёмка в изоляторе не сняла расхождение")
	}

	w.Add(nctest.Record(catalog.ItemPresentationRecorded, "FL:0005", nctest.T0, map[string]any{"step_key": "welding.zt3_acceptance",
		"presentation_no": 1, "presented_to": "qc", "presented_by": "master-1"}))
	w.Add(nctest.Run("FL:0005", nctest.T0.Add(-time.Hour), "RUN-5", "op-7"))
	w.Settle()
	q, _ = svc.Queue(ctx, app.QueueFilter{Kind: "presentation"}, platform.Moment{}, platform.Page{})
	if len(q.Items) != 1 || q.Items[0].PresentationN == nil {
		t.Fatalf("точка предъявления в очереди: %+v", q.Items)
	}
	_, err := svc.ResolvePresentation(nctest.As("op-7", "quality_inspector"), "FL:0005", app.ResolvePresentation{CommandHeader: hdr(0),
		StepKey: "welding.zt3_acceptance", ClosingPoint: "ZT-3", Resolution: "accept", PresentationNo: 1, MethodEventIDs: []string{"x"}})
	if codeOf(err) != errcodes.AccessSeparationOfDuties {
		t.Fatalf("разделение обязанностей: %v", err)
	}
}
