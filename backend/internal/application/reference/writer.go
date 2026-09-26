package reference

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
	dom "ant/internal/domain/reference"
)

// Источники записей справочника (source_id).
const (
	// SourceAPI — решения администратора данных, метролога, мастера через API.
	SourceAPI = "ant-api"
	// SourceGenesis — затравка справочников генезисом (AD-33).
	SourceGenesis = "ant-genesis"
)

// Record — запись справочника к записи в журнал.
type Record struct {
	// EventID — event_id (команда: = command_id, AD-7; затравка — UUIDv5).
	EventID string
	Type    catalog.Type
	// Stream — `reference:‹вид›:‹id›` (каталог, streams.reference).
	Stream     string
	RunID      string
	OccurredAt time.Time
	Data       any
	// Actor — подписант решения (псевдоним); у затравки — genesis.
	Actor string
	// Provenance — класс происхождения (AD-2): решения — personal, затравка —
	// genesis. Справочники, от которых зависят предусловия (поверка, смены,
	// календарь), не меняются записями server-attested.
	Provenance jc.JournalEntryProvenanceClass
	SourceID   string
	// BasisSeq, GuardStreams — проверка AD-39: после basis_seq в потоках гарда
	// нет новых записей guard_relevant (иначе 409 journal.stale_state).
	BasisSeq     int64
	GuardStreams []string
	PolicySeq    int64
	WorkplaceID  string
}

// Writer — ведомый порт записи справочника в журнал (через journal.Append, AD-44).
type Writer interface {
	Write(ctx context.Context, r Record) (platform.Receipt, error)
}

// JournalWriter — Writer над журналом. Конверт — DSSE без подписей (демо
// без агента токена, Д-30); подписант конверта — `‹псевдоним›@1` до реестра
// ключей (эпик 05).
type JournalWriter struct {
	Journal     appjournal.JournalStore
	DomainBuild string
	// Now — InfraClock для received_at (AD-37); nil — time.Now.
	Now func() time.Time
}

var _ Writer = JournalWriter{}

// ErrDuplicate — запись с тем же event_id уже есть (повтор затравки или команды).
var ErrDuplicate = appjournal.ErrDuplicate

// Write записывает запись. Повтор с тем же event_id возвращает прежнюю
// квитанцию (AD-7).
func (w JournalWriter) Write(ctx context.Context, r Record) (platform.Receipt, error) {
	id := strings.ToLower(r.EventID)
	if _, err := uuid.Parse(id); err != nil || id == "" {
		id = uuid.NewV7().String()
	}
	info, ok := catalog.Lookup(r.Type)
	if !ok || info.Emitter != string(dom.Module) {
		return platform.Receipt{}, errors.New("reference: запись чужого типа: " + string(r.Type))
	}
	data, err := json.Marshal(r.Data)
	if err != nil {
		return platform.Receipt{}, err
	}
	occurred := engineapp.FormatTime(r.OccurredAt)
	guard := r.GuardStreams
	if guard == nil {
		guard = []string{}
	}
	cmd := map[string]any{"command_id": id, "basis_seq": r.BasisSeq, "guard_streams": guard, "policy_seq": r.PolicySeq, "signature_level": 0}
	if r.WorkplaceID != "" {
		cmd["workplace_id"] = r.WorkplaceID
	}
	source := r.SourceID
	if source == "" {
		source = SourceAPI
	}
	env := map[string]any{
		"event_id": id, "event_type": string(r.Type), "schema_version": info.CurrentVersion, "source_id": source,
		"occurred_at": occurred, "correlation_id": id, "causation_id": nil, "command": cmd,
		"integrity": map[string]any{"format_version": 1, "crypto_profile": "gost", "signers": []string{signer(r.Actor)}},
		"data":      json.RawMessage(data),
	}
	if r.RunID != "" {
		env["run_id"] = r.RunID
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
	prov := r.Provenance
	if prov == "" {
		prov = jc.JournalEntryProvenanceClassPersonal
	}
	kind := jc.JournalEntryEntryKind(info.Kind)
	e := jc.JournalEntry{
		Chain: jc.JournalEntryChainMain, EntryKind: kind, EventType: string(r.Type),
		SchemaVersion: info.CurrentVersion, EventID: id, SourceID: source, Stream: r.Stream,
		OccurredAt: occurred, ReceivedAt: engineapp.FormatTime(now()), CorrelationID: id,
		ProvenanceClass: prov, DomainBuild: w.DomainBuild,
	}
	if r.RunID != "" {
		run := r.RunID
		e.RunID = &run
	}
	if r.BasisSeq > 0 {
		b := int(r.BasisSeq)
		e.BasisSeq = &b
	}
	rq := appjournal.AppendRequest{Batch: []appjournal.Pending{{Entry: e, Envelope: sealed}}}
	if r.BasisSeq > 0 {
		for _, s := range guard {
			rq.Checks = append(rq.Checks, appjournal.Check{Stream: s, BasisSeq: r.BasisSeq})
		}
	}
	res, err := w.Journal.Append(ctx, rq)
	if errors.Is(err, appjournal.ErrDuplicate) {
		return w.replay(ctx, r, id)
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

// replay — квитанция уже записанной записи (повтор команды, AD-7).
func (w JournalWriter) replay(ctx context.Context, r Record, id string) (platform.Receipt, error) {
	var after int64
	for {
		es, err := w.Journal.Read(ctx, appjournal.ReadQuery{Stream: r.Stream, EventType: string(r.Type), AfterSeq: after, Limit: readLimit})
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
		if len(es) < readLimit {
			return platform.Receipt{}, errors.New("reference: повтор " + id + ": запись не найдена")
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
