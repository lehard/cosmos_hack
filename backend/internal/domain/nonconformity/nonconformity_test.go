package nonconformity_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	ev "ant/internal/contracts/events"
	"ant/internal/contracts/statuses"
	"ant/internal/domain/kernel"
	nc "ant/internal/domain/nonconformity"
	"ant/internal/domain/quality"
)

var t0 = time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)

const item = "FL:0001"

type journal struct {
	n    int
	recs []kernel.Record
}

func (j *journal) add(t catalog.Type, data any, actor string) kernel.Record {
	j.n++
	b, _ := json.Marshal(data)
	info, _ := catalog.Lookup(t)
	r := kernel.Record{
		Seq: int64(j.n), EventID: fmt.Sprintf("00000000-0000-7000-8000-%012d", j.n), Type: t, Kind: info.Kind,
		ItemID: item, Stream: "item:" + item, OccurredAt: t0.Add(time.Duration(j.n) * time.Minute), Actor: actor, Data: b,
	}
	j.recs = append(j.recs, r)
	return r
}

// fold — свёртка модуля отдельно: Reduce, React, намерения к nonconformity — Apply в конце шага.
// Возвращает состояние и вывод React последнего шага (намерения — только его записи).
func fold(env nc.Env, recs []kernel.Record, intents map[string][]kernel.Intent) (nc.State, kernel.Output) {
	var s nc.State
	for _, r := range recs {
		s = nc.Reduce(s, r, env, nc.Upstream{})
		_ = nc.React(s, env, nc.Upstream{})
		for _, in := range intents[r.EventID] {
			s = nc.Apply(s, in)
		}
	}
	return s, nc.React(s, env, nc.Upstream{})
}

func draftIntent(r kernel.Record, outcome string, signals ...string) kernel.Intent {
	sev := ev.Severity("major")
	oc := ev.DecisionNonconformityDraftedV1ReactionOutcome(outcome)
	bk := ev.DecisionNonconformityDraftedV1BasisKindInspectionResult
	sk := ev.StepKey("welding.weld")
	req := nc.DraftRequest{Severity: &sev, ReactionOutcome: &oc, BasisKind: &bk, StepKey: &sk}
	for _, s := range signals {
		req.SignalIds = append(req.SignalIds, ev.ObjectID(s))
	}
	return kernel.NewIntent("nonconformity", quality.Module, nc.IntentDraft, req, r)
}

func reactionsOf(out kernel.Output, t catalog.Type) []kernel.Reaction {
	var rs []kernel.Reaction
	for _, r := range out.Reactions {
		if r.Type == t {
			rs = append(rs, r)
		}
	}
	return rs
}

func code(err error) errcodes.Code {
	var r *kernel.Refusal
	if errors.As(err, &r) {
		return r.Code
	}
	return ""
}

// FR-51, FR-50 режим 1: намерение quality даёт черновик и реакцию
// decision.nonconformity.drafted; реакция карты «изолировать» — блок правилом.
func TestDraftIntentAndContainment(t *testing.T) {
	var j journal
	res := j.add(catalog.InspectionResultRecorded, map[string]any{"method": "camera", "phase": "after_operation", "outcome": "defect_found", "processing_state": "final"}, "")
	s, out := fold(nc.Env{}, j.recs, map[string][]kernel.Intent{res.EventID: {draftIntent(res, "isolate", "SIG-1")}})
	if len(s.NCs) != 1 || s.NCs[0].Status != nc.StatusDraft || s.NCs[0].ID != nc.DraftNCID(item, []string{"SIG-1"}) {
		t.Fatalf("черновик: %+v", s.NCs)
	}
	if d := reactionsOf(out, catalog.DecisionNonconformityDrafted); len(d) != 1 || d[0].Slot.TriggerKey != s.NCs[0].ID {
		t.Fatalf("реакция черновика: %+v", out.Reactions)
	}
	if c := reactionsOf(out, catalog.DecisionContainmentApplied); len(c) != 1 {
		t.Fatalf("блок правилом: %+v", out.Reactions)
	}
	if s.ContainmentLevel() != string(statuses.ContainmentItemHold) || !s.Blocked() {
		t.Fatalf("ось сдерживания: %s", s.ContainmentLevel())
	}
	// Повтор намерения с теми же сигналами — тот же черновик.
	s2 := nc.Apply(s, draftIntent(res, "isolate", "SIG-1"))
	if len(s2.NCs) != 1 {
		t.Fatalf("повтор намерения: %d", len(s2.NCs))
	}
}

// FR-51, FR-52: отклонение сигнала не меняет исходный сигнал — решение
// отдельной записью; отклонение без причины невозможно.
func TestRejectSignalKeepsSource(t *testing.T) {
	var j journal
	res := j.add(catalog.InspectionResultRecorded, map[string]any{}, "")
	intents := map[string][]kernel.Intent{res.EventID: {draftIntent(res, "manual_review", "SIG-1")}}
	s, _ := fold(nc.Env{}, j.recs, intents)
	before := s.NCs[0].Draft
	srcData := string(res.Data)

	cmd := kernel.Command{Action: nc.ActRejectSignal, Actor: "qc-1", Payload: nc.SignalRejectedData{SignalIDs: []string{"SIG-1"}, Reason: nc.Reason{Text: "  "}}}
	if c := code(nc.Guard(s, nc.Env{}, nc.Upstream{}, cmd)); c != errcodes.NonconformityRejectReasonRequired {
		t.Fatalf("без причины: %q", c)
	}
	data := nc.SignalRejectedData{SignalIDs: []string{"SIG-1"}, Reason: nc.Reason{Text: "Блик на шве, не дефект"}}
	cmd.Payload = data
	if err := nc.Guard(s, nc.Env{}, nc.Upstream{}, cmd); err != nil {
		t.Fatal(err)
	}
	rej := j.add(catalog.DecisionSignalRejected, data, "qc-1")
	s, out := fold(nc.Env{}, j.recs, intents)
	n := s.NCs[0]
	if n.Status != nc.StatusClosed || n.Resolution != nc.ResolutionSignalRejected {
		t.Fatalf("черновик после отклонения: %+v", n)
	}
	if fmt.Sprint(n.Draft) != fmt.Sprint(before) || string(j.recs[0].Data) != srcData {
		t.Fatal("исходный сигнал изменился")
	}
	if len(s.Decisions) != 1 || s.Decisions[0].EventID != rej.EventID || s.Decisions[0].Actor != "qc-1" {
		t.Fatalf("решение — отдельная запись: %+v", s.Decisions)
	}
	// Черновик остаётся в журнале как был: реакция черновика не исчезла.
	if len(reactionsOf(out, catalog.DecisionNonconformityDrafted)) != 1 {
		t.Fatal("реакция черновика исчезла")
	}
	// Повторное отклонение — отказ.
	if c := code(nc.Guard(s, nc.Env{}, nc.Upstream{}, cmd)); c != errcodes.NonconformityInvalidTransition {
		t.Fatalf("повтор отклонения: %q", c)
	}
}

// FR-53, FR-54: «как есть» без разрешения не подписывается; режим 4 — не
// исполняется без подписей; «годно по разрешению» ≠ «годно».
func TestUseAsIsNeedsConcessionAndApprovals(t *testing.T) {
	var j journal
	res := j.add(catalog.InspectionResultRecorded, map[string]any{}, "")
	intents := map[string][]kernel.Intent{res.EventID: {draftIntent(res, "manual_review", "SIG-1")}}
	s, _ := fold(nc.Env{}, j.recs, intents)
	id := s.NCs[0].ID

	disp := nc.DispositionSetData{NCID: id, Disposition: "use_as_is", Reason: nc.Reason{Text: "в допуске по чертежу"}}
	cmd := kernel.Command{Action: nc.ActDisposition, Payload: disp}
	if c := code(nc.Guard(s, nc.Env{}, nc.Upstream{}, cmd)); c != errcodes.NonconformityInvalidTransition {
		t.Fatalf("решение по черновику: %q", c)
	}
	j.add(catalog.DecisionNonconformityConfirmed, nc.ConfirmedData{NCID: id, SignalIDs: []string{"SIG-1"}, Severity: "major", Reason: nc.Reason{Text: "подтверждаю"}}, "qc-1")
	s, out := fold(nc.Env{}, j.recs, intents)
	// Ось качества по подтверждению quality ведёт сам (читает решение, эпик 20).
	if s.NCs[0].Status != nc.StatusConfirmed || len(out.Intents) != 0 {
		t.Fatalf("подтверждение: %+v %+v", s.NCs[0], out.Intents)
	}
	if c := code(nc.Guard(s, nc.Env{}, nc.Upstream{}, cmd)); c != errcodes.NonconformityConcessionRequired {
		t.Fatalf("«как есть» без разрешения: %q", c)
	}
	disp.ConcessionID, disp.ApprovalsStatus, disp.DocumentID = "CON-1", nc.ApprovalsPending, "DOC-1"
	cmd.Payload = disp
	if err := nc.Guard(s, nc.Env{}, nc.Upstream{}, cmd); err != nil {
		t.Fatal(err)
	}
	j.add(catalog.DecisionDispositionSet, disp, "chief-1")
	s, out = fold(nc.Env{}, j.recs, intents)
	if n := s.NCs[0]; n.Executed || s.Disposition() != "none" || len(out.Intents) != 0 {
		t.Fatalf("режим 4 без подписей исполнен: %+v %s", n, s.Disposition())
	}
	j.add(catalog.DocumentRouteClosed, map[string]any{"document_id": "DOC-1", "version": 1, "doc_digest": "x", "signature_event_ids": []string{}}, "")
	s, out = fold(nc.Env{}, j.recs, intents)
	if !s.NCs[0].Executed || s.Disposition() != "use_as_is" {
		t.Fatalf("после закрытия маршрута: %+v", s.NCs[0])
	}
	if len(out.Intents) != 1 || out.Intents[0].Payload != statuses.QualityAcceptedWithConcession {
		t.Fatalf("«годно по разрешению»: %+v", out.Intents)
	}
}

// FR-53: «вернуть поставщику» — только необработанное изделие.
func TestReturnOnlyUnprocessed(t *testing.T) {
	var j journal
	j.add(catalog.OperationRunStarted, map[string]any{"operation_run_id": "RUN-1", "operation_code": "010", "step_key": "welding.weld", "operator_id": "op-7"}, "")
	res := j.add(catalog.InspectionResultRecorded, map[string]any{}, "")
	intents := map[string][]kernel.Intent{res.EventID: {draftIntent(res, "manual_review", "SIG-1")}}
	j.add(catalog.DecisionNonconformityConfirmed, nc.ConfirmedData{NCID: nc.DraftNCID(item, []string{"SIG-1"}), SignalIDs: []string{"SIG-1"}, Severity: "major", Reason: nc.Reason{Text: "да"}}, "qc-1")
	s, _ := fold(nc.Env{}, j.recs, intents)
	cmd := kernel.Command{Action: nc.ActDisposition, Payload: nc.DispositionSetData{NCID: s.NCs[0].ID, Disposition: "return_to_supplier", Reason: nc.Reason{Text: "брак заготовки"}}}
	if c := code(nc.Guard(s, nc.Env{}, nc.Upstream{}, cmd)); c != errcodes.NonconformityReturnOnlyUnprocessed {
		t.Fatalf("возврат обработанного: %q", c)
	}
}

// FR-56: участник изготовления не принимает изделие на точке предъявления.
func TestSeparationOfDuties(t *testing.T) {
	var j journal
	j.add(catalog.OperationRunStarted, map[string]any{"operation_run_id": "RUN-1", "operation_code": "010", "step_key": "welding.weld", "operator_id": "op-7"}, "")
	res := j.add(catalog.InspectionResultRecorded, map[string]any{}, "")
	j.add(catalog.ItemPresentationRecorded, map[string]any{"step_key": "welding.zt3_acceptance", "presentation_no": 1, "presented_to": "qc", "presented_by": "master-1"}, "")
	s, _ := fold(nc.Env{}, j.recs, nil)
	p := s.PendingPresentation()
	if p == nil || p.ClosingPoint != "ZT-3" {
		t.Fatalf("предъявление: %+v", p)
	}
	data := nc.PresentationResolvedData{StepKey: p.StepKey, ClosingPoint: p.ClosingPoint, Resolution: "accept", PresentationNo: 1, MethodEventIDs: []string{res.EventID}}
	cmd := kernel.Command{Action: nc.ActPresentation, Actor: "op-7", Payload: data}
	if c := code(nc.Guard(s, nc.Env{}, nc.Upstream{}, cmd)); c != errcodes.AccessSeparationOfDuties {
		t.Fatalf("участник принимает: %q", c)
	}
	cmd.Actor = "qc-1"
	if err := nc.Guard(s, nc.Env{}, nc.Upstream{}, cmd); err != nil {
		t.Fatal(err)
	}
	data.MethodEventIDs = []string{"нет-такого"}
	cmd.Payload = data
	if c := code(nc.Guard(s, nc.Env{}, nc.Upstream{}, cmd)); c != errcodes.NonconformityMethodResultMissing {
		t.Fatalf("без результата метода: %q", c)
	}
}

// FR-55: «изолировано в системе, физически не перемещено» до приёмки в изоляторе.
func TestIsolationPhysicalMove(t *testing.T) {
	var j journal
	j.add(catalog.DecisionItemIsolated, nc.IsolatedData{DecisionDueAt: "2026-09-29T08:00:00.000Z", Reason: nc.Reason{Text: "прожог"}}, "qc-1")
	s, out := fold(nc.Env{}, j.recs, nil)
	if !s.Isolated() || !s.PhysicallyNotMoved() || s.Isolation.DecisionDueAt == nil || !s.Blocked() {
		t.Fatalf("изоляция: %+v", s.Isolation)
	}
	if len(out.Intents) != 1 || out.Intents[0].Name != "isolate" {
		t.Fatalf("положение «в изоляции» — намерением process: %+v", out.Intents)
	}
	j.add(catalog.OperationMovementReceived, map[string]any{"to_location_id": "ISO-1", "destination_kind": "isolator", "inspection_on_receipt": "not_inspected", "received_by": "st-1"}, "")
	s, out = fold(nc.Env{}, j.recs, nil)
	if len(out.Intents) != 0 {
		t.Fatal("намерение повторяется на следующих шагах")
	}
	if s.PhysicallyNotMoved() || s.Isolation.MovedEventID == "" {
		t.Fatalf("после приёмки в изоляторе: %+v", s.Isolation)
	}
}

// FR-62, AD-27: блок по области риска ставит правило; снятие — задача
// человеку (реакция исчезает, блок остаётся), по делегированному правилу — само.
func TestIncidentScopeContainment(t *testing.T) {
	var j journal
	j.add(catalog.IncidentMembershipChanged, map[string]any{"incident_id": "INC-1", "scope_version": 1, "status": "suspect", "action": "block"}, "")
	s, out := fold(nc.Env{}, j.recs, nil)
	if !s.Blocked() || len(reactionsOf(out, catalog.DecisionContainmentApplied)) != 1 {
		t.Fatalf("блок по области: %s %+v", s.ContainmentLevel(), out.Reactions)
	}
	j.add(catalog.IncidentMembershipChanged, map[string]any{"incident_id": "INC-1", "scope_version": 2, "status": "excluded", "action": "release"}, "")
	s, out = fold(nc.Env{}, j.recs, nil)
	if !s.Blocked() || len(reactionsOf(out, catalog.DecisionContainmentApplied)) != 0 {
		t.Fatalf("снятие без делегирования: %s %+v", s.ContainmentLevel(), out.Reactions)
	}
	// Человек снимает при ушедшем основании.
	rel := kernel.Command{Action: nc.ActContainmentRelease, Payload: nc.ContainmentReleasedData{ReleasedEventIDs: []string{"incident:INC-1"}, Reason: nc.Reason{Text: "исключено"}}}
	if err := nc.Guard(s, nc.Env{}, nc.Upstream{}, rel); err != nil {
		t.Fatal(err)
	}
	s, out = fold(nc.Env{DelegatedIncidentRelease: true}, j.recs, nil)
	if s.Blocked() || len(reactionsOf(out, catalog.DecisionContainmentApplied)) != 0 {
		t.Fatalf("снятие делегированным правилом: %s", s.ContainmentLevel())
	}
}

// FR-151: регистрация окна нарушения специального процесса — несоответствие
// без дефекта, блок правилом, решение — комиссия.
func TestSpecialProcessRegistrationInFold(t *testing.T) {
	var j journal
	j.add(catalog.OperationRunStarted, map[string]any{"operation_run_id": "RUN-1", "operation_code": "020", "step_key": "welding.weld", "operator_id": "op-7"}, "")
	a, err := nc.RegisterWindowNC(mlRequest())
	if err != nil {
		t.Fatal(err)
	}
	r := j.add(catalog.DecisionNonconformityRegistered, a.Data, "")
	r.Stream = a.Stream
	s, out := fold(nc.Env{}, j.recs, nil)
	rs := out.Reactions
	if len(s.NCs) != 1 || !s.NCs[0].Commission || s.NCs[0].Status != nc.StatusConfirmed || !s.Blocked() {
		t.Fatalf("несоответствие окна: %+v", s.NCs)
	}
	found := false
	for _, re := range rs {
		found = found || re.Type == catalog.DecisionContainmentApplied
	}
	if !found {
		t.Fatalf("блок правилом не вычислен: %+v", rs)
	}
	// S04-03: без найденного дефекта изделие ждёт комиссию, «не годно» правило не ставит.
	for _, it := range out.Intents {
		if it.Name == "set_quality" {
			t.Fatalf("ось качества от правила окна: %+v", it)
		}
	}
}

// FR-49: точка чистоты — первые N изделий после снятия остановки получают
// усиленный контроль.
func TestCleanPoint(t *testing.T) {
	var s nc.StageState
	rec := func(t catalog.Type, it string, n int, data any) kernel.Record {
		b, _ := json.Marshal(data)
		return kernel.Record{EventID: fmt.Sprintf("00000000-0000-7000-8000-%012d", n), Type: t, ItemID: it, OccurredAt: t0.Add(time.Duration(n) * time.Minute), Data: b}
	}
	var out []kernel.Addressed
	step := func(r kernel.Record) {
		var a []kernel.Addressed
		s, a = nc.Stage(s, r)
		out = append(out, a...)
	}
	step(rec(catalog.DecisionProcessHoldSet, "", 1, nc.ProcessHoldSetData{HoldID: "H-1", Level: "process_point_stop", EquipmentID: "WELD-1", Reason: nc.Reason{Text: "ток вне уставки"}}))
	step(rec(catalog.OperationRunStarted, "FL:1", 2, map[string]any{"operation_run_id": "R1", "equipment_id": "WELD-1"}))
	if len(out) != 0 {
		t.Fatal("точка чистоты до снятия")
	}
	if err := nc.HoldGuard(s, kernel.Command{Payload: nc.ProcessHoldReleasedData{HoldID: "H-9"}}); code(err) != errcodes.NonconformityProcessHoldNotActive {
		t.Fatalf("снятие несуществующей: %v", err)
	}
	step(rec(catalog.DecisionProcessHoldReleased, "", 3, nc.ProcessHoldReleasedData{HoldID: "H-1", CleanPointItems: 2, Reason: nc.Reason{Text: "устранено"}}))
	for i, it := range []string{"FL:2", "FL:3", "FL:4"} {
		step(rec(catalog.OperationRunStarted, it, 4+i, map[string]any{"operation_run_id": "R" + it, "equipment_id": "WELD-1"}))
	}
	if len(out) != 2 || out[0].Stream != "item:FL:2" || out[1].Stream != "item:FL:3" {
		t.Fatalf("точка чистоты: %+v", out)
	}
}

// FR-54: книга разрешений — отзыв, срок, область, вид решения.
func TestConcessionGuard(t *testing.T) {
	var j journal
	g := j.add(catalog.DecisionConcessionGranted, nc.ConcessionGrantedData{ConcessionID: "CON-1", Kind: "use_as_is", Limit: 2,
		ScopeRangeFrom: "FL:0001", ScopeRangeTo: "FL:0010", ValidUntil: "2026-10-01T00:00:00.000Z", Reason: nc.Reason{Text: "ТУ п. 4.2"}}, "chief")
	b := nc.NewConcessionBook()
	b.Apply(g)
	c := b.Items["CON-1"]
	if err := nc.ConcessionGuard(c, "CON-1", "FL:0002", "use_as_is", t0); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		item, kind string
		at         time.Time
	}{{"FL:0100", "use_as_is", t0}, {"FL:0002", "repair", t0}, {"FL:0002", "use_as_is", t0.AddDate(0, 1, 0)}} {
		if code(nc.ConcessionGuard(c, "CON-1", tc.item, tc.kind, tc.at)) != errcodes.NonconformityConcessionNotApplicable {
			t.Fatalf("неприменимо: %+v", tc)
		}
	}
	b.Apply(j.add(catalog.DecisionConcessionRevoked, nc.ConcessionRevokedData{ConcessionID: "CON-1", Reason: nc.Reason{Text: "отзыв"}}, "chief"))
	if c.Status(t0) != nc.ConcessionRevoked || code(nc.ConcessionGuard(c, "CON-1", "FL:0002", "use_as_is", t0)) != errcodes.NonconformityConcessionNotApplicable {
		t.Fatal("отозванное применимо")
	}
}

// NC-G1, S05/S10A: групповое решение «переделка» по несоответствию окна
// покрывает подтверждённое раньше несоответствие изделия (ось «решение по
// изделию» одна, AD-30) и выводит изделие из изоляции на исполнение;
// подтверждённое позже — ждёт своего решения.
func TestGroupDispositionCoversEarlierNC(t *testing.T) {
	var j journal
	j.add(catalog.OperationRunStarted, map[string]any{"operation_run_id": "RUN-1", "operation_code": "020", "step_key": "welding.weld", "operator_id": "op-7"}, "")
	insp := j.add(catalog.InspectionResultRecorded, map[string]any{"step_key": "welding.kt3_camera", "outcome": "defect_indicated"}, "")
	j.add(catalog.DecisionNonconformityConfirmed, nc.ConfirmedData{NCID: nc.DraftNCID(item, []string{"SIG-1"}), SignalIDs: []string{"SIG-1"}, Severity: "major",
		Reason: nc.Reason{Text: "прожог"}}, "qc-1")
	iso := j.add(catalog.DecisionItemIsolated, nc.IsolatedData{Reason: nc.Reason{Text: "до решения"}}, "qc-1")
	a, err := nc.RegisterWindowNC(mlRequest())
	if err != nil {
		t.Fatal(err)
	}
	j.add(catalog.DecisionNonconformityRegistered, a.Data, "")
	intents := map[string][]kernel.Intent{insp.EventID: {draftIntent(insp, "isolate", "SIG-1")}}
	s, _ := fold(nc.Env{}, j.recs, intents)
	if len(s.NCs) != 2 || !s.Isolated() || !s.Blocked() {
		t.Fatalf("до решения: %+v", s)
	}
	group := s.NCs[1].ID
	j.add(catalog.DecisionDispositionSet, nc.DispositionSetData{NCID: group, Disposition: "rework", Reason: nc.Reason{Text: "переделка ×6"}}, "tech-1")
	s, _ = fold(nc.Env{}, j.recs, intents)
	if !s.Covered(s.NCs[0]) || s.Isolated() || s.Blocked() {
		t.Fatalf("групповое решение не покрыло НС изделия: covered=%v isolated=%v containment=%+v (изоляция %s)", s.Covered(s.NCs[0]), s.Isolated(), s.Containment, iso.EventID)
	}
	accept := kernel.Command{Action: nc.ActPresentation, Payload: nc.PresentationResolvedData{StepKey: "welding.zt3_acceptance", ClosingPoint: "ZT-3",
		Resolution: "accept", PresentationNo: 2}}
	if err := nc.Guard(s, nc.Env{}, nc.Upstream{}, accept); err != nil {
		t.Fatalf("приёмка после группового решения: %v", err)
	}
	// Новое несоответствие после решения — не покрыто: приёмка ждёт.
	j.add(catalog.DecisionNonconformityConfirmed, nc.ConfirmedData{NCID: nc.DraftNCID(item, []string{"SIG-2"}), SignalIDs: []string{"SIG-2"}, Severity: "major",
		Reason: nc.Reason{Text: "поры"}}, "qc-1")
	insp2 := j.add(catalog.InspectionResultRecorded, map[string]any{"step_key": "welding.kt3_radiography", "outcome": "defect_indicated"}, "")
	intents[insp2.EventID] = []kernel.Intent{draftIntent(insp2, "manual_review", "SIG-2")}
	j.recs[len(j.recs)-2], j.recs[len(j.recs)-1] = j.recs[len(j.recs)-1], j.recs[len(j.recs)-2]
	s, _ = fold(nc.Env{}, j.recs, intents)
	if err := nc.Guard(s, nc.Env{}, nc.Upstream{}, accept); code(err) != errcodes.NonconformityItemBlocked {
		t.Fatalf("новое несоответствие не держит приёмку: %v", err)
	}
}

// SHOW-IS2: Ф-003 в области риска инцидента ИС-2 (блок правилом) с
// подтверждённым прожогом; «Переделка» — изделие выведено из области на
// исполнение решения: блок области снят, приёмку на ЗТ-3 после переварки он
// не держит; следующая версия области блок сама не возвращает.
func TestReworkReleasesIncidentScopeHold(t *testing.T) {
	var j journal
	j.add(catalog.OperationRunStarted, map[string]any{"operation_run_id": "RUN-1", "operation_code": "030", "step_key": "welding.weld", "operator_id": "op-7"}, "")
	insp := j.add(catalog.InspectionResultRecorded, map[string]any{"step_key": "welding.kt3_camera", "outcome": "defect_indicated"}, "")
	id := nc.DraftNCID(item, []string{"SIG-1"})
	j.add(catalog.DecisionNonconformityConfirmed, nc.ConfirmedData{NCID: id, SignalIDs: []string{"SIG-1"}, Severity: "critical", Reason: nc.Reason{Text: "прожог У2"}}, "qc-1")
	j.add(catalog.IncidentMembershipChanged, map[string]any{"incident_id": "INC-IS2", "scope_version": 1, "status": "confirmed", "action": "block"}, "")
	intents := map[string][]kernel.Intent{insp.EventID: {draftIntent(insp, "isolate", "SIG-1")}}
	s, _ := fold(nc.Env{}, j.recs, intents)
	if !s.Blocked() {
		t.Fatalf("до решения блок области: %+v", s.Containment)
	}
	j.add(catalog.DecisionDispositionSet, nc.DispositionSetData{NCID: id, Disposition: "rework", Reason: nc.Reason{Text: "переварка У2"}}, "tech-1")
	j.add(catalog.IncidentMembershipChanged, map[string]any{"incident_id": "INC-IS2", "scope_version": 3, "status": "confirmed", "action": "block"}, "")
	s, _ = fold(nc.Env{}, j.recs, intents)
	if s.Blocked() {
		t.Fatalf("после «Переделки» блок области держит: %+v", s.Containment)
	}
	accept := kernel.Command{Action: nc.ActPresentation, Payload: nc.PresentationResolvedData{StepKey: "welding.zt3_acceptance", ClosingPoint: "ZT-3",
		Resolution: "accept", PresentationNo: 2}}
	if err := nc.Guard(s, nc.Env{}, nc.Upstream{}, accept); err != nil {
		t.Fatalf("ЗТ-3 после переварки: %v", err)
	}
}
