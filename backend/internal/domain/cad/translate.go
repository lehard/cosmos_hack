package cad

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
)

// Limit — лимит ремонтов зоны из нормативного слоя: значение и шаг ТП,
// на котором он действует (BPMN `reworkLimit` при `reworkLimitScope="zone"`).
type Limit struct {
	Value   int
	StepKey string
}

// Env — нормативный слой, с которым сверяется перевод связей (AD-17):
// лимиты ремонтов и шаги, исполняющие ограничения зон. Получается из версии
// процесса (application/cad.EnvFromBPMN); пустой Env — перевод без подсказок
// шагов, лимит шва — только из файла.
type Env struct {
	// ReworkLimits — лимит ремонтов по связи (W-1): шаг с областью лимита
	// «зона», зоны которого — участки этой связи (W-1.U1…W-1.U8).
	ReworkLimits map[string]Limit
	// ClosingSteps — шаг, закрывающий доступ к зоне (`closesZoneAccess`): зона → шаг.
	ClosingSteps map[string]string
	// ZoneSteps — операции, работающие с зоной (`zoneRef`), в порядке процесса.
	ZoneSteps map[string][]string
}

// Nomenclature — номенклатура учётной системы для сверки обозначений (FR-95):
// наши типы изделий, у которых есть соответствие в системе. nil — номенклатура
// ещё не получена: сверка не выполнялась.
type Nomenclature struct {
	// System — onec или galaktika.
	System    string
	ItemTypes map[string]bool
}

// Tracking — вид учёта позиции.
const (
	TrackSerial = "serial"
	TrackLot    = "lot"
	TrackNone   = "none"
)

// Node — позиция дерева состава: наш тип, родитель, количество на родителя
// и на одну сборку верхнего уровня, учёт.
type Node struct {
	Position    int
	ItemTypeID  string
	Designation string
	Name        string
	Kind        string
	// Parent — тип изделия родителя (пусто у корня).
	Parent string
	Depth  int
	Qty    int
	// Total — количество на одну сборку верхнего уровня.
	Total    int
	Tracking string
	Comp     Component
}

// Zone — зона изделия из связи сборки (FR-46).
type Zone struct {
	ID, Name, Kind, LinkID string
	// ItemTypeID — тип изделия, на котором лежит зона (наименьший общий
	// родитель соединяемых позиций).
	ItemTypeID string
}

// Виды ограничений нормативного слоя.
const (
	ConstraintReworkLimit  = "rework_limit"
	ConstraintClosesAccess = "closes_access"
	ConstraintInstallOrder = "install_order"
	ConstraintFastening    = "fastening"
	ConstraintTorque       = "torque"
)

// Constraint — ограничение нормативного слоя из связи сборки.
type Constraint struct {
	ID, Kind, LinkID, ZoneID string
	// Limit — лимит ремонтов (nil — не задан ни файлом, ни ТП); LimitSource — откуда.
	Limit       *int
	LimitSource string
	// Closes — зоны, к которым соединение закрывает доступ.
	Closes []string
	// First, Then — порядок установки (типы изделий).
	First, Then string
	StepKey     string
	Qty         int
	Value       *int
	Unit        string
	Tolerance   *int
	Note        string
}

// Mapping — соответствие обозначения КД нашему типу изделия
// (reference.external_id.mapped, система kompas).
type Mapping struct {
	ExternalID, InternalID string
}

// Причины расхождения с номенклатурой учётной системы.
const (
	ReasonNotInERP     = "not_in_erp"
	ReasonNotChecked   = "not_checked"
	discrepancyAllNote = "номенклатура учётной системы ещё не получена — сверка всех позиций не выполнялась"
)

// Discrepancy — позиция без соответствия в номенклатуре учётной системы:
// отчёт администратору и технологу, без автосоздания (FR-95).
type Discrepancy struct {
	ItemTypeID, Designation, System, Reason, Note string
}

// Result — сборка в объектах нормативного слоя.
type Result struct {
	// AssemblyItemTypeID — наш тип изделия сборки.
	AssemblyItemTypeID string
	// Nodes — дерево состава в порядке обхода (корень первым, дети — по позиции).
	Nodes         []Node
	Zones         []Zone
	Constraints   []Constraint
	Mappings      []Mapping
	Discrepancies []Discrepancy
}

// Node — позиция дерева по типу изделия.
func (r Result) Node(itemType string) (Node, bool) {
	for _, n := range r.Nodes {
		if n.ItemTypeID == itemType {
			return n, true
		}
	}
	return Node{}, false
}

// Constraint — ограничение по связи и виду.
func (r Result) Constraint(link, kind string) (Constraint, bool) {
	for _, c := range r.Constraints {
		if c.LinkID == link && c.Kind == kind {
			return c, true
		}
	}
	return Constraint{}, false
}

// Translate — сборка → дерево состава, зоны, ограничения, соответствия ID и
// расхождения (чистая функция, AD-4). Структурные проблемы (Validate) —
// отказ импорта. nom — номенклатура учётной системы (nil — не получена).
func Translate(a Assembly, env Env, nom *Nomenclature) (Result, []Problem) {
	if ps := Validate(a); len(ps) > 0 {
		return Result{}, ps
	}
	root := ItemTypeID(a.Designation)
	r := Result{AssemblyItemTypeID: root}
	byDes := map[string]Component{}
	for _, c := range a.Components {
		byDes[c.Designation] = c
	}
	typeOf := func(des string) string {
		if des == a.AssemblyID || des == a.Designation {
			return root
		}
		return ItemTypeID(des)
	}
	// Уникальность наших ID: два обозначения не должны давать один тип.
	seen := map[string]string{root: a.Designation}
	for i, c := range a.Components {
		id := ItemTypeID(c.Designation)
		if prev, dup := seen[id]; dup {
			return Result{}, []Problem{{Field: "/components/" + strconv.Itoa(i) + "/designation",
				Detail: fmt.Sprintf("обозначения «%s» и «%s» дают один тип изделия %s", prev, c.Designation, id)}}
		}
		seen[id] = c.Designation
	}

	// Дерево состава: корень, затем дети по позиции (обход в глубину).
	r.Nodes = append(r.Nodes, Node{ItemTypeID: root, Designation: a.Designation, Name: a.Name, Kind: KindSubassembly, Qty: 1, Total: 1, Tracking: TrackSerial})
	var walk func(parent string, parentTotal, depth int)
	walk = func(parent string, parentTotal, depth int) {
		var kids []Component
		for _, c := range a.Components {
			if typeOf(c.Parent) == parent {
				kids = append(kids, c)
			}
		}
		slices.SortFunc(kids, func(x, y Component) int { return x.Position - y.Position })
		for _, c := range kids {
			n := Node{Position: c.Position, ItemTypeID: ItemTypeID(c.Designation), Designation: c.Designation, Name: c.Name, Kind: c.Kind,
				Parent: parent, Depth: depth, Qty: c.Qty, Total: c.Qty * parentTotal, Tracking: tracking(c), Comp: c}
			r.Nodes = append(r.Nodes, n)
			walk(n.ItemTypeID, n.Total, depth+1)
		}
	}
	walk(root, 1, 1)

	// Связи → зоны и ограничения.
	parentOf := func(t string) string {
		n, _ := r.Node(t)
		return n.Parent
	}
	owner := func(x, y string) string { return commonOwner(typeOf(x), typeOf(y), parentOf) }
	name := func(des string) string {
		if c, ok := byDes[des]; ok && c.Name != "" {
			return c.Name
		}
		return des
	}
	for _, l := range a.Links {
		switch l.Type {
		case LinkWeld:
			r.Zones = append(r.Zones, Zone{ID: l.ID, Kind: "weld_section", LinkID: l.ID, ItemTypeID: owner(l.From, l.To),
				Name: fmt.Sprintf("Сварной шов %s: %s + %s", l.ID, name(l.From), name(l.To))})
			c := Constraint{ID: l.ID + "/" + ConstraintReworkLimit, Kind: ConstraintReworkLimit, LinkID: l.ID, ZoneID: l.ID,
				Note: "Шов — объект учёта лимита ремонтов (FR-18): доработка сверх лимита без отдельного разрешения блокируется."}
			switch lim, ok := env.ReworkLimits[l.ID]; {
			case l.ReworkLimit > 0:
				v := l.ReworkLimit
				c.Limit, c.LimitSource = &v, "файл сборки"
			case ok && lim.Value > 0:
				v := lim.Value
				c.Limit, c.StepKey = &v, lim.StepKey
				c.LimitSource = "ТП: шаг " + lim.StepKey + " (reworkLimit, область — зона)"
			default:
				c.Note += " Лимит не задан ни файлом, ни ТП — задаёт технолог."
			}
			r.Constraints = append(r.Constraints, c)
		case LinkBoltedJoint:
			r.Zones = append(r.Zones, Zone{ID: l.ID, Kind: "joint", LinkID: l.ID, ItemTypeID: owner(l.From, l.To),
				Name: fmt.Sprintf("Болтовое соединение %s: %s — %s", l.ID, name(l.From), name(l.To))})
			closes := slices.Clone(l.ClosesAccessTo)
			if len(closes) == 0 {
				closes = sealsBetween(a, l.From, l.To)
			}
			c := Constraint{ID: l.ID + "/" + ConstraintClosesAccess, Kind: ConstraintClosesAccess, LinkID: l.ID, ZoneID: l.ID, Closes: closes,
				Note: "Закрывает доступ к зоне: проверка закрываемых зон — до закрытия (скрытые работы, FR-20); разборка — запись вмешательства (FR-21)."}
			for _, z := range closes {
				if st := env.ClosingSteps[z]; st != "" {
					c.StepKey = st
					break
				}
			}
			if len(closes) == 0 {
				c.Note += " Закрываемые зоны в файле не названы и уплотнений между деталями нет — уточняет технолог."
			}
			r.Constraints = append(r.Constraints, c)
			if len(l.Fasteners) > 0 || l.Qty > 0 {
				f := Constraint{ID: l.ID + "/" + ConstraintFastening, Kind: ConstraintFastening, LinkID: l.ID, ZoneID: l.ID, Qty: l.Qty}
				var fs []string
				for _, x := range l.Fasteners {
					fs = append(fs, name(x)+" ("+x+")")
				}
				f.Note = "Крепёж: " + strings.Join(fs, ", ")
				if l.TighteningSequence != "" {
					f.Note += "; порядок затяжки — " + l.TighteningSequence
				}
				r.Constraints = append(r.Constraints, f)
			}
		case LinkSeal:
			x, y := l.Between[0], l.Between[1]
			r.Zones = append(r.Zones, Zone{ID: l.ID, Kind: "groove", LinkID: l.ID, ItemTypeID: owner(x, y),
				Name: fmt.Sprintf("Уплотнение %s: %s между %s и %s", l.ID, name(l.Part), name(x), name(y))})
			then := closingPart(a, x, y)
			c := Constraint{ID: l.ID + "/" + ConstraintInstallOrder, Kind: ConstraintInstallOrder, LinkID: l.ID, ZoneID: l.ID,
				First: typeOf(l.Part), Then: typeOf(then),
				Note: fmt.Sprintf("Порядок установки (FR-17): %s устанавливается до %s; нарушенный порядок или просроченная партия уплотнений — блокировка.", name(l.Part), name(then))}
			if steps := env.ZoneSteps[l.ID]; len(steps) > 0 {
				c.StepKey = steps[0]
			}
			r.Constraints = append(r.Constraints, c)
		default:
			kind := "joint"
			if l.Type != LinkThreadedJoint {
				kind = "other"
			}
			r.Zones = append(r.Zones, Zone{ID: l.ID, Kind: kind, LinkID: l.ID, ItemTypeID: owner(l.From, l.To),
				Name: fmt.Sprintf("Соединение %s: %s — %s", l.ID, name(l.From), name(l.To))})
			if l.TorqueNm > 0 {
				v, tol := l.TorqueNm, l.TolerancePct
				c := Constraint{ID: l.ID + "/" + ConstraintTorque, Kind: ConstraintTorque, LinkID: l.ID, ZoneID: l.ID, Value: &v, Unit: "Н·м", Tolerance: &tol,
					Note: "Момент затяжки под контролем ключа"}
				if l.Locking != "" {
					c.Note += "; " + l.Locking
				}
				r.Constraints = append(r.Constraints, c)
			}
		}
	}

	// Соответствия обозначений КД нашим типам (FR-95) и сверка с учётной системой.
	for _, n := range r.Nodes {
		r.Mappings = append(r.Mappings, Mapping{ExternalID: n.Designation, InternalID: n.ItemTypeID})
	}
	r.Discrepancies = Discrepancies(r, nom)
	return r, nil
}

// Discrepancies — позиции сборки без соответствия в номенклатуре учётной
// системы; номенклатуры нет — одна запись «сверка не выполнялась».
func Discrepancies(r Result, nom *Nomenclature) []Discrepancy {
	if len(r.Nodes) == 0 {
		return nil
	}
	if nom == nil {
		root := r.Nodes[0]
		return []Discrepancy{{ItemTypeID: root.ItemTypeID, Designation: root.Designation, System: "onec", Reason: ReasonNotChecked, Note: discrepancyAllNote}}
	}
	var out []Discrepancy
	for _, n := range r.Nodes {
		if !nom.ItemTypes[n.ItemTypeID] {
			out = append(out, Discrepancy{ItemTypeID: n.ItemTypeID, Designation: n.Designation, System: nom.System, Reason: ReasonNotInERP,
				Note: "обозначение не найдено в номенклатуре — соответствие не создано, нужна проверка администратора и технолога"})
		}
	}
	return out
}

// tracking — учёт позиции: по заводским номерам (детали, сборочные единицы,
// покупное оборудование с номером), по партиям (крепёж, уплотнения со сроком
// хранения), иначе без индивидуального учёта.
func tracking(c Component) string {
	switch {
	case c.SerialTracked:
		return TrackSerial
	case c.LotTracked || c.ShelfLifeTracked:
		return TrackLot
	case c.Kind == KindDetail || c.Kind == KindSubassembly:
		return TrackSerial
	}
	return TrackNone
}

// commonOwner — тип изделия, на котором лежит зона связи x–y: первый предок x,
// который совпадает с y или его предком (и наоборот).
func commonOwner(x, y string, parent func(string) string) string {
	chain := func(t string) []string {
		var out []string
		for p := parent(t); p != ""; p = parent(p) {
			out = append(out, p)
		}
		return out
	}
	selfAndUp := func(t string) map[string]bool {
		m := map[string]bool{t: true}
		for _, p := range chain(t) {
			m[p] = true
		}
		return m
	}
	up := selfAndUp(y)
	for _, p := range chain(x) {
		if up[p] {
			return p
		}
	}
	upX := selfAndUp(x)
	for _, p := range chain(y) {
		if upX[p] {
			return p
		}
	}
	return ""
}

// sealsBetween — зоны уплотнений между двумя позициями: к ним болтовое
// соединение этих позиций закрывает доступ.
func sealsBetween(a Assembly, x, y string) []string {
	var out []string
	for _, l := range a.Links {
		if l.Type == LinkSeal && len(l.Between) == 2 &&
			(l.Between[0] == x && l.Between[1] == y || l.Between[0] == y && l.Between[1] == x) {
			out = append(out, l.ID)
		}
	}
	return out
}

// closingPart — какая из пары x–y закрывает уплотнение: позиция from
// болтового соединения этой пары (крышка на фланец); иначе первая из пары.
func closingPart(a Assembly, x, y string) string {
	for _, l := range a.Links {
		if l.Type == LinkBoltedJoint && (l.From == x && l.To == y || l.From == y && l.To == x) {
			return l.From
		}
	}
	return x
}

// ZoneIDs — зоны результата по порядку (для людей и тестов).
func (r Result) ZoneIDs() []string {
	m := map[string]bool{}
	for _, z := range r.Zones {
		m[z.ID] = true
	}
	return slices.Sorted(maps.Keys(m))
}
