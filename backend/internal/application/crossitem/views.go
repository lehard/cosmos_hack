package crossitem

import (
	"time"

	"ant/internal/application/platform"
)

// Формы межизделийной стадии (AD-42): партии, группы, привязка событий.
// Генеалогия изделия показывается в паспорте (item.genealogy.read).

// Lot — партия, садка или плавка (genealogy.lot.*, FR-15, FR-45).
type Lot struct {
	LotID              string     `json:"lot_id"`
	ItemTypeID         string     `json:"item_type_id,omitempty"`
	Supplier           string     `json:"supplier,omitempty"`
	ExternalRef        string     `json:"external_ref,omitempty" doc:"Номер партии во внешней системе (соответствие — reference.external_id.mapped)."`
	DeclaredQuantity   int        `json:"declared_quantity" minimum:"0" doc:"Количество по документам поставщика."`
	ActualQuantity     *int       `nullable:"true" json:"actual_quantity" minimum:"0" doc:"Фактическое количество при регистрации; null — ещё не зарегистрирована."`
	IssuedQuantity     int        `json:"issued_quantity" minimum:"0"`
	CertificatePresent *bool      `nullable:"true" json:"certificate_present"`
	Status             string     `json:"status" enum:"received,registered,accepted,rejected,on_hold,issued,consumed" doc:"Состояние партии; блок партии — ось «сдерживание» nonconformity."`
	Containment        string     `json:"containment,omitempty" doc:"Сдерживание партии (ось containment, владелец nonconformity)."`
	ReceivedAt         *time.Time `json:"received_at,omitempty"`
	RegisteredAt       *time.Time `json:"registered_at,omitempty"`
	BasisSeq           int64      `json:"basis_seq"`
	Kind               string     `json:"kind,omitempty" doc:"Вид: lot — партия, heat — плавка (FR-45)."`
	HeatNo             string     `json:"heat_no,omitempty" doc:"Номер плавки партии."`
}

// LotList — партии.
type LotList struct {
	Items      []Lot  `json:"items"`
	NextCursor string `json:"next_cursor,omitempty"`
}

// LotItem — изделие, сделанное из партии или выданное из неё.
type LotItem struct {
	ItemID   string `json:"item_id"`
	Label    string `json:"label"`
	Relation string `json:"relation" enum:"made_from_lot,issued" doc:"Связь генеалогии."`
	// Assembled, Via — изделие собрано из изделия партии (FR-45: «партия → все
	// изделия, включая собранные»).
	Assembled bool   `json:"assembled,omitempty" doc:"Сборка, в которую вошло изделие партии (FR-45)."`
	Via       string `json:"via,omitempty" doc:"Изделие партии, через которое сборка попала в список."`
}

// LotIssue — выдача из партии.
type LotIssue struct {
	EventID      string    `json:"event_id"`
	Quantity     int       `json:"quantity" minimum:"1"`
	OrderID      string    `json:"order_id,omitempty"`
	ToLocationID string    `json:"to_location_id,omitempty"`
	IssuedBy     string    `json:"issued_by"`
	At           time.Time `json:"at"`
}

// LotCard — карточка партии: входной контроль, выдачи, изделия, документы.
type LotCard struct {
	Lot
	Issues    []LotIssue          `json:"issues"`
	Items     []LotItem           `json:"items"`
	Documents []platform.DrillRef `json:"documents" doc:"Акт входного контроля, ярлык соответствия, выписка поставщика."`
}

// ItemGroup — временная группа изделий: садка, групповая операция, транспорт (genealogy.group.*, FR-15).
type ItemGroup struct {
	GroupID       string     `json:"group_id"`
	Kind          string     `json:"kind" enum:"charge,batch_operation,transport,other"`
	ItemIDs       []string   `json:"item_ids"`
	WitnessItemID string     `json:"witness_item_id,omitempty" doc:"Образец-свидетель."`
	FormedAt      time.Time  `json:"formed_at"`
	DissolvedAt   *time.Time `json:"dissolved_at,omitempty"`
}

// TraceItem — изделие в ответе «партия / плавка / садка → все изделия» (FR-45).
type TraceItem struct {
	ItemID    string `json:"item_id"`
	Label     string `json:"label"`
	Relation  string `json:"relation" doc:"made_from_lot — из партии или плавки; grouped_with — в садке; assembled — собрано из них."`
	Via       string `json:"via,omitempty" doc:"Изделие, через которое сборка попала в список."`
	Assembled bool   `json:"assembled"`
}

// Trace — «партия / плавка / садка → все изделия, включая собранные» (FR-45).
type Trace struct {
	LotID   string      `json:"lot_id,omitempty"`
	HeatNo  string      `json:"heat_no,omitempty"`
	GroupID string      `json:"group_id,omitempty"`
	Items   []TraceItem `json:"items"`
}

// UnboundEvent — событие без изделия в межизделийной стадии (AD-41, FR-34):
// носитель, кандидаты, текущая привязка — для ручной привязки контролёром.
type UnboundEvent struct {
	EventID    string    `json:"event_id"`
	EventType  string    `json:"event_type"`
	OccurredAt time.Time `json:"occurred_at"`
	SourceID   string    `json:"source_id,omitempty"`
	CarrierRef string    `json:"carrier_ref,omitempty"`
	Candidates []string  `json:"candidates"`
	BoundTo    string    `json:"bound_to,omitempty"`
	Basis      string    `json:"basis,omitempty" doc:"carrier — по носителю; manual — человеком."`
	RunID      string    `json:"run_id,omitempty"`
}

// UnboundList — события без изделия: неразрешённые и неоднозначные.
type UnboundList struct {
	Items []UnboundEvent `json:"items"`
}

// ItemGroupList — группы.
type ItemGroupList struct {
	Items []ItemGroup `json:"items"`
}
