// Пакет proctest — сценарии изделия по стартовому процессу фланца для тестов
// и прогонов модуля process: записи журнала, какими их пишут приём (факты
// терминала и MES) и решения людей с подписью (FR-19). Используют тесты
// application/process (фейки в памяти) и storage/process (своя БД).
//
// Слой: application (тестовая опора, без ввода-вывода); в сборку ролей не входит.
// Владелец: эпик 17 (процесс).
package proctest

import (
	"context"
	"fmt"
	"time"

	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	jc "ant/internal/contracts/journal"
	"ant/internal/domain/kernel"
)

// Journey — записи изделия в порядке поступления.
type Journey struct {
	Item  string
	Hash  string
	T0    time.Time
	Facts []engineapp.Out
	n     int
}

// New — сценарий изделия item по версии с хешем hash от момента t0.
func New(item, hash string, t0 time.Time) *Journey { return &Journey{Item: item, Hash: hash, T0: t0} }

// At — момент t0 + h часов.
func (j *Journey) At(h float64) time.Time { return j.T0.Add(time.Duration(h * float64(time.Hour))) }

// Add — запись типа tp в h часов от начала.
func (j *Journey) Add(tp catalog.Type, h float64, data map[string]any) string {
	j.n++
	id := kernel.UUIDv5(constants.NsAnt, fmt.Sprintf("proctest\x1f%s\x1f%d", j.Item, j.n))
	info, _ := catalog.Lookup(tp)
	j.Facts = append(j.Facts, engineapp.Out{EventID: id, Type: tp, Kind: info.Kind, Stream: "item:" + j.Item, ItemID: j.Item,
		OccurredAt: j.At(h), Correlation: id, Data: data})
	return id
}

// Register — запуск изделия по заданию 1С (Д-5).
func (j *Journey) Register(h float64) string {
	return j.Add(catalog.ItemItemRegistered, h, map[string]any{"item_id": j.Item, "item_type_id": "FL-100.00.000", "item_revision": "Б",
		"process_version_hash": j.Hash, "normative_rev": "flange-1", "lot_ids": []string{"LOT-FL-1"}})
}

// Decide — решение на точке предъявления (подпись человека).
func (j *Journey) Decide(step, resolution string, h float64) string {
	return j.Add(catalog.DecisionPresentationResolved, h, map[string]any{"step_key": step, "closing_point": "ZT", "resolution": resolution,
		"presentation_no": 1, "method_event_ids": []string{}})
}

// Run — выполнение операции; to = 0 — не завершено.
func (j *Journey) Run(id, step string, from, to float64) {
	j.Add(catalog.OperationRunStarted, from, map[string]any{"operation_run_id": id, "operation_code": "0", "step_key": step, "operator_id": "W-07"})
	if to > 0 {
		j.Add(catalog.OperationRunFinished, to, map[string]any{"operation_run_id": id, "completion": "completed"})
	}
}

// Received — приём изделия на шаге-перемещении.
func (j *Journey) Received(step string, h float64) {
	j.Add(catalog.OperationMovementReceived, h, map[string]any{"to_location_id": "WS", "destination_kind": "workshop",
		"inspection_on_receipt": "no_damage", "received_by": "M-01", "step_key": step})
}

// Inspect — результат контроля на шаге.
func (j *Journey) Inspect(step, outcome string, h float64, zones ...string) {
	j.Add(catalog.InspectionResultRecorded, h, map[string]any{"method": "camera", "phase": "after_operation", "step_key": step,
		"outcome": outcome, "processing_state": "completed", "zone_ids": zones})
}

// Dispose — решение по изделию на ЗТ-Р.
func (j *Journey) Dispose(disposition string, h float64) {
	j.Add(catalog.DecisionDispositionSet, h, map[string]any{"nc_id": "NC-1", "disposition": disposition, "reason": map[string]string{"code": "x", "text": "x"}})
}

// ToWelding — от задания 1С до подготовки кромок (входной контроль, раздача
// по цехам, мехобработка, ЗТ-2, передача в сварочный цех, слияние с патрубком).
func (j *Journey) ToWelding() {
	j.Register(0)
	j.Decide("incoming.zt1_lot_acceptance", "accept", 1)
	j.Run("RUN-M1-1", "machining.cnc", 2, 3)
	j.Decide("machining.zt2_acceptance", "accept", 4)
	j.Received("incoming.issue_pipe", 4.5)
	j.Received("welding.receive", 5)
}

// Append — записи в журнал так, как их пишет приём: факты — происхождение
// device, решения людей — personal (подпись человека, AD-2).
func Append(ctx context.Context, codec *engineapp.Codec, store appjournal.JournalStore, outs []engineapp.Out) error {
	for _, o := range outs {
		p, err := codec.Encode(ctx, o)
		if err != nil {
			return err
		}
		p.Entry.SourceID = "terminal-1"
		p.Entry.ProvenanceClass = jc.JournalEntryProvenanceClassDevice
		if o.Kind == catalog.KindDecision {
			p.Entry.ProvenanceClass = jc.JournalEntryProvenanceClassPersonal
		}
		if _, err := store.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{p}}); err != nil {
			return err
		}
	}
	return nil
}
