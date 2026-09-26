// Пакет nctest — опоры тестов модуля nonconformity (слой application): записи
// фактов и адресованных записей в журнал, свёртка с моделью модуля quality
// (черновик несоответствия по результату контроля — функция-намерение
// nonconformity.Draft) и сценарии, общие для журнала в памяти и своей БД
// (make dev-db). Используется только из тестов.
package nctest

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"sync/atomic"
	"time"

	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	jc "ant/internal/contracts/journal"
	"ant/internal/domain/engine"
	dj "ant/internal/domain/journal"
	"ant/internal/domain/kernel"
	nc "ant/internal/domain/nonconformity"
	"ant/internal/domain/quality"
)

// T0 — начало сценариев.
var T0 = time.Date(2026, 9, 28, 7, 0, 0, 0, time.UTC)

var n atomic.Int64

// ID — уникальный UUID вида v7 для тестовых записей.
func ID() string {
	k := n.Add(1)
	return fmt.Sprintf("0192f000-0000-7000-8000-%012d", k)
}

// Record — запись журнала (факт или адресованная запись) с конвертом DSSE.
func Record(t catalog.Type, itemID string, at time.Time, data any) appjournal.Pending {
	return RecordP(t, itemID, at, data, Partitions)
}

// Partitions — число партиций P записей Record.
var Partitions = 1

// RecordP — Record с числом партиций p.
func RecordP(t catalog.Type, itemID string, at time.Time, data any, p int) appjournal.Pending {
	info, _ := catalog.Lookup(t)
	id := ID()
	b, _ := json.Marshal(data)
	env, _ := json.Marshal(map[string]any{"event_id": id, "event_type": string(t), "schema_version": 1, "source_id": "test-source",
		"source_kind": "machine", "occurred_at": dj.FormatTime(at), "correlation_id": id, "item_id": itemID, "data": json.RawMessage(b),
		"integrity": map[string]any{"format_version": 1, "crypto_profile": "gost", "signers": []string{"test@1"}}})
	sealed, _ := json.Marshal(map[string]any{"payloadType": engineapp.PayloadTypeEvent, "payload": base64.StdEncoding.EncodeToString(env), "signatures": []any{}})
	kind := jc.JournalEntryEntryKind(info.Kind)
	prov := jc.JournalEntryProvenanceClassDevice
	if info.Kind != catalog.KindFact {
		prov = jc.JournalEntryProvenanceClassServerAttested
	}
	e := jc.JournalEntry{Chain: jc.JournalEntryChainMain, EntryKind: kind, EventType: string(t), SchemaVersion: 1, EventID: id,
		SourceID: "test-source", Stream: "item:" + itemID, OccurredAt: dj.FormatTime(at), ReceivedAt: dj.FormatTime(at),
		CorrelationID: id, ProvenanceClass: prov, DomainBuild: dj.ZeroLink.String()}
	if itemID != "" {
		it := itemID
		e.ItemID = &it
		e.Partition = kernel.PartitionOf(itemID, p)
	}
	return appjournal.Pending{Entry: e, Envelope: sealed}
}

// Inspection — результат контроля изделия с признаком дефекта.
func Inspection(itemID string, at time.Time, zone string) appjournal.Pending {
	return Record(catalog.InspectionResultRecorded, itemID, at, map[string]any{"method": "camera", "phase": "after_operation",
		"outcome": "defect_indicated", "processing_state": "completed", "step_key": "welding.weld", "zone_ids": []string{zone},
		"analyzer_confidence_bp": 8700, "observation_quality_bp": 9100,
		"stages": []map[string]any{{"stage": "locate", "version": "1.2.0", "confidence_bp": 9000}}})
}

// Run — начало выполнения операции изделия исполнителем operator.
func Run(itemID string, at time.Time, runID, operator string) appjournal.Pending {
	return Record(catalog.OperationRunStarted, itemID, at, map[string]any{"operation_run_id": runID, "operation_code": "020",
		"step_key": "welding.weld", "equipment_id": "WELD-1", "operator_id": operator})
}

// Registered — регистрация несоответствия окна нарушения специального
// процесса (адресованная запись стадии, FR-151).
func Registered(itemID string, at time.Time, runID string) (appjournal.Pending, string) {
	window := ID()
	ncID := nc.WindowNCID(window, runID)
	return Record(catalog.DecisionNonconformityRegistered, itemID, at, nc.RegisteredData{NCID: ncID, ViolationWindowEventID: window,
		OperationRunID: runID, StepKey: "welding.weld"}), ncID
}

// SignalOf — id сигнала результата контроля (модель quality).
func SignalOf(eventID string) string { return "SIG-" + eventID[len(eventID)-6:] }

// DraftingFold — свёртка движка с моделью модуля quality (эпик 20
// параллельно): результат контроля с признаком дефекта → намерение
// «черновик несоответствия» (реакция карты — «ручной осмотр»). Намерение
// применяется до React nonconformity того же шага — так, как это должен
// делать движок для намерений к поздним модулям.
func DraftingFold(b engine.Bundle, input []kernel.Record) (engine.Snapshot, []kernel.Reaction) {
	in := engine.SortInput(input)
	var s engine.Snapshot
	bySlot := map[string]kernel.Reaction{}
	for _, r := range in {
		var out kernel.Output
		s, out = engine.Step(s, b, r)
		for _, re := range out.Reactions {
			bySlot[re.Slot.Key()] = re
		}
		if r.Type != catalog.InspectionResultRecorded {
			continue
		}
		var d struct {
			Outcome string   `json:"outcome"`
			ZoneIDs []string `json:"zone_ids"`
		}
		if json.Unmarshal(r.Data, &d) != nil || d.Outcome != "defect_indicated" {
			continue
		}
		sev := ev.Severity("major")
		oc := ev.DecisionNonconformityDraftedV1ReactionOutcomeManualReview
		bk := ev.DecisionNonconformityDraftedV1BasisKindInspectionResult
		sk := ev.StepKey("welding.weld")
		ref := "RM-weld@3#12"
		req := nc.DraftRequest{SignalIds: []ev.ObjectID{ev.ObjectID(SignalOf(r.EventID))}, Severity: &sev, ReactionOutcome: &oc,
			BasisKind: &bk, StepKey: &sk, ReactionMapRef: &ref}
		if len(d.ZoneIDs) > 0 {
			z := ev.ObjectID(d.ZoneIDs[0])
			req.ZoneID = &z
		}
		s.Nonconformity = nc.Apply(s.Nonconformity, kernel.NewIntent(nc.Module, quality.Module, nc.IntentDraft, req, r))
		for _, re := range nc.React(s.Nonconformity, b.Nonconformity, nc.Upstream{}).Reactions {
			bySlot[re.Slot.Key()] = re
		}
	}
	out := make([]kernel.Reaction, 0, len(bySlot))
	for _, k := range slices.Sorted(maps.Keys(bySlot)) {
		out = append(out, bySlot[k])
	}
	return s, out
}
