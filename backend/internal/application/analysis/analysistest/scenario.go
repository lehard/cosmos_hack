package analysistest

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	appanalysis "ant/internal/application/analysis"
	engineapp "ant/internal/application/engine"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
)

// Scenario — «станок сломался» через операции analysis в режиме live
// (FR-9, FR-58…FR-62, FR-135): одна и та же проверка на журнале в памяти и на
// своей БД. Область риска 34 → 13 → 6 по решениям технолога с основаниями,
// правило системы только расширяет, разбор обстоятельств на трёх дорожках,
// «подтверждённая» причина — только решением человека.
func Scenario(t *testing.T, w *World, store engineapp.ProjectionStore) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(w.Story(ctx))
	must(w.Settle(ctx))
	svc := w.Service(store, At(23, 11, 45))
	tec := Principal(ctx, "TEC-01")
	m := platform.Moment{Axis: platform.AxisOccurred}

	incs, err := svc.Incidents(ctx, m, platform.Page{})
	must(err)
	if len(incs.Items) != 1 {
		t.Fatalf("инцидентов %d, ждали 1", len(incs.Items))
	}
	inc := incs.Items[0]
	if inc.Size != 34 || inc.InitialSize != 34 || inc.ScopeVersion != 1 || inc.CommonFactor == nil || inc.CommonFactor.Value != "IS-2" {
		t.Fatalf("инцидент: %+v", inc)
	}
	rs, err := svc.RiskScope(ctx, inc.IncidentID, m)
	must(err)
	if len(rs.Items) != 34 || rs.LastKnownGood == nil || !strings.HasPrefix(rs.LastKnownGood.Label, "F-006") || rs.Window == nil {
		t.Fatalf("область v1: %d изделий, отсчёт %+v", len(rs.Items), rs.LastKnownGood)
	}
	src := map[string]Weld{}
	for _, x := range StoryWelds() {
		src[x.Item] = x
	}
	pick := func(rs appanalysis.RiskScope, pred func(Weld) bool) []string {
		var out []string
		for _, it := range rs.Items {
			if it.Known != "excluded" && pred(src[it.ItemID]) {
				out = append(out, it.ItemID)
			}
		}
		return out
	}
	narrow := func(rs appanalysis.RiskScope, items, evidence []string, reason string) (platform.Receipt, error) {
		return svc.NarrowScope(tec, rs.IncidentID, appanalysis.ChangeScope{CommandHeader: header(w, rs.BasisSeq),
			ItemIDs: items, EvidenceEventIDs: evidence, Reason: appanalysis.Reason{Text: reason}})
	}

	// Сужение без основания — отказ гарда incident.basis_required (FR-61).
	is1 := pick(rs, func(x Weld) bool { return x.Src == "IS-1" })
	if _, err := narrow(rs, is1, nil, "без доказательств"); code(err) != errcodes.IncidentBasisRequired {
		t.Fatalf("сужение без основания: %v", err)
	}

	// v2: журнал ИС-1 непрерывный и в уставке — ИС-1 исключается.
	logIS1, err := w.Fact(ctx, catalog.EquipmentCycleSummarized, "", "equipment:IS-1", At(23, 11, 40), map[string]any{"equipment_id": "IS-1",
		"window_start": "2026-09-21T12:00:00.000Z", "window_end": "2026-09-23T08:00:00.000Z",
		"parameters": []map[string]any{{"parameter": "current", "out_of_setpoint_ms": 0}}})
	must(err)
	rc, err := narrow(rs, is1, []string{logIS1}, "Журнал ИС-1 непрерывный и в уставке; общий фактор — ИС-2")
	must(err)
	if rc.Seq == 0 || len(rc.EventIDs) != 1 {
		t.Fatalf("квитанция: %+v", rc)
	}
	must(w.Settle(ctx))
	rsV2, err := svc.RiskScope(ctx, inc.IncidentID, m)
	must(err)

	// v3: опоздавший журнал ИС-2 — ток вне уставки с Вт 10:20.
	late, err := w.Fact(ctx, catalog.EquipmentDeviationDetected, "", "equipment:IS-2", At(22, 10, 20), map[string]any{"equipment_id": "IS-2",
		"deviation_kind": "out_of_setpoint", "started_at": "2026-09-22T07:20:00.000Z", "parameter": "current",
		"value": map[string]any{"value": 176, "unit": "A", "scale": 0}})
	must(err)
	early := pick(rsV2, func(x Weld) bool { return x.Src == "IS-2" && x.To.Before(At(22, 10, 20)) })
	_, err = narrow(rsV2, early, []string{late}, "Опоздавший журнал ИС-2: ток впервые вне уставки во Вт 10:20; сварки до этого — в уставке")
	must(err)
	must(w.Settle(ctx))
	rsV3, err := svc.RiskScope(ctx, inc.IncidentID, m)
	must(err)

	var sizes []int
	for _, v := range rsV3.Versions {
		sizes = append(sizes, v.Size)
		if v.Reason == nil || v.Reason.Text == "" || len(v.EvidenceEventIDs) == 0 || v.RecordedAt.IsZero() {
			t.Fatalf("версия без основания: %+v", v)
		}
		if v.Change == "narrowed" && (v.Author == nil || *v.Author != "TEC-01") {
			t.Fatalf("сужение без автора: %+v", v)
		}
	}
	if !slices.Equal(sizes, []int{34, 13, 6}) {
		t.Fatalf("область тает %v, ждали 34 → 13 → 6", sizes)
	}
	var left []string
	for _, it := range rsV3.Items {
		if it.Known != "excluded" {
			left = append(left, it.Label)
		}
	}
	if !slices.Equal(left, []string{"F-015", "F-017", "F-019", "F-021", "F-023", "F-025"}) {
		t.Fatalf("итог: %v", left)
	}
	if b := rsV3.Versions[2].Breakdown; b.InProduction+b.MovedOn+b.Assembled+b.Shipped != 6 {
		t.Fatalf("разбивка: %+v", b)
	}

	// Устаревший basis_seq — 409 journal.stale_state (AD-39).
	if _, err := narrow(rsV2, []string{"ENT01:F-015"}, []string{late}, "повтор по старым данным"); code(err) != errcodes.JournalStaleState {
		t.Fatalf("устаревший basis_seq: %v", err)
	}

	// Правило системы расширяет область (новая сварка на ИС-2), не исключая.
	must(w.WeldRun(ctx, W("F-038", "IS-2", "W22", 23, 12, 30, 13, 0)))
	must(w.Settle(ctx))
	rsV4, err := svc.RiskScope(ctx, inc.IncidentID, m)
	must(err)
	last := rsV4.Versions[len(rsV4.Versions)-1]
	if last.Size != 7 || last.Change != "computed" || last.Author != nil {
		t.Fatalf("расширение правилом: %+v", last)
	}

	// Разбор обстоятельств Ф-017: три дорожки на одной шкале, окно, отклонение режима.
	_, err = w.Fact(ctx, catalog.EquipmentDeviationDetected, "", "equipment:IS-2", At(23, 10, 45), map[string]any{"equipment_id": "IS-2",
		"deviation_kind": "out_of_setpoint", "started_at": "2026-09-23T07:45:00.000Z", "ended_at": "2026-09-23T07:55:00.000Z",
		"parameter": "current", "value": map[string]any{"value": 176, "unit": "A", "scale": 0}})
	must(err)
	ci, err := svc.Circumstances(ctx, "NC-01", m)
	must(err)
	lanes := map[string]int{}
	for _, r := range ci.Records {
		lanes[r.Lane]++
	}
	if ci.Window == nil || ci.Operation == nil || lanes["item"] < 2 || lanes["person"] < 2 || lanes["equipment"] < 1 {
		t.Fatalf("разбор: окно %+v, операция %+v, дорожки %v", ci.Window, ci.Operation, lanes)
	}
	hs, err := svc.Hypotheses(ctx, "NC-01", m)
	must(err)
	for _, h := range hs.Hypotheses {
		if h.Status != "proposed_by_system" {
			t.Fatalf("система поставила статус %q гипотезе %s", h.Status, h.HypothesisID)
		}
	}
	eqID := hypothesis(hs, "equipment")
	if eqID == "" {
		t.Fatalf("нет гипотезы «оборудование»: %+v", hs.Hypotheses)
	}

	// Ошибку исполнителя без письменного объяснения подтвердить нельзя (FR-59).
	conclude := func(category string) error {
		_, err := svc.ConcludeCause(tec, inc.IncidentID, appanalysis.ConcludeCause{CommandHeader: header(w, 0), NCIDs: []string{"NC-01"},
			Conclusion: "confirmed", Category: category, Verification: "Контрольный образец на ИС-2", Reason: appanalysis.Reason{Text: "Решение комиссии"}})
		return err
	}
	if err := conclude("performer"); code(err) != errcodes.IncidentExplanationRequired {
		t.Fatalf("ошибка исполнителя без объяснения: %v", err)
	}
	must(conclude("equipment"))
	if _, err := svc.RequestMeasurement(tec, "NC-01", appanalysis.RequestMeasurement{CommandHeader: header(w, 0), HypothesisID: eqID, What: "Ток сварки на контрольном образце"}); err != nil {
		t.Fatal(err)
	}
	if perf := hypothesis(hs, "performer"); perf != "" {
		_, err := svc.RejectHypothesis(tec, "NC-01", appanalysis.RejectHypothesis{CommandHeader: header(w, 0), HypothesisID: perf,
			Reason: appanalysis.Reason{Text: "Перестановка по инструкции, режим источника вне уставки"}})
		must(err)
	}
	must(w.Settle(ctx))
	hs, err = svc.Hypotheses(ctx, "NC-01", m)
	must(err)
	for _, h := range hs.Hypotheses {
		want := "proposed_by_system"
		switch h.Category {
		case "equipment":
			want = "confirmed"
		case "performer":
			want = "rejected"
		}
		if h.Status != want {
			t.Fatalf("гипотеза %s: статус %q, ждали %q", h.HypothesisID, h.Status, want)
		}
	}

	// Общие факторы группы и группы несоответствий (FR-135).
	groups, err := svc.Groups(ctx, m)
	must(err)
	if len(groups.Items) != 1 || groups.Items[0].NCCount != 1 || groups.Items[0].Investigation != "cause_confirmed" {
		t.Fatalf("группы: %+v", groups.Items)
	}
	cf, err := svc.CommonFactors(ctx, groups.Items[0].GroupKey, m)
	must(err)
	if len(cf.Rows) == 0 || cf.Rows[0].Matches != 1 || cf.Rows[0].Value == nil {
		t.Fatalf("общие факторы: %+v", cf.Rows)
	}
}

func hypothesis(hs appanalysis.Hypotheses, category string) string {
	for _, h := range hs.Hypotheses {
		if h.Category == category {
			return h.HypothesisID
		}
	}
	return ""
}

func header(w *World, basis int64) platform.CommandHeader {
	return platform.CommandHeader{CommandID: w.ID("command"), BasisSeq: basis}
}

// code — код ошибки порта.
func code(err error) errcodes.Code {
	var pe *platform.Error
	if errors.As(err, &pe) {
		return pe.Code
	}
	if pe, ok := platform.AsError(err); ok {
		return pe.Code
	}
	return ""
}
