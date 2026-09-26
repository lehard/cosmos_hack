package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"ant/cmd/internal/config"
	appingest "ant/internal/application/ingest"
	appjournal "ant/internal/application/journal"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	jc "ant/internal/contracts/journal"
	domingest "ant/internal/domain/ingest"
	dj "ant/internal/domain/journal"
	"ant/internal/domain/kernel"
	journalstore "ant/internal/infrastructure/storage/journal"
	"ant/internal/infrastructure/storage/journal/clock"
)

// Часы процесса (AD-37, эпик 16). Режим часов — свойство журнала: в профиле
// demo журнал ведётся в режиме scenario — доменное «сейчас» задаёт прогон
// сценария записями time.clock.ticked (simulation), все роли читают его из
// журнала с учётом прогона (application/journal.WithRun).

// scenarioClock — в профиле demo журнал в режиме часов scenario.
func scenarioClock(cfg *config.Config) bool { return cfg.Profile == config.ProfileDemo }

// journalClock — DomainClock процесса: часы журнала (ключ domain_clock:
// journal); в режиме scenario, пока у прогона (или журнала) нет ни одного
// тика, — recorded_at головы журнала, журнал пуст — системное время. Так
// столы и команды работают и до первого прогона.
type journalClock struct {
	j     *clock.Journal
	store *journalstore.Store
}

var _ appjournal.DomainClock = journalClock{}

// Now — доменное «сейчас» (AD-37).
func (c journalClock) Now(ctx context.Context) (time.Time, error) {
	t, err := c.j.Now(ctx)
	if !errors.Is(err, clock.ErrNoTick) {
		return t, err
	}
	h, ok, err := c.store.RecordedHead(ctx)
	if err != nil {
		return time.Time{}, err
	}
	if ok {
		return h, nil
	}
	return time.Now().UTC(), nil
}

// domainClock — доменные часы ядра процесса.
func (c *core) domainClock() journalClock {
	return journalClock{j: c.clock, store: c.journal}
}

// clockModeEventID — event_id записи режима часов: один на журнал (повтор —
// дубль, AD-7).
var clockModeEventID = kernel.UUIDv5(constants.NsAnt, "time.clock.mode_set|scenario")

// clockModeSource — source_id записи режима часов (служебная запись ядра до генезиса).
const clockModeSource = "ant-init"

// ensureClockMode — запись time.clock.mode_set {mode: scenario} в профиле
// demo, если режима в журнале ещё нет. TODO(05): режим часов пишет блок
// генезиса `ant init`; до него — здесь, при первом старте ядра.
func (c *core) ensureClockMode(ctx context.Context, env *environment) {
	if !scenarioClock(env.cfg) {
		return
	}
	c.clockMu.Lock()
	defer c.clockMu.Unlock()
	mode, err := c.clock.Mode(ctx)
	if err != nil {
		env.log.Warn("часы: режим журнала не прочитан — жду migrate", "err", err)
		return
	}
	if mode == clock.ModeScenario {
		return
	}
	now := dj.FormatTime(time.Now().UTC())
	payload := map[string]any{"event_id": clockModeEventID, "event_type": string(catalog.TimeClockModeSet), "schema_version": 1,
		"source_id": clockModeSource, "occurred_at": now, "correlation_id": clockModeEventID, "causation_id": nil,
		"integrity": map[string]any{"format_version": 1, "crypto_profile": "gost", "signers": []string{engineKeyRef}},
		"data":      map[string]any{"mode": clock.ModeScenario}}
	raw, err := json.Marshal(payload)
	if err == nil {
		raw, err = domingest.Canonicalize(raw)
	}
	var dsse []byte
	if err == nil {
		dsse, err = json.Marshal(map[string]any{"payloadType": appingest.PayloadTypeEvent,
			"payload": base64.StdEncoding.EncodeToString(raw), "signatures": []any{}})
	}
	if err != nil {
		env.log.Error("часы: запись режима не собрана", "err", err)
		return
	}
	e := jc.JournalEntry{Chain: jc.JournalEntryChainMain, EntryKind: jc.JournalEntryEntryKindService, EventType: string(catalog.TimeClockModeSet),
		SchemaVersion: 1, EventID: clockModeEventID, SourceID: clockModeSource, Stream: "global",
		Partition: env.cfg.Engine.Partitions, OccurredAt: now, ReceivedAt: now, CorrelationID: clockModeEventID,
		ProvenanceClass: jc.JournalEntryProvenanceClassGenesis, DomainBuild: c.codec.DomainBuild}
	_, err = c.journal.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{{Entry: e, Envelope: dsse}}})
	if err != nil && !errors.Is(err, appjournal.ErrDuplicate) {
		env.log.Warn("часы: режим scenario не записан", "err", err)
		return
	}
	env.log.Info("часы: журнал в режиме scenario (профиль demo, до генезиса эпика 05)")
}
