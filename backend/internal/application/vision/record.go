package vision

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"
	"uuid"

	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	jc "ant/internal/contracts/journal"
	"ant/internal/domain/engine"
	dom "ant/internal/domain/vision"
)

// Источники записей модуля.
const (
	// SourceAPI — source_id решений, принятых операциями API.
	SourceAPI = "ant-api"
	// SourceGenesis — source_id записей демо-затравки паспортов (AD-33).
	SourceGenesis = "ant-genesis"
)

// entry — запись-решение модуля vision к записи в журнал (AD-2: вид «решение»).
type entry struct {
	ID         string
	Type       catalog.Type
	Stream     string
	Data       any
	OccurredAt time.Time
	ReceivedAt time.Time
	SourceID   string
	Signer     string
	Provenance jc.JournalEntryProvenanceClass
	// Command — блок command конверта (обязателен у решений, AD-39).
	Command map[string]any
	// RecordedAt — задать recorded_at (часы scenario, AD-37); пусто — Append ставит сам.
	RecordedAt string
	Meta       platform.CommandMeta
	// RunID — прогон сценария (AD-38): возврат после отката в прогоне — в том же прогоне.
	RunID string
}

// pending — открытые поля и конверт DSSE без подписей (демо без агента
// токена, Д-30; подпись личным ключом — эпики 27, 38).
func pending(e entry, domainBuild string, partition int) (appjournal.Pending, error) {
	info, ok := catalog.Lookup(e.Type)
	if !ok || info.Emitter != string(dom.Module) || info.Kind != catalog.KindDecision {
		return appjournal.Pending{}, errors.New("vision: решение чужого типа или не решение: " + string(e.Type))
	}
	data, err := json.Marshal(e.Data)
	if err != nil {
		return appjournal.Pending{}, err
	}
	occurred := engineapp.FormatTime(e.OccurredAt)
	env := map[string]any{
		"event_id": e.ID, "event_type": string(e.Type), "schema_version": info.CurrentVersion, "source_id": e.SourceID,
		"source_kind": "manual_entry", "occurred_at": occurred, "correlation_id": e.ID, "causation_id": nil, "command": e.Command,
		"integrity": map[string]any{"format_version": 1, "crypto_profile": "gost", "signers": []string{e.Signer}},
		"data":      json.RawMessage(data),
	}
	if e.RunID != "" {
		env["run_id"] = e.RunID
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
	je := jc.JournalEntry{
		Chain: jc.JournalEntryChainMain, EntryKind: jc.JournalEntryEntryKindDecision, EventType: string(e.Type),
		SchemaVersion: info.CurrentVersion, EventID: e.ID, SourceID: e.SourceID, Stream: e.Stream,
		OccurredAt: occurred, ReceivedAt: engineapp.FormatTime(e.ReceivedAt), CorrelationID: e.ID,
		ProvenanceClass: e.Provenance, DomainBuild: domainBuild, RecordedAt: e.RecordedAt,
		// Записи вне изделия — партиция стадии за пределами 0…P-1 (как у приёма).
		Partition: partition,
	}
	if e.RunID != "" {
		run := e.RunID
		je.RunID = &run
	}
	if e.Meta.BasisSeq > 0 {
		b := int(e.Meta.BasisSeq)
		je.BasisSeq = &b
	}
	if e.Meta.PolicySeq > 0 {
		p := int(e.Meta.PolicySeq)
		je.PolicySeq = &p
	}
	return appjournal.Pending{Entry: je, Envelope: sealed}, nil
}

// signer — подписант конверта `‹псевдоним›@1` до реестра ключей (эпик 05).
func signer(person string) string {
	if person == "" {
		return "anonymous@1"
	}
	return person + "@1"
}

// commandID — event_id решения = command_id клиента (AD-7); не UUID — новый UUIDv7.
func commandID(meta platform.CommandMeta) string {
	id := strings.ToLower(meta.CommandID)
	if _, err := uuid.Parse(id); err != nil || id == "" {
		id = uuid.NewV7().String()
	}
	return id
}

// write — одна запись-решение через journal.Append (AD-44) с проверкой AD-39
// потока паспорта: после basis_seq в нём нет новых записей guard_relevant.
// Повтор с тем же command_id возвращает прежнюю квитанцию (AD-7). Запись
// журнала критических действий (AD-28) — CriticalActions эпика 29.
func (s *Service) write(ctx context.Context, t catalog.Type, stream string, data any, meta platform.CommandMeta, occurred time.Time, runID string) (platform.Receipt, error) {
	id := commandID(meta)
	if r, ok, err := s.replay(ctx, stream, t, id); err != nil || ok {
		return r, err
	}
	actor := platform.PrincipalFrom(ctx).PersonID
	cmd := map[string]any{"command_id": id, "basis_seq": meta.BasisSeq, "guard_streams": []string{stream}, "policy_seq": meta.PolicySeq,
		"signature_level": 2}
	if meta.WorkplaceID != "" {
		cmd["workplace_id"] = meta.WorkplaceID
	}
	// Д-59: подпись команды, принятая декоратором (signing.CheckCommand), — рядом с записью.
	prov, err := platform.SignRecord(ctx, cmd, "")
	if err != nil {
		return platform.Receipt{}, err
	}
	e := entry{ID: id, Type: t, Stream: stream, Data: data, OccurredAt: occurred, ReceivedAt: s.d.Now(), SourceID: SourceAPI,
		Signer: signer(actor), Provenance: jc.JournalEntryProvenanceClassPersonal, Command: cmd, Meta: meta, RunID: runID}
	if info, ok := catalog.Lookup(t); ok && prov != "" && slices.Contains(info.Provenance, prov) {
		e.Provenance = jc.JournalEntryProvenanceClass(prov)
	}
	if s.cfg.ScenarioClock {
		e.RecordedAt = engineapp.FormatTime(occurred)
	}
	p, err := pending(e, s.cfg.DomainBuild, s.cfg.Partitions)
	if err != nil {
		return platform.Receipt{}, err
	}
	rq := appjournal.AppendRequest{Batch: []appjournal.Pending{p},
		Checks: []appjournal.Check{{Stream: stream, BasisSeq: meta.BasisSeq}}}
	res, err := s.d.Journal.Append(ctx, rq)
	if errors.Is(err, appjournal.ErrDuplicate) {
		r, _, err := s.replay(ctx, stream, t, id)
		return r, err
	}
	if err != nil {
		if pe, ok := appjournal.Problem(err); ok {
			return platform.Receipt{}, pe
		}
		return platform.Receipt{}, err
	}
	r := platform.Receipt{CommandID: id, EventIDs: []string{id}, RecordedAt: occurred}
	if len(res.Seqs) > 0 {
		r.Seq = res.Seqs[0]
	}
	if !s.cfg.ScenarioClock && !res.Committed.IsZero() {
		r.RecordedAt = res.Committed.UTC()
	}
	return r, nil
}

// replayed — повтор команды (AD-7) проверяется до гарда: повтор не должен
// получать отказ гарда из-за собственной первой записи.
func (s *Service) replayed(ctx context.Context, stream string, t catalog.Type, meta platform.CommandMeta) (platform.Receipt, bool, error) {
	id := strings.ToLower(meta.CommandID)
	if _, err := uuid.Parse(id); err != nil || id == "" {
		return platform.Receipt{}, false, nil
	}
	return s.replay(ctx, stream, t, id)
}

// replay — команда с этим command_id уже записана (AD-7): прежняя квитанция.
func (s *Service) replay(ctx context.Context, stream string, t catalog.Type, id string) (platform.Receipt, bool, error) {
	q := appjournal.ReadQuery{Stream: stream, EventType: string(t), Limit: 1000}
	for {
		page, err := s.d.Journal.Read(ctx, q)
		if err != nil {
			return platform.Receipt{}, false, err
		}
		for _, e := range page {
			if e.EventID == id {
				at, _ := time.Parse(time.RFC3339Nano, e.RecordedAt)
				return platform.Receipt{CommandID: id, Seq: int64(e.Seq), EventIDs: []string{id}, RecordedAt: at.UTC(), Replayed: true}, true, nil
			}
		}
		if len(page) < q.Limit {
			return platform.Receipt{}, false, nil
		}
		q.AfterSeq = int64(page[len(page)-1].Seq)
	}
}
