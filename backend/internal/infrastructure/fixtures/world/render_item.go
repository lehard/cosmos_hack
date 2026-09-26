package world

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	itemapp "ant/internal/application/item"
	"ant/internal/infrastructure/fixtures/loader"
)

// ProcessVersionID — версия процесса, по которой запущены все изделия мира.
const ProcessVersionID = "flange-1"

func itemStatus(st ItemState) itemapp.ItemStatus {
	return itemapp.ItemStatus{Position: st.Position, Quality: st.Quality, Disposition: st.Disposition, Containment: st.Containment, ErpAccounting: st.ERP, Summary: st.Summary}
}

func lotOfItem(it *Item) string {
	if strings.HasPrefix(it.ID, "F-2") {
		return "LOT-BLANK-0911"
	}
	return "LOT-ZF-201"
}

// renderItems — список, поиск, паспорт, журнал изменений, генеалогия (FR-42…FR-45).
func renderItems(c *Ctx) []loader.Response {
	var out []loader.Response
	all := itemapp.ItemList{Items: []itemapp.ItemRow{}}
	byOrder := map[string]*itemapp.ItemList{}
	for _, it := range c.Existing() {
		st := c.S(it)
		row := itemapp.ItemRow{ItemID: FullID(it.ID), Label: it.Label, ItemTypeID: it.ItemType, StepKey: st.Step, VersionLabel: "v1", LotID: lotOfItem(it), Status: itemStatus(st)}
		all.Items = append(all.Items, row)
		if byOrder[it.Order] == nil {
			byOrder[it.Order] = &itemapp.ItemList{Items: []itemapp.ItemRow{}}
		}
		byOrder[it.Order].Items = append(byOrder[it.Order].Items, row)
		out = append(out,
			resp("item.item.lookup", itemapp.ItemLookup{ItemID: FullID(it.ID), CarrierRef: "ant:carrier:dpm_datamatrix:" + it.ID}, "q", it.ID),
			resp("item.passport.read", c.passport(it), "item_id", FullID(it.ID)),
			resp("item.history.list", c.history(it), "item_id", FullID(it.ID)),
			resp("item.genealogy.read", c.genealogy(it), "item_id", FullID(it.ID)),
		)
	}
	out = append(out, resp("item.item.list", all))
	for _, o := range sortedKeys(byOrder) {
		out = append(out, resp("item.item.list", *byOrder[o], "order_id", o))
	}
	return out
}

func (c *Ctx) itemEvents(it *Item) []*Event {
	var out []*Event
	for _, e := range c.M.Events {
		if e.Step <= c.N && e.Item == it {
			out = append(out, e)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Occurred.Before(out[j].Occurred) })
	return out
}

func sourceKindOf(e *Event) string {
	switch e.SourceKind {
	case "manual_entry", "machine", "sensor", "camera", "external_system", "import":
		return e.SourceKind
	}
	return ""
}

func signatureOf(e *Event) []itemapp.ItemSignature {
	signer := e.Author
	if signer == "" {
		signer = e.Source
	}
	level := 0
	if e.Kind == "decision" {
		level = 2
	}
	class := e.Provenance
	if class == "" {
		class = "server_attested"
	}
	return []itemapp.ItemSignature{{SignerID: signer, Class: class, Level: level, Check: "unchecked"}}
}

func (c *Ctx) passport(it *Item) itemapp.ItemPassport {
	st := c.S(it)
	p := itemapp.ItemPassport{ItemID: FullID(it.ID), Label: it.Label, ItemTypeID: it.ItemType, ItemRevision: "Б", ProcessVersion: ProcessVersionID,
		StepKey: st.Step, OrderID: it.Order, LotIDs: []string{lotOfItem(it)}, Identification: "unique", Status: itemStatus(st),
		Entries: []itemapp.PassportEntry{}, Documents: []itemapp.ItemDocumentRef{}, Zones: []itemapp.ItemZone{}, Incidents: []string{}, Nonconformities: []string{}, BasisSeq: c.ItemSeq(it)}
	if it.RingLot != "" {
		p.LotIDs = append(p.LotIDs, it.RingLot)
	}
	if st.Step >= "assembly.seal_install" && strings.HasPrefix(st.Step, "assembly") || strings.HasPrefix(st.Step, "testing") || strings.HasPrefix(st.Step, "final") {
		p.LotIDs = append(p.LotIDs, "LOT-C-301", "LOT-SL-401", "LOT-FS-501")
	}
	for _, e := range c.itemEvents(it) {
		pe := itemapp.PassportEntry{EventID: e.ID, EventType: e.Type, Kind: e.Kind, Seq: e.Seq, OccurredAt: e.Occurred, RecordedAt: e.Recorded, StepKey: e.StepKey,
			Author: e.Author, SourceKind: sourceKindOf(e), Summary: e.Summary, Signatures: signatureOf(e)}
		if pe.Author == "" {
			pe.Author = e.Source
		}
		if pe.SourceKind != "" {
			pe.Reliability = "high"
		}
		p.Entries = append(p.Entries, pe)
	}
	welded := firstWeld(it) != nil && !firstWeld(it).From.After(c.T)
	for _, z := range []string{"W-1.U1", "W-1.U2", "W-1.U3", "W-1.U4", "W-1.U5", "W-1.U6", "W-1.U7", "W-1.U8"} {
		p.Zones = append(p.Zones, itemapp.ItemZone{ZoneID: z, Title: "Шов W-1, участок " + zoneTitle(z)})
	}
	assembled := strings.HasPrefix(st.Step, "testing") || strings.HasPrefix(st.Step, "final") || st.Step == "assembly.zt4_acceptance"
	p.Zones = append(p.Zones, itemapp.ItemZone{ZoneID: "S-1", Title: "Канавка уплотнения", Closed: assembled, ClosedBy: map[bool]string{true: "assembly.cover_install"}[assembled]})
	p.Zones = append(p.Zones, itemapp.ItemZone{ZoneID: "J-1", Title: "Болтовое соединение крышки (12 болтов)"})
	if st.Step == "assembly.intervention" {
		p.Zones[len(p.Zones)-2].OpenIntervention = "INT-" + it.ID[2:] + "-1"
	}
	_ = welded
	p.Carriers = []itemapp.ItemCarrier{{CarrierType: "dpm_datamatrix", Value: it.ID, Temporary: false, State: "verified", AppliedAt: it.Launch.Add(minutes(5))}}
	for _, in := range c.M.Incidents {
		if _, ok := st.Incidents[in.Spec.ID]; ok {
			p.Incidents = append(p.Incidents, in.Spec.ID)
		}
	}
	for _, n := range c.M.NCs {
		if slices.Contains(n.Items, it) && !n.SignalAt.After(c.T) {
			p.Nonconformities = append(p.Nonconformities, n.ID)
		}
	}
	p.Documents = append(p.Documents, c.M.documentsFor(it, c.T)...)
	return p
}

// documentsFor — документы, собранные из истории изделия (FR-65): заявление о
// несоответствии, акт о браке (групповой), журнал изолятора.
func (m *Model) documentsFor(it *Item, t time.Time) []itemapp.ItemDocumentRef {
	var out []itemapp.ItemDocumentRef
	for _, n := range m.NCs {
		if !slices.Contains(n.Items, it) || n.ConfirmedAt.After(t) {
			continue
		}
		status := "in_route"
		if d := n.DispositionAt(m); d != nil && !d.After(t) {
			status = "closed"
		}
		out = append(out, itemapp.ItemDocumentRef{DocumentID: "DOC-" + n.ID, Template: "nc-statement@1", Title: "Заявление о несоответствии " + n.Number, Status: status, Digest: Digest([]byte("DOC-" + n.ID))})
		if len(n.Spec.Items) > 0 && n.Spec.Disposition != nil && !n.Spec.Disposition.At.Time().After(t) {
			out = append(out, itemapp.ItemDocumentRef{DocumentID: "DOC-ACT-" + n.ID, Template: "defect-act@1", Title: fmt.Sprintf("Акт о браке на %d изделий (%s)", len(n.Items), n.Number), Status: "closed", Digest: Digest([]byte("DOC-ACT-" + n.ID))})
		}
	}
	return out
}

// history — журнал изменений паспорта (FR-43): было / стало / кто / причина по осям статусов.
func (c *Ctx) history(it *Item) itemapp.ItemHistory {
	h := itemapp.ItemHistory{Items: []itemapp.ItemHistoryEntry{}}
	for _, e := range c.itemEvents(it) {
		before, after := it.State(e.Occurred.Add(-time.Millisecond)), it.State(e.Occurred)
		add := func(field, b, a string) {
			if b != a {
				h.Items = append(h.Items, itemapp.ItemHistoryEntry{Seq: e.Seq, EventID: e.ID, EventType: e.Type, RecordedAt: e.Recorded, Field: field, Before: b, After: a, Author: e.Author, Reason: e.Summary})
			}
		}
		add("step_key", before.Step, after.Step)
		add("position", before.Position, after.Position)
		add("quality", before.Quality, after.Quality)
		add("disposition", before.Disposition, after.Disposition)
		add("containment", before.Containment, after.Containment)
		add("erp_accounting", before.ERP, after.ERP)
		for _, id := range sortedKeys(after.Incidents) {
			add("incident:"+id, before.Incidents[id], after.Incidents[id])
		}
	}
	return h
}

// genealogy — генеалогия: партии заготовки и компонентов, кольцо (AD-42: владелец данных — crossitem).
func (c *Ctx) genealogy(it *Item) itemapp.ItemGenealogy {
	st := c.S(it)
	g := itemapp.ItemGenealogy{ItemID: FullID(it.ID), Nodes: []itemapp.GenealogyNode{{Ref: FullID(it.ID), Kind: "item", Label: it.Label, Summary: st.Summary}}}
	g.Nodes = append(g.Nodes, itemapp.GenealogyNode{Ref: lotOfItem(it), Kind: "lot", Label: "Партия заготовок " + map[string]string{"LOT-ZF-201": "ЗФ-201", "LOT-BLANK-0911": "ЗП-0911"}[lotOfItem(it)], ParentRef: FullID(it.ID), Position: "ФЛ-100.01.001"})
	w := firstWeld(it)
	unlinked := false
	for _, u := range c.M.Spec.ComponentUnlinks {
		if u.Item == it.ID && !u.At.Time().After(c.T) {
			unlinked = true
		}
	}
	if it.Ring != "" && w != nil && !w.From.After(c.T) && !unlinked {
		g.Nodes = append(g.Nodes, itemapp.GenealogyNode{Ref: FullID(it.Ring), Kind: "item", Label: labelOf(it.Ring), ParentRef: FullID(it.ID), Position: "ФЛ-100.01.002"})
		if it.RingLot != "" {
			g.Nodes = append(g.Nodes, itemapp.GenealogyNode{Ref: it.RingLot, Kind: "lot", Label: "Партия колец " + lotLabel(c.M, it.RingLot), ParentRef: FullID(it.Ring)})
		}
	}
	if strings.HasPrefix(st.Step, "assembly.") && st.Step != "assembly.receive" || strings.HasPrefix(st.Step, "testing") || strings.HasPrefix(st.Step, "final") {
		for _, l := range [][3]string{{"LOT-C-301", "Партия крышек КР-301", "ФЛ-100.00.003"}, {"LOT-SL-401", "Партия уплотнений УП-401", "ФЛ-100.00.004"}, {"LOT-FS-501", "Партия крепежа КП-501", "ФЛ-100.00.005"}} {
			g.Nodes = append(g.Nodes, itemapp.GenealogyNode{Ref: l[0], Kind: "lot", Label: l[1], ParentRef: FullID(it.ID), Position: l[2]})
		}
	}
	return g
}

func lotLabel(m *Model, id string) string {
	for _, l := range m.Spec.Lots {
		if l.ID == id {
			return l.Label
		}
	}
	return id
}
