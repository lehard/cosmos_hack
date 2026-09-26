package nonconformity

import (
	"strings"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/statuses"
	"ant/internal/domain/kernel"
	"ant/internal/domain/process"
	"ant/internal/domain/quality"
)

// React вычисляет реакции модуля по состоянию после записи и намерения к
// ранним модулям (AD-3, AD-40). Реакции строятся только через
// kernel.NewReaction с собственными типами модуля:
//   - decision.nonconformity.drafted — черновик каждого несоответствия по
//     сигналу (режим 1, FR-50); слот: nonconformity.draft / item / nc_id;
//   - decision.containment.applied — действующее сдерживание правилом (FR-49,
//     FR-62, FR-151); слот: правило / item / ключ основания. Если основание
//     правила ушло ниже уровня (снятие не делегировано), реакция исчезает —
//     движок не снимает защиту, а ставит задачу человеку (AD-3), блок в оси
//     остаётся до решения человека (AD-27).
//
// Намерения — только на шаге своей записи (State.Effects): quality.SetQuality
// (подтверждение, «годно по разрешению»), process.Isolate (изоляция),
// process.AdvancePresentation (решение на точке предъявления).
func React(s State, env Env, up Upstream) kernel.Output {
	_, _ = env, up
	var out kernel.Output
	if s.ItemID == "" {
		return out
	}
	subject := "item:" + s.ItemID
	for _, n := range s.NCs {
		if n.Origin != OriginSignal {
			continue
		}
		re := must(kernel.NewReaction(Module, catalog.DecisionNonconformityDrafted,
			kernel.Slot{RuleID: RuleDraft, Subject: subject, TriggerKey: n.ID}, n.Draft, stubs(n.Causes, n.FoundAt)...))
		re.AutomationMode = 1
		out.Reactions = append(out.Reactions, re)
	}
	for _, c := range s.Containment {
		if c.By != ByRule || c.Released || levelRank(c.Basis) < levelRank(c.Level) {
			continue
		}
		causes := c.Causes
		if len(causes) == 0 {
			causes = []string{c.Key}
		}
		if c.BasisEventID != "" && c.BasisEventID != c.Key {
			causes = append([]string{c.BasisEventID}, causes...)
		}
		data := ContainmentAppliedData{Level: c.Level, Basis: sortedIDs(causes)}
		if id, ok := strings.CutPrefix(c.Key, "incident:"); ok {
			data.IncidentID = id
			data.Basis = sortedIDs([]string{c.BasisEventID})
		}
		mode := 1
		if c.Rule == RuleIncidentScope || c.Rule == RuleCleanPoint {
			mode = 2
		}
		re := must(kernel.NewReaction(Module, catalog.DecisionContainmentApplied,
			kernel.Slot{RuleID: c.Rule, Subject: subject, TriggerKey: c.Key}, data, stubs(data.Basis, c.At)...))
		re.AutomationMode = mode
		out.Reactions = append(out.Reactions, re)
	}
	for _, e := range s.Effects {
		cause := kernel.Record{EventID: e.Cause, OccurredAt: e.CauseAt}
		switch e.Kind {
		case "set_quality":
			out.Intents = append(out.Intents, quality.SetQuality(Module, statuses.Quality(e.Value), cause))
		case "isolate":
			out.Intents = append(out.Intents, process.Isolate(Module, e.Value, cause))
		case "advance_presentation":
			out.Intents = append(out.Intents, process.AdvancePresentation(Module, process.Presentation{StepKey: e.StepKey, Resolution: e.Value}, cause))
		}
	}
	return out
}

// stubs — записи-причины по id и времени: реакции нужны только event_id и
// наибольший occurred_at причин (AD-3, AD-37).
func stubs(ids []string, at time.Time) []kernel.Record {
	out := make([]kernel.Record, 0, len(ids))
	for _, id := range ids {
		out = append(out, kernel.Record{EventID: id, OccurredAt: at})
	}
	return out
}

func sortedIDs(ids []string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id != "" {
			out = append(out, id)
		}
	}
	return compactSorted(out)
}

func must(r kernel.Reaction, err error) kernel.Reaction {
	if err != nil {
		// Тип и эмитент постоянны: ошибка — рассинхронизация с каталогом.
		panic(err)
	}
	return r
}
