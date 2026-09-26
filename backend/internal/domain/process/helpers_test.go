package process

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/domain/kernel"
)

// miniBPMN — учебный процесс в стиле фланца: подготовка кромок → сварка
// (специальный процесс, граничный таймер окна 8 ч until_started, лимит 3 на
// зону) → камера → ЗТ-3 → сборка с закрытием зоны → ЗТ-6 → «в 1С» → выпуск;
// подпроцесс брака вызывается из двух мест и возвращает в точку вызова.
const miniBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
  xmlns:ant="urn:ant:bpmn-ext:1" id="Defs_Mini" targetNamespace="urn:test" expressionLanguage="urn:ant:expr:1">
  <bpmn:message id="Msg_Order" name="Задание из 1С"/>
  <bpmn:message id="Msg_Release" name="В 1С: выпуск"/>
  <bpmn:process id="Main" isExecutable="true">
    <bpmn:laneSet id="LS">
      <bpmn:lane id="Lane_WC" name="Сварочный цех">
        <bpmn:extensionElements><ant:properties stepKey="lane.wc" workshop="WS-WC" warehouse="WH-WC"/></bpmn:extensionElements>
        <bpmn:flowNodeRef>P</bpmn:flowNodeRef><bpmn:flowNodeRef>W</bpmn:flowNodeRef>
      </bpmn:lane>
    </bpmn:laneSet>
    <bpmn:startEvent id="S0" name="Задание"><bpmn:extensionElements><ant:properties stepKey="order.received" triggerEventType="erp.order.received"/></bpmn:extensionElements>
      <bpmn:messageEventDefinition id="MS0" messageRef="Msg_Order"/></bpmn:startEvent>
    <bpmn:exclusiveGateway id="J_P"><bpmn:extensionElements><ant:properties stepKey="welding.edge_prep.merge"/></bpmn:extensionElements></bpmn:exclusiveGateway>
    <bpmn:task id="P" name="Подготовка кромок"><bpmn:documentation>Кромки.</bpmn:documentation>
      <bpmn:extensionElements><ant:properties stepKey="welding.edge_prep" stepKind="operation" reworkLimit="3" reworkLimitScope="loop"/></bpmn:extensionElements></bpmn:task>
    <bpmn:task id="W" name="Сварка"><bpmn:extensionElements>
      <ant:properties stepKey="welding.weld" stepKind="operation" specialProcess="true" reworkLimit="3" reworkLimitScope="zone"/>
      <ant:zoneRef zone="U1"/><ant:zoneRef zone="U2"/>
      <ant:precondition kind="qualification" mode="block" ref="welder"/>
      <ant:precondition kind="time_window" mode="block" ref="edge_prep&lt;=PT8H"/>
    </bpmn:extensionElements></bpmn:task>
    <bpmn:boundaryEvent id="WT" attachedToRef="W" cancelActivity="true"><bpmn:extensionElements><ant:properties stepKey="welding.edge_window_expired" timerScope="until_started"/></bpmn:extensionElements>
      <bpmn:timerEventDefinition id="TW"><bpmn:timeDuration xsi:type="bpmn:tFormalExpression">PT8H</bpmn:timeDuration></bpmn:timerEventDefinition></bpmn:boundaryEvent>
    <bpmn:exclusiveGateway id="J_K"><bpmn:extensionElements><ant:properties stepKey="welding.kt3.merge"/></bpmn:extensionElements></bpmn:exclusiveGateway>
    <bpmn:serviceTask id="K" name="КТ-3 камера"><bpmn:extensionElements><ant:properties stepKey="welding.kt3" stepKind="automated_inspection" inspectionPoint="KT-3"/>
      <ant:inspection method="camera" phase="after_operation" coverage="W-PORE-S"/><ant:requirement characteristic="шов" tolerance="нет" kdRef="КД"/><ant:zoneRef zone="U1"/><ant:zoneRef zone="U2"/></bpmn:extensionElements></bpmn:serviceTask>
    <bpmn:userTask id="G" name="ЗТ-3"><bpmn:extensionElements><ant:properties stepKey="welding.zt3" stepKind="human_inspection" closingPoint="ZT-3"/>
      <ant:presentationPoint authority="qc_acceptance" role="quality_inspector" waitLimitMinutes="60" repeatAuthority="qc_acceptance_repeat"/></bpmn:extensionElements></bpmn:userTask>
    <bpmn:exclusiveGateway id="GD"><bpmn:extensionElements><ant:properties stepKey="welding.zt3_decision"/></bpmn:extensionElements></bpmn:exclusiveGateway>
    <bpmn:callActivity id="NC1" calledElement="NC"><bpmn:extensionElements><ant:properties stepKey="welding.nonconformity"/></bpmn:extensionElements></bpmn:callActivity>
    <bpmn:exclusiveGateway id="NG1"><bpmn:extensionElements><ant:properties stepKey="welding.nonconformity_outcome"/></bpmn:extensionElements></bpmn:exclusiveGateway>
    <bpmn:endEvent id="X1" name="Списано"><bpmn:extensionElements><ant:properties stepKey="welding.scrapped"/></bpmn:extensionElements><bpmn:terminateEventDefinition id="TX1"/></bpmn:endEvent>
    <bpmn:exclusiveGateway id="J_A"><bpmn:extensionElements><ant:properties stepKey="assembly.cover.merge"/></bpmn:extensionElements></bpmn:exclusiveGateway>
    <bpmn:task id="A" name="Установка крышки"><bpmn:extensionElements><ant:properties stepKey="assembly.cover" stepKind="operation" closesZoneAccess="Z9"/></bpmn:extensionElements></bpmn:task>
    <bpmn:userTask id="G2" name="ЗТ-6"><bpmn:extensionElements><ant:properties stepKey="final.zt6" stepKind="human_inspection" closingPoint="ZT-6"/>
      <ant:presentationPoint authority="qc_acceptance" role="quality_inspector" repeatAuthority="qc_acceptance_repeat"/>
      <ant:precondition kind="open_intervention" mode="block"/></bpmn:extensionElements></bpmn:userTask>
    <bpmn:exclusiveGateway id="GD2"><bpmn:extensionElements><ant:properties stepKey="final.zt6_decision"/></bpmn:extensionElements></bpmn:exclusiveGateway>
    <bpmn:callActivity id="NC2" calledElement="NC"><bpmn:extensionElements><ant:properties stepKey="final.nonconformity"/></bpmn:extensionElements></bpmn:callActivity>
    <bpmn:exclusiveGateway id="NG2"><bpmn:extensionElements><ant:properties stepKey="final.nonconformity_outcome"/></bpmn:extensionElements></bpmn:exclusiveGateway>
    <bpmn:endEvent id="X2" name="Списано"><bpmn:extensionElements><ant:properties stepKey="final.scrapped"/></bpmn:extensionElements><bpmn:terminateEventDefinition id="TX2"/></bpmn:endEvent>
    <bpmn:intermediateThrowEvent id="E1" name="В 1С: выпуск"><bpmn:extensionElements><ant:properties stepKey="final.erp_release" erpAction="release"/></bpmn:extensionElements>
      <bpmn:messageEventDefinition id="ME1" messageRef="Msg_Release"/></bpmn:intermediateThrowEvent>
    <bpmn:endEvent id="END" name="Выпущено"><bpmn:extensionElements><ant:properties stepKey="final.released"/></bpmn:extensionElements></bpmn:endEvent>
    <bpmn:sequenceFlow id="f1" sourceRef="S0" targetRef="J_P"/>
    <bpmn:sequenceFlow id="f2" sourceRef="J_P" targetRef="P"/>
    <bpmn:sequenceFlow id="f3" sourceRef="P" targetRef="W"/>
    <bpmn:sequenceFlow id="f4" sourceRef="W" targetRef="J_K"/>
    <bpmn:sequenceFlow id="f5" sourceRef="WT" targetRef="J_P"/>
    <bpmn:sequenceFlow id="f6" sourceRef="J_K" targetRef="K"/>
    <bpmn:sequenceFlow id="f7" sourceRef="K" targetRef="G"/>
    <bpmn:sequenceFlow id="f8" sourceRef="G" targetRef="GD"/>
    <bpmn:sequenceFlow id="f9" sourceRef="GD" targetRef="J_A"><bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">decision == 'accept'</bpmn:conditionExpression></bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="f10" sourceRef="GD" targetRef="J_K"><bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">decision == 'insufficient_data'</bpmn:conditionExpression></bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="f11" sourceRef="GD" targetRef="NC1"><bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">decision == 'reject'</bpmn:conditionExpression></bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="f12" sourceRef="NC1" targetRef="NG1"/>
    <bpmn:sequenceFlow id="f13" sourceRef="NG1" targetRef="J_P"><bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">nc.outcome == 'rework_or_repair'</bpmn:conditionExpression></bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="f14" sourceRef="NG1" targetRef="X1"><bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">nc.outcome == 'scrapped'</bpmn:conditionExpression></bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="f15" sourceRef="J_A" targetRef="A"/>
    <bpmn:sequenceFlow id="f16" sourceRef="A" targetRef="G2"/>
    <bpmn:sequenceFlow id="f17" sourceRef="G2" targetRef="GD2"/>
    <bpmn:sequenceFlow id="f18" sourceRef="GD2" targetRef="E1"><bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">decision == 'accept' or decision == 'accept_with_concession'</bpmn:conditionExpression></bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="f19" sourceRef="GD2" targetRef="NC2"><bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">decision == 'reject'</bpmn:conditionExpression></bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="f20" sourceRef="NC2" targetRef="NG2"/>
    <bpmn:sequenceFlow id="f21" sourceRef="NG2" targetRef="J_A"><bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">nc.outcome == 'rework_or_repair'</bpmn:conditionExpression></bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="f22" sourceRef="NG2" targetRef="X2"><bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">nc.outcome == 'scrapped'</bpmn:conditionExpression></bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="f23" sourceRef="E1" targetRef="END"/>
  </bpmn:process>
  <bpmn:process id="NC" isExecutable="true">
    <bpmn:startEvent id="N0"><bpmn:extensionElements><ant:properties stepKey="nc.start"/></bpmn:extensionElements></bpmn:startEvent>
    <bpmn:task id="N1" name="Изоляция"><bpmn:extensionElements><ant:properties stepKey="nc.isolation" stepKind="movement"/></bpmn:extensionElements></bpmn:task>
    <bpmn:userTask id="N2" name="ЗТ-Р"><bpmn:extensionElements><ant:properties stepKey="nc.disposition" stepKind="human_inspection" closingPoint="ZT-R"/>
      <ant:presentationPoint authority="nc_disposition" role="technologist" waitLimitWorkDays="3"/></bpmn:extensionElements></bpmn:userTask>
    <bpmn:exclusiveGateway id="NGW"><bpmn:extensionElements><ant:properties stepKey="nc.disposition_decision"/></bpmn:extensionElements></bpmn:exclusiveGateway>
    <bpmn:endEvent id="NR"><bpmn:extensionElements><ant:properties stepKey="nc.end_rework" outcome="rework_or_repair"/></bpmn:extensionElements></bpmn:endEvent>
    <bpmn:endEvent id="NS"><bpmn:extensionElements><ant:properties stepKey="nc.end_scrapped" outcome="scrapped"/></bpmn:extensionElements></bpmn:endEvent>
    <bpmn:sequenceFlow id="n1" sourceRef="N0" targetRef="N1"/>
    <bpmn:sequenceFlow id="n2" sourceRef="N1" targetRef="N2"/>
    <bpmn:sequenceFlow id="n3" sourceRef="N2" targetRef="NGW"/>
    <bpmn:sequenceFlow id="n4" sourceRef="NGW" targetRef="NR"><bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">disposition == 'rework' or disposition == 'repair'</bpmn:conditionExpression></bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="n5" sourceRef="NGW" targetRef="NS"><bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">disposition == 'scrap'</bpmn:conditionExpression></bpmn:sequenceFlow>
  </bpmn:process>
</bpmn:definitions>`

var t0 = time.Date(2026, 9, 22, 6, 0, 0, 0, time.UTC)

// at — t0 + h десятых часа (6 мин): домен без float (AD-4).
func at(h int) time.Time { return t0.Add(time.Duration(h) * 6 * time.Minute) }

// world — изделие учебного процесса: копит записи входа и сворачивает их.
type world struct {
	t    *testing.T
	env  Env
	item string
	in   []kernel.Record
	seq  int64
}

func newWorld(t *testing.T) *world {
	t.Helper()
	d, vs := Load([]byte(miniBPMN), LoadOptions{})
	if errs := Errors(vs); len(errs) > 0 {
		t.Fatalf("учебный процесс не загрузился: %v", errs)
	}
	return &world{t: t, item: "ent01:FL-1", env: Env{VersionID: "mini-1", VersionHash: VersionHash([]byte(miniBPMN)), Def: d}}
}

// add — запись входа (факт устройства или решение человека по каталогу).
func (w *world) add(tp catalog.Type, h int, data map[string]any) kernel.Record {
	w.seq++
	info, ok := catalog.Lookup(tp)
	kind := catalog.KindFact
	if ok {
		kind = info.Kind
	}
	prov := "device"
	if kind == catalog.KindDecision {
		prov = "personal"
	}
	b, err := json.Marshal(data)
	if err != nil {
		w.t.Fatal(err)
	}
	r := kernel.Record{Seq: w.seq, EventID: fmt.Sprintf("00000000-0000-7000-8000-%012d", w.seq), Type: tp, Kind: kind, Provenance: prov,
		ItemID: w.item, Stream: "item:" + w.item, OccurredAt: at(h), ReceivedAt: at(h), RecordedAt: at(h), Data: b}
	w.in = append(w.in, r)
	return r
}

// fold — свёртка входа модулем process (как движок: Reduce, React, Apply).
func (w *world) fold() (State, []kernel.Reaction) {
	var s State
	bySlot := map[string]kernel.Reaction{}
	for _, r := range w.in {
		s = Reduce(s, r, w.env, Upstream{})
		for _, re := range React(s, w.env, Upstream{}).Reactions {
			bySlot[re.Slot.Key()] = re
		}
	}
	var out []kernel.Reaction
	for _, k := range sortedKeys(bySlot) {
		out = append(out, bySlot[k])
	}
	return s, out
}

func (w *world) register(h int) {
	w.add(catalog.ItemItemRegistered, h, map[string]any{"item_id": w.item, "item_type_id": "FL-100.00.000", "item_revision": "Б",
		"process_version_hash": w.env.VersionHash, "normative_rev": "n1", "lot_ids": []string{"LOT-1"}})
}

func (w *world) start(run, step string, h int, extra ...string) {
	d := map[string]any{"operation_run_id": run, "operation_code": "010", "step_key": step, "operator_id": "WLD-01"}
	for i := 0; i+1 < len(extra); i += 2 {
		d[extra[i]] = extra[i+1]
	}
	w.add(catalog.OperationRunStarted, h, d)
}

func (w *world) finish(run string, h int) {
	w.add(catalog.OperationRunFinished, h, map[string]any{"operation_run_id": run, "completion": "completed"})
}

func (w *world) inspect(step string, h int, outcome string, zones ...string) {
	w.add(catalog.InspectionResultRecorded, h, map[string]any{"method": "camera", "phase": "after_operation", "step_key": step,
		"outcome": outcome, "processing_state": "completed", "zone_ids": zones})
}

func (w *world) decide(step, resolution string, h int) kernel.Record {
	return w.add(catalog.DecisionPresentationResolved, h, map[string]any{"step_key": step, "closing_point": "ZT", "resolution": resolution,
		"presentation_no": 1, "method_event_ids": []string{}})
}

func (w *world) dispose(disposition string, h int) {
	w.add(catalog.DecisionDispositionSet, h, map[string]any{"nc_id": "NC-1", "disposition": disposition, "reason": map[string]string{"code": "x", "text": "x"}})
}

func steps(s State) []string { return s.Steps() }

func reactionsOf(rs []kernel.Reaction, t catalog.Type) []kernel.Reaction {
	var out []kernel.Reaction
	for _, r := range rs {
		if r.Type == t {
			out = append(out, r)
		}
	}
	return out
}
