package quality

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"ant/internal/contracts/normative"
)

// Env — часть нормативного слоя, закреплённая при запуске изделия (AD-17), и
// срез справочников на момент наблюдения (AD-31), нужные модулю quality:
// классификатор видов дефектов (FR-125), карта реакций (FR-48), точки контроля
// шагов процесса с методом и покрытием (FR-12, FR-14), зоны изделия и
// паспорта допуска анализатора (AD-29). Собирает BundleSource движка (эпик 17:
// process) функциями application/quality (EnvFromFiles) и PassportsFrom.
//
// Пустой Env — «нормы нет»: вид любого дефекта неизвестен, у карты контроля
// нет допущенного анализатора (уровень доверия 0), карта реакций пуста — и
// каждый признак дефекта или «оценка невозможна» идёт к человеку правилом по
// умолчанию (ограничивать безопаснее, чем разрешать, FR-144).
type Env struct {
	// RuleRev — ревизия нормативного слоя (normative_rev реакций, FR-50).
	RuleRev string `json:"rule_rev,omitempty"`
	// Classifier — классификатор видов дефектов (normative/defects, FR-125).
	Classifier normative.DefectClassifier `json:"classifier"`
	// ReactionMap — карта реакций (normative/reactions, FR-48).
	ReactionMap normative.ReactionMap `json:"reaction_map"`
	// Steps — шаги процесса со свойствами контроля (ant:inspection,
	// ant:requirement, ant:zoneRef, точки контроля и закрывающие точки) в
	// порядке описания процесса.
	Steps []StepSpec `json:"steps,omitempty"`
	// Zones — зоны изделия с видом (reference item-types, FR-46).
	Zones []ZoneSpec `json:"zones,omitempty"`
	// Passports — паспорта допуска анализатора с историей статусов (AD-29).
	Passports []Passport `json:"passports,omitempty"`
}

// IsZero — Env не собран (пустой нормативный слой).
func (e Env) IsZero() bool {
	return e.RuleRev == "" && len(e.Classifier.DefectTypes) == 0 && len(e.ReactionMap.Rules) == 0 &&
		len(e.Steps) == 0 && len(e.Zones) == 0 && len(e.Passports) == 0
}

// StepSpec — шаг процесса глазами контроля качества (расширение BPMN
// urn:ant:bpmn-ext:1, AD-17).
type StepSpec struct {
	// StepKey — стабильный ключ шага.
	StepKey string `json:"step_key"`
	// Order — порядок шага в описании процесса.
	Order int `json:"order"`
	// Stage — участок процесса (первый сегмент step_key: incoming, welding …).
	Stage string `json:"stage"`
	// StepKind — operation | automated_inspection | human_inspection | movement | storage.
	StepKind string `json:"step_kind,omitempty"`
	// InspectionPoint — точка контроля KT-…; ClosingPoint — закрывающая точка ZT-….
	InspectionPoint string `json:"inspection_point,omitempty"`
	ClosingPoint    string `json:"closing_point,omitempty"`
	// SpecialProcess — специальный процесс (FR-151).
	SpecialProcess bool `json:"special_process,omitempty"`
	// Inspections — методы контроля шага с покрытием видов дефектов (FR-14).
	Inspections []InspectionSpec `json:"inspections,omitempty"`
	// Requirements — требования КД шага (FR-12, FR-48: «нет требования —
	// вопрос технологу»).
	Requirements []Requirement `json:"requirements,omitempty"`
	// Zones — зоны изделия шага.
	Zones []string `json:"zones,omitempty"`
	// ReactionMapRef — карта реакций шага `‹id›@‹версия›`.
	ReactionMapRef string `json:"reaction_map_ref,omitempty"`
}

// IsInspection — шаг контроля (КТ или закрывающая точка с методом).
func (s StepSpec) IsInspection() bool {
	return s.StepKind == "automated_inspection" || s.StepKind == "human_inspection" || len(s.Inspections) > 0
}

// InspectionSpec — метод контроля шага (ant:inspection, FR-14).
type InspectionSpec struct {
	Method string `json:"method"`
	Phase  string `json:"phase,omitempty"`
	// Coverage — коды видов дефектов, которые метод на этой точке способен выявить.
	Coverage []string `json:"coverage,omitempty"`
	// RecipeRef — карта контроля `‹id›@‹версия›` (для анализатора — ключ паспорта допуска).
	RecipeRef string `json:"recipe_ref,omitempty"`
	// ObservationQualityMinBP — порог качества наблюдения, б. п.: ниже — «оценка
	// невозможна» (FR-36); 0 — порог не задан.
	ObservationQualityMinBP int `json:"observation_quality_min_bp,omitempty"`
}

// Requirement — требование КД (ant:requirement).
type Requirement struct {
	Characteristic string `json:"characteristic,omitempty"`
	Tolerance      string `json:"tolerance,omitempty"`
	KDRef          string `json:"kd_ref,omitempty"`
}

// Ref — строка ссылки на требование для записи сигнала.
func (r Requirement) Ref() string {
	parts := []string{}
	for _, p := range []string{r.KDRef, r.Characteristic, r.Tolerance} {
		if p != "" {
			parts = append(parts, p)
		}
	}
	s := strings.Join(parts, "; ")
	if len(s) > 256 {
		s = s[:256]
	}
	return s
}

// ZoneSpec — зона изделия и её вид (surface, weld_section, joint …).
type ZoneSpec struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
}

// Passport — паспорт допуска анализатора для карты контроля (AD-29, FR-98):
// уровень доверия и история статусов (допущен, приостановлен, возвращён,
// выведен). Откат действует на будущее: наблюдение оценивается по статусу
// паспорта на своё occurred_at, старые наблюдения сохраняют «проанализировано
// версией …».
type Passport struct {
	PassportID      string `json:"passport_id"`
	RecipeRef       string `json:"recipe_ref"`
	AnalyzerVersion string `json:"analyzer_version"`
	// Stage — shadow | pilot | active; тень (shadow) — всегда уровень 0.
	Stage      string `json:"stage"`
	TrustLevel int    `json:"trust_level"`
	// History — смены статуса по времени: допуск, приостановка, возврат, вывод.
	History []PassportStatus `json:"history"`
}

// PassportStatus — статус паспорта с момента At.
type PassportStatus struct {
	At      time.Time `json:"at"`
	Active  bool      `json:"active"`
	EventID string    `json:"event_id"`
	// Note — admitted | suspended:‹trigger› | reinstated | retired.
	Note string `json:"note"`
}

// activeAt — статус паспорта на момент t: последняя смена не позже t.
func (p Passport) activeAt(t time.Time) (bool, string) {
	active, note := false, "not_admitted"
	for _, h := range p.History {
		if h.At.After(t) {
			break
		}
		active, note = h.Active, h.Note
	}
	return active, note
}

// DefectType — вид дефекта классификатора.
type DefectType = normative.DefectClassifierDefectTypesElem

// defectType — вид по коду классификатора.
func (e Env) defectType(code string) (DefectType, bool) {
	for _, t := range e.Classifier.DefectTypes {
		if t.Code == code {
			return t, true
		}
	}
	return DefectType{}, false
}

// methodDetects — метод способен выявить вид дефекта по классификатору (FR-14).
func (e Env) methodDetects(method, code string) bool {
	t, ok := e.defectType(code)
	if !ok {
		return false
	}
	for _, m := range t.Methods {
		if string(m) == method {
			return true
		}
	}
	return false
}

// step — шаг по ключу.
func (e Env) step(key string) (StepSpec, bool) {
	if key == "" {
		return StepSpec{}, false
	}
	for _, s := range e.Steps {
		if s.StepKey == key {
			return s, true
		}
	}
	return StepSpec{}, false
}

// pointFor — шаг контроля, к которому относится наблюдение: по step_key, иначе
// по точке контроля и методу.
func (e Env) pointFor(stepKey, point, method string) (StepSpec, bool) {
	if s, ok := e.step(stepKey); ok {
		return s, true
	}
	if point == "" {
		return StepSpec{}, false
	}
	for _, s := range e.Steps {
		if s.InspectionPoint != point && s.ClosingPoint != point {
			continue
		}
		for _, in := range s.Inspections {
			if in.Method == method {
				return s, true
			}
		}
	}
	return StepSpec{}, false
}

// zoneKind — вид зоны; "" — зона неизвестна справочнику.
func (e Env) zoneKind(zone string) string {
	for _, z := range e.Zones {
		if z.ID == zone {
			return z.Kind
		}
	}
	// Участок зоны (W-1.U2) — вид родительской зоны (W-1).
	if i := strings.LastIndexByte(zone, '.'); i > 0 {
		return e.zoneKind(zone[:i])
	}
	return ""
}

// mapRef — ссылка на карту реакций `‹id›@‹версия›`.
func (e Env) mapRef() string {
	if e.ReactionMap.ID == "" {
		return "no-reaction-map@0"
	}
	return e.ReactionMap.ID + "@" + strconv.Itoa(e.ReactionMap.Version)
}

// stageOf — участок процесса шага: из плана, иначе первый сегмент step_key.
func (e Env) stageOf(stepKey string) string {
	if s, ok := e.step(stepKey); ok && s.Stage != "" {
		return s.Stage
	}
	stage, _, _ := strings.Cut(stepKey, ".")
	return stage
}

// passportFor — паспорт допуска карты контроля на момент наблюдения (AD-29):
// уровень доверия и пояснение. Нет паспорта, паспорт приостановлен или
// выведен, анализатор не той версии, теневой режим — уровень 0: наблюдение
// только записывается, контроль ручной.
func (e Env) passportFor(recipe, analyzer string, at time.Time) (Passport, int, string) {
	if recipe == "" {
		return Passport{}, 0, "no_recipe"
	}
	cands := []Passport{}
	for _, p := range e.Passports {
		if p.RecipeRef == recipe {
			cands = append(cands, p)
		}
	}
	if len(cands) == 0 {
		return Passport{}, 0, "no_qualified_analyzer"
	}
	slices.SortFunc(cands, func(a, b Passport) int { return strings.Compare(a.PassportID, b.PassportID) })
	note := "no_qualified_analyzer"
	for _, p := range cands {
		active, n := p.activeAt(at)
		if !active {
			note = n
			continue
		}
		if analyzer != "" && p.AnalyzerVersion != "" && p.AnalyzerVersion != analyzer {
			note = "analyzer_version_mismatch"
			continue
		}
		if p.Stage == "shadow" {
			return p, 0, "shadow"
		}
		return p, min(max(p.TrustLevel, 0), 4), "admitted"
	}
	return Passport{}, 0, note
}
