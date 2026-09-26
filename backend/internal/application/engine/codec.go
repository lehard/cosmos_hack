package engine

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	appjournal "ant/internal/application/journal"
	"ant/internal/contracts/catalog"
	jc "ant/internal/contracts/journal"
	"ant/internal/contracts/upcast"
	"ant/internal/domain/engine"
	"ant/internal/domain/kernel"
)

// Константы записей, которые пишут воркер и стадия.
const (
	// PayloadTypeEvent — payloadType конверта события (contracts/crypto/payload-classes.yaml).
	PayloadTypeEvent = "application/vnd.ant.event+json; v=1"
	// SourceEngine — source_id записей движка (реакции, служебные записи воркера).
	SourceEngine = "ant-engine"
	// TimeLayout — время в конверте и открытых полях: RFC 3339 UTC, ровно
	// три знака после секунд («Соглашения/Время»).
	TimeLayout = "2006-01-02T15:04:05.000Z"
)

// Decoded — запись журнала в представлении домена и метаданные реакции.
type Decoded struct {
	Entry  jc.JournalEntry
	Record kernel.Record
	// Reaction — метаданные реакции из конверта; nil у не-реакций.
	Reaction *ReactionMeta
	// Info — строка каталога типа (Role — роль-эмитент, AD-40).
	Info catalog.Info
}

// ReactionMeta — метаданные реакции конверта (contracts/events/common/envelope.v1.json).
type ReactionMeta struct {
	RuleID         string   `json:"rule_id"`
	RuleRev        string   `json:"rule_rev,omitempty"`
	AutomationMode int      `json:"automation_mode"`
	Slot           SlotMeta `json:"slot"`
	Version        int      `json:"version"`
	Supersedes     *string  `json:"supersedes"`
	RevisedDueTo   *string  `json:"revised_due_to,omitempty"`
	Causes         []string `json:"causes"`
	BasisSeq       int64    `json:"basis_seq"`
}

// SlotMeta — слот реакции в конверте (AD-3).
type SlotMeta struct {
	RuleID     string `json:"rule_id"`
	Subject    string `json:"subject"`
	TriggerKey string `json:"trigger_key"`
}

// envelopeJSON — конверт события v1 (кейс §4.4, FR-27) в той части, которую
// пишет и читает движок; схема открыта (FR-29), лишние поля сохраняются в
// исходных байтах журнала.
type envelopeJSON struct {
	EventID       string          `json:"event_id"`
	EventType     string          `json:"event_type"`
	SchemaVersion int             `json:"schema_version"`
	SourceID      string          `json:"source_id"`
	SourceKind    string          `json:"source_kind,omitempty"`
	OccurredAt    string          `json:"occurred_at"`
	CorrelationID string          `json:"correlation_id"`
	CausationID   *string         `json:"causation_id"`
	RunID         string          `json:"run_id,omitempty"`
	ItemID        string          `json:"item_id,omitempty"`
	Corrects      *correctsJSON   `json:"corrects,omitempty"`
	Reaction      *ReactionMeta   `json:"reaction,omitempty"`
	Command       *commandJSON    `json:"command,omitempty"`
	Integrity     integrityJSON   `json:"integrity"`
	Data          json.RawMessage `json:"data"`
}

type correctsJSON struct {
	EventID string          `json:"event_id"`
	Reason  json.RawMessage `json:"reason,omitempty"`
}

type commandJSON struct {
	CommandID  string `json:"command_id,omitempty"`
	BasisSeq   int64  `json:"basis_seq"`
	OnBehalfOf string `json:"on_behalf_of,omitempty"`
}

type integrityJSON struct {
	FormatVersion int      `json:"format_version"`
	CryptoProfile string   `json:"crypto_profile"`
	Signers       []string `json:"signers"`
}

type dsseJSON struct {
	Payload     string `json:"payload"`
	PayloadType string `json:"payloadType"`
}

// Codec — чтение записей журнала в представление домена и сборка записей
// движка (AD-44, AD-20): расшифрованный конверт (JournalStore.Open) → DSSE →
// конверт события → kernel.Record с data, повышенным до текущей версии схемы.
type Codec struct {
	Store appjournal.JournalStore
	// Sealer — подпись конверта записей движка ключом «движок» (AD-3).
	Sealer Sealer
	// KeyRef, Profile — ключ движка key_id@версия и криптопрофиль (AD-10).
	KeyRef  string
	Profile string
	// DomainBuild — хеш доменного пакета, свернувшего запись (AD-9).
	DomainBuild string
	// Partitions — число партиций P (AD-6).
	Partitions int
}

// Decode — запись журнала в представлении домена.
func (c *Codec) Decode(ctx context.Context, e jc.JournalEntry) (Decoded, error) {
	d := Decoded{Entry: e}
	info, ok := catalog.Lookup(catalog.Type(e.EventType))
	if !ok {
		return d, fmt.Errorf("запись seq %d: тип %q не из каталога", e.Seq, e.EventType)
	}
	d.Info = info
	env, err := c.Store.Open(ctx, e)
	if err != nil {
		return d, fmt.Errorf("запись seq %d: конверт: %w", e.Seq, err)
	}
	var ds dsseJSON
	if err := json.Unmarshal(env.Raw, &ds); err != nil {
		return d, fmt.Errorf("запись seq %d: DSSE: %w", e.Seq, err)
	}
	payload, err := base64.StdEncoding.DecodeString(ds.Payload)
	if err != nil {
		return d, fmt.Errorf("запись seq %d: payload: %w", e.Seq, err)
	}
	var ej envelopeJSON
	if err := json.Unmarshal(payload, &ej); err != nil {
		return d, fmt.Errorf("запись seq %d: конверт события: %w", e.Seq, err)
	}
	data, err := upcastData(e.EventType, ej.SchemaVersion, info.CurrentVersion, ej.Data)
	if err != nil {
		return d, fmt.Errorf("запись seq %d: %w", e.Seq, err)
	}
	r := kernel.Record{
		Seq: int64(e.Seq), EventID: e.EventID, Type: info.Type, SchemaVersion: info.CurrentVersion,
		Kind: catalog.Kind(e.EntryKind), SourceID: e.SourceID, SourceKind: ej.SourceKind,
		Provenance: string(e.ProvenanceClass), Stream: e.Stream, CorrelationID: e.CorrelationID, Data: data,
	}
	if e.ItemID != nil {
		r.ItemID = *e.ItemID
	}
	if e.RunID != nil {
		r.RunID = *e.RunID
	}
	if e.CausationID != nil {
		r.CausationID = *e.CausationID
	}
	if e.BasisSeq != nil {
		r.BasisSeq = int64(*e.BasisSeq)
	}
	if ej.Corrects != nil {
		r.Corrects = ej.Corrects.EventID
	}
	if r.Kind == catalog.KindDecision && len(ej.Integrity.Signers) > 0 {
		r.Actor = ej.Integrity.Signers[0]
		if ej.Command != nil && ej.Command.OnBehalfOf != "" {
			r.Actor = ej.Command.OnBehalfOf
		}
	}
	if r.OccurredAt, err = parseTime(e.OccurredAt); err != nil {
		return d, fmt.Errorf("запись seq %d: occurred_at: %w", e.Seq, err)
	}
	r.ReceivedAt, _ = parseTime(e.ReceivedAt)
	r.RecordedAt, _ = parseTime(e.RecordedAt)
	d.Record = r
	d.Reaction = ej.Reaction
	return d, nil
}

// Recorded — записанная версия слота реакции для сравнения (AD-3).
func (d Decoded) Recorded() (engine.Recorded, bool) {
	if d.Reaction == nil {
		return engine.Recorded{}, false
	}
	m := d.Reaction
	return engine.Recorded{
		EventID: d.Record.EventID, Seq: d.Record.Seq, Module: kernel.Module(d.Info.Emitter), Type: d.Info.Type,
		Slot:    kernel.Slot{RuleID: m.Slot.RuleID, Subject: m.Slot.Subject, TriggerKey: m.Slot.TriggerKey},
		Version: m.Version, RuleRev: m.RuleRev, AutomationMode: m.AutomationMode, Causes: m.Causes,
		OccurredAt: d.Record.OccurredAt, Data: d.Record.Data,
	}, true
}

// upcastData повышает data до текущей версии схемы (AD-20) и возвращает
// канонический JSON.
func upcastData(eventType string, from, to int, raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 {
		raw = json.RawMessage("{}")
	}
	if from != 0 && from < to {
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.UseNumber()
		var m map[string]any
		if err := dec.Decode(&m); err != nil {
			return nil, fmt.Errorf("data: %w", err)
		}
		up, err := upcast.Upcast(eventType, from, to, m)
		if err != nil {
			return nil, fmt.Errorf("повышение %s v%d→v%d: %w", eventType, from, to, err)
		}
		b, err := json.Marshal(up)
		if err != nil {
			return nil, err
		}
		raw = b
	}
	return engine.Canonical(raw)
}

func parseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	return t.UTC(), err
}

// FormatTime — время записи по соглашению (три знака после секунд, UTC).
func FormatTime(t time.Time) string { return t.UTC().Format(TimeLayout) }

// Out — запись, которую движок (воркер, стадия, проектор) готовит к Append.
type Out struct {
	EventID     string
	Type        catalog.Type
	Kind        catalog.Kind
	Stream      string
	ItemID      string
	RunID       string
	OccurredAt  time.Time
	Correlation string
	Causation   string
	BasisSeq    int64
	Reaction    *ReactionMeta
	NormRev     string
	Data        any
}

// Encode собирает запись для journal.Append: открытые поля (seq, звено и
// время фиксации ставит Append) и подписанный конверт DSSE (AD-10, AD-44).
func (c *Codec) Encode(ctx context.Context, o Out) (appjournal.Pending, error) {
	info, ok := catalog.Lookup(o.Type)
	if !ok {
		return appjournal.Pending{}, fmt.Errorf("тип %q не из каталога", o.Type)
	}
	data, err := json.Marshal(o.Data)
	if err != nil {
		return appjournal.Pending{}, fmt.Errorf("%s: data: %w", o.Type, err)
	}
	if string(data) == "null" {
		data = json.RawMessage("{}")
	}
	var causation *string
	if o.Causation != "" {
		causation = &o.Causation
	}
	corr := o.Correlation
	if corr == "" {
		corr = o.EventID
	}
	profile := c.Profile
	if profile == "" {
		profile = "gost"
	}
	env := envelopeJSON{
		EventID: o.EventID, EventType: string(o.Type), SchemaVersion: info.CurrentVersion, SourceID: SourceEngine,
		OccurredAt: FormatTime(o.OccurredAt), CorrelationID: corr, CausationID: causation, RunID: o.RunID, ItemID: o.ItemID,
		Reaction:  o.Reaction,
		Integrity: integrityJSON{FormatVersion: 1, CryptoProfile: profile, Signers: []string{c.KeyRef}},
		Data:      data,
	}
	payload, err := engine.Canonical(env)
	if err != nil {
		return appjournal.Pending{}, err
	}
	sealed, err := c.Sealer.Seal(ctx, payload)
	if err != nil {
		return appjournal.Pending{}, fmt.Errorf("подпись конверта %s: %w", o.EventID, err)
	}
	e := jc.JournalEntry{
		Chain: jc.JournalEntryChainMain, EntryKind: jc.JournalEntryEntryKind(o.Kind), EventType: string(o.Type),
		SchemaVersion: info.CurrentVersion, EventID: o.EventID, SourceID: SourceEngine, Stream: o.Stream,
		OccurredAt: FormatTime(o.OccurredAt), CorrelationID: corr, CausationID: causation,
		ProvenanceClass: jc.JournalEntryProvenanceClassServerAttested, DomainBuild: c.DomainBuild,
	}
	if o.ItemID != "" {
		id := o.ItemID
		e.ItemID = &id
		e.Partition = kernel.PartitionOf(o.ItemID, c.Partitions)
	}
	if o.RunID != "" {
		run := o.RunID
		e.RunID = &run
	}
	if o.BasisSeq > 0 {
		b := int(o.BasisSeq)
		e.BasisSeq = &b
	}
	if o.NormRev != "" {
		rev := o.NormRev
		e.NormativeRev = &rev
	}
	if m := o.Reaction; m != nil {
		rule, v := m.RuleID, m.Version
		e.RuleID, e.Version = &rule, &v
		e.ReactionSlot = &jc.JournalEntryReactionSlot{RuleID: m.Slot.RuleID, Subject: m.Slot.Subject, TriggerKey: m.Slot.TriggerKey}
		if m.Supersedes != nil {
			s := *m.Supersedes
			e.Supersedes = &s
		}
	}
	return appjournal.Pending{Entry: e, Envelope: sealed}, nil
}

// ReactionOut — запись версии слота из плана сравнения (AD-3): basis_seq —
// основание пачки, causation_id — запись-триггер, «пересмотрен из-за записи»
// — revised_due_to конверта (FR-32).
func ReactionOut(p engine.Planned, itemID, runID, correlation, trigger, normRev string, basis int64) Out {
	r := p.Reaction
	m := &ReactionMeta{
		RuleID: r.Slot.RuleID, RuleRev: r.RuleRev, AutomationMode: r.AutomationMode,
		Slot:    SlotMeta{RuleID: r.Slot.RuleID, Subject: r.Slot.Subject, TriggerKey: r.Slot.TriggerKey},
		Version: p.Version, Causes: r.Causes, BasisSeq: basis,
	}
	if m.Causes == nil {
		m.Causes = []string{}
	}
	if p.Supersedes != "" {
		s := p.Supersedes
		m.Supersedes = &s
	}
	if p.RevisedDueTo != "" {
		s := p.RevisedDueTo
		m.RevisedDueTo = &s
	}
	stream := r.Slot.Subject
	if stream == "" || !strings.Contains(stream, ":") {
		stream = "item:" + itemID
	}
	return Out{
		EventID: p.EventID, Type: r.Type, Kind: catalog.KindReaction, Stream: stream, ItemID: itemID, RunID: runID,
		OccurredAt: r.OccurredAt, Correlation: correlation, Causation: trigger, BasisSeq: basis,
		Reaction: m, NormRev: normRev, Data: r.Data,
	}
}
