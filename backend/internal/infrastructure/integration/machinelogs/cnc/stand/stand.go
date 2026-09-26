package stand

import (
	"time"

	"ant/internal/contracts/procs"
	"ant/internal/infrastructure/integration/machinelogs/scripted"
)

// Программа станка ЧПУ (FR-149): токарная обработка по программе O1001
// ред. B инструментом T05 (ресурс 75 циклов; каждое выполнение — цикл).
//   - нормальное выполнение: автомат, нагрузка шпинделя 55–78 % при уставке
//     не выше 100 %, коррекция подачи 100 %;
//   - выполнение с двумя отклонениями: оператор вручную поднимает подачу до
//     130 % (ручное изменение режима), нагрузка шпинделя доходит до 118 %
//     (перегрузка); после — дефект на контроле (сценарий кейса).

const toolLimit = 75

func load(v int) scripted.Sample {
	return scripted.Sample{Parameter: "spindle_load", Value: v, Unit: "%"}
}

func feed(v int) scripted.Sample {
	return scripted.Sample{Parameter: "feed_override", Value: v, Unit: "%"}
}

func start() scripted.Step {
	return scripted.Step{Cycle: "start", Events: []scripted.Event{
		{Category: "program", Value: "O1001@B"},
		{Category: "tool", Value: "T05"},
		{Category: "execution", Value: "ACTIVE"},
		{Category: "controller_mode", Value: "AUTOMATIC"},
		{Category: "condition", Value: "NORMAL"},
	}, Samples: []scripted.Sample{load(55), feed(100)}}
}

func end() scripted.Step {
	return scripted.Step{Cycle: "end", Events: []scripted.Event{{Category: "execution", Value: "READY"}}, Samples: []scripted.Sample{load(20), feed(100)}}
}

// Executions — программы выполнений станка ЧПУ.
func Executions() map[string]scripted.Execution {
	return map[string]scripted.Execution{
		"normal": {Kind: "normal", Steps: []scripted.Step{
			start(),
			{Samples: []scripted.Sample{load(68), feed(100)}},
			{Samples: []scripted.Sample{load(78), feed(100)}},
			{Samples: []scripted.Sample{load(72), feed(100)}},
			end(),
		}},
		"deviations": {Kind: "deviations", Steps: []scripted.Step{
			start(),
			{Events: []scripted.Event{{Category: "override", Value: "130", Code: "feed"}}, Samples: []scripted.Sample{load(84), feed(130)}},
			{Samples: []scripted.Sample{load(118), feed(130)}},
			{Events: []scripted.Event{{Category: "override", Value: "100", Code: "feed"}}, Samples: []scripted.Sample{load(74), feed(100)}},
			end(),
		}},
	}
}

// New — stand станка ЧПУ equipmentID, отдающий телеметрию edge-агенту edgeURL.
func New(name, equipmentID, edgeURL string, interval time.Duration) *scripted.Stand {
	return &scripted.Stand{
		Name: name, EquipmentID: equipmentID, EdgeURL: edgeURL, Interval: interval,
		Emulates: "станок ЧПУ " + equipmentID + ": MTConnect (EXECUTION, CONTROLLER_MODE, CONDITION, программа, инструмент, коррекция подачи, нагрузка шпинделя) → edge-агент",
		Setpoints: []procs.StandTelemetryV1SetpointsElem{
			{Parameter: "spindle_load", Nominal: scripted.Ptr(80), Lower: scripted.Ptr(0), Upper: scripted.Ptr(100), Unit: "%"},
			{Parameter: "feed_override", Nominal: scripted.Ptr(100), Lower: scripted.Ptr(100), Upper: scripted.Ptr(100), Unit: "%"},
		},
		Executions: Executions(),
		Prepare: func(n int, s scripted.Step) scripted.Step {
			// Ресурс инструмента растёт с каждым выполнением; на пределе — замена.
			used := 70 + n
			for used > toolLimit {
				used -= toolLimit
			}
			evs := make([]scripted.Event, len(s.Events))
			for i, e := range s.Events {
				if e.Category == "tool" {
					e.ToolUsed, e.ToolLimit = scripted.Ptr(used), scripted.Ptr(toolLimit)
				}
				evs[i] = e
			}
			s.Events = evs
			return s
		},
	}
}
