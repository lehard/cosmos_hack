package cad

import (
	"encoding/json"
	"slices"
	"time"

	engineapp "ant/internal/application/engine"
	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	dom "ant/internal/domain/cad"
	"ant/internal/domain/kernel"
)

// Проекции модуля cad (AD-45, писатель — cad; роль projector): чистые
// свёртки журнала по ключу, пересобираются `ant rebuild`.
const (
	// ProjectionAssembly — импортированная сборка по отпечатку файла.
	ProjectionAssembly = "cad.assembly"
	// ProjectionAssemblyIndex — отпечатки сборок в порядке импорта (ключ — all).
	ProjectionAssemblyIndex = "cad.assembly_index"
)

// Imported — сборка глазами журнала: data факта cad.assembly.imported, его
// позиция и время.
type Imported struct {
	Data ev.CadAssemblyImportedV1 `json:"data"`
	Seq  int64                    `json:"seq"`
	At   time.Time                `json:"at"`
}

// Projections — проекции модуля cad для реестра движка.
func Projections() []engineapp.GlobalProjection {
	return []engineapp.GlobalProjection{
		{Name: ProjectionAssembly, Writer: dom.Module, Keys: assemblyKeys, Step: assemblyStep},
		{Name: ProjectionAssemblyIndex, Writer: dom.Module, Keys: indexKeys, Step: indexStep},
	}
}

// MustRegister регистрирует проекции cad в реестре движка (cmd/ant, engineRegistry).
func MustRegister(r *engineapp.Registry) {
	for _, p := range Projections() {
		if err := r.AddGlobal(p); err != nil {
			panic(err)
		}
	}
}

func digestOf(r kernel.Record) string {
	if r.Type != catalog.CadAssemblyImported {
		return ""
	}
	var d struct {
		FileDigest string `json:"file_digest"`
	}
	if json.Unmarshal(r.Data, &d) != nil {
		return ""
	}
	return d.FileDigest
}

func assemblyKeys(r kernel.Record) []string {
	if d := digestOf(r); d != "" {
		return []string{d}
	}
	return nil
}

func assemblyStep(_ string, prev json.RawMessage, r kernel.Record) (json.RawMessage, error) {
	if len(prev) > 0 {
		return prev, nil // тот же файл — та же сборка (AD-7)
	}
	var d ev.CadAssemblyImportedV1
	if err := json.Unmarshal(r.Data, &d); err != nil {
		return nil, err
	}
	return json.Marshal(Imported{Data: d, Seq: r.Seq, At: r.OccurredAt})
}

func indexKeys(r kernel.Record) []string {
	if digestOf(r) != "" {
		return []string{"all"}
	}
	return nil
}

func indexStep(_ string, prev json.RawMessage, r kernel.Record) (json.RawMessage, error) {
	var ds []string
	if len(prev) > 0 {
		if err := json.Unmarshal(prev, &ds); err != nil {
			return nil, err
		}
	}
	if d := digestOf(r); !slices.Contains(ds, d) {
		ds = append(ds, d)
	}
	return json.Marshal(ds)
}

// View — сборка для операции cad.assembly.list.
func View(im Imported) CadAssembly {
	d := im.Data
	a := CadAssembly{AssemblyDesignation: d.AssemblyDesignation, Revision: d.Revision, FileDigest: string(d.FileDigest),
		GeometryPresent: d.GeometryPresent, ImportedAt: im.At, Seq: im.Seq, Components: []CadComponent{}, Links: []CadLink{},
		AssemblyItemTypeID: oidS(d.AssemblyItemTypeID), AssemblyName: strS(d.AssemblyName), LifecycleLetter: strS(d.LifecycleLetter),
		GeometryNote: strS(d.GeometryNote), SourceSystem: strS(d.SourceSystem)}
	for _, c := range d.Components {
		x := CadComponent{Position: c.Position, Designation: c.Designation, Quantity: c.Quantity, LotTracked: c.LotTracked,
			ItemTypeID: oidS(c.ItemTypeID), Name: strS(c.Name), ParentItemTypeID: oidS(c.ParentItemTypeID),
			Material: strS(c.Material)}
		if c.Kind != nil {
			x.Kind = string(*c.Kind)
		}
		if c.TotalQuantity != nil {
			x.TotalQuantity = *c.TotalQuantity
		}
		if c.Tracking != nil {
			x.Tracking = string(*c.Tracking)
		}
		if c.MakeOrBuy != nil {
			x.MakeOrBuy = string(*c.MakeOrBuy)
		}
		x.ShelfLifeTracked = c.ShelfLifeTracked != nil && *c.ShelfLifeTracked
		a.Components = append(a.Components, x)
	}
	for _, l := range d.Links {
		a.Links = append(a.Links, CadLink{LinkID: l.LinkID, Kind: string(l.Kind), Components: l.Components,
			LinkType: strS(l.LinkType), ZoneID: oidS(l.ZoneID), Note: strS(l.Note)})
	}
	for _, z := range d.Zones {
		a.Zones = append(a.Zones, CadZone{ZoneID: string(z.ZoneID), Name: z.Name, Kind: string(z.Kind), LinkID: z.LinkID, ItemTypeID: oidS(z.ItemTypeID)})
	}
	for _, c := range d.Constraints {
		x := CadConstraint{ConstraintID: c.ConstraintID, Kind: string(c.Kind), LinkID: c.LinkID, ZoneID: oidS(c.ZoneID), Limit: c.Limit,
			LimitSource: strS(c.LimitSource), FirstItemTypeID: oidS(c.FirstItemTypeID), ThenItemTypeID: oidS(c.ThenItemTypeID),
			Value: c.Value, Unit: strS(c.Unit), TolerancePct: c.TolerancePct, Note: strS(c.Note)}
		for _, z := range c.ClosesZoneIds {
			x.ClosesZoneIDs = append(x.ClosesZoneIDs, string(z))
		}
		if c.StepKey != nil {
			x.StepKey = string(*c.StepKey)
		}
		if c.Quantity != nil {
			x.Quantity = *c.Quantity
		}
		a.Constraints = append(a.Constraints, x)
	}
	for _, c := range d.Characteristics {
		a.Characteristics = append(a.Characteristics, CadCharacteristic{CharacteristicID: c.CharacteristicID, Name: c.Name, Methods: c.Methods, Note: strS(c.Note)})
	}
	for _, x := range d.Discrepancies {
		a.Discrepancies = append(a.Discrepancies, CadDiscrepancy{ItemTypeID: string(x.ItemTypeID), Designation: x.Designation,
			System: string(x.System), Reason: string(x.Reason), Note: strS(x.Note)})
	}
	return a
}

func oidS(p *ev.ObjectID) string {
	if p == nil {
		return ""
	}
	return string(*p)
}

func strS(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
