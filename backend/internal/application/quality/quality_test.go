package quality_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"testing"
	"time"

	engineapp "ant/internal/application/engine"
	"ant/internal/application/platform"
	app "ant/internal/application/quality"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/statuses"
	"ant/internal/domain/engine"
	"ant/internal/domain/kernel"
	"ant/internal/domain/quality"
)

// Сквозные проверки модуля quality на настоящем нормативном слое
// (normative/ репозитория) и настоящей свёртке движка (domain/engine.Fold).

var t0 = time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)

func repoEnv(t *testing.T, passportLevel int) quality.Env {
	t.Helper()
	env, err := app.EnvFromFS(os.DirFS("../../../.."), "norm-v1")
	if err != nil {
		t.Fatal(err)
	}
	if passportLevel >= 0 {
		env.Passports = quality.PassportsFrom([]kernel.Record{rec(t, 0, catalog.AnalyzerPassportAdmitted, "analyzer_passport:PP-KT3", map[string]any{
			"passport_id": "PP-KT3", "stage": "active", "trust_level": passportLevel, "recipe_ref": "kt3-weld@1", "document_id": "DOC-PP-KT3",
			"versions": map[string]any{"recipe_ref": "kt3-weld@1", "analyzer_version": "vqc-weld 2.3.1", "contract_version": "1.0"}})})
	}
	return env
}

func rec(t *testing.T, n int, typ catalog.Type, stream string, data any) kernel.Record {
	t.Helper()
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	info, _ := catalog.Lookup(typ)
	item := ""
	if len(stream) > 5 && stream[:5] == "item:" {
		item = stream[5:]
	}
	at := t0.Add(time.Duration(n) * time.Minute)
	return kernel.Record{Seq: int64(n + 1), EventID: fmt.Sprintf("00000000-0000-7000-8000-%012d", n+1), Type: typ, SchemaVersion: 1,
		Kind: info.Kind, ItemID: item, Stream: stream, OccurredAt: at, ReceivedAt: at, Data: raw}
}

type input struct {
	t    *testing.T
	item string
	recs []kernel.Record
}

func (in *input) add(typ catalog.Type, data map[string]any) kernel.Record {
	r := rec(in.t, len(in.recs)+1, typ, "item:"+in.item, data)
	in.recs = append(in.recs, r)
	return r
}

func (in *input) weld() {
	in.add(catalog.OperationRunStarted, map[string]any{"operation_run_id": "RUN-W1", "operation_code": "WELD", "step_key": "welding.weld", "operator_id": "O17"})
}

func (in *input) camera(outcome string, q, c int, defects ...map[string]any) kernel.Record {
	d := map[string]any{"method": "camera", "phase": "after_operation", "outcome": outcome, "processing_state": "completed",
		"step_key": "welding.kt3_camera", "inspection_point": "KT-3", "zone_ids": []string{"W-1"}, "observation_quality_bp": q, "analyzer_confidence_bp": c,
		"stages":   []any{map[string]any{"stage": "localize", "version": "vqc-weld 2.3.1", "confidence_bp": c}, map[string]any{"stage": "classify", "version": "vqc-weld 2.3.1", "confidence_bp": c}},
		"versions": map[string]any{"recipe_ref": "kt3-weld@1", "analyzer_version": "vqc-weld 2.3.1", "contract_version": "1.0"}}
	if len(defects) > 0 {
		d["defects"] = defects
	}
	return in.add(catalog.InspectionResultRecorded, d)
}

func (in *input) xray(outcome string, defects ...map[string]any) kernel.Record {
	d := map[string]any{"method": "radiography", "phase": "after_operation", "outcome": outcome, "processing_state": "completed",
		"step_key": "welding.kt3_radiography", "inspection_point": "KT-3", "zone_ids": []string{"W-1"}}
	if len(defects) > 0 {
		d["defects"] = defects
	}
	return in.add(catalog.InspectionResultRecorded, d)
}

func sign(code, zone, sev string, conf int) map[string]any {
	d := map[string]any{"zone_id": zone, "severity": sev}
	if code != "" {
		d["defect_type_code"] = code
	}
	if conf > 0 {
		d["stage_confidence_bp"] = conf
	}
	return d
}

func foldQ(env quality.Env, recs []kernel.Record) (engine.Snapshot, []kernel.Reaction) {
	return engine.Fold(engine.Bundle{Quality: env}, recs)
}

func only(rs []kernel.Reaction, t catalog.Type) []kernel.Reaction {
	out := []kernel.Reaction{}
	for _, r := range rs {
		if r.Type == t {
			out = append(out, r)
		}
	}
	return out
}

func TestEnvFromRepoNormative(t *testing.T) {
	env := repoEnv(t, -1)
	if env.ReactionMap.ID != "flange-reactions" || len(env.ReactionMap.Rules) < 10 || len(env.Classifier.DefectTypes) < 15 {
		t.Fatalf("нормативный слой: карта %q, правил %d, видов %d", env.ReactionMap.ID, len(env.ReactionMap.Rules), len(env.Classifier.DefectTypes))
	}
	var kt3 quality.StepSpec
	for _, s := range env.Steps {
		if s.StepKey == "welding.kt3_camera" {
			kt3 = s
		}
	}
	if kt3.InspectionPoint != "KT-3" || len(kt3.Inspections) != 1 || kt3.Inspections[0].ObservationQualityMinBP != 6000 ||
		!slices.Contains(kt3.Inspections[0].Coverage, "W-BURNTHRU") || len(kt3.Requirements) != 1 {
		t.Fatalf("КТ-3 камера: %+v", kt3)
	}
	if len(env.Zones) == 0 {
		t.Fatal("нет зон изделия")
	}
}

// Ожидаемое эпика 20: плохой кадр без признаков — «оценка невозможна» и
// повторный контроль, а не «годно» (FR-36, кейс §4.5).
func TestBadFrameUnableAndRecheck(t *testing.T) {
	in := &input{t: t, item: "ENT01:F-017"}
	in.weld()
	in.camera("no_defect_indicated", 2500, 9900)
	s, rs := foldQ(repoEnv(t, 3), in.recs)
	if s.Quality.Axis != statuses.QualityUnableToAssess {
		t.Fatalf("ось: %s", s.Quality.Axis)
	}
	sig := only(rs, catalog.QualitySignalRaised)
	if len(sig) != 1 {
		t.Fatalf("сигналы: %+v", sig)
	}
	var d map[string]any
	b, _ := json.Marshal(sig[0].Data)
	_ = json.Unmarshal(b, &d)
	if d["reaction_outcome"] != "manual_review" || d["reaction_map_ref"] != "flange-reactions@1#R-02" {
		t.Fatalf("сигнал: %s", b)
	}
	if !hasTask(s.Quality, "recheck") {
		t.Fatalf("нет задачи повторного контроля: %+v", s.Quality.Requests)
	}
	if len(only(rs, catalog.QualityInspectionAutoPassed)) != 0 {
		t.Fatal("плохой кадр пропущен к следующему контролю")
	}
}

// Ожидаемое эпика 20: низкая уверенность при критической тяжести — блок (FR-48).
func TestLowConfidenceCriticalIsHold(t *testing.T) {
	in := &input{t: t, item: "ENT01:F-021"}
	in.weld()
	in.xray("defect_indicated", sign("W-LOF", "W-1.U3", "critical", 1500))
	s, _ := foldQ(repoEnv(t, 3), in.recs)
	sg := s.Quality.Signals[0]
	if sg.Assessment.RuleID != "R-03" || sg.Assessment.Outcome != quality.ReactIsolate || sg.Assessment.Containment != statuses.ContainmentItemHold || !sg.Assessment.DraftNC {
		t.Fatalf("критическая тяжесть при уверенности 0,15: %+v", sg.Assessment)
	}
}

// Ожидаемое эпика 20: сигнал без требования КД — вопрос технологу (FR-48).
func TestNoRequirementQuestion(t *testing.T) {
	in := &input{t: t, item: "ENT01:F-019"}
	in.weld()
	in.camera("defect_indicated", 9000, 8000, sign("W-SOOT", "W-1.U4", "minor", 8000))
	s, rs := foldQ(repoEnv(t, 3), in.recs)
	sg := s.Quality.Signals[0]
	if sg.TypeKnown || sg.Assessment.RuleID != "R-06" || sg.Assessment.Outcome != quality.ReactQuestion || sg.Assessment.DraftNC {
		t.Fatalf("вид без требования: %+v", sg)
	}
	if s.Quality.Axis == statuses.QualityNonconforming {
		t.Fatal("вопрос технологу — не брак")
	}
	if len(only(rs, catalog.QualitySignalRaised)) != 1 {
		t.Fatal("вопрос технологу — всё же сигнал")
	}
}

// Ожидаемое эпика 20: повторные наблюдения не растят показатели (FR-37, AD-45).
func TestRepeatedObservationsKeepMetrics(t *testing.T) {
	in := &input{t: t, item: "ENT01:F-017"}
	in.weld()
	in.camera("defect_indicated", 9000, 8600, sign("W-BURNTHRU", "W-1.U2", "critical", 8600))
	s1, _ := foldQ(repoEnv(t, 3), in.recs)
	in.xray("defect_indicated", sign("W-BURNTHRU", "W-1.U2", "critical", 0))
	in.camera("defect_indicated", 9000, 9000, sign("W-BURNTHRU", "W-1.U2", "critical", 9000))
	s2, rs := foldQ(repoEnv(t, 3), in.recs)
	m1, m2 := metrics(t, s1), metrics(t, s2)
	if m1[app.MetricDefects] != 1 || m2[app.MetricDefects] != 1 || m2[app.MetricItemsWithDefect] != 1 || m2[app.MetricSignals] != 1 {
		t.Fatalf("показатели: до %v, после %v", m1, m2)
	}
	if m2[app.MetricObservations] != 3 {
		t.Fatalf("наблюдения считаются раздельно: %v", m2)
	}
	if n := len(only(rs, catalog.QualityObservationLinked)); n != 2 {
		t.Fatalf("связанных наблюдений: %d", n)
	}
}

// NFR-DET-1: свёртка детерминирована и не зависит от порядка поступления.
func TestFoldDeterministic(t *testing.T) {
	in := &input{t: t, item: "ENT01:F-023"}
	in.weld()
	in.camera("defect_indicated", 9000, 6500, sign("W-UNDERCUT", "W-1.U1", "major", 6500))
	in.xray("unable_to_assess")
	in.add(catalog.ItemPresentationRecorded, map[string]any{"step_key": "welding.zt3_acceptance", "presentation_no": 1, "presented_to": "qc", "presented_by": "K1"})
	env := repoEnv(t, 3)
	s1, r1 := foldQ(env, in.recs)
	rev := slices.Clone(in.recs)
	slices.Reverse(rev)
	s2, r2 := foldQ(env, rev)
	h1, _ := engine.StateHash(s1, r1)
	h2, _ := engine.StateHash(s2, r2)
	if h1 != h2 || len(r1) == 0 {
		t.Fatalf("хеши: %s ≠ %s", h1, h2)
	}
}

func metrics(t *testing.T, s engine.Snapshot) map[string]int64 {
	t.Helper()
	rows, err := app.Contributions("x", s, nil)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]int64{}
	for _, r := range rows {
		out[r.Metric] += r.Value
	}
	return out
}

func hasTask(s quality.State, kind string) bool {
	for _, r := range s.Requests {
		if r.Kind == quality.RequestTask && r.TaskKind == kind {
			return true
		}
	}
	return false
}

// memStore — проекции в памяти: как engine.projections после эффектов Append.
type memStore map[string]json.RawMessage

func (m memStore) Get(_ context.Context, name, key string) (json.RawMessage, bool, error) {
	v, ok := m[name+"\x1f"+key]
	return v, ok, nil
}

// apply — эффекты проекций изделия и глобальной проекции-перечня.
func (m memStore) apply(t *testing.T, reg *engineapp.Registry, item string, s engine.Snapshot, rs []kernel.Reaction, in []kernel.Record) {
	t.Helper()
	effs, err := reg.ItemEffects(item, s, rs, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range effs {
		if p, ok := e.(engineapp.ProjectionPut); ok {
			m[p.Name+"\x1f"+p.Key] = p.Value
		}
	}
	for _, g := range reg.Globals() {
		for _, r := range in {
			for _, k := range g.Keys(r) {
				v, err := g.Step(k, m[g.Name+"\x1f"+k], r)
				if err != nil {
					t.Fatal(err)
				}
				m[g.Name+"\x1f"+k] = v
			}
		}
	}
}

// Живая реализация операций quality над проекциями (AD-36, AD-45).
func TestServiceOverProjections(t *testing.T) {
	env := repoEnv(t, 3)
	reg := engineapp.NewRegistry()
	if err := app.Register(reg); err != nil {
		t.Fatal(err)
	}
	store := memStore{}
	a := &input{t: t, item: "ENT01:F-017"}
	a.weld()
	a.camera("defect_indicated", 9000, 8600, sign("W-BURNTHRU", "W-1.U2", "critical", 8600))
	a.xray("defect_indicated", sign("W-BURNTHRU", "W-1.U2", "critical", 0))
	s, rs := foldQ(env, a.recs)
	store.apply(t, reg, a.item, s, rs, a.recs)
	b := &input{t: t, item: "ENT01:F-018"}
	b.weld()
	b.camera("no_defect_indicated", 2000, 9900)
	s, rs = foldQ(env, b.recs)
	store.apply(t, reg, b.item, s, rs, b.recs)

	svc := app.NewService(app.WithStore(store), app.WithEnv(env))
	ctx := context.Background()
	m := platform.Moment{Axis: platform.AxisOccurred}
	sigs, err := svc.Signals(ctx, app.SignalFilter{}, m, platform.Page{})
	if err != nil || len(sigs.Items) != 2 {
		t.Fatalf("сигналы: %+v %v", sigs, err)
	}
	var burn app.QualitySignal
	for _, x := range sigs.Items {
		if x.ItemID == "ENT01:F-017" {
			burn = x
		}
	}
	if burn.ReactionOutcome != "isolate" || len(burn.Stages) != 0 && burn.Stages[0].Stage == "" || len(burn.ObservationIDs) != 2 {
		t.Fatalf("сигнал прожога: %+v", burn)
	}
	one, err := svc.Signal(ctx, burn.SignalID, m)
	if err != nil || one.SignalID != burn.SignalID {
		t.Fatalf("сигнал по id: %v", err)
	}
	defs, err := svc.Defects(ctx, "", m, platform.Page{})
	if err != nil || defs.DefectCount != 1 || defs.ItemsWithDefect != 1 || defs.Items[0].Observations != 2 {
		t.Fatalf("дефекты: %+v %v", defs, err)
	}
	cov, err := svc.Coverage(ctx, "ENT01:F-018", m)
	if err != nil || cov.QualityState != "unable_to_assess" || cov.Complete {
		t.Fatalf("полнота: %+v %v", cov, err)
	}
	ins, err := svc.Inspections(ctx, "ENT01:F-018", m, platform.Page{})
	if err != nil || len(ins.Items) != 1 || ins.Items[0].Outcome != "unable_to_assess" || ins.Items[0].ReportedOutcome != "no_defect_indicated" {
		t.Fatalf("результаты контроля: %+v %v", ins, err)
	}
	rm, err := svc.ReactionMap(ctx, m)
	if err != nil || rm.Ref != "flange-reactions@1" || len(rm.Rules) != len(env.ReactionMap.Rules) || rm.Rules[0].Trigger == "" {
		t.Fatalf("карта реакций: %+v %v", rm, err)
	}
	if _, err := app.NewService().Signals(ctx, app.SignalFilter{}, m, platform.Page{}); err == nil {
		t.Fatal("без хранилища — 501")
	}
}
