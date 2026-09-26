package analysis

import (
	"encoding/json"
	"strconv"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/domain/kernel"
)

// Представления чтения данных чужих фактов, которые разбирает модуль analysis.
//
// Почему не сгенерированные типы contracts/events: у сгенерированного
// events.Timestamp (`type Timestamp time.Time`) нет UnmarshalJSON/MarshalJSON,
// поэтому поля времени (started_at, window_start, interval_start…) не
// разбираются из JSON. Здесь — только нужные поля с теми же ключами JSON, что
// в схемах contracts/events (источник правды — схемы, AD-20); время — time.Time.
// Правку генератора (методы JSON у Timestamp) эпик 22 передал дирижёру.

// runStartedData — operation.run.started (FR-47, FR-148).
type runStartedData struct {
	OperationRunID string     `json:"operation_run_id"`
	StepKey        string     `json:"step_key"`
	OperationCode  string     `json:"operation_code"`
	EquipmentID    *string    `json:"equipment_id"`
	OperatorID     *string    `json:"operator_id"`
	ProgramRef     *string    `json:"program_ref"`
	ReworkOf       *string    `json:"rework_of"`
	WorkplaceID    *string    `json:"workplace_id"`
	StartedAt      *time.Time `json:"operation_started_at"`
}

// runFinishedData — operation.run.finished.
type runFinishedData struct {
	OperationRunID string     `json:"operation_run_id"`
	Completion     string     `json:"completion"`
	FinishedAt     *time.Time `json:"operation_finished_at"`
	StartedAt      *time.Time `json:"operation_started_at"`
}

// intervalData — operation.run.interval_resolved (единственный алгоритм интервала — process, AD-42).
type intervalData struct {
	OperationRunID string     `json:"operation_run_id"`
	EquipmentID    *string    `json:"equipment_id"`
	IntervalStart  time.Time  `json:"interval_start"`
	IntervalEnd    *time.Time `json:"interval_end"`
}

// defectData — признак дефекта в результате контроля (FR-37).
type defectData struct {
	ZoneID         string  `json:"zone_id"`
	DefectTypeCode *string `json:"defect_type_code"`
	ComponentRef   *string `json:"component_ref"`
	Location       *string `json:"location"`
}

// evidenceData — материал результата контроля (кадр, протокол).
type evidenceData struct {
	MaterialAddress string `json:"material_address"`
	IsIllustration  bool   `json:"is_illustration"`
}

// inspectionData — inspection.result.recorded (FR-36, FR-38).
type inspectionData struct {
	Outcome         string         `json:"outcome"`
	Phase           string         `json:"phase"`
	Method          string         `json:"method"`
	ZoneIDs         []string       `json:"zone_ids"`
	Defects         []defectData   `json:"defects"`
	OperationRunID  *string        `json:"operation_run_id"`
	InspectionPoint *string        `json:"inspection_point"`
	StepKey         *string        `json:"step_key"`
	ConclusionRef   *string        `json:"conclusion_ref"`
	LotID           *string        `json:"lot_id"`
	QualityBP       *int           `json:"observation_quality_bp"`
	EvidenceRefs    []evidenceData `json:"evidence_refs"`
}

// ncConfirmedData — decision.nonconformity.confirmed (FR-51).
type ncConfirmedData struct {
	NcID           string   `json:"nc_id"`
	DefectTypeCode *string  `json:"defect_type_code"`
	Severity       string   `json:"severity"`
	SignalIDs      []string `json:"signal_ids"`
}

// operatorData — действия исполнителя (семейство operator): обход, отклонение,
// смена режима, наблюдение OperatorVision, пропуск проверки.
type operatorData struct {
	OperationRunID *string `json:"operation_run_id"`
	OperatorID     *string `json:"operator_id"`
	Bypassed       string  `json:"bypassed"`
	Observation    string  `json:"observation"`
	Description    string  `json:"description"`
	Mode           string  `json:"mode"`
	Reason         *struct {
		Text string `json:"text"`
	} `json:"reason"`
}

// movementData — operation.movement.sent / received.
type movementData struct {
	FromLocationID string `json:"from_location_id"`
	ToLocationID   string `json:"to_location_id"`
	LocationID     string `json:"location_id"`
}

// assemblyData — item.assembly.recorded (генеалогия, эпик 18).
type assemblyData struct {
	AssemblyItemID  string  `json:"assembly_item_id"`
	ComponentItemID *string `json:"component_item_id"`
	ComponentLotID  *string `json:"component_lot_id"`
	ComponentTypeID string  `json:"component_type_id"`
}

// lotIssuedData — genealogy.lot.issued: партия выдана изделиям.
type lotIssuedData struct {
	LotID   string   `json:"lot_id"`
	ItemIDs []string `json:"item_ids"`
}

// measurementData — измерение: целое + единица + масштаб (AD-4).
type measurementData struct {
	Value int    `json:"value"`
	Unit  string `json:"unit"`
	Scale int    `json:"scale"`
}

// toleranceData — уставка.
type toleranceData struct {
	Nominal *measurementData `json:"nominal"`
	Lower   *measurementData `json:"lower"`
	Upper   *measurementData `json:"upper"`
}

// parameterSummaryData — сводка параметра за цикл.
type parameterSummaryData struct {
	Parameter       string           `json:"parameter"`
	Max             *measurementData `json:"max"`
	Mean            *measurementData `json:"mean"`
	OutOfSetpointMs *int             `json:"out_of_setpoint_ms"`
	Setpoint        *toleranceData   `json:"setpoint"`
}

// equipmentData — факты семейства equipment (FR-121, FR-147): отклонение,
// сводка цикла, состояние, инструмент, программа.
type equipmentData struct {
	EquipmentID     string                 `json:"equipment_id"`
	DeviationKind   string                 `json:"deviation_kind"`
	StartedAt       *time.Time             `json:"started_at"`
	EndedAt         *time.Time             `json:"ended_at"`
	Parameter       *string                `json:"parameter"`
	Value           *measurementData       `json:"value"`
	Setpoint        *toleranceData         `json:"setpoint"`
	WindowStart     *time.Time             `json:"window_start"`
	WindowEnd       *time.Time             `json:"window_end"`
	Parameters      []parameterSummaryData `json:"parameters"`
	ToolID          string                 `json:"tool_id"`
	FixtureID       *string                `json:"fixture_id"`
	ProgramRef      string                 `json:"program_ref"`
	ProgramRevision string                 `json:"program_revision"`
	Planned         *bool                  `json:"planned"`
	Condition       string                 `json:"condition"`
	Execution       string                 `json:"execution"`
	ControllerMode  string                 `json:"controller_mode"`
}

// decodeAs разбирает data записи в представление чтения; ошибка разбора —
// пустое значение и false (запись с неверными данными не ломает свёртку:
// схему проверил приём, AD-20).
func decodeAs[T any](r kernel.Record) (T, bool) {
	var v T
	if len(r.Data) == 0 {
		return v, false
	}
	if err := json.Unmarshal(r.Data, &v); err != nil {
		return v, false
	}
	return v, true
}

// family — семейство типа записи (первый сегмент).
func family(t catalog.Type) string {
	s := string(t)
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			return s[:i]
		}
	}
	return s
}

// deref — значение строки или пусто.
func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// fmtMeasurement — измерение для подписи: «176 A», «1.5 mm» (без float, AD-4).
func fmtMeasurement(m *measurementData) string {
	if m == nil {
		return ""
	}
	s := fmtScaled(m.Value, m.Scale)
	if m.Unit != "" {
		s += " " + m.Unit
	}
	return s
}

// fmtScaled — целое с масштабом как десятичная строка.
func fmtScaled(v, scale int) string {
	neg := v < 0
	if neg {
		v = -v
	}
	s := strconv.Itoa(v)
	if scale > 0 {
		for len(s) <= scale {
			s = "0" + s
		}
		s = s[:len(s)-scale] + "." + s[len(s)-scale:]
	}
	if neg {
		s = "-" + s
	}
	return s
}

// fmtTolerance — уставка для подписи: «160 ± 10 A» или «150…170 A».
func fmtTolerance(t *toleranceData) string {
	if t == nil {
		return ""
	}
	switch {
	case t.Lower != nil && t.Upper != nil:
		return fmtScaled(t.Lower.Value, t.Lower.Scale) + "…" + fmtMeasurement(t.Upper)
	case t.Nominal != nil:
		return fmtMeasurement(t.Nominal)
	case t.Upper != nil:
		return "≤ " + fmtMeasurement(t.Upper)
	case t.Lower != nil:
		return "≥ " + fmtMeasurement(t.Lower)
	}
	return ""
}
