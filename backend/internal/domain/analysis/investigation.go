package analysis

import (
	"fmt"
	"strings"

	"ant/internal/contracts/errcodes"
	"ant/internal/domain/kernel"
)

// Расследование инцидента на столе технолога: стадия, «что сделать следующим»
// и гард закрытия (FR-59, FR-61, FR-64; кейс §2.3 «две причины»). Стадия —
// словарь investigation_stage contracts/statuses.yaml. Чистые функции над
// фактами расследования — их считают и live (из проекции analysis.incident),
// и мир заготовок, чтобы стол видел одно и то же.

// Стадии расследования (словарь investigation_stage).
const (
	StageScopeDefined       = "scope_defined"
	StageHypothesis         = "hypothesis"
	StageCauseConfirmed     = "cause_confirmed"
	StageActionAssigned     = "action_assigned"
	StageEffectivenessCheck = "effectiveness_check"
	StageClosed             = "closed"
)

// Ветки причины (кейс §2.3): почему возник и почему не обнаружили раньше.
const (
	BranchWhyMade   = "why_made"
	BranchWhyMissed = "why_missed"
)

// Branches — обе ветки причины в порядке вопросов человека.
var Branches = []string{BranchWhyMade, BranchWhyMissed}

// BranchTitle — ветка словами.
func BranchTitle(b string) string {
	switch b {
	case BranchWhyMissed:
		return "почему не остановили раньше"
	default:
		return "почему возник"
	}
}

// Что закрывает incident.incident.closed (поле scope).
const (
	CloseRiskScope     = "risk_scope"
	CloseInvestigation = "investigation"
)

// KnownCounts — изделия текущей области по оси «что известно» (FR-62).
type KnownCounts struct {
	Confirmed int `json:"confirmed"`
	Suspect   int `json:"suspect"`
	Unknown   int `json:"unknown"`
	Excluded  int `json:"excluded"`
}

// Add — учесть изделие со статусом в инциденте.
func (c *KnownCounts) Add(status string) {
	switch status {
	case StatusConfirmed:
		c.Confirmed++
	case StatusSuspect:
		c.Suspect++
	case StatusUnknown:
		c.Unknown++
	case StatusExcluded:
		c.Excluded++
	}
}

// InvestigationFacts — что известно о расследовании к моменту чтения.
type InvestigationFacts struct {
	IncidentID string
	// Closed — расследование закрыто (incident.incident.closed со scope=investigation).
	Closed bool
	// ScopeClosed — область риска закрыта (решение по изделиям принято).
	ScopeClosed bool
	// Narrowed — было хотя бы одно сужение области.
	Narrowed bool
	Counts   KnownCounts
	// Hypotheses — живых (не отклонённых) гипотез.
	Hypotheses int
	// NextCheck — «что проверить следующим» у ведущей гипотезы (текст).
	NextCheck string
	// Causes — ветка → вывод (confirmed | not_established).
	Causes map[string]string
	// Actions — статусы мер: assigned | implemented | effective | failed | reopened.
	Actions []string
}

// InvestigationStage — стадия расследования: от закрытия назад к области.
func InvestigationStage(f InvestigationFacts) string {
	var implemented, evaluated, assigned bool
	for _, a := range f.Actions {
		switch a {
		case ActionAssigned, ActionReopened:
			assigned = true
		case ActionImplemented:
			implemented = true
		default:
			evaluated = true
		}
	}
	switch {
	case f.Closed:
		return StageClosed
	case implemented || evaluated:
		return StageEffectivenessCheck
	case assigned:
		return StageActionAssigned
	case len(f.Causes) > 0:
		return StageCauseConfirmed
	case f.Hypotheses > 0:
		return StageHypothesis
	}
	return StageScopeDefined
}

// InvestigationNextStep — «что сделать следующим» одной строкой (текст
// сервера); пусто — делать нечего (расследование закрыто).
func InvestigationNextStep(f InvestigationFacts) string {
	if f.Closed {
		return ""
	}
	open := f.Counts.Suspect + f.Counts.Unknown
	if !f.ScopeClosed && open > 0 && (!f.Narrowed || len(f.Causes) == 0 && f.NextCheck == "") {
		return fmt.Sprintf("Сузить область риска по основаниям: %s под подозрением, %s без данных — исключать только с доказательством",
			items(f.Counts.Suspect), items(f.Counts.Unknown))
	}
	if _, ok := f.Causes[BranchWhyMade]; !ok {
		if f.NextCheck != "" {
			return "Проверить ведущую гипотезу: " + lowerFirst(f.NextCheck)
		}
		if f.Hypotheses == 0 {
			return "Записать гипотезу причины «почему возник» с доводами"
		}
		return "Подтвердить причину «почему возник» проверкой или записать «причина не установлена»"
	}
	if _, ok := f.Causes[BranchWhyMissed]; !ok {
		return "Ответить, почему не остановили раньше: гипотеза ветки «почему пропустили» и её проверка"
	}
	var assigned, implemented, failed int
	for _, a := range f.Actions {
		switch a {
		case ActionAssigned, ActionReopened:
			assigned++
		case ActionImplemented:
			implemented++
		case "failed":
			failed++
		}
	}
	switch {
	case len(f.Actions) == 0:
		return "Назначить меры по обеим причинам: предупредить повторение и улучшить обнаружение, у каждой — план проверки эффективности"
	case failed > 0:
		return fmt.Sprintf("Мера не помогла (%d): пересмотреть причину или назначить новую меру", failed)
	case assigned > 0:
		return fmt.Sprintf("Внедрить меры: %d из %d ещё не внедрены", assigned, len(f.Actions))
	case implemented > 0:
		return fmt.Sprintf("Проверить эффективность мер по плану: %d ждут оценки", implemented)
	}
	return "Закрыть расследование: обе причины отвечены, меры эффективны"
}

func items(n int) string {
	return fmt.Sprintf("%d изд.", n)
}

// CloseBlocker — почему расследование ещё нельзя закрыть: код отказа
// (contracts/errors.yaml), параметры и текст для людей.
type CloseBlocker struct {
	Code   errcodes.Code
	Params []string
	Text   string
}

// CloseBlockers — всё, что мешает закрыть расследование (FR-64, кейс §2.3):
// ветка причины без вывода (подтверждена или «не установлена»), нет мер или
// мера не признана эффективной. Пусто — закрыть можно.
func CloseBlockers(f InvestigationFacts) []CloseBlocker {
	if f.Closed {
		return []CloseBlocker{{Code: errcodes.IncidentInvestigationClosed, Params: []string{"incident_id", f.IncidentID}, Text: "Расследование уже закрыто"}}
	}
	var out []CloseBlocker
	for _, b := range Branches {
		if _, ok := f.Causes[b]; !ok {
			out = append(out, CloseBlocker{Code: errcodes.IncidentCauseBranchOpen,
				Params: []string{"incident_id", f.IncidentID, "branch", b, "branch_title", BranchTitle(b)},
				Text:   "Нет вывода о причине «" + BranchTitle(b) + "»: подтвердите причину или запишите «причина не установлена»"})
		}
	}
	why := ""
	if len(f.Actions) == 0 {
		why = "мер не назначено"
	} else {
		var open []string
		for _, a := range f.Actions {
			if a != ActionEffective {
				open = append(open, a)
			}
		}
		if len(open) > 0 {
			why = fmt.Sprintf("%d из %d мер не признаны эффективными (%s)", len(open), len(f.Actions), strings.Join(open, ", "))
		}
	}
	if why != "" {
		out = append(out, CloseBlocker{Code: errcodes.IncidentEffectivenessUnchecked, Params: []string{"incident_id", f.IncidentID, "why", why},
			Text: "Эффективность мер не проверена: " + why})
	}
	return out
}

// GuardCloseInvestigation — закрыть расследование можно, когда блокеров нет;
// отказ — первым блокером. Область риска закрывается отдельно и раньше
// (S05) — этот гард её не касается.
func GuardCloseInvestigation(f InvestigationFacts) error {
	if bs := CloseBlockers(f); len(bs) > 0 {
		return kernel.Refuse(bs[0].Code, bs[0].Params...)
	}
	return nil
}
