package cad

import (
	"encoding/hex"
	"slices"
	"strconv"

	"go.stargrave.org/gogost/v7/gost34112012256"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
)

// Module — модуль-эмитент семейства cad (AD-40).
const Module kernel.Module = "cad"

// SourceKompas — source_id импорта файла условной сборки (источник вида
// «импорт», AD-2): факт идёт через обычный приём, как любой внешний источник.
const SourceKompas = "cad.kompas"

// Digest — отпечаток файла: тот же H, что у журнала и материалов
// (streebog256:‹hex›, AD-44). Повторный импорт того же файла — дубль (FR-31).
func Digest(file []byte) string {
	h := gost34112012256.New()
	h.Write(file)
	return constants.DigestPrefix + hex.EncodeToString(h.Sum(nil))
}

// ImportEventID — event_id факта cad.assembly.imported: UUIDv5 от отпечатка
// файла; тот же файл — тот же id, приём отвечает «дубль» (AD-7).
func ImportEventID(digest string) string {
	return kernel.UUIDv5(constants.NsAnt, string(catalog.CadAssemblyImported)+"\x1f"+digest)
}

// MappingEventID — event_id соответствия обозначения КД нашему типу изделия.
func MappingEventID(m Mapping) string {
	return kernel.UUIDv5(constants.NsAnt, "kompas.map\x1fitem_type\x1f"+m.ExternalID+"\x1f"+m.InternalID)
}

// MappingFact — data записи reference.external_id.mapped (система kompas,
// вид item_type): эмитент — reference, шлюз передаёт соответствие через
// приём (AD-18); конфликт — сигнал, не перезапись.
func MappingFact(m Mapping) ev.ReferenceExternalIDMappedV1 {
	return ev.ReferenceExternalIDMappedV1{System: ev.ReferenceExternalIDMappedV1SystemKompas,
		ObjectKind: ev.ReferenceExternalIDMappedV1ObjectKindItemType, ExternalID: m.ExternalID, InternalID: ev.ObjectID(m.InternalID)}
}

// Fact — data факта cad.assembly.imported: состав и связи в форме v1 и
// расширение эпика 31 — дерево (тип, родитель, количество на сборку, учёт),
// зоны, ограничения, характеристики, расхождения; geometry_present = false
// явно, с пояснением (кейс §5.3).
func Fact(a Assembly, r Result) ev.CadAssemblyImportedV1 {
	d := ev.CadAssemblyImportedV1{AssemblyDesignation: a.Designation, Revision: a.Version, FileDigest: ev.Digest(a.Digest),
		GeometryPresent: a.GeometryPresent, Components: []ev.CadComponent{}}
	d.AssemblyItemTypeID = oid(r.AssemblyItemTypeID)
	d.AssemblyName, d.LifecycleLetter = str(a.Name), str(a.Letter)
	d.FormatVersion, d.SourceSystem, d.GeometryNote = str(a.FormatVersion), str(a.Source.System), str(a.GeometryNote)
	d.MappingRule = str(a.MappingRule)
	if !a.Source.ExportedAt.IsZero() {
		t := ev.NewTimestamp(a.Source.ExportedAt)
		d.ExportedAt = &t
	}
	pos := map[string]string{}
	for _, n := range r.Nodes[1:] {
		pos[n.Designation] = strconv.Itoa(n.Position)
		c := ev.CadComponent{Position: strconv.Itoa(n.Position), Designation: n.Designation, Quantity: n.Qty, LotTracked: n.Tracking == TrackLot}
		c.ItemTypeID, c.ParentItemTypeID, c.Name = oid(n.ItemTypeID), oid(n.Parent), str(n.Name)
		total := n.Total
		c.TotalQuantity = &total
		tr := ev.CadAssemblyImportedV1ComponentsElemTracking(n.Tracking)
		c.Tracking = &tr
		k := ev.CadAssemblyImportedV1ComponentsElemKind(n.Kind)
		if !slices.Contains([]string{KindDetail, KindSubassembly, KindStandardPart, KindFastener, KindPurchasedEquipment}, n.Kind) {
			k = ev.CadAssemblyImportedV1ComponentsElemKindOther
		}
		c.Kind = &k
		if n.Comp.MakeOrBuy == "make" || n.Comp.MakeOrBuy == "buy" {
			mb := ev.CadAssemblyImportedV1ComponentsElemMakeOrBuy(n.Comp.MakeOrBuy)
			c.MakeOrBuy = &mb
		}
		if n.Comp.ShelfLifeTracked {
			t := true
			c.ShelfLifeTracked = &t
		}
		c.Unit, c.Material = str(n.Comp.Unit), str(n.Comp.Material)
		d.Components = append(d.Components, c)
	}
	for _, l := range a.Links {
		x := ev.CadLink{LinkID: l.ID, Components: linkPositions(l, pos), ZoneID: oid(l.ID), Note: str(l.Note)}
		switch l.Type {
		case LinkWeld, LinkBoltedJoint, LinkSeal:
			x.Kind = ev.CadAssemblyImportedV1LinksElemKind(l.Type)
		default:
			x.Kind = ev.CadAssemblyImportedV1LinksElemKindOther
			x.LinkType = str(l.Type)
		}
		d.Links = append(d.Links, x)
	}
	for _, z := range r.Zones {
		d.Zones = append(d.Zones, ev.CadZone{ZoneID: ev.ObjectID(z.ID), Name: z.Name, Kind: ev.CadAssemblyImportedV1ZonesElemKind(z.Kind),
			LinkID: z.LinkID, ItemTypeID: oid(z.ItemTypeID)})
	}
	for _, c := range r.Constraints {
		x := ev.CadConstraint{ConstraintID: c.ID, Kind: ev.CadAssemblyImportedV1ConstraintsElemKind(c.Kind), LinkID: c.LinkID, ZoneID: oid(c.ZoneID),
			Limit: c.Limit, LimitSource: str(c.LimitSource), FirstItemTypeID: oid(c.First), ThenItemTypeID: oid(c.Then),
			Value: c.Value, Unit: str(c.Unit), TolerancePct: c.Tolerance, Note: str(c.Note)}
		for _, z := range c.Closes {
			x.ClosesZoneIds = append(x.ClosesZoneIds, ev.ObjectID(z))
		}
		if c.StepKey != "" {
			s := ev.StepKey(c.StepKey)
			x.StepKey = &s
		}
		if c.Qty > 0 {
			q := c.Qty
			x.Quantity = &q
		}
		d.Constraints = append(d.Constraints, x)
	}
	for _, c := range a.Characteristics {
		d.Characteristics = append(d.Characteristics, ev.CadCharacteristic{CharacteristicID: c.ID, Name: c.Name, Methods: c.Control, Note: str(c.Note)})
	}
	for _, x := range r.Discrepancies {
		d.Discrepancies = append(d.Discrepancies, ev.CadDiscrepancy{ItemTypeID: ev.ObjectID(x.ItemTypeID), Designation: x.Designation,
			System: ev.CadAssemblyImportedV1DiscrepanciesElemSystem(x.System), Reason: ev.CadAssemblyImportedV1DiscrepanciesElemReason(x.Reason), Note: str(x.Note)})
	}
	return d
}

// linkPositions — позиции, которые соединяет связь (в порядке упоминания, без повторов).
func linkPositions(l Link, pos map[string]string) []string {
	var out []string
	add := func(des string) {
		if p, ok := pos[des]; ok && !slices.Contains(out, p) {
			out = append(out, p)
		}
	}
	add(l.From)
	add(l.To)
	for _, x := range l.Between {
		add(x)
	}
	add(l.Part)
	for _, x := range l.Fasteners {
		add(x)
	}
	if len(out) == 0 {
		out = []string{"0"} // связь с корнем сборки: позиция 0 — сама сборка
	}
	return out
}

func oid(s string) *ev.ObjectID {
	if s == "" {
		return nil
	}
	x := ev.ObjectID(s)
	return &x
}

func str(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
