package quality

import (
	"slices"
	"testing"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/normative"
	"ant/internal/contracts/statuses"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/process"
)

// FR-36, кейс §4.5: плохой кадр без признаков — «оценка невозможна» и
// повторный контроль, а не «годно»; повтор закрывает сигнал, но «годно»
// ставит только решение человека на точке предъявления.
func TestBadFrameIsUnableAndRecheckNotConforming(t *testing.T) {
	env := withPassport(testEnv(), 3)
	var b builder
	b.run("RUN-W1", "welding.weld")
	bad := b.camera(OutcomeNoDefect, 3000, 9800)
	s, outs := fold(env, b.out)

	o := s.Observations[0]
	if o.Outcome != OutcomeUnable || o.UnableReason != "poor_image" || o.Reported != OutcomeNoDefect || o.Reinterpreted != "poor_observation" {
		t.Fatalf("плохой кадр: %+v", o)
	}
	if s.Axis != statuses.QualityUnableToAssess {
		t.Fatalf("ось: %s, ожидали unable_to_assess", s.Axis)
	}
	sigs := reactionsOf(outs, catalog.QualitySignalRaised)
	requireN(t, "сигналы", sigs, 1)
	d := sigs[0].Data.(ev.QualitySignalRaisedV1)
	if d.ReactionOutcome != ReactManual || d.ReactionMapRef != "flange-reactions@1#R-02" || d.Severity != "unknown" {
		t.Fatalf("сигнал плохого кадра: %+v", d)
	}
	if !slices.Equal(sigs[0].Causes, []string{bad.EventID}) {
		t.Fatalf("причины: %v", sigs[0].Causes)
	}
	if p := point(s, "welding.kt3_camera"); p.Status != "unable" {
		t.Fatalf("точка КТ-3: %+v", p)
	}
	if !hasRequest(s, RequestTask, "recheck") || !hasContain(s, statuses.ContainmentAdditionalCheck) {
		t.Fatalf("нет задачи повторного контроля и доп. проверки: %+v", s.Requests)
	}
	for _, tc := range s.Types {
		if tc.Status == "no_defect" {
			t.Fatalf("вид %s засчитан проверенным по плохому кадру", tc.Code)
		}
	}

	// Повторный контроль: хороший кадр — сигнал закрыт, но не «годно».
	b.camera(OutcomeNoDefect, 9000, 9800)
	s, _ = fold(env, b.out)
	if s.Axis != statuses.QualityNotInspected {
		t.Fatalf("после повтора ось: %s, ожидали not_inspected (годность ставит человек)", s.Axis)
	}
	if sg := s.Signals[0]; sg.State != SignalResolved {
		t.Fatalf("сигнал «оценка невозможна» не закрыт повтором: %+v", sg)
	}
	// Решение контролёра на ЗТ-3 — «годно».
	b.xray(OutcomeNoDefect)
	dec := b.add(catalog.DecisionPresentationResolved, map[string]any{"step_key": "welding.zt3_acceptance", "closing_point": "ZT-3",
		"resolution": "accept", "presentation_no": 1, "method_event_ids": []string{bad.EventID}})
	s, outs = fold(env, b.out)
	if s.Axis != statuses.QualityConforming {
		t.Fatalf("после решения «годно» ось: %s", s.Axis)
	}
	it := outs[len(outs)-1].Intents
	requireN(t, "намерения", it, 1)
	p := it[0].Payload.(process.Presentation)
	if it[0].Target != process.Module || p.StepKey != "welding.zt3_acceptance" || p.Resolution != "accept" || p.DecisionEventID != dec.EventID {
		t.Fatalf("продвижение точки предъявления: %+v", it[0])
	}
}

// FR-36: «прервано» и «сбой» у источника — «оценка невозможна».
func TestAbortedProcessingIsUnable(t *testing.T) {
	var b builder
	b.add(catalog.InspectionResultRecorded, map[string]any{"method": "radiography", "phase": "after_operation", "outcome": "no_defect_indicated",
		"processing_state": "failed", "step_key": "welding.kt3_radiography"})
	s, _ := fold(testEnv(), b.out)
	if o := s.Observations[0]; o.Outcome != OutcomeUnable || o.UnableReason != "analyzer_failure" {
		t.Fatalf("сбой обработки: %+v", o)
	}
	if s.Axis != statuses.QualityUnableToAssess {
		t.Fatalf("ось: %s", s.Axis)
	}
}

// FR-48: низкая уверенность при критической тяжести ведёт к блоку.
func TestLowConfidenceCriticalBlocks(t *testing.T) {
	cases := []struct {
		name string
		add  func(b *builder)
	}{
		{"рентген, несплавление", func(b *builder) {
			b.xray(OutcomeDefect, defect("W-LOF", "W-1", "У3", "unknown", 1500))
		}},
		{"камера, уровень доверия 3, прожог", func(b *builder) {
			b.camera(OutcomeDefect, 9000, 1500, defect("W-BURNTHRU", "W-1", "У2", "unknown", 1500))
		}},
		{"камера, неизвестный вид, но источник: критический", func(b *builder) {
			b.camera(OutcomeDefect, 9000, 1200, defect("", "W-1", "У5", "critical", 1200))
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var b builder
			b.run("RUN-W1", "welding.weld")
			c.add(&b)
			s, outs := fold(withPassport(testEnv(), 3), b.out)
			requireN(t, "сигналы", s.Signals, 1)
			a := s.Signals[0].Assessment
			if a.Containment != statuses.ContainmentItemHold || !s.Signals[0].Raised {
				t.Fatalf("критическая тяжесть при уверенности 0,15 — не блок: %+v", a)
			}
			if !hasContain(s, statuses.ContainmentItemHold) {
				t.Fatalf("нет запроса блока: %+v", s.Requests)
			}
			if s.Axis != statuses.QualitySignal {
				t.Fatalf("ось: %s (сигнал — не брак)", s.Axis)
			}
			requireN(t, "реакции сигнала", reactionsOf(outs, catalog.QualitySignalRaised), 1)
		})
	}
}

// AD-29: анализатору с уровнем доверия 2 блок не делегирован — «под
// подозрением» и доп. контроль, решает человек; 0 — только запись.
func TestTrustLevelLimitsAnalyzer(t *testing.T) {
	var b builder
	b.run("RUN-W1", "welding.weld")
	b.camera(OutcomeDefect, 9000, 9500, defect("W-BURNTHRU", "W-1", "У2", "critical", 9500))
	s, outs := fold(withPassport(testEnv(), 2), b.out)
	a := s.Signals[0].Assessment
	if a.Outcome != ReactManual || a.Proposed != ReactIsolate || a.Containment != statuses.ContainmentAdditionalCheck || !slices.Contains(a.Limits, "trust_level_2") {
		t.Fatalf("уровень 2: %+v", a)
	}
	d := reactionsOf(outs, catalog.QualitySignalRaised)[0].Data.(ev.QualitySignalRaisedV1)
	if d.TrustLevel == nil || *d.TrustLevel != 2 {
		t.Fatalf("уровень доверия в сигнале: %+v", d)
	}

	s, outs = fold(testEnv(), b.out) // паспорта нет — уровень 0
	if o := s.Observations[0]; o.TrustLevel != 0 || o.TrustNote != "no_qualified_analyzer" {
		t.Fatalf("без паспорта: %+v", o)
	}
	if len(reactionsOf(outs, catalog.QualitySignalRaised)) != 0 || s.Axis != statuses.QualityNotInspected {
		t.Fatalf("уровень 0 — только запись, а реакция есть: ось %s", s.Axis)
	}
	if len(s.Defects) != 1 {
		t.Fatalf("наблюдение уровня 0 должно остаться в записи: %+v", s.Defects)
	}
	// Предъявление на ЗТ-3 без результата человека — «нет данных, контроль ручной».
	b.add(catalog.ItemPresentationRecorded, map[string]any{"step_key": "welding.zt3_acceptance", "presentation_no": 1, "presented_to": "qc", "presented_by": "K1"})
	s, _ = fold(testEnv(), b.out)
	if p := point(s, "welding.kt3_camera"); p.Status != "missing" || p.MissingReason != "point_manual_mode" {
		t.Fatalf("КТ-3 без допущенного анализатора: %+v", p)
	}

	// Уровень 1 — рекомендация контролёру задачей, без сигнала.
	s, outs = fold(withPassport(testEnv(), 1), b.out[:2])
	if len(reactionsOf(outs, catalog.QualitySignalRaised)) != 0 || !hasRequest(s, RequestTask, "decision_required") {
		t.Fatalf("уровень 1: %+v", s.Requests)
	}
}

// FR-48: сигнал без требования КД — вопрос технологу, а не брак.
func TestNoRequirementIsQuestionToTechnologist(t *testing.T) {
	var b builder
	b.run("RUN-W1", "welding.weld")
	b.xray(OutcomeDefect, defect("X-ODD-MARK", "W-1", "У4", "minor", 0))
	s, outs := fold(testEnv(), b.out)
	sg := s.Signals[0]
	if sg.TypeKnown || sg.Assessment.Outcome != ReactQuestion || sg.Assessment.DraftNC || sg.Assessment.RuleID != "R-06" {
		t.Fatalf("неизвестный вид: %+v", sg)
	}
	if hasRequest(s, RequestDraftNC, "") {
		t.Fatalf("вопрос технологу — не брак: черновика несоответствия быть не должно: %+v", s.Requests)
	}
	var task Request
	for _, r := range s.Requests {
		if r.Kind == RequestTask {
			task = r
		}
	}
	if task.RoleID != "technologist" {
		t.Fatalf("вопрос не технологу: %+v", task)
	}
	d := reactionsOf(outs, catalog.QualitySignalRaised)[0].Data.(ev.QualitySignalRaisedV1)
	if d.DefectTypeKnown == nil || *d.DefectTypeKnown || d.ReactionOutcome != ReactQuestion || d.RequirementRef != nil {
		t.Fatalf("данные сигнала: %+v", d)
	}
	// Известный вид в зоне, для вида которой он не описан, — тоже вопрос.
	var b2 builder
	b2.add(catalog.InspectionResultRecorded, map[string]any{"method": "visual_human", "phase": "other", "outcome": "defect_indicated",
		"processing_state": "completed", "defects": []any{defect("W-UNDERCUT", "F-FACE", "", "major", 0)}})
	s, _ = fold(testEnv(), b2.out)
	if s.Signals[0].Assessment.Outcome != ReactQuestion {
		t.Fatalf("подрез на торце фланца: %+v", s.Signals[0].Assessment)
	}
}

// FR-37: один физический дефект — ключ без вида; повторные наблюдения и
// уточнение вида не увеличивают показатели; новая операция — новый дефект.
func TestRepeatedObservationsDoNotMultiply(t *testing.T) {
	env := withPassport(testEnv(), 3)
	var b builder
	b.run("RUN-W1", "welding.weld")
	first := b.camera(OutcomeDefect, 9000, 7000, defect("", "W-1.U2", "40-80 мм", "major", 7000))
	second := b.camera(OutcomeDefect, 9000, 7200, defect("W-PORE-S", "W-1.U2", "40-80 мм", "major", 7200))
	third := b.xray(OutcomeDefect, defect("W-BURNTHRU", "W-1.U2", "", "critical", 0))
	s, outs := fold(env, b.out)
	requireN(t, "дефекты", s.Defects, 1)
	d := s.Defects[0]
	if d.TypeCode != "W-BURNTHRU" || d.Severity != "critical" || !slices.Equal(d.Observations, []string{first.EventID, second.EventID, third.EventID}) {
		t.Fatalf("дефект: %+v", d)
	}
	requireN(t, "сигналы", s.Signals, 1)
	requireN(t, "дефект установлен", reactionsOf(outs, catalog.QualityDefectIdentified), 1)
	requireN(t, "наблюдение связано", reactionsOf(outs, catalog.QualityObservationLinked), 2)
	if a := s.Signals[0].Assessment; a.Outcome != ReactIsolate {
		t.Fatalf("строже из наблюдений — изоляция: %+v", a)
	}

	// Повтор того же факта с исправлением (FR-122) — не новое наблюдение.
	b.add(catalog.InspectionResultRecorded, map[string]any{"method": "radiography", "phase": "after_operation", "outcome": "defect_indicated",
		"processing_state": "completed", "step_key": "welding.kt3_radiography", "defects": []any{defect("W-BURNTHRU", "W-1.U2", "", "critical", 0)}})
	b.out[len(b.out)-1].Corrects = third.EventID
	s, _ = fold(env, b.out)
	requireN(t, "дефекты после исправления", s.Defects, 1)
	if n := len(s.Defects[0].Observations); n != 3 {
		t.Fatalf("исправление заменяет наблюдение: %d", n)
	}

	// Повтор операции (FR-47) — то же место после новой операции — новый дефект.
	b.run("RUN-W2", "welding.weld")
	b.camera(OutcomeDefect, 9000, 7000, defect("W-PORE-S", "W-1.U2", "40-80 мм", "major", 7000))
	s, _ = fold(env, b.out)
	requireN(t, "дефекты после повтора операции", s.Defects, 2)
}

// FR-35: ожидаемый, но не пришедший результат — «нет данных» и задача, не «годно».
func TestCompletenessMissingIsNotConforming(t *testing.T) {
	var b builder
	b.run("RUN-W1", "welding.weld")
	b.xray(OutcomeNoDefect)
	pres := b.add(catalog.ItemPresentationRecorded, map[string]any{"step_key": "welding.zt3_acceptance", "presentation_no": 1, "presented_to": "qc", "presented_by": "K1"})
	s, outs := fold(withPassport(testEnv(), 3), b.out)
	p := point(s, "welding.kt3_camera")
	if p.Status != "missing" || p.MissingReason != "result_not_received" || p.EventID != pres.EventID {
		t.Fatalf("точка без результата: %+v", p)
	}
	miss := reactionsOf(outs, catalog.QualityInspectionMissing)
	requireN(t, "нет данных", miss, 1)
	if md := miss[0].Data.(ev.QualityInspectionMissingV1); md.StepKey != "welding.kt3_camera" || md.AfterOperationRunID == nil || *md.AfterOperationRunID != "RUN-W1" {
		t.Fatalf("нет данных: %+v", md)
	}
	if !hasRequest(s, RequestTask, "inspection_missing") {
		t.Fatalf("нет задачи: %+v", s.Requests)
	}
	if bl := PresentationBlockers(s, withPassport(testEnv(), 3), "welding.zt3_acceptance"); len(bl) != 1 || bl[0].Code != "inspection_missing" {
		t.Fatalf("блокирующие причины: %+v", bl)
	}
	// Решение «годно» при отсутствующем контроле не делает изделие годным.
	b.add(catalog.DecisionPresentationResolved, map[string]any{"step_key": "welding.zt3_acceptance", "closing_point": "ZT-3",
		"resolution": "accept", "presentation_no": 1, "method_event_ids": []string{}})
	s, _ = fold(withPassport(testEnv(), 3), b.out)
	if s.Axis != statuses.QualityNotInspected {
		t.Fatalf("ось при «нет данных»: %s", s.Axis)
	}
	// Пропуск проверки исполнителем — «нет данных» с причиной.
	var b2 builder
	b2.run("RUN-W1", "welding.weld")
	b2.add(catalog.OperatorCheckSkipped, map[string]any{"operator_id": "O17", "step_key": "welding.kt3_camera", "inspection_point": "KT-3"})
	s, _ = fold(testEnv(), b2.out)
	if p := point(s, "welding.kt3_camera"); p.Status != "missing" || p.MissingReason != "check_skipped" {
		t.Fatalf("пропуск проверки: %+v", p)
	}
}

// FR-48, AD-27, AD-29: уверенное «признаков нет» пропускается к следующему
// контролю только анализатором с уровнем доверия 4; «годно» не ставится.
func TestAutoPassOnlyAtTrustFour(t *testing.T) {
	var b builder
	b.run("RUN-W1", "welding.weld")
	b.camera(OutcomeNoDefect, 9000, 9600)
	s, outs := fold(withPassport(testEnv(), 4), b.out)
	ap := reactionsOf(outs, catalog.QualityInspectionAutoPassed)
	requireN(t, "автопропуск", ap, 1)
	if ap[0].AutomationMode != 2 || s.Axis != statuses.QualityNotInspected {
		t.Fatalf("автопропуск: режим %d, ось %s", ap[0].AutomationMode, s.Axis)
	}
	_, outs = fold(withPassport(testEnv(), 3), b.out)
	requireN(t, "автопропуск при уровне 3", reactionsOf(outs, catalog.QualityInspectionAutoPassed), 0)
}

// FR-36: значение вне допуска — признак дефекта, даже если источник сказал «признаков нет».
func TestMeasurementOutsideTolerance(t *testing.T) {
	var b builder
	b.add(catalog.InspectionResultRecorded, map[string]any{"method": "cmm", "phase": "after_operation", "outcome": "no_defect_indicated",
		"processing_state": "completed", "step_key": "machining.kt2_cmm", "zone_ids": []string{"F-FACE"},
		"measurements": []any{map[string]any{"characteristic": "Ø120", "verdict": "within",
			"value":     map[string]any{"value": 12035, "scale": 2, "unit": "mm"},
			"tolerance": map[string]any{"lower": map[string]any{"value": 1199, "scale": 1, "unit": "mm"}, "upper": map[string]any{"value": 1202, "scale": 1, "unit": "mm"}}}}})
	s, _ := fold(testEnv(), b.out)
	o := s.Observations[0]
	if o.Outcome != OutcomeDefect || o.Reinterpreted != "measurement_outside" || o.Measurements[0].Verdict != "outside" {
		t.Fatalf("измерение: %+v", o)
	}
	if s.Defects[0].TypeCode != "M-DIM" || s.Signals[0].RequirementRef == "" {
		t.Fatalf("дефект по измерению: %+v / %+v", s.Defects[0], s.Signals[0])
	}
}

// Решения контролёра: подтверждение — «несоответствие», отклонение — сигнал снят.
func TestReviewsDriveAxis(t *testing.T) {
	var b builder
	b.run("RUN-W1", "welding.weld")
	b.xray(OutcomeDefect, defect("W-PORE-S", "W-1", "У1", "major", 0))
	s, _ := fold(testEnv(), b.out)
	id := s.Signals[0].SignalID
	b2 := b
	b2.out = slices.Clone(b.out)
	b2.add(catalog.DecisionSignalRejected, map[string]any{"signal_ids": []string{id}, "reason": map[string]any{"code": "false_alarm", "text": "блик"}})
	s2, _ := fold(testEnv(), b2.out)
	if s2.Axis != statuses.QualityNotInspected || s2.Signals[0].State != SignalRejected {
		t.Fatalf("после отклонения: %s %+v", s2.Axis, s2.Signals[0])
	}
	b.add(catalog.DecisionNonconformityConfirmed, map[string]any{"nc_id": "NC-1", "signal_ids": []string{id}, "severity": "major", "reason": map[string]any{"code": "confirmed", "text": "поры"}})
	s, _ = fold(testEnv(), b.out)
	if s.Axis != statuses.QualityNonconforming || s.Signals[0].State != SignalConfirmed {
		t.Fatalf("после подтверждения: %s", s.Axis)
	}
	// Намерение SetQuality позднего модуля меняет ось (AD-30).
	s = Apply(s, SetQuality("nonconformity", statuses.QualityAcceptedWithConcession))
	if s.Axis != statuses.QualityAcceptedWithConcession {
		t.Fatalf("SetQuality: %s", s.Axis)
	}
}

// FR-100: дефект найден позже, чем мог, — пропуск брака с ранними наблюдениями.
func TestEscapeRecorded(t *testing.T) {
	var b builder
	b.run("RUN-W1", "welding.weld")
	missed := b.camera(OutcomeNoDefect, 9000, 9000)
	b.xray(OutcomeDefect, defect("W-PORE-S", "W-1.U3", "", "major", 0))
	s, outs := fold(withPassport(testEnv(), 3), b.out)
	requireN(t, "пропуски", s.Escapes, 1)
	if e := s.Escapes[0]; !e.MethodCovers || !slices.Equal(e.Missed, []string{missed.EventID}) || e.AnalyzerVersion != "vqc-weld 2.3.1" {
		t.Fatalf("пропуск брака: %+v", e)
	}
	requireN(t, "реакции пропуска", reactionsOf(outs, catalog.QualityEscapeRecorded), 1)
}

// FR-125: неизвестный вид принимается с флагом; без карты реакций — всё к
// человеку правилом по умолчанию; без нормативного слоя — «не проверено» и
// реакций нет (сверять не с чем), но факты и выводы в состоянии.
func TestEmptyNormIsSafe(t *testing.T) {
	var b builder
	b.xray(OutcomeDefect, defect("ZZZ", "W-1", "", "major", 0))
	b.xray(OutcomeUnable)
	env := testEnv()
	env.ReactionMap = normative.ReactionMap{}
	s, outs := fold(env, b.out)
	for _, sg := range s.Signals {
		if !sg.Raised || sg.Assessment.Outcome != ReactManual || sg.Assessment.RuleID != "default" {
			t.Fatalf("без карты реакций — к человеку: %+v", sg)
		}
	}
	if s.Signals[0].TypeKnown {
		t.Fatalf("вид вне классификатора известен")
	}
	requireN(t, "сигналы", reactionsOf(outs, catalog.QualitySignalRaised), 2)

	s, outs = fold(Env{}, b.out)
	requireN(t, "реакции без нормативного слоя", outs[len(outs)-1].Reactions, 0)
	requireN(t, "дефекты без нормативного слоя", s.Defects, 1)
	if s.Axis != statuses.QualityNotInspected {
		t.Fatalf("без нормы ось: %s", s.Axis)
	}
}

// AD-3: реакции строятся только своими типами и детерминированы.
func TestReactionsDeterministic(t *testing.T) {
	var b builder
	b.run("RUN-W1", "welding.weld")
	b.camera(OutcomeDefect, 9000, 7000, defect("W-UNDERCUT", "W-1", "У1", "major", 7000))
	b.xray(OutcomeDefect, defect("W-LOF", "W-1", "У6", "critical", 0))
	_, o1 := fold(withPassport(testEnv(), 3), b.out)
	_, o2 := fold(withPassport(testEnv(), 3), b.out)
	r1, r2 := o1[len(o1)-1].Reactions, o2[len(o2)-1].Reactions
	requireN(t, "реакции", r2, len(r1))
	for i := range r1 {
		if r1[i].ID(1) != r2[i].ID(1) || r1[i].Module != Module {
			t.Fatalf("реакция %d различается: %+v / %+v", i, r1[i], r2[i])
		}
	}
}

func point(s State, step string) PointStatus {
	for _, p := range s.Points {
		if p.StepKey == step {
			return p
		}
	}
	return PointStatus{}
}

func hasRequest(s State, kind, task string) bool {
	for _, r := range s.Requests {
		if r.Kind == kind && (task == "" || r.TaskKind == task) {
			return true
		}
	}
	return false
}

func hasContain(s State, c statuses.Containment) bool {
	for _, r := range s.Requests {
		if r.Kind == RequestContain && r.Containment == c {
			return true
		}
	}
	return false
}
