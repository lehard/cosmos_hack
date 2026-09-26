package nonconformity_test

import (
	"testing"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	"ant/internal/contracts/statuses"
	"ant/internal/domain/kernel"
	nc "ant/internal/domain/nonconformity"
)

// Д-81, FR-32, FR-146: пересмотр решения на точке — новая запись; «оставить
// в силе» — только если приёмка прошла бы сейчас; «отозвать приёмку» — блок
// человека и качество «не проверено», без несоответствия; повторный отзыв и
// пересмотр без основания — отказ.
func TestPresentationReview(t *testing.T) {
	var j journal
	j.add(catalog.OperationRunStarted, map[string]any{"operation_run_id": "RUN-1", "operation_code": "020", "step_key": "welding.weld", "operator_id": "op-7"}, "")
	res := j.add(catalog.InspectionResultRecorded, map[string]any{}, "")
	j.add(catalog.ItemPresentationRecorded, map[string]any{"step_key": "welding.zt3_acceptance", "presentation_no": 1, "presented_to": "qc", "presented_by": "master-1"}, "")
	dec := j.add(catalog.DecisionPresentationResolved, nc.PresentationResolvedData{StepKey: "welding.zt3_acceptance", ClosingPoint: "ZT-3", Resolution: "accept",
		PresentationNo: 1, MethodEventIDs: []string{res.EventID}}, "qc-1")
	late := j.add(catalog.EquipmentDeviationDetected, map[string]any{"parameter": "current_a"}, "")
	s, _ := fold(nc.Env{}, j.recs, nil)
	if s.Blocked() {
		t.Fatal("метка «принято до новых данных» сама не блокирует изделие")
	}
	review := func(outcome, reason string) nc.PresentationReviewedData {
		return nc.PresentationReviewedData{ReviewedEventID: dec.EventID, StepKey: "welding.zt3_acceptance", ClosingPoint: "ZT-3", PresentationNo: 1,
			Outcome: outcome, NewFactIDs: []string{late.EventID}, Reason: nc.Reason{Text: reason}}
	}
	cmd := func(actor string, p nc.PresentationReviewedData) kernel.Command {
		return kernel.Command{Action: nc.ActPresentationReview, Actor: actor, Payload: p}
	}
	if c := code(nc.Guard(s, nc.Env{}, nc.Upstream{}, cmd("qc-1", review(nc.ReviewUpheld, " ")))); c != errcodes.NonconformityInvalidTransition {
		t.Fatalf("без основания: %q", c)
	}
	if c := code(nc.Guard(s, nc.Env{}, nc.Upstream{}, cmd("op-7", review(nc.ReviewUpheld, "режим в допуске")))); c != errcodes.AccessSeparationOfDuties {
		t.Fatalf("участник оставляет приёмку в силе: %q", c)
	}
	if err := nc.Guard(s, nc.Env{}, nc.Upstream{}, cmd("qc-1", review(nc.ReviewUpheld, "режим в допуске"))); err != nil {
		t.Fatalf("оставить в силе без блока: %v", err)
	}
	unknown := review(nc.ReviewRevoked, "нет")
	unknown.ReviewedEventID = "00000000-0000-7000-8000-999999999999"
	if c := code(nc.Guard(s, nc.Env{}, nc.Upstream{}, cmd("qc-1", unknown))); c != errcodes.NonconformityInvalidTransition {
		t.Fatalf("чужое решение: %q", c)
	}

	// Блок по новым фактам (правило, область риска): оставить в силе нельзя, отозвать — можно.
	hold := j.add(catalog.DecisionContainmentSet, nc.ContainmentSetData{Level: string(statuses.ContainmentItemHold), Reason: nc.Reason{Text: "область риска"}}, "hqc-1")
	s, _ = fold(nc.Env{}, j.recs, nil)
	if c := code(nc.Guard(s, nc.Env{}, nc.Upstream{}, cmd("qc-1", review(nc.ReviewUpheld, "режим в допуске")))); c != errcodes.NonconformityItemBlocked {
		t.Fatalf("оставить в силе при блоке: %q", c)
	}
	if err := nc.Guard(s, nc.Env{}, nc.Upstream{}, cmd("op-7", review(nc.ReviewRevoked, "ток вне уставки"))); err != nil {
		t.Fatalf("отзыв — защитное направление, разделение обязанностей не мешает: %v", err)
	}
	j.add(catalog.DecisionContainmentReleased, nc.ContainmentReleasedData{ReleasedEventIDs: []string{hold.EventID}, Reason: nc.Reason{Text: "снято"}}, "hqc-1")

	rv := j.add(catalog.DecisionPresentationReviewed, review(nc.ReviewRevoked, "ток 176 А при уставке 160 ± 10 А"), "qc-1")
	s, _ = fold(nc.Env{}, j.recs, nil)
	p := s.Presentations[0]
	if !p.Revoked() || p.ReviewEventID != rv.EventID || p.ResolvedEventID != dec.EventID || p.Resolution != "accept" {
		t.Fatalf("отзыв не записан поверх прежнего решения: %+v", p)
	}
	if !s.Blocked() || len(s.NCs) != 0 || s.Disposition() != "none" {
		t.Fatalf("отзыв: блок=%v НС=%d решение=%s", s.Blocked(), len(s.NCs), s.Disposition())
	}
	st := nc.Reduce(nc.State{}, j.recs[0], nc.Env{}, nc.Upstream{})
	for _, r := range j.recs[1:] {
		st = nc.Reduce(st, r, nc.Env{}, nc.Upstream{})
	}
	var quality string
	for _, e := range st.Effects {
		if e.Kind == "set_quality" && e.Cause == rv.EventID {
			quality = e.Value
		}
	}
	if quality != string(statuses.QualityNotInspected) {
		t.Fatalf("качество после отзыва: %q (%+v)", quality, st.Effects)
	}
	if c := code(nc.Guard(s, nc.Env{}, nc.Upstream{}, cmd("qc-1", review(nc.ReviewRevoked, "ещё раз")))); c != errcodes.NonconformityInvalidTransition {
		t.Fatalf("повторный отзыв: %q", c)
	}
	// Снимает блок отзыва уполномоченный человек — существующей операцией.
	rel := kernel.Command{Action: nc.ActContainmentRelease, Payload: nc.ContainmentReleasedData{ReleasedEventIDs: []string{rv.EventID}, Reason: nc.Reason{Text: "доп. проверка — годно"}}}
	if err := nc.Guard(s, nc.Env{}, nc.Upstream{}, rel); err != nil {
		t.Fatalf("снятие блока отзыва: %v", err)
	}
}
