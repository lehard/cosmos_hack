package analysis

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
	dom "ant/internal/domain/analysis"
	"ant/internal/domain/engine"
	"ant/internal/domain/kernel"
)

// Ведомые порты модуля analysis.

// Decision — решение человека к записи (AD-2: вид «решение», AD-39): тип
// модуля analysis, поток объекта, данные и метаданные команды.
type Decision struct {
	Type   catalog.Type
	Stream string
	// ItemID — изделие, если решение в потоке изделия.
	ItemID string
	Data   any
	Meta   platform.CommandMeta
	// Actor — псевдоним автора (Principal.PersonID).
	Actor string
	// OccurredAt — доменное «сейчас» при приёме команды (AD-37).
	OccurredAt time.Time
	// GuardStreams — потоки, которые проверил гард (AD-39).
	GuardStreams []string
	// SignatureLevel — уровень подписи операции (AD-13).
	SignatureLevel int
}

// DecisionWriter — ведомый порт записи решений (AD-44: в журнал пишет только
// journal.Append): одна команда — одна пачка с проверками AD-39.
type DecisionWriter interface {
	Write(ctx context.Context, d Decision) (platform.Receipt, error)
}

// JournalDecisions — DecisionWriter над журналом (JournalStore.Append).
// Конверт решения — DSSE без подписей (демо без агента токена, Д-30): подпись
// личным ключом собирает агент токена (эпики 27, 38); подписант конверта —
// `‹псевдоним›@1` до реестра ключей (эпик 05).
type JournalDecisions struct {
	Journal     appjournal.JournalStore
	DomainBuild string
	Partitions  int
	// Now — InfraClock для received_at (AD-37); nil — time.Now.
	Now func() time.Time
}

var _ DecisionWriter = JournalDecisions{}

// SourceAPI — source_id решений, принятых операциями API.
const SourceAPI = "ant-api"

// Write записывает решение. Повтор с тем же command_id (event_id решения =
// command_id) возвращает прежнюю квитанцию (AD-7).
func (w JournalDecisions) Write(ctx context.Context, d Decision) (platform.Receipt, error) {
	id := strings.ToLower(d.Meta.CommandID)
	if _, err := uuid.Parse(id); err != nil || id == "" {
		id = uuid.NewV7().String()
	}
	info, ok := catalog.Lookup(d.Type)
	if !ok || info.Emitter != string(dom.Module) || info.Kind != catalog.KindDecision {
		return platform.Receipt{}, errors.New("analysis: решение чужого типа или не решение: " + string(d.Type))
	}
	data, err := json.Marshal(d.Data)
	if err != nil {
		return platform.Receipt{}, err
	}
	occurred := engineapp.FormatTime(d.OccurredAt)
	guard := d.GuardStreams
	if guard == nil {
		guard = []string{}
	}
	cmd := map[string]any{"command_id": id, "basis_seq": d.Meta.BasisSeq, "guard_streams": guard, "policy_seq": d.Meta.PolicySeq,
		"signature_level": d.SignatureLevel}
	if d.Meta.WorkplaceID != "" {
		cmd["workplace_id"] = d.Meta.WorkplaceID
	}
	env := map[string]any{
		"event_id": id, "event_type": string(d.Type), "schema_version": info.CurrentVersion, "source_id": SourceAPI,
		"occurred_at": occurred, "correlation_id": id, "causation_id": nil, "command": cmd,
		"integrity": map[string]any{"format_version": 1, "crypto_profile": "gost", "signers": []string{signer(d.Actor)}},
		"data":      json.RawMessage(data),
	}
	if d.ItemID != "" {
		env["item_id"] = d.ItemID
	}
	canon, err := engine.Canonical(env)
	if err != nil {
		return platform.Receipt{}, err
	}
	sealed, _ := json.Marshal(struct {
		PayloadType string   `json:"payloadType"`
		Payload     string   `json:"payload"`
		Signatures  []string `json:"signatures"`
	}{engineapp.PayloadTypeEvent, base64.StdEncoding.EncodeToString(canon), []string{}})
	now := time.Now
	if w.Now != nil {
		now = w.Now
	}
	e := jc.JournalEntry{
		Chain: jc.JournalEntryChainMain, EntryKind: jc.JournalEntryEntryKindDecision, EventType: string(d.Type),
		SchemaVersion: info.CurrentVersion, EventID: id, SourceID: SourceAPI, Stream: d.Stream,
		OccurredAt: occurred, ReceivedAt: engineapp.FormatTime(now()), CorrelationID: id,
		ProvenanceClass: jc.JournalEntryProvenanceClassPersonal, DomainBuild: w.DomainBuild,
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
	rq := appjournal.AppendRequest{Batch: []appjournal.Pending{{Entry: e, Envelope: sealed}}}
	if d.Meta.BasisSeq > 0 {
		for _, s := range guard {
			rq.Checks = append(rq.Checks, appjournal.Check{Stream: s, BasisSeq: d.Meta.BasisSeq})
		}
	}
	res, err := w.Journal.Append(ctx, rq)
	if errors.Is(err, appjournal.ErrDuplicate) {
		return w.replay(ctx, d, id)
	}
	if err != nil {
		if pe, ok := appjournal.Problem(err); ok {
			return platform.Receipt{}, pe
		}
		return platform.Receipt{}, err
	}
	rc := platform.Receipt{CommandID: id, EventIDs: []string{id}, RecordedAt: res.Committed}
	if len(res.Seqs) > 0 {
		rc.Seq = res.Seqs[0]
	}
	return rc, nil
}

// replay — квитанция уже записанного решения (повтор команды, AD-7).
func (w JournalDecisions) replay(ctx context.Context, d Decision, id string) (platform.Receipt, error) {
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
			return platform.Receipt{}, errors.New("analysis: повтор команды " + id + ": запись не найдена")
		}
	}
}

// signer — подписант конверта решения: key_ref `‹псевдоним›@1` (нижний регистр).
func signer(actor string) string {
	a := strings.ToLower(actor)
	if a == "" {
		a = "anonymous"
	}
	return a + "@1"
}
