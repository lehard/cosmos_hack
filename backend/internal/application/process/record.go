package process

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
	dp "ant/internal/domain/process"
)

// SourceAPI — source_id решений, принятых операциями API модуля process.
const SourceAPI = "ant-api"

// Recorder — запись решений модуля process (normative.version.*) в журнал
// через journal.Append (AD-44): поток версии `process_version:‹id›`, вид
// «решение», повтор с тем же command_id — прежняя квитанция (AD-7).
// Конверт — DSSE без подписей до агента токена (Д-30; эпики 27, 38).
type Recorder struct {
	Journal     appjournal.JournalStore
	DomainBuild string
	// Partitions — P движка: записи вне изделия — в партиции стадии P (как у приёма).
	Partitions int
	// Now — InfraClock для received_at.
	Now func() time.Time
}

// VersionStream — поток версии процесса (catalog.yaml → streams.process_version).
func VersionStream(versionID string) string { return "process_version:" + versionID }

// Record — решение типа t в поток stream от имени actor.
func (r *Recorder) Record(ctx context.Context, t catalog.Type, stream, actor string, meta platform.CommandMeta, at time.Time, data any) (platform.Receipt, error) {
	info, ok := catalog.Lookup(t)
	if !ok || info.Emitter != string(dp.Module) || info.Kind != catalog.KindDecision {
		return platform.Receipt{}, errors.New("process: решение чужого типа или не решение: " + string(t))
	}
	id := strings.ToLower(meta.CommandID)
	if _, err := uuid.Parse(id); err != nil || id == "" {
		id = uuid.NewV7().String()
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return platform.Receipt{}, err
	}
	signer := "anonymous@1"
	if actor != "" {
		signer = actor + "@1"
	}
	occurred := engineapp.FormatTime(at)
	env := map[string]any{
		"event_id": id, "event_type": string(t), "schema_version": info.CurrentVersion, "source_id": SourceAPI,
		"source_kind": "manual_entry", "occurred_at": occurred, "correlation_id": id, "causation_id": nil,
		"command":   map[string]any{"command_id": id, "basis_seq": meta.BasisSeq, "guard_streams": []string{stream}, "policy_seq": meta.PolicySeq},
		"integrity": map[string]any{"format_version": 1, "crypto_profile": "gost", "signers": []string{signer}},
		"data":      json.RawMessage(raw),
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
	if r.Now != nil {
		now = r.Now
	}
	e := jc.JournalEntry{
		Chain: jc.JournalEntryChainMain, EntryKind: jc.JournalEntryEntryKindDecision, EventType: string(t),
		SchemaVersion: info.CurrentVersion, EventID: id, SourceID: SourceAPI, Stream: stream, Partition: r.Partitions,
		OccurredAt: occurred, ReceivedAt: engineapp.FormatTime(now()), CorrelationID: id,
		ProvenanceClass: jc.JournalEntryProvenanceClassPersonal, DomainBuild: r.DomainBuild,
	}
	if meta.BasisSeq > 0 {
		b := int(meta.BasisSeq)
		e.BasisSeq = &b
	}
	res, err := r.Journal.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{{Entry: e, Envelope: sealed}}})
	if errors.Is(err, appjournal.ErrDuplicate) {
		return platform.Receipt{CommandID: id, EventIDs: []string{id}, Replayed: true}, nil
	}
	if err != nil {
		if pe, ok := appjournal.Problem(err); ok {
			return platform.Receipt{}, pe
		}
		return platform.Receipt{}, err
	}
	rc := platform.Receipt{CommandID: id, EventIDs: []string{id}, RecordedAt: at}
	if len(res.Seqs) > 0 {
		rc.Seq = res.Seqs[0]
	}
	return rc, nil
}
