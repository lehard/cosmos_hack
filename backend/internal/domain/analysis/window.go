package analysis

import (
	"maps"
	"slices"

	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
)

// onViolationWindow — окно нарушения режима специального процесса (FR-151,
// FR-61; решение Д-40): machinelogs выдаёт equipment.violation.window_resolved
// в том же проходе стадии, что и analysis (crossitem.Pass). Изделия выполнений
// окна — «под подозрением» в области: открытый инцидент по тому же
// оборудованию и шагу расширяется новой версией (правило только добавляет,
// AD-27), окно инцидента растёт до окна нарушения; открытого нет — окно
// открывает инцидент по оборудованию.
func (s StageState) onViolationWindow(r kernel.Record) (StageState, []kernel.Addressed) {
	d, err := kernel.Decode[ev.EquipmentViolationWindowResolvedV1](r)
	items := sortedItems(d.AffectedItemIds)
	if err != nil || len(items) == 0 || d.EquipmentID == "" {
		return s, nil
	}
	eq, step := string(d.EquipmentID), ""
	if d.StepKey != nil {
		step = string(*d.StepKey)
	}
	start, end := d.WindowStart.Time(), d.WindowEnd.Time()
	var out []kernel.Addressed
	matched := false
	for _, id := range s.openIncidents() {
		inc := s.Incidents[id]
		if inc.Factor != FactorMachine || inc.Value != eq || step != "" && inc.StepKey != step {
			continue
		}
		matched = true
		inc.Windows = appendUnique(slices.Clone(inc.Windows), r.EventID)
		s.Incidents[id] = inc
		var changed []string
		for _, it := range items {
			changed = append(changed, inc.AutoAdd(it, StatusSuspect, "")...)
		}
		if len(changed) == 0 {
			continue
		}
		if start.Before(inc.WindowStart) {
			inc.WindowStart = start
		}
		if end.After(inc.WindowEnd) {
			inc.WindowEnd = end
		}
		out = append(out, s.version(&inc, ChangeComputed, "", changed, nil, []string{r.EventID}, r)...)
		s.Incidents[id] = inc
	}
	if matched {
		return s, out
	}
	inc := Incident{ID: IncidentID(r.EventID), Label: "Нарушение режима " + eq, Factor: FactorMachine, Value: eq, StepKey: step,
		Factors: []FactorRef{{FactorMachine, eq}}, WindowStart: start, WindowEnd: end, Members: map[string]Member{},
		Windows: []string{r.EventID}}
	for _, it := range items {
		inc.Members[it] = Member{Status: StatusSuspect, Action: ActionCheck}
	}
	opened, err := kernel.NewAddressed(Module, catalog.IncidentIncidentOpened, "incident:"+inc.ID, inc.ID, incidentOpened(inc, r.EventID), r)
	if err != nil {
		return s, nil
	}
	out = append(out, opened)
	out = append(out, s.version(&inc, ChangeComputed, "", slices.Sorted(maps.Keys(inc.Members)), nil, []string{r.EventID}, r)...)
	s.Incidents[inc.ID] = inc
	return s, out
}

// onWindowNC — несоответствие окна нарушения спецпроцесса
// (decision.nonconformity.registered — функция-намерение nonconformity в
// шаге machinelogs того же прохода, FR-151; стык эпиков 21 и 22, эпик 16):
// несоответствие добавляется в инцидент, который это окно открыло или
// расширило, — вход разбора обстоятельств и гипотез. Область не меняется:
// изделие уже в ней «под подозрением» по окну.
func (s StageState) onWindowNC(r kernel.Record) (StageState, []kernel.Addressed) {
	d, err := kernel.Decode[ev.DecisionNonconformityRegisteredV1](r)
	if err != nil || d.NcID == "" || d.ViolationWindowEventID == "" {
		return s, nil
	}
	nc, win := string(d.NcID), string(d.ViolationWindowEventID)
	for _, id := range slices.Sorted(maps.Keys(s.Incidents)) {
		inc := s.Incidents[id]
		if inc.Closed || !slices.Contains(inc.Windows, win) || slices.Contains(inc.NCs, nc) {
			continue
		}
		inc.NCs = appendUnique(slices.Clone(inc.NCs), nc)
		s.Incidents[id] = inc
	}
	return s, nil
}
