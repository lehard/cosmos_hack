package analysis

import (
	"slices"
	"strconv"
	"time"
)

// Разбор обстоятельств несоответствия (FR-58, FR-59, FR-153): окно возможного
// возникновения, события трёх дорожек на одной шкале, нехватка сведений и
// гипотезы — «возможные обстоятельства», не «причина». Чистая функция
// состояния изделия и (если подключён порт) временной линии оборудования.

// Категории гипотез причины (FR-59).
const (
	CatIncoming       = "incoming"
	CatEquipment      = "equipment"
	CatPerformer      = "performer"
	CatHandling       = "handling"
	CatAssembly       = "assembly"
	CatDocumentation  = "documentation"
	CatNotEstablished = "not_established"
)

// Нехватка сведений (FR-58, FR-143) — значения перечисления контракта.
const (
	MissingTool           = "tool_unknown"
	MissingCycleEnd       = "cycle_end_time_unknown"
	MissingAfter          = "no_observation_after_operation"
	MissingBefore         = "no_observation_before_operation"
	MissingOperator       = "operator_unknown"
	MissingEquipmentLog   = "equipment_log_missing"
	MissingOther          = "other"
	confidenceStrong      = 8500
	confidencePossible    = 5000
	confidenceWeak        = 2000
	categoricalConfidence = confidenceStrong
)

// Window — окно возможного возникновения (FR-58): от последнего подтверждённо
// нормального состояния зоны до первой находки.
type Window struct {
	Start      time.Time `json:"start"`
	End        time.Time `json:"end"`
	LowerBound string    `json:"lower_bound,omitempty"`
	UpperBound string    `json:"upper_bound,omitempty"`
}

// Hypothesis — гипотеза причины, вычисленная системой (FR-59): категория,
// уверенность вывода (не вероятность вины; nil — оценить нельзя), доводы
// «за» и «против». Статус «подтверждённая» система не ставит никогда —
// только решение человека incident.cause.concluded.
type Hypothesis struct {
	ID              string   `json:"hypothesis_id"`
	Category        string   `json:"category"`
	ConfidenceBP    *int     `json:"confidence_bp,omitempty"`
	Supporting      []string `json:"supporting_event_ids"`
	Contradicting   []string `json:"contradicting_event_ids"`
	Statement       string   `json:"statement"`
	MeasurementHint string   `json:"measurement_hint,omitempty"`
}

// Profile — профиль несоответствия для общих факторов и похожих случаев
// (FR-135, FR-60, FR-148): вид дефекта, операция и её факторы. У входного
// дефекта факторов операции нет — он не связывается с исполнителем и
// оборудованием операции (FR-58).
type Profile struct {
	NCID          string    `json:"nc_id"`
	ItemID        string    `json:"item_id,omitempty"`
	DefectType    string    `json:"defect_type,omitempty"`
	StepKey       string    `json:"step_key,omitempty"`
	Operation     string    `json:"operation,omitempty"`
	Equipment     string    `json:"equipment,omitempty"`
	Program       string    `json:"program,omitempty"`
	Performer     string    `json:"performer,omitempty"`
	Tool          string    `json:"tool,omitempty"`
	Fixture       string    `json:"fixture,omitempty"`
	MaterialBatch string    `json:"material_batch,omitempty"`
	At            time.Time `json:"at"`
	Incoming      bool      `json:"incoming,omitempty"`
}

// Analysis — итог разбора обстоятельств несоответствия.
type Analysis struct {
	NCID string `json:"nc_id"`
	// Operation — выполнение операции, вокруг которого разбор; nil — не установлено
	// или дефект входной (не связывается с операцией).
	Operation *Run    `json:"operation,omitempty"`
	Window    *Window `json:"window,omitempty"`
	// Records — события дорожек в порядке времени.
	Records    []Mark       `json:"records"`
	Missing    []string     `json:"missing_information"`
	Hypotheses []Hypothesis `json:"hypotheses"`
	// Categorical — категоричный вывод; при нехватке сведений — false (кейс §5.1).
	Categorical bool `json:"conclusion_is_categorical"`
	// Incoming — дефект входной: обнаружен до операции или в теле компонента.
	Incoming bool    `json:"incoming,omitempty"`
	Profile  Profile `json:"profile"`
	// Causes — записи, на которых построен вывод (для причин реакции, AD-3).
	Causes []string `json:"causes,omitempty"`
}

// HypothesisID — идентификатор гипотезы системы: `HYP-‹nc›-‹категория›`
// (стабилен между версиями вывода — на него ссылаются отклонение и измерение).
func HypothesisID(ncID, category string) string { return "HYP-" + ncID + "-" + category }

// Case возвращает несоответствие изделия по id.
func (s State) Case(ncID string) (Case, bool) {
	for _, c := range s.Cases {
		if c.NCID == ncID {
			return c, true
		}
	}
	return Case{}, false
}

// Analyze — разбор обстоятельств несоответствия ncID изделия (FR-58, FR-59).
// eq — события оборудования из порта временной линии (AD-42, эпик 23); nil —
// порт не подключён: выводов об оборудовании нет (ни гипотезы, ни «журнала
// нет»), вывод не категоричен. ok = false — несоответствия у изделия нет.
func Analyze(s State, itemID, ncID string, eq []EquipmentEvent) (Analysis, bool) {
	c, ok := s.Case(ncID)
	if !ok {
		return Analysis{}, false
	}
	a := Analysis{NCID: ncID, Records: []Mark{}, Missing: []string{}, Hypotheses: []Hypothesis{}}
	a.Profile = Profile{NCID: ncID, ItemID: itemID, DefectType: c.DefectType, At: c.At}
	causes := []string{c.EventID}

	first, lastClean := findingsAround(s, c)
	if first != nil {
		causes = append(causes, first.EventID)
		if a.Profile.DefectType == "" && len(first.DefectTypes) > 0 {
			a.Profile.DefectType = first.DefectTypes[0]
		}
	}
	end := c.At
	if first != nil {
		end = first.At
	}
	if lastClean != nil && first != nil {
		a.Window = &Window{Start: lastClean.At, End: first.At, LowerBound: lastClean.EventID, UpperBound: first.EventID}
		causes = append(causes, lastClean.EventID)
	}
	run := operationRun(s, lastClean, end)
	a.Incoming = isIncoming(first, run)
	a.Profile.Incoming = a.Incoming

	// Дорожка изделия: результаты контроля и перемещения до находки включительно.
	for _, m := range s.Marks {
		if m.Lane == LaneItem && !m.OccurredAt.After(c.At) {
			a.Records = append(a.Records, m)
		}
	}

	if a.Incoming {
		// FR-58: входной дефект не связывается с исполнителем операции — ни
		// гипотезы «исполнитель», ни событий его дорожки, ни факторов операции.
		h := Hypothesis{ID: HypothesisID(ncID, CatIncoming), Category: CatIncoming, ConfidenceBP: bp(confidenceStrong),
			Supporting: ids(first), Contradicting: []string{},
			Statement:       "Входной дефект: обнаружен до операции или в теле компонента — операцию изделия он не затрагивает (возможное обстоятельство)",
			MeasurementHint: "Выборочный контроль компонентов той же партии"}
		a.Hypotheses = append(a.Hypotheses, h)
		if lastClean == nil && first != nil && first.Phase != "incoming" {
			a.Missing = append(a.Missing, MissingBefore)
		}
		a.Categorical = len(a.Missing) == 0
		sortRecords(a.Records)
		a.Causes = sortedUnique(causes)
		return a, true
	}

	if run != nil {
		r := *run
		a.Operation = &r
		a.Profile.StepKey, a.Profile.Operation, a.Profile.Equipment, a.Profile.Program = r.StepKey, r.Operation, r.Equipment, r.Program
		if r.OperatorKnown {
			a.Profile.Performer = r.Operator
		} else {
			a.Missing = append(a.Missing, MissingOperator)
		}
		if r.Finished == nil {
			a.Missing = append(a.Missing, MissingCycleEnd)
		}
		if first == nil || (r.Finished != nil && first.At.Before(*r.Finished)) {
			a.Missing = append(a.Missing, MissingAfter)
		}
		causes = append(causes, r.StartEventID)
	}
	if lastClean == nil {
		a.Missing = append(a.Missing, MissingBefore)
	}

	// Дорожка человека: действия исполнителя в окне (или в интервале операции).
	from, to := windowBounds(a.Window, run, end)
	var personIDs []string
	for _, m := range s.Marks {
		if m.Lane != LanePerson || m.OccurredAt.Before(from) || m.OccurredAt.After(to) {
			continue
		}
		a.Records = append(a.Records, m)
		if isOperatorAction(m) {
			personIDs = append(personIDs, m.EventID)
		}
	}
	var moveIDs []string
	if run != nil && run.Finished != nil {
		for _, m := range s.Marks {
			if m.Variant == "movement" && m.OccurredAt.After(*run.Finished) && !m.OccurredAt.After(end) {
				moveIDs = append(moveIDs, m.EventID)
			}
		}
	}

	// Дорожка оборудования и гипотеза «оборудование» — только если порт подключён.
	if eq != nil && run != nil && run.Equipment != "" {
		var devIDs []string
		found, regime := false, false
		runEnd := end
		if run.Finished != nil {
			runEnd = *run.Finished
		}
		for _, e := range eq {
			if e.EquipmentID != run.Equipment {
				continue
			}
			if e.EventType == "equipment.tool.changed" && !e.OccurredAt.After(run.Started) {
				a.Profile.Tool, a.Profile.Fixture = e.ToolID, e.FixtureID
			}
			if !overlaps(e, from, to) {
				continue
			}
			a.Records = append(a.Records, Mark{EventID: e.EventID, EventType: e.EventType, Variant: e.Variant, Lane: LaneEquipment,
				OccurredAt: e.OccurredAt, EndedAt: e.EndedAt, Seq: e.Seq, SourceKind: e.SourceKind, Params: e.Params})
			if overlaps(e, run.Started, runEnd) {
				found = true
				if e.Deviation {
					devIDs = append(devIDs, e.EventID)
					// Выход параметра режима за уставку (отклонение или цикл) — сильный довод.
					regime = regime || e.Variant == "out_of_setpoint" || e.Variant == "overload" || e.Variant == "cycle"
				}
			}
		}
		if a.Profile.Tool == "" {
			a.Missing = append(a.Missing, MissingTool)
		}
		switch {
		case !found:
			a.Missing = append(a.Missing, MissingEquipmentLog)
			a.Hypotheses = append(a.Hypotheses, Hypothesis{ID: HypothesisID(ncID, CatEquipment), Category: CatEquipment,
				Supporting: []string{}, Contradicting: []string{},
				Statement:       "Оборудование " + run.Equipment + ": журнала за интервал операции нет — оценить нельзя",
				MeasurementHint: "Досылка журнала источника или контрольный образец на " + run.Equipment})
		case len(devIDs) > 0:
			conf := confidencePossible
			if regime {
				conf = confidenceStrong
			}
			a.Hypotheses = append(a.Hypotheses, Hypothesis{ID: HypothesisID(ncID, CatEquipment), Category: CatEquipment,
				ConfidenceBP: bp(conf), Supporting: devIDs, Contradicting: []string{},
				Statement:       "Оборудование " + run.Equipment + ": отклонений режима в интервале операции — " + strconv.Itoa(len(devIDs)) + " (возможное обстоятельство)",
				MeasurementHint: "Контрольный образец на " + run.Equipment + " при уставке режима"})
			causes = append(causes, devIDs...)
		default:
			// Журнал есть и в уставке — довод «против» гипотезы оборудования.
			var clean []string
			for _, e := range eq {
				if e.EquipmentID == run.Equipment && overlaps(e, run.Started, runEnd) {
					clean = append(clean, e.EventID)
				}
			}
			a.Hypotheses = append(a.Hypotheses, Hypothesis{ID: HypothesisID(ncID, CatEquipment), Category: CatEquipment,
				ConfidenceBP: bp(confidenceWeak), Supporting: []string{}, Contradicting: clean,
				Statement: "Оборудование " + run.Equipment + ": журнал за интервал операции в уставке"})
		}
	}

	if len(personIDs) > 0 {
		a.Hypotheses = append(a.Hypotheses, Hypothesis{ID: HypothesisID(ncID, CatPerformer), Category: CatPerformer,
			ConfidenceBP: bp(confidenceWeak), Supporting: personIDs, Contradicting: []string{},
			Statement:       "Действия исполнителя в интервале операции: " + strconv.Itoa(len(personIDs)) + " — обстоятельство, не вина (ошибку подтверждает только человек после объяснения работника)",
			MeasurementHint: "Сверка с технологической инструкцией; письменное объяснение — только после подтверждения причины"})
	}
	if len(moveIDs) > 0 {
		a.Hypotheses = append(a.Hypotheses, Hypothesis{ID: HypothesisID(ncID, CatHandling), Category: CatHandling,
			ConfidenceBP: bp(confidenceWeak), Supporting: moveIDs, Contradicting: []string{},
			Statement: "Перемещение или хранение между операцией и находкой (возможное обстоятельство)"})
	}
	supported := false
	for _, h := range a.Hypotheses {
		supported = supported || len(h.Supporting) > 0
	}
	if !supported {
		a.Hypotheses = append(a.Hypotheses, Hypothesis{ID: HypothesisID(ncID, CatNotEstablished), Category: CatNotEstablished,
			Supporting: []string{}, Contradicting: []string{}, Statement: "Сведений недостаточно — причина не установлена"})
	}
	a.Categorical = categorical(a, eq != nil)
	sortRecords(a.Records)
	a.Causes = sortedUnique(causes)
	return a, true
}

// categorical — вывод категоричен, только если сведений хватает (нехватки нет,
// оборудование оценено), ровно одна гипотеза сильная и нет другой хотя бы
// возможной (кейс §5.1 «Неопределённость», FR-58).
func categorical(a Analysis, equipmentAssessed bool) bool {
	if len(a.Missing) > 0 || !equipmentAssessed {
		return false
	}
	strong, other := 0, 0
	for _, h := range a.Hypotheses {
		switch {
		case h.ConfidenceBP == nil:
			other++
		case *h.ConfidenceBP >= categoricalConfidence:
			strong++
		case *h.ConfidenceBP >= confidencePossible:
			other++
		}
	}
	return strong == 1 && other == 0
}

// findingsAround — первая находка дефекта (по зонам находки, приведшей к
// несоответствию) и последняя чистая проверка тех же зон до неё (FR-58).
func findingsAround(s State, c Case) (first, lastClean *Finding) {
	var trigger *Finding
	for i := range s.Findings {
		f := &s.Findings[i]
		if f.Outcome == "defect_indicated" && !f.At.After(c.At) && (trigger == nil || !f.At.Before(trigger.At)) {
			trigger = f
		}
	}
	if trigger == nil {
		return nil, nil
	}
	zones := trigger.DefectZones
	if len(zones) == 0 {
		zones = trigger.Zones
	}
	first = trigger
	for i := range s.Findings {
		f := &s.Findings[i]
		if f.Outcome == "defect_indicated" && f.At.Before(first.At) && covers(f, zones, true) {
			first = f
		}
	}
	for i := range s.Findings {
		f := &s.Findings[i]
		if f.Outcome == "no_defect_indicated" && f.At.Before(first.At) && covers(f, zones, false) &&
			(lastClean == nil || f.At.After(lastClean.At)) {
			lastClean = f
		}
	}
	return first, lastClean
}

// covers — проверка охватывала хотя бы одну из зон (нет зон у проверки или у
// находки — считается изделие целиком).
func covers(f *Finding, zones []string, defect bool) bool {
	fz := f.Zones
	if defect && len(f.DefectZones) > 0 {
		fz = f.DefectZones
	}
	if len(zones) == 0 || len(fz) == 0 {
		return true
	}
	for _, z := range zones {
		if slices.Contains(fz, z) {
			return true
		}
	}
	return false
}

// operationRun — выполнение операции в окне: последнее начатое после
// последней чистой проверки и не позже находки; предпочтительно с
// оборудованием.
func operationRun(s State, lastClean *Finding, end time.Time) *Run {
	var best *Run
	for i := range s.Runs {
		r := &s.Runs[i]
		if r.Started.After(end) || (lastClean != nil && r.Started.Before(lastClean.At)) {
			continue
		}
		switch {
		case best == nil:
			best = r
		case (r.Equipment != "") != (best.Equipment != ""):
			if r.Equipment != "" {
				best = r
			}
		case r.Started.After(best.Started):
			best = r
		}
	}
	return best
}

// isIncoming — дефект входной: находка на входном контроле или до операции,
// дефект в теле компонента, или находка раньше начала операции (FR-58, FR-59).
func isIncoming(first *Finding, run *Run) bool {
	if first == nil {
		return false
	}
	if first.Phase == "incoming" || first.Phase == "before_operation" || len(first.Components) > 0 {
		return true
	}
	return run != nil && first.At.Before(run.Started)
}

// isOperatorAction — событие дорожки человека, которое может быть
// обстоятельством (обход, отклонение, смена режима, наблюдение, пропуск), а не
// штатные начало и конец операции.
func isOperatorAction(m Mark) bool {
	switch m.EventType {
	case "operator.override.performed", "operator.deviation.reported", "operator.mode.changed",
		"operator.action.observed", "operator.check.skipped":
		return true
	}
	return false
}

// windowBounds — интервал дорожек человека и оборудования: окно возможного
// возникновения, иначе — интервал операции, иначе — только момент находки.
func windowBounds(w *Window, run *Run, end time.Time) (time.Time, time.Time) {
	switch {
	case w != nil:
		return w.Start, w.End
	case run != nil:
		return run.Started, end
	}
	return end, end
}

// overlaps — событие (момент или интервал) пересекает [from, to].
func overlaps(e EquipmentEvent, from, to time.Time) bool {
	endAt := e.OccurredAt
	if e.EndedAt != nil {
		endAt = *e.EndedAt
	}
	return !e.OccurredAt.After(to) && !endAt.Before(from)
}

func ids(f *Finding) []string {
	if f == nil {
		return []string{}
	}
	return []string{f.EventID}
}

func bp(v int) *int { return &v }

func sortRecords(ms []Mark) {
	slices.SortStableFunc(ms, func(a, b Mark) int {
		switch {
		case a.OccurredAt.Before(b.OccurredAt):
			return -1
		case b.OccurredAt.Before(a.OccurredAt):
			return 1
		case a.EventID < b.EventID:
			return -1
		case a.EventID > b.EventID:
			return 1
		}
		return 0
	})
}

func sortedUnique(xs []string) []string {
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		if x != "" {
			out = append(out, x)
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}
