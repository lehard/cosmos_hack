package vision

import (
	"encoding/json"
	"testing"

	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
)

// inspection — наблюдение камеры КТ-3 с качеством кадра q в прогоне run.
func inspection(b *builder, run string, q int, outcome string) kernel.Record {
	r := b.add(catalog.InspectionResultRecorded, map[string]any{"method": "camera", "phase": "after_operation", "outcome": outcome,
		"processing_state": "completed", "observation_quality_bp": q, "analyzer_confidence_bp": 9100, "versions": full().Contract(),
		"zone_ids": []string{"W-1.U3"}, "inspection_point": "KT-3"})
	r.RunID, r.ItemID = run, "ENT01:F-"+run
	b.out[len(b.out)-1] = r
	return r
}

func feed(w Watch, rs ...kernel.Record) (Watch, []kernel.Reaction) {
	var out []kernel.Reaction
	for _, r := range rs {
		var re []kernel.Reaction
		w, re = w.Step(r)
		out = append(out, re...)
	}
	return w, out
}

// FR-101: «свет изменился» — три кадра подряд хуже 0,70, анализатор отвечает —
// дрейф → приостановка в прогоне; другой прогон и живая работа не затронуты.
func TestRollbackDrift(t *testing.T) {
	b := &builder{}
	adm := b.add(catalog.AnalyzerPassportAdmitted, admitted("AP-1", 3, nil))
	w, _ := UnmarshalState(nil)
	w, out := feed(w, adm,
		inspection(b, "r1", 5500, "no_defect_indicated"),
		inspection(b, "r1", 9000, "no_defect_indicated"), // хороший кадр рвёт серию
		inspection(b, "r1", 5500, "no_defect_indicated"),
		inspection(b, "r2", 5500, "no_defect_indicated"),
		inspection(b, "r1", 3000, "unable_to_assess"), // честный отказ — не дрейф
		inspection(b, "r1", 5400, "no_defect_indicated"))
	if len(out) != 0 {
		t.Fatalf("рано: %d приостановок", len(out))
	}
	last := inspection(b, "r1", 5300, "no_defect_indicated")
	w, out = feed(w, last)
	if len(out) != 1 {
		t.Fatalf("ждали одну приостановку, получили %d", len(out))
	}
	re := out[0]
	d, err := kernel.Decode[ev.AnalyzerPassportSuspendedV1](kernel.Record{Type: re.Type, Data: mustJSON(t, re.Data)})
	if err != nil {
		t.Fatal(err)
	}
	if d.Trigger != TriggerDrift || d.Fallback != FallbackManual || len(d.Basis) != DriftWindow || d.Note == nil {
		t.Fatalf("приостановка: %+v", d)
	}
	if re.Slot.Subject != Stream("AP-1") || re.AutomationMode != 2 {
		t.Fatalf("слот/режим: %+v", re)
	}
	// Ещё кадры в приостановке — второй записи нет; r2 и живая работа — свои окна.
	if _, again := feed(w, inspection(b, "r1", 5000, "no_defect_indicated")); len(again) != 0 {
		t.Fatal("повторная приостановка")
	}
	p := w.find("AP-1")
	if rw, _ := p.RunOf("r2"); rw.Suspended || len(rw.Recent) != 1 {
		t.Fatalf("прогон r2 затронут: %+v", rw)
	}
	if rw, _ := p.RunOf(""); rw.Suspended {
		t.Fatal("живая работа затронута")
	}
	// Возврат начальником ОТК в прогоне — окно заново; следующая приостановка — новый слот.
	susp := b.add(catalog.AnalyzerPassportSuspended, map[string]any{"passport_id": "AP-1", "trigger": "drift", "fallback": "manual_control", "basis": []string{}})
	susp.RunID, susp.EventID = "r1", re.ID(1)
	rein := b.add(catalog.AnalyzerPassportReinstated, map[string]any{"passport_id": "AP-1", "suspension_event_id": re.ID(1), "reason": map[string]any{"text": "свет вернули"}})
	rein.RunID = "r1"
	w, _ = feed(w, susp, rein)
	_, out = feed(w, inspection(b, "r1", 5000, "no_defect_indicated"), inspection(b, "r1", 5000, "no_defect_indicated"), inspection(b, "r1", 5000, "no_defect_indicated"))
	if len(out) != 1 || out[0].ID(1) == re.ID(1) {
		t.Fatalf("после возврата: %d приостановок", len(out))
	}
}

// FR-100/101: пропуск брака той же версией при способном методе — откат к
// предыдущему допущенному паспорту; «метод не видит» — не ошибка модели.
func TestRollbackEscape(t *testing.T) {
	b := &builder{}
	old := b.add(catalog.AnalyzerPassportAdmitted, admitted("AP-0", 3, nil))
	adm := b.add(catalog.AnalyzerPassportAdmitted, admitted("AP-1", 3, map[string]any{"previous_passport_id": "AP-0"}))
	v := "vqc-weld 2.3.1"
	blind := b.add(catalog.QualityEscapeRecorded, map[string]any{"defect_id": "D-1", "missed_observation_event_ids": []string{}, "method_covers_defect": false, "analyzer_version": v})
	esc := b.add(catalog.QualityEscapeRecorded, map[string]any{"defect_id": "D-2", "missed_observation_event_ids": []string{"00000000-0000-7000-8000-000000000099"}, "method_covers_defect": true, "analyzer_version": v})
	w, _ := UnmarshalState(nil)
	_, out := feed(w, old, adm, blind)
	if len(out) != 0 {
		t.Fatal("«метод не видит» не должен откатывать")
	}
	// Оба паспорта той же версии; AP-0 приостанавливается первым, AP-1 откатывается уже на ручной.
	_, out = feed(w, old, adm, esc)
	if len(out) != 2 {
		t.Fatalf("ждали приостановку обоих паспортов версии, получили %d", len(out))
	}
	for _, re := range out {
		d, _ := kernel.Decode[ev.AnalyzerPassportSuspendedV1](kernel.Record{Type: re.Type, Data: mustJSON(t, re.Data)})
		if d.Trigger != TriggerEscape {
			t.Fatalf("триггер %s", d.Trigger)
		}
	}
}

// Проверка эталонным набором не пройдена — откат к предыдущему паспорту.
func TestRollbackReferenceSet(t *testing.T) {
	b := &builder{}
	old := b.add(catalog.AnalyzerPassportAdmitted, map[string]any{"passport_id": "AP-0", "stage": "active", "trust_level": 2, "recipe_ref": "kt3-weld@1",
		"versions": Versions{RecipeRef: "kt3-weld@1", AnalyzerVersion: "vqc-weld 2.2.0", ContractVersion: "1.0"}.Contract(), "document_id": "DOC-0"})
	adm := b.add(catalog.AnalyzerPassportAdmitted, admitted("AP-1", 3, map[string]any{"previous_passport_id": "AP-0"}))
	chk := b.add(catalog.AnalyzerCheckRecorded, map[string]any{"passport_id": "AP-1", "check_kind": "reference_set", "passed": false, "versions": full().Contract()})
	w, _ := UnmarshalState(nil)
	_, out := feed(w, old, adm, chk)
	if len(out) != 1 {
		t.Fatalf("ждали приостановку, получили %d", len(out))
	}
	d, _ := kernel.Decode[ev.AnalyzerPassportSuspendedV1](kernel.Record{Type: out[0].Type, Data: mustJSON(t, out[0].Data)})
	if d.Trigger != TriggerReferenceSet || d.Fallback != FallbackPrevious || d.FallbackPassportID == nil || *d.FallbackPassportID != "AP-0" {
		t.Fatalf("приостановка: %+v", d)
	}
}

// FR-100: список на перепроверку — та же версия модели, та же зона, «признаков нет».
func TestRecheckList(t *testing.T) {
	b := &builder{}
	missed, _ := SeenFrom(inspection(b, "a", 9000, "no_defect_indicated"))
	c1, _ := SeenFrom(inspection(b, "b", 9000, "no_defect_indicated"))
	c2, _ := SeenFrom(inspection(b, "c", 9000, "defect_indicated"))
	other := inspection(b, "d", 9000, "no_defect_indicated")
	c3, _ := SeenFrom(other)
	c3.Versions.AnalyzerVersion = "vqc-weld 3.0.0"
	got := RecheckList([]Seen{missed}, []Seen{missed, c1, c2, c3}, missed.ItemID)
	if len(got) != 1 || got[0].ItemID != c1.ItemID {
		t.Fatalf("перепроверка: %+v", got)
	}
}

// FR-98: по давнему наблюдению — какими версиями и почему; откат позже не
// переписывает «проанализировано версией …».
func TestExplain(t *testing.T) {
	b := &builder{}
	adm := b.add(catalog.AnalyzerPassportAdmitted, admitted("AP-1", 3, nil))
	obs := inspection(b, "", 9100, "defect_indicated")
	susp := b.add(catalog.AnalyzerPassportSuspended, map[string]any{"passport_id": "AP-1", "trigger": "drift", "fallback": "manual_control", "basis": []string{}})
	o, ok := SeenFrom(obs)
	if !ok {
		t.Fatal("не наблюдение камеры")
	}
	a := Explain(o, Fold([]kernel.Record{adm, susp}))
	if a.PassportID != "AP-1" || a.StatusThen != StatusActive || a.StatusNow != StatusSuspended || a.LevelThen != 3 || !a.Suspicious {
		t.Fatalf("объяснение: %+v", a)
	}
	if len(a.Reasons) < 3 {
		t.Fatalf("причины: %v", a.Reasons)
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
