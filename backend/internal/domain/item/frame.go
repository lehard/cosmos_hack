package item

import (
	"encoding/json"
	"slices"
	"strings"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/domain/kernel"
)

// Module — имя модуля-эмитента (AD-40).
const Module kernel.Module = "item"

// Идентификация изделия (AD-16, перечисление ItemPassport.identification).
const (
	IdentUnique       = "unique"
	IdentProbable     = "probable"
	IdentAmbiguous    = "ambiguous"
	IdentUnidentified = "unidentified"
)

// Состояния носителя (ItemCarrier.state).
const (
	CarrierApplied    = "applied"
	CarrierVerified   = "verified"
	CarrierUnreadable = "unreadable"
	CarrierRemoved    = "removed"
	CarrierReplaced   = "replaced"
)

// Статусы зоны (FR-46): не проверялась / проверена / устарела после
// вмешательства; «закрыта» — признак Closed отдельно от статуса проверки.
const (
	ZoneNotInspected = "not_inspected"
	ZoneInspected    = "inspected"
	ZoneStale        = "stale"
)

// Carrier — носитель идентификатора изделия (AD-16): тип, значение, зона,
// временный или постоянный, состояние по событиям нанесён / проверен /
// нечитаем / снят / заменён.
type Carrier struct {
	Type      string     `json:"type"`
	Value     string     `json:"value"`
	ZoneID    string     `json:"zone_id,omitempty"`
	Temporary bool       `json:"temporary,omitempty"`
	State     string     `json:"state"`
	LotID     string     `json:"lot_id,omitempty"`
	AppliedAt time.Time  `json:"applied_at"`
	RemovedAt *time.Time `json:"removed_at,omitempty"`
	EventID   string     `json:"event_id"`
	// VerifiedAt — последнее успешное считывание.
	VerifiedAt *time.Time `json:"verified_at,omitempty"`
}

// Active — носитель действует (не снят и не заменён).
func (c Carrier) Active() bool { return c.State != CarrierRemoved && c.State != CarrierReplaced }

// Physical — носитель физический (не внутренний ID и не ручной ввод).
func (c Carrier) Physical() bool { return c.Type != "internal_id" && c.Type != "manual_entry" }

// Question — «идентификация под сомнением» (AD-16): причина, кандидаты,
// основания; открыт до повторной идентификации человеком с подписью.
type Question struct {
	// Key — ключ срабатывания слота реакции: id записи-причины.
	Key        string    `json:"key"`
	Cause      string    `json:"cause"`
	Candidates []string  `json:"candidates,omitempty"`
	Basis      []string  `json:"basis"`
	At         time.Time `json:"at"`
	// SubjectEventID — событие без изделия, привязка которого неоднозначна.
	SubjectEventID string `json:"subject_event_id,omitempty"`
	Closed         bool   `json:"closed,omitempty"`
	ClosedBy       string `json:"closed_by,omitempty"`
}

// Zone — зона изделия (FR-46): статус проверки, закрытый доступ, открытое
// вмешательство.
type Zone struct {
	ZoneID           string `json:"zone_id"`
	Title            string `json:"title,omitempty"`
	Status           string `json:"status"`
	Closed           bool   `json:"closed,omitempty"`
	ClosedBy         string `json:"closed_by,omitempty"`
	OpenIntervention string `json:"open_intervention,omitempty"`
	LastInspection   string `json:"last_inspection,omitempty"`
}

// Intervention — вмешательство в собранное изделие (FR-21).
type Intervention struct {
	ID         string     `json:"id"`
	Zones      []string   `json:"zones"`
	Removed    []string   `json:"removed,omitempty"`
	Purpose    string     `json:"purpose,omitempty"`
	OpenedAt   time.Time  `json:"opened_at"`
	OpenedBy   string     `json:"opened_by,omitempty"`
	EventID    string     `json:"event_id"`
	ClosedAt   *time.Time `json:"closed_at,omitempty"`
	CloseEvent string     `json:"close_event,omitempty"`
	Retest     bool       `json:"retest,omitempty"`
	// WasClosed — зоны, доступ к которым был закрыт до вмешательства.
	WasClosed []string `json:"was_closed,omitempty"`
}

// Component — компонент, установленный в это изделие (item.assembly.recorded).
type Component struct {
	ItemID   string    `json:"item_id,omitempty"`
	LotID    string    `json:"lot_id,omitempty"`
	TypeID   string    `json:"type_id"`
	Position string    `json:"position,omitempty"`
	Quantity int       `json:"quantity,omitempty"`
	Method   string    `json:"method,omitempty"`
	EventID  string    `json:"event_id"`
	At       time.Time `json:"at"`
}

// Link — связь генеалогии, адресованная изделию стадией (genealogy.link.added).
type Link struct {
	Relation  string    `json:"relation"`
	ParentID  string    `json:"parent_item_id,omitempty"`
	ChildID   string    `json:"child_item_id,omitempty"`
	LotID     string    `json:"lot_id,omitempty"`
	GroupID   string    `json:"group_id,omitempty"`
	Position  string    `json:"position,omitempty"`
	Inherited bool      `json:"inherited,omitempty"`
	Basis     string    `json:"basis_event_id"`
	EventID   string    `json:"event_id"`
	At        time.Time `json:"at"`
}

// Witness — результат образца-свидетеля группы (genealogy.witness.propagated, FR-15).
type Witness struct {
	GroupID      string    `json:"group_id"`
	GroupKind    string    `json:"group_kind"`
	WitnessID    string    `json:"witness_item_id"`
	InspectionID string    `json:"inspection_event_id"`
	Outcome      string    `json:"outcome"`
	Method       string    `json:"method,omitempty"`
	Conclusion   string    `json:"conclusion_ref,omitempty"`
	EventID      string    `json:"event_id"`
	At           time.Time `json:"at"`
}

// Hold — сдерживание, пришедшее по генеалогии (genealogy.containment.propagated):
// ось «сдерживание» меняет nonconformity (AD-30), паспорт показывает основание.
type Hold struct {
	Level         string   `json:"level"`
	Source        string   `json:"source"`
	SourceEventID string   `json:"source_event_id"`
	LotID         string   `json:"lot_id,omitempty"`
	SourceItemID  string   `json:"source_item_id,omitempty"`
	Path          []string `json:"path,omitempty"`
	Released      bool     `json:"released,omitempty"`
	EventID       string   `json:"event_id"`
}

// Binding — событие без изделия, привязанное к изделию стадией или человеком
// (binding.link.resolved, AD-41, FR-34).
type Binding struct {
	SubjectEventID string          `json:"subject_event_id"`
	Basis          string          `json:"basis"`
	Reliability    string          `json:"reliability"`
	Candidates     []string        `json:"candidates,omitempty"`
	Mine           bool            `json:"mine"`
	BoundTo        string          `json:"bound_to,omitempty"`
	CarrierRef     string          `json:"carrier_ref,omitempty"`
	SubjectType    string          `json:"subject_type,omitempty"`
	SubjectAt      *time.Time      `json:"subject_at,omitempty"`
	SubjectData    json.RawMessage `json:"subject_data,omitempty"`
	EventID        string          `json:"event_id"`
}

// RefChange — изменение справочника с действием в прошлом, затронувшее
// изделие (reference.change.affects_item, AD-31).
type RefChange struct {
	ReferenceEventID string `json:"reference_event_id"`
	Kind             string `json:"kind"`
	ValidFrom        string `json:"valid_from"`
	EventID          string `json:"event_id"`
}

// State — состояние модуля item в свёртке одного изделия (AD-5): паспорт —
// регистрация и закреплённая версия, носители и идентификация, зоны и
// вмешательства, сборка и генеалогия (адресованные записи стадии), выпуск.
// Поля экспортируемые (хеш состояния, Д-22); поздние модули читают их через
// Upstream.Item (например, process — изоляцию при «идентификации под сомнением»).
type State struct {
	ItemID            string    `json:"item_id,omitempty"`
	RunID             string    `json:"run_id,omitempty"`
	Registered        bool      `json:"registered,omitempty"`
	RegisteredAt      time.Time `json:"registered_at"`
	RegisteredEventID string    `json:"registered_event_id,omitempty"`
	ItemTypeID        string    `json:"item_type_id,omitempty"`
	ItemRevision      string    `json:"item_revision,omitempty"`
	ProcessVersion    string    `json:"process_version,omitempty"`
	NormativeRev      string    `json:"normative_rev,omitempty"`
	OrderID           string    `json:"order_id,omitempty"`
	EntryStepKey      string    `json:"entry_step_key,omitempty"`
	IsAssembly        bool      `json:"is_assembly,omitempty"`
	SplitFrom         string    `json:"split_from,omitempty"`
	LotIDs            []string  `json:"lot_ids,omitempty"`
	// StepKey — шаг последнего начатого выполнения; Running — выполнения без окончания.
	StepKey string   `json:"step_key,omitempty"`
	Running []string `json:"running,omitempty"`

	Carriers      []Carrier      `json:"carriers,omitempty"`
	Questions     []Question     `json:"questions,omitempty"`
	Zones         []Zone         `json:"zones,omitempty"`
	Interventions []Intervention `json:"interventions,omitempty"`
	Components    []Component    `json:"components,omitempty"`
	Links         []Link         `json:"links,omitempty"`
	Witnesses     []Witness      `json:"witnesses,omitempty"`
	Holds         []Hold         `json:"holds,omitempty"`
	Bindings      []Binding      `json:"bindings,omitempty"`
	RefChanges    []RefChange    `json:"ref_changes,omitempty"`
	Presentations []string       `json:"presentations,omitempty"`

	Released   bool       `json:"released,omitempty"`
	ReleasedAt *time.Time `json:"released_at,omitempty"`
	Warehouse  string     `json:"warehouse,omitempty"`
}

// Env — закреплённая при запуске изделия часть нормативного слоя модуля item
// (AD-17): номенклатура с зонами и связями по КД (normative/reference/…/
// item-types.yaml, AD-31). Собирает application/item (Bundles).
type Env struct {
	Types map[string]TypeDef `json:"types,omitempty"`
}

// TypeDef — тип изделия: маркировка по КД, зоны и связи.
type TypeDef struct {
	Marking string    `json:"marking,omitempty"`
	Zones   []ZoneDef `json:"zones,omitempty"`
	Links   []LinkDef `json:"links,omitempty"`
}

// ZoneDef — зона по КД.
type ZoneDef struct {
	ID   string `json:"zone_id"`
	Name string `json:"name"`
	Kind string `json:"kind,omitempty"`
}

// LinkDef — связь по КД: шов, соединение; закрывает доступ к зонам.
type LinkDef struct {
	ID             string   `json:"link_id"`
	Kind           string   `json:"kind"`
	Zones          []string `json:"zones,omitempty"`
	ClosesAccessTo []string `json:"closes_access_to,omitempty"`
}

// Identification — идентификация изделия (AD-16): под сомнением —
// неоднозначна (кандидаты) или утрачена; иначе — по лучшему носителю.
func (s State) Identification() string {
	for _, q := range s.Questions {
		if !q.Closed {
			if q.Cause == "ambiguous_binding" {
				return IdentAmbiguous
			}
			return IdentUnidentified
		}
	}
	for _, c := range s.Carriers {
		if c.Active() && c.Physical() && c.State != CarrierUnreadable && !c.Temporary {
			return IdentUnique
		}
	}
	for _, c := range s.Carriers {
		if c.Active() && c.Physical() && c.State != CarrierUnreadable {
			return IdentProbable
		}
	}
	if s.Registered {
		return IdentProbable
	}
	return IdentUnidentified
}

// Questioned — идентификация под сомнением: изделие изолируется до повторной
// идентификации человеком с подписью (AD-16). Читают process (положение
// «изолировано») и гарды операций над изделием.
func (s State) Questioned() bool {
	return slices.ContainsFunc(s.Questions, func(q Question) bool { return !q.Closed })
}

// OpenQuestions — открытые вопросы идентификации.
func (s State) OpenQuestions() []Question {
	var out []Question
	for _, q := range s.Questions {
		if !q.Closed {
			out = append(out, q)
		}
	}
	return out
}

// ActiveCarriers — действующие носители.
func (s State) ActiveCarriers() []Carrier {
	var out []Carrier
	for _, c := range s.Carriers {
		if c.Active() {
			out = append(out, c)
		}
	}
	return out
}

// Parents — сборки, в которые вошло изделие (по связям стадии).
func (s State) Parents() []string {
	var out []string
	for _, l := range s.Links {
		if l.Relation == "component_of" && l.ChildID == s.ItemID && l.ParentID != "" && !slices.Contains(out, l.ParentID) {
			out = append(out, l.ParentID)
		}
	}
	return out
}

// HoldLevel — наибольший действующий уровень сдерживания, пришедший по
// генеалогии; пусто — нет.
func (s State) HoldLevel() string {
	best := ""
	for _, h := range s.Holds {
		if !h.Released && holdRank(h.Level) > holdRank(best) {
			best = h.Level
		}
	}
	return best
}

func holdRank(l string) int {
	switch l {
	case "observe":
		return 1
	case "additional_check":
		return 2
	case "item_hold":
		return 3
	case "lot_hold":
		return 4
	}
	return 0
}

// Reduce применяет запись входа изделия (факт, решение, адресованную запись
// стадии) к состоянию модуля (AD-5). Реакции в свёртку не входят (AD-3).
func Reduce(s State, r kernel.Record, env Env) State {
	if s.ItemID == "" {
		s.ItemID = r.ItemID
	}
	if s.RunID == "" {
		s.RunID = r.RunID
	}
	switch r.Type {
	case catalog.ItemItemRegistered:
		s = s.registered(r, env)
	case catalog.ItemCarrierApplied:
		s = s.carrierApplied(r)
	case catalog.ItemCarrierVerified:
		s = s.carrierVerified(r)
	case catalog.ItemCarrierRemoved:
		s = s.carrierRemoved(r)
	case catalog.ItemIdentificationConfirmed:
		s = s.identificationConfirmed(r)
	case catalog.ItemAssemblyRecorded:
		s = s.assembly(r, env)
	case catalog.ItemInterventionOpened:
		s = s.interventionOpened(r, env)
	case catalog.ItemInterventionClosed:
		s = s.interventionClosed(r)
	case catalog.ItemPresentationRecorded:
		s.Presentations = append(slices.Clone(s.Presentations), r.EventID)
	case catalog.ItemReleaseRecorded:
		if d, ok := decode[struct {
			WarehouseID string `json:"warehouse_id"`
		}](r); ok && !s.Released {
			at := r.OccurredAt
			s.Released, s.ReleasedAt, s.Warehouse = true, &at, d.WarehouseID
		}
	case catalog.OperationRunStarted, catalog.OperationRunFinished:
		s = s.run(r)
	case catalog.InspectionResultRecorded:
		s = s.inspected(r, env)
	case catalog.GenealogyLinkAdded:
		s = s.link(r)
	case catalog.GenealogyWitnessPropagated:
		s = s.witness(r)
	case catalog.GenealogyContainmentPropagated:
		s = s.hold(r)
	case catalog.BindingLinkResolved:
		s = s.binding(r)
	case catalog.ReferenceChangeAffectsItem:
		if d, ok := decode[struct {
			ReferenceEventID string `json:"reference_event_id"`
			ReferenceKind    string `json:"reference_kind"`
			ValidFrom        string `json:"valid_from"`
		}](r); ok {
			s.RefChanges = append(slices.Clone(s.RefChanges), RefChange{ReferenceEventID: d.ReferenceEventID, Kind: d.ReferenceKind, ValidFrom: d.ValidFrom, EventID: r.EventID})
		}
	}
	return s
}

func decode[T any](r kernel.Record) (T, bool) {
	var v T
	if len(r.Data) == 0 {
		return v, false
	}
	return v, json.Unmarshal(r.Data, &v) == nil
}

// LocalLabel — номер детали для людей по id: локальная часть без кода
// предприятия и префикса прогона; машинные F-/R-/C- — кириллицей, как на бирке.
func LocalLabel(id string) string {
	local := id
	if i := strings.IndexByte(local, ':'); i >= 0 {
		local = local[i+1:]
	}
	if i := strings.LastIndexByte(local, '/'); i >= 0 {
		local = local[i+1:]
	}
	for _, r := range [][2]string{{"F-", "Ф-"}, {"R-", "К-"}, {"C-", "КР-"}} {
		if strings.HasPrefix(local, r[0]) {
			return r[1] + local[len(r[0]):]
		}
	}
	return local
}

// DisplayLabel — метка изделия для задач, очередей и окон (кейс §4.6: люди
// видят номер детали, а не внутренний id): номер с бирки (Ф-001), иначе
// DM-код, иначе номер из id. Префикс прогона у значения носителя снимается.
func (s State) DisplayLabel(itemID string) string {
	for _, typ := range []string{"tag_qr", "dpm_datamatrix", "route_card"} {
		for _, c := range s.Carriers {
			if c.Type != typ || !c.Active() || c.Value == "" {
				continue
			}
			v := c.Value
			if i := strings.LastIndexByte(v, '/'); i >= 0 {
				v = v[i+1:]
			}
			v = strings.TrimPrefix(v, "TAG:")
			if typ == "dpm_datamatrix" {
				return v
			}
			return LocalLabel(v)
		}
	}
	return LocalLabel(itemID)
}
