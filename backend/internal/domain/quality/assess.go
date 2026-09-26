package quality

import (
	"slices"
	"strings"

	"ant/internal/contracts/normative"
	"ant/internal/contracts/statuses"
)

// Исходы карты реакций (FR-48).
const (
	ReactPassToNext = "pass_to_next"
	ReactManual     = "manual_review"
	ReactIsolate    = "isolate"
	ReactQuestion   = "question_to_technologist"
)

// Классы действий (AD-27, FR-144).
const (
	ClassRecord     = "record"
	ClassProtective = "protective"
	ClassPermissive = "permissive"
)

// Facts — вход карты реакций: вид × тяжесть × уверенность × качество
// наблюдения (FR-48) плюс основание, требование КД, уровень доверия паспорта.
type Facts struct {
	// Basis — основание сигнала (inspection_result, check_skipped, operator_report …).
	Basis string
	// Outcome — действующий исход наблюдения (FR-36); пусто у оснований не из контроля.
	Outcome        string
	DefectType     string
	TypeKnown      bool
	Severity       string
	ConfidenceBP   *int
	QualityBelow   bool
	HasRequirement bool
	// Analyzer, Trust — сигнал анализатора и уровень доверия паспорта (AD-29).
	Analyzer bool
	Trust    int
	StepKey  string
	// Closing — закрывающая точка: пропуск без человека запрещён (FR-44).
	Closing bool
	// CoverageOK — метод покрывает все виды дефектов точки (FR-14).
	CoverageOK bool
}

// Assessment — реакция по карте с ограничениями «ограничивать безопаснее,
// чем разрешать» (FR-144, AD-27): режим правила (FR-50), класс действия и
// уровень доверия паспорта анализатора (AD-29). Outcome пуст — реакции нет
// (например, «признаков нет» без права на автоматический пропуск: изделие
// ждёт человека на закрывающей точке, «годно» система не ставит).
type Assessment struct {
	// RuleID, RuleTitle — правило карты; MapRef — `‹id›@‹версия›#‹правило›`.
	RuleID    string `json:"rule_id"`
	RuleTitle string `json:"rule_title,omitempty"`
	MapRef    string `json:"map_ref"`
	// RuleMode — режим правила 1–5; Mode — с каким режимом реакция исполнена.
	RuleMode int `json:"rule_mode"`
	Mode     int `json:"mode"`
	// Outcome — итоговая реакция; Proposed — реакция карты до ограничений.
	Outcome  string `json:"outcome,omitempty"`
	Proposed string `json:"proposed,omitempty"`
	// Containment — уровень сдерживания, который предлагается модулю nonconformity.
	Containment statuses.Containment `json:"containment"`
	DraftNC     bool                 `json:"draft_nc,omitempty"`
	Task        string               `json:"task,omitempty"`
	// SampledRecheckBP — доля выборочной перепроверки пропущенных, б. п.
	SampledRecheckBP int    `json:"sampled_recheck_bp,omitempty"`
	ActionClass      string `json:"action_class"`
	// Limits — почему реакция ограничена: mode_requires_human,
	// trust_level_‹n›, permissive_not_delegated, critical_hold.
	Limits []string `json:"limits,omitempty"`
	// Suppressed — реакции нет по уровню доверия 0–1 (только запись или
	// рекомендация контролёру, AD-29).
	Suppressed bool `json:"suppressed,omitempty"`
	// Recommend — рекомендация контролёру (уровень доверия 1).
	Recommend bool `json:"recommend,omitempty"`
}

// containmentRank — строгость уровня сдерживания.
func containmentRank(c statuses.Containment) int {
	switch c {
	case statuses.ContainmentObserve:
		return 1
	case statuses.ContainmentAdditionalCheck:
		return 2
	case statuses.ContainmentItemHold:
		return 3
	case statuses.ContainmentLotHold:
		return 4
	}
	return 0
}

// outcomeRank — строгость исхода карты реакций.
func outcomeRank(o string) int {
	switch o {
	case ReactPassToNext:
		return 1
	case ReactQuestion:
		return 2
	case ReactManual:
		return 3
	case ReactIsolate:
		return 4
	}
	return 0
}

// sortedRules — правила карты по приоритету, затем по id (детерминированно).
func sortedRules(m normative.ReactionMap) []normative.ReactionMapRulesElem {
	rs := slices.Clone(m.Rules)
	slices.SortStableFunc(rs, func(a, b normative.ReactionMapRulesElem) int {
		if a.Priority != b.Priority {
			return a.Priority - b.Priority
		}
		return strings.Compare(a.ID, b.ID)
	})
	return rs
}

func has[T ~string](list []T, v string) bool {
	for _, x := range list {
		if string(x) == v {
			return true
		}
	}
	return false
}

// matches — условие правила карты на факты (FR-48). Отсутствующее поле
// условия — «любое значение».
func matches(r normative.ReactionMapRulesElem, f Facts) bool {
	m := r.Match
	if len(m.BasisKind) > 0 && !has(m.BasisKind, f.Basis) {
		return false
	}
	if len(m.BasisKind) == 0 && f.Basis != BasisInspection {
		// Правила без основания описывают результаты контроля.
		return false
	}
	if len(m.Outcome) > 0 && !has(m.Outcome, f.Outcome) {
		return false
	}
	if len(m.DefectTypes) > 0 && !has(m.DefectTypes, f.DefectType) {
		return false
	}
	if len(m.Severity) > 0 && !has(m.Severity, f.Severity) {
		return false
	}
	if m.ConfidenceMinBp != nil && (f.ConfidenceBP == nil || *f.ConfidenceBP < *m.ConfidenceMinBp) {
		return false
	}
	if m.NoRequirement != nil && *m.NoRequirement == f.HasRequirement {
		return false
	}
	if m.ObservationQualityBelowRecipe != nil && *m.ObservationQualityBelowRecipe != f.QualityBelow {
		return false
	}
	if m.TrustLevelMin != nil && (!f.Analyzer || f.Trust < *m.TrustLevelMin) {
		return false
	}
	if len(m.DeviationKinds) > 0 || m.SpecialProcess != nil {
		// Отклонения оборудования приходят в поток оборудования (machinelogs,
		// crossitem); в свёртке изделия таких фактов нет.
		return false
	}
	return true
}

// Assess — реакция карты реакций на факты с ограничениями (FR-48, FR-50,
// FR-144, AD-27, AD-29). Порядок: первое подходящее правило по приоритету →
// режим правила → класс действия → уровень доверия паспорта → «критическая
// тяжесть — не ниже блока».
func Assess(env Env, f Facts) Assessment {
	a := Assessment{MapRef: env.mapRef()}
	matched := false
	for _, r := range sortedRules(env.ReactionMap) {
		if !matches(r, f) {
			continue
		}
		matched = true
		a.RuleID, a.RuleTitle, a.RuleMode = r.ID, r.Title, r.AutomationMode
		a.Outcome = string(r.Reaction.Outcome)
		a.Containment = statuses.Containment(r.Reaction.ContainmentLevel)
		a.DraftNC = r.Reaction.DraftNc
		if r.Reaction.Task != nil {
			a.Task = string(*r.Reaction.Task)
		}
		if r.Reaction.SampledRecheckBp != nil {
			a.SampledRecheckBP = *r.Reaction.SampledRecheckBp
		}
		break
	}
	if !matched {
		if f.Basis == BasisInspection && f.Outcome == OutcomeNoDefect {
			// «Признаков нет» без правила — реакции нет: изделие идёт к
			// человеку на закрывающей точке, «годно» не ставится (FR-36).
			return Assessment{MapRef: a.MapRef, RuleID: "none", ActionClass: ClassRecord, Containment: statuses.ContainmentNone}
		}
		// Нет правила — к человеку (ограничивать безопаснее, FR-144).
		a.RuleID, a.RuleTitle, a.RuleMode = "default", "Нет правила карты — к человеку", 1
		a.Outcome, a.Containment, a.Task = ReactManual, statuses.ContainmentAdditionalCheck, "decision_required"
		if f.Outcome == OutcomeUnable {
			a.Task = "recheck"
		}
	}
	a.MapRef += "#" + a.RuleID
	a.Mode = max(a.RuleMode, 1)

	// Режимы 3–5 — система только предлагает, решает человек (FR-50).
	if a.RuleMode >= 3 && a.Outcome == ReactIsolate {
		a.Proposed, a.Outcome = a.Outcome, ReactManual
		a.Containment = capContainment(a.Containment, statuses.ContainmentAdditionalCheck)
		a.Limits = append(a.Limits, "mode_requires_human")
		a.Mode = 1
	}

	// Разрешающее действие — только по явно делегированному правилу режима 2,
	// анализатором с уровнем доверия 4, при достаточном качестве наблюдения,
	// не на закрывающей точке и при полном покрытии видов методом (AD-27, AD-29, FR-14).
	if a.Outcome == ReactPassToNext {
		ok := a.RuleMode == 2 && f.Analyzer && f.Trust >= 4 && !f.QualityBelow && !f.Closing && f.CoverageOK && f.Outcome == OutcomeNoDefect
		if !ok {
			return Assessment{MapRef: a.MapRef, RuleID: a.RuleID, RuleTitle: a.RuleTitle, RuleMode: a.RuleMode, Mode: 1,
				Proposed: ReactPassToNext, ActionClass: ClassRecord, Containment: statuses.ContainmentNone,
				Limits: []string{"permissive_not_delegated"}}
		}
		a.ActionClass = ClassPermissive
		return a
	}
	a.ActionClass = ClassProtective

	// Уровень доверия паспорта анализатора (AD-29): 0 — только запись;
	// 1 — рекомендация контролёру; 2 — «под подозрением» и доп. контроль;
	// 3 — дополнительно блок и изоляция.
	if f.Analyzer {
		switch {
		case f.Trust <= 0:
			a.Suppressed = true
			a.Limits = append(a.Limits, "trust_level_0")
		case f.Trust == 1:
			a.Suppressed, a.Recommend = true, true
			a.Limits = append(a.Limits, "trust_level_1")
		case f.Trust == 2:
			if a.Outcome == ReactIsolate {
				a.Proposed, a.Outcome = ReactIsolate, ReactManual
				a.Limits = append(a.Limits, "trust_level_2")
			}
			a.Containment = capContainment(a.Containment, statuses.ContainmentAdditionalCheck)
		}
	}

	// Критическая тяжесть — не ниже блока при любой уверенности (FR-48:
	// «низкая уверенность при критической тяжести всё равно ведёт к блоку»).
	// Блок — защитное обратимое действие (режим 1, AD-27); анализатору ниже
	// уровня доверия 3 блок не делегирован — решает человек.
	if f.Severity == "critical" && !a.Suppressed && (!f.Analyzer || f.Trust >= 3) &&
		containmentRank(a.Containment) < containmentRank(statuses.ContainmentItemHold) {
		a.Containment = statuses.ContainmentItemHold
		if a.Outcome == ReactManual {
			a.Outcome = ReactIsolate
			a.Task = "isolate_move"
		}
		a.DraftNC = a.DraftNC || a.Outcome == ReactIsolate
		a.Limits = append(a.Limits, "critical_hold")
	}
	return a
}

// capContainment — уровень сдерживания не выше cap.
func capContainment(c, limit statuses.Containment) statuses.Containment {
	if containmentRank(c) > containmentRank(limit) {
		return limit
	}
	return c
}

// stricter — строже из двух реакций одного физического дефекта: повторное
// наблюдение не ослабляет защиту (AD-3).
func stricter(a, b Assessment) Assessment {
	if a.Outcome == "" {
		return b
	}
	if outcomeRank(b.Outcome) > outcomeRank(a.Outcome) ||
		(outcomeRank(b.Outcome) == outcomeRank(a.Outcome) && containmentRank(b.Containment) > containmentRank(a.Containment)) {
		b.DraftNC = b.DraftNC || a.DraftNC
		return b
	}
	a.DraftNC = a.DraftNC || b.DraftNC
	if containmentRank(b.Containment) > containmentRank(a.Containment) {
		a.Containment = b.Containment
	}
	return a
}
