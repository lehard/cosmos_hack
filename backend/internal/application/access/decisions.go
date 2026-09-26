package access

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
)

// SourceAPI — source_id решений, принятых операциями API.
const SourceAPI = "ant-api"

// Record — запись решения модуля access: тип (эмитент — access, AD-40), поток, data.
type Record struct {
	Type   catalog.Type
	Stream string
	Data   any
}

// Batch — решение человека одной пачкой записей (AD-44): например,
// активация учётной записи = сотрудник заведён + учётная запись активирована
// + роль назначена — атомарно.
type Batch struct {
	Records []Record
	Meta    platform.CommandMeta
	// Actor — псевдоним автора (Principal.PersonID).
	Actor string
	// OccurredAt — доменное «сейчас» при приёме команды (AD-37).
	OccurredAt time.Time
}

// DecisionWriter — ведомый порт записи решений (в журнал пишет только journal.Append).
type DecisionWriter interface {
	Write(ctx context.Context, d Batch) (platform.Receipt, error)
}

// JournalDecisions — DecisionWriter над журналом (JournalStore.Append).
// Конверт — DSSE без подписей (демо без агента токена, Д-30); подписант —
// `‹псевдоним›@1` до реестра ключей (эпик 05).
type JournalDecisions struct {
	Journal     appjournal.JournalStore
	DomainBuild string
	// Now — InfraClock для received_at (AD-37); nil — time.Now.
	Now func() time.Time
}

var _ DecisionWriter = JournalDecisions{}

// Write записывает решение. Повтор с тем же command_id (event_id первой
// записи = command_id) возвращает прежнюю квитанцию (AD-7).
func (w JournalDecisions) Write(ctx context.Context, d Batch) (platform.Receipt, error) {
	if len(d.Records) == 0 {
		return platform.Receipt{}, errors.New("access: пустое решение")
	}
	first := strings.ToLower(d.Meta.CommandID)
	if _, err := uuid.Parse(first); err != nil || first == "" {
		first = uuid.NewV7().String()
	}
	now := time.Now
	if w.Now != nil {
		now = w.Now
	}
	occurred := engineapp.FormatTime(d.OccurredAt)
	var rq appjournal.AppendRequest
	ids := make([]string, 0, len(d.Records))
	for i, r := range d.Records {
		info, ok := catalog.Lookup(r.Type)
		if !ok || info.Emitter != "access" || info.Kind != catalog.KindDecision {
			return platform.Receipt{}, errors.New("access: решение чужого типа или не решение: " + string(r.Type))
		}
		id := first
		if i > 0 {
			id = uuid.NewV7().String()
		}
		ids = append(ids, id)
		data, err := json.Marshal(r.Data)
		if err != nil {
			return platform.Receipt{}, err
		}
		cmd := map[string]any{"command_id": first, "basis_seq": d.Meta.BasisSeq, "guard_streams": []string{r.Stream}, "policy_seq": d.Meta.PolicySeq, "signature_level": 0}
		env := map[string]any{
			"event_id": id, "event_type": string(r.Type), "schema_version": info.CurrentVersion, "source_id": SourceAPI,
			"occurred_at": occurred, "correlation_id": first, "causation_id": nil, "command": cmd,
			"integrity": map[string]any{"format_version": 1, "crypto_profile": "gost", "signers": []string{signer(d.Actor)}},
			"data":      json.RawMessage(data),
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
		e := jc.JournalEntry{
			Chain: jc.JournalEntryChainMain, EntryKind: jc.JournalEntryEntryKindDecision, EventType: string(r.Type),
			SchemaVersion: info.CurrentVersion, EventID: id, SourceID: SourceAPI, Stream: r.Stream,
			OccurredAt: occurred, ReceivedAt: engineapp.FormatTime(now()), CorrelationID: first,
			ProvenanceClass: jc.JournalEntryProvenanceClassPersonal, DomainBuild: w.DomainBuild,
		}
		if i > 0 {
			c := first
			e.CausationID = &c
		}
		rq.Batch = append(rq.Batch, appjournal.Pending{Entry: e, Envelope: sealed})
	}
	res, err := w.Journal.Append(ctx, rq)
	if errors.Is(err, appjournal.ErrDuplicate) {
		return platform.Receipt{CommandID: first, EventIDs: []string{first}, Replayed: true}, nil
	}
	if err != nil {
		if pe, ok := appjournal.Problem(err); ok {
			return platform.Receipt{}, pe
		}
		return platform.Receipt{}, err
	}
	rc := platform.Receipt{CommandID: first, EventIDs: ids, RecordedAt: res.Committed}
	if len(res.Seqs) > 0 {
		rc.Seq = res.Seqs[len(res.Seqs)-1]
	}
	return rc, nil
}

// signer — подписант конверта решения: key_ref `‹псевдоним›@1` (нижний регистр).
func signer(actor string) string {
	a := strings.ToLower(actor)
	if a == "" {
		a = "anonymous"
	}
	return a + "@1"
}
