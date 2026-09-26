package crossitem

import (
	"encoding/json"
	"maps"
	"slices"
	"strings"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/domain/kernel"
	"ant/internal/domain/reference"
)

// Собственные правила стадии (AD-41, AD-42; FR-15, FR-34, FR-45): шаг own
// над Own.Genealogy. Вход — запись стадии; выход — адресованные записи
// (occurred_at — наибольший среди причин, id — AddressedID от ключа):
//
//   - генеалогия: регистрация (партии, разделение 1→N с переносом
//     происхождения), выдача партии, носитель с партией, сборка N→1 —
//     genealogy.link.added обоим изделиям связи;
//   - временные группы и образец-свидетель: результат контроля свидетеля
//     садки — genealogy.witness.propagated каждому изделию группы;
//   - распространение сдерживания: блок партии — всем изделиям партии и
//     собранным из них; блок компонента — вверх по дереву сборки; позднее
//     попавшее в партию или сборку изделие получает действующий блок —
//     genealogy.containment.propagated (ось меняет nonconformity, AD-30);
//     снятие основания — та же запись с released, блок снимает человек (AD-27);
//   - привязка событий без изделия: разрешение носителя по реестру стадии на
//     occurred_at — binding.link.resolved одному изделию или каждому из
//     кандидатов (неоднозначное событие не угадывается); ручная привязка
//     человеком — binding.link.resolved по binding.link.assigned;
//   - изменение справочника с действием в прошлом — reference.change.affects_item
//     изделиям, у которых после даты действия было затронутое (функция
//     владельца типа reference.AffectsItem).

// Поля data входа стадии, которые читает own (локальные структуры: время —
// строкой RFC 3339, чтобы не зависеть от JSON-методов сгенерированного Timestamp).
type (
	registeredData struct {
		ItemID     string   `json:"item_id"`
		ItemTypeID string   `json:"item_type_id"`
		LotIDs     []string `json:"lot_ids"`
		SplitFrom  string   `json:"split_from"`
	}
	carrierData struct {
		CarrierType   string  `json:"carrier_type"`
		Value         string  `json:"value"`
		IsTemporary   bool    `json:"is_temporary"`
		ReplacesValue *string `json:"replaces_value"`
		LotID         string  `json:"lot_id"`
	}
	assemblyData struct {
		AssemblyItemID  string `json:"assembly_item_id"`
		ComponentItemID string `json:"component_item_id"`
		ComponentLotID  string `json:"component_lot_id"`
		ComponentTypeID string `json:"component_type_id"`
		Quantity        int    `json:"quantity"`
		Position        string `json:"position"`
	}
	lotRegisteredData struct {
		LotID              string `json:"lot_id"`
		ActualQuantity     int    `json:"actual_quantity"`
		CertificatePresent bool   `json:"certificate_present"`
		LotKind            string `json:"lot_kind"`
		HeatNo             string `json:"heat_no"`
		ItemTypeID         string `json:"item_type_id"`
	}
	erpLotData struct {
		LotID          string `json:"lot_id"`
		ExternalNumber string `json:"external_number"`
		SupplierID     string `json:"supplier_id"`
		ItemTypeID     string `json:"item_type_id"`
		HeatNo         string `json:"heat_no"`
		Quantity       int    `json:"quantity"`
	}
	lotIssuedData struct {
		LotID        string   `json:"lot_id"`
		ItemIDs      []string `json:"item_ids"`
		OrderID      string   `json:"order_id"`
		Quantity     int      `json:"quantity"`
		IssuedBy     string   `json:"issued_by"`
		ToLocationID string   `json:"to_location_id"`
	}
	lotResolvedData struct {
		LotID      string `json:"lot_id"`
		Resolution string `json:"resolution"`
	}
	groupData struct {
		GroupID       string   `json:"group_id"`
		Kind          string   `json:"kind"`
		ItemIDs       []string `json:"item_ids"`
		WitnessItemID string   `json:"witness_item_id"`
	}
	inspectionData struct {
		Outcome         string `json:"outcome"`
		Method          string `json:"method"`
		StepKey         string `json:"step_key"`
		InspectionPoint string `json:"inspection_point"`
		ConclusionRef   string `json:"conclusion_ref"`
	}
	containmentData struct {
		Level            string   `json:"level"`
		IncidentID       string   `json:"incident_id"`
		LotID            string   `json:"lot_id"`
		Basis            []string `json:"basis"`
		ReleasedEventIDs []string `json:"released_event_ids"`
	}
	runStartedData struct {
		OperationRunID string     `json:"operation_run_id"`
		StepKey        string     `json:"step_key"`
		EquipmentID    string     `json:"equipment_id"`
		StartedAt      *time.Time `json:"operation_started_at"`
	}
	assignedData struct {
		SubjectEventID string `json:"subject_event_id"`
		ItemID         string `json:"item_id"`
		PreviousItemID string `json:"previous_item_id"`
		Method         string `json:"method"`
	}
	refData struct {
		ItemTypeID  string     `json:"item_type_id"`
		EquipmentID string     `json:"equipment_id"`
		LocationID  string     `json:"location_id"`
		Result      string     `json:"result"`
		ValidFrom   *time.Time `json:"valid_from"`
	}
)

func decodeData[T any](r kernel.Record) (T, bool) {
	var v T
	if len(r.Data) == 0 {
		return v, false
	}
	return v, json.Unmarshal(r.Data, &v) == nil
}

// genealogyStep — собственный шаг стадии над генеалогией (AD-42).
func genealogyStep(g Genealogy, r kernel.Record) (Genealogy, []kernel.Addressed) {
	g.init()
	st := &step{g: &g, r: r}
	if r.ItemID != "" {
		st.touch(r.ItemID)
	}
	if r.ItemID == "" && r.Kind == catalog.KindFact && (r.CarrierRef != "" || awaitsItem(r)) {
		// Событие изделия, которое приём не разрешил (AD-41): сначала привязка.
		st.unbound()
		return g, st.out
	}
	switch r.Type {
	case catalog.ItemItemRegistered:
		st.registered()
	case catalog.ItemCarrierApplied:
		st.carrierApplied()
	case catalog.ItemCarrierRemoved:
		st.carrierRemoved()
	case catalog.ItemAssemblyRecorded:
		st.assembly()
	case catalog.ItemReleaseRecorded:
		if r.ItemID != "" {
			n := g.Items[r.ItemID]
			n.Released = true
			g.Items[r.ItemID] = n
		}
	case catalog.OperationRunStarted:
		st.runStarted()
	case catalog.GenealogyLotRegistered, catalog.ErpLotReceived:
		st.lotRegistered()
	case catalog.GenealogyLotIssued:
		st.lotIssued()
	case catalog.DecisionLotResolved:
		st.lotResolved()
	case catalog.GenealogyGroupFormed:
		st.groupFormed()
	case catalog.GenealogyGroupDissolved:
		if d, ok := decodeData[groupData](r); ok {
			if grp, ok := g.Groups[d.GroupID]; ok && grp.DissolvedAt == nil {
				at := r.OccurredAt
				grp.DissolvedAt = &at
				g.Groups[d.GroupID] = grp
			}
		}
	case catalog.InspectionResultRecorded:
		st.witness()
	case catalog.DecisionContainmentSet, catalog.DecisionContainmentApplied:
		st.containment()
	case catalog.DecisionContainmentReleased:
		st.containmentReleased()
	case catalog.BindingLinkAssigned:
		st.assigned()
	case catalog.ReferenceItemTypeDefined, catalog.ReferenceEquipmentDefined, catalog.ReferenceEquipmentVerified:
		st.referenceChanged()
	}
	return g, st.out
}

// step — контекст одного шага own: генеалогия, запись и выход.
type step struct {
	g   *Genealogy
	r   kernel.Record
	out []kernel.Addressed
}

func (st *step) emit(t catalog.Type, itemID, key string, data any, causes ...kernel.Record) {
	if itemID == "" {
		return
	}
	a, err := kernel.NewAddressed(Module, t, "item:"+itemID, key, data, append([]kernel.Record{st.r}, causes...)...)
	if err != nil {
		panic(err) // тип каталога и эмитент закреплены кодом: ошибка — ошибка сборки
	}
	st.out = append(st.out, a)
}

// touch — изделие известно генеалогии; последний момент активности.
func (st *step) touch(itemID string) ItemNode {
	n := st.g.Items[itemID]
	if st.r.OccurredAt.After(n.LastAt) {
		n.LastAt = st.r.OccurredAt
	}
	if n.RunID == "" {
		n.RunID = st.r.RunID
	}
	st.g.Items[itemID] = n
	return n
}

// ── генеалогия ──

// linkData — data genealogy.link.added v1.
type linkData struct {
	ParentItemID string `json:"parent_item_id,omitempty"`
	ChildItemID  string `json:"child_item_id,omitempty"`
	LotID        string `json:"lot_id,omitempty"`
	GroupID      string `json:"group_id,omitempty"`
	Position     string `json:"position,omitempty"`
	Relation     string `json:"relation"`
	BasisEventID string `json:"basis_event_id"`
	Inherited    bool   `json:"inherited,omitempty"`
}

// addLot — изделие сделано из партии: связь партии изделию; действующий блок
// партии доходит до изделия и собранных из него.
func (st *step) addLot(itemID, lotID string, inherited bool) {
	if itemID == "" || lotID == "" {
		return
	}
	n := st.g.Items[itemID]
	if slices.Contains(n.Lots, lotID) {
		return
	}
	n.Lots = append(slices.Clone(n.Lots), lotID)
	st.g.Items[itemID] = n
	l := st.lot(lotID)
	if !slices.Contains(l.Items, itemID) {
		l.Items = append(slices.Clone(l.Items), itemID)
		slices.Sort(l.Items)
	}
	st.g.Lots[lotID] = l
	st.emit(catalog.GenealogyLinkAdded, itemID, "lot|"+lotID+"|"+itemID,
		linkData{ChildItemID: itemID, LotID: lotID, Relation: "made_from_lot", BasisEventID: st.r.EventID, Inherited: inherited})
	if l.Hold != "" {
		st.propagate(Inbound{SourceEventID: l.HoldEventID, Source: "lot", Level: l.Hold, LotID: lotID}, append([]string{itemID}, st.g.Ancestors(itemID)...))
	}
}

// lot — узел партии (создаётся при первом упоминании).
func (st *step) lot(lotID string) LotNode {
	l, ok := st.g.Lots[lotID]
	if !ok {
		l = LotNode{LotID: lotID, Status: LotReceived, Kind: LotKindLot, RunID: st.r.RunID}
	}
	if st.r.Seq > l.BasisSeq {
		l.BasisSeq = st.r.Seq
	}
	return l
}

func (st *step) registered() {
	d, ok := decodeData[registeredData](st.r)
	if !ok || d.ItemID == "" {
		return
	}
	n := st.touch(d.ItemID)
	if !n.Registered {
		n.Registered, n.RegisteredAt, n.TypeID = true, st.r.OccurredAt, d.ItemTypeID
		st.g.Items[d.ItemID] = n
		// Внутренний ID — тоже носитель (AD-16): internal_id действует с регистрации.
		st.openCarrier(d.ItemID, "internal_id", d.ItemID, false)
	}
	lots := slices.Clone(d.LotIDs)
	slices.Sort(lots)
	for _, l := range slices.Compact(lots) {
		st.addLot(d.ItemID, l, false)
	}
	if d.SplitFrom != "" && d.SplitFrom != d.ItemID {
		st.split(d.SplitFrom, d.ItemID)
	}
}

// split — разделение 1→N (FR-15): связь split_from обоим, перенос
// происхождения (партии исходного) и действующего сдерживания исходного.
func (st *step) split(parent, child string) {
	p := st.g.Items[parent]
	if !slices.Contains(p.SplitInto, child) {
		p.SplitInto = append(slices.Clone(p.SplitInto), child)
		st.g.Items[parent] = p
	}
	c := st.g.Items[child]
	c.SplitFrom = parent
	st.g.Items[child] = c
	ld := linkData{ParentItemID: parent, ChildItemID: child, Relation: "split_from", BasisEventID: st.r.EventID}
	st.emit(catalog.GenealogyLinkAdded, parent, "split|"+parent+"|"+child+"|p", ld)
	st.emit(catalog.GenealogyLinkAdded, child, "split|"+parent+"|"+child+"|c", ld)
	for _, l := range p.Lots {
		st.addLot(child, l, true)
	}
	for _, h := range p.Holds {
		if !h.Released && h.Level != "none" {
			st.propagate(Inbound{SourceEventID: h.EventID, Source: "split_parent", Level: h.Level, SourceItemID: parent}, []string{child})
		}
	}
	for _, in := range p.Inbound {
		if !in.Released {
			in.Path = append(slices.Clone(in.Path), parent)
			st.propagate(in, []string{child})
		}
	}
}

func (st *step) assembly() {
	d, ok := decodeData[assemblyData](st.r)
	if !ok || d.AssemblyItemID == "" {
		return
	}
	asm := st.touch(d.AssemblyItemID)
	for _, c := range asm.Components {
		if c.EventID == st.r.EventID {
			return
		}
	}
	comp := Component{ItemID: d.ComponentItemID, LotID: d.ComponentLotID, TypeID: d.ComponentTypeID, Position: d.Position,
		Quantity: d.Quantity, EventID: st.r.EventID, At: st.r.OccurredAt}
	if comp.ItemID != "" && (comp.ItemID == d.AssemblyItemID || st.g.Contains(comp.ItemID, d.AssemblyItemID)) {
		return // цикл в дереве сборки: гард api отклоняет (item.assembly_cycle), стадия не строит
	}
	asm.Components = append(slices.Clone(asm.Components), comp)
	st.g.Items[d.AssemblyItemID] = asm
	up := append([]string{d.AssemblyItemID}, st.g.Ancestors(d.AssemblyItemID)...)
	if comp.ItemID != "" {
		c := st.touch(comp.ItemID)
		c.Parent, c.Position = d.AssemblyItemID, d.Position
		st.g.Items[comp.ItemID] = c
		ld := linkData{ParentItemID: d.AssemblyItemID, ChildItemID: comp.ItemID, Position: d.Position, Relation: "component_of", BasisEventID: st.r.EventID}
		st.emit(catalog.GenealogyLinkAdded, d.AssemblyItemID, "asm|"+d.AssemblyItemID+"|"+comp.ItemID+"|p", ld)
		st.emit(catalog.GenealogyLinkAdded, comp.ItemID, "asm|"+d.AssemblyItemID+"|"+comp.ItemID+"|c", ld)
		// Сдерживание компонента и всего, что под ним, — вверх по дереву сборки (AD-42).
		for _, id := range append([]string{comp.ItemID}, st.g.Descendants(comp.ItemID)...) {
			n := st.g.Items[id]
			for _, h := range n.Holds {
				if !h.Released && h.Level != "none" {
					st.propagate(Inbound{SourceEventID: h.EventID, Source: "component", Level: h.Level, SourceItemID: id, Path: []string{id}}, up)
				}
			}
			for _, in := range n.Inbound {
				if !in.Released {
					in.Path = append(slices.Clone(in.Path), id)
					st.propagate(in, up)
				}
			}
		}
	}
	if comp.LotID != "" {
		// Партионный компонент (крепёж, уплотнение): сборка сделана из партии.
		st.addLot(d.AssemblyItemID, comp.LotID, false)
	}
}

func (st *step) runStarted() {
	d, ok := decodeData[runStartedData](st.r)
	if !ok || st.r.ItemID == "" || d.OperationRunID == "" {
		return
	}
	at := st.r.OccurredAt
	if d.StartedAt != nil {
		at = d.StartedAt.UTC()
	}
	n := st.g.Items[st.r.ItemID]
	if slices.ContainsFunc(n.Runs, func(x RunMark) bool { return x.RunID == d.OperationRunID }) {
		return
	}
	n.Runs = append(slices.Clone(n.Runs), RunMark{RunID: d.OperationRunID, StepKey: d.StepKey, Equipment: d.EquipmentID, Started: at})
	st.g.Items[st.r.ItemID] = n
}

// ── партии ──

func (st *step) lotRegistered() {
	var id string
	var l LotNode
	if st.r.Type == catalog.ErpLotReceived {
		d, ok := decodeData[erpLotData](st.r)
		if !ok || d.LotID == "" {
			return
		}
		id, l = d.LotID, st.lot(d.LotID)
		l.ExternalRef, l.Supplier, l.TypeID, l.Declared = d.ExternalNumber, d.SupplierID, d.ItemTypeID, d.Quantity
		if d.HeatNo != "" {
			l.HeatNo = d.HeatNo
		}
		if l.ReceivedAt == nil {
			at := st.r.OccurredAt
			l.ReceivedAt = &at
		}
	} else {
		d, ok := decodeData[lotRegisteredData](st.r)
		if !ok || d.LotID == "" {
			return
		}
		id, l = d.LotID, st.lot(d.LotID)
		q, cert, at := d.ActualQuantity, d.CertificatePresent, st.r.OccurredAt
		l.Actual, l.Certificate, l.RegisteredAt = &q, &cert, &at
		if l.ReceivedAt == nil {
			l.ReceivedAt = &at
		}
		if d.LotKind != "" {
			l.Kind = d.LotKind
		}
		if d.HeatNo != "" {
			l.HeatNo = d.HeatNo
		}
		if d.ItemTypeID != "" {
			l.TypeID = d.ItemTypeID
		}
		if l.Status == LotReceived {
			l.Status = LotRegistered
		}
	}
	l.Events = appendOnce(l.Events, st.r.EventID)
	st.g.Lots[id] = l
}

func (st *step) lotIssued() {
	d, ok := decodeData[lotIssuedData](st.r)
	if !ok || d.LotID == "" {
		return
	}
	l := st.lot(d.LotID)
	if slices.Contains(l.Events, st.r.EventID) {
		return
	}
	items := slices.Clone(d.ItemIDs)
	slices.Sort(items)
	items = slices.Compact(items)
	l.Issued += d.Quantity
	l.Issues = append(slices.Clone(l.Issues), LotIssue{EventID: st.r.EventID, Quantity: d.Quantity, OrderID: d.OrderID,
		ToLocation: d.ToLocationID, IssuedBy: d.IssuedBy, ItemIDs: items, At: st.r.OccurredAt})
	l.Events = appendOnce(l.Events, st.r.EventID)
	if l.Status == LotAccepted || l.Status == LotRegistered || l.Status == LotReceived {
		l.Status = LotIssued
	}
	st.g.Lots[d.LotID] = l
	for _, it := range items {
		st.touch(it)
		st.addLot(it, d.LotID, false)
	}
}

// lotResolved — решение по партии на входном контроле: отказ — блок партии
// (lot_hold) всем изделиям партии и собранным из них (FR-49, AD-42).
func (st *step) lotResolved() {
	d, ok := decodeData[lotResolvedData](st.r)
	if !ok || d.LotID == "" {
		return
	}
	l := st.lot(d.LotID)
	l.Events = appendOnce(l.Events, st.r.EventID)
	switch d.Resolution {
	case "accept", "accept_partially":
		if l.Status != LotIssued {
			l.Status = LotAccepted
		}
	case "reject":
		l.Status = LotRejected
	case "insufficient_data":
		l.Status = LotOnHold
	}
	st.g.Lots[d.LotID] = l
	if d.Resolution == "reject" {
		st.holdLot(d.LotID, "lot_hold", st.r.EventID, "")
	}
}

// holdLot — блок партии: изделиям партии и собранным из них.
func (st *step) holdLot(lotID, level, eventID, sourceItem string) {
	l := st.lot(lotID)
	if l.HoldEventID == eventID {
		return
	}
	l.Hold, l.HoldEventID = level, eventID
	if l.Status != LotRejected {
		l.Status = LotOnHold
	}
	st.g.Lots[lotID] = l
	var direct []string
	for _, it := range st.g.LotItems(lotID) {
		if it.Relation == RelMadeFromLot && it.ItemID != sourceItem {
			direct = append(direct, it.ItemID)
		}
	}
	for _, id := range direct {
		st.propagate(Inbound{SourceEventID: eventID, Source: "lot", Level: level, LotID: lotID, SourceItemID: sourceItem},
			append([]string{id}, st.g.Ancestors(id)...))
	}
	if sourceItem != "" {
		st.propagate(Inbound{SourceEventID: eventID, Source: "lot", Level: level, LotID: lotID, SourceItemID: sourceItem}, st.g.Ancestors(sourceItem))
	}
}

// ── временные группы и образец-свидетель ──

func (st *step) groupFormed() {
	d, ok := decodeData[groupData](st.r)
	if !ok || d.GroupID == "" {
		return
	}
	if _, dup := st.g.Groups[d.GroupID]; dup {
		return
	}
	items := slices.Clone(d.ItemIDs)
	if d.WitnessItemID != "" {
		items = append(items, d.WitnessItemID)
	}
	slices.Sort(items)
	items = slices.Compact(items)
	st.g.Groups[d.GroupID] = GroupNode{GroupID: d.GroupID, Kind: d.Kind, Items: items, Witness: d.WitnessItemID,
		FormedAt: st.r.OccurredAt, EventID: st.r.EventID, RunID: st.r.RunID}
	for _, id := range items {
		n := st.touch(id)
		n.Groups = appendOnce(n.Groups, d.GroupID)
		st.g.Items[id] = n
		st.emit(catalog.GenealogyLinkAdded, id, "grp|"+d.GroupID+"|"+id,
			linkData{ChildItemID: id, GroupID: d.GroupID, Relation: "grouped_with", BasisEventID: st.r.EventID})
	}
}

// witnessData — data genealogy.witness.propagated v1.
type witnessData struct {
	GroupID           string `json:"group_id"`
	GroupKind         string `json:"group_kind"`
	WitnessItemID     string `json:"witness_item_id"`
	InspectionEventID string `json:"inspection_event_id"`
	Outcome           string `json:"outcome"`
	Method            string `json:"method,omitempty"`
	StepKey           string `json:"step_key,omitempty"`
	InspectionPoint   string `json:"inspection_point,omitempty"`
	ConclusionRef     string `json:"conclusion_ref,omitempty"`
}

// witness — результат контроля образца-свидетеля после формирования группы
// — каждому изделию группы (FR-15: «результат образца-свидетеля садки
// отражается в паспортах всех изделий садки»).
func (st *step) witness() {
	if st.r.ItemID == "" {
		return
	}
	d, ok := decodeData[inspectionData](st.r)
	if !ok || d.Outcome == "" {
		return
	}
	for _, gid := range st.g.Items[st.r.ItemID].Groups {
		grp := st.g.Groups[gid]
		if grp.Witness != st.r.ItemID || st.r.OccurredAt.Before(grp.FormedAt) || slices.Contains(grp.Witnessed, st.r.EventID) {
			continue
		}
		grp.Witnessed = append(slices.Clone(grp.Witnessed), st.r.EventID)
		st.g.Groups[gid] = grp
		wd := witnessData{GroupID: gid, GroupKind: grp.Kind, WitnessItemID: st.r.ItemID, InspectionEventID: st.r.EventID,
			Outcome: d.Outcome, Method: d.Method, StepKey: d.StepKey, InspectionPoint: d.InspectionPoint, ConclusionRef: d.ConclusionRef}
		for _, id := range grp.Items {
			if id != st.r.ItemID {
				st.emit(catalog.GenealogyWitnessPropagated, id, "wit|"+gid+"|"+st.r.EventID+"|"+id, wd)
			}
		}
	}
}

// ── сдерживание ──

// containmentOut — data genealogy.containment.propagated v1.
type containmentOut struct {
	Level         string   `json:"level"`
	Source        string   `json:"source"`
	SourceEventID string   `json:"source_event_id"`
	LotID         string   `json:"lot_id,omitempty"`
	SourceItemID  string   `json:"source_item_id,omitempty"`
	Path          []string `json:"path,omitempty"`
	Released      bool     `json:"released,omitempty"`
	Basis         []string `json:"basis"`
}

// propagate — сдерживание in каждому изделию targets (кроме уже получивших
// его от того же источника): запись и отметка Inbound у изделия.
func (st *step) propagate(in Inbound, targets []string) {
	for _, id := range targets {
		if id == "" || id == in.SourceItemID {
			continue
		}
		n := st.g.Items[id]
		i := slices.IndexFunc(n.Inbound, func(x Inbound) bool { return x.SourceEventID == in.SourceEventID })
		if i >= 0 && n.Inbound[i].Released == in.Released {
			continue
		}
		n.Inbound = slices.Clone(n.Inbound)
		if i >= 0 {
			n.Inbound[i].Released = in.Released
		} else {
			n.Inbound = append(n.Inbound, in)
		}
		st.g.Items[id] = n
		key := "hold|" + in.SourceEventID + "|" + id
		if in.Released {
			key += "|released"
		}
		basis := []string{in.SourceEventID}
		if st.r.EventID != in.SourceEventID {
			basis = append(basis, st.r.EventID)
		}
		slices.Sort(basis)
		st.emit(catalog.GenealogyContainmentPropagated, id, key, containmentOut{Level: in.Level, Source: in.Source, SourceEventID: in.SourceEventID,
			LotID: in.LotID, SourceItemID: in.SourceItemID, Path: slices.Clone(in.Path), Released: in.Released, Basis: slices.Compact(basis)})
	}
}

// containment — сдерживание изделия человеком или правилом: вверх по дереву
// сборки; блок партии (lot_hold) — всем изделиям его партий. Сдерживание по
// области инцидента не распространяется здесь (область расширяет analysis) и
// эхо собственной записи стадии тоже.
func (st *step) containment() {
	id := st.r.ItemID
	d, ok := decodeData[containmentData](st.r)
	if !ok || id == "" || d.Level == "" || d.Level == "none" || d.IncidentID != "" {
		return
	}
	if st.r.Type == catalog.DecisionContainmentApplied && (d.LotID != "" || st.echo(d.Basis)) {
		return
	}
	n := st.g.Items[id]
	if slices.ContainsFunc(n.Holds, func(h Hold) bool { return h.EventID == st.r.EventID }) {
		return
	}
	n.Holds = append(slices.Clone(n.Holds), Hold{EventID: st.r.EventID, Level: d.Level})
	st.g.Items[id] = n
	if anc := st.g.Ancestors(id); len(anc) > 0 {
		st.propagate(Inbound{SourceEventID: st.r.EventID, Source: "component", Level: d.Level, SourceItemID: id, Path: []string{id}}, anc)
	}
	if d.Level == "lot_hold" {
		for _, l := range st.g.LotsOf(id) {
			st.holdLot(l, d.Level, st.r.EventID, id)
		}
	}
}

// echo — сдерживание правилом, основанное на адресованной записи стадии.
func (st *step) echo(basis []string) bool {
	for _, b := range basis {
		for _, id := range slices.Sorted(maps.Keys(st.g.Items)) {
			for _, in := range st.g.Items[id].Inbound {
				if in.SourceEventID == b {
					return true
				}
			}
		}
	}
	return false
}

// containmentReleased — основание снято у источника: изделиям, получившим
// блок от него, — та же запись с released (решает человек, AD-27).
func (st *step) containmentReleased() {
	d, ok := decodeData[containmentData](st.r)
	if !ok || st.r.ItemID == "" {
		return
	}
	n := st.g.Items[st.r.ItemID]
	n.Holds = slices.Clone(n.Holds)
	var released []string
	for i := range n.Holds {
		if slices.Contains(d.ReleasedEventIDs, n.Holds[i].EventID) && !n.Holds[i].Released {
			n.Holds[i].Released = true
			released = append(released, n.Holds[i].EventID)
		}
	}
	st.g.Items[st.r.ItemID] = n
	for _, ev := range released {
		for _, id := range slices.Sorted(maps.Keys(st.g.Items)) {
			for _, in := range st.g.Items[id].Inbound {
				if in.SourceEventID == ev && !in.Released {
					in.Released = true
					st.propagate(in, []string{id})
				}
			}
		}
		for _, lid := range slices.Sorted(maps.Keys(st.g.Lots)) {
			if l := st.g.Lots[lid]; l.HoldEventID == ev {
				l.Hold, l.HoldEventID = "", ""
				st.g.Lots[lid] = l
			}
		}
	}
}

// ── носители и привязка событий без изделия ──

func carrierKey(t, v string) string { return t + ":" + v }

func (st *step) openCarrier(itemID, t, v string, temporary bool) {
	key := carrierKey(t, v)
	spans := st.g.Carriers[key]
	for _, sp := range spans {
		if sp.ItemID == itemID && sp.To == nil {
			return
		}
	}
	st.g.Carriers[key] = append(slices.Clone(spans), CarrierSpan{ItemID: itemID, Type: t, Value: v, Temporary: temporary, From: st.r.OccurredAt, EventID: st.r.EventID})
}

func (st *step) closeCarrier(itemID, t, v string) {
	key := carrierKey(t, v)
	spans := slices.Clone(st.g.Carriers[key])
	for i := range spans {
		if spans[i].ItemID == itemID && spans[i].To == nil {
			at := st.r.OccurredAt
			spans[i].To = &at
		}
	}
	st.g.Carriers[key] = spans
}

func (st *step) carrierApplied() {
	d, ok := decodeData[carrierData](st.r)
	if !ok || st.r.ItemID == "" || d.Value == "" {
		return
	}
	if d.ReplacesValue != nil && *d.ReplacesValue != "" {
		for _, key := range slices.Sorted(maps.Keys(st.g.Carriers)) {
			if strings.HasSuffix(key, ":"+*d.ReplacesValue) {
				t, _, _ := strings.Cut(key, ":")
				st.closeCarrier(st.r.ItemID, t, *d.ReplacesValue)
			}
		}
	}
	st.openCarrier(st.r.ItemID, d.CarrierType, d.Value, d.IsTemporary)
	st.addLot(st.r.ItemID, d.LotID, false)
}

func (st *step) carrierRemoved() {
	d, ok := decodeData[carrierData](st.r)
	if !ok || st.r.ItemID == "" {
		return
	}
	st.closeCarrier(st.r.ItemID, d.CarrierType, d.Value)
}

// resolvedOut — data binding.link.resolved v1.
type resolvedOut struct {
	SubjectEventID     string       `json:"subject_event_id"`
	ItemID             string       `json:"item_id,omitempty"`
	Candidates         []string     `json:"candidates,omitempty"`
	BindingBasis       string       `json:"binding_basis"`
	BindingReliability string       `json:"binding_reliability"`
	CarrierRef         string       `json:"carrier_ref,omitempty"`
	PreviousItemID     string       `json:"previous_item_id,omitempty"`
	Subject            *subjectCopy `json:"subject,omitempty"`
}

// subjectCopy — копия события без изделия в binding.link.resolved.
type subjectCopy struct {
	EventType     string          `json:"event_type"`
	SchemaVersion int             `json:"schema_version,omitempty"`
	OccurredAt    string          `json:"occurred_at"`
	SourceID      string          `json:"source_id,omitempty"`
	SourceKind    string          `json:"source_kind,omitempty"`
	Data          json.RawMessage `json:"data"`
}

func (u Unbound) subject() *subjectCopy {
	data := u.Data
	if len(data) == 0 {
		data = json.RawMessage("{}")
	}
	return &subjectCopy{EventType: u.Type, SchemaVersion: u.SchemaVersion, OccurredAt: u.OccurredAt.UTC().Format("2006-01-02T15:04:05.000Z"),
		SourceID: u.SourceID, SourceKind: u.SourceKind, Data: data}
}

// awaitsItem — факт изделия без номера изделия и без носителя (AD-41, FR-34,
// S12 №1): приём кладёт его в поток стадии (global), изделие назначит только
// человек — событие ждёт в очереди ручной привязки. Факт партии (есть
// lot_id: выборочный контроль колец на складе) изделия не ждёт.
func awaitsItem(r kernel.Record) bool {
	if r.Stream != "global" {
		return false
	}
	if info, ok := catalog.Lookup(r.Type); !ok || info.Stream != "item" {
		return false
	}
	var d struct {
		LotID string `json:"lot_id"`
	}
	_ = json.Unmarshal(r.Data, &d)
	return d.LotID == ""
}

// unbound — событие без изделия с носителем (AD-41): разрешение по реестру
// носителей стадии на occurred_at. Одно изделие — привязка; несколько —
// каждому кандидату запись с перечнем кандидатов (не угадываем); ни одного —
// событие ждёт ручной привязки.
func (st *step) unbound() {
	r := st.r
	if _, seen := st.g.Unbound[r.EventID]; seen {
		return
	}
	u := Unbound{EventID: r.EventID, Type: string(r.Type), SchemaVersion: r.SchemaVersion, OccurredAt: r.OccurredAt,
		SourceID: r.SourceID, SourceKind: r.SourceKind, CarrierRef: r.CarrierRef, Level: r.IdentificationLevel, Data: r.Data, RunID: r.RunID}
	cands := st.g.CarrierAt(normalizeCarrier(r.CarrierRef), r.OccurredAt)
	u.Candidates = cands
	switch len(cands) {
	case 0:
	case 1:
		u.BoundTo, u.Basis = cands[0], "carrier"
		level := r.IdentificationLevel
		if level == "" || level == "ambiguous" || level == "unidentified" {
			level = "probable"
		}
		st.emit(catalog.BindingLinkResolved, cands[0], "bind|"+r.EventID+"|"+cands[0],
			resolvedOut{SubjectEventID: r.EventID, ItemID: cands[0], BindingBasis: "carrier", BindingReliability: level, CarrierRef: r.CarrierRef, Subject: u.subject()})
	default:
		for _, c := range cands {
			st.emit(catalog.BindingLinkResolved, c, "bind|"+r.EventID+"|"+c,
				resolvedOut{SubjectEventID: r.EventID, Candidates: slices.Clone(cands), BindingBasis: "carrier", BindingReliability: "ambiguous", CarrierRef: r.CarrierRef})
		}
	}
	st.g.Unbound[r.EventID] = u
}

// normalizeCarrier — ключ носителя `тип:значение` из carrier_ref (в том числе
// в виде QR `ant:carrier:‹тип›:‹значение›`).
func normalizeCarrier(ref string) string {
	return strings.TrimPrefix(ref, "ant:carrier:")
}

// assigned — ручная привязка или перепривязка человеком (binding.link.assigned,
// AD-41): изделию — привязка с копией события; прежнему изделию и прежним
// кандидатам — запись, что событие теперь у другого.
func (st *step) assigned() {
	d, ok := decodeData[assignedData](st.r)
	if !ok || d.SubjectEventID == "" || d.ItemID == "" {
		return
	}
	u, known := st.g.Unbound[d.SubjectEventID]
	prev := d.PreviousItemID
	if prev == "" && known {
		prev = u.BoundTo
	}
	out := resolvedOut{SubjectEventID: d.SubjectEventID, ItemID: d.ItemID, BindingBasis: "manual", BindingReliability: "unique"}
	if known {
		out.CarrierRef, out.Subject = u.CarrierRef, u.subject()
	}
	if prev != d.ItemID {
		out.PreviousItemID = prev
	}
	key := "bind|" + d.SubjectEventID + "|" + st.r.EventID
	st.emit(catalog.BindingLinkResolved, d.ItemID, key+"|"+d.ItemID, out)
	others := slices.Clone(u.Candidates)
	if prev != "" {
		others = append(others, prev)
	}
	slices.Sort(others)
	for _, o := range slices.Compact(others) {
		if o != d.ItemID {
			st.emit(catalog.BindingLinkResolved, o, key+"|"+o, resolvedOut{SubjectEventID: d.SubjectEventID, ItemID: d.ItemID,
				BindingBasis: "manual", BindingReliability: "unique", PreviousItemID: o})
		}
	}
	if known {
		u.BoundTo, u.Basis = d.ItemID, "manual"
		st.g.Unbound[d.SubjectEventID] = u
	}
}

// ── изменения справочника задним числом ──

// referenceChanged — изменение справочника с датой действия в прошлом (AD-31):
// изделия, у которых после даты действия и до записи изменения было
// затронутое (выполнение на оборудовании; изделие этого типа), получают
// reference.change.affects_item — пересвёртка по действовавшему справочнику.
func (st *step) referenceChanged() {
	d, ok := decodeData[refData](st.r)
	if !ok {
		return
	}
	from := st.r.OccurredAt
	if d.ValidFrom != nil {
		from = d.ValidFrom.UTC()
	}
	known := st.r.RecordedAt
	if known.IsZero() || known.Before(st.r.OccurredAt) {
		known = st.r.OccurredAt
	}
	if !from.Before(known) {
		return // действует с сейчас или в будущем — изделия увидят его своим срезом
	}
	kind := reference.KindItemType
	if d.EquipmentID != "" {
		kind = reference.KindEquipment
	}
	for _, id := range slices.Sorted(maps.Keys(st.g.Items)) {
		n := st.g.Items[id]
		hit := false
		switch kind {
		case reference.KindEquipment:
			hit = slices.ContainsFunc(n.Runs, func(x RunMark) bool {
				return x.Equipment == d.EquipmentID && !x.Started.Before(from) && x.Started.Before(known)
			})
		default:
			hit = n.TypeID == d.ItemTypeID && d.ItemTypeID != "" && !n.LastAt.Before(from)
		}
		if !hit {
			continue
		}
		a, err := reference.AffectsItem(id, st.r, kind, from)
		if err != nil {
			panic(err)
		}
		st.out = append(st.out, a)
	}
}

func appendOnce(xs []string, x string) []string {
	if x == "" || slices.Contains(xs, x) {
		return xs
	}
	return append(slices.Clone(xs), x)
}
