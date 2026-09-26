package stand

import (
	"time"

	"ant/internal/contracts/procs"
	"ant/internal/infrastructure/integration/machinelogs/scripted"
)

// Программа сварочного источника (FR-149, FR-151): шов W-1 по программе
// ПС-4 (WPS: ток 160 ± 10 А, напряжение 24 ± 2 В).
//   - нормальное выполнение: ток 158–166 А, автомат;
//   - выполнение с двумя отклонениями: ток выше уставки (176–182 А) и ручное
//     изменение режима (режим управления «ручной»); сварка — специальный
//     процесс: нарушение режима — несоответствие всем изделиям окна, даже без
//     найденного дефекта.

func current(v int) scripted.Sample {
	return scripted.Sample{Parameter: "current", Value: v, Scale: 1, Unit: "A"}
}

func voltage(v int) scripted.Sample {
	return scripted.Sample{Parameter: "voltage", Value: v, Scale: 1, Unit: "V"}
}

func start() scripted.Step {
	return scripted.Step{Cycle: "start", Events: []scripted.Event{
		{Category: "program", Value: "PS-4@3"},
		{Category: "execution", Value: "ACTIVE"},
		{Category: "controller_mode", Value: "AUTOMATIC"},
		{Category: "condition", Value: "NORMAL"},
	}, Samples: []scripted.Sample{current(1580), voltage(238)}}
}

func end() scripted.Step {
	return scripted.Step{Cycle: "end", Events: []scripted.Event{{Category: "execution", Value: "READY"}}}
}

// Executions — программы выполнений сварочного источника.
func Executions() map[string]scripted.Execution {
	return map[string]scripted.Execution{
		"normal": {Kind: "normal", Steps: []scripted.Step{
			start(),
			{Samples: []scripted.Sample{current(1620), voltage(240)}},
			{Samples: []scripted.Sample{current(1660), voltage(242)}},
			{Samples: []scripted.Sample{current(1610), voltage(239)}},
			end(),
		}},
		"deviations": {Kind: "deviations", Steps: []scripted.Step{
			start(),
			{Samples: []scripted.Sample{current(1760), voltage(245)}},
			{Events: []scripted.Event{{Category: "controller_mode", Value: "MANUAL"}}, Samples: []scripted.Sample{current(1820), voltage(247)}},
			{Samples: []scripted.Sample{current(1790), voltage(246)}},
			{Events: []scripted.Event{{Category: "controller_mode", Value: "AUTOMATIC"}}, Samples: []scripted.Sample{current(1630), voltage(240)}},
			end(),
		}},
	}
}

// New — stand сварочного источника equipmentID, отдающий телеметрию edge-агенту edgeURL.
func New(name, equipmentID, edgeURL string, interval time.Duration) *scripted.Stand {
	return &scripted.Stand{
		Name: name, EquipmentID: equipmentID, EdgeURL: edgeURL, Interval: interval,
		Emulates: "сварочный источник " + equipmentID + ": регистрация режима шва (ток, напряжение, режим управления, исправность) → edge-агент",
		Setpoints: []procs.StandTelemetryV1SetpointsElem{
			{Parameter: "current", Nominal: scripted.Ptr(1600), Lower: scripted.Ptr(1500), Upper: scripted.Ptr(1700), Scale: 1, Unit: "A"},
			{Parameter: "voltage", Nominal: scripted.Ptr(240), Lower: scripted.Ptr(220), Upper: scripted.Ptr(260), Scale: 1, Unit: "V"},
		},
		Executions: Executions(),
	}
}
