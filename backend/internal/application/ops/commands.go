package ops

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"uuid"

	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	ev "ant/internal/contracts/events"
	jc "ant/internal/contracts/journal"
	"ant/internal/domain/engine"
	"ant/internal/domain/kernel"
	dom "ant/internal/domain/ops"
)

// SourceAPI — source_id решений, принятых операциями API.
const SourceAPI = "ant-api"

// RetryProcessing — повторить обработку изделия (ops.processing.retry,
// AD-45): решение администратора ops.processing.retried в потоке изделия.
// Запись — триггер свёртки: воркер снимает «обработка остановлена» и
// сворачивает изделие заново (как `ant rebuild --item`). Изделие не
// остановлено — 404; failure_event_id не последний сбой — 409 journal.stale_state.
func (s *Service) RetryProcessing(ctx context.Context, itemID string, in RetryProcessing) (platform.Receipt, error) {
	if !s.live {
		return Unimplemented{}.RetryProcessing(ctx, itemID, in)
	}
	meta := in.CommandMeta()
	if rc, ok, err := s.replay(ctx, meta.CommandID); ok || err != nil {
		return rc, err
	}
	failures, retries, err := failuresAndRetries(ctx, s.cfg.Journal, s.cfg.Codec)
	if err != nil {
		return platform.Receipt{}, err
	}
	var stopped *dom.Stopped
	for _, st := range dom.StoppedItems(failures, retries) {
		if st.Failure.ItemID == itemID {
			stopped = &st
		}
	}
	if stopped == nil {
		e := platform.Fail(errcodes.ApiNotFound, "object", "Остановленное изделие", "id", itemID)
		e.Detail = "Обработка изделия " + itemID + " не остановлена — повторять нечего"
		return platform.Receipt{}, e
	}
	if in.FailureEventID != "" && !strings.EqualFold(in.FailureEventID, stopped.Failure.EventID) {
		e := platform.Fail(errcodes.JournalStaleState, "stream", "item:"+itemID, "basis_seq", strings.TrimSpace(in.FailureEventID))
		e.Detail = "Последний сбой изделия — " + stopped.Failure.EventID + "; обновите список и повторите"
		e.BasisSeq = stopped.Failure.Seq
		return platform.Receipt{}, e
	}
	data := ev.OpsProcessingRetriedV1{FailureEventID: ev.UUID(stopped.Failure.EventID)}
	if r := in.Reason; r != nil && strings.TrimSpace(r.Text) != "" {
		data.Reason = &ev.Reason{Text: ev.Text(r.Text)}
		if r.Code != nil && *r.Code != "" {
			c := ev.Code(*r.Code)
			data.Reason.Code = &c
		}
	}
	return s.decide(ctx, decision{Type: catalog.OpsProcessingRetried, Stream: "item:" + itemID, ItemID: itemID,
		Causation: stopped.Failure.EventID, Data: data, Meta: meta})
}

// DisableSource — отключить источник событий (ops.source.disable, AD-28):
// критическое защитное действие администратора; запись CA делает journal.Append.
func (s *Service) DisableSource(ctx context.Context, sourceID string, in SwitchSource) (platform.Receipt, error) {
	if !s.live {
		return Unimplemented{}.DisableSource(ctx, sourceID, in)
	}
	return s.switchSource(ctx, catalog.OpsSourceDisabled, sourceID, in)
}

// EnableSource — включить источник (ops.source.enable, AD-28): критическое
// разрешающее действие администратора.
func (s *Service) EnableSource(ctx context.Context, sourceID string, in SwitchSource) (platform.Receipt, error) {
	if !s.live {
		return Unimplemented{}.EnableSource(ctx, sourceID, in)
	}
	return s.switchSource(ctx, catalog.OpsSourceEnabled, sourceID, in)
}

func (s *Service) switchSource(ctx context.Context, t catalog.Type, sourceID string, in SwitchSource) (platform.Receipt, error) {
	meta := in.CommandMeta()
	reason := ev.Reason{Text: ev.Text(strings.TrimSpace(in.Reason.Text))}
	if in.Reason.Code != nil && *in.Reason.Code != "" {
		c := ev.Code(*in.Reason.Code)
		reason.Code = &c
	}
	var data any = ev.OpsSourceDisabledV1{SourceID: ev.SourceID(sourceID), Reason: reason}
	if t == catalog.OpsSourceEnabled {
		data = ev.OpsSourceEnabledV1{SourceID: ev.SourceID(sourceID), Reason: reason}
	}
	return s.decide(ctx, decision{Type: t, Stream: "source:" + sourceID, Data: data, Meta: meta})
}

// decision — решение администратора (AD-2: вид «решение»).
type decision struct {
	Type      catalog.Type
	Stream    string
	ItemID    string
	Causation string
	Data      any
	Meta      platform.CommandMeta
}

// decide записывает решение одной записью через journal.Append: конверт DSSE
// без подписей (демо без агента токена, Д-30), подписант `‹псевдоним›@1`,
// event_id = command_id; повтор с тем же command_id возвращает прежнюю
// квитанцию (AD-7); basis_seq — проверка потока гарда (AD-39).
func (s *Service) decide(ctx context.Context, d decision) (platform.Receipt, error) {
	info, ok := catalog.Lookup(d.Type)
	if !ok || info.Emitter != "ops" || info.Kind != catalog.KindDecision {
		return platform.Receipt{}, errors.New("ops: решение чужого типа или не решение: " + string(d.Type))
	}
	id := strings.ToLower(d.Meta.CommandID)
	if _, err := uuid.Parse(id); err != nil || id == "" {
		id = uuid.NewV7().String()
	}
	data, err := json.Marshal(d.Data)
	if err != nil {
		return platform.Receipt{}, err
	}
	actor := strings.ToLower(platform.PrincipalFrom(ctx).PersonID)
	if actor == "" {
		actor = "anonymous"
	}
	occurred := engineapp.FormatTime(s.domainNow(ctx))
	run := appjournal.RunFrom(ctx)
	env := map[string]any{
		"event_id": id, "event_type": string(d.Type), "schema_version": info.CurrentVersion, "source_id": SourceAPI,
		"occurred_at": occurred, "correlation_id": id, "causation_id": nil,
		"command":   map[string]any{"command_id": id, "basis_seq": d.Meta.BasisSeq, "guard_streams": []string{d.Stream}, "policy_seq": d.Meta.PolicySeq},
		"integrity": map[string]any{"format_version": 1, "crypto_profile": "gost", "signers": []string{actor + "@1"}},
		"data":      json.RawMessage(data),
	}
	if d.Causation != "" {
		env["causation_id"] = d.Causation
	}
	if d.ItemID != "" {
		env["item_id"] = d.ItemID
	}
	if run != "" {
		env["run_id"] = run
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
		Chain: jc.JournalEntryChainMain, EntryKind: jc.JournalEntryEntryKindDecision, EventType: string(d.Type),
		SchemaVersion: info.CurrentVersion, EventID: id, SourceID: SourceAPI, Stream: d.Stream,
		OccurredAt: occurred, ReceivedAt: engineapp.FormatTime(s.cfg.Now()), CorrelationID: id,
		ProvenanceClass: jc.JournalEntryProvenanceClassPersonal, DomainBuild: s.cfg.DomainBuild,
	}
	if d.Causation != "" {
		c := d.Causation
		e.CausationID = &c
	}
	if d.ItemID != "" {
		item := d.ItemID
		e.ItemID = &item
		// Партиция изделия (AD-6, AD-41): запись — триггер свёртки воркера.
		e.Partition = kernel.PartitionOf(d.ItemID, s.cfg.Partitions)
	}
	if run != "" {
		e.RunID = &run
	}
	rq := appjournal.AppendRequest{Batch: []appjournal.Pending{{Entry: e, Envelope: sealed}}}
	if d.Meta.BasisSeq > 0 {
		b := int(d.Meta.BasisSeq)
		e.BasisSeq = &b
		rq.Batch[0].Entry = e
		rq.Checks = []appjournal.Check{{Stream: d.Stream, BasisSeq: d.Meta.BasisSeq}}
	}
	res, err := s.cfg.Journal.Append(ctx, rq)
	if errors.Is(err, appjournal.ErrDuplicate) {
		if rc, ok, rerr := s.replay(ctx, id); ok || rerr != nil {
			return rc, rerr
		}
		return platform.Receipt{CommandID: id, EventIDs: []string{id}, Replayed: true}, nil
	}
	if err != nil {
		if pe, ok := appjournal.Problem(err); ok {
			return platform.Receipt{}, pe
		}
		return platform.Receipt{}, err
	}
	s.cfg.Log.Info("ops: решение записано", "event_id", id, "correlation_id", id, "item_id", d.ItemID, "run_id", run,
		"event_type", string(d.Type))
	rc := platform.Receipt{CommandID: id, EventIDs: []string{id}, RecordedAt: res.Committed}
	if len(res.Seqs) > 0 {
		rc.Seq = res.Seqs[0]
	}
	if len(res.CARefs) > 0 {
		rc.CARef = res.CARefs[0]
	}
	return rc, nil
}

// replay — прежняя квитанция команды с тем же command_id (AD-7): решение
// ops уже записано с event_id = command_id.
func (s *Service) replay(ctx context.Context, commandID string) (platform.Receipt, bool, error) {
	id := strings.ToLower(commandID)
	if _, err := uuid.Parse(id); err != nil || id == "" {
		return platform.Receipt{}, false, nil
	}
	for _, t := range []catalog.Type{catalog.OpsProcessingRetried, catalog.OpsSourceDisabled, catalog.OpsSourceEnabled, catalog.OpsIntegrationStateSet} {
		ds, err := s.cfg.Journal.Read(ctx, appjournal.ReadQuery{EventType: string(t), Backward: true, Limit: 200})
		if err != nil {
			return platform.Receipt{}, false, err
		}
		for _, e := range ds {
			if e.EventID == id {
				rc := platform.Receipt{CommandID: id, Seq: int64(e.Seq), EventIDs: []string{id}, Replayed: true}
				rc.RecordedAt, _ = parseTime(e.CommittedAt)
				return rc, true, nil
			}
		}
	}
	return platform.Receipt{}, false, nil
}
