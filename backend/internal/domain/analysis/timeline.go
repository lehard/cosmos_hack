package analysis

import (
	"slices"
	"strings"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/domain/kernel"
)

// Дорожки экрана «Разбор обстоятельств» (FR-153): изделие, человек, оборудование.
const (
	LaneItem      = "item"
	LanePerson    = "person"
	LaneEquipment = "equipment"
)

// Mark — событие на дорожке разбора: ссылка на запись журнала и подпись
// (FR-153). Смысл — только контрактный: «возможные обстоятельства», не «причина».
type Mark struct {
	EventID    string            `json:"event_id"`
	EventType  string            `json:"event_type"`
	Variant    string            `json:"variant,omitempty"`
	Lane       string            `json:"lane"`
	OccurredAt time.Time         `json:"occurred_at"`
	EndedAt    *time.Time        `json:"ended_at,omitempty"`
	Seq        int64             `json:"seq,omitempty"`
	SourceKind string            `json:"source_kind,omitempty"`
	RunID      string            `json:"run_id,omitempty"`
	Params     map[string]string `json:"params,omitempty"`
	Evidence   []string          `json:"evidence,omitempty"`
}

// Run — выполнение операции изделия (FR-47, FR-148): шаг, оборудование,
// программа, исполнитель и интервал. Исполнитель неизвестен — пусто и
// OperatorKnown = false (FR-123: не додумывать).
type Run struct {
	RunID         string     `json:"run_id"`
	StepKey       string     `json:"step_key"`
	Operation     string     `json:"operation,omitempty"`
	Equipment     string     `json:"equipment,omitempty"`
	Program       string     `json:"program,omitempty"`
	Operator      string     `json:"operator,omitempty"`
	OperatorKnown bool       `json:"operator_known"`
	ReworkOf      string     `json:"rework_of,omitempty"`
	Started       time.Time  `json:"started"`
	Finished      *time.Time `json:"finished,omitempty"`
	StartEventID  string     `json:"start_event_id,omitempty"`
	FinishEventID string     `json:"finish_event_id,omitempty"`
}

// Finding — результат контроля изделия, нужный окну возможного возникновения
// (FR-58): исход, фаза, зоны, зоны и виды дефектов, компоненты.
type Finding struct {
	EventID     string    `json:"event_id"`
	Seq         int64     `json:"seq,omitempty"`
	At          time.Time `json:"at"`
	Outcome     string    `json:"outcome"`
	Phase       string    `json:"phase,omitempty"`
	Method      string    `json:"method,omitempty"`
	RunID       string    `json:"run_id,omitempty"`
	Zones       []string  `json:"zones,omitempty"`
	DefectZones []string  `json:"defect_zones,omitempty"`
	DefectTypes []string  `json:"defect_types,omitempty"`
	Components  []string  `json:"components,omitempty"`
}

// Case — несоответствие изделия (decision.nonconformity.confirmed).
type Case struct {
	NCID       string    `json:"nc_id"`
	EventID    string    `json:"event_id"`
	Seq        int64     `json:"seq,omitempty"`
	At         time.Time `json:"at"`
	DefectType string    `json:"defect_type,omitempty"`
	Severity   string    `json:"severity,omitempty"`
}

// Membership — статус изделия в инциденте по двум осям (FR-62): что известно
// (confirmed | suspect | excluded | unknown) и что делать (observe | check |
// block | release). Пишет только analysis (incident.membership.changed, AD-30).
type Membership struct {
	IncidentID    string `json:"incident_id"`
	ScopeVersion  int    `json:"scope_version"`
	Status        string `json:"status"`
	Action        string `json:"action"`
	ViaAssemblyOf string `json:"via_assembly_of,omitempty"`
	EventID       string `json:"event_id"`
}

// State — состояние модуля analysis в свёртке одного изделия (AD-5): события
// трёх дорожек, выполнения операций, результаты контроля, несоответствия и
// статус в инцидентах. Значение без ссылок; поля экспортируются для хеша
// состояния (Д-22).
type State struct {
	// ItemID — изделие свёртки (из записей входа).
	ItemID    string       `json:"item_id,omitempty"`
	Marks     []Mark       `json:"marks,omitempty"`
	Runs      []Run        `json:"runs,omitempty"`
	Findings  []Finding    `json:"findings,omitempty"`
	Cases     []Case       `json:"cases,omitempty"`
	Incidents []Membership `json:"incidents,omitempty"`
	// Equipment — события оборудования, привязанные к изделию приёмом
	// (edge-агент знал изделие, AD-41): дорожка оборудования без порта.
	Equipment []EquipmentEvent `json:"equipment,omitempty"`
	// Released — изделие выпущено (item.release.recorded): «отгружено» в
	// разбивке области (FR-61).
	Released bool `json:"released,omitempty"`
	// Assembled — изделие собрано или вошло в сборку (item.assembly.recorded, genealogy.link.added).
	Assembled bool `json:"assembled,omitempty"`
}

// run — индекс выполнения по id.
func (s *State) run(id string) int {
	for i := range s.Runs {
		if s.Runs[i].RunID == id {
			return i
		}
	}
	return -1
}

// reduceItem применяет запись входа изделия к состоянию разбора (AD-5):
// реакции в свёртку не входят (AD-3), кроме адресованных изделию записей
// стадии своего модуля (incident.membership.changed) — это вход изделия.
func reduceItem(s State, r kernel.Record) State {
	if s.ItemID == "" && r.ItemID != "" {
		s.ItemID = r.ItemID
	}
	if e, ok := EquipmentEventOf(r); ok {
		s.Equipment = append(slices.Clone(s.Equipment), e)
		return s
	}
	switch r.Type {
	case catalog.OperationRunStarted:
		d, ok := decodeAs[runStartedData](r)
		if !ok || d.OperationRunID == "" {
			return s
		}
		at := r.OccurredAt
		if d.StartedAt != nil {
			at = d.StartedAt.UTC()
		}
		run := Run{RunID: d.OperationRunID, StepKey: d.StepKey, Operation: d.OperationCode, Equipment: deref(d.EquipmentID),
			Program: deref(d.ProgramRef), Operator: deref(d.OperatorID), OperatorKnown: d.OperatorID != nil && *d.OperatorID != "",
			ReworkOf: deref(d.ReworkOf), Started: at, StartEventID: r.EventID}
		s.Runs = slices.Clone(s.Runs)
		if i := s.run(run.RunID); i >= 0 {
			run.Finished, run.FinishEventID = s.Runs[i].Finished, s.Runs[i].FinishEventID
			s.Runs[i] = run
		} else {
			s.Runs = append(s.Runs, run)
		}
		params := map[string]string{"step": d.StepKey}
		if run.Equipment != "" {
			params["equipment"] = run.Equipment
		}
		if run.Program != "" {
			params["program"] = run.Program
		}
		if run.OperatorKnown {
			params["operator"] = run.Operator
		}
		s = addMark(s, r, LanePerson, "run_started", run.RunID, params, nil)
	case catalog.OperationRunFinished:
		d, ok := decodeAs[runFinishedData](r)
		if !ok {
			return s
		}
		at := r.OccurredAt
		if d.FinishedAt != nil {
			at = d.FinishedAt.UTC()
		}
		s.Runs = slices.Clone(s.Runs)
		if i := s.run(d.OperationRunID); i >= 0 {
			s.Runs[i].Finished, s.Runs[i].FinishEventID = &at, r.EventID
		}
		s = addMark(s, r, LanePerson, "run_finished", d.OperationRunID, map[string]string{"completion": d.Completion}, nil)
	case catalog.OperationRunIntervalResolved:
		d, ok := decodeAs[intervalData](r)
		if !ok {
			return s
		}
		s.Runs = slices.Clone(s.Runs)
		if i := s.run(d.OperationRunID); i >= 0 {
			if s.Runs[i].Equipment == "" {
				s.Runs[i].Equipment = deref(d.EquipmentID)
			}
			if !d.IntervalStart.IsZero() {
				s.Runs[i].Started = d.IntervalStart.UTC()
			}
			if d.IntervalEnd != nil {
				end := d.IntervalEnd.UTC()
				s.Runs[i].Finished = &end
			}
		}
	case catalog.InspectionResultRecorded:
		d, ok := decodeAs[inspectionData](r)
		if !ok {
			return s
		}
		f := Finding{EventID: r.EventID, Seq: r.Seq, At: r.OccurredAt, Outcome: d.Outcome, Phase: d.Phase, Method: d.Method,
			RunID: deref(d.OperationRunID), Zones: slices.Clone(d.ZoneIDs)}
		for _, x := range d.Defects {
			f.DefectZones = appendUnique(f.DefectZones, x.ZoneID)
			f.DefectTypes = appendUnique(f.DefectTypes, deref(x.DefectTypeCode))
			f.Components = appendUnique(f.Components, deref(x.ComponentRef))
		}
		s.Findings = append(slices.Clone(s.Findings), f)
		params := map[string]string{"method": d.Method}
		if d.Phase != "" {
			params["phase"] = d.Phase
		}
		if p := deref(d.InspectionPoint); p != "" {
			params["point"] = p
		}
		if c := deref(d.ConclusionRef); c != "" {
			params["conclusion"] = c
		}
		if len(f.DefectTypes) > 0 {
			params["defect"] = strings.Join(f.DefectTypes, ", ")
		}
		if len(f.DefectZones) > 0 {
			params["zone"] = strings.Join(f.DefectZones, ", ")
		}
		var ev []string
		for _, e := range d.EvidenceRefs {
			if !e.IsIllustration && e.MaterialAddress != "" {
				ev = append(ev, e.MaterialAddress)
			}
		}
		s = addMark(s, r, LaneItem, d.Outcome, f.RunID, params, ev)
	case catalog.OperationMovementSent, catalog.OperationMovementReceived:
		d, _ := decodeAs[movementData](r)
		params := map[string]string{}
		if d.FromLocationID != "" {
			params["from"] = d.FromLocationID
		}
		if d.ToLocationID != "" {
			params["to"] = d.ToLocationID
		}
		s = addMark(s, r, LaneItem, "movement", "", params, nil)
	case catalog.OperatorOverridePerformed, catalog.OperatorDeviationReported, catalog.OperatorModeChanged,
		catalog.OperatorActionObserved, catalog.OperatorCheckSkipped, catalog.OperatorStepConfirmed:
		d, _ := decodeAs[operatorData](r)
		params := map[string]string{}
		if d.OperatorID != nil && *d.OperatorID != "" {
			params["operator"] = *d.OperatorID
		}
		variant := strings.TrimPrefix(string(r.Type), "operator.")
		switch {
		case d.Bypassed != "":
			params["bypassed"] = d.Bypassed
		case d.Observation != "":
			variant = d.Observation
		}
		if d.Description != "" {
			params["note"] = d.Description
		}
		if d.Reason != nil && d.Reason.Text != "" {
			params["note"] = d.Reason.Text
		}
		s = addMark(s, r, LanePerson, variant, deref(d.OperationRunID), params, nil)
	case catalog.DecisionNonconformityConfirmed:
		d, ok := decodeAs[ncConfirmedData](r)
		if !ok || d.NcID == "" {
			return s
		}
		c := Case{NCID: d.NcID, EventID: r.EventID, Seq: r.Seq, At: r.OccurredAt, DefectType: deref(d.DefectTypeCode), Severity: d.Severity}
		s.Cases = slices.Clone(s.Cases)
		for i := range s.Cases {
			if s.Cases[i].NCID == c.NCID {
				s.Cases[i] = c
				return s
			}
		}
		s.Cases = append(s.Cases, c)
	case catalog.ItemReleaseRecorded:
		s.Released = true
	case catalog.ItemAssemblyRecorded, catalog.GenealogyLinkAdded:
		s.Assembled = true
	case catalog.IncidentMembershipChanged:
		d, ok := decodeAs[membershipData](r)
		if !ok {
			return s
		}
		m := Membership{IncidentID: d.IncidentID, ScopeVersion: d.ScopeVersion, Status: d.Status, Action: d.Action, ViaAssemblyOf: deref(d.ViaAssemblyOf), EventID: r.EventID}
		s.Incidents = slices.Clone(s.Incidents)
		for i := range s.Incidents {
			if s.Incidents[i].IncidentID == m.IncidentID {
				if m.ScopeVersion >= s.Incidents[i].ScopeVersion {
					s.Incidents[i] = m
				}
				return s
			}
		}
		s.Incidents = append(s.Incidents, m)
	}
	return s
}

// membershipData — incident.membership.changed.
type membershipData struct {
	IncidentID    string  `json:"incident_id"`
	ScopeVersion  int     `json:"scope_version"`
	Status        string  `json:"status"`
	Action        string  `json:"action"`
	ViaAssemblyOf *string `json:"via_assembly_of"`
}

func addMark(s State, r kernel.Record, lane, variant, runID string, params map[string]string, ev []string) State {
	if len(params) == 0 {
		params = nil
	}
	m := Mark{EventID: r.EventID, EventType: string(r.Type), Variant: variant, Lane: lane, OccurredAt: r.OccurredAt,
		Seq: r.Seq, SourceKind: r.SourceKind, RunID: runID, Params: params, Evidence: ev}
	s.Marks = append(slices.Clone(s.Marks), m)
	return s
}

func appendUnique(xs []string, v string) []string {
	if v == "" || slices.Contains(xs, v) {
		return xs
	}
	return append(xs, v)
}

// EquipmentEvent — событие временной линии оборудования для разбора (FR-147,
// FR-148): отклонение режима, сводка цикла, состояние, смена инструмента или
// программы. Временной линией владеет межизделийная стадия (AD-42), фактами —
// machinelogs (эпик 23); разбор получает их через порт.
type EquipmentEvent struct {
	EventID     string            `json:"event_id"`
	EventType   string            `json:"event_type"`
	Variant     string            `json:"variant,omitempty"`
	EquipmentID string            `json:"equipment_id"`
	OccurredAt  time.Time         `json:"occurred_at"`
	EndedAt     *time.Time        `json:"ended_at,omitempty"`
	Seq         int64             `json:"seq,omitempty"`
	SourceKind  string            `json:"source_kind,omitempty"`
	Params      map[string]string `json:"params,omitempty"`
	// Deviation — отклонение режима: выход за уставку, перегрузка, авария,
	// ручное изменение режима, внеплановая смена программы, цикл вне уставки.
	Deviation bool `json:"deviation,omitempty"`
	// ToolID, FixtureID — для смены инструмента.
	ToolID    string `json:"tool_id,omitempty"`
	FixtureID string `json:"fixture_id,omitempty"`
}

// EquipmentEventOf — событие оборудования из записи журнала семейства
// equipment; false — запись не про оборудование.
func EquipmentEventOf(r kernel.Record) (EquipmentEvent, bool) {
	if family(r.Type) != "equipment" {
		return EquipmentEvent{}, false
	}
	d, ok := decodeAs[equipmentData](r)
	if !ok || d.EquipmentID == "" {
		return EquipmentEvent{}, false
	}
	e := EquipmentEvent{EventID: r.EventID, EventType: string(r.Type), EquipmentID: d.EquipmentID, OccurredAt: r.OccurredAt,
		Seq: r.Seq, SourceKind: r.SourceKind}
	params := map[string]string{}
	switch r.Type {
	case catalog.EquipmentDeviationDetected:
		e.Variant, e.Deviation = d.DeviationKind, true
		if d.StartedAt != nil {
			e.OccurredAt = d.StartedAt.UTC()
		}
		if d.EndedAt != nil {
			end := d.EndedAt.UTC()
			e.EndedAt = &end
		}
		if d.Parameter != nil {
			params["parameter"] = *d.Parameter
		}
		if v := fmtMeasurement(d.Value); v != "" {
			params["value"] = v
		}
		if v := fmtTolerance(d.Setpoint); v != "" {
			params["setpoint"] = v
		}
	case catalog.EquipmentCycleSummarized:
		e.Variant = "cycle"
		if d.WindowStart != nil {
			e.OccurredAt = d.WindowStart.UTC()
		}
		if d.WindowEnd != nil {
			end := d.WindowEnd.UTC()
			e.EndedAt = &end
		}
		for _, p := range d.Parameters {
			if p.OutOfSetpointMs != nil && *p.OutOfSetpointMs > 0 {
				e.Deviation = true
			}
			if _, done := params["parameter"]; done && !e.Deviation {
				continue
			}
			params["parameter"] = p.Parameter
			if v := fmtMeasurement(p.Max); v != "" {
				params["value"] = v
			}
			if v := fmtTolerance(p.Setpoint); v != "" {
				params["setpoint"] = v
			}
		}
	case catalog.EquipmentStateChanged:
		e.Variant = d.Condition
		if e.Variant == "" {
			e.Variant = d.Execution
		}
		e.Deviation = d.Condition == "fault"
		if d.ControllerMode != "" {
			params["mode"] = d.ControllerMode
		}
	case catalog.EquipmentToolChanged:
		e.Variant, e.ToolID, e.FixtureID = "tool_changed", d.ToolID, deref(d.FixtureID)
		params["tool"] = d.ToolID
	case catalog.EquipmentProgramChanged:
		e.Variant = "program_changed"
		params["program"] = d.ProgramRef
		if d.ProgramRevision != "" {
			params["revision"] = d.ProgramRevision
		}
		e.Deviation = d.Planned != nil && !*d.Planned
	default:
		e.Variant = strings.TrimPrefix(string(r.Type), "equipment.")
	}
	if len(params) > 0 {
		e.Params = params
	}
	return e, true
}
