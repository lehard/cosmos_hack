package signing

import (
	"context"
	"strings"
	"uuid"

	"ant/internal/application/journal"
	"ant/internal/contracts/catalog"
	jc "ant/internal/contracts/journal"
	dom "ant/internal/domain/signing"
)

// JournalKeyAlerts — временный мост шины безопасности до эпика 29 (как мост
// приёма JournalSecurityBus): строит служебную запись security.key.alert
// (эмитент по каталогу — модуль security, AD-24, AD-40) для той же пачки
// Append — «повторный ключ», «чужой ключ», «журнал агента расходится с
// сервером». Эпик 29 заменяет мост своей реализацией порта Alerts.
type JournalKeyAlerts struct {
	Service *Service
}

// KeyAlert — запись тревоги по ключу.
func (b JournalKeyAlerts) KeyAlert(_ context.Context, a dom.KeyAlert) ([]journal.Pending, error) {
	s := b.Service
	info, _ := catalog.Lookup(catalog.SecurityKeyAlert)
	id := uuid.NewV7().String()
	now := s.now().Format(TimeLayout)
	data := map[string]any{"alert": a.Alert}
	if dom.ValidKeyRef(a.KeyRef) {
		data["key_ref"] = a.KeyRef
	}
	if a.Person != "" {
		data["person_id"] = a.Person
	}
	if a.Detail != "" {
		data["detail"] = a.Detail
	}
	ev := map[string]any{"event_id": id, "event_type": string(catalog.SecurityKeyAlert), "schema_version": info.CurrentVersion,
		"source_id": SourceAPI, "source_kind": "external_system", "occurred_at": now, "correlation_id": id, "causation_id": nil,
		"integrity": map[string]any{"format_version": 1, "crypto_profile": dom.ProfileGost, "signers": []string{"gateway-ingest@1"}},
		"data":      data}
	c, err := dom.CanonicalOf(ev)
	if err != nil {
		return nil, err
	}
	keyID := "unknown"
	if k, _, err := dom.SplitKeyRef(a.KeyRef); err == nil {
		keyID = k
	}
	e := jc.JournalEntry{Chain: jc.JournalEntryChainMain, EntryKind: jc.JournalEntryEntryKindService,
		EventType: string(catalog.SecurityKeyAlert), SchemaVersion: info.CurrentVersion, EventID: id, SourceID: SourceAPI,
		Stream: "key:" + strings.ToLower(keyID), Partition: s.cfg.Partitions, OccurredAt: now, ReceivedAt: now, CorrelationID: id,
		ProvenanceClass: jc.JournalEntryProvenanceClassServerAttested, DomainBuild: s.cfg.DomainBuild}
	return []journal.Pending{{Entry: e, Envelope: dom.Seal(dom.PayloadType(dom.ClassEvent, 1), c).Marshal()}}, nil
}
