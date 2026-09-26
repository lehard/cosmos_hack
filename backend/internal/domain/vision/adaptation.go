package vision

import (
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"

	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
)

// Контур адаптации VisionQC (эпик 40): что делает анализатор, по каким
// версиям и почему — для людей, которые решают о допуске и откате.
//   - Seen — наблюдение анализатора с вектором версий (AD-29);
//   - Explain — по давнему наблюдению: какими версиями и почему изделие
//     признано подозрительным, какой паспорт действовал тогда (FR-98);
//   - RecheckList — пропуск брака: изделия, пропущенные той же версией модели
//     в той же зоне, — на перепроверку (FR-100).

// SeenStage — ступень анализатора (FR-38).
type SeenStage struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	ConfidenceBP *int   `json:"confidence_bp,omitempty"`
	Output       string `json:"output,omitempty"`
}

// SeenDefect — признак дефекта в ответе анализатора.
type SeenDefect struct {
	Code         string `json:"code,omitempty"`
	Zone         string `json:"zone"`
	Severity     string `json:"severity"`
	ConfidenceBP *int   `json:"confidence_bp,omitempty"`
}

// Seen — наблюдение анализатора (inspection.result.recorded камерой).
type Seen struct {
	EventID         string       `json:"event_id"`
	ItemID          string       `json:"item_id,omitempty"`
	RunID           string       `json:"run_id,omitempty"`
	OccurredAt      time.Time    `json:"occurred_at"`
	Point           string       `json:"point,omitempty"`
	StepKey         string       `json:"step_key,omitempty"`
	Outcome         string       `json:"outcome"`
	ProcessingState string       `json:"processing_state"`
	QualityBP       *int         `json:"quality_bp,omitempty"`
	ConfidenceBP    *int         `json:"confidence_bp,omitempty"`
	Zones           []string     `json:"zones,omitempty"`
	Versions        Versions     `json:"versions"`
	Stages          []SeenStage  `json:"stages,omitempty"`
	Defects         []SeenDefect `json:"defects,omitempty"`
	Limitations     []string     `json:"limitations,omitempty"`
}

// SeenFrom — наблюдение анализатора из записи; не камера — false.
func SeenFrom(r kernel.Record) (Seen, bool) {
	if r.Type != catalog.InspectionResultRecorded {
		return Seen{}, false
	}
	d, err := kernel.Decode[ev.InspectionResultRecordedV1](r)
	if err != nil || (d.Method != ev.InspectionMethodCamera && len(d.Stages) == 0) {
		return Seen{}, false
	}
	o := Seen{EventID: r.EventID, ItemID: r.ItemID, RunID: r.RunID, OccurredAt: r.OccurredAt, Outcome: string(d.Outcome),
		ProcessingState: string(d.ProcessingState), QualityBP: bp(d.ObservationQualityBp), ConfidenceBP: bp(d.AnalyzerConfidenceBp),
		Limitations: d.Limitations}
	if d.InspectionPoint != nil {
		o.Point = *d.InspectionPoint
	}
	if d.StepKey != nil {
		o.StepKey = string(*d.StepKey)
	}
	if d.Versions != nil {
		o.Versions = FromContract(*d.Versions)
	}
	for _, z := range d.ZoneIds {
		o.Zones = append(o.Zones, string(z))
	}
	for _, st := range d.Stages {
		s := SeenStage{Name: string(st.Stage), Version: st.Version, ConfidenceBP: bp(st.ConfidenceBp)}
		if st.OutputNote != nil {
			s.Output = *st.OutputNote
		}
		o.Stages = append(o.Stages, s)
	}
	for _, df := range d.Defects {
		x := SeenDefect{Zone: string(df.ZoneID), Severity: string(df.Severity), ConfidenceBP: bp(df.StageConfidenceBp)}
		if df.DefectTypeCode != nil {
			x.Code = *df.DefectTypeCode
		}
		o.Defects = append(o.Defects, x)
		if !slices.Contains(o.Zones, x.Zone) {
			o.Zones = append(o.Zones, x.Zone)
		}
	}
	return o, true
}

// outcomeText — исход по-русски.
func outcomeText(o string) string {
	switch o {
	case string(ev.InspectionOutcomeDefectIndicated):
		return "признаки дефекта"
	case string(ev.InspectionOutcomeNoDefectIndicated):
		return "признаков нет"
	case string(ev.InspectionOutcomeUnableToAssess):
		return "оценка невозможна"
	}
	return o
}

func pct(p *int) string {
	if p == nil {
		return "не сообщено"
	}
	return Share(*p)
}

// PassportFor — паспорт карты контроля и версии анализатора наблюдения (как
// его выбирает quality: карта контроля + версия анализатора, AD-29).
func (r Registry) PassportFor(v Versions) (Passport, bool) {
	for _, p := range r.Passports {
		if p.RecipeRef == v.RecipeRef && p.Versions.AnalyzerVersion == v.AnalyzerVersion {
			return p, true
		}
	}
	return Passport{}, false
}

// Account — ответ «какими версиями и почему» по наблюдению (FR-98).
type Account struct {
	Seen    Seen     `json:"seen"`
	Missing []string `json:"missing_versions,omitempty"`
	// PassportID, StatusThen, LevelThen — паспорт на момент наблюдения и
	// уровень доверия, с которым система реагировала (0 — только запись).
	PassportID  string   `json:"passport_id,omitempty"`
	StatusThen  string   `json:"status_then"`
	StatusNow   string   `json:"status_now"`
	Stage       string   `json:"stage,omitempty"`
	LevelThen   int      `json:"level_then"`
	AllowedThen []string `json:"allowed_then"`
	// Suspicious — изделие признано подозрительным по этому наблюдению.
	Suspicious bool     `json:"suspicious"`
	Reasons    []string `json:"reasons"`
}

// Explain — «какими версиями и почему» по наблюдению: вектор версий, ступени,
// признаки, паспорт на момент наблюдения (откат действует на будущее — старое
// наблюдение сохраняет «проанализировано версией …»). Чистая функция (AD-4).
func Explain(o Seen, reg Registry) Account {
	a := Account{Seen: o, StatusThen: StatusNotAdmitted, StatusNow: StatusNotAdmitted, AllowedThen: AllowedAutoActions(0)}
	a.Missing = o.Versions.Missing()
	v := o.Versions
	a.Reasons = append(a.Reasons, "Анализатор "+nz(v.AnalyzerVersion)+" по карте контроля "+nz(v.RecipeRef)+" ("+o.OccurredAt.UTC().Format("02.01.2006 15:04")+
		" UTC): исход «"+outcomeText(o.Outcome)+"», уверенность "+pct(o.ConfidenceBP)+", качество кадра "+pct(o.QualityBP)+".")
	if len(o.Stages) > 0 {
		parts := []string{}
		for _, st := range o.Stages {
			s := st.Name + " (" + st.Version + ", " + pct(st.ConfidenceBP) + ")"
			if st.Output != "" {
				s += ": " + st.Output
			}
			parts = append(parts, s)
		}
		a.Reasons = append(a.Reasons, "Ступени анализатора: "+strings.Join(parts, " → ")+".")
	}
	for _, d := range o.Defects {
		a.Reasons = append(a.Reasons, "Признак "+nz(d.Code)+" в зоне "+d.Zone+", тяжесть "+d.Severity+", уверенность ступени "+pct(d.ConfidenceBP)+".")
	}
	if len(a.Missing) > 0 {
		a.Reasons = append(a.Reasons, "Вектор версий неполный — неизвестно: "+strings.Join(a.Missing, ", ")+" (уровень доверия 0).")
	}
	p, ok := reg.PassportFor(v)
	if !ok {
		a.Reasons = append(a.Reasons, "Для этой карты контроля и версии анализатора нет допущенного паспорта — только запись, контроль ручной.")
	} else {
		a.PassportID, a.Stage = p.PassportID, p.Stage
		a.StatusThen, a.StatusNow = p.StatusAt(o.OccurredAt), p.Current().Status
		if a.StatusThen == StatusActive && p.Stage != "shadow" && len(a.Missing) == 0 {
			a.LevelThen = min(max(p.TrustLevel, 0), 4)
		}
		a.AllowedThen = AllowedAutoActions(a.LevelThen)
		a.Reasons = append(a.Reasons, "Паспорт "+p.PassportID+" на момент наблюдения: "+statusText(a.StatusThen)+", стадия "+p.Stage+
			", уровень доверия "+itoa(a.LevelThen)+" — допустимо: "+strings.Join(a.AllowedThen, ", ")+".")
		for _, h := range p.History {
			if h.Status == StatusSuspended && h.At.After(o.OccurredAt) {
				a.Reasons = append(a.Reasons, "Позже ("+h.At.UTC().Format("02.01.2006 15:04")+" UTC) паспорт приостановлен: "+h.Trigger+
					" — откат действует на будущее; это наблюдение сохраняет «проанализировано версией "+nz(v.AnalyzerVersion)+"».")
				break
			}
		}
	}
	a.Suspicious = o.Outcome == string(ev.InspectionOutcomeDefectIndicated) && a.LevelThen >= 2
	switch {
	case a.Suspicious:
		a.Reasons = append(a.Reasons, "Итог: признак дефекта при уровне доверия "+itoa(a.LevelThen)+" — изделие поставлено «под подозрением» автоматически, решение — за контролёром.")
	case o.Outcome == string(ev.InspectionOutcomeDefectIndicated):
		a.Reasons = append(a.Reasons, "Итог: признак дефекта — рекомендация контролёру (уровень доверия "+itoa(a.LevelThen)+" не даёт автоматических ограничений).")
	}
	return a
}

func statusText(s string) string {
	switch s {
	case StatusActive:
		return "действовал"
	case StatusSuspended:
		return "приостановлен"
	case StatusRetired:
		return "выведен"
	}
	return "не допущен"
}

func nz(s string) string {
	if s == "" {
		return Unknown
	}
	return s
}

func itoa(n int) string { return strconv.Itoa(n) }

// Recheck — изделие на перепроверку после пропуска брака (FR-100).
type Recheck struct {
	ItemID         string    `json:"item_id"`
	ObservationIDs []string  `json:"observation_ids"`
	LastAt         time.Time `json:"last_at"`
}

// RecheckList — изделия, которые та же версия анализатора по той же карте
// контроля признала «признаков нет» в той же зоне, что и пропущенные
// наблюдения (FR-100): их вывод мог быть той же ошибкой модели. Изделие, где
// брак уже найден, в список не входит. Порядок — по изделию.
func RecheckList(missed, all []Seen, escapedItem string) []Recheck {
	by := map[string]*Recheck{}
	for _, c := range all {
		if c.ItemID == "" || c.ItemID == escapedItem || c.Outcome != string(ev.InspectionOutcomeNoDefectIndicated) {
			continue
		}
		if !slices.ContainsFunc(missed, func(m Seen) bool { return sameModelZone(m, c) }) {
			continue
		}
		rc := by[c.ItemID]
		if rc == nil {
			rc = &Recheck{ItemID: c.ItemID}
			by[c.ItemID] = rc
		}
		if !slices.Contains(rc.ObservationIDs, c.EventID) {
			rc.ObservationIDs = append(rc.ObservationIDs, c.EventID)
		}
		if c.OccurredAt.After(rc.LastAt) {
			rc.LastAt = c.OccurredAt
		}
	}
	out := []Recheck{}
	for _, k := range slices.Sorted(maps.Keys(by)) {
		slices.Sort(by[k].ObservationIDs)
		out = append(out, *by[k])
	}
	return out
}

// sameModelZone — та же версия анализатора и карта контроля, общая зона (без
// зон — та же точка контроля).
func sameModelZone(m, c Seen) bool {
	if m.Versions.AnalyzerVersion == "" || m.Versions.AnalyzerVersion != c.Versions.AnalyzerVersion || m.Versions.RecipeRef != c.Versions.RecipeRef {
		return false
	}
	if len(m.Zones) == 0 || len(c.Zones) == 0 {
		return m.Point != "" && m.Point == c.Point
	}
	for _, z := range m.Zones {
		if slices.Contains(c.Zones, z) {
			return true
		}
	}
	return false
}
