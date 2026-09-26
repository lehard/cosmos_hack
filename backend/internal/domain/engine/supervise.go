package engine

import (
	"encoding/json"
	"slices"
	"strings"

	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
	"ant/internal/domain/notifications"
)

// decisionsBeforeNewData — правило движка AD-5 / FR-32: решение человека не
// отменяется поздним событием; если после основания решения (basis_seq, а без
// него — seq самого решения) стала известна запись, возникшая не позже
// решения, движок ставит автору задачу «решение принято до новых данных —
// пересмотрите». Причины — решение и все такие записи (отсортированы), поэтому
// новая поздняя запись даёт следующую версию задачи. Эмитент типа задачи —
// notifications (AD-40): запись строит его функция notifications.ReviewTask.
func decisionsBeforeNewData(in []kernel.Record) []kernel.Reaction {
	var out []kernel.Reaction
	// Д-81: пересмотр (decision.presentation.reviewed) закрывает задачу по
	// решению, если записан позже всех её причин; новая поздняя запись после
	// пересмотра возвращает задачу следующей версией. Сам пересмотр задачей
	// не помечается — её получает исходное решение.
	reviewed := map[string]int64{}
	for _, r := range in {
		if r.Type != catalog.DecisionPresentationReviewed {
			continue
		}
		var x struct {
			ReviewedEventID string `json:"reviewed_event_id"`
		}
		if json.Unmarshal(r.Data, &x) == nil && x.ReviewedEventID != "" {
			reviewed[x.ReviewedEventID] = max(reviewed[x.ReviewedEventID], r.Seq)
		}
	}
	for _, d := range in {
		if d.Kind != catalog.KindDecision || d.Type == catalog.DecisionPresentationReviewed {
			continue
		}
		known := d.BasisSeq
		if known == 0 {
			known = d.Seq
		}
		causes := []kernel.Record{d}
		for _, r := range in {
			if r.Kind == catalog.KindDecision || r.Seq <= known || r.OccurredAt.After(d.OccurredAt) || !reviewRelevant(r.Type) {
				continue
			}
			causes = append(causes, r)
		}
		if len(causes) == 1 {
			continue
		}
		if at, ok := reviewed[d.EventID]; ok && !slices.ContainsFunc(causes[1:], func(r kernel.Record) bool { return r.Seq > at }) {
			continue
		}
		subject := d.Stream
		if subject == "" {
			subject = "item:" + d.ItemID
		}
		slot := kernel.Slot{RuleID: RuleDecisionBeforeNewData, Subject: subject, TriggerKey: d.EventID}
		person := ""
		if d.Actor != "" && !strings.Contains(d.Actor, "@") {
			person = d.Actor
		}
		re, err := notifications.ReviewTask(slot, ev.TaskTaskCreatedV1KindReviewAfterNewData,
			"Решение принято до новых данных — пересмотрите", ReviewerRole, person, causes...)
		if err != nil {
			// Тип и эмитент постоянны: ошибка здесь — рассинхронизация с каталогом.
			panic(err)
		}
		out = append(out, re)
	}
	return out
}

// reviewRelevant — поздняя запись может поменять основание решения: связи
// генеалогии (genealogy.*) и носители (item.carrier.*) — учёт состава и
// идентификации, а не данные о качестве; пересмотра решения ОТК они не требуют.
func reviewRelevant(t catalog.Type) bool {
	s := string(t)
	return !strings.HasPrefix(s, "genealogy.") && !strings.HasPrefix(s, "item.carrier.")
}
