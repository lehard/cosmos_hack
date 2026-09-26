package mes

import (
	"context"
	"errors"
	"fmt"
	"time"

	dom "ant/internal/domain/mes"
)

// Ведомые порты модуля mes (AD-18, AD-35: внешняя система — только за портом).

// Channel — канал MES на нашем языке: сверка контракта, блок или снятие
// блока изделия или партии, входящие задания и события операций. Адаптер:
// B2MML-JSON (infrastructure/integration/mes/b2mml); конкретная MES (1С:MES,
// Галактика MES, ГОЛЬФСТРИМ) — другой адаптер того же порта.
type Channel interface {
	Info() ChannelInfo
	// Check — сверка ответной стороны (AD-18); расхождение — *ContractError.
	Check(ctx context.Context) (string, error)
	// Post — блок или снятие; ошибка данных — ответ rejected; транспорт —
	// *TransportError (повтор с тем же номером); контракт — *ContractError.
	Post(ctx context.Context, m HoldMessage) (Response, error)
	// Pull — входящие сообщения MES; повторное чтение даёт те же номера.
	Pull(ctx context.Context) (Inbound, error)
}

// ChannelInfo — канал обмена: система, адрес, stand, версия контракта.
type ChannelInfo struct {
	System, Endpoint, ContractVersion string
	Stand                             bool
}

// HoldMessage — блок или снятие блока в MES.
type HoldMessage struct {
	// MessageID — номер сообщения (BODID): UUIDv5 от бизнес-ключа (AD-7).
	MessageID string
	Key       string
	Hold      bool
	// ItemID или LotID — наш субъект; ExternalID — ID субъекта в MES по
	// соответствию (пусто — MES получает наш ID).
	ItemID, LotID, ExternalID string
	// Reason — пояснение для мастера.
	Reason     string
	OccurredAt time.Time
	Attempt    int
}

// Response — ответ MES.
type Response struct {
	// Outcome — accepted | duplicate | rejected.
	Outcome string
	Code    string
	Message string
}

// Inbound — входящие сообщения MES на нашем языке.
type Inbound struct {
	Jobs   []dom.Job
	Events []dom.OperationEvent
	// Rejected — сообщения не по контракту (номер и причина): фактов не дают.
	Rejected []dom.Deferred
}

// TransportError — транспортная ошибка: 5xx, таймаут, разрыв, нет подтверждения.
type TransportError struct {
	HTTPStatus int
	Err        error
}

func (e *TransportError) Error() string {
	if e.HTTPStatus > 0 {
		return fmt.Sprintf("транспорт MES: HTTP %d: %v", e.HTTPStatus, e.Err)
	}
	return "транспорт MES: " + e.Err.Error()
}

func (e *TransportError) Unwrap() error { return e.Err }

// ContractError — несовместимость контракта или сообщение не по схеме до
// отправки (Local, FR-111): канал degraded, блоки не отправляются.
type ContractError struct {
	Detail string
	Local  bool
}

func (e *ContractError) Error() string { return "контракт MES: " + e.Detail }

// AsTransport, AsContract — классификация ошибки канала.
func AsTransport(err error) (*TransportError, bool) {
	var t *TransportError
	return t, errors.As(err, &t)
}

// AsContract — ошибка несовместимости контракта.
func AsContract(err error) (*ContractError, bool) {
	var c *ContractError
	return c, errors.As(err, &c)
}

// Intake — ведомый порт входа (AD-18: входящие — через обычный приём).
type Intake interface {
	Submit(ctx context.Context, sourceID string, envelopes [][]byte) error
}
