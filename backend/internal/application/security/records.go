package security

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	appjournal "ant/internal/application/journal"
	"ant/internal/application/signing"
	"ant/internal/contracts/catalog"
	jc "ant/internal/contracts/journal"
	dj "ant/internal/domain/journal"
)

// SourceSecurity — source_id записей модуля security (шина безопасности,
// журнал критических действий).
const SourceSecurity = "ant-security"

// KeyRef — ключ модуля security key_id@версия (класс server-attested, AD-2).
// TODO(05): ключ из тома ключей и реестра доверия; до эпика 05 конверт без подписи.
const KeyRef = "security@1"

// PayloadTypeEvent — payloadType пакета события (AD-10).
const PayloadTypeEvent = "application/vnd.ant.event+json; v=1"

// Encoder — сборка записей модуля security для journal.Append (AD-44):
// эмитент по каталогу — security (AD-40), класс происхождения
// server_attested; конверт DSSE подписывает Signer ключом KeyRef (nil —
// конверт без подписи, демо до эпика 05).
type Encoder struct {
	Signer      signing.Signer
	DomainBuild string
	// StagePartition — партиция записей вне изделия (за пределами 0…P-1, как у приёма).
	StagePartition int
	// Now — InfraClock для received_at (AD-37); nil — time.Now.
	Now func() time.Time
}

// Out — запись модуля security.
type Out struct {
	EventID     string
	Type        catalog.Type
	Stream      string
	Partition   *int
	RunID       string
	OccurredAt  time.Time
	Correlation string
	Causation   string
	Data        any
}

// Encode — запись для journal.Append: открытые поля (seq, звено и время
// фиксации ставит Append) и конверт DSSE.
func (c Encoder) Encode(ctx context.Context, o Out) (appjournal.Pending, error) {
	info, ok := catalog.Lookup(o.Type)
	if !ok || info.Emitter != "security" {
		return appjournal.Pending{}, fmt.Errorf("security: тип %q — не тип модуля security", o.Type)
	}
	data, err := json.Marshal(o.Data)
	if err != nil {
		return appjournal.Pending{}, err
	}
	corr := o.Correlation
	if corr == "" {
		corr = o.EventID
	}
	env := map[string]any{
		"event_id": o.EventID, "event_type": string(o.Type), "schema_version": info.CurrentVersion, "source_id": SourceSecurity,
		"occurred_at": dj.FormatTime(o.OccurredAt), "correlation_id": corr, "causation_id": nil,
		"integrity": map[string]any{"format_version": 1, "crypto_profile": "gost", "signers": []string{KeyRef}},
		"data":      json.RawMessage(data),
	}
	var causation *string
	if o.Causation != "" {
		env["causation_id"] = o.Causation
		causation = &o.Causation
	}
	if o.RunID != "" {
		env["run_id"] = o.RunID
	}
	raw, err := json.Marshal(env)
	if err != nil {
		return appjournal.Pending{}, err
	}
	payload, err := dj.Canonical(raw)
	if err != nil {
		return appjournal.Pending{}, err
	}
	var sealed []byte
	if c.Signer != nil {
		if sealed, err = c.Signer.Sign(ctx, PayloadTypeEvent, payload, KeyRef); err != nil {
			return appjournal.Pending{}, fmt.Errorf("подпись %s: %w", o.Type, err)
		}
	} else {
		sealed, _ = json.Marshal(map[string]any{"payloadType": PayloadTypeEvent,
			"payload": base64.StdEncoding.EncodeToString(payload), "signatures": []any{}})
	}
	now := time.Now
	if c.Now != nil {
		now = c.Now
	}
	e := jc.JournalEntry{
		Chain: jc.JournalEntryChainMain, EntryKind: jc.JournalEntryEntryKind(info.Kind), EventType: string(o.Type),
		SchemaVersion: info.CurrentVersion, EventID: o.EventID, SourceID: SourceSecurity, Stream: o.Stream,
		Partition: c.StagePartition, OccurredAt: dj.FormatTime(o.OccurredAt), ReceivedAt: dj.FormatTime(now()),
		CorrelationID: corr, CausationID: causation, ProvenanceClass: jc.JournalEntryProvenanceClassServerAttested,
		DomainBuild: c.DomainBuild,
	}
	if o.Partition != nil {
		e.Partition = *o.Partition
	}
	if o.RunID != "" {
		run := o.RunID
		e.RunID = &run
	}
	return appjournal.Pending{Entry: e, Envelope: sealed}, nil
}

// Event — конверт события после DSSE (events/common/envelope.v1.json): поля,
// которые читает модуль security.
type Event struct {
	EventID     string          `json:"event_id"`
	EventType   string          `json:"event_type"`
	SourceID    string          `json:"source_id"`
	OccurredAt  string          `json:"occurred_at"`
	CausationID *string         `json:"causation_id"`
	RunID       string          `json:"run_id"`
	ItemID      string          `json:"item_id"`
	Data        json.RawMessage `json:"data"`
	Corrects    *struct {
		EventID string `json:"event_id"`
	} `json:"corrects"`
	Command *struct {
		OnBehalfOf string `json:"on_behalf_of"`
	} `json:"command"`
	Integrity struct {
		Signers []string `json:"signers"`
	} `json:"integrity"`
}

// DSSE — конверт DSSE записи.
type DSSE struct {
	PayloadType string `json:"payloadType"`
	Payload     string `json:"payload"`
	Signatures  []struct {
		KeyID string `json:"keyid"`
		Sig   string `json:"sig"`
	} `json:"signatures"`
}

// ParseEnvelope — событие из канонического конверта DSSE записи.
func ParseEnvelope(envelope []byte) (Event, DSSE, error) {
	var d DSSE
	if err := json.Unmarshal(envelope, &d); err != nil {
		return Event{}, d, fmt.Errorf("DSSE: %w", err)
	}
	payload, err := base64.StdEncoding.DecodeString(d.Payload)
	if err != nil {
		return Event{}, d, fmt.Errorf("payload: %w", err)
	}
	var ev Event
	if err := json.Unmarshal(payload, &ev); err != nil {
		return Event{}, d, fmt.Errorf("конверт события: %w", err)
	}
	return ev, d, nil
}

// open — событие записи журнала (расшифрованный конверт, commit сверен).
func open(ctx context.Context, store appjournal.JournalStore, e jc.JournalEntry) (Event, error) {
	env, err := store.Open(ctx, e)
	if err != nil {
		return Event{}, fmt.Errorf("запись %s/%d: %w", e.Chain, e.Seq, err)
	}
	ev, _, err := ParseEnvelope(env.Raw)
	return ev, err
}
