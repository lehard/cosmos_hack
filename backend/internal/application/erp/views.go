package erp

import (
	"time"

	"ant/internal/application/platform"
)

// Формы ответов учёта (1С, Галактика): задания, исходящие учётные сообщения,
// состояние обмена (FR-90, FR-91, FR-96; AD-7, AD-18). Ось «учёт в 1С» меняет
// только квитанция (AD-30); состояние сообщения — словарь erp_message_status.

// ErpOrder — производственное задание из учётной системы (erp.order.received, кейс §1.5).
type ErpOrder struct {
	OrderID        string             `json:"order_id"`
	ExternalSystem string             `json:"external_system" enum:"onec,galaktika"`
	ExternalNumber string             `json:"external_number" doc:"Номер задания во внешней системе."`
	ItemTypeID     string             `json:"item_type_id"`
	ItemRevision   *string            `json:"item_revision,omitempty"`
	Quantity       int                `json:"quantity" minimum:"0"`
	Launched       int                `json:"launched" minimum:"0" doc:"Изделий запущено по заданию."`
	DueDate        *string            `json:"due_date,omitempty" doc:"Срок (дата)."`
	ReceivedAt     time.Time          `json:"received_at"`
	Ref            *platform.DrillRef `json:"ref,omitempty"`
}

// ErpOrderList — задания учётных систем.
type ErpOrderList struct {
	Items      []ErpOrder `json:"items"`
	NextCursor string     `json:"next_cursor,omitempty"`
}

// ErpMessageAttempt — попытка отправки и её квитанция или ошибка.
type ErpMessageAttempt struct {
	At                  time.Time `json:"at"`
	Outcome             string    `json:"outcome" enum:"accepted,duplicate,rejected,transport_error"`
	ExternalDocumentRef *string   `json:"external_document_ref,omitempty"`
	ErrorCode           *string   `json:"error_code,omitempty"`
	ErrorMessage        *string   `json:"error_message,omitempty"`
}

// ErpMessage — исходящее учётное сообщение по бизнес-ключу (AD-7): субъект,
// учётное действие, закрывающая точка; версии при исправлениях.
type ErpMessage struct {
	BusinessKey     string              `json:"business_key"`
	ExternalSystem  string              `json:"external_system" enum:"onec,galaktika"`
	Action          string              `json:"action" enum:"accept_into_work,warehouse_transfer,scrap_transfer_rework,scrap_transfer_writeoff,scrap_transfer_reprocess,return_to_supplier,release,inspection_result,return_from_defect" doc:"Учётное действие порта учёта; return_from_defect — «возврат из брака в производство» (Д-17)."`
	ItemID          *string             `json:"item_id,omitempty"`
	LotID           *string             `json:"lot_id,omitempty"`
	MessageVersion  int                 `json:"message_version" minimum:"1"`
	Status          string              `json:"status" enum:"queued,sent,acknowledged,rejected,quarantined" doc:"Словарь erp_message_status (не ось)."`
	AccountingState *string             `json:"accounting_state,omitempty" doc:"Ось «учёт в 1С» после квитанции (axis_erp_accounting)."`
	AfterRework     bool                `json:"after_rework"`
	RequestEventID  string              `json:"request_event_id" doc:"Реакция erp.posting.requested."`
	RequestedAt     time.Time           `json:"requested_at"`
	Attempts        []ErpMessageAttempt `json:"attempts"`
	BasisSeq        int64               `json:"basis_seq"`
}

// ErpMessageList — исходящие учётные сообщения.
type ErpMessageList struct {
	Items      []ErpMessage `json:"items"`
	NextCursor string       `json:"next_cursor,omitempty"`
}

// ErpChannel — состояние канала обмена с внешней системой (AD-18: при
// расхождении метаданных — degraded, отправка остановлена).
type ErpChannel struct {
	System          string     `json:"system" enum:"onec,galaktika,mes,kompas"`
	State           string     `json:"state" enum:"ok,degraded,disabled"`
	Endpoint        string     `json:"endpoint" doc:"Адрес (stand или реальная система)."`
	Stand           bool       `json:"stand" doc:"Работает stand — эмулятор кейса."`
	ContractVersion string     `json:"contract_version"`
	Detail          *string    `json:"detail,omitempty"`
	LastExchangeAt  *time.Time `nullable:"true" json:"last_exchange_at"`
	Queued          int        `json:"queued" minimum:"0"`
	Quarantined     int        `json:"quarantined" minimum:"0"`
}

// ErpChannelList — каналы обмена.
type ErpChannelList struct {
	Items []ErpChannel `json:"items"`
}

// ErpReason — основание решения: код и текст.
type ErpReason struct {
	Code *string `json:"code,omitempty"`
	Text string  `json:"text" minLength:"1" maxLength:"2000"`
}

// ResendPosting — ручная переотправка сообщения из карантина (erp.posting.resend_requested, AD-18).
type ResendPosting struct {
	platform.CommandHeader
	RequestEventID string    `json:"request_event_id" format:"uuid"`
	Reason         ErpReason `json:"reason"`
}

// CompensatePosting — решение человека по новой версии сообщения с тем же
// бизнес-ключом (erp.posting.compensation_decided, AD-7): отправить исправление
// (сторно + новое) или оставить как отправлено.
type CompensatePosting struct {
	platform.CommandHeader
	SupersededRequestEventID string    `json:"superseded_request_event_id" format:"uuid"`
	NewRequestEventID        string    `json:"new_request_event_id,omitempty" format:"uuid"`
	Decision                 string    `json:"decision" enum:"send_correction,keep_as_sent"`
	Reason                   ErpReason `json:"reason"`
}

// MessageFilter — фильтр исходящих сообщений.
type MessageFilter struct {
	ItemID string
	Status string
	System string
}
