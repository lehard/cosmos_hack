package nonconformity

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
	"ant/internal/application/security"
	"ant/internal/contracts/catalog"
	jc "ant/internal/contracts/journal"
	"ant/internal/domain/engine"
	"ant/internal/domain/kernel"
	dom "ant/internal/domain/nonconformity"
	domsecurity "ant/internal/domain/security"
)

// SourceAPI — source_id решений, принятых операциями API модуля.
const SourceAPI = "ant-api"

// decision — решение человека к записи (AD-2: вид «решение»; AD-39).
type decision struct {
	Type   catalog.Type
	Stream string
	// ItemID — изделие, если решение в потоке изделия.
	ItemID string
	RunID  string
	Data   any
	Meta   platform.CommandMeta
	// Actor — псевдоним автора (Principal.PersonID).
	Actor      string
	OccurredAt time.Time
	// Checks — проверки AD-39 (потоки гарда, расход лимита разрешения).
	Checks []appjournal.Check
	// Grants — лимит разрешения, открываемый записью (decision.concession.granted).
	Grants         []appjournal.ConcessionGrant
	SignatureLevel int
	GuardStreams   []string
}

// signer — подписант конверта `‹псевдоним›@1` до реестра ключей (эпик 05).
func signer(person string) string {
	if person == "" {
		return "anonymous@1"
	}
	return person + "@1"
}

// write — одна запись-решение через journal.Append (AD-44) с проверками
// AD-39 в той же транзакции: после basis_seq в потоках гарда нет новых
// записей guard_relevant, остаток лимита разрешения достаточен (расход —
// атомарно с решением). Иначе — 409 journal.stale_state или
// journal.concession_exhausted. Повтор с тем же command_id (event_id решения
// = command_id) возвращает прежнюю квитанцию (AD-7).
//
// Конверт — DSSE без подписей (демо без агента токена, Д-30): подпись
// личным ключом собирает агент токена (эпики 27, 38); запись критического
// действия (AD-28) — через CriticalActions (security.Execute, эпик 29).
func (s *Service) write(ctx context.Context, d decision) (platform.Receipt, error) {
	id := strings.ToLower(d.Meta.CommandID)
	if _, err := uuid.Parse(id); err != nil || id == "" {
		id = uuid.NewV7().String()
	}
	info, ok := catalog.Lookup(d.Type)
	if !ok || info.Emitter != string(dom.Module) || info.Kind != catalog.KindDecision {
		return platform.Receipt{}, errors.New("nonconformity: решение чужого типа или не решение: " + string(d.Type))
	}
	if r, ok, err := s.replayed(ctx, d.Stream, d.Type, id); err != nil || ok {
		return r, err
	}
	p, err := s.pending(d, id)
	if err != nil {
		return platform.Receipt{}, err
	}
	rq := appjournal.AppendRequest{Batch: []appjournal.Pending{p}, Checks: d.Checks, ConcessionGrants: d.Grants}
	// AD-28: критическое действие — через сервис доверенных решений: запись
	// CA строится в транзакции этого Append (эпик 29).
	var res appjournal.AppendResult
	err = security.Execute(ctx, domsecurity.Command{ActorID: d.Actor, PolicySeq: d.Meta.PolicySeq}, func(ctx context.Context) error {
		var err error
		res, err = s.d.Journal.Append(ctx, rq)
		return err
	})
	if errors.Is(err, appjournal.ErrDuplicate) {
		return s.replay(ctx, d, id)
	}
	if err != nil {
		if pe, ok := appjournal.Problem(err); ok {
			return platform.Receipt{}, pe
		}
		return platform.Receipt{}, err
	}
	r := platform.Receipt{CommandID: id, EventIDs: []string{id}, RecordedAt: d.OccurredAt}
	if len(res.Seqs) > 0 {
		r.Seq = res.Seqs[0]
	}
	if !s.cfg.ScenarioClock && !res.Committed.IsZero() {
		r.RecordedAt = res.Committed.UTC()
	}
	return r, nil
}

// pending — запись-решение d с event_id id: конверт DSSE и строка журнала.
func (s *Service) pending(d decision, id string) (appjournal.Pending, error) {
	info, _ := catalog.Lookup(d.Type)
	data, err := json.Marshal(d.Data)
	if err != nil {
		return appjournal.Pending{}, err
	}
	occurred := engineapp.FormatTime(d.OccurredAt)
	guard := d.GuardStreams
	if guard == nil {
		guard = []string{}
	}
	// Команда группового решения пишет по записи в поток каждого изделия:
	// event_id у записей свои, command_id и correlation_id — команды.
	cmdID := strings.ToLower(d.Meta.CommandID)
	if _, err := uuid.Parse(cmdID); err != nil || cmdID == "" {
		cmdID = id
	}
	cmd := map[string]any{"command_id": cmdID, "basis_seq": d.Meta.BasisSeq, "guard_streams": guard, "policy_seq": d.Meta.PolicySeq,
		"signature_level": d.SignatureLevel}
	if d.Meta.WorkplaceID != "" {
		cmd["workplace_id"] = d.Meta.WorkplaceID
	}
	env := map[string]any{
		"event_id": id, "event_type": string(d.Type), "schema_version": info.CurrentVersion, "source_id": SourceAPI,
		"source_kind": "manual_entry", "occurred_at": occurred, "correlation_id": cmdID, "causation_id": nil, "command": cmd,
		"integrity": map[string]any{"format_version": 1, "crypto_profile": "gost", "signers": []string{signer(d.Actor)}},
		"data":      json.RawMessage(data),
	}
	if d.ItemID != "" {
		env["item_id"] = d.ItemID
	}
	if d.RunID != "" {
		env["run_id"] = d.RunID
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
		Chain: jc.JournalEntryChainMain, EntryKind: jc.JournalEntryEntryKindDecision, EventType: string(d.Type),
		SchemaVersion: info.CurrentVersion, EventID: id, SourceID: SourceAPI, Stream: d.Stream,
		OccurredAt: occurred, ReceivedAt: engineapp.FormatTime(s.d.Now()), CorrelationID: cmdID,
		ProvenanceClass: jc.JournalEntryProvenanceClassPersonal, DomainBuild: s.cfg.DomainBuild,
	}
	if s.cfg.ScenarioClock {
		e.RecordedAt = occurred
	}
	if d.ItemID != "" {
		item := d.ItemID
		e.ItemID = &item
		e.Partition = kernel.PartitionOf(d.ItemID, s.cfg.Partitions)
	} else {
		// Записи вне изделия — партиция стадии за пределами 0…P-1 (как у приёма).
		e.Partition = s.cfg.Partitions
	}
	if d.RunID != "" {
		run := d.RunID
		e.RunID = &run
	}
	if d.Meta.BasisSeq > 0 {
		b := int(d.Meta.BasisSeq)
		e.BasisSeq = &b
	}
	if d.Meta.PolicySeq > 0 {
		p := int(d.Meta.PolicySeq)
		e.PolicySeq = &p
	}
	return appjournal.Pending{Entry: e, Envelope: sealed}, nil
}

// replayed — команда с этим command_id уже записана (AD-7): прежняя
// квитанция. Проверяется до гарда: повтор не должен получать отказ гарда или
// 409 из-за собственной первой записи.
func (s *Service) replayed(ctx context.Context, stream string, t catalog.Type, commandID string) (platform.Receipt, bool, error) {
	id := strings.ToLower(commandID)
	if _, err := uuid.Parse(id); err != nil || id == "" {
		return platform.Receipt{}, false, nil
	}
	r, err := s.replay(ctx, decision{Stream: stream, Type: t}, id)
	if err != nil || r.Seq == 0 {
		return platform.Receipt{}, false, err
	}
	return r, true, nil
}

// replay — прежняя квитанция команды с тем же command_id (AD-7).
func (s *Service) replay(ctx context.Context, d decision, id string) (platform.Receipt, error) {
	es, err := s.readAll(ctx, appjournal.ReadQuery{Stream: d.Stream, EventType: string(d.Type)})
	if err != nil {
		return platform.Receipt{}, err
	}
	for _, e := range es {
		if e.EventID == id {
			t, _ := time.Parse(time.RFC3339Nano, e.RecordedAt)
			return platform.Receipt{CommandID: id, Seq: int64(e.Seq), EventIDs: []string{id}, RecordedAt: t.UTC(), Replayed: true}, nil
		}
	}
	return platform.Receipt{CommandID: id, EventIDs: []string{id}, Replayed: true}, nil
}
