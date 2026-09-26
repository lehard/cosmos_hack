package item

import (
	"context"
	"encoding/json"
	"slices"
	"strconv"
	"strings"
	"time"

	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/domain/crossitem"
	"ant/internal/domain/engine"
	dom "ant/internal/domain/item"
	"ant/internal/domain/kernel"
)

// Чтение модуля item в режиме live: паспорт и журнал изменений — та же
// свёртка изделия, что у воркера, на момент (AD-22); генеалогия и поиск по
// носителю — состояние межизделийной стадии (AD-41, AD-42); список — проекции.

// carrierTypes — типы носителей (AD-16) в порядке поиска по голому значению.
var carrierTypes = []string{"dpm_datamatrix", "tag_qr", "container_cell", "route_card", "post_context", "manual_entry", "internal_id"}

// Lookup — изделие по номеру детали или содержимому скана (item.item.lookup):
// носитель `ant:carrier:‹тип›:‹значение›` или `‹тип›:‹значение›` — по реестру
// носителей стадии на момент (с учётом «снят / заменён», AD-41); голое
// значение — любой тип носителя, затем внутренний ID и его локальная часть.
func (s *Service) Lookup(ctx context.Context, q string, m platform.Moment) (ItemLookup, error) {
	if !s.live() {
		return s.Unimplemented.Lookup(ctx, q, m)
	}
	st, err := s.stage(ctx)
	if err != nil {
		return ItemLookup{}, err
	}
	g := st.Own.Genealogy
	when := time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)
	if m.AsOf != nil {
		when = *m.AsOf
	}
	q = strings.TrimSpace(q)
	ref := strings.TrimPrefix(q, "ant:carrier:")
	var keys []string
	if t, v, ok := strings.Cut(ref, ":"); ok && slices.Contains(carrierTypes, t) {
		keys = append(keys, t+":"+v)
	} else {
		for _, t := range carrierTypes {
			keys = append(keys, t+":"+ref)
		}
	}
	for _, k := range keys {
		if ids := g.CarrierAt(k, when); len(ids) > 0 {
			return ItemLookup{ItemID: ids[0], CarrierRef: "ant:carrier:" + k, Candidates: candidatesOf(ids)}, nil
		}
	}
	// Номер детали: полный ID или локальная часть (Ф-017 → F-017).
	local := strings.ToUpper(ref)
	for _, r := range [][2]string{{"КР-", "C-"}, {"Ф-", "F-"}, {"К-", "R-"}} {
		local = strings.Replace(local, r[0], r[1], 1)
	}
	var hits []string
	for _, id := range sortedKeys(g.Items) {
		if !g.Items[id].Registered {
			continue
		}
		if strings.EqualFold(id, ref) || strings.HasSuffix(strings.ToUpper(id), ":"+local) || strings.HasSuffix(strings.ToUpper(id), "/"+local) {
			hits = append(hits, id)
		}
	}
	if len(hits) == 0 {
		return ItemLookup{}, notFound(q)
	}
	return ItemLookup{ItemID: hits[0], Candidates: candidatesOf(hits)}, nil
}

func candidatesOf(ids []string) []string {
	if len(ids) < 2 {
		return nil
	}
	return ids
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

// List — изделия по шагу, статусу, партии, заданию (item.item.list): перечень
// item.index и строки item.row; партия — включая собранные из неё (FR-45).
func (s *Service) List(ctx context.Context, f ItemFilter, m platform.Moment, p platform.Page) (ItemList, error) {
	if !s.live() {
		return s.Unimplemented.List(ctx, f, m, p)
	}
	var idx IndexRecord
	raw, ok, err := s.cfg.Projections.Get(ctx, ProjectionIndex, IndexKey)
	if err != nil {
		return ItemList{}, err
	}
	if ok {
		if err := json.Unmarshal(raw, &idx); err != nil {
			return ItemList{}, err
		}
	}
	var lotItems []string
	if f.LotID != "" {
		st, err := s.stage(ctx)
		if err != nil {
			return ItemList{}, err
		}
		for _, x := range st.Own.Genealogy.LotItems(f.LotID) {
			lotItems = append(lotItems, x.ItemID)
		}
	}
	out := ItemList{Items: []ItemRow{}}
	for _, id := range idx.IDs {
		r, ok, err := s.row(ctx, id)
		if err != nil {
			return ItemList{}, err
		}
		if !ok || (m.RunID != "" && r.RunID != m.RunID) || (f.StepKey != "" && r.StepKey != f.StepKey) || (f.Summary != "" && r.Status.Summary != f.Summary) ||
			(f.OrderID != "" && r.OrderID != f.OrderID) || (f.LotID != "" && !slices.Contains(lotItems, id) && !slices.Contains(r.LotIDs, f.LotID)) {
			continue
		}
		out.Items = append(out.Items, r.ItemRow)
	}
	return page(out, p), nil
}

func page(l ItemList, p platform.Page) ItemList {
	from, _ := strconv.Atoi(p.Cursor)
	limit := p.Limit
	if limit <= 0 {
		limit = 100
	}
	if from > len(l.Items) {
		from = len(l.Items)
	}
	to := min(from+limit, len(l.Items))
	res := ItemList{Items: l.Items[from:to]}
	if to < len(l.Items) {
		res.NextCursor = strconv.Itoa(to)
	}
	return res
}

// Passport — паспорт изделия (item.passport.read, FR-42): исходные сигналы,
// анализ системы, решения людей — раздельно, с автором, временем, подписью и
// статусом её проверки; оси статуса §3b; зоны, носители, генеалогия.
func (s *Service) Passport(ctx context.Context, itemID string, m platform.Moment) (ItemPassport, error) {
	if !s.live() {
		return s.Unimplemented.Passport(ctx, itemID, m)
	}
	a, err := s.itemAt(ctx, itemID, m)
	if err != nil {
		return ItemPassport{}, err
	}
	it := a.Snap.Item
	p := ItemPassport{ItemID: itemID, Label: Label(itemID), ItemTypeID: it.ItemTypeID, ItemRevision: it.ItemRevision, ProcessVersion: it.ProcessVersion,
		StepKey: it.StepKey, OrderID: it.OrderID, LotIDs: nonNil(it.LotIDs), Identification: it.Identification(), Status: StatusOf(a.Snap),
		Documents: []ItemDocumentRef{}, Zones: []ItemZone{}, Carriers: []ItemCarrier{}, Incidents: []string{}, Nonconformities: []string{},
		BasisSeq: a.Snap.BasisSeq, SplitFrom: it.SplitFrom}
	p.Entries = entries(a, it)
	// FR-65: документы изделия, собранные из истории (модуль documents, эпик 28).
	for _, r := range a.Snap.Documents.Refs() {
		p.Documents = append(p.Documents, ItemDocumentRef{DocumentID: r.DocumentID, Template: r.TemplateRef, Title: r.Title, Status: r.Status, Digest: r.Digest})
	}
	for _, z := range it.Zones {
		p.Zones = append(p.Zones, ItemZone{ZoneID: z.ZoneID, Title: z.Title, Closed: z.Closed, ClosedBy: z.ClosedBy, OpenIntervention: z.OpenIntervention,
			InspectionStatus: z.Status, LastInspection: z.LastInspection})
	}
	for _, c := range it.Carriers {
		p.Carriers = append(p.Carriers, ItemCarrier{CarrierType: c.Type, Value: c.Value, ZoneID: c.ZoneID, Temporary: c.Temporary, State: c.State,
			AppliedAt: c.AppliedAt, RemovedAt: c.RemovedAt})
	}
	for _, mb := range incidents(a.Snap.Analysis) {
		p.Incidents = append(p.Incidents, mb.IncidentID)
		p.IncidentStatuses = append(p.IncidentStatuses, ItemIncident{IncidentID: mb.IncidentID, Status: mb.Status, Action: mb.Action,
			ScopeVersion: mb.ScopeVersion, ViaAssemblyOf: mb.ViaAssemblyOf})
	}
	for _, c := range a.Snap.Analysis.Cases {
		if !slices.Contains(p.Nonconformities, c.NCID) {
			p.Nonconformities = append(p.Nonconformities, c.NCID)
		}
	}
	for _, q := range it.Questions {
		qid := kernel.Reaction{Slot: dom.QuestionSlot(itemID, q.Key)}.ID(1)
		for _, r := range a.Recorded {
			if r.Type == catalog.ItemIdentificationQuestioned && r.Slot.TriggerKey == q.Key {
				qid = r.EventID
			}
		}
		p.Questions = append(p.Questions, IdentificationQuestion{Cause: q.Cause, Candidates: q.Candidates, Basis: nonNil(q.Basis), At: q.At,
			QuestionedID: qid, SubjectEventID: q.SubjectEventID, Open: !q.Closed, ClosedBy: q.ClosedBy})
	}
	for _, w := range it.Witnesses {
		p.Witnesses = append(p.Witnesses, WitnessResult{GroupID: w.GroupID, GroupKind: w.GroupKind, WitnessItemID: w.WitnessID,
			InspectionEventID: w.InspectionID, Outcome: w.Outcome, Method: w.Method, ConclusionRef: w.Conclusion, At: w.At})
	}
	for _, h := range it.Holds {
		p.Holds = append(p.Holds, GenealogyHold{Level: h.Level, Source: h.Source, SourceEventID: h.SourceEventID, LotID: h.LotID,
			SourceItemID: h.SourceItemID, Path: h.Path, Released: h.Released})
	}
	for _, iv := range it.Interventions {
		p.Interventions = append(p.Interventions, ItemIntervention{InterventionID: iv.ID, ZoneIDs: nonNil(iv.Zones), Purpose: iv.Purpose,
			OpenedAt: iv.OpenedAt, ClosedAt: iv.ClosedAt, RetestRequired: iv.Retest})
	}
	for _, rc := range it.RefChanges {
		p.RefChanges = append(p.RefChanges, ItemRefChange{ReferenceEventID: rc.ReferenceEventID, Kind: rc.Kind, ValidFrom: rc.ValidFrom})
	}
	return p, nil
}

func nonNil[T any](xs []T) []T {
	if xs == nil {
		return []T{}
	}
	return xs
}

// entries — записи паспорта по времени (FR-42, кейс §7.2): факты, решения,
// реакции движка и адресованные записи стадии, плюс события без изделия,
// привязанные к нему (AD-41).
func entries(a at, it dom.State) []PassportEntry {
	out := make([]PassportEntry, 0, len(a.Prefix)+len(a.Recorded))
	for _, r := range a.Prefix {
		out = append(out, entryOf(r))
	}
	for _, r := range a.Recorded {
		out = append(out, PassportEntry{EventID: r.EventID, EventType: string(r.Type), Kind: "reaction", Seq: r.Seq, OccurredAt: r.OccurredAt,
			RecordedAt: r.OccurredAt, StepKey: stepKeyOf(r.Data), Author: "engine", Summary: summaryOf(r.Type, r.Data),
			Signatures: []ItemSignature{{SignerID: "engine", Class: "server_attested", Check: "unchecked"}}})
	}
	for _, b := range it.BoundEvents() {
		at := time.Time{}
		if b.SubjectAt != nil {
			at = *b.SubjectAt
		}
		e := PassportEntry{EventID: b.SubjectEventID, EventType: b.SubjectType, Kind: "fact", OccurredAt: at, RecordedAt: at,
			StepKey: stepKeyOf(b.SubjectData), Summary: "Привязано: " + summaryOf(catalog.Type(b.SubjectType), b.SubjectData),
			BindingBasis: b.Basis, BindingReliability: b.Reliability, Bound: true, Reliability: reliabilityOf(b.Reliability),
			Signatures: []ItemSignature{{SignerID: "source", Class: "device", Check: "unchecked"}}}
		for _, r := range a.Prefix {
			if r.EventID == b.EventID {
				e.Seq = r.Seq
			}
		}
		out = append(out, e)
	}
	slices.SortStableFunc(out, func(x, y PassportEntry) int {
		if c := x.OccurredAt.Compare(y.OccurredAt); c != 0 {
			return c
		}
		return int(x.Seq - y.Seq)
	})
	return out
}

// sourceKinds — допустимые пометки источника факта (FR-140).
var sourceKinds = []string{"manual_entry", "machine", "sensor", "camera", "external_system", "import"}

func entryOf(r kernel.Record) PassportEntry {
	kind := string(r.Kind)
	if kind == "" {
		kind = "fact"
	}
	e := PassportEntry{EventID: r.EventID, EventType: string(r.Type), Kind: kind, Seq: r.Seq, OccurredAt: r.OccurredAt, RecordedAt: r.RecordedAt,
		StepKey: stepKeyOf(r.Data), Summary: summaryOf(r.Type, r.Data), Corrects: r.Corrects}
	if e.RecordedAt.IsZero() {
		e.RecordedAt = r.ReceivedAt
	}
	author := r.Actor
	if author == "" {
		author = r.SourceID
	}
	e.Author = author
	if slices.Contains(sourceKinds, r.SourceKind) {
		e.SourceKind = r.SourceKind
	}
	class := strings.ReplaceAll(r.Provenance, "-", "_")
	if class == "" {
		class = "server_attested"
	}
	e.Signatures = []ItemSignature{{SignerID: author, Class: class, Check: "unchecked"}}
	if r.Kind == catalog.KindFact {
		e.BindingBasis, e.BindingReliability = "internal_id", "unique"
		if r.CarrierRef != "" {
			e.BindingBasis = "carrier"
			if r.IdentificationLevel != "" {
				e.BindingReliability = r.IdentificationLevel
			}
		}
		e.Reliability = reliabilityOf(e.BindingReliability)
	}
	if r.Type == catalog.BindingLinkResolved {
		var d struct {
			ItemID      string   `json:"item_id"`
			Candidates  []string `json:"candidates"`
			Basis       string   `json:"binding_basis"`
			Reliability string   `json:"binding_reliability"`
		}
		_ = json.Unmarshal(r.Data, &d)
		e.BindingBasis, e.BindingReliability, e.Candidates = d.Basis, d.Reliability, d.Candidates
	}
	return e
}

// reliabilityOf — надёжность факта (FR-140) по надёжности привязки (FR-34).
func reliabilityOf(level string) string {
	switch level {
	case "unique":
		return "high"
	case "probable":
		return "medium"
	case "ambiguous":
		return "low"
	case "unidentified":
		return "unknown"
	}
	return ""
}

func stepKeyOf(data json.RawMessage) string {
	var d struct {
		StepKey string `json:"step_key"`
	}
	_ = json.Unmarshal(data, &d)
	return d.StepKey
}

// summaryOf — краткое содержание записи для ленты истории: название типа
// каталога и главное из data.
func summaryOf(t catalog.Type, data json.RawMessage) string {
	info, _ := catalog.Lookup(t)
	title := info.Title
	if title == "" {
		title = string(t)
	}
	var d map[string]any
	_ = json.Unmarshal(data, &d)
	var parts []string
	for _, k := range []string{"outcome", "value", "level", "relation", "component_item_id", "component_lot_id", "lot_id", "group_id", "cause_kind", "step_key", "zone_ids"} {
		if v, ok := d[k]; ok && v != nil && v != "" {
			parts = append(parts, k+": "+compact(v))
		}
	}
	if len(parts) == 0 {
		return title
	}
	return title + " (" + strings.Join(parts, "; ") + ")"
}

func compact(v any) string {
	switch x := v.(type) {
	case string:
		return x
	default:
		b, _ := json.Marshal(x)
		return string(b)
	}
}

// History — журнал изменений паспорта (item.history.list, FR-43): проекция
// той же свёртки — шаг за шагом по входу изделия, что изменилось в паспорте:
// было / стало / кто / причина. Исправления — новыми записями (FR-122).
func (s *Service) History(ctx context.Context, itemID string, m platform.Moment, p platform.Page) (ItemHistory, error) {
	if !s.live() {
		return s.Unimplemented.History(ctx, itemID, m, p)
	}
	a, err := s.itemAt(ctx, itemID, m)
	if err != nil {
		return ItemHistory{}, err
	}
	b, _, err := s.cfg.Bundles.Bundle(ctx, itemID, a.Prefix)
	if err != nil {
		return ItemHistory{}, err
	}
	rows := history(b, a.Prefix)
	from, _ := strconv.Atoi(p.Cursor)
	limit := p.Limit
	if limit <= 0 {
		limit = 200
	}
	from = min(from, len(rows))
	to := min(from+limit, len(rows))
	out := ItemHistory{Items: rows[from:to]}
	if to < len(rows) {
		out.NextCursor = strconv.Itoa(to)
	}
	return out, nil
}

// history — изменения паспорта по шагам свёртки (FR-43).
func history(b engine.Bundle, in []kernel.Record) []ItemHistoryEntry {
	var snap engine.Snapshot
	prev := viewOf(snap)
	rows := []ItemHistoryEntry{}
	for _, r := range engine.SortInput(in) {
		snap, _ = engine.Step(snap, b, r)
		cur := viewOf(snap)
		author := r.Actor
		if author == "" {
			author = r.SourceID
		}
		for _, f := range sortedKeys(cur) {
			if cur[f] == prev[f] {
				continue
			}
			rows = append(rows, ItemHistoryEntry{Seq: r.Seq, EventID: r.EventID, EventType: string(r.Type), RecordedAt: r.RecordedAt,
				Field: f, Before: prev[f], After: cur[f], Author: author, Reason: reasonOf(r.Data)})
		}
		for _, f := range sortedKeys(prev) {
			if _, ok := cur[f]; !ok {
				rows = append(rows, ItemHistoryEntry{Seq: r.Seq, EventID: r.EventID, EventType: string(r.Type), RecordedAt: r.RecordedAt,
					Field: f, Before: prev[f], Author: author, Reason: reasonOf(r.Data)})
			}
		}
		prev = cur
	}
	return rows
}

// viewOf — поля паспорта, изменения которых идут в журнал (FR-43).
func viewOf(s engine.Snapshot) map[string]string {
	it := s.Item
	v := map[string]string{}
	if !it.Registered {
		return v
	}
	st := StatusOf(s)
	v["registration"] = it.ItemTypeID + " ред. " + it.ItemRevision
	v["identification"] = it.Identification()
	v["status.position"], v["status.quality"], v["status.containment"], v["status.summary"] = st.Position, st.Quality, st.Containment, st.Summary
	if it.StepKey != "" {
		v["step"] = it.StepKey
	}
	for _, c := range it.Carriers {
		if c.Type != "internal_id" {
			v["carrier:"+c.Type+":"+c.Value] = c.State
		}
	}
	for _, z := range it.Zones {
		if z.Status != dom.ZoneNotInspected || z.Closed {
			val := z.Status
			if z.Closed {
				val += ", доступ закрыт"
			}
			v["zone:"+z.ZoneID] = val
		}
	}
	for _, iv := range it.Interventions {
		v["intervention:"+iv.ID] = map[bool]string{true: "закрыто", false: "открыто"}[iv.ClosedAt != nil]
	}
	for _, c := range it.Components {
		ref := c.ItemID
		if ref == "" {
			ref = "партия " + c.LotID
		}
		v["component:"+c.EventID] = ref
	}
	for _, l := range it.Links {
		v["genealogy:"+l.EventID] = l.Relation + " " + firstNonEmpty(l.ParentID, l.LotID, l.GroupID, l.ChildID)
	}
	for _, w := range it.Witnesses {
		v["witness:"+w.InspectionID] = w.Outcome
	}
	for _, h := range it.Holds {
		v["hold:"+h.SourceEventID] = h.Level + map[bool]string{true: " (основание снято)", false: ""}[h.Released]
	}
	for _, b := range it.Bindings {
		v["binding:"+b.SubjectEventID] = firstNonEmpty(b.BoundTo, "кандидаты: "+strings.Join(b.Candidates, ", "))
	}
	for _, m := range incidents(s.Analysis) {
		v["incident:"+m.IncidentID] = m.Status + " / " + m.Action
	}
	if it.Released {
		v["release"] = it.Warehouse
	}
	return v
}

func firstNonEmpty(xs ...string) string {
	for _, x := range xs {
		if x != "" {
			return x
		}
	}
	return ""
}

func reasonOf(data json.RawMessage) string {
	var d struct {
		Reason *struct {
			Text string `json:"text"`
		} `json:"reason"`
		Purpose *struct {
			Text string `json:"text"`
		} `json:"purpose"`
	}
	_ = json.Unmarshal(data, &d)
	switch {
	case d.Reason != nil:
		return d.Reason.Text
	case d.Purpose != nil:
		return d.Purpose.Text
	}
	return ""
}

// Genealogy — генеалогия изделия (item.genealogy.read, FR-45): владелец —
// межизделийная стадия (AD-42), паспорт её показывает: вверх (куда вошло),
// вниз (из чего собрано), партии, разделение, садки.
func (s *Service) Genealogy(ctx context.Context, itemID string, m platform.Moment) (ItemGenealogy, error) {
	if !s.live() {
		return s.Unimplemented.Genealogy(ctx, itemID, m)
	}
	st, err := s.stage(ctx)
	if err != nil {
		return ItemGenealogy{}, err
	}
	g := st.Own.Genealogy
	n, ok := g.Items[itemID]
	if !ok {
		return ItemGenealogy{}, notFound(itemID)
	}
	out := ItemGenealogy{ItemID: itemID, Nodes: []GenealogyNode{}, Up: g.Ancestors(itemID), Down: g.Descendants(itemID), Lots: g.LotsOf(itemID), Groups: n.Groups}
	add := func(node GenealogyNode) {
		if node.Kind == "item" {
			node.Label = Label(node.Ref)
			if r, ok, _ := s.row(ctx, node.Ref); ok {
				node.Summary = r.Status.Summary
			}
		}
		out.Nodes = append(out.Nodes, node)
	}
	add(GenealogyNode{Ref: itemID, Kind: "item", ParentRef: n.Parent, Position: n.Position})
	child := itemID
	for i, a := range out.Up {
		an := g.Items[a]
		add(GenealogyNode{Ref: a, Kind: "item", ParentRef: an.Parent, Position: g.Items[child].Position, Relation: "assembly", Depth: -(i + 1)})
		child = a
	}
	depth := map[string]int{itemID: 0}
	for _, d := range out.Down {
		dn := g.Items[d]
		depth[d] = depth[dn.Parent] + 1
		add(GenealogyNode{Ref: d, Kind: "item", ParentRef: dn.Parent, Position: dn.Position, Relation: "component_of", Depth: depth[d]})
	}
	for _, l := range out.Lots {
		ln := g.Lots[l]
		label := l
		if ln.HeatNo != "" {
			label += " (плавка " + ln.HeatNo + ")"
		}
		add(GenealogyNode{Ref: l, Kind: "lot", Label: label, ParentRef: itemID, Relation: crossitem.RelMadeFromLot})
	}
	if n.SplitFrom != "" {
		add(GenealogyNode{Ref: n.SplitFrom, Kind: "item", Relation: "split_from", Depth: -1})
	}
	for _, c := range n.SplitInto {
		add(GenealogyNode{Ref: c, Kind: "item", ParentRef: itemID, Relation: "split_into", Depth: 1})
	}
	for _, gid := range n.Groups {
		for _, o := range g.Groups[gid].Items {
			if o != itemID {
				add(GenealogyNode{Ref: o, Kind: "item", Relation: "grouped_with", Position: gid})
			}
		}
	}
	return out, nil
}
