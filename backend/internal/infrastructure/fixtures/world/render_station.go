package world

import (
	"fmt"
	"strings"

	ncapp "ant/internal/application/nonconformity"
	"ant/internal/infrastructure/fixtures/loader"
)

// Окно операции участка на заготовках (интерфейс 6, nonconformity.station.read;
// FR-49): счётчики шага за смену, действующие остановки точки процесса мира
// (process_holds: оборудование → шаг его поста), предложение системы и
// действия. Доступность «Снять остановку» ставит адаптер по вошедшему.

// stepEquipment — оборудование шагов процесса мира (пост → оборудование, render_ops.go).
var stepEquipment = map[string]string{
	"machining.cnc": "CNC-1", "machining.kt2_cmm": "CMM-1", "welding.weld": "IS-2", "welding.kt3_radiography": "XRAY-LAB",
	"testing.leak_test": "LEAK-1", "assembly.torque": "TW-1",
}

// holdID — id остановки мира: оборудование и день остановки.
func (c *Ctx) holdID(h ProcessHold) string {
	return fmt.Sprintf("HOLD-%s-%s", h.Equipment, h.Set.Time().In(c.M.clk.loc).Format("0102"))
}

// holdStep — шаг, на котором стоит оборудование остановки.
func holdStep(equipment string) string {
	for k, v := range stepEquipment {
		if v == equipment {
			return k
		}
	}
	return ""
}

// renderStation — окно операции участка по шагам операций и контроля.
func renderStation(c *Ctx) []loader.Response {
	var out []loader.Response
	cs := c.CountersFrom(c.window("shift").From)
	for _, n := range c.M.BpmnOrder {
		if n.StepKey == "" || strings.HasSuffix(n.StepKey, ".merge") {
			continue
		}
		switch nodeKind(n) {
		case "operation", "automatedInspection", "humanInspection":
		default:
			continue
		}
		v := ncapp.StationView{StepKey: n.StepKey, StepLabel: n.Name, EquipmentID: stepEquipment[n.StepKey], ActiveHolds: []ncapp.StationHold{}, BasisSeq: c.Seq()}
		if cnt := cs[n.StepKey]; cnt != nil {
			v.Counters = &ncapp.StationCounters{Queue: cnt.Queue, InProgress: cnt.InProgress, Passed: cnt.Passed, Defects: cnt.Defects, Nonconformities: cnt.NCs}
		} else {
			v.Counters = &ncapp.StationCounters{}
		}
		for _, h := range c.M.Spec.ProcessHolds {
			if h.Set.Time().After(c.T) || holdStep(h.Equipment) != n.StepKey {
				continue
			}
			sh := ncapp.StationHold{HoldID: c.holdID(h), Reason: h.Reason, Since: h.Set.Time().UTC(), Level: "process_point_stop",
				ReleaseCondition: h.ReleaseCondition, EquipmentID: h.Equipment}
			for _, e := range c.Visible() {
				if e.Type == "decision.process_hold.set" && e.Params["equipment_id"] == h.Equipment {
					sh.SetEventID = e.ID
				}
			}
			for _, in := range c.M.Incidents {
				if !in.Spec.Opened.Time().After(c.T) {
					sh.IncidentID = in.Spec.ID
				}
			}
			v.ActiveHolds = append(v.ActiveHolds, sh)
			v.EquipmentID = h.Equipment
		}
		// Предложение системы: остановлено — снять после условия; иначе — остановить,
		// если на шаге открыты несоответствия (решает человек).
		switch {
		case len(v.ActiveHolds) > 0:
			h := v.ActiveHolds[0]
			v.Suggestion = &ncapp.StationSuggestion{Outcome: "release", Why: []string{
				"Остановка действует с " + h.Since.In(c.M.clk.loc).Format("02.01 15:04") + ": " + h.Reason,
				"Снять — когда выполнено условие: " + h.ReleaseCondition,
				"После снятия первые изделия пройдут под усиленным контролем (точка чистоты)"}}
		case v.Counters.Nonconformities > 0:
			v.Suggestion = &ncapp.StationSuggestion{Outcome: "stop", Why: []string{
				fmt.Sprintf("На шаге открытых несоответствий: %d", v.Counters.Nonconformities),
				"Пока причина не найдена, новые изделия на шаге могут получить тот же дефект"}}
		}
		v.Actions = ncapp.StationActions(v.StepKey, v.StepLabel, v.EquipmentID, v.ActiveHolds, false)
		out = append(out, resp("nonconformity.station.read", v, "step_key", n.StepKey))
	}
	return out
}
