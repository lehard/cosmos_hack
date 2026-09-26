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

// ItemGroupList — группы.
type ItemGroupList struct {
	Items []ItemGroup `json:"items"`
}
