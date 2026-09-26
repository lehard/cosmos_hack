package engine

import (
	"strings"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
)

// decisionsBeforeNewData — правило движка AD-5 / FR-32: решение человека не
// отменяется поздним событием; если после основания решения (basis_seq, а без
// него — seq самого решения) стала известна запись, возникшая не позже
// решения, движок ставит автору задачу «решение принято до новых данных —
// пересмотрите». Причины — решение и все такие записи (отсортированы), поэтому
// новая поздняя запись даёт следующую версию задачи. Эмитент типа задачи —
// notifications (AD-40).
func decisionsBeforeNewData(in []kernel.Record) []kernel.Reaction {
	var out []kernel.Reaction
	for _, d := range in {
		if d.Kind != catalog.KindDecision {
			continue
		}
		known := d.BasisSeq
		if known == 0 {
			known = d.Seq
		}
		causes := []kernel.Record{d}
		for _, r := range in {
			if r.Kind == catalog.KindDecision || r.Seq <= known || r.OccurredAt.After(d.OccurredAt) {
				continue
			}
			causes = append(causes, r)
		}
		if len(causes) == 1 {
			continue
		}
		subject := d.Stream
		if subject == "" {
			subject = "item:" + d.ItemID
		}
		slot := kernel.Slot{RuleID: RuleDecisionBeforeNewData, Subject: subject, TriggerKey: d.EventID}
		data := ev.TaskTaskCreatedV1{
			AssigneeRoleID: ReviewerRole,
			Kind:           ev.TaskTaskCreatedV1KindReviewAfterNewData,
			SubjectRef:     ev.StreamRef(subject),
			TaskID:         ev.ObjectID(kernel.UUIDv5(constants.NsAnt, "task\x1f"+slot.Key())),
			Title:          "Решение принято до новых данных — пересмотрите",
		}
		if d.Actor != "" && !strings.Contains(d.Actor, "@") {
			p := ev.PersonRef(d.Actor)
			data.AssigneePersonID = &p
		}
		re, err := kernel.NewReaction("notifications", catalog.TaskTaskCreated, slot, data, causes...)
		if err != nil {
			// Тип и эмитент постоянны: ошибка здесь — рассинхронизация с каталогом.
			panic(err)
		}
		re.AutomationMode = 1
		out = append(out, re)
	}
	return out
}
