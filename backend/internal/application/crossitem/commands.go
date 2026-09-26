package crossitem

import "ant/internal/application/platform"

// RegisterLot — регистрация поступившей партии кладовщиком (genealogy.lot.registered, FR-15).
type RegisterLot struct {
	platform.CommandHeader
	ActualQuantity     int  `json:"actual_quantity" minimum:"0"`
	PackagingOK        bool `json:"packaging_ok"`
	CertificatePresent bool `json:"certificate_present"`
}

// IssueLot — выдача из партии в производство (genealogy.lot.issued, FR-45).
type IssueLot struct {
	platform.CommandHeader
	Quantity     int      `json:"quantity" minimum:"1"`
	ItemIDs      []string `json:"item_ids,omitempty"`
	OrderID      string   `json:"order_id,omitempty" maxLength:"128"`
	ToLocationID string   `json:"to_location_id,omitempty" maxLength:"128"`
}

// BindingReason — основание ручной привязки.
type BindingReason struct {
	Code string `json:"code,omitempty" maxLength:"64"`
	Text string `json:"text" maxLength:"2000"`
}

// AssignBinding — ручная привязка события без изделия или перепривязка
// (binding.link.assigned, AD-41, FR-34): решение контролёра; обоих изделий
// касается пересвёртка; критическое действие группы protected_data.
type AssignBinding struct {
	platform.CommandHeader
	SubjectEventID string        `json:"subject_event_id" format:"uuid" doc:"Событие, которое привязывается."`
	ItemID         string        `json:"item_id" maxLength:"128"`
	PreviousItemID string        `json:"previous_item_id,omitempty" maxLength:"128"`
	Method         string        `json:"method" enum:"scan,select_expected,manual_entry"`
	Reason         BindingReason `json:"reason"`
}

// FormGroup — сформировать временную группу: садку, групповую операцию,
// транспорт (genealogy.group.formed, FR-15): результат образца-свидетеля
// распространяется на всю группу.
type FormGroup struct {
	platform.CommandHeader
	GroupID       string   `json:"group_id,omitempty" maxLength:"128" doc:"ID группы; пусто — система выдаёт сама."`
	Kind          string   `json:"kind" enum:"charge,batch_operation,transport,other"`
	ItemIDs       []string `json:"item_ids" minItems:"1" maxItems:"500"`
	WitnessItemID string   `json:"witness_item_id,omitempty" maxLength:"128" doc:"Образец-свидетель."`
}

// DissolveGroup — расформировать группу (genealogy.group.dissolved, FR-15).
type DissolveGroup struct {
	platform.CommandHeader
}
