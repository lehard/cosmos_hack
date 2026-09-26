package ingest

import (
	"context"
	"strings"

	"ant/internal/application/journal"
	"ant/internal/contracts/catalog"
	jc "ant/internal/contracts/journal"
)

// PendingCAPlaceholder — заполнитель main_commit в записи критического
// действия: настоящее обязательство основной записи и номер CA-‹n› ставит
// journal.Append в той же транзакции (AD-28, AD-44).
var PendingCAPlaceholder = "streebog256:" + strings.Repeat("0", 64)

// JournalSecurityBus — временный мост шины безопасности до эпика 29: строит
// записи security.signature.invalid и security.idempotency.conflict (эмитент
// по каталогу — модуль security, AD-40) и запись цепочки критических действий
// для конфликта (критичный тип, группа admin_security). Эпик 29 заменяет мост
// своей реализацией порта SecurityBus (domain/security.BuildCA) без изменения
// конвейера приёма.
type JournalSecurityBus struct {
	Gateway *Service
}

// SignatureInvalid — AD-2: ошибка подписи дополнительно даёт событие security.
func (b JournalSecurityBus) SignatureInvalid(ctx context.Context, f SignatureFailure) ([]journal.Pending, []journal.Pending, error) {
	s := b.Gateway
	data := map[string]any{"failure": f.Failure}
	if sourceIDRe(f.SourceID) {
		data["source_id"] = f.SourceID
	}
	if f.KeyRef != "" && keyRefRe(f.KeyRef) {
		data["key_ref"] = f.KeyRef
	}
	if isUUID(f.EventID) {
		data["subject_event_id"] = f.EventID
	}
	if f.QuarantineID != "" {
		data["quarantine_event_id"] = f.QuarantineID
	}
	p, _, err := s.buildService(ctx, serviceRecord{Type: catalog.SecuritySignatureInvalid, Data: data, Stream: "global",
		Partition: s.cfg.StagePartition, OccurredAt: f.OccurredAt, ReceivedAt: f.OccurredAt, CausationID: f.QuarantineID})
	if err != nil {
		return nil, nil, err
	}
	return []journal.Pending{p}, nil, nil
}

// IdempotencyConflict — FR-31, AD-7: конфликт целостности — событие security и
// запись в журнал критических действий.
func (b JournalSecurityBus) IdempotencyConflict(ctx context.Context, c IdempotencyConflict) ([]journal.Pending, []journal.Pending, error) {
	s := b.Gateway
	stream := "source:" + c.SourceID
	main, mainID, err := s.buildService(ctx, serviceRecord{Type: catalog.SecurityIdempotencyConflict, Data: map[string]any{
		"source_id": c.SourceID, "event_id": c.EventID, "first_payload_digest": c.FirstDigest, "conflicting_payload_digest": c.ConflictDigest,
	}, Stream: stream, Partition: s.cfg.StagePartition, OccurredAt: c.OccurredAt, ReceivedAt: c.OccurredAt, CausationID: c.QuarantineID})
	if err != nil {
		return nil, nil, err
	}
	basis := []string{mainID}
	if c.QuarantineID != "" {
		basis = append(basis, c.QuarantineID)
	}
	ca, _, err := s.buildService(ctx, serviceRecord{Type: catalog.SecurityCriticalActionRecorded, Data: map[string]any{
		"ca_no": 1, "ca_group": "admin_security", "action_type": string(catalog.SecurityIdempotencyConflict),
		"main_event_id": mainID, "main_commit": PendingCAPlaceholder, "object_ref": stream,
		"before": "принято: " + c.FirstDigest, "after": "пришло другое содержимое: " + c.ConflictDigest,
		"basis_event_ids": basis,
	}, Stream: stream, Partition: s.cfg.StagePartition, OccurredAt: c.OccurredAt, ReceivedAt: c.OccurredAt, CausationID: mainID})
	if err != nil {
		return nil, nil, err
	}
	ca.Entry.Chain = jc.JournalEntryChainCa
	return []journal.Pending{main}, []journal.Pending{ca}, nil
}

func keyRefRe(s string) bool {
	id, ver, ok := strings.Cut(s, "@")
	return ok && id != "" && ver != "" && ver[0] != '0' && strings.Trim(ver, "0123456789") == "" && strings.ToLower(id) == id
}
