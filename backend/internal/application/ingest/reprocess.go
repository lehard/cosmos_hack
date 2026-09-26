package ingest

import (
	"context"
	"errors"
	"uuid"

	"ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
)

// replay — прежний ответ на повтор команды с тем же command_id (AD-7).
func (s *Service) replay(id string) (any, bool) {
	s.cmdMu.Lock()
	defer s.cmdMu.Unlock()
	v, ok := s.cmds[id]
	return v, ok
}

func (s *Service) remember(id string, v any) {
	if id == "" {
		return
	}
	s.cmdMu.Lock()
	s.cmds[id] = v
	s.cmdMu.Unlock()
}

// commandBlock — блок command конверта записи-решения (AD-39).
func commandBlock(m platform.CommandMeta, stream string) map[string]any {
	basis := max(m.BasisSeq, 1)
	policy := max(m.PolicySeq, 1)
	b := map[string]any{"command_id": m.CommandID, "basis_seq": basis, "guard_streams": []string{stream},
		"policy_seq": policy, "signature_level": 0}
	if m.WorkplaceID != "" {
		b["workplace_id"] = m.WorkplaceID
	}
	return b
}

// ReprocessCommand — FR-30: администратор повторно обрабатывает сообщение карантина
// (например, после появления схемы и повышателя новой версии) или закрывает
// его. Итог — решение ingest.message.reprocessed со ссылкой на запись карантина
// и, если принято, на принятую запись.
func (s *Service) ReprocessCommand(ctx context.Context, cmd Cmd[ReprocessInput]) (ReprocessResult, error) {
	if err := s.ready(); err != nil {
		return ReprocessResult{}, platform.NotImplemented("ingest.quarantine.reprocess")
	}
	if cmd.Meta.CommandID == "" {
		cmd.Meta.CommandID = uuid.NewV7().String()
	}
	if v, ok := s.replay(cmd.Meta.CommandID); ok {
		r := v.(ReprocessResult)
		r.Receipt.Replayed = true
		return r, nil
	}
	rec, err := s.deps.Quarantine.Get(ctx, cmd.Body.QuarantineID)
	if errors.Is(err, ErrNotFound) {
		return ReprocessResult{}, platform.Fail(errcodes.ApiNotFound, "object", "Запись карантина", "id", cmd.Body.QuarantineID)
	}
	if err != nil {
		return ReprocessResult{}, err
	}
	if !rec.Status.Unresolved() {
		e := platform.Fail(errcodes.ApiValidationFailed, "field", "quarantine_id", "reason", "запись уже обработана")
		e.Detail = "Запись карантина уже обработана: " + string(rec.Status)
		return ReprocessResult{}, e
	}
	out := ReprocessResult{Outcome: "discarded"}
	status := QuarantineDiscarded
	if !cmd.Body.Discard {
		raw, err := s.content(ctx, rec)
		if err != nil {
			return ReprocessResult{}, err
		}
		r, err := s.process(ctx, raw, msgCtx{})
		if err != nil {
			return ReprocessResult{}, err
		}
		out.Result = r
		switch r.Outcome {
		case OutcomeAccepted, OutcomeAcceptedWithFlag, OutcomeDuplicate:
			out.Outcome, status = "accepted", QuarantineAccepted
		default:
			out.Outcome, status = "still_invalid", QuarantineStillInvalid
		}
	}
	now, err := s.deps.DomainClock.Now(ctx)
	if err != nil {
		return ReprocessResult{}, err
	}
	data := map[string]any{"quarantine_event_id": rec.ID, "outcome": out.Outcome}
	if out.Result.EventID != "" && out.Outcome == "accepted" && isUUID(out.Result.EventID) {
		data["accepted_event_id"] = out.Result.EventID
	}
	if cmd.Meta.Reason != "" {
		data["reason"] = map[string]any{"code": "admin", "text": truncate(cmd.Meta.Reason, 1000)}
	}
	stream := "source:" + rec.SourceID
	extra := map[string]any{"command": commandBlock(cmd.Meta, stream)}
	pend, id, err := s.buildService(ctx, serviceRecord{Type: catalog.IngestMessageReprocessed, Data: data, Stream: stream,
		Partition: s.cfg.StagePartition, OccurredAt: now, ReceivedAt: now, CausationID: rec.ID, EventID: cmd.Meta.CommandID,
		Extra: extra, Provenance: "personal"})
	if err != nil {
		return ReprocessResult{}, err
	}
	ar, err := s.deps.Journal.Append(ctx, journal.AppendRequest{Batch: []journal.Pending{pend},
		Project: func(ctx context.Context, _ journal.AppendResult) error {
			return s.deps.Quarantine.Resolve(ctx, rec.ID, status, id)
		}})
	if err != nil {
		return ReprocessResult{}, err
	}
	out.Receipt = platform.Receipt{CommandID: cmd.Meta.CommandID, EventIDs: []string{id}, RecordedAt: now}
	if len(ar.Seqs) > 0 {
		out.Receipt.Seq = ar.Seqs[0]
	}
	s.remember(cmd.Meta.CommandID, out)
	return out, nil
}
