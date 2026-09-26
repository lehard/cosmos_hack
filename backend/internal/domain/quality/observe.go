package quality

import (
	"strings"
	"time"

	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
)

// Исходы результата контроля (FR-36, кейс §4.4).
const (
	OutcomeDefect   = string(ev.InspectionOutcomeDefectIndicated)
	OutcomeNoDefect = string(ev.InspectionOutcomeNoDefectIndicated)
	OutcomeUnable   = string(ev.InspectionOutcomeUnableToAssess)
)

// Observation — результат контроля любого метода в едином виде (FR-36): исход
// источника и действующий исход по правилам интерпретации. «Признаков нет»
// при плохом наблюдении, прерванная или сбойная обработка, неизмеренная
// характеристика — это «оценка невозможна», а не «годно» (кейс §4.5).
// Уверенность анализатора и качество наблюдения хранятся раздельно (FR-38).
type Observation struct {
	EventID    string    `json:"event_id"`
	Seq        int64     `json:"seq,omitempty"`
	Pos        int       `json:"pos"`
	OccurredAt time.Time `json:"occurred_at"`
	// SourceKind — пометка источника (FR-140).
	SourceKind string `json:"source_kind,omitempty"`
	// Corrects — исправляемое наблюдение (FR-122); Superseded — наблюдение
	// исправлено более поздней записью и в выводах не участвует.
	Corrects   string `json:"corrects,omitempty"`
	Superseded bool   `json:"superseded,omitempty"`

	StepKey         string `json:"step_key,omitempty"`
	InspectionPoint string `json:"inspection_point,omitempty"`
	Method          string `json:"method"`
	Phase           string `json:"phase,omitempty"`
	OperationRunID  string `json:"operation_run_id,omitempty"`
	// Window — окно физического дефекта: выполнение операции, после которого
	// сделано наблюдение (FR-37: «в пределах выполнения операции или до
	// следующей операции»).
	Window string `json:"window,omitempty"`
	// Planned — наблюдение относится к точке контроля плана.
	Planned bool `json:"planned"`
	// Closing — наблюдение на закрывающей точке (автоматический пропуск запрещён).
	Closing bool `json:"closing,omitempty"`

	// Reported — исход источника; Outcome — действующий исход (FR-36).
	Reported string `json:"reported"`
	Outcome  string `json:"outcome"`
	// UnableReason — код причины «оценка невозможна».
	UnableReason string `json:"unable_reason,omitempty"`
	// Reinterpreted — почему система изменила исход источника:
	// processing_not_completed | poor_observation | measurement_outside |
	// not_measured | defect_without_signs.
	Reinterpreted   string `json:"reinterpreted,omitempty"`
	ProcessingState string `json:"processing_state"`

	// ConfidenceBP — уверенность анализатора (не вероятность брака, NFR-UI-4).
	ConfidenceBP *int `json:"confidence_bp,omitempty"`
	// QualityBP — качество наблюдения; QualityMinBP — порог карты контроля.
	QualityBP    *int `json:"quality_bp,omitempty"`
	QualityMinBP int  `json:"quality_min_bp,omitempty"`

	// Analyzer — наблюдение анализатора (VisionQC): к нему применяется уровень
	// доверия паспорта (AD-29); иначе — человек, прибор, лаборатория.
	Analyzer        bool   `json:"analyzer"`
	AnalyzerVersion string `json:"analyzer_version,omitempty"`
	RecipeRef       string `json:"recipe_ref,omitempty"`
	// TrustLevel — уровень доверия паспорта на момент наблюдения; TrustNote —
	// admitted | no_qualified_analyzer | suspended:… | shadow | analyzer_version_mismatch.
	TrustLevel int    `json:"trust_level"`
	TrustNote  string `json:"trust_note,omitempty"`
	PassportID string `json:"passport_id,omitempty"`

	// Stages — ступени анализатора (FR-38): локализация → классификация.
	Stages []Stage `json:"stages,omitempty"`
	// Versions — вектор версий наблюдения (AD-29).
	Versions     map[string]string `json:"versions,omitempty"`
	Defects      []DefectSign      `json:"defects,omitempty"`
	Measurements []Measurement     `json:"measurements,omitempty"`
	Zones        []string          `json:"zones,omitempty"`
	Limitations  []string          `json:"limitations,omitempty"`
	Evidence     []string          `json:"evidence,omitempty"`
	// Recommendation — совет анализатора; решает карта реакций (FR-38, FR-48).
	Recommendation string `json:"recommendation,omitempty"`
	Inspector      string `json:"inspector,omitempty"`
	ConclusionRef  string `json:"conclusion_ref,omitempty"`
}

// Stage — ступень анализатора с версией и уверенностью (FR-38).
type Stage struct {
	Stage        string `json:"stage"`
	Version      string `json:"version"`
	ConfidenceBP *int   `json:"confidence_bp,omitempty"`
	Note         string `json:"note,omitempty"`
}

// DefectSign — признак дефекта в наблюдении (FR-37).
type DefectSign struct {
	// TypeCode — код вида; Known — вид есть в классификаторе (FR-125).
	TypeCode string `json:"type_code,omitempty"`
	Known    bool   `json:"known"`
	// Severity — тяжесть: строже из указанной источником и классификатором.
	Severity    string `json:"severity"`
	Zone        string `json:"zone"`
	Location    string `json:"location,omitempty"`
	Description string `json:"description,omitempty"`
	// ConfidenceBP — уверенность ступени по этому признаку (иначе — наблюдения).
	ConfidenceBP *int `json:"confidence_bp,omitempty"`
	// Measurable — признак несёт измерение против допуска (требование задано).
	Measurable bool `json:"measurable,omitempty"`
}

// Measurement — измерение «значение против допуска» (FR-36): оценка
// источника и оценка системы по целым с масштабом (AD-4).
type Measurement struct {
	Characteristic string `json:"characteristic"`
	Value          string `json:"value,omitempty"`
	Tolerance      string `json:"tolerance,omitempty"`
	// SourceVerdict — within | outside | not_measured у источника;
	// Verdict — действующая: строже из источника и расчёта.
	SourceVerdict string `json:"source_verdict"`
	Verdict       string `json:"verdict"`
}

// severityRank — строгость тяжести (ГОСТ 15467).
func severityRank(s string) int {
	switch s {
	case "critical":
		return 3
	case "major":
		return 2
	case "minor":
		return 1
	}
	return 0
}

// stricterSeverity — строже из двух; неизвестная не ослабляет известную.
func stricterSeverity(a, b string) string {
	if severityRank(b) > severityRank(a) {
		return b
	}
	if a == "" {
		return "unknown"
	}
	return a
}

// observe переводит факт inspection.result.recorded в Observation (FR-36):
// действующий исход, тяжесть по классификатору, уровень доверия паспорта.
func observe(r kernel.Record, d ev.InspectionResultRecordedV1, pos int, env Env) Observation {
	o := Observation{
		EventID: r.EventID, Seq: r.Seq, Pos: pos, OccurredAt: r.OccurredAt, SourceKind: r.SourceKind, Corrects: r.Corrects,
		Method: string(d.Method), Phase: string(d.Phase), Reported: string(d.Outcome), ProcessingState: string(d.ProcessingState),
		ConfidenceBP: bpPtr(d.AnalyzerConfidenceBp), QualityBP: bpPtr(d.ObservationQualityBp), Limitations: d.Limitations,
	}
	if d.StepKey != nil {
		o.StepKey = string(*d.StepKey)
	}
	if d.InspectionPoint != nil {
		o.InspectionPoint = *d.InspectionPoint
	}
	if d.OperationRunID != nil {
		o.OperationRunID = string(*d.OperationRunID)
	}
	if d.InspectorID != nil {
		o.Inspector = *d.InspectorID
	}
	if d.ConclusionRef != nil {
		o.ConclusionRef = *d.ConclusionRef
	}
	if d.Recommendation != nil {
		o.Recommendation = string(*d.Recommendation)
	}
	for _, z := range d.ZoneIds {
		o.Zones = append(o.Zones, string(z))
	}
	for _, e := range d.EvidenceRefs {
		o.Evidence = append(o.Evidence, string(e.MaterialAddress))
	}
	for _, st := range d.Stages {
		s := Stage{Stage: string(st.Stage), Version: st.Version, ConfidenceBP: bpPtr(st.ConfidenceBp)}
		if st.OutputNote != nil {
			s.Note = *st.OutputNote
		}
		o.Stages = append(o.Stages, s)
	}
	if v := d.Versions; v != nil {
		o.AnalyzerVersion, o.RecipeRef = v.AnalyzerVersion, v.RecipeRef
		o.Versions = map[string]string{"recipe_ref": v.RecipeRef, "analyzer_version": v.AnalyzerVersion, "contract_version": v.ContractVersion}
		for _, kv := range []struct {
			k string
			p *string
		}{{"item_revision", v.ItemRevision}, {"camera_config", v.CameraConfig}, {"calibration", v.Calibration},
			{"threshold_profile", v.ThresholdProfile}, {"app_version", v.AppVersion}} {
			if kv.p != nil {
				o.Versions[kv.k] = *kv.p
			}
		}
	}

	// Точка плана: порог качества наблюдения карты контроля и признак закрывающей точки.
	if st, ok := env.pointFor(o.StepKey, o.InspectionPoint, o.Method); ok {
		o.Planned = true
		if o.StepKey == "" {
			o.StepKey = st.StepKey
		}
		if o.InspectionPoint == "" {
			o.InspectionPoint = st.InspectionPoint
		}
		o.Closing = st.ClosingPoint != ""
		for _, in := range st.Inspections {
			if in.Method == o.Method {
				o.QualityMinBP = in.ObservationQualityMinBP
				if o.RecipeRef == "" {
					o.RecipeRef = in.RecipeRef
				}
			}
		}
	}

	// Анализатор (VisionQC): ступени или камера без контролёра-человека (AD-29).
	o.Analyzer = len(d.Stages) > 0 || (d.Method == ev.InspectionMethodCamera && o.Inspector == "")
	if o.Analyzer {
		p, lvl, note := env.passportFor(o.RecipeRef, o.AnalyzerVersion, o.OccurredAt)
		o.PassportID, o.TrustLevel, o.TrustNote = p.PassportID, lvl, note
	}

	// Признаки дефектов: вид по классификатору, тяжесть — строже из двух.
	for _, df := range d.Defects {
		s := DefectSign{Zone: string(df.ZoneID), Severity: string(df.Severity), ConfidenceBP: bpPtr(df.StageConfidenceBp)}
		if df.DefectTypeCode != nil {
			s.TypeCode = *df.DefectTypeCode
		}
		if df.Location != nil {
			s.Location = strings.TrimSpace(*df.Location)
		}
		if df.Description != nil {
			s.Description = string(*df.Description)
		}
		if t, ok := env.defectType(s.TypeCode); ok {
			s.Known = true
			s.Severity = stricterSeverity(s.Severity, string(t.Severity))
		}
		s.Measurable = df.Tolerance != nil
		if s.ConfidenceBP == nil {
			s.ConfidenceBP = o.ConfidenceBP
		}
		o.Defects = append(o.Defects, s)
	}
	for _, m := range d.Measurements {
		o.Measurements = append(o.Measurements, measure(m))
	}
	interpret(&o, d, env)
	return o
}

// interpret — действующий исход наблюдения (FR-36, кейс §4.5).
func interpret(o *Observation, d ev.InspectionResultRecordedV1, env Env) {
	o.Outcome = o.Reported
	if d.UnableReason != nil {
		o.UnableReason = string(*d.UnableReason)
	}
	outside, notMeasured := []Measurement{}, false
	for _, m := range o.Measurements {
		switch m.Verdict {
		case "outside":
			outside = append(outside, m)
		case "not_measured":
			notMeasured = true
		}
	}
	switch {
	// «Прервано» и «сбой» у источника — «оценка невозможна» при любом исходе.
	case o.ProcessingState != string(ev.ProcessingStateCompleted):
		o.Outcome = OutcomeUnable
		if o.ProcessingState == string(ev.ProcessingStateFailed) {
			o.UnableReason = "analyzer_failure"
		} else {
			o.UnableReason = "processing_aborted"
		}
		if o.Reported != OutcomeUnable {
			o.Reinterpreted = "processing_not_completed"
		}
	case o.Reported == OutcomeUnable:
		if o.UnableReason == "" {
			o.UnableReason = "other"
		}
	// Значение вне допуска — признак дефекта, что бы ни сказал источник.
	case len(outside) > 0:
		o.Outcome = OutcomeDefect
		if o.Reported != OutcomeDefect {
			o.Reinterpreted = "measurement_outside"
		}
		if len(o.Defects) == 0 {
			for _, m := range outside {
				o.Defects = append(o.Defects, measuredSign(*o, m, env))
			}
		}
	case o.Reported == OutcomeDefect:
		if len(o.Defects) == 0 {
			// Признак без описания — один признак неизвестного вида на зону наблюдения.
			zone := ""
			if len(o.Zones) > 0 {
				zone = o.Zones[0]
			}
			o.Defects = append(o.Defects, DefectSign{Zone: zone, Severity: "unknown", ConfidenceBP: o.ConfidenceBP})
			o.Reinterpreted = "defect_without_signs"
		}
	// «Признаков нет» при плохом кадре — не годность, а «оценка невозможна».
	case o.QualityMinBP > 0 && o.QualityBP != nil && *o.QualityBP < o.QualityMinBP:
		o.Outcome = OutcomeUnable
		o.UnableReason = "poor_image"
		o.Reinterpreted = "poor_observation"
	case notMeasured:
		o.Outcome = OutcomeUnable
		o.UnableReason = "not_measured"
		o.Reinterpreted = "not_measured"
	}
}

// measuredSign — признак дефекта по измерению вне допуска: вид — измеряемый
// вид классификатора, который выявляет метод наблюдения на этой точке (если он
// один), иначе неизвестен.
func measuredSign(o Observation, m Measurement, env Env) DefectSign {
	s := DefectSign{Severity: "unknown", Location: m.Characteristic, Measurable: true, ConfidenceBP: o.ConfidenceBP}
	if len(o.Zones) > 0 {
		s.Zone = o.Zones[0]
	}
	codes := []string{}
	if st, ok := env.step(o.StepKey); ok {
		for _, in := range st.Inspections {
			if in.Method != o.Method {
				continue
			}
			for _, c := range in.Coverage {
				if t, ok := env.defectType(c); ok && t.Measurable {
					codes = append(codes, c)
				}
			}
		}
	}
	if len(codes) == 1 {
		t, _ := env.defectType(codes[0])
		s.TypeCode, s.Known, s.Severity = t.Code, true, string(t.Severity)
	}
	return s
}

// measure — измерение против допуска: оценка системы по целым с масштабом;
// действующая оценка — строже из источника и расчёта.
func measure(m ev.MeasurementResult) Measurement {
	out := Measurement{Characteristic: m.Characteristic, SourceVerdict: string(m.Verdict), Tolerance: tolText(m.Tolerance)}
	if m.Value != nil {
		out.Value = measText(*m.Value)
	}
	out.Verdict = out.SourceVerdict
	if m.Value == nil {
		out.Verdict = "not_measured"
		return out
	}
	if v, ok := verdict(*m.Value, m.Tolerance); ok && v == "outside" {
		out.Verdict = "outside"
	}
	return out
}

// verdict — значение против допуска; ok=false — единицы не совпадают.
func verdict(v ev.Measurement, t ev.Tolerance) (string, bool) {
	res := "within"
	for _, b := range []struct {
		m     *ev.Measurement
		lower bool
	}{{t.Lower, true}, {t.Upper, false}} {
		if b.m == nil {
			continue
		}
		if b.m.Unit != v.Unit {
			return "", false
		}
		c := compare(v, *b.m)
		if (b.lower && c < 0) || (!b.lower && c > 0) {
			res = "outside"
		}
	}
	return res, true
}

// compare — сравнение двух измерений одной единицы с разным масштабом.
func compare(a, b ev.Measurement) int {
	av, bv := int64(a.Value), int64(b.Value)
	for s := a.Scale; s < b.Scale; s++ {
		av *= 10
	}
	for s := b.Scale; s < a.Scale; s++ {
		bv *= 10
	}
	switch {
	case av < bv:
		return -1
	case av > bv:
		return 1
	}
	return 0
}

// measText — измерение текстом без float: мантисса, масштаб, единица.
func measText(m ev.Measurement) string {
	neg := m.Value < 0
	v := m.Value
	if neg {
		v = -v
	}
	digits := itoa(v)
	if m.Scale > 0 {
		for len(digits) <= m.Scale {
			digits = "0" + digits
		}
		digits = digits[:len(digits)-m.Scale] + "," + digits[len(digits)-m.Scale:]
	}
	if neg {
		digits = "−" + digits
	}
	return digits + " " + m.Unit
}

func tolText(t ev.Tolerance) string {
	parts := []string{}
	if t.Nominal != nil {
		parts = append(parts, "номинал "+measText(*t.Nominal))
	}
	if t.Lower != nil {
		parts = append(parts, "не менее "+measText(*t.Lower))
	}
	if t.Upper != nil {
		parts = append(parts, "не более "+measText(*t.Upper))
	}
	return strings.Join(parts, ", ")
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	b := []byte{}
	for v > 0 {
		b = append([]byte{byte('0' + v%10)}, b...)
		v /= 10
	}
	return string(b)
}

func bpPtr(b *ev.Bp) *int {
	if b == nil {
		return nil
	}
	v := int(*b)
	return &v
}
