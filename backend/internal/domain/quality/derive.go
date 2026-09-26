package quality

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"ant/internal/contracts/constants"
	"ant/internal/contracts/statuses"
	"ant/internal/domain/kernel"
)

// Основания сигнала (quality.signal.raised basis_kind).
const (
	BasisInspection = "inspection_result"
	BasisSkipped    = "check_skipped"
	BasisOperator   = "operator_report"
)

// Состояния сигнала.
const (
	SignalOpen      = "open"
	SignalConfirmed = "confirmed"
	SignalRejected  = "rejected"
	// SignalResolved — «оценка невозможна» или пропуск проверки закрыты
	// новым результатом той же точки (повторный контроль).
	SignalResolved = "resolved"
)

// Defect — физический дефект (FR-37, «Соглашения/Дефект»): ключ — изделие,
// зона и место в пределах выполнения операции или до следующей операции, БЕЗ
// вида дефекта. Повторные наблюдения связываются с ним и не увеличивают
// показатели; уточнение вида в новом наблюдении обновляет вид.
type Defect struct {
	DefectID string `json:"defect_id"`
	Zone     string `json:"zone"`
	Location string `json:"location,omitempty"`
	Window   string `json:"window,omitempty"`
	// TypeCode — текущий вид по последнему наблюдению с видом; TypeSource — это наблюдение.
	TypeCode   string `json:"type_code,omitempty"`
	TypeKnown  bool   `json:"type_known"`
	TypeSource string `json:"type_source,omitempty"`
	// Severity — строже из всех наблюдений.
	Severity         string    `json:"severity"`
	FirstObservation string    `json:"first_observation"`
	Observations     []string  `json:"observations"`
	FirstPos         int       `json:"first_pos"`
	LastPos          int       `json:"last_pos"`
	IdentifiedAt     time.Time `json:"identified_at"`
	Methods          []string  `json:"methods"`
	// TypeChanges — наблюдения, уточнившие вид (event_id → новый вид).
	TypeChanges []TypeChange `json:"type_changes,omitempty"`
}

// TypeChange — уточнение вида дефекта наблюдением.
type TypeChange struct {
	ObservationID string `json:"observation_id"`
	TypeCode      string `json:"type_code"`
}

// Signal — сигнал о признаке дефекта или нарушении (кейс §2.3: первый из
// четырёх статусов, не брак): основание, вид, тяжесть, уверенность и качество
// наблюдения раздельно, реакция карты реакций с ограничениями.
type Signal struct {
	SignalID string `json:"signal_id"`
	Basis    string `json:"basis"`
	// Unable — сигнал «оценка невозможна» (повторный контроль), не признак дефекта.
	Unable       bool   `json:"unable,omitempty"`
	UnableReason string `json:"unable_reason,omitempty"`
	DefectID     string `json:"defect_id,omitempty"`
	Zone         string `json:"zone,omitempty"`
	TypeCode     string `json:"type_code,omitempty"`
	TypeKnown    bool   `json:"type_known"`
	Severity     string `json:"severity"`
	StepKey      string `json:"step_key,omitempty"`
	// ObservationID — последнее наблюдение сигнала; Observations — все.
	ObservationID  string   `json:"observation_id,omitempty"`
	Observations   []string `json:"observations,omitempty"`
	ConfidenceBP   *int     `json:"confidence_bp,omitempty"`
	QualityBP      *int     `json:"quality_bp,omitempty"`
	TrustLevel     *int     `json:"trust_level,omitempty"`
	RequirementRef string   `json:"requirement_ref,omitempty"`
	// Assessment — реакция карты реакций (FR-48) с ограничениями (FR-144).
	Assessment Assessment `json:"assessment"`
	// Raised — сигнал поднят реакцией quality.signal.raised (уровень доверия
	// паспорта 0–1 — только запись или рекомендация, AD-29).
	Raised     bool      `json:"raised"`
	State      string    `json:"state"`
	ReviewedBy string    `json:"reviewed_by,omitempty"`
	FirstPos   int       `json:"first_pos"`
	LastPos    int       `json:"last_pos"`
	RaisedAt   time.Time `json:"raised_at"`
	// Causes — записи-основания реакции (отсортированы).
	Causes []string `json:"causes"`
}

// PointStatus — полнота контроля по точке плана (FR-35): результат получен,
// ждём, нет данных (с причиной) или «оценка невозможна» — нужен повтор.
type PointStatus struct {
	StepKey         string `json:"step_key"`
	InspectionPoint string `json:"inspection_point,omitempty"`
	ClosingPoint    string `json:"closing_point,omitempty"`
	Method          string `json:"method"`
	Stage           string `json:"stage"`
	Order           int    `json:"order"`
	Required        bool   `json:"required"`
	// Status — received | pending | missing | unable.
	Status        string `json:"status"`
	MissingReason string `json:"missing_reason,omitempty"`
	// EventID — результат или запись, по которой обнаружено отсутствие.
	EventID string `json:"event_id,omitempty"`
	// Window — выполнение операции участка, после которого ждём результат.
	Window      string   `json:"window,omitempty"`
	DetectedPos int      `json:"detected_pos,omitempty"`
	Causes      []string `json:"causes,omitempty"`
}

// TypeCoverage — проверен ли вид дефекта методом, способным его выявить (FR-14):
// «не обнаружено» методом, не покрывающим вид, — не годность.
type TypeCoverage struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Severity string `json:"severity"`
	// Status — defect | no_defect | unable | not_checked («не проверено
	// методом, способным выявить»).
	Status   string   `json:"status"`
	Methods  []string `json:"methods,omitempty"`
	EventIDs []string `json:"event_ids,omitempty"`
}

// Escape — пропуск брака (FR-100): дефект найден позже, чем мог, — ранние
// наблюдения «признаков нет» той же зоны.
type Escape struct {
	DefectID        string   `json:"defect_id"`
	Missed          []string `json:"missed"`
	MethodCovers    bool     `json:"method_covers"`
	AnalyzerVersion string   `json:"analyzer_version,omitempty"`
	Causes          []string `json:"causes"`
}

// AutoPass — уверенное «признаков нет» пропущено к следующему контролю по
// делегированному правилу режима 2 при уровне доверия 4 (FR-48, AD-27, AD-29).
type AutoPass struct {
	ObservationID string `json:"observation_id"`
	PassportID    string `json:"passport_id"`
	TrustLevel    int    `json:"trust_level"`
	Sampled       bool   `json:"sampled"`
	RuleRef       string `json:"rule_ref"`
}

// derive пересчитывает выводы из входов (чистая функция входов и Env).
func (s *State) derive(env Env) {
	active := s.activeObservations()
	s.Defects = defects(s.ItemID, active, env)
	s.Signals = s.signals(active, env)
	if env.IsZero() {
		// Нормативный слой не закреплён (нет версии процесса): сверять не с
		// чем — факты и выводы записываются, но сигналы не поднимаются и
		// изделие «не проверено». В работе Env собирает BundleSource всегда.
		for i := range s.Signals {
			s.Signals[i].Raised = false
			s.Signals[i].Assessment.Limits = append(s.Signals[i].Assessment.Limits, "no_normative_layer")
		}
	}
	s.Points = s.points(active, env)
	s.Types = typeCoverage(active, s.Defects, env)
	s.Escapes = escapes(active, s.Defects, env)
	s.AutoPasses = autoPasses(active, env)
	s.Requests = requests(*s)
	s.Axis, s.AxisBasis = axis(*s)
}

func (s *State) activeObservations() []Observation {
	out := []Observation{}
	for _, o := range s.Observations {
		if !o.Superseded {
			out = append(out, o)
		}
	}
	return out
}

// defectKey — место физического дефекта без вида (FR-37).
func defectKey(itemID, window, zone, loc string) string {
	return kernel.UUIDv5(constants.NsAnt, "quality.defect\x1f"+itemID+"\x1f"+window+"\x1f"+zone+"\x1f"+loc)
}

// sameSpot — то же место: та же зона и то же место в зоне; место, не
// указанное одной из сторон, не различает дефекты.
func sameSpot(d Defect, window, zone, loc string) bool {
	return d.Window == window && d.Zone == zone && (d.Location == loc || d.Location == "" || loc == "")
}

func defects(itemID string, obs []Observation, env Env) []Defect {
	out := []Defect{}
	for _, o := range obs {
		if o.Outcome != OutcomeDefect {
			continue
		}
		for _, sg := range o.Defects {
			i := slices.IndexFunc(out, func(d Defect) bool { return sameSpot(d, o.Window, sg.Zone, sg.Location) })
			if i < 0 {
				out = append(out, Defect{
					DefectID: defectKey(itemID, o.Window, sg.Zone, sg.Location), Zone: sg.Zone, Location: sg.Location, Window: o.Window,
					TypeCode: sg.TypeCode, TypeKnown: sg.Known, Severity: stricterSeverity("unknown", sg.Severity),
					FirstObservation: o.EventID, Observations: []string{o.EventID}, FirstPos: o.Pos, LastPos: o.Pos,
					IdentifiedAt: o.OccurredAt, Methods: []string{o.Method},
				})
				if sg.TypeCode != "" {
					out[len(out)-1].TypeSource = o.EventID
				}
				continue
			}
			d := &out[i]
			if d.Location == "" && sg.Location != "" {
				d.Location = sg.Location
			}
			if !slices.Contains(d.Observations, o.EventID) {
				d.Observations = append(d.Observations, o.EventID)
			}
			d.LastPos = o.Pos
			if !slices.Contains(d.Methods, o.Method) {
				d.Methods = append(d.Methods, o.Method)
			}
			d.Severity = stricterSeverity(d.Severity, sg.Severity)
			// Уточнение вида обновляет вид, а не создаёт новый дефект.
			if sg.TypeCode != "" && sg.TypeCode != d.TypeCode {
				d.TypeCode, d.TypeKnown, d.TypeSource = sg.TypeCode, sg.Known, o.EventID
				d.TypeChanges = append(d.TypeChanges, TypeChange{ObservationID: o.EventID, TypeCode: sg.TypeCode})
			}
		}
	}
	_ = env
	return out
}

// facts — вход карты реакций по признаку дефекта наблюдения.
func facts(o Observation, sg *DefectSign, env Env) Facts {
	f := Facts{Basis: BasisInspection, Outcome: o.Outcome, Analyzer: o.Analyzer, Trust: o.TrustLevel, StepKey: o.StepKey,
		Closing: o.Closing, QualityBelow: o.QualityMinBP > 0 && o.QualityBP != nil && *o.QualityBP < o.QualityMinBP,
		ConfidenceBP: o.ConfidenceBP, Severity: "unknown", HasRequirement: true}
	if o.Reinterpreted == "poor_observation" {
		// Карта описывает этот случай на исходе источника: «признаков нет» при
		// качестве наблюдения ниже порога карты контроля (R-02).
		f.Outcome = o.Reported
	}
	if sg != nil {
		f.DefectType, f.TypeKnown, f.Severity = sg.TypeCode, sg.Known, sg.Severity
		if sg.ConfidenceBP != nil {
			f.ConfidenceBP = sg.ConfidenceBP
		}
		f.HasRequirement, _ = requirement(o, *sg, env)
	}
	if !o.Analyzer && f.ConfidenceBP == nil {
		// Заключение человека или прибора — утверждение, а не вероятность.
		full := 10000
		f.ConfidenceBP = &full
	}
	f.CoverageOK = coverageOK(o, env)
	return f
}

// requirement — есть ли требование КД для признака (FR-48: «сигнал без
// требования КД — вопрос технологу, а не брак»). Требование: допуск в самом
// признаке; ant:requirement шага наблюдения для видов, которые покрывает его
// контроль; вид классификатора (норма, утверждённая кворумом), описанный для
// вида зоны. Неизвестный вид или вид в зоне, для которой он не описан, —
// требования нет. Пустой классификатор — судить не по чему: требование
// считается заданным (к человеку ведёт правило по умолчанию).
func requirement(o Observation, sg DefectSign, env Env) (bool, string) {
	if sg.Measurable {
		return true, "допуск в результате контроля"
	}
	if len(env.Classifier.DefectTypes) == 0 {
		return true, ""
	}
	t, ok := env.defectType(sg.TypeCode)
	if !ok {
		// Неизвестный вид не описан ни классификатором, ни требованием шага.
		return false, ""
	}
	if st, ok := env.step(o.StepKey); ok && len(st.Requirements) > 0 {
		for _, in := range st.Inspections {
			if slices.Contains(in.Coverage, t.Code) {
				return true, st.Requirements[0].Ref()
			}
		}
	}
	if kind := env.zoneKind(sg.Zone); kind != "" && len(t.ZoneKinds) > 0 && !has(t.ZoneKinds, kind) {
		return false, ""
	}
	return true, "классификатор дефектов v" + strconv.Itoa(env.Classifier.Version) + ": " + t.Code + " «" + t.Name + "»"
}

// coverageOK — метод наблюдения способен выявить все виды, которые точка
// обязана покрыть (FR-14): только тогда «признаков нет» можно пропустить.
func coverageOK(o Observation, env Env) bool {
	st, ok := env.step(o.StepKey)
	if !ok {
		return false
	}
	for _, in := range st.Inspections {
		if in.Method != o.Method {
			continue
		}
		if len(in.Coverage) == 0 {
			return false
		}
		for _, c := range in.Coverage {
			if !env.methodDetects(o.Method, c) {
				return false
			}
		}
		return true
	}
	return false
}

// signalID — идентификатор сигнала: UUIDv5 от основания.
func signalID(parts ...string) string {
	return kernel.UUIDv5(constants.NsAnt, "quality.signal\x1f"+strings.Join(parts, "\x1f"))
}

// signals — сигналы изделия: по физическому дефекту (одна «эпоха» до решения
// контролёра; новое наблюдение после решения — новый сигнал), по «оценка
// невозможна», по пропуску проверки и по сообщению исполнителя.
func (s *State) signals(obs []Observation, env Env) []Signal {
	out := []Signal{}
	byID := func(id string) *Observation {
		for i := range obs {
			if obs[i].EventID == id {
				return &obs[i]
			}
		}
		return nil
	}
	for _, d := range s.Defects {
		var cur *Signal
		for _, oid := range d.Observations {
			o := byID(oid)
			if o == nil {
				continue
			}
			if cur != nil && s.reviewedBefore(cur.SignalID, o.Pos) {
				out = append(out, *cur)
				cur = nil
			}
			sg := signOf(*o, d)
			a := Assess(env, facts(*o, &sg, env))
			if cur == nil {
				cur = &Signal{SignalID: signalID(d.DefectID, o.EventID), Basis: BasisInspection, DefectID: d.DefectID, Zone: d.Zone,
					StepKey: o.StepKey, FirstPos: o.Pos, RaisedAt: o.OccurredAt, Assessment: a}
			} else {
				cur.Assessment = mergeAssessment(cur.Assessment, a)
			}
			cur.ObservationID, cur.LastPos = o.EventID, o.Pos
			cur.Observations = append(cur.Observations, o.EventID)
			cur.TypeCode, cur.TypeKnown = d.TypeCode, d.TypeKnown
			cur.Severity = stricterSeverity(cur.Severity, sg.Severity)
			cur.ConfidenceBP, cur.QualityBP = sg.ConfidenceBP, o.QualityBP
			if o.Analyzer {
				lvl := o.TrustLevel
				cur.TrustLevel = &lvl
			} else {
				cur.TrustLevel = nil
			}
			if ok, ref := requirement(*o, sg, env); ok {
				cur.RequirementRef = ref
			}
		}
		if cur != nil {
			out = append(out, *cur)
		}
	}
	for _, o := range obs {
		if o.Outcome != OutcomeUnable {
			continue
		}
		a := Assess(env, facts(o, nil, env))
		sg := Signal{SignalID: signalID("unable", o.EventID), Basis: BasisInspection, Unable: true, UnableReason: o.UnableReason,
			Severity: "unknown", StepKey: o.StepKey, ObservationID: o.EventID, Observations: []string{o.EventID},
			ConfidenceBP: o.ConfidenceBP, QualityBP: o.QualityBP, Assessment: a, FirstPos: o.Pos, LastPos: o.Pos, RaisedAt: o.OccurredAt}
		if len(o.Zones) > 0 {
			sg.Zone = o.Zones[0]
		}
		if o.Analyzer {
			lvl := o.TrustLevel
			sg.TrustLevel = &lvl
		}
		// Повторный контроль той же точки закрывает «оценка невозможна».
		if next := laterResult(obs, o); next != "" {
			sg.State, sg.ReviewedBy = SignalResolved, next
		}
		out = append(out, sg)
	}
	for _, sk := range s.Skips {
		a := Assess(env, Facts{Basis: BasisSkipped, StepKey: sk.StepKey, Severity: "unknown", HasRequirement: true})
		sg := Signal{SignalID: signalID("skip", sk.EventID), Basis: BasisSkipped, Severity: "unknown", StepKey: sk.StepKey,
			Assessment: a, FirstPos: sk.Pos, LastPos: sk.Pos, RaisedAt: sk.OccurredAt, Causes: []string{sk.EventID}}
		for _, o := range obs {
			if o.StepKey == sk.StepKey && o.Pos > sk.Pos && o.Outcome != OutcomeUnable {
				sg.State, sg.ReviewedBy = SignalResolved, o.EventID
				break
			}
		}
		out = append(out, sg)
	}
	for _, rp := range s.Reports {
		a := Assess(env, Facts{Basis: BasisOperator, Severity: "unknown", HasRequirement: true})
		out = append(out, Signal{SignalID: signalID("report", rp.EventID), Basis: BasisOperator, Severity: "unknown", Zone: rp.Zone,
			Assessment: a, FirstPos: rp.Pos, LastPos: rp.Pos, RaisedAt: rp.OccurredAt, Causes: []string{rp.EventID}})
	}
	for i := range out {
		sg := &out[i]
		if sg.Causes == nil {
			sg.Causes = slices.Clone(sg.Observations)
		}
		slices.Sort(sg.Causes)
		sg.Causes = slices.Compact(sg.Causes)
		sg.Raised = sg.Assessment.Outcome != "" && !sg.Assessment.Suppressed
		if sg.State == "" {
			sg.State = SignalOpen
		}
		for _, rv := range s.Reviews {
			if slices.Contains(rv.SignalIDs, sg.SignalID) {
				sg.State, sg.ReviewedBy = rv.Verdict, rv.EventID
			}
		}
	}
	slices.SortStableFunc(out, func(a, b Signal) int {
		if a.FirstPos != b.FirstPos {
			return a.FirstPos - b.FirstPos
		}
		return strings.Compare(a.SignalID, b.SignalID)
	})
	return out
}

// mergeAssessment — реакция сигнала по нескольким наблюдениям одного дефекта:
// строже из поднятых; наблюдение, которое только записывается (уровень
// доверия 0–1), не ослабляет и не усиливает поднятую реакцию.
func mergeAssessment(a, b Assessment) Assessment {
	switch {
	case a.Suppressed && !b.Suppressed:
		return b
	case !a.Suppressed && b.Suppressed:
		return a
	case a.Suppressed && b.Suppressed:
		b.Recommend = a.Recommend || b.Recommend
		return b
	}
	return stricter(a, b)
}

// signOf — признак дефекта d в наблюдении o.
func signOf(o Observation, d Defect) DefectSign {
	for _, sg := range o.Defects {
		if sameSpot(d, o.Window, sg.Zone, sg.Location) {
			return sg
		}
	}
	return DefectSign{Zone: d.Zone, Severity: "unknown"}
}

// reviewedBefore — контролёр принял решение по сигналу до позиции pos.
func (s *State) reviewedBefore(signal string, pos int) bool {
	for _, rv := range s.Reviews {
		if rv.Pos < pos && slices.Contains(rv.SignalIDs, signal) {
			return true
		}
	}
	return false
}

// laterResult — более поздний результат той же точки, который можно оценить.
func laterResult(obs []Observation, o Observation) string {
	for _, n := range obs {
		if n.Pos <= o.Pos || n.Outcome == OutcomeUnable || n.Method != o.Method {
			continue
		}
		if (o.StepKey != "" && n.StepKey == o.StepKey) || (o.StepKey == "" && n.InspectionPoint == o.InspectionPoint) {
			if !n.Analyzer || n.TrustLevel >= 1 {
				return n.EventID
			}
		}
	}
	return ""
}

// counts — результат засчитывается в полноту: человек, прибор или анализатор
// с допущенным паспортом (уровень 0 — только запись, контроль ручной, AD-29).
func counts(o Observation) bool { return !o.Analyzer || o.TrustLevel >= 1 }

// points — полнота контроля по плану (FR-35): ожидаемый, но не пришедший
// результат — «нет данных», когда изделие ушло дальше (предъявлено на
// закрывающей точке своего участка) или проверку пропустили. Повтор операции
// участка до точки (FR-47) и запрос повторного контроля снова ждут результата;
// операции шагов после точки её результат не старят.
func (s *State) points(obs []Observation, env Env) []PointStatus {
	out := []PointStatus{}
	for _, st := range env.Steps {
		for _, in := range st.Inspections {
			p := PointStatus{StepKey: st.StepKey, InspectionPoint: st.InspectionPoint, ClosingPoint: st.ClosingPoint, Method: in.Method,
				Stage: st.Stage, Order: st.Order, Required: st.IsInspection(), Status: "pending"}
			from := 0
			for _, r := range s.Runs {
				// Операция шага, стоящего в процессе позже точки, её результат
				// не старит: КТ-4d (до крышки) не «устаревает» от установки
				// крышки, крепежа и затяжки (FR-35, FR-47 — повтор относится к
				// операциям до точки).
				if rs, ok := env.step(r.StepKey); ok && rs.Order > st.Order {
					continue
				}
				if r.Stage == st.Stage && !r.Inspection && r.Pos > from {
					from, p.Window = r.Pos, r.RunID
				}
			}
			for _, rc := range s.Rechecks {
				if rc.Method == in.Method && rc.Pos > from && zonesMeet(rc.Zones, st.Zones) {
					from = rc.Pos
				}
			}
			var last *Observation
			shadowOnly := false
			for i := range obs {
				o := &obs[i]
				if o.StepKey != st.StepKey || o.Method != in.Method || o.Pos <= from {
					continue
				}
				if !counts(*o) {
					shadowOnly = true
					continue
				}
				last = o
			}
			switch {
			case last != nil && last.Outcome == OutcomeUnable:
				p.Status, p.EventID = "unable", last.EventID
			case last != nil:
				p.Status, p.EventID = "received", last.EventID
			default:
				s.missing(&p, st, from, shadowOnly, env)
			}
			out = append(out, p)
		}
	}
	return out
}

// missing — отметить точку «нет данных», если изделие ушло дальше.
func (s *State) missing(p *PointStatus, st StepSpec, from int, shadowOnly bool, env Env) {
	for _, sk := range s.Skips {
		if sk.Pos > from && (sk.StepKey == st.StepKey || (sk.InspectionPoint != "" && sk.InspectionPoint == st.InspectionPoint)) {
			p.Status, p.MissingReason, p.EventID, p.DetectedPos, p.Causes = "missing", "check_skipped", sk.EventID, sk.Pos, []string{sk.EventID}
			return
		}
	}
	for _, pr := range s.Presentations {
		if pr.Pos <= from {
			continue
		}
		cs, ok := env.step(pr.StepKey)
		if !ok || cs.Stage != st.Stage || cs.Order < st.Order || cs.ClosingPoint == "" {
			continue
		}
		if cs.StepKey == st.StepKey {
			// Своя закрывающая точка: предъявление — ещё не результат; решение
			// контролёра по ней и есть контроль человеком этой точки.
			switch pr.Resolution {
			case "":
				continue
			case "insufficient_data":
				p.Status, p.EventID = "unable", pr.EventID
			default:
				p.Status, p.EventID = "received", pr.EventID
			}
			return
		}
		reason := "result_not_received"
		if shadowOnly {
			reason = "point_manual_mode"
		}
		p.Status, p.MissingReason, p.EventID, p.DetectedPos, p.Causes = "missing", reason, pr.EventID, pr.Pos, []string{pr.EventID}
		return
	}
}

func zonesMeet(a, b []string) bool {
	if len(a) == 0 || len(b) == 0 {
		return true
	}
	for _, z := range a {
		for _, y := range b {
			if zoneWithin(z, y) || zoneWithin(y, z) {
				return true
			}
		}
	}
	return false
}

// zoneWithin — зона a — это b или её участок (W-1.U2 ⊂ W-1).
func zoneWithin(a, b string) bool { return a == b || strings.HasPrefix(a, b+".") }

// typeCoverage — виды дефектов плана: проверен ли каждый методом, способным
// его выявить (FR-14).
func typeCoverage(obs []Observation, ds []Defect, env Env) []TypeCoverage {
	planned := []string{}
	for _, st := range env.Steps {
		for _, in := range st.Inspections {
			for _, c := range in.Coverage {
				if !slices.Contains(planned, c) {
					planned = append(planned, c)
				}
			}
		}
	}
	out := []TypeCoverage{}
	for _, t := range env.Classifier.DefectTypes {
		if !slices.Contains(planned, t.Code) {
			continue
		}
		tc := TypeCoverage{Code: t.Code, Name: t.Name, Severity: string(t.Severity), Status: "not_checked"}
		for _, d := range ds {
			if d.TypeCode == t.Code {
				tc.Status = "defect"
				tc.EventIDs = append(tc.EventIDs, d.Observations...)
			}
		}
		if tc.Status == "not_checked" {
			unable := false
			for _, o := range obs {
				if !counts(o) || !env.methodDetects(o.Method, t.Code) {
					continue
				}
				switch o.Outcome {
				case OutcomeNoDefect, OutcomeDefect:
					tc.Status = "no_defect"
					tc.EventIDs = append(tc.EventIDs, o.EventID)
					if !slices.Contains(tc.Methods, o.Method) {
						tc.Methods = append(tc.Methods, o.Method)
					}
				case OutcomeUnable:
					unable = true
				}
			}
			if tc.Status == "not_checked" && unable {
				tc.Status = "unable"
			}
		}
		out = append(out, tc)
	}
	return out
}

// escapes — пропуски брака (FR-100): для каждого дефекта — ранние наблюдения
// «признаков нет» другой точки, охватившие зону дефекта, в том же окне или
// методом, способным выявить вид.
func escapes(obs []Observation, ds []Defect, env Env) []Escape {
	out := []Escape{}
	for _, d := range ds {
		var first Observation
		for _, o := range obs {
			if o.EventID == d.FirstObservation {
				first = o
			}
		}
		e := Escape{DefectID: d.DefectID}
		for _, o := range obs {
			if o.Pos >= d.FirstPos || o.Outcome != OutcomeNoDefect || !counts(o) || o.StepKey == first.StepKey {
				continue
			}
			if !coversZone(o.Zones, d.Zone) {
				continue
			}
			capable := env.methodDetects(o.Method, d.TypeCode)
			if o.Window != d.Window && !capable {
				continue
			}
			e.Missed = append(e.Missed, o.EventID)
			e.MethodCovers = e.MethodCovers || capable
			if e.AnalyzerVersion == "" && o.Analyzer {
				e.AnalyzerVersion = o.AnalyzerVersion
			}
		}
		if len(e.Missed) == 0 {
			continue
		}
		e.Causes = append(slices.Clone(e.Missed), d.FirstObservation)
		slices.Sort(e.Causes)
		out = append(out, e)
	}
	return out
}

func coversZone(zones []string, zone string) bool {
	if zone == "" {
		return false
	}
	for _, z := range zones {
		if zoneWithin(zone, z) || zoneWithin(z, zone) {
			return true
		}
	}
	return false
}

// autoPasses — автоматические пропуски к следующему контролю (FR-48, AD-27,
// AD-29): «признаков нет» анализатора с уровнем доверия 4 по правилу режима 2.
// Выборочная перепроверка человеком — детерминированно по хешу наблюдения.
func autoPasses(obs []Observation, env Env) []AutoPass {
	out := []AutoPass{}
	for _, o := range obs {
		if o.Outcome != OutcomeNoDefect || !o.Analyzer {
			continue
		}
		a := Assess(env, facts(o, nil, env))
		if a.Outcome != ReactPassToNext {
			continue
		}
		out = append(out, AutoPass{ObservationID: o.EventID, PassportID: o.PassportID, TrustLevel: o.TrustLevel,
			Sampled: sampled(o.EventID, a.SampledRecheckBP), RuleRef: a.MapRef})
	}
	return out
}

// sampled — попадает ли наблюдение в выборочную перепроверку с долей bp:
// первые 4 шестнадцатеричных знака UUIDv5 наблюдения по модулю 10000.
func sampled(eventID string, bp int) bool {
	if bp <= 0 {
		return false
	}
	h := kernel.UUIDv5(constants.NsAnt, "quality.sample\x1f"+eventID)
	v, err := strconv.ParseUint(h[:4], 16, 32)
	if err != nil {
		return true
	}
	return int(v%10000) < bp
}

// axis — ось «состояние качества» (§3b, AD-30). База — последнее решение
// человека на точке предъявления или намерение SetQuality; после неё строже
// базы: подтверждённое несоответствие → открытый сигнал о признаке дефекта →
// «оценка невозможна» → «нет данных» (не проверено). «Признаков нет» годности
// не даёт никогда (FR-36): «годно» ставит только решение человека.
func axis(s State) (statuses.Quality, []string) {
	base, basePos, basis := statuses.QualityNotInspected, 0, []string{}
	type mark struct {
		pos    int
		value  statuses.Quality
		causes []string
	}
	marks := []mark{}
	for _, p := range s.Presentations {
		v, ok := map[string]statuses.Quality{
			"accept": statuses.QualityConforming, "accept_with_concession": statuses.QualityAcceptedWithConcession,
			"reject": statuses.QualityNonconforming, "insufficient_data": statuses.QualityUnableToAssess,
		}[p.Resolution]
		if ok {
			marks = append(marks, mark{p.Pos, v, []string{p.EventID}})
		}
	}
	for _, o := range s.Overrides {
		marks = append(marks, mark{o.Pos, o.Value, o.Causes})
	}
	slices.SortStableFunc(marks, func(a, b mark) int { return a.pos - b.pos })
	for _, m := range marks {
		base, basePos, basis = m.value, m.pos, m.causes
	}
	nc, sig, unable, missing := []string{}, []string{}, []string{}, []string{}
	for _, rv := range s.Reviews {
		if rv.Verdict == SignalConfirmed && rv.Pos > basePos {
			nc = append(nc, rv.EventID)
		}
	}
	for _, sg := range s.Signals {
		if !sg.Raised || sg.State != SignalOpen || sg.LastPos <= basePos {
			continue
		}
		switch {
		case sg.Unable:
			unable = append(unable, sg.Causes...)
		case sg.Basis != BasisSkipped:
			sig = append(sig, sg.Causes...)
		}
	}
	for _, p := range s.Points {
		// «Нет данных» не снимает и решение человека: пока результата нет,
		// изделие не «годно» (FR-35).
		if p.Status == "missing" && p.Required {
			missing = append(missing, p.Causes...)
		}
	}
	pick := func(v statuses.Quality, c []string) (statuses.Quality, []string) {
		slices.Sort(c)
		return v, slices.Compact(c)
	}
	switch {
	case len(nc) > 0:
		return pick(statuses.QualityNonconforming, nc)
	case len(sig) > 0 && base != statuses.QualityNonconforming:
		return pick(statuses.QualitySignal, sig)
	case len(unable) > 0 && base != statuses.QualityNonconforming:
		return pick(statuses.QualityUnableToAssess, unable)
	case len(missing) > 0 && base != statuses.QualityNonconforming && base != statuses.QualityUnableToAssess:
		return pick(statuses.QualityNotInspected, missing)
	}
	return pick(base, basis)
}

// PlanPoints — полнота контроля по плану для состояния s (FR-35): для
// изделия без данных контроля все точки плана — «ждём».
func PlanPoints(s State, env Env) []PointStatus {
	return s.points(s.activeObservations(), env)
}
