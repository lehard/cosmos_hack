package crossitem

import (
	"context"
	"encoding/json"
	"slices"
	"strconv"
	"strings"
	"time"

	engineapp "ant/internal/application/engine"
	itemapp "ant/internal/application/item"
	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	"ant/internal/contracts/errcodes"
	dom "ant/internal/domain/crossitem"
	"ant/internal/domain/kernel"
)

// Config — зависимости живой реализации (AD-36, режим live).
type Config struct {
	// Projections — проекции движка: состояние стадии (кэш её свёртки, AD-42).
	Projections engineapp.ProjectionStore
	// Writer — запись фактов и решений (journal.Append, AD-44).
	Writer itemapp.Writer
	// Clock — доменное «сейчас» при приёме команды (AD-37); nil — системное.
	Clock appjournal.DomainClock
}

// Service — реализация live ведущих портов модуля crossitem (AD-36): чтение —
// генеалогия, партии и группы из состояния межизделийной стадии; команды —
// гард над ним, затем запись в журнал с проверкой потока объекта (AD-39).
// Без зависимостей (NewService) — заглушка: операции отвечают 501.
type Service struct {
	Unimplemented
	cfg Config
}

// NewService создаёт заглушку live (все операции — 501).
func NewService() *Service { return &Service{} }

// NewLive создаёт живую реализацию.
func NewLive(cfg Config) *Service { return &Service{cfg: cfg} }

var (
	_ Queries  = (*Service)(nil)
	_ Commands = (*Service)(nil)
)

func (s *Service) live() bool { return s.cfg.Projections != nil }

func (s *Service) now(ctx context.Context) (time.Time, error) {
	if s.cfg.Clock == nil {
		return time.Now().UTC(), nil
	}
	return s.cfg.Clock.Now(ctx)
}

func (s *Service) stage(ctx context.Context) (dom.Genealogy, error) {
	var st dom.Stage
	raw, ok, err := s.cfg.Projections.Get(ctx, StateProjection, stateKey)
	if err != nil || !ok {
		return st.Own.Genealogy, err
	}
	err = json.Unmarshal(raw, &st)
	return st.Own.Genealogy, err
}

func notFound(object, id string) error {
	e := platform.Fail(errcodes.ApiNotFound, "object", object, "id", id)
	e.Detail = object + " «" + id + "» не найден"
	return e
}

func lotView(l dom.LotNode) Lot {
	v := Lot{LotID: l.LotID, ItemTypeID: l.TypeID, Supplier: l.Supplier, ExternalRef: l.ExternalRef, DeclaredQuantity: l.Declared,
		ActualQuantity: l.Actual, IssuedQuantity: l.Issued, CertificatePresent: l.Certificate, Status: l.Status, Containment: l.Hold,
		ReceivedAt: l.ReceivedAt, RegisteredAt: l.RegisteredAt, BasisSeq: l.BasisSeq, Kind: l.Kind, HeatNo: l.HeatNo}
	if v.Status == "" {
		v.Status = dom.LotReceived
	}
	return v
}

// Lots — партии и плавки (crossitem.lot.list, FR-15).
func (s *Service) Lots(ctx context.Context, status string, m platform.Moment, p platform.Page) (LotList, error) {
	if !s.live() {
		return s.Unimplemented.Lots(ctx, status, m, p)
	}
	g, err := s.stage(ctx)
	if err != nil {
		return LotList{}, err
	}
	out := LotList{Items: []Lot{}}
	for _, id := range sortedKeys(g.Lots) {
		l := g.Lots[id]
		if (status != "" && l.Status != status) || (m.RunID != "" && l.RunID != m.RunID) {
			continue
		}
		out.Items = append(out.Items, lotView(l))
	}
	from, _ := strconv.Atoi(p.Cursor)
	limit := p.Limit
	if limit <= 0 {
		limit = 100
	}
	from = min(from, len(out.Items))
	to := min(from+limit, len(out.Items))
	if to < len(out.Items) {
		out.NextCursor = strconv.Itoa(to)
	}
	out.Items = out.Items[from:to]
	return out, nil
}

// Lot — карточка партии (crossitem.lot.read, FR-15, FR-45): выдачи и все
// изделия партии, включая собранные.
func (s *Service) Lot(ctx context.Context, lotID string, m platform.Moment) (LotCard, error) {
	if !s.live() {
		return s.Unimplemented.Lot(ctx, lotID, m)
	}
	g, err := s.stage(ctx)
	if err != nil {
		return LotCard{}, err
	}
	l, ok := g.Lots[lotID]
	if !ok {
		return LotCard{}, notFound("Партия", lotID)
	}
	c := LotCard{Lot: lotView(l), Issues: []LotIssue{}, Items: []LotItem{}, Documents: []platform.DrillRef{}}
	for _, is := range l.Issues {
		c.Issues = append(c.Issues, LotIssue{EventID: is.EventID, Quantity: is.Quantity, OrderID: is.OrderID, ToLocationID: is.ToLocation,
			IssuedBy: is.IssuedBy, At: is.At})
	}
	issued := map[string]bool{}
	for _, is := range l.Issues {
		for _, id := range is.ItemIDs {
			issued[id] = true
		}
	}
	for _, it := range g.LotItems(lotID) {
		rel := "made_from_lot"
		if issued[it.ItemID] && !slices.Contains(g.Items[it.ItemID].Lots, lotID) {
			rel = "issued"
		}
		c.Items = append(c.Items, LotItem{ItemID: it.ItemID, Label: itemapp.Label(it.ItemID), Relation: rel,
			Assembled: it.Relation == dom.RelAssembled, Via: it.Via})
	}
	return c, nil
}

// Groups — временные группы изделий (crossitem.group.list, FR-15).
func (s *Service) Groups(ctx context.Context, m platform.Moment) (ItemGroupList, error) {
	if !s.live() {
		return s.Unimplemented.Groups(ctx, m)
	}
	g, err := s.stage(ctx)
	if err != nil {
		return ItemGroupList{}, err
	}
	out := ItemGroupList{Items: []ItemGroup{}}
	for _, id := range sortedKeys(g.Groups) {
		gr := g.Groups[id]
		if m.RunID != "" && gr.RunID != m.RunID {
			continue
		}
		out.Items = append(out.Items, ItemGroup{GroupID: gr.GroupID, Kind: gr.Kind, ItemIDs: nonNil(gr.Items), WitnessItemID: gr.Witness,
			FormedAt: gr.FormedAt, DissolvedAt: gr.DissolvedAt})
	}
	return out, nil
}

// Trace — «партия / плавка / садка → все изделия, включая собранные» (FR-45).
func (s *Service) Trace(ctx context.Context, lotID, heatNo, groupID string, m platform.Moment) (Trace, error) {
	if !s.live() {
		return s.Unimplemented.Trace(ctx, lotID, heatNo, groupID, m)
	}
	g, err := s.stage(ctx)
	if err != nil {
		return Trace{}, err
	}
	var items []dom.LotItem
	switch {
	case lotID != "":
		if _, ok := g.Lots[lotID]; !ok {
			return Trace{}, notFound("Партия", lotID)
		}
		items = g.LotItems(lotID)
	case heatNo != "":
		items = g.HeatItems(heatNo)
	case groupID != "":
		if _, ok := g.Groups[groupID]; !ok {
			return Trace{}, notFound("Группа", groupID)
		}
		items = g.GroupItems(groupID)
	default:
		return Trace{}, platform.Fail(errcodes.ApiValidationFailed, "field", "lot_id", "reason", "нужен lot_id, heat_no или group_id")
	}
	out := Trace{LotID: lotID, HeatNo: heatNo, GroupID: groupID, Items: []TraceItem{}}
	for _, it := range items {
		out.Items = append(out.Items, TraceItem{ItemID: it.ItemID, Label: itemapp.Label(it.ItemID), Relation: it.Relation, Via: it.Via,
			Assembled: it.Relation == dom.RelAssembled})
	}
	return out, nil
}

// Unbound — события без изделия (AD-41, FR-34): неразрешённые и неоднозначные
// — очередь ручной привязки; привязанные человеком тоже видны.
func (s *Service) Unbound(ctx context.Context, m platform.Moment) (UnboundList, error) {
	if !s.live() {
		return s.Unimplemented.Unbound(ctx, m)
	}
	g, err := s.stage(ctx)
	if err != nil {
		return UnboundList{}, err
	}
	out := UnboundList{Items: []UnboundEvent{}}
	for _, id := range sortedKeys(g.Unbound) {
		u := g.Unbound[id]
		if m.RunID != "" && u.RunID != m.RunID {
			continue
		}
		out.Items = append(out.Items, UnboundEvent{EventID: u.EventID, EventType: u.Type, OccurredAt: u.OccurredAt, SourceID: u.SourceID,
			CarrierRef: u.CarrierRef, Candidates: nonNil(u.Candidates), BoundTo: u.BoundTo, Basis: u.Basis, RunID: u.RunID})
	}
	slices.SortStableFunc(out.Items, func(a, b UnboundEvent) int { return a.OccurredAt.Compare(b.OccurredAt) })
	return out, nil
}

// ── команды ──

func (s *Service) write(ctx context.Context, meta platform.CommandMeta, level int, recs ...itemapp.Record) (platform.Receipt, error) {
	now, err := s.now(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	actor := platform.PrincipalFrom(ctx).PersonID
	for i := range recs {
		recs[i].Meta, recs[i].Actor, recs[i].OccurredAt, recs[i].SignatureLevel = meta, actor, now, level
		if recs[i].GuardStreams == nil {
			recs[i].GuardStreams = []string{recs[i].Stream}
		}
	}
	return s.cfg.Writer.Write(ctx, dom.Module, recs)
}

func personOf(ctx context.Context) string {
	if p := platform.PrincipalFrom(ctx).PersonID; p != "" {
		return p
	}
	return "anonymous"
}

// RegisterLot — регистрация поступившей партии (genealogy.lot.registered, FR-15).
func (s *Service) RegisterLot(ctx context.Context, lotID string, in RegisterLot) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.RegisterLot(ctx, lotID, in)
	}
	g, err := s.stage(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	if l, ok := g.Lots[lotID]; ok && l.RegisteredAt != nil {
		return platform.Receipt{}, platform.Fail(errcodes.ItemLotAlreadyRegistered, "lot_id", lotID)
	}
	d := map[string]any{"lot_id": lotID, "actual_quantity": in.ActualQuantity, "packaging_ok": in.PackagingOK,
		"certificate_present": in.CertificatePresent, "registered_by": personOf(ctx)}
	return s.write(ctx, in.CommandMeta(), 1, itemapp.Record{Type: catalog.GenealogyLotRegistered, Stream: "lot:" + lotID, Data: d})
}

// IssueLot — выдача из партии в производство (genealogy.lot.issued, FR-45):
// только принятой партии (гард lot_accepted).
func (s *Service) IssueLot(ctx context.Context, lotID string, in IssueLot) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.IssueLot(ctx, lotID, in)
	}
	g, err := s.stage(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	l, ok := g.Lots[lotID]
	if !ok {
		return platform.Receipt{}, notFound("Партия", lotID)
	}
	if l.Status != dom.LotAccepted && l.Status != dom.LotIssued {
		return platform.Receipt{}, platform.Fail(errcodes.ItemLotNotAccepted, "lot_id", lotID, "status", l.Status)
	}
	d := map[string]any{"lot_id": lotID, "quantity": in.Quantity, "issued_by": personOf(ctx)}
	if len(in.ItemIDs) > 0 {
		d["item_ids"] = in.ItemIDs
	}
	if in.OrderID != "" {
		d["order_id"] = in.OrderID
	}
	if in.ToLocationID != "" {
		d["to_location_id"] = in.ToLocationID
	}
	return s.write(ctx, in.CommandMeta(), 1, itemapp.Record{Type: catalog.GenealogyLotIssued, Stream: "lot:" + lotID, Data: d})
}

// AssignBinding — ручная привязка или перепривязка события без изделия
// (binding.link.assigned, AD-41, FR-34): решение контролёра в потоке изделия;
// стадия пишет binding.link.resolved новому и прежнему изделию.
func (s *Service) AssignBinding(ctx context.Context, in AssignBinding) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.AssignBinding(ctx, in)
	}
	g, err := s.stage(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	u, ok := g.Unbound[strings.ToLower(in.SubjectEventID)]
	if !ok {
		return platform.Receipt{}, platform.Fail(errcodes.ItemBindingSubjectUnknown, "subject_event_id", in.SubjectEventID)
	}
	if n, ok := g.Items[in.ItemID]; !ok || !n.Registered {
		return platform.Receipt{}, platform.Fail(errcodes.ItemNotRegistered, "item_id", in.ItemID)
	}
	d := map[string]any{"subject_event_id": u.EventID, "item_id": in.ItemID, "method": in.Method,
		"reason": map[string]any{"text": in.Reason.Text}}
	if in.Reason.Code != "" {
		d["reason"].(map[string]any)["code"] = in.Reason.Code
	}
	prev := in.PreviousItemID
	if prev == "" {
		prev = u.BoundTo
	}
	if prev != "" && prev != in.ItemID {
		d["previous_item_id"] = prev
	}
	r := itemapp.Record{Type: catalog.BindingLinkAssigned, Stream: "item:" + in.ItemID, ItemID: in.ItemID, Data: d}
	if prev != "" && prev != in.ItemID {
		r.GuardStreams = []string{"item:" + in.ItemID, "item:" + prev}
	}
	return s.write(ctx, in.CommandMeta(), 2, r)
}

// FormGroup — сформировать временную группу (genealogy.group.formed, FR-15).
func (s *Service) FormGroup(ctx context.Context, in FormGroup) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.FormGroup(ctx, in)
	}
	g, err := s.stage(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	id := in.GroupID
	if id == "" {
		u := kernel.UUIDv5(constants.NsAnt, "group\x1f"+strings.ToLower(in.CommandID))
		id = "GRP-" + strings.ToUpper(u[:8])
	}
	if _, dup := g.Groups[id]; dup {
		return platform.Receipt{}, platform.Fail(errcodes.ItemGroupInvalid, "reason", "группа "+id+" уже есть")
	}
	items := slices.Clone(in.ItemIDs)
	slices.Sort(items)
	items = slices.Compact(items)
	for _, it := range append(slices.Clone(items), in.WitnessItemID) {
		if it == "" {
			continue
		}
		if n, ok := g.Items[it]; !ok || !n.Registered {
			return platform.Receipt{}, platform.Fail(errcodes.ItemGroupInvalid, "reason", "изделие "+it+" не зарегистрировано")
		}
	}
	d := map[string]any{"group_id": id, "kind": in.Kind, "item_ids": items}
	if in.WitnessItemID != "" {
		d["witness_item_id"] = in.WitnessItemID
	}
	return s.write(ctx, in.CommandMeta(), 1, itemapp.Record{Type: catalog.GenealogyGroupFormed, Stream: "group:" + id, Data: d})
}

// DissolveGroup — расформировать группу (genealogy.group.dissolved, FR-15).
func (s *Service) DissolveGroup(ctx context.Context, groupID string, in DissolveGroup) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.DissolveGroup(ctx, groupID, in)
	}
	g, err := s.stage(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	if gr, ok := g.Groups[groupID]; !ok || gr.DissolvedAt != nil {
		return platform.Receipt{}, platform.Fail(errcodes.ItemGroupNotActive, "group_id", groupID)
	}
	return s.write(ctx, in.CommandMeta(), 1, itemapp.Record{Type: catalog.GenealogyGroupDissolved, Stream: "group:" + groupID,
		Data: map[string]any{"group_id": groupID}})
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

func nonNil[T any](xs []T) []T {
	if xs == nil {
		return []T{}
	}
	return xs
}
