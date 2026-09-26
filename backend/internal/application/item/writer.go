package item

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"uuid"

	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	jc "ant/internal/contracts/journal"
	"ant/internal/domain/engine"
	"ant/internal/domain/kernel"
)

// Запись фактов и решений, принятых операциями API модулей item и crossitem
// (AD-2, AD-39, AD-44): в журнал пишет только journal.Append; одна команда —
// одна пачка с проверками потоков гарда. Конверт — DSSE без подписей (демо
// без агента токена, Д-30); факт, введённый человеком через API, помечен
// source_kind = manual_entry (FR-140: ручная отметка не выдаётся за данные станка).

// SourceAPI — source_id записей, принятых операциями API.
const SourceAPI = "ant-api"

// Record — запись к журналу: тип модуля-владельца, поток, данные, метаданные команды.
type Record struct {
	Type   catalog.Type
	Stream string
	// ItemID — изделие, если запись в потоке изделия (партиция — по нему, AD-6).
	ItemID string
	Data   any
	Meta   platform.CommandMeta
	// EventID — id записи; пусто — command_id (повтор команды — та же запись, AD-7).
	EventID string
	// Actor — псевдоним автора (Principal.PersonID).
	Actor string
	// OccurredAt — доменное «сейчас» при приёме команды (AD-37).
	OccurredAt time.Time
	// GuardStreams — потоки, которые проверил гард (AD-39).
	GuardStreams []string
	// SignatureLevel — уровень подписи операции (AD-13).
	SignatureLevel int
}

// Writer — ведомый порт записи команд модулей item и crossitem.
type Writer interface {
	// Write записывает пачку записей одной команды; квитанция — по первой.
	Write(ctx context.Context, owner kernel.Module, recs []Record) (platform.Receipt, error)
}

// JournalWriter — Writer над журналом (JournalStore.Append).
type JournalWriter struct {
	Journal     appjournal.JournalStore
	DomainBuild string
	Partitions  int
	// Now — InfraClock для received_at (AD-37); nil — time.Now.
	Now func() time.Time
}

var _ Writer = JournalWriter{}

// Write записывает записи одной пачкой. Повтор с тем же command_id (event_id
// первой записи = command_id) возвращает прежнюю квитанцию (AD-7).
func (w JournalWriter) Write(ctx context.Context, owner kernel.Module, recs []Record) (platform.Receipt, error) {
	if len(recs) == 0 {
		return platform.Receipt{}, errors.New("item: пустая команда")
	}
	cmdID := strings.ToLower(recs[0].Meta.CommandID)
	if _, err := uuid.Parse(cmdID); err != nil || cmdID == "" {
		cmdID = uuid.NewV7().String()
	}
	now := time.Now
	if w.Now != nil {
		now = w.Now
	}
	var rq appjournal.AppendRequest
	ids := make([]string, 0, len(recs))
	checked := map[string]bool{}
	for i, d := range recs {
		info, ok := catalog.Lookup(d.Type)
		if !ok || info.Emitter != string(owner) || (info.Kind != catalog.KindDecision && info.Kind != catalog.KindFact) {
			return platform.Receipt{}, errors.New("item: запись чужого типа или не факт и не решение: " + string(d.Type))
		}
		id := strings.ToLower(d.EventID)
		switch {
		case id != "":
		case i == 0:
			id = cmdID
		default:
			id = kernel.UUIDv5(cmdID, string(d.Type)+"\x1f"+d.Stream)
		}
		ids = append(ids, id)
		pend, err := w.pending(d, info, id, cmdID, now())
		if err != nil {
			return platform.Receipt{}, err
		}
		rq.Batch = append(rq.Batch, pend)
		if d.Meta.BasisSeq > 0 {
			for _, s := range d.GuardStreams {
				if !checked[s] {
					checked[s] = true
					rq.Checks = append(rq.Checks, appjournal.Check{Stream: s, BasisSeq: d.Meta.BasisSeq})
				}
			}
		}
	}
	res, err := w.Journal.Append(ctx, rq)
	if errors.Is(err, appjournal.ErrDuplicate) {
		return w.replay(ctx, recs[0], ids[0])
	}
	if err != nil {
		if pe, ok := appjournal.Problem(err); ok {
			return platform.Receipt{}, pe
		}
		return platform.Receipt{}, err
	}
	rc := platform.Receipt{CommandID: cmdID, EventIDs: ids, RecordedAt: res.Committed}
	if len(res.Seqs) > 0 {
		rc.Seq = res.Seqs[0]
	}
	return rc, nil
}

func (w JournalWriter) pending(d Record, info catalog.Info, id, cmdID string, received time.Time) (appjournal.Pending, error) {
	data, err := json.Marshal(d.Data)
	if err != nil {
		return appjournal.Pending{}, err
	}
	occurred := engineapp.FormatTime(d.OccurredAt)
	guard := d.GuardStreams
	if guard == nil {
		guard = []string{}
	}
	env := map[string]any{
		"event_id": id, "event_type": string(d.Type), "schema_version": info.CurrentVersion, "source_id": SourceAPI,
		"occurred_at": occurred, "correlation_id": cmdID, "causation_id": nil,
		"integrity": map[string]any{"format_version": 1, "crypto_profile": "gost", "signers": []string{signer(d.Actor)}},
		"data":      json.RawMessage(data),
	}
	kind := jc.JournalEntryEntryKindDecision
	if info.Kind == catalog.KindFact {
		// Факт, введённый человеком через API (FR-140).
		kind = jc.JournalEntryEntryKindFact
		env["source_kind"] = "manual_entry"
	} else {
		cmd := map[string]any{"command_id": cmdID, "basis_seq": d.Meta.BasisSeq, "guard_streams": guard, "policy_seq": d.Meta.PolicySeq,
			"signature_level": d.SignatureLevel}
		if d.Meta.WorkplaceID != "" {
			cmd["workplace_id"] = d.Meta.WorkplaceID
		}
		env["command"] = cmd
	}
	if d.ItemID != "" {
		env["item_id"] = d.ItemID
	}
	canon, err := engine.Canonical(env)
	if err != nil {
		return appjournal.Pending{}, err
	}
	sealed, _ := json.Marshal(struct {
		PayloadType string   `json:"payloadType"`
		Payload     string   `json:"payload"`
		Signatures  []string `json:"signatures"`
	}{engineapp.PayloadTypeEvent, base64.StdEncoding.EncodeToString(canon), []string{}})
	e := jc.JournalEntry{
		Chain: jc.JournalEntryChainMain, EntryKind: kind, EventType: string(d.Type), SchemaVersion: info.CurrentVersion,
		EventID: id, SourceID: SourceAPI, Stream: d.Stream, OccurredAt: occurred, ReceivedAt: engineapp.FormatTime(received),
		CorrelationID: cmdID, ProvenanceClass: jc.JournalEntryProvenanceClassPersonal, DomainBuild: w.DomainBuild,
	}
	if d.ItemID != "" {
		item := d.ItemID
		e.ItemID = &item
		e.Partition = kernel.PartitionOf(d.ItemID, w.Partitions)
	}
	if d.Meta.BasisSeq > 0 {
		b := int(d.Meta.BasisSeq)
		e.BasisSeq = &b
	}
	return appjournal.Pending{Entry: e, Envelope: sealed}, nil
}

// replay — квитанция уже записанной команды (повтор, AD-7).
func (w JournalWriter) replay(ctx context.Context, d Record, id string) (platform.Receipt, error) {
	var after int64
	for {
		es, err := w.Journal.Read(ctx, appjournal.ReadQuery{Stream: d.Stream, EventType: string(d.Type), AfterSeq: after, Limit: 1000})
		if err != nil {
			return platform.Receipt{}, err
		}
		for _, e := range es {
			after = int64(e.Seq)
			if e.EventID == id {
				t, _ := time.Parse(time.RFC3339Nano, e.CommittedAt)
				return platform.Receipt{CommandID: id, Seq: int64(e.Seq), EventIDs: []string{id}, RecordedAt: t, Replayed: true}, nil
			}
		}
		if len(es) < 1000 {
			return platform.Receipt{}, errors.New("item: повтор команды " + id + ": запись не найдена")
		}
	}
}

// signer — подписант конверта: key_ref `‹псевдоним›@1` (нижний регистр).
func signer(actor string) string {
	a := strings.ToLower(actor)
	if a == "" {
		a = "anonymous"
	}
	return a + "@1"
}
