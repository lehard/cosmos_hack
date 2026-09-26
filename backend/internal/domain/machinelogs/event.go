package machinelogs

import (
	"encoding/json"
	"fmt"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/domain/kernel"
)

// Measure — измерение без float (AD-4): значение = Value × 10^(−Scale) в
// единице Unit (код UCUM).
type Measure struct {
	Value int64  `json:"value"`
	Scale int    `json:"scale"`
	Unit  string `json:"unit"`
}

// Tolerance — уставка: номинал и границы; отсутствующая граница не ограничена.
type Tolerance struct {
	Nominal *Measure `json:"nominal,omitempty"`
	Lower   *Measure `json:"lower,omitempty"`
	Upper   *Measure `json:"upper,omitempty"`
}

// pow10 — 10^n в целых (n ≤ 18).
func pow10(n int) int64 {
	p := int64(1)
	for range n {
		p *= 10
	}
	return p
}

// Cmp сравнивает два измерения одной единицы с разным масштабом: приводит
// к большему масштабу целым умножением (AD-4). −1, 0, 1.
func Cmp(a, b Measure) int {
	s := max(a.Scale, b.Scale)
	x, y := a.Value*pow10(s-a.Scale), b.Value*pow10(s-b.Scale)
	switch {
	case x < y:
		return -1
	case x > y:
		return 1
	}
	return 0
}

// Contains — значение в пределах уставки; known=false — оценка невозможна
// (нет границ или единицы не совпадают, FR-123).
func (t Tolerance) Contains(v Measure) (in, known bool) {
	if t.Lower == nil && t.Upper == nil {
		return false, false
	}
	if (t.Lower != nil && t.Lower.Unit != v.Unit) || (t.Upper != nil && t.Upper.Unit != v.Unit) {
		return false, false
	}
	if t.Lower != nil && Cmp(v, *t.Lower) < 0 {
		return false, true
	}
	if t.Upper != nil && Cmp(v, *t.Upper) > 0 {
		return false, true
	}
	return true, true
}

// ParamSummary — сводка параметра на окно цикла (FR-147, слой «как шёл процесс»).
type ParamSummary struct {
	Parameter       string     `json:"parameter"`
	Mean            *Measure   `json:"mean,omitempty"`
	Max             *Measure   `json:"max,omitempty"`
	Min             *Measure   `json:"min,omitempty"`
	Setpoint        *Tolerance `json:"setpoint,omitempty"`
	OutOfSetpointMs *int64     `json:"out_of_setpoint_ms,omitempty"`
}

// InRange — сводка в пределах уставки: nil — оценка невозможна (нет уставки
// или значений); false — максимум или минимум вне уставки либо источник
// сообщил время вне уставки.
func (p ParamSummary) InRange() *bool {
	if p.Setpoint == nil {
		return nil
	}
	out := p.OutOfSetpointMs != nil && *p.OutOfSetpointMs > 0
	known := p.OutOfSetpointMs != nil
	for _, m := range []*Measure{p.Max, p.Min, p.Mean} {
		if m == nil {
			continue
		}
		in, ok := p.Setpoint.Contains(*m)
		if ok {
			known = true
			out = out || !in
		}
	}
	if !known {
		return nil
	}
	v := !out
	return &v
}

// Event — событие оборудования в представлении домена: один тип на смысл
// от любого источника с пометкой источника (AD-29, FR-140). Поля данных
// заполнены по типу события; Start/End — интервал события на шкале времени:
// начало и конец отклонения, окно сводки цикла, иначе момент возникновения.
type Event struct {
	EventID    string       `json:"event_id"`
	Seq        int64        `json:"seq,omitempty"`
	Type       catalog.Type `json:"type"`
	RunID      string       `json:"run_id,omitempty"`
	SourceKind string       `json:"source_kind,omitempty"`
	OccurredAt time.Time    `json:"occurred_at"`
	RecordedAt time.Time    `json:"recorded_at,omitzero"`
	Start      time.Time    `json:"start"`
	End        *time.Time   `json:"end,omitempty"`

	EquipmentID string `json:"equipment_id"`
	StationID   string `json:"station_id,omitempty"`

	Execution      string `json:"execution,omitempty"`
	ControllerMode string `json:"controller_mode,omitempty"`
	Condition      string `json:"condition,omitempty"`
	Code           string `json:"code,omitempty"`

	ProgramRef      string `json:"program_ref,omitempty"`
	ProgramRevision string `json:"program_revision,omitempty"`
	Planned         *bool  `json:"planned,omitempty"`

	ToolID        string `json:"tool_id,omitempty"`
	ToolLifeUsed  *int64 `json:"tool_life_used,omitempty"`
	ToolLifeLimit *int64 `json:"tool_life_limit,omitempty"`

	DeviationKind string     `json:"deviation_kind,omitempty"`
	Parameter     string     `json:"parameter,omitempty"`
	Value         *Measure   `json:"value,omitempty"`
	Setpoint      *Tolerance `json:"setpoint,omitempty"`

	CycleRef   string         `json:"cycle_ref,omitempty"`
	RawRef     string         `json:"raw_ref,omitempty"`
	Parameters []ParamSummary `json:"parameters,omitempty"`

	// Data — исходные данные события; хранит только межизделийная стадия,
	// чтобы переслать их в поток изделия записью привязки.
	Data json.RawMessage `json:"data,omitempty"`
}

// Until — конец интервала события (End или Start).
func (e Event) Until() time.Time {
	if e.End != nil {
		return *e.End
	}
	return e.Start
}

// eventData — данные всех типов семейства equipment (текущие версии схем,
// после повышения, AD-20) в одной структуре разбора.
type eventData struct {
	EquipmentID     string         `json:"equipment_id"`
	StationID       string         `json:"station_id"`
	Execution       string         `json:"execution"`
	ControllerMode  string         `json:"controller_mode"`
	Condition       string         `json:"condition"`
	Code            string         `json:"code"`
	Parameter       string         `json:"parameter"`
	Value           *Measure       `json:"value"`
	Setpoint        *Tolerance     `json:"setpoint"`
	ProgramRef      string         `json:"program_ref"`
	ProgramRevision string         `json:"program_revision"`
	Planned         *bool          `json:"planned"`
	ToolID          string         `json:"tool_id"`
	ToolLifeUsed    *int64         `json:"tool_life_used"`
	ToolLifeLimit   *int64         `json:"tool_life_limit"`
	DeviationKind   string         `json:"deviation_kind"`
	StartedAt       *time.Time     `json:"started_at"`
	EndedAt         *time.Time     `json:"ended_at"`
	WindowStart     *time.Time     `json:"window_start"`
	WindowEnd       *time.Time     `json:"window_end"`
	CycleRef        string         `json:"cycle_ref"`
	RawRef          string         `json:"raw_ref"`
	Parameters      []ParamSummary `json:"parameters"`
}

// IsEquipmentFact — факт оборудования, который выделяет edge-агент или вводит
// человек: состояние, программа, инструмент, сводка цикла, отклонение.
func IsEquipmentFact(t catalog.Type) bool {
	switch t {
	case catalog.EquipmentStateChanged, catalog.EquipmentProgramChanged, catalog.EquipmentToolChanged,
		catalog.EquipmentCycleSummarized, catalog.EquipmentDeviationDetected:
		return true
	}
	return false
}

// ParseData разбирает данные события оборудования типа t, возникшего в
// occurred. Интервал: отклонение — started_at…ended_at, сводка — окно цикла,
// остальное — момент occurred.
func ParseData(t catalog.Type, data json.RawMessage, occurred time.Time) (Event, error) {
	var d eventData
	if err := json.Unmarshal(data, &d); err != nil {
		return Event{}, fmt.Errorf("machinelogs: данные %s: %w", t, err)
	}
	e := Event{
		Type: t, OccurredAt: occurred, Start: occurred,
		EquipmentID: d.EquipmentID, StationID: d.StationID,
		Execution: d.Execution, ControllerMode: d.ControllerMode, Condition: d.Condition, Code: d.Code,
		ProgramRef: d.ProgramRef, ProgramRevision: d.ProgramRevision, Planned: d.Planned,
		ToolID: d.ToolID, ToolLifeUsed: d.ToolLifeUsed, ToolLifeLimit: d.ToolLifeLimit,
		DeviationKind: d.DeviationKind, Parameter: d.Parameter, Value: d.Value, Setpoint: d.Setpoint,
		CycleRef: d.CycleRef, RawRef: d.RawRef, Parameters: d.Parameters,
	}
	switch t {
	case catalog.EquipmentDeviationDetected:
		if d.StartedAt != nil {
			e.Start = d.StartedAt.UTC()
		}
		if d.EndedAt != nil {
			end := d.EndedAt.UTC()
			e.End = &end
		}
	case catalog.EquipmentCycleSummarized:
		if d.WindowStart != nil {
			e.Start = d.WindowStart.UTC()
		}
		if d.WindowEnd != nil {
			end := d.WindowEnd.UTC()
			e.End = &end
		}
	}
	if e.EquipmentID == "" {
		return Event{}, fmt.Errorf("machinelogs: %s без equipment_id", t)
	}
	return e, nil
}

// ParseEvent — событие оборудования из записи журнала (факт семейства
// equipment); ok=false — запись не событие оборудования.
func ParseEvent(r kernel.Record) (Event, bool, error) {
	if !IsEquipmentFact(r.Type) {
		return Event{}, false, nil
	}
	e, err := ParseData(r.Type, r.Data, r.OccurredAt.UTC())
	if err != nil {
		return Event{}, true, err
	}
	e.EventID, e.Seq, e.RunID = r.EventID, r.Seq, r.RunID
	e.SourceKind = SourceKind(r.SourceKind)
	e.RecordedAt = r.RecordedAt.UTC()
	return e, true, nil
}
