package item

import (
	"encoding/json"
	"slices"
	"strings"
	"time"

	"ant/internal/domain/kernel"
)

// Применение записей к паспорту изделия (FR-42…FR-46, AD-16). Каждая функция
// — чистое преобразование State: срезы копируются перед изменением.

func (s State) registered(r kernel.Record, env Env) State {
	d, ok := decode[struct {
		ItemID             string   `json:"item_id"`
		ItemTypeID         string   `json:"item_type_id"`
		ItemRevision       string   `json:"item_revision"`
		OrderID            string   `json:"order_id"`
		LotIDs             []string `json:"lot_ids"`
		ProcessVersionHash string   `json:"process_version_hash"`
		NormativeRev       string   `json:"normative_rev"`
		EntryStepKey       string   `json:"entry_step_key"`
		IsAssembly         bool     `json:"is_assembly"`
		SplitFrom          string   `json:"split_from"`
	}](r)
	if !ok || s.Registered {
		return s
	}
	s.Registered, s.RegisteredAt, s.RegisteredEventID = true, r.OccurredAt, r.EventID
	s.ItemTypeID, s.ItemRevision, s.OrderID = d.ItemTypeID, d.ItemRevision, d.OrderID
	s.ProcessVersion, s.NormativeRev, s.EntryStepKey = d.ProcessVersionHash, d.NormativeRev, d.EntryStepKey
	s.IsAssembly, s.SplitFrom = d.IsAssembly, d.SplitFrom
	for _, l := range d.LotIDs {
		s.LotIDs = appendOnce(s.LotIDs, l)
	}
	// Внутренний ID — носитель с регистрации (AD-16): поиск по номеру детали.
	id := d.ItemID
	if id == "" {
		id = r.ItemID
	}
	s.Carriers = append(slices.Clone(s.Carriers), Carrier{Type: "internal_id", Value: id, State: CarrierApplied, AppliedAt: r.OccurredAt, EventID: r.EventID})
	// Зоны по КД типа изделия (FR-46).
	if t, ok := env.Types[d.ItemTypeID]; ok {
		for _, z := range t.Zones {
			if !slices.ContainsFunc(s.Zones, func(x Zone) bool { return x.ZoneID == z.ID }) {
				s.Zones = append(s.Zones, Zone{ZoneID: z.ID, Title: z.Name, Status: ZoneNotInspected})
			}
		}
	}
	return s
}

func (s State) carrierApplied(r kernel.Record) State {
	d, ok := decode[struct {
		CarrierType   string  `json:"carrier_type"`
		Value         string  `json:"value"`
		ZoneID        string  `json:"zone_id"`
		IsTemporary   bool    `json:"is_temporary"`
		ReplacesValue *string `json:"replaces_value"`
		LotID         string  `json:"lot_id"`
	}](r)
	if !ok || d.Value == "" {
		return s
	}
	s.Carriers = slices.Clone(s.Carriers)
	if d.ReplacesValue != nil && *d.ReplacesValue != "" {
		for i := range s.Carriers {
			if s.Carriers[i].Value == *d.ReplacesValue && s.Carriers[i].Active() {
				at := r.OccurredAt
				s.Carriers[i].State, s.Carriers[i].RemovedAt = CarrierReplaced, &at
			}
		}
	}
	if slices.ContainsFunc(s.Carriers, func(c Carrier) bool { return c.Type == d.CarrierType && c.Value == d.Value && c.Active() }) {
		return s
	}
	s.Carriers = append(s.Carriers, Carrier{Type: d.CarrierType, Value: d.Value, ZoneID: d.ZoneID, Temporary: d.IsTemporary,
		State: CarrierApplied, LotID: d.LotID, AppliedAt: r.OccurredAt, EventID: r.EventID})
	s.LotIDs = appendOnce(s.LotIDs, d.LotID)
	return s
}

// carrierVerified — результат считывания: нечитаем или не совпал —
// «идентификация под сомнением» (AD-16).
func (s State) carrierVerified(r kernel.Record) State {
	d, ok := decode[struct {
		CarrierType string `json:"carrier_type"`
		Value       string `json:"value"`
		ReadOutcome string `json:"read_outcome"`
	}](r)
	if !ok {
		return s
	}
	s.Carriers = slices.Clone(s.Carriers)
	for i := range s.Carriers {
		c := &s.Carriers[i]
		if !c.Active() || c.Type != d.CarrierType || (d.Value != "" && d.ReadOutcome == "readable" && c.Value != d.Value) {
			continue
		}
		switch d.ReadOutcome {
		case "readable":
			at := r.OccurredAt
			c.State, c.VerifiedAt = CarrierVerified, &at
		case "unreadable":
			c.State = CarrierUnreadable
		}
	}
	switch d.ReadOutcome {
	case "unreadable":
		s = s.question(r, "carrier_unreadable", nil, "")
	case "mismatch":
		s = s.question(r, "carrier_mismatch", nil, "")
	}
	return s
}

// carrierRemoved — снят носитель; если у изделия не осталось ни одного
// физического носителя, идентификация утрачена (AD-16).
func (s State) carrierRemoved(r kernel.Record) State {
	d, ok := decode[struct {
		CarrierType string `json:"carrier_type"`
		Value       string `json:"value"`
	}](r)
	if !ok {
		return s
	}
	s.Carriers = slices.Clone(s.Carriers)
	hadPhysical := false
	for i := range s.Carriers {
		c := &s.Carriers[i]
		if c.Active() && c.Physical() {
			hadPhysical = true
		}
		if c.Active() && c.Type == d.CarrierType && c.Value == d.Value {
			at := r.OccurredAt
			c.State, c.RemovedAt = CarrierRemoved, &at
		}
	}
	left := slices.ContainsFunc(s.Carriers, func(c Carrier) bool { return c.Active() && c.Physical() })
	if hadPhysical && !left && !s.Released {
		s = s.question(r, "carrier_missing", nil, "")
	}
	return s
}

// question — открыть «идентификация под сомнением» по записи r.
func (s State) question(r kernel.Record, cause string, candidates []string, subject string) State {
	key := r.EventID
	if slices.ContainsFunc(s.Questions, func(q Question) bool { return q.Key == key }) {
		return s
	}
	s.Questions = append(slices.Clone(s.Questions), Question{Key: key, Cause: cause, Candidates: slices.Clone(candidates),
		Basis: []string{r.EventID}, At: r.OccurredAt, SubjectEventID: subject})
	return s
}

// identificationConfirmed — повторная идентификация человеком с подписью
// снимает сомнение (AD-16): указанное (по id реакции или записи-причины), а
// если такого нет — все открытые.
func (s State) identificationConfirmed(r kernel.Record) State {
	d, _ := decode[struct {
		QuestionedEventID string `json:"questioned_event_id"`
	}](r)
	s.Questions = slices.Clone(s.Questions)
	hit := false
	for i := range s.Questions {
		q := &s.Questions[i]
		if !q.Closed && s.questionMatches(*q, d.QuestionedEventID) {
			q.Closed, q.ClosedBy, hit = true, r.EventID, true
		}
	}
	if !hit {
		for i := range s.Questions {
			if q := &s.Questions[i]; !q.Closed {
				q.Closed, q.ClosedBy = true, r.EventID
			}
		}
	}
	return s
}

// questionMatches — id совпадает с записью-причиной или с id реакции
// item.identification.questioned этого вопроса (версии 1…9, AD-3).
func (s State) questionMatches(q Question, id string) bool {
	if id == "" {
		return false
	}
	if id == q.Key {
		return true
	}
	re := kernel.Reaction{Slot: QuestionSlot(s.ItemID, q.Key)}
	for v := 1; v < 10; v++ {
		if re.ID(v) == id {
			return true
		}
	}
	return false
}

// QuestionSlot — слот реакции «идентификация под сомнением» (AD-3).
func QuestionSlot(itemID, key string) kernel.Slot {
	return kernel.Slot{RuleID: RuleIdentification, Subject: "item:" + itemID, TriggerKey: key}
}

func (s State) assembly(r kernel.Record, env Env) State {
	d, ok := decode[struct {
		AssemblyItemID  string `json:"assembly_item_id"`
		ComponentItemID string `json:"component_item_id"`
		ComponentLotID  string `json:"component_lot_id"`
		ComponentTypeID string `json:"component_type_id"`
		Quantity        int    `json:"quantity"`
		Position        string `json:"position"`
		BindingMethod   string `json:"binding_method"`
	}](r)
	if !ok || (d.AssemblyItemID != "" && d.AssemblyItemID != s.ItemID) {
		return s
	}
	if slices.ContainsFunc(s.Components, func(c Component) bool { return c.EventID == r.EventID }) {
		return s
	}
	s.Components = append(slices.Clone(s.Components), Component{ItemID: d.ComponentItemID, LotID: d.ComponentLotID, TypeID: d.ComponentTypeID,
		Position: d.Position, Quantity: d.Quantity, Method: d.BindingMethod, EventID: r.EventID, At: r.OccurredAt})
	// Связь по КД, закрывающая доступ к зонам (FR-20, FR-46): соединение
	// установлено — доступ к зонам закрыт.
	if d.Position != "" {
		for _, l := range env.Types[s.ItemTypeID].Links {
			if l.ID != d.Position {
				continue
			}
			s.Zones = slices.Clone(s.Zones)
			for _, z := range l.ClosesAccessTo {
				if i := s.zone(z); i >= 0 && !s.Zones[i].Closed && s.Zones[i].OpenIntervention == "" {
					s.Zones[i].Closed, s.Zones[i].ClosedBy = true, l.ID
				}
			}
		}
	}
	return s
}

func (s State) zone(id string) int {
	return slices.IndexFunc(s.Zones, func(z Zone) bool { return z.ZoneID == id })
}

// ensureZone — зона, упомянутая записью, но не описанная в Env.
func (s *State) ensureZone(id string) int {
	if i := s.zone(id); i >= 0 {
		return i
	}
	s.Zones = append(s.Zones, Zone{ZoneID: id, Status: ZoneNotInspected})
	return len(s.Zones) - 1
}

// inspected — результат контроля по зонам: зона проверена (FR-46); результат
// в открытом вмешательстве не снимает «устарела» — это делает закрытие.
func (s State) inspected(r kernel.Record, _ Env) State {
	d, ok := decode[struct {
		ZoneIDs []string `json:"zone_ids"`
		Outcome string   `json:"outcome"`
	}](r)
	if !ok || len(d.ZoneIDs) == 0 || d.Outcome == "unable_to_assess" {
		return s
	}
	s.Zones = slices.Clone(s.Zones)
	for _, z := range d.ZoneIDs {
		i := s.ensureZone(z)
		s.Zones[i].LastInspection = r.EventID
		if s.Zones[i].OpenIntervention == "" {
			s.Zones[i].Status = ZoneInspected
		}
	}
	return s
}

func (s State) interventionOpened(r kernel.Record, _ Env) State {
	d, ok := decode[struct {
		InterventionID    string   `json:"intervention_id"`
		ZoneIDs           []string `json:"zone_ids"`
		RemovedComponents []string `json:"removed_components"`
		Purpose           struct {
			Text string `json:"text"`
		} `json:"purpose"`
	}](r)
	if !ok || d.InterventionID == "" || slices.ContainsFunc(s.Interventions, func(x Intervention) bool { return x.ID == d.InterventionID }) {
		return s
	}
	iv := Intervention{ID: d.InterventionID, Zones: slices.Clone(d.ZoneIDs), Removed: slices.Clone(d.RemovedComponents), Purpose: d.Purpose.Text,
		OpenedAt: r.OccurredAt, OpenedBy: r.Actor, EventID: r.EventID}
	s.Zones = slices.Clone(s.Zones)
	for _, z := range d.ZoneIDs {
		i := s.ensureZone(z)
		if s.Zones[i].Closed {
			iv.WasClosed = append(iv.WasClosed, z)
		}
		// Вскрыли: зона устарела после вмешательства, доступ открыт (FR-21, FR-46).
		s.Zones[i].Status, s.Zones[i].Closed, s.Zones[i].OpenIntervention = ZoneStale, false, d.InterventionID
	}
	s.Interventions = append(slices.Clone(s.Interventions), iv)
	return s
}

func (s State) interventionClosed(r kernel.Record) State {
	d, ok := decode[struct {
		InterventionID  string   `json:"intervention_id"`
		RecheckEventIDs []string `json:"recheck_event_ids"`
		RetestRequired  bool     `json:"retest_required"`
	}](r)
	if !ok {
		return s
	}
	s.Interventions = slices.Clone(s.Interventions)
	for i := range s.Interventions {
		iv := &s.Interventions[i]
		if iv.ID != d.InterventionID || iv.ClosedAt != nil {
			continue
		}
		at := r.OccurredAt
		iv.ClosedAt, iv.CloseEvent, iv.Retest = &at, r.EventID, d.RetestRequired
		s.Zones = slices.Clone(s.Zones)
		for _, z := range iv.Zones {
			if j := s.zone(z); j >= 0 {
				s.Zones[j].Status, s.Zones[j].OpenIntervention = ZoneInspected, ""
				if slices.Contains(iv.WasClosed, z) {
					s.Zones[j].Closed = true
				}
			}
		}
	}
	return s
}

// OpenInterventions — вмешательства без закрытия.
func (s State) OpenInterventions() []Intervention {
	var out []Intervention
	for _, iv := range s.Interventions {
		if iv.ClosedAt == nil {
			out = append(out, iv)
		}
	}
	return out
}

func (s State) run(r kernel.Record) State {
	d, ok := decode[struct {
		OperationRunID string `json:"operation_run_id"`
		StepKey        string `json:"step_key"`
	}](r)
	if !ok || d.OperationRunID == "" {
		return s
	}
	if strings.HasSuffix(string(r.Type), ".started") {
		s.StepKey = d.StepKey
		s.Running = appendOnce(s.Running, d.OperationRunID)
		return s
	}
	s.Running = slices.DeleteFunc(slices.Clone(s.Running), func(x string) bool { return x == d.OperationRunID })
	return s
}

func (s State) link(r kernel.Record) State {
	d, ok := decode[struct {
		ParentItemID string `json:"parent_item_id"`
		ChildItemID  string `json:"child_item_id"`
		LotID        string `json:"lot_id"`
		GroupID      string `json:"group_id"`
		Position     string `json:"position"`
		Relation     string `json:"relation"`
		BasisEventID string `json:"basis_event_id"`
		Inherited    bool   `json:"inherited"`
	}](r)
	if !ok || slices.ContainsFunc(s.Links, func(l Link) bool { return l.EventID == r.EventID }) {
		return s
	}
	s.Links = append(slices.Clone(s.Links), Link{Relation: d.Relation, ParentID: d.ParentItemID, ChildID: d.ChildItemID, LotID: d.LotID,
		GroupID: d.GroupID, Position: d.Position, Inherited: d.Inherited, Basis: d.BasisEventID, EventID: r.EventID, At: r.OccurredAt})
	if d.Relation == "made_from_lot" && d.ChildItemID == s.ItemID {
		s.LotIDs = appendOnce(s.LotIDs, d.LotID)
	}
	return s
}

func (s State) witness(r kernel.Record) State {
	d, ok := decode[struct {
		GroupID           string `json:"group_id"`
		GroupKind         string `json:"group_kind"`
		WitnessItemID     string `json:"witness_item_id"`
		InspectionEventID string `json:"inspection_event_id"`
		Outcome           string `json:"outcome"`
		Method            string `json:"method"`
		ConclusionRef     string `json:"conclusion_ref"`
	}](r)
	if !ok || slices.ContainsFunc(s.Witnesses, func(w Witness) bool { return w.EventID == r.EventID }) {
		return s
	}
	s.Witnesses = append(slices.Clone(s.Witnesses), Witness{GroupID: d.GroupID, GroupKind: d.GroupKind, WitnessID: d.WitnessItemID,
		InspectionID: d.InspectionEventID, Outcome: d.Outcome, Method: d.Method, Conclusion: d.ConclusionRef, EventID: r.EventID, At: r.OccurredAt})
	return s
}

func (s State) hold(r kernel.Record) State {
	d, ok := decode[struct {
		Level         string   `json:"level"`
		Source        string   `json:"source"`
		SourceEventID string   `json:"source_event_id"`
		LotID         string   `json:"lot_id"`
		SourceItemID  string   `json:"source_item_id"`
		Path          []string `json:"path"`
		Released      bool     `json:"released"`
	}](r)
	if !ok {
		return s
	}
	s.Holds = slices.Clone(s.Holds)
	if i := slices.IndexFunc(s.Holds, func(h Hold) bool { return h.SourceEventID == d.SourceEventID }); i >= 0 {
		s.Holds[i].Released = d.Released
		return s
	}
	s.Holds = append(s.Holds, Hold{Level: d.Level, Source: d.Source, SourceEventID: d.SourceEventID, LotID: d.LotID,
		SourceItemID: d.SourceItemID, Path: d.Path, Released: d.Released, EventID: r.EventID})
	return s
}

// binding — привязка события без изделия (AD-41): однозначная — событие
// изделия; неоднозначная — кандидат и «идентификация под сомнением»;
// перепривязка к другому — событие больше не учитывается у этого изделия и
// сомнение по нему снимается.
func (s State) binding(r kernel.Record) State {
	d, ok := decode[struct {
		SubjectEventID     string   `json:"subject_event_id"`
		ItemID             string   `json:"item_id"`
		Candidates         []string `json:"candidates"`
		BindingBasis       string   `json:"binding_basis"`
		BindingReliability string   `json:"binding_reliability"`
		CarrierRef         string   `json:"carrier_ref"`
		Subject            *struct {
			EventType  string          `json:"event_type"`
			OccurredAt string          `json:"occurred_at"`
			Data       json.RawMessage `json:"data"`
		} `json:"subject"`
	}](r)
	if !ok || d.SubjectEventID == "" {
		return s
	}
	b := Binding{SubjectEventID: d.SubjectEventID, Basis: d.BindingBasis, Reliability: d.BindingReliability, Candidates: slices.Clone(d.Candidates),
		Mine: d.ItemID == s.ItemID && d.ItemID != "", BoundTo: d.ItemID, CarrierRef: d.CarrierRef, EventID: r.EventID}
	if d.Subject != nil {
		b.SubjectType, b.SubjectData = d.Subject.EventType, d.Subject.Data
		if t, err := time.Parse(time.RFC3339Nano, d.Subject.OccurredAt); err == nil {
			t = t.UTC()
			b.SubjectAt = &t
		}
	}
	s.Bindings = slices.Clone(s.Bindings)
	if i := slices.IndexFunc(s.Bindings, func(x Binding) bool { return x.SubjectEventID == d.SubjectEventID }); i >= 0 {
		if b.SubjectType == "" {
			b.SubjectType, b.SubjectData, b.SubjectAt = s.Bindings[i].SubjectType, s.Bindings[i].SubjectData, s.Bindings[i].SubjectAt
		}
		s.Bindings[i] = b
	} else {
		s.Bindings = append(s.Bindings, b)
	}
	switch {
	case d.ItemID == "" && len(d.Candidates) > 1:
		s = s.question(r, "ambiguous_binding", d.Candidates, d.SubjectEventID)
	case d.ItemID != "":
		// Привязку определили (человек или однозначный носитель): сомнение по этому событию снято.
		s.Questions = slices.Clone(s.Questions)
		for i := range s.Questions {
			if q := &s.Questions[i]; !q.Closed && q.SubjectEventID == d.SubjectEventID && d.BindingBasis == "manual" {
				q.Closed, q.ClosedBy = true, r.EventID
			}
		}
	}
	return s
}

// BoundEvents — события без изделия, привязанные к этому изделию (однозначно
// или человеком): запись с содержимым для паспорта и свёрток модулей.
func (s State) BoundEvents() []Binding {
	var out []Binding
	for _, b := range s.Bindings {
		if b.Mine {
			out = append(out, b)
		}
	}
	return out
}

func appendOnce(xs []string, x string) []string {
	if x == "" || slices.Contains(xs, x) {
		return xs
	}
	return append(slices.Clone(xs), x)
}
