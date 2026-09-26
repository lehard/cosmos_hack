package ingest

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
	"uuid"

	"ant/internal/application/journal"
	"ant/internal/contracts/catalog"
	jc "ant/internal/contracts/journal"
	dom "ant/internal/domain/ingest"
)

// PayloadTypeEvent — payloadType пакета события (contracts/crypto/payload-classes.yaml).
const PayloadTypeEvent = "application/vnd.ant.event+json; v=1"

// ts — время в формате контракта: RFC 3339 UTC, ровно три знака после секунд.
func ts(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000Z") }

// parseTS — разбор времени контракта.
func parseTS(s string) (time.Time, error) { return time.Parse("2006-01-02T15:04:05.000Z", s) }

func strp(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func intp(n int64) *int {
	if n == 0 {
		return nil
	}
	v := int(n)
	return &v
}

// dsse — конверт DSSE без подписей вокруг канонического события: так хранится
// событие, принятое без подписи в профиле demo, и служебная запись, пока нет
// ключа шлюза (эпик 05). Подписанное сообщение хранится в исходных байтах (AD-20).
func dsse(canon []byte) []byte {
	b, _ := json.Marshal(struct {
		PayloadType string   `json:"payloadType"`
		Payload     string   `json:"payload"`
		Signatures  []string `json:"signatures"`
	}{PayloadTypeEvent, base64.StdEncoding.EncodeToString(canon), []string{}})
	return b
}

// entryMeta — открытые поля записи журнала, общие для факта и служебных записей.
type entryMeta struct {
	Type          catalog.Type
	Version       int
	EventID       string
	SourceID      string
	SourceSeq     int64
	RunID         string
	ItemID        string
	CarrierRef    string
	Stream        string
	Partition     int
	OccurredAt    time.Time
	ReceivedAt    time.Time
	CorrelationID string
	CausationID   string
	Provenance    string
}

// entry строит запись для Append: открытые поля (seq, committed_at, commit,
// link и sealed ставит Append, AD-44) и конверт.
func (s *Service) entry(m entryMeta, envelope []byte) journal.Pending {
	info, _ := catalog.Lookup(m.Type)
	e := jc.JournalEntry{
		Chain:           jc.JournalEntryChainMain,
		EntryKind:       jc.JournalEntryEntryKind(info.Kind),
		EventType:       string(m.Type),
		SchemaVersion:   m.Version,
		EventID:         m.EventID,
		SourceID:        m.SourceID,
		SourceSeq:       intp(m.SourceSeq),
		RunID:           strp(m.RunID),
		ItemID:          strp(m.ItemID),
		CarrierRef:      strp(m.CarrierRef),
		Stream:          m.Stream,
		Partition:       m.Partition,
		OccurredAt:      ts(m.OccurredAt),
		ReceivedAt:      ts(m.ReceivedAt),
		RecordedAt:      ts(m.ReceivedAt),
		CorrelationID:   m.CorrelationID,
		CausationID:     jc.JournalEntryCausationID(strp(m.CausationID)),
		ProvenanceClass: jc.JournalEntryProvenanceClass(m.Provenance),
		DomainBuild:     s.cfg.DomainBuild,
	}
	return journal.Pending{Entry: e, Envelope: envelope}
}

// serviceRecord — служебная запись приёма (карантин, флаг, импорт, потеря):
// конверт события от шлюза приёма (класс server_attested, AD-2), подписанный
// ключом шлюза, если он подключён.
type serviceRecord struct {
	Type          catalog.Type
	Data          any
	Stream        string
	Partition     int
	ItemID        string
	RunID         string
	OccurredAt    time.Time
	ReceivedAt    time.Time
	CorrelationID string
	CausationID   string
	// EventID — задан для детерминированных записей (UUIDv5); пусто — UUIDv7.
	EventID string
	// Extra — дополнительные поля конверта (command для решений, reaction для реакций).
	Extra map[string]any
	// Provenance — класс происхождения; пусто — server_attested.
	Provenance string
}

// buildService строит служебную запись и её конверт.
func (s *Service) buildService(ctx context.Context, r serviceRecord) (journal.Pending, string, error) {
	id := r.EventID
	if id == "" {
		id = uuid.NewV7().String()
	}
	corr := r.CorrelationID
	if corr == "" {
		corr = id
	}
	env := map[string]any{
		"event_id":       id,
		"event_type":     string(r.Type),
		"schema_version": 1,
		"source_id":      s.cfg.GatewaySource,
		"occurred_at":    ts(r.OccurredAt),
		"correlation_id": corr,
		"causation_id":   nil,
		"integrity":      map[string]any{"format_version": 1, "crypto_profile": "gost", "signers": []string{s.cfg.GatewayKey}},
		"data":           r.Data,
	}
	if r.CausationID != "" {
		env["causation_id"] = r.CausationID
	}
	if r.RunID != "" {
		env["run_id"] = r.RunID
	}
	if r.ItemID != "" {
		env["item_id"] = r.ItemID
	}
	for k, v := range r.Extra {
		env[k] = v
	}
	raw, err := json.Marshal(env)
	if err != nil {
		return journal.Pending{}, "", err
	}
	canon, err := dom.Canonicalize(raw)
	if err != nil {
		return journal.Pending{}, "", fmt.Errorf("служебная запись %s: %w", r.Type, err)
	}
	envelope := dsse(canon)
	if s.deps.ServerSigner != nil {
		if envelope, err = s.deps.ServerSigner.Sign(ctx, PayloadTypeEvent, canon, s.cfg.GatewayKey); err != nil {
			return journal.Pending{}, "", err
		}
	}
	prov := r.Provenance
	if prov == "" {
		prov = "server_attested"
	}
	p := s.entry(entryMeta{
		Type: r.Type, Version: 1, EventID: id, SourceID: s.cfg.GatewaySource, RunID: r.RunID, ItemID: r.ItemID,
		Stream: r.Stream, Partition: r.Partition, OccurredAt: r.OccurredAt, ReceivedAt: r.ReceivedAt,
		CorrelationID: corr, CausationID: r.CausationID, Provenance: prov,
	}, envelope)
	return p, id, nil
}
