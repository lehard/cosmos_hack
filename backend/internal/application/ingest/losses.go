package ingest

import (
	"context"
	"errors"
	"fmt"

	"ant/internal/application/journal"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	dom "ant/internal/domain/ingest"
	"ant/internal/domain/kernel"
)

// LossRule — правило реакции «потеря данных источника» (AD-7).
const LossRule = "ingest.source_loss_window"

// LossReport — заявленная потеря данных источника.
type LossReport struct {
	SourceID string
	From, To int64
	EventID  string
	Seq      int64
}

// CheckLosses — сигнал «потеря данных источника» (AD-7, FR-35, FR-39): разрывы
// source_seq, не закрытые досылкой за окно ожидания, дают реакцию
// ingest.source.loss_suspected; досылка в окне сигнал отменяет (разрыв закрыт
// раньше, чем истекло окно). Вызывает роль scheduler (эпик 24) по своему
// расписанию; id реакции — UUIDv5 от слота, поэтому повтор не дублирует запись.
func (s *Service) CheckLosses(ctx context.Context) ([]LossReport, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	now, err := s.deps.DomainClock.Now(ctx)
	if err != nil {
		return nil, err
	}
	srcs, err := s.deps.Registry.Sources(ctx)
	if err != nil {
		return nil, err
	}
	var out []LossReport
	for _, st := range srcs {
		unlock := s.lock(st.SourceID)
		cur, err := s.deps.Registry.SourceState(ctx, st.SourceID)
		if err != nil {
			unlock()
			return out, err
		}
		cur.SourceID = st.SourceID
		next, due := dom.DueLosses(cur, now, s.cfg.LossWindow)
		if len(due) == 0 {
			unlock()
			continue
		}
		heads, err := s.deps.Journal.Head(ctx)
		if err != nil {
			unlock()
			return out, err
		}
		var pend []journal.Pending
		var reps []LossReport
		for _, g := range due {
			slot := kernel.Slot{RuleID: LossRule, Subject: "source:" + st.SourceID, TriggerKey: fmt.Sprintf("%d-%d", g.From, g.To)}
			id := kernel.UUIDv5(constants.NsAnt, slot.Key()+"\x1f1")
			react := map[string]any{"rule_id": LossRule, "automation_mode": 2,
				"slot":    map[string]any{"rule_id": slot.RuleID, "subject": slot.Subject, "trigger_key": slot.TriggerKey},
				"version": 1, "supersedes": nil, "causes": []string{}, "basis_seq": max(heads.MainSeq, 1)}
			p, _, err := s.buildService(ctx, serviceRecord{Type: catalog.IngestSourceLossSuspected, EventID: id,
				Data:   map[string]any{"source_id": st.SourceID, "missing_from_seq": g.From, "missing_to_seq": g.To},
				Stream: "source:" + st.SourceID, Partition: s.cfg.StagePartition, OccurredAt: g.OpenedAt, ReceivedAt: now,
				Extra: map[string]any{"reaction": react}})
			if err != nil {
				unlock()
				return out, err
			}
			p.Entry.RuleID = &slot.RuleID
			pend = append(pend, p)
			reps = append(reps, LossReport{SourceID: st.SourceID, From: g.From, To: g.To, EventID: id})
		}
		ar, err := s.deps.Journal.Append(ctx, journal.AppendRequest{Batch: pend,
			Project: func(ctx context.Context, _ journal.AppendResult) error {
				return s.deps.Registry.SaveSourceState(ctx, next)
			}})
		if errors.Is(err, journal.ErrDuplicate) {
			// Сигнал по этому разрыву уже записан (id реакции — от слота): отметить разрыв.
			err = s.deps.Registry.SaveSourceState(ctx, next)
			reps = nil
		}
		unlock()
		if err != nil {
			return out, err
		}
		for i := range reps {
			if i < len(ar.Seqs) {
				reps[i].Seq = ar.Seqs[i]
			}
		}
		out = append(out, reps...)
	}
	return out, nil
}
