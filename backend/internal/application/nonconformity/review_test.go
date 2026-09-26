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
