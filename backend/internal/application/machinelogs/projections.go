package machinelogs

import (
	"encoding/json"
	"slices"
	"time"

	engineapp "ant/internal/application/engine"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/domain/engine"
	"ant/internal/domain/kernel"
	ml "ant/internal/domain/machinelogs"
)

// Проекции модуля machinelogs (AD-45; писатель — machinelogs). Пишутся
// только эффектами в транзакции Append движка: проекция изделия — воркером
// по итогу свёртки, глобальные — ролью projector; пересобираются `ant
// rebuild`. Ключи содержат прогон сценария (AD-38): изделия и оборудование
// двух прогонов не смешиваются.
const (
	// ProjectionItemRuns — профили выполнения операций изделия (FR-148): итог
	// свёртки изделия (State модуля), ключ — item_id.
	ProjectionItemRuns = "machinelogs.item_runs"
	// ProjectionRunIndex — выполнение операции → изделие, оборудование, шаг
	// (ключ ‹прогон›|‹operation_run_id›).
	ProjectionRunIndex = "machinelogs.run_index"
	// ProjectionEquipment — состояние оборудования (ключ ‹прогон›|‹equipment_id›)
	// и перечень оборудования прогона (ключ ‹прогон›|*).
	ProjectionEquipment = "machinelogs.equipment"
	// ProjectionTimeline — журнал оборудования по суткам (ключ
	// ‹прогон›|‹equipment_id›|ГГГГ-ММ-ДД): четыре слоя, сводки, не телеметрия.
	ProjectionTimeline = "machinelogs.timeline"
	// ProjectionViolations — окна нарушений специального процесса прогона
	// (ключ ‹прогон›) и их несоответствия (FR-151).
	ProjectionViolations = "machinelogs.violations"
)

// indexKey — ключ перечня оборудования прогона.
const indexKey = "*"

func key(parts ...string) string {
	out := parts[0]
	for _, p := range parts[1:] {
		out += "|" + p
	}
	return out
}

func day(t time.Time) string { return t.UTC().Format("2006-01-02") }

// ItemRuns — значение проекции machinelogs.item_runs.
type ItemRuns struct {
	ItemID string       `json:"item_id"`
	Runs   []ml.Profile `json:"runs"`
}

// RunRef — значение проекции machinelogs.run_index.
type RunRef struct {
	ItemID      string `json:"item_id"`
	EquipmentID string `json:"equipment_id,omitempty"`
	StepKey     string `json:"step_key,omitempty"`
}

// EquipmentIndex — перечень оборудования прогона.
type EquipmentIndex struct {
	IDs []string `json:"ids"`
}

// DayTimeline — события оборудования за сутки (с исходными данными — для
// состояния на момент, AD-22).
type DayTimeline struct {
	Events []ml.Event `json:"events"`
}

// ViolationNC — несоответствие окна нарушения (FR-151).
type ViolationNC struct {
	NcID           string `json:"nc_id"`
	ItemID         string `json:"item_id"`
	OperationRunID string `json:"operation_run_id"`
}

// ViolationRow — окно нарушения в проекции.
type ViolationRow struct {
	EventID      string        `json:"event_id"`
	EquipmentID  string        `json:"equipment_id"`
	StepKey      string        `json:"step_key,omitempty"`
	Start        time.Time     `json:"window_start"`
	End          time.Time     `json:"window_end"`
	DeviationIDs []string      `json:"deviation_event_ids"`
	RunIDs       []string      `json:"operation_run_ids"`
	ItemIDs      []string      `json:"item_ids"`
	NCs          []ViolationNC `json:"nonconformities"`
	OccurredAt   time.Time     `json:"occurred_at"`
	RecordedAt   time.Time     `json:"recorded_at"`
}

// Violations — значение проекции machinelogs.violations.
type Violations struct {
	Windows []ViolationRow `json:"windows"`
}

// RegisterProjections регистрирует проекции machinelogs в реестре движка
// (cmd/ant/engine.go, engineRegistry). env — нормативная часть (шаги-
// специальные процессы) для состояния оборудования.
func RegisterProjections(reg *engineapp.Registry, env ml.Env) error {
	if err := reg.AddItem(engineapp.ItemProjection{Name: ProjectionItemRuns, Writer: ml.Module, View: itemRunsView}); err != nil {
		return err
	}
	for _, g := range []engineapp.GlobalProjection{
		{Name: ProjectionRunIndex, Writer: ml.Module, Keys: runIndexKeys, Step: runIndexStep},
		{Name: ProjectionEquipment, Writer: ml.Module, Keys: equipmentKeys, Step: equipmentStep(env), Entity: equipmentEntity},
		{Name: ProjectionTimeline, Writer: ml.Module, Keys: timelineKeys, Step: timelineStep},
		{Name: ProjectionViolations, Writer: ml.Module, Keys: violationKeys, Step: violationStep},
	} {
		if err := reg.AddGlobal(g); err != nil {
			return err
		}
	}
	return nil
}

func itemRunsView(itemID string, s engine.Snapshot, _ []kernel.Reaction) (any, error) {
	ps := s.Machinelogs.Profiles()
	if len(ps) == 0 {
		return nil, nil
	}
	return ItemRuns{ItemID: itemID, Runs: ps}, nil
}

func decode[T any](prev json.RawMessage) (T, error) {
	var v T
	if len(prev) == 0 {
		return v, nil
	}
	err := json.Unmarshal(prev, &v)
	return v, err
}

func runIndexKeys(r kernel.Record) []string {
	if r.Type != catalog.OperationRunStarted && r.Type != catalog.OperationRunIntervalResolved {
		return nil
	}
	if id := ml.RunID(r); id != "" {
		return []string{key(r.RunID, id)}
	}
	return nil
}

func runIndexStep(_ string, prev json.RawMessage, r kernel.Record) (json.RawMessage, error) {
	v, err := decode[RunRef](prev)
	if err != nil {
		return nil, err
	}
	var d struct {
		StepKey     string `json:"step_key"`
		EquipmentID string `json:"equipment_id"`
	}
	if err := json.Unmarshal(r.Data, &d); err != nil {
		return nil, err
	}
	if r.ItemID != "" {
		v.ItemID = r.ItemID
	}
	if d.StepKey != "" {
		v.StepKey = d.StepKey
	}
	if d.EquipmentID != "" {
		v.EquipmentID = d.EquipmentID
	}
	return json.Marshal(v)
}

func equipmentKeys(r kernel.Record) []string {
	if eid := ml.EquipmentOf(r); eid != "" {
		return []string{key(r.RunID, eid), key(r.RunID, indexKey)}
	}
	return nil
}

func equipmentStep(env ml.Env) func(string, json.RawMessage, kernel.Record) (json.RawMessage, error) {
	return func(k string, prev json.RawMessage, r kernel.Record) (json.RawMessage, error) {
		if k == key(r.RunID, indexKey) {
			v, err := decode[EquipmentIndex](prev)
			if err != nil {
				return nil, err
			}
			if eid := ml.EquipmentOf(r); !slices.Contains(v.IDs, eid) {
				v.IDs = append(v.IDs, eid)
				slices.Sort(v.IDs)
			}
			return json.Marshal(v)
		}
		v, err := decode[ml.Status](prev)
		if err != nil {
			return nil, err
		}
		return json.Marshal(ml.ApplyStatus(v, env, r))
	}
}

func equipmentEntity(k string) (platform.EntityKind, string, bool) {
	eid := splitLast(k)
	if eid == indexKey {
		return "", "", false
	}
	return platform.EntityEquipment, eid, true
}

// splitLast — последний сегмент ключа «a|b|c».
func splitLast(k string) string {
	for i := len(k) - 1; i >= 0; i-- {
		if k[i] == '|' {
			return k[i+1:]
		}
	}
	return k
}

func timelineKeys(r kernel.Record) []string {
	if !ml.IsEquipmentFact(r.Type) {
		return nil
	}
	ev, _, err := ml.ParseEvent(r)
	if err != nil {
		return nil
	}
	return []string{key(r.RunID, ev.EquipmentID, day(ev.Start))}
}

func timelineStep(_ string, prev json.RawMessage, r kernel.Record) (json.RawMessage, error) {
	v, err := decode[DayTimeline](prev)
	if err != nil {
		return nil, err
	}
	ev, _, err := ml.ParseEvent(r)
	if err != nil {
		return nil, err
	}
	if slices.ContainsFunc(v.Events, func(e ml.Event) bool { return e.EventID == ev.EventID }) {
		return prev, nil
	}
	ev.Data = r.Data
	v.Events = append(v.Events, ev)
	slices.SortStableFunc(v.Events, func(a, b ml.Event) int {
		if c := a.Start.Compare(b.Start); c != 0 {
			return c
		}
		if a.EventID < b.EventID {
			return -1
		}
		return 1
	})
	return json.Marshal(v)
}

func violationKeys(r kernel.Record) []string {
	if r.Type == catalog.EquipmentViolationWindowResolved || r.Type == catalog.DecisionNonconformityRegistered {
		return []string{key(r.RunID)}
	}
	return nil
}

func violationStep(_ string, prev json.RawMessage, r kernel.Record) (json.RawMessage, error) {
	v, err := decode[Violations](prev)
	if err != nil {
		return nil, err
	}
	switch r.Type {
	case catalog.EquipmentViolationWindowResolved:
		var d struct {
			EquipmentID string    `json:"equipment_id"`
			StepKey     string    `json:"step_key"`
			Start       time.Time `json:"window_start"`
			End         time.Time `json:"window_end"`
			Deviations  []string  `json:"deviation_event_ids"`
			Runs        []string  `json:"affected_operation_run_ids"`
			Items       []string  `json:"affected_item_ids"`
		}
		if err := json.Unmarshal(r.Data, &d); err != nil {
			return nil, err
		}
		i := slices.IndexFunc(v.Windows, func(w ViolationRow) bool { return w.EventID == r.EventID })
		row := ViolationRow{EventID: r.EventID, NCs: []ViolationNC{}}
		if i >= 0 {
			row = v.Windows[i]
		}
		row.EquipmentID, row.StepKey, row.Start, row.End = d.EquipmentID, d.StepKey, d.Start.UTC(), d.End.UTC()
		row.DeviationIDs, row.OccurredAt, row.RecordedAt = d.Deviations, r.OccurredAt.UTC(), r.RecordedAt.UTC()
		for _, x := range d.Runs {
			row.RunIDs = addStr(row.RunIDs, x)
		}
		for _, x := range d.Items {
			row.ItemIDs = addStr(row.ItemIDs, x)
		}
		if i >= 0 {
			v.Windows[i] = row
		} else {
			v.Windows = append(v.Windows, row)
		}
	case catalog.DecisionNonconformityRegistered:
		var d struct {
			NcID   string `json:"nc_id"`
			Window string `json:"violation_window_event_id"`
			Run    string `json:"operation_run_id"`
		}
		if err := json.Unmarshal(r.Data, &d); err != nil {
			return nil, err
		}
		i := slices.IndexFunc(v.Windows, func(w ViolationRow) bool { return w.EventID == d.Window })
		if i < 0 {
			// Несоответствие пришло раньше записи окна (та же пачка стадии) —
			// строка окна создаётся, данные окна дополнит его запись.
			v.Windows = append(v.Windows, ViolationRow{EventID: d.Window, NCs: []ViolationNC{}})
			i = len(v.Windows) - 1
		}
		w := v.Windows[i]
		if !slices.ContainsFunc(w.NCs, func(n ViolationNC) bool { return n.NcID == d.NcID }) {
			w.NCs = append(slices.Clone(w.NCs), ViolationNC{NcID: d.NcID, ItemID: r.ItemID, OperationRunID: d.Run})
		}
		w.RunIDs = addStr(w.RunIDs, d.Run)
		w.ItemIDs = addStr(w.ItemIDs, r.ItemID)
		v.Windows[i] = w
	}
	slices.SortStableFunc(v.Windows, func(a, b ViolationRow) int { return a.Start.Compare(b.Start) })
	return json.Marshal(v)
}

func addStr(xs []string, x string) []string {
	if x == "" || slices.Contains(xs, x) {
		return xs
	}
	out := append(slices.Clone(xs), x)
	slices.Sort(out)
	return out
}
