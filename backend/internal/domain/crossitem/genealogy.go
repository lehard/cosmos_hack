package crossitem

import (
	"encoding/json"
	"maps"
	"slices"
	"time"
)

// Генеалогия межизделийной стадии (AD-42, FR-15, FR-45): владеет ею стадия,
// item только показывает её в паспорте через порт стадии. Здесь — данные и
// чистые запросы вверх и вниз по дереву сборки, «партия / плавка / садка →
// все изделия, включая собранные». Изменяет данные только шаг стадии
// (own.go); запросы читают их и в api, и в функциях модулей стадии (analysis
// берёт генеалогию через свой порт Genealogy — GenealogyView ниже).

// Статусы партии (перечисление Lot.status операций crossitem).
const (
	LotReceived   = "received"
	LotRegistered = "registered"
	LotAccepted   = "accepted"
	LotRejected   = "rejected"
	LotOnHold     = "on_hold"
	LotIssued     = "issued"
)

// Виды партии (genealogy.lot.registered.lot_kind).
const (
	LotKindLot  = "lot"
	LotKindHeat = "heat"
)

// Связи изделия с партией в карточке партии (LotItem.relation).
const (
	RelMadeFromLot = "made_from_lot"
	RelIssued      = "issued"
	RelAssembled   = "assembled"
)

// Местонахождение изделия для разбивки области риска (как у analysis, FR-61).
const (
	LocInProduction = "in_production"
	LocMovedOn      = "moved_on"
	LocAssembled    = "assembled"
	LocShipped      = "shipped"
)

// Component — компонент в сборке: экземпляр или партия (item.assembly.recorded).
type Component struct {
	ItemID   string    `json:"item_id,omitempty"`
	LotID    string    `json:"lot_id,omitempty"`
	TypeID   string    `json:"type_id,omitempty"`
	Position string    `json:"position,omitempty"`
	Quantity int       `json:"quantity,omitempty"`
	EventID  string    `json:"event_id"`
	At       time.Time `json:"at"`
}

// RunMark — выполнение операции изделия: шаг, оборудование, начало (для
// местонахождения и изменений справочника задним числом).
type RunMark struct {
	RunID     string    `json:"run_id"`
	StepKey   string    `json:"step_key,omitempty"`
	Equipment string    `json:"equipment,omitempty"`
	Started   time.Time `json:"started"`
}

// Hold — сдерживание изделия-источника (decision.containment.*): уровень и
// снято ли основание.
type Hold struct {
	EventID  string `json:"event_id"`
	Level    string `json:"level"`
	Released bool   `json:"released,omitempty"`
}

// Inbound — сдерживание, пришедшее к изделию по генеалогии (от партии или
// компонента): его стадия переносит дальше при сборке.
type Inbound struct {
	SourceEventID string   `json:"source_event_id"`
	Source        string   `json:"source"`
	Level         string   `json:"level"`
	LotID         string   `json:"lot_id,omitempty"`
	SourceItemID  string   `json:"source_item_id,omitempty"`
	Path          []string `json:"path,omitempty"`
	Released      bool     `json:"released,omitempty"`
}

// ItemNode — изделие в генеалогии.
type ItemNode struct {
	TypeID       string      `json:"type_id,omitempty"`
	RunID        string      `json:"run_id,omitempty"`
	RegisteredAt time.Time   `json:"registered_at"`
	Registered   bool        `json:"registered,omitempty"`
	Lots         []string    `json:"lots,omitempty"`
	Parent       string      `json:"parent,omitempty"`
	Position     string      `json:"position,omitempty"`
	Components   []Component `json:"components,omitempty"`
	SplitFrom    string      `json:"split_from,omitempty"`
	SplitInto    []string    `json:"split_into,omitempty"`
	Groups       []string    `json:"groups,omitempty"`
	Runs         []RunMark   `json:"runs,omitempty"`
	Released     bool        `json:"released,omitempty"`
	LastAt       time.Time   `json:"last_at"`
	Holds        []Hold      `json:"holds,omitempty"`
	Inbound      []Inbound   `json:"inbound,omitempty"`
}

// LotIssue — выдача из партии (genealogy.lot.issued).
type LotIssue struct {
	EventID    string    `json:"event_id"`
	Quantity   int       `json:"quantity"`
	OrderID    string    `json:"order_id,omitempty"`
	ToLocation string    `json:"to_location_id,omitempty"`
	IssuedBy   string    `json:"issued_by,omitempty"`
	ItemIDs    []string  `json:"item_ids,omitempty"`
	At         time.Time `json:"at"`
}

// LotNode — партия или плавка.
type LotNode struct {
	LotID        string     `json:"lot_id"`
	Kind         string     `json:"kind,omitempty"`
	HeatNo       string     `json:"heat_no,omitempty"`
	TypeID       string     `json:"type_id,omitempty"`
	Supplier     string     `json:"supplier,omitempty"`
	ExternalRef  string     `json:"external_ref,omitempty"`
	Declared     int        `json:"declared_quantity,omitempty"`
	Actual       *int       `json:"actual_quantity,omitempty"`
	Issued       int        `json:"issued_quantity,omitempty"`
	Certificate  *bool      `json:"certificate_present,omitempty"`
	Status       string     `json:"status"`
	Hold         string     `json:"hold,omitempty"`
	HoldEventID  string     `json:"hold_event_id,omitempty"`
	ReceivedAt   *time.Time `json:"received_at,omitempty"`
	RegisteredAt *time.Time `json:"registered_at,omitempty"`
	Items        []string   `json:"items,omitempty"`
	Issues       []LotIssue `json:"issues,omitempty"`
	Events       []string   `json:"events,omitempty"`
	BasisSeq     int64      `json:"basis_seq"`
	RunID        string     `json:"run_id,omitempty"`
}

// GroupNode — временная группа: садка, групповая операция, транспорт (FR-15).
type GroupNode struct {
	GroupID     string     `json:"group_id"`
	Kind        string     `json:"kind"`
	Items       []string   `json:"items"`
	Witness     string     `json:"witness,omitempty"`
	FormedAt    time.Time  `json:"formed_at"`
	DissolvedAt *time.Time `json:"dissolved_at,omitempty"`
	EventID     string     `json:"event_id"`
	RunID       string     `json:"run_id,omitempty"`
	// Witnessed — результаты контроля свидетеля, уже распространённые на группу.
	Witnessed []string `json:"witnessed,omitempty"`
}

// CarrierSpan — носитель у изделия на интервале [From, To) (AD-16, AD-41).
type CarrierSpan struct {
	ItemID    string     `json:"item_id"`
	Type      string     `json:"type"`
	Value     string     `json:"value"`
	Temporary bool       `json:"temporary,omitempty"`
	From      time.Time  `json:"from"`
	To        *time.Time `json:"to,omitempty"`
	EventID   string     `json:"event_id"`
}

// Unbound — событие без изделия, пришедшее в стадию (AD-41): носитель,
// кандидаты и текущая привязка.
type Unbound struct {
	EventID       string          `json:"event_id"`
	Type          string          `json:"type"`
	SchemaVersion int             `json:"schema_version,omitempty"`
	OccurredAt    time.Time       `json:"occurred_at"`
	SourceID      string          `json:"source_id,omitempty"`
	SourceKind    string          `json:"source_kind,omitempty"`
	CarrierRef    string          `json:"carrier_ref,omitempty"`
	Level         string          `json:"level,omitempty"`
	Candidates    []string        `json:"candidates,omitempty"`
	BoundTo       string          `json:"bound_to,omitempty"`
	Basis         string          `json:"basis,omitempty"`
	Data          json.RawMessage `json:"data,omitempty"`
	RunID         string          `json:"run_id,omitempty"`
}

// Genealogy — генеалогия, партии, садки, плавки, носители и события без
// изделия (AD-42). Значение сериализуется в состояние стадии.
type Genealogy struct {
	Items    map[string]ItemNode      `json:"items,omitempty"`
	Lots     map[string]LotNode       `json:"lots,omitempty"`
	Groups   map[string]GroupNode     `json:"groups,omitempty"`
	Carriers map[string][]CarrierSpan `json:"carriers,omitempty"`
	Unbound  map[string]Unbound       `json:"unbound,omitempty"`
}

func (g *Genealogy) init() {
	if g.Items == nil {
		g.Items = map[string]ItemNode{}
	}
	if g.Lots == nil {
		g.Lots = map[string]LotNode{}
	}
	if g.Groups == nil {
		g.Groups = map[string]GroupNode{}
	}
	if g.Carriers == nil {
		g.Carriers = map[string][]CarrierSpan{}
	}
	if g.Unbound == nil {
		g.Unbound = map[string]Unbound{}
	}
}

// Ancestors — сборки, в которые вошло изделие, снизу вверх: родитель,
// родитель родителя… (запрос «вверх по дереву», FR-45).
func (g Genealogy) Ancestors(itemID string) []string {
	var out []string
	seen := map[string]bool{itemID: true}
	for p := g.Items[itemID].Parent; p != "" && !seen[p]; p = g.Items[p].Parent {
		seen[p] = true
		out = append(out, p)
	}
	return out
}

// Descendants — компоненты-экземпляры сборки на всех уровнях (запрос «вниз
// по дереву», FR-45), в порядке обхода в ширину.
func (g Genealogy) Descendants(itemID string) []string {
	var out []string
	seen := map[string]bool{itemID: true}
	queue := []string{itemID}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, c := range g.Items[cur].Components {
			if c.ItemID == "" || seen[c.ItemID] {
				continue
			}
			seen[c.ItemID] = true
			out = append(out, c.ItemID)
			queue = append(queue, c.ItemID)
		}
	}
	return out
}

// Contains — сборка asm содержит изделие itemID на каком-либо уровне.
func (g Genealogy) Contains(asm, itemID string) bool {
	return slices.Contains(g.Descendants(asm), itemID)
}

// LotItem — изделие, связанное с партией: сделано из неё (напрямую или через
// компонент) или собрано из изделий партии.
type LotItem struct {
	ItemID   string `json:"item_id"`
	Relation string `json:"relation"`
	// Via — изделие партии, через которое сборка попала в список.
	Via string `json:"via,omitempty"`
}

// LotItems — «партия → все изделия, включая собранные» (FR-45): изделия,
// сделанные из партии (регистрация, выдача, носитель с партией, партионный
// компонент сборки), и все сборки над ними. Порядок детерминирован.
func (g Genealogy) LotItems(lotID string) []LotItem {
	direct := slices.Clone(g.Lots[lotID].Items)
	for _, id := range slices.Sorted(maps.Keys(g.Items)) {
		if slices.Contains(g.Items[id].Lots, lotID) && !slices.Contains(direct, id) {
			direct = append(direct, id)
		}
	}
	slices.Sort(direct)
	return g.closure(direct, RelMadeFromLot)
}

// HeatItems — «плавка → все изделия, включая собранные» (FR-45): изделия
// всех партий плавки heatNo (или партии-плавки с этим id).
func (g Genealogy) HeatItems(heatNo string) []LotItem {
	var direct []string
	for _, id := range slices.Sorted(maps.Keys(g.Lots)) {
		l := g.Lots[id]
		if l.HeatNo != heatNo && (l.Kind != LotKindHeat || l.LotID != heatNo) {
			continue
		}
		for _, it := range g.LotItems(id) {
			if it.Relation == RelMadeFromLot && !slices.Contains(direct, it.ItemID) {
				direct = append(direct, it.ItemID)
			}
		}
	}
	slices.Sort(direct)
	return g.closure(direct, RelMadeFromLot)
}

// GroupItems — «садка → все изделия, включая собранные» (FR-45).
func (g Genealogy) GroupItems(groupID string) []LotItem {
	direct := slices.Clone(g.Groups[groupID].Items)
	slices.Sort(direct)
	return g.closure(slices.Compact(direct), "grouped_with")
}

// closure — изделия direct с отношением rel и сборки над ними (assembled).
func (g Genealogy) closure(direct []string, rel string) []LotItem {
	out := make([]LotItem, 0, len(direct))
	seen := map[string]bool{}
	for _, id := range direct {
		if !seen[id] {
			seen[id] = true
			out = append(out, LotItem{ItemID: id, Relation: rel})
		}
	}
	for _, id := range direct {
		for _, a := range g.Ancestors(id) {
			if !seen[a] {
				seen[a] = true
				out = append(out, LotItem{ItemID: a, Relation: RelAssembled, Via: id})
			}
		}
	}
	return out
}

// LotsOf — партии изделия: свои и партионных компонентов сборки (без
// компонентов-экземпляров).
func (g Genealogy) LotsOf(itemID string) []string {
	n := g.Items[itemID]
	out := slices.Clone(n.Lots)
	for _, c := range n.Components {
		if c.LotID != "" && !slices.Contains(out, c.LotID) {
			out = append(out, c.LotID)
		}
	}
	return out
}

// CarrierAt — изделия, у которых носитель key («тип:значение») действовал в
// момент at (с учётом «снят / заменён», AD-41). Порядок — по id изделия.
func (g Genealogy) CarrierAt(key string, at time.Time) []string {
	var out []string
	for _, sp := range g.Carriers[key] {
		if sp.From.After(at) || (sp.To != nil && !sp.To.After(at)) {
			continue
		}
		if !slices.Contains(out, sp.ItemID) {
			out = append(out, sp.ItemID)
		}
	}
	slices.Sort(out)
	return out
}

// ActiveCarrier — изделие, у которого носитель key действует сейчас (по
// последнему известному состоянию); пусто — не действует.
func (g Genealogy) ActiveCarrier(key string) string {
	for _, sp := range g.Carriers[key] {
		if sp.To == nil {
			return sp.ItemID
		}
	}
	return ""
}

// GenealogyView — генеалогия стадии через порт analysis.Genealogy (AD-42):
// сборки изделия, партия компонента, местонахождение. Индекс analysis
// (эпик 22) переключается на неё — функции модулей стадии получают состояние
// стадии (Д-40).
type GenealogyView struct{ G Genealogy }

// Parents — сборки, в которые вошло изделие (непосредственно).
func (v GenealogyView) Parents(itemID string) []string {
	if p := v.G.Items[itemID].Parent; p != "" {
		return []string{p}
	}
	return nil
}

// LotOf — партия компонента-экземпляра; пусто — неизвестна.
func (v GenealogyView) LotOf(componentID string) string {
	if ls := v.G.Items[componentID].Lots; len(ls) > 0 {
		return ls[0]
	}
	return ""
}

// Location — местонахождение изделия относительно операции stepKey (FR-61):
// отгружено, собрано, ушло дальше по маршруту или в производстве.
func (v GenealogyView) Location(itemID, stepKey string) string {
	n := v.G.Items[itemID]
	switch {
	case n.Released:
		return LocShipped
	case n.Parent != "" || len(n.Components) > 0:
		return LocAssembled
	}
	var last time.Time
	found := false
	for _, r := range n.Runs {
		if r.StepKey == stepKey && (!found || r.Started.After(last)) {
			last, found = r.Started, true
		}
	}
	for _, r := range n.Runs {
		if found && r.StepKey != stepKey && r.Started.After(last) {
			return LocMovedOn
		}
	}
	return LocInProduction
}

// Genealogy — генеалогия стадии через порт analysis (AD-42).
func (s Stage) Genealogy() GenealogyView { return GenealogyView{G: s.Own.Genealogy} }
