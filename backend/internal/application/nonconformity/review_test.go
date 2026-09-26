package nonconformity_test

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	app "ant/internal/application/nonconformity"
	"ant/internal/application/nonconformity/nctest"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/domain/engine"
	"ant/internal/domain/kernel"
)

// reviewFold — свёртка с черновиком несоответствия (nctest.DraftingFold) и
// задачами движка «решение принято до новых данных» (engine.Fold, AD-3).
func reviewFold(b engine.Bundle, in []kernel.Record) (engine.Snapshot, []kernel.Reaction) {
	s, out := nctest.DraftingFold(b, in)
	_, all := engine.Fold(b, in)
	for _, re := range all {
		if re.Type == catalog.TaskTaskCreated {
			out = append(out, re)
		}
	}
	return s, out
}

// Очередь: решение на точке предъявления принято до пришедших позже данных —
// строка kind = review; что пришло — словами, id записи — source_event_id,
// кодов записи в заголовке нет.
func TestQueueReviewAfterNewData(t *testing.T) {
	w := nctest.NewWorld(t)
	svc := app.NewService(app.WithDeps(app.Deps{Journal: w.J, Codec: w.Codec, Fold: reviewFold, Routes: app.DemoRoutes{},
		Now: func() time.Time { return w.Now }}), app.WithConfig(app.Config{Partitions: 1}))
	item := "FL:0009"
	signalNC(t, w, svc, item)
	w.Add(nctest.Record(catalog.DecisionPresentationResolved, item, nctest.T0.Add(2*time.Hour), map[string]any{"step_key": "welding.zt3_acceptance",
		"closing_point": "ZT-3", "resolution": "accept", "presentation_no": 1}))
	late := nctest.Record(catalog.EquipmentCycleSummarized, item, nctest.T0.Add(time.Hour), map[string]any{"equipment_id": "WELD-1",
		"window_start": nctest.T0.Format(time.RFC3339), "window_end": nctest.T0.Add(time.Hour).Format(time.RFC3339), "parameters": []map[string]any{{"parameter": "current_a"}}})
	w.Add(late)
	w.Settle()
	q, err := svc.Queue(context.Background(), app.QueueFilter{Kind: "review"}, platform.Moment{}, platform.Page{})
	if err != nil || len(q.Items) != 1 {
		t.Fatalf("пересмотр в очереди: %+v %v", q.Items, err)
	}
	r := q.Items[0]
	if r.SourceEventID == nil || *r.SourceEventID != late.Entry.EventID || r.ReviewSince == nil || !strings.Contains(r.Title, "ZT-3") ||
		strings.Contains(r.Title, late.Entry.EventID) || strings.Contains(r.Title, string(catalog.EquipmentCycleSummarized)) {
		t.Fatalf("строка пересмотра: %+v", r)
	}
	// Контекст решения для пересмотра: прежнее решение и что пришло после (с числами режима).
	pv, err := svc.Presentation(nctest.As("qc-2", "quality_inspector"), item, platform.Moment{})
	if err != nil || pv.Review == nil || pv.Presentation.ClosingPoint != "ZT-3" || pv.Presentation.StepKey != "welding.zt3_acceptance" {
		t.Fatalf("предъявление для пересмотра: %+v %v", pv, err)
	}
	if pv.Review.Decision.Kind != "decision" || len(pv.Review.NewFacts) != 1 || pv.Review.NewFacts[0].EventID != late.Entry.EventID {
		t.Fatalf("пересмотр: %+v", pv.Review)
	}
}

// Точка предъявления: поля команды решения; исполнитель операции принять не
// может (разделение обязанностей), без результатов методов «принять» нет;
// вернуть — можно; без предъявления — ошибка.
func TestPresentationRead(t *testing.T) {
	w := nctest.NewWorld(t)
	svc := w.Service(app.DemoRoutes{})
	w.Add(nctest.Record(catalog.ItemPresentationRecorded, "FL:0006", nctest.T0, map[string]any{"step_key": "welding.zt3_acceptance",
		"presentation_no": 1, "presented_to": "qc", "presented_by": "master-1"}))
	w.Add(nctest.Run("FL:0006", nctest.T0.Add(-time.Hour), "RUN-6", "op-7"))
	w.Settle()
	pv, err := svc.Presentation(nctest.As("op-7", "quality_inspector"), "FL:0006", platform.Moment{})
	if err != nil || pv.Presentation.StepKey != "welding.zt3_acceptance" || pv.Presentation.PresentationNo != 1 || pv.Presentation.EventID == "" || pv.Review != nil {
		t.Fatalf("предъявление: %+v %v", pv, err)
	}
	if slices.Contains(pv.Presentation.AllowedResolutions, "accept") {
		t.Fatalf("исполнитель операции: %v", pv.Presentation.AllowedResolutions)
	}
	pv, _ = svc.Presentation(nctest.As("qc-2", "quality_inspector"), "FL:0006", platform.Moment{})
	if !slices.Contains(pv.Presentation.AllowedResolutions, "reject") || slices.Contains(pv.Presentation.AllowedResolutions, "accept_with_concession") {
		t.Fatalf("контролёр: %v", pv.Presentation.AllowedResolutions)
	}
	if _, err := svc.Presentation(context.Background(), "FL:0404", platform.Moment{}); err == nil {
		t.Fatal("без изделия — ошибка")
	}
}

type reviewClock struct{ w *nctest.World }

func (c reviewClock) Now(context.Context) (time.Time, error) { return c.w.Now, nil }

// Д-81: пересмотр приёмки на ЗТ-3 «годно при неполных данных» — сервер
// отдаёт основание решения (с отметкой «данных не было»), почему новые факты
// значимы, рекомендацию отдельно от политики и решения с последствиями;
// «отозвать приёмку» пишет новую запись поверх прежней, изделие — на блок,
// задача пересмотра закрыта.
func TestReviewRevokeAcceptance(t *testing.T) {
	w := nctest.NewWorld(t)
	svc := app.NewService(app.WithDeps(app.Deps{Journal: w.J, Codec: w.Codec, Fold: reviewFold, Routes: app.DemoRoutes{}, DomainClock: reviewClock{w},
		Now: func() time.Time { return w.Now }}), app.WithConfig(app.Config{DomainBuild: "streebog256:" + strings.Repeat("0", 64), Partitions: 1}))
	item := "FL:0011"
	insp := nctest.Record(catalog.InspectionResultRecorded, item, nctest.T0.Add(time.Hour), map[string]any{"method": "camera", "phase": "after_operation",
		"outcome": "no_defect_indicated", "processing_state": "completed", "step_key": "welding.kt3_camera"})
	dec := nctest.Record(catalog.DecisionPresentationResolved, item, nctest.T0.Add(2*time.Hour), map[string]any{"step_key": "welding.zt3_acceptance",
		"closing_point": "ZT-3", "resolution": "accept", "presentation_no": 1, "method_event_ids": []string{insp.Entry.EventID},
		"reason": map[string]any{"text": "журнал режима ИС-2 недоступен; принято по камере"}})
	w.Add(nctest.Run(item, nctest.T0, "RUN-11", "op-7"), insp,
		nctest.Record(catalog.ItemPresentationRecorded, item, nctest.T0.Add(90*time.Minute), map[string]any{"step_key": "welding.zt3_acceptance",
			"presentation_no": 1, "presented_to": "qc", "presented_by": "master-1"}), dec)
	w.Settle()
	late := nctest.Record(catalog.EquipmentDeviationDetected, item, nctest.T0.Add(20*time.Minute), map[string]any{"equipment_id": "WELD-1",
		"deviation_kind": "parameter_out_of_range", "started_at": nctest.T0.Add(20 * time.Minute).Format(time.RFC3339), "parameter": "current_a",
		"value":    map[string]any{"value": 176, "scale": 0, "unit": "A"},
		"setpoint": map[string]any{"lower": map[string]any{"value": 150, "scale": 0, "unit": "A"}, "upper": map[string]any{"value": 170, "scale": 0, "unit": "A"}}})
	w.Add(late)
	w.Settle()

	ctx := nctest.As("qc-2", "quality_inspector")
	pv, err := svc.Presentation(ctx, item, platform.Moment{})
	if err != nil || pv.Review == nil {
		t.Fatalf("пересмотр: %+v %v", pv, err)
	}
	rv := pv.Review
	if rv.Decision.EventID != dec.Entry.EventID || !strings.Contains(rv.Decision.Summary, "при неполных данных") || !strings.Contains(rv.Decision.Summary, "ИС-2") {
		t.Fatalf("прежнее решение: %+v", rv.Decision)
	}
	absent := slices.IndexFunc(rv.KnownAtDecision, func(r app.NCRecordRef) bool { return r.Absent })
	if absent < 0 || rv.KnownAtDecision[absent].EventID != late.Entry.EventID || len(rv.KnownAtDecision) != 2 {
		t.Fatalf("основание при подписи: %+v", rv.KnownAtDecision)
	}
	joined := strings.Join(rv.WhySignificant, " | ")
	if !strings.Contains(joined, "176 A вне уставки 150…170 A") || !strings.Contains(joined, "RUN-11") {
		t.Fatalf("почему значимо: %v", rv.WhySignificant)
	}
	if pv.Recommendation == nil || pv.Recommendation.Outcome != "revoked" {
		t.Fatalf("рекомендация: %+v", pv.Recommendation)
	}
	if len(pv.Actions) != 2 || *pv.Actions[0].Outcome != "revoked" || !pv.Actions[0].Allowed || len(pv.Actions[0].Consequences) < 4 || len(pv.Actions[0].TechnicalConsequences) != 2 ||
		pv.Actions[0].Operation != "nonconformity.presentation.review" {
		t.Fatalf("решения: %+v", pv.Actions)
	}

	// Без основания — отказ; отзыв — новая запись поверх прежней.
	if _, err := svc.ReviewPresentation(ctx, item, app.ReviewPresentation{CommandHeader: hdr(pv.BasisSeq), ReviewedEventID: dec.Entry.EventID,
		Outcome: "revoked", Reason: reason(" ")}); err == nil {
		t.Fatal("пересмотр без основания принят")
	}
	r, err := svc.ReviewPresentation(ctx, item, app.ReviewPresentation{CommandHeader: hdr(pv.BasisSeq), ReviewedEventID: dec.Entry.EventID,
		Outcome: "revoked", Reason: reason("Ток 176 А вне уставки на сварке шва до приёмки")})
	if err != nil || r.Seq == 0 {
		t.Fatalf("отзыв: %+v %v", r, err)
	}
	w.Settle()
	if _, err := svc.Presentation(ctx, item, platform.Moment{}); err == nil {
		t.Fatal("после пересмотра открытого пересмотра нет")
	}
	q, _ := svc.Queue(context.Background(), app.QueueFilter{Kind: "review"}, platform.Moment{}, platform.Page{})
	if len(q.Items) != 0 {
		t.Fatalf("строка пересмотра осталась: %+v", q.Items)
	}
	if _, err := svc.ReviewPresentation(ctx, item, app.ReviewPresentation{CommandHeader: hdr(0), ReviewedEventID: dec.Entry.EventID,
		Outcome: "revoked", Reason: reason("ещё раз")}); err == nil {
		t.Fatal("повторный отзыв принят")
	}
}
