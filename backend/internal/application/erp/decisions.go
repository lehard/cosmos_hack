package erp

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
	dom "ant/internal/domain/erp"
)

// Decision — решение человека над исходящим сообщением (AD-2: вид «решение»):
// переотправка из карантина или решение об исправлении.
type Decision struct {
	Type catalog.Type
	Key  string
	Data any
	Meta platform.CommandMeta
	// Actor — псевдоним автора (Principal.PersonID).
	Actor string
	// OccurredAt — доменное «сейчас» при приёме команды (AD-37).
	OccurredAt time.Time
	// RunID — прогон сценария сообщения (AD-38).
	RunID string
}

// DecisionWriter — ведомый порт записи решений (AD-44: пишет только journal.Append).
type DecisionWriter interface {
	Write(ctx context.Context, d Decision) (platform.Receipt, error)
}

// SourceAPI — source_id решений, принятых операциями API.
const SourceAPI = "ant-api"

// JournalDecisions — DecisionWriter над журналом: конверт DSSE без подписей
// (демо без агента токена, Д-30), подписант — `‹псевдоним›@1`; повтор с тем же
// command_id возвращает прежнюю квитанцию (AD-7).
type JournalDecisions struct {
	Journal     appjournal.JournalStore
	DomainBuild string
	// Now — InfraClock для received_at (AD-37); nil — time.Now.
	Now func() time.Time
}

var _ DecisionWriter = JournalDecisions{}

// Write записывает решение в поток сообщения `erp_message:‹ключ›`.
func (w JournalDecisions) Write(ctx context.Context, d Decision) (platform.Receipt, error) {
	id := strings.ToLower(d.Meta.CommandID)
	if _, err := uuid.Parse(id); err != nil || id == "" {
		id = uuid.NewV7().String()
	}
	info, ok := catalog.Lookup(d.Type)
	if !ok || info.Emitter != string(dom.Module) || info.Kind != catalog.KindDecision {
		return platform.Receipt{}, errors.New("erp: решение чужого типа или не решение: " + string(d.Type))
	}
	data, err := json.Marshal(d.Data)
	if err != nil {
		return platform.Receipt{}, err
	}
	occurred := engineapp.FormatTime(d.OccurredAt)
	actor := strings.ToLower(d.Actor)
	if actor == "" {
		actor = "anonymous"
	}
	stream := dom.Stream(d.Key)
	env := map[string]any{
		"event_id": id, "event_type": string(d.Type), "schema_version": info.CurrentVersion, "source_id": SourceAPI,
		"occurred_at": occurred, "correlation_id": id, "causation_id": nil,
		"command":   map[string]any{"command_id": id, "basis_seq": d.Meta.BasisSeq, "guard_streams": []string{stream}, "policy_seq": d.Meta.PolicySeq},
		"integrity": map[string]any{"format_version": 1, "crypto_profile": "gost", "signers": []string{actor + "@1"}},
		"data":      json.RawMessage(data),
	}
	if d.RunID != "" {
		env["run_id"] = d.RunID
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
		SchemaVersion: info.CurrentVersion, EventID: id, SourceID: SourceAPI, Stream: stream,
		OccurredAt: occurred, ReceivedAt: engineapp.FormatTime(now()), CorrelationID: id,
		ProvenanceClass: jc.JournalEntryProvenanceClassPersonal, DomainBuild: w.DomainBuild,
	}
	if d.RunID != "" {
		run := d.RunID
		e.RunID = &run
	}
	if d.Meta.BasisSeq > 0 {
		b := int(d.Meta.BasisSeq)
		e.BasisSeq = &b
	}
	rq := appjournal.AppendRequest{Batch: []appjournal.Pending{{Entry: e, Envelope: sealed}}}
	if d.Meta.BasisSeq > 0 {
		rq.Checks = []appjournal.Check{{Stream: stream, BasisSeq: d.Meta.BasisSeq}}
	}
	res, err := w.Journal.Append(ctx, rq)
	if errors.Is(err, appjournal.ErrDuplicate) {
		return w.replay(ctx, stream, d.Type, id)
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

// replay — прежняя квитанция команды с тем же command_id (AD-7).
func (w JournalDecisions) replay(ctx context.Context, stream string, t catalog.Type, id string) (platform.Receipt, error) {
	var after int64
	for {
		es, err := w.Journal.Read(ctx, appjournal.ReadQuery{Stream: stream, EventType: string(t), AfterSeq: after, Limit: 1000})
		if err != nil {
			return platform.Receipt{}, err
		}
		for _, e := range es {
			after = int64(e.Seq)
			if e.EventID == id {
				at, _ := time.Parse(time.RFC3339Nano, e.CommittedAt)
				return platform.Receipt{CommandID: id, Seq: int64(e.Seq), EventIDs: []string{id}, RecordedAt: at, Replayed: true}, nil
			}
		}
		if len(es) < 1000 {
			return platform.Receipt{}, errors.New("erp: повтор команды " + id + ": запись не найдена")
		}
	}
}
