package process_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	app "ant/internal/application/process"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	"ant/internal/domain/engine"
	"ant/internal/domain/kernel"
	dp "ant/internal/domain/process"
)

// Стартовый процесс фланца (normative/process/flange-process.bpmn, 106 узлов)
// через полную композицию движка (domain/engine.Fold) с нормативным слоем от
// Bundles — как у воркера.

const flangePath = "../../../../normative/process/flange-process.bpmn"

var t0 = time.Date(2026, 9, 22, 6, 0, 0, 0, time.UTC)

func at(h float64) time.Time { return t0.Add(time.Duration(h * float64(time.Hour))) }

func flangeXML(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile(flangePath)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

type journey struct {
	t       *testing.T
	store   *app.MemVersions
	bundles *app.Bundles
	item    string
	hash    string
	in      []kernel.Record
	seq     int64
}

func newJourney(t *testing.T) *journey {
	t.Helper()
	xml := flangeXML(t)
	store := &app.MemVersions{}
	seed, err := app.EnsureSeed(context.Background(), store, xml, t0.Add(-24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	// Кэш проверенной версии выключен: каждая свёртка перечитывает байты.
	b := &app.Bundles{Store: store, TTL: time.Nanosecond}
	return &journey{t: t, store: store, bundles: b, item: "ent01:FL-0001", hash: seed.Hash}
}

func (j *journey) add(tp catalog.Type, h float64, data map[string]any) kernel.Record {
	j.seq++
	info, _ := catalog.Lookup(tp)
	prov := "device"
	if info.Kind == catalog.KindDecision {
		prov = "personal"
	}
	b, err := json.Marshal(data)
	if err != nil {
		j.t.Fatal(err)
	}
	r := kernel.Record{Seq: j.seq, EventID: fmt.Sprintf("00000000-0000-7000-8000-%012d", j.seq), Type: tp, Kind: info.Kind, Provenance: prov,
		ItemID: j.item, Stream: "item:" + j.item, OccurredAt: at(h), ReceivedAt: at(h), RecordedAt: at(h), Data: b}
	j.in = append(j.in, r)
	return r
}

func (j *journey) fold() (engine.Snapshot, []kernel.Reaction, engine.Bundle) {
	j.t.Helper()
	b, _, err := j.bundles.Bundle(context.Background(), j.item, j.in)
	if err != nil {
		j.t.Fatal(err)
	}
	s, rs := engine.Fold(b, j.in)
	return s, rs, b
}

func (j *journey) steps() []string {
	s, _, _ := j.fold()
	out := s.Process.Steps()
	slices.Sort(out)
	return out
}

func (j *journey) want(want ...string) {
	j.t.Helper()
	slices.Sort(want)
	if got := j.steps(); !slices.Equal(got, want) {
		s, _, _ := j.fold()
		j.t.Fatalf("изделие на %v, ожидалось %v; отказы %+v; ошибки %v", got, want, s.Process.Refusals, s.Process.Errors)
	}
}

func (j *journey) register(h float64) {
	j.add(catalog.ItemItemRegistered, h, map[string]any{"item_id": j.item, "item_type_id": "FL-100.00.000", "item_revision": "Б",
		"process_version_hash": j.hash, "normative_rev": "flange-1", "lot_ids": []string{"LOT-FL-1", "LOT-PIPE-1"}})
}

func (j *journey) decide(step, res string, h float64) kernel.Record {
	return j.add(catalog.DecisionPresentationResolved, h, map[string]any{"step_key": step, "closing_point": "ZT", "resolution": res,
		"presentation_no": 1, "method_event_ids": []string{}})
}

func (j *journey) run(id, step string, from, to float64) {
	j.add(catalog.OperationRunStarted, from, map[string]any{"operation_run_id": id, "operation_code": "0", "step_key": step, "operator_id": "W-07"})
	if to > 0 {
		j.add(catalog.OperationRunFinished, to, map[string]any{"operation_run_id": id, "completion": "completed"})
	}
}

func (j *journey) received(step string, h float64) {
	j.add(catalog.OperationMovementReceived, h, map[string]any{"to_location_id": "WS", "destination_kind": "workshop",
		"inspection_on_receipt": "no_damage", "received_by": "M-01", "step_key": step})
}

func (j *journey) inspect(step, outcome string, h float64, zones ...string) {
	j.add(catalog.InspectionResultRecorded, h, map[string]any{"method": "camera", "phase": "after_operation", "step_key": step,
		"outcome": outcome, "processing_state": "completed", "zone_ids": zones})
}

func (j *journey) dispose(d string, h float64) {
	j.add(catalog.DecisionDispositionSet, h, map[string]any{"nc_id": "NC-1", "disposition": d, "reason": map[string]string{"code": "x", "text": "x"}})
}

// toWelding — от задания 1С до подготовки кромок: входной контроль (ЗТ-1),
// маркировка, раздача по цехам, мехобработка, ЗТ-2, передача и приёмка в
// сварочном цехе, слияние фланца с патрубком.
func (j *journey) toWelding() {
	j.register(0)
	j.want("incoming.lot_registration")
	j.decide("incoming.zt1_lot_acceptance", dp.ResolutionAccept, 1)
	j.want("incoming.marking")
	j.run("RUN-M1-1", "machining.cnc", 2, 3) // догон через раздачу по цехам
	j.want("incoming.issue_assembly_parts", "incoming.issue_pipe", "machining.kt2_camera")
	j.decide("machining.zt2_acceptance", dp.ResolutionAccept, 4)
	j.received("incoming.issue_pipe", 4.5)
	j.received("welding.receive", 5)
	j.want("incoming.issue_assembly_parts", "welding.edge_prep")
}

// FR-11: стартовый процесс загружается и исполняется; старт по сообщению из
// 1С, параллельная раздача и слияние, сообщения «в 1С».
func TestFlangeRunsToWelding(t *testing.T) {
	j := newJourney(t)
	j.toWelding()
	s, rs, b := j.fold()
	if got := b.Machinelogs.SpecialSteps; len(got) != 1 || got[0] != "welding.weld" {
		t.Fatalf("признак «специальный процесс» не передан machinelogs: %v", got)
	}
	var actions []string
	for _, r := range rs {
		if r.Type == catalog.OperationMessageThrown {
			var d struct {
				Step   string `json:"step_key"`
				Action string `json:"erp_action"`
			}
			raw, _ := json.Marshal(r.Data)
			_ = json.Unmarshal(raw, &d)
			actions = append(actions, d.Step+":"+d.Action)
		}
	}
	slices.Sort(actions)
	want := []string{"incoming.erp_accept_blank:accept_into_work", "incoming.erp_accept_pipe:accept_into_work", "welding.erp_transfer_in:warehouse_transfer"}
	if !slices.Equal(actions, want) {
		t.Fatalf("сообщения «в 1С»: %v", actions)
	}
	if !slices.ContainsFunc(s.Process.Gaps, func(g dp.Gap) bool { return g.StepKey == "incoming.marking" }) {
		t.Fatalf("маркировка без данных не помечена: %+v", s.Process.Gaps)
	}
}

// Готово-1: истечение окна «кромки → сварка» уводит на повторную подготовку.
func TestFlangeEdgeWindowExpired(t *testing.T) {
	j := newJourney(t)
	j.toWelding()
	j.run("RUN-W1-1", "welding.edge_prep", 6, 7)
	j.want("incoming.issue_assembly_parts", "welding.weld")
	j.run("RUN-W2-1", "welding.weld", 15.5, 0) // 8,5 ч после кромок
	s, rs, _ := j.fold()
	if tk := s.Process.TokenAt("welding.edge_prep"); tk == nil {
		t.Fatalf("изделие не вернулось на подготовку кромок: %v", s.Process.Steps())
	}
	found := false
	for _, r := range rs {
		raw, _ := json.Marshal(r.Data)
		if r.Type == catalog.OperationPreconditionFailed && strings.Contains(string(raw), `"time_window"`) {
			found = true
		}
	}
	if !found {
		t.Fatal("нет реакции нарушения окна")
	}
}

// Готово-2: подпроцесс брака из разных мест возвращает в точку вызова.
func TestFlangeNonconformityReturnsToCallPoint(t *testing.T) {
	j := newJourney(t)
	j.register(0)
	j.decide("incoming.zt1_lot_acceptance", dp.ResolutionAccept, 1)
	j.run("RUN-M1-1", "machining.cnc", 2, 3)
	j.decide("machining.zt2_acceptance", dp.ResolutionReject, 4) // вызов из мехобработки
	j.want("incoming.issue_assembly_parts", "incoming.issue_pipe", "nc.isolation")
	j.dispose("rework", 5)
	j.want("incoming.issue_assembly_parts", "incoming.issue_pipe", "machining.cnc") // возврат: переделка ≤ 1
	j.run("RUN-M1-2", "machining.cnc", 6, 7)
	j.decide("machining.zt2_acceptance", dp.ResolutionAccept, 8)
	j.received("incoming.issue_pipe", 8.5)
	j.received("welding.receive", 9)
	j.run("RUN-W1-1", "welding.edge_prep", 10, 11)
	j.run("RUN-W2-1", "welding.weld", 12, 13)
	j.inspect("welding.kt3_camera", "defect_indicated", 14, "W-1.U3")
	j.inspect("welding.kt3_radiography", "no_defect_indicated", 15, "W-1.U3")
	j.decide("welding.zt3_acceptance", dp.ResolutionReject, 16) // вызов из сварки
	j.want("incoming.issue_assembly_parts", "nc.isolation")
	j.dispose("repair", 17)
	// Ремонт через разрешение на отклонение: точка разрешения ждёт решения.
	j.want("incoming.issue_assembly_parts", "nc.concession")
	j.add(catalog.DecisionDispositionSet, 18, map[string]any{"nc_id": "NC-2", "disposition": "repair", "concession_id": "CON-1",
		"reason": map[string]string{"code": "x", "text": "x"}})
	j.want("incoming.issue_assembly_parts", "welding.edge_prep") // возврат в сварку, не в мехобработку
	s, _, _ := j.fold()
	if tk := s.Process.TokenAt("welding.edge_prep"); len(tk.Stack) != 0 || tk.Vars[dp.VarNCOutcome].S != "rework_or_repair" {
		t.Fatalf("токен после возврата: %+v", tk)
	}
}

// Готово-3: четвёртая доработка зоны при лимите 3 без разрешения блокируется.
func TestFlangeFourthReworkBlocked(t *testing.T) {
	j := newJourney(t)
	j.toWelding()
	h := 6.0
	for i := 1; i <= 4; i++ {
		j.run(fmt.Sprintf("RUN-W1-%d", i), "welding.edge_prep", h, h+0.5)
		j.run(fmt.Sprintf("RUN-W2-%d", i), "welding.weld", h+1, h+1.5)
		j.inspect("welding.kt3_camera", "defect_indicated", h+2, "W-1.U3")
		j.inspect("welding.kt3_radiography", "no_defect_indicated", h+2.5, "W-1.U3")
		j.decide("welding.zt3_acceptance", dp.ResolutionReject, h+3)
		j.dispose("rework", h+3.5)
		h += 4
	}
	// Кромки — петля с лимитом 3: 5-е выполнение сверх лимита, по разрешению.
	j.add(catalog.DecisionReworkLimitWaived, h-0.1, map[string]any{"zone_id": "welding.edge_prep", "used": 4, "limit": 3, "extra_allowed": 1,
		"reason": map[string]string{"code": "x", "text": "x"}})
	j.run("RUN-W1-5", "welding.edge_prep", h, h+0.5)
	s, _, b := j.fold()
	if s.Process.TokenAt("welding.weld") == nil {
		t.Fatalf("изделие не на сварке: %v", s.Process.Steps())
	}
	err := engine.Guard(s, b, kernel.Command{Action: "process.operation.start", OccurredAt: at(h + 1),
		Payload: dp.StartCommand{StepKey: "welding.weld", RunID: "RUN-W2-5", OperatorID: "W-07"}})
	var r *kernel.Refusal
	if !errors.As(err, &r) || r.Code != errcodes.ProcessReworkLimitExceeded || r.Params["zone"] != "W-1.U3" || r.Params["used"] != "4" || r.Params["limit"] != "3" {
		t.Fatalf("4-я доработка зоны W-1.U3 не заблокирована гардом: %v", err)
	}
}

// Готово-4: нельзя пройти точку предъявления без подписи.
func TestFlangeGateRequiresSignature(t *testing.T) {
	j := newJourney(t)
	j.register(0)
	j.decide("incoming.zt1_lot_acceptance", dp.ResolutionAccept, 1)
	j.run("RUN-M1-1", "machining.cnc", 2, 3)
	j.inspect("machining.kt2_camera", "no_defect_indicated", 3.5, "F-FACE")
	j.inspect("machining.kt2_cmm", "no_defect_indicated", 3.7, "F-FACE")
	unsigned := j.decide("machining.zt2_acceptance", dp.ResolutionAccept, 4)
	j.in[len(j.in)-1].Provenance = "server_attested"
	j.run("RUN-M5-1", "machining.send_to_welding", 5, 6)
	j.received("welding.receive", 7)
	j.want("incoming.issue_assembly_parts", "incoming.issue_pipe", "machining.zt2_acceptance")
	s, _, b := j.fold()
	err := engine.Guard(s, b, kernel.Command{Action: "process.operation.start", OccurredAt: at(8), Payload: dp.StartCommand{StepKey: "welding.edge_prep"}})
	var r *kernel.Refusal
	if !errors.As(err, &r) || r.Code != errcodes.NonconformityGateWithoutSignature || r.Params["role"] != "qc_acceptance" {
		t.Fatalf("гард пропустил изделие за ЗТ-2 без подписи: %v", err)
	}
	if !slices.ContainsFunc(s.Process.Refusals, func(x dp.Refusal) bool { return x.EventID == unsigned.EventID }) {
		t.Fatalf("попытка без подписи не записана: %+v", s.Process.Refusals)
	}
}

// Готово-5: изменённый в обход системы XML действующей версии не исполняется.
func TestFlangeTamperedXMLNotExecuted(t *testing.T) {
	j := newJourney(t)
	j.register(0)
	j.want("incoming.lot_registration")
	xml := flangeXML(t)
	tampered := []byte(strings.Replace(string(xml), `reworkLimit="3" reworkLimitScope="zone"`, `reworkLimit="99" reworkLimitScope="zone"`, 1))
	if string(tampered) == string(xml) {
		t.Fatal("подделка не изменила XML")
	}
	j.store.Tamper(app.SeedVersionID, tampered)
	j.decide("incoming.zt1_lot_acceptance", dp.ResolutionAccept, 1)
	s, rs, b := j.fold()
	if b.Process.Refusal == nil || b.Process.Refusal.Code != errcodes.ProcessVersionTampered {
		t.Fatalf("изменённая версия не отклонена: %+v", b.Process.Refusal)
	}
	if len(s.Process.Tokens) != 0 || s.Process.Refused != string(errcodes.ProcessVersionTampered) || len(rs) != 0 {
		t.Fatalf("изменённая версия исполнена: токены %+v, реакции %d", s.Process.Tokens, len(rs))
	}
	err := engine.Guard(s, b, kernel.Command{Action: "process.operation.start", Payload: dp.StartCommand{StepKey: "incoming.marking"}})
	var r *kernel.Refusal
	if !errors.As(err, &r) || r.Code != errcodes.ProcessVersionTampered {
		t.Fatalf("гард принял команду по изменённой версии: %v", err)
	}
	// Возврат исходных байтов — версия снова исполняется.
	j.store.Tamper(app.SeedVersionID, xml)
	j.want("incoming.marking")
}

// Эпик 16: изделие, запущенное с шага выдачи заготовки (entry_step_key —
// после параллельной раздачи по цехам), проходит слияние «фланец + патрубок»:
// ветвь патрубка оно не проходило, ждать её некому.
func TestFlangeEntryAfterSplitPassesJoin(t *testing.T) {
	j := newJourney(t)
	j.add(catalog.ItemItemRegistered, 0, map[string]any{"item_id": j.item, "item_type_id": "FL-100.00.000", "item_revision": "Б",
		"process_version_hash": j.hash, "normative_rev": "flange-1", "lot_ids": []string{"LOT-FL-1"}, "entry_step_key": "incoming.issue_blank"})
	j.want("incoming.issue_blank")
	j.run("RUN-M1-1", "machining.cnc", 1, 2)
	j.decide("machining.zt2_acceptance", dp.ResolutionAccept, 3)
	j.received("welding.receive", 4)
	j.want("welding.edge_prep")
}
