package item

import "ant/internal/application/platform"

// RegisterItem — зарегистрировать изделие и запустить в работу (item.item.registered, FR-42, AD-16):
// ID рождается в системе и из метки не выводится; закрепляет версию процесса (AD-17).
type RegisterItem struct {
	platform.CommandHeader
	LocalID      string   `json:"local_id,omitempty" maxLength:"96" doc:"Локальный номер изделия (например, F-031); пусто — система выдаёт сама. Из метки не выводится (AD-16)."`
	ItemTypeID   string   `json:"item_type_id" maxLength:"128"`
	ItemRevision string   `json:"item_revision" maxLength:"64"`
	OrderID      string   `json:"order_id,omitempty" maxLength:"128" doc:"Задание 1С (сквозной сценарий §1.5)."`
	LotIDs       []string `json:"lot_ids,omitempty"`
	EntryStepKey string   `json:"entry_step_key,omitempty" maxLength:"128"`
	IsAssembly   bool     `json:"is_assembly,omitempty"`
}

// ApplyCarrier — нанести носитель (item.carrier.applied, AD-16); перемаркировка — replaces_value.
type ApplyCarrier struct {
	platform.CommandHeader
	CarrierType   string `json:"carrier_type" enum:"dpm_datamatrix,tag_qr,container_cell,route_card,post_context,manual_entry,internal_id"`
	Value         string `json:"value" minLength:"1" maxLength:"256"`
	ZoneID        string `json:"zone_id,omitempty" maxLength:"128"`
	IsTemporary   bool   `json:"is_temporary"`
	ReplacesValue string `json:"replaces_value,omitempty" maxLength:"256"`
}

// RemoveCarrier — снять носитель (item.carrier.removed).
type RemoveCarrier struct {
	platform.CommandHeader
	CarrierType string `json:"carrier_type" enum:"dpm_datamatrix,tag_qr,container_cell,route_card,post_context,manual_entry,internal_id"`
	Value       string `json:"value" minLength:"1" maxLength:"256"`
	ReasonText  string `json:"reason,omitempty" maxLength:"1000"`
}

// RecordPresentation — предъявить изделие ОТК или представителю заказчика (item.presentation.recorded, FR-19).
type RecordPresentation struct {
	platform.CommandHeader
	StepKey        string `json:"step_key" maxLength:"128"`
	PresentationNo int    `json:"presentation_no" minimum:"1" doc:"Номер предъявления: повторное — подписант выше."`
	PresentedTo    string `json:"presented_to" enum:"qc,customer_representative"`
}

// OpenIntervention — открыть вмешательство в собранное изделие (item.intervention.opened, FR-21).
type OpenIntervention struct {
	platform.CommandHeader
	ZoneIDs           []string `json:"zone_ids" minItems:"1"`
	RemovedComponents []string `json:"removed_components,omitempty"`
	Purpose           string   `json:"purpose" minLength:"1" maxLength:"1000"`
}

// CloseIntervention — закрыть вмешательство после повторной проверки зоны (item.intervention.closed; критическое).
type CloseIntervention struct {
	platform.CommandHeader
	RecheckEventIDs []string `json:"recheck_event_ids" minItems:"1" doc:"Результаты повторной проверки зоны."`
	RetestRequired  bool     `json:"retest_required,omitempty"`
}

// ConfirmIdentification — повторная идентификация человеком с подписью (item.identification.confirmed, AD-16; критическое).
type ConfirmIdentification struct {
	platform.CommandHeader
	Method            string `json:"method" enum:"dpm_datamatrix,tag_qr,container_cell,route_card,post_context,manual_entry,internal_id"`
	CarrierValue      string `json:"carrier_value,omitempty" maxLength:"256"`
	QuestionedEventID string `json:"questioned_event_id" format:"uuid" doc:"Запись «идентификация под сомнением»."`
}

// RecordAssembly — установить компонент в сборку (item.assembly.recorded → genealogy.link.added, FR-45).
type RecordAssembly struct {
	platform.CommandHeader
	ComponentItemID string `json:"component_item_id,omitempty" maxLength:"128"`
	ComponentLotID  string `json:"component_lot_id,omitempty" maxLength:"128"`
	ComponentTypeID string `json:"component_type_id" maxLength:"128"`
	Quantity        int    `json:"quantity,omitempty" minimum:"1"`
	Position        string `json:"position,omitempty" maxLength:"128"`
	BindingMethod   string `json:"binding_method" enum:"dpm_datamatrix,tag_qr,container_cell,route_card,post_context,manual_entry,internal_id"`
	OperationRunID  string `json:"operation_run_id,omitempty" maxLength:"128"`
}

// RecordRelease — принять изделие на склад выпуска (item.release.recorded, закрывающая точка → 1С «выпуск»).
type RecordRelease struct {
	platform.CommandHeader
	WarehouseID  string `json:"warehouse_id" maxLength:"128"`
	ConcessionID string `json:"concession_id,omitempty" maxLength:"128"`
	AfterRework  bool   `json:"after_rework" doc:"Принято после переделки (AD-18)."`
}

// SplitPart — изделие, выделяемое разделением.
type SplitPart struct {
	ItemID       string   `json:"item_id,omitempty" maxLength:"128" doc:"Внутренний ID части; пусто — ‹исходный›-‹n›."`
	ItemTypeID   string   `json:"item_type_id,omitempty" maxLength:"128" doc:"Тип части; пусто — тип исходного."`
	ItemRevision string   `json:"item_revision,omitempty" maxLength:"64"`
	LotIDs       []string `json:"lot_ids,omitempty" doc:"Дополнительные партии части (партии исходного переносятся сами)."`
}

// SplitItem — разделение 1→N (FR-15): каждая часть регистрируется
// (item.item.registered с split_from); происхождение исходного переносит
// межизделийная стадия (genealogy.link.added, relation = split_from).
type SplitItem struct {
	platform.CommandHeader
	Parts []SplitPart `json:"parts" minItems:"1" maxItems:"100"`
}
