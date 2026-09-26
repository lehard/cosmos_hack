package erp

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	dom "ant/internal/domain/erp"
)

// Ведомые порты модуля erp (AD-18, AD-35: внешняя система — только за
// портом, адаптеры — stand-ы и реальные системы с контрактными тестами).

// Ledger — порт учёта учётной системы на нашем языке (AD-18): исходящие
// учётные действия и результат контроля, сверка ответной стороны, входящие
// задания, номенклатура и партии. Адаптеры: 1С (OData v3 + HTTP-сервис qc.v1,
// infrastructure/integration/erp/onec), Галактика (эпик 31).
type Ledger interface {
	// Info — система, адрес, stand ли это, версия контракта адаптера.
	Info() LedgerInfo
	// Check — сверка ответной стороны при старте и по расписанию (AD-18):
	// метаданные и версия контракта. Расхождение — *ContractError.
	Check(ctx context.Context) (Checked, error)
	// Post — отправить сообщение. Квитанция, повтор и ошибка данных — ответ;
	// транспортная ошибка — *TransportError (повтор с тем же номером);
	// несовместимость контракта — *ContractError (канал degraded).
	Post(ctx context.Context, m Outgoing) (Response, error)
	// Pull — входящие факты порта учёта (задания, номенклатура, партии) и
	// соответствия внешних ID; повторное чтение даёт те же event_id (AD-7).
	Pull(ctx context.Context) ([]Inbound, error)
}

// LedgerInfo — канал обмена: система, адрес, stand, версия контракта.
type LedgerInfo struct {
	System          string
	Endpoint        string
	Stand           bool
	ContractVersion string
}

// Checked — итог сверки ответной стороны.
type Checked struct {
	// ContractVersion — версия контракта, которую объявила ответная сторона.
	ContractVersion string
	// Detail — что сверено (для людей).
	Detail string
}

// Outgoing — исходящее сообщение: номер (ключ идемпотентности), версия,
// учётное действие и содержимое на нашем языке; перевод в формат учётной
// системы — дело адаптера.
type Outgoing struct {
	MessageID  string
	Version    int
	Request    ev.ErpPostingRequestedV1
	OccurredAt time.Time
	// CorrectsMessageID — исправление: сторно прежнего сообщения и новое
	// содержимое одной транзакцией учётной системы (AD-7, по решению человека).
	CorrectsMessageID string
	// Attempt — номер попытки (1…).
	Attempt int
}

// Action — учётное действие сообщения.
func (o Outgoing) Action() dom.Action { return dom.Action(o.Request.Action) }

// Response — ответ учётной системы на сообщение.
type Response struct {
	// Outcome — accepted (квитанция), duplicate (повтор: та же квитанция),
	// rejected (ошибка данных, без автоповтора).
	Outcome     ev.ErpPostingRespondedV1Outcome
	Receipt     string
	DocumentRef string
	// Code, Message, Field — ошибка учётной системы (для rejected); ErrorCode —
	// наш код из contracts/errors.yaml.
	Code, Message, Field string
	ErrorCode            string
	HTTPStatus           int
}

// TransportError — транспортная ошибка: 5xx, таймаут, разрыв, нет ответа.
// Повтор — только при ней, с тем же номером сообщения (FR-96).
type TransportError struct {
	HTTPStatus int
	Err        error
}

func (e *TransportError) Error() string {
	if e.HTTPStatus > 0 {
		return fmt.Sprintf("транспорт: HTTP %d: %v", e.HTTPStatus, e.Err)
	}
	return "транспорт: " + e.Err.Error()
}

func (e *TransportError) Unwrap() error { return e.Err }

// ContractError — несовместимость контракта с ответной стороной или
// сообщение, не проходящее схему контракта до отправки (FR-111, О8): канал
// degraded, результат не отправляется.
type ContractError struct {
	Detail string
	// Local — ошибка поймана до отправки (сообщение не по схеме контракта).
	Local bool
}

func (e *ContractError) Error() string { return "контракт: " + e.Detail }

// AsTransport, AsContract — классификация ошибки порта учёта.
func AsTransport(err error) (*TransportError, bool) {
	var t *TransportError
	return t, errors.As(err, &t)
}

// AsContract — ошибка несовместимости контракта.
func AsContract(err error) (*ContractError, bool) {
	var c *ContractError
	return c, errors.As(err, &c)
}

// Inbound — входящий факт порта учёта: тип своего семейства (или
// reference.external_id.mapped), детерминированный event_id (UUIDv5 от
// внешнего ключа и версии данных), время и data по схеме контракта.
type Inbound struct {
	Type       catalog.Type
	EventID    string
	OccurredAt time.Time
	Data       any
}

// Intake — ведомый порт входа (AD-18: входящие — через обычный приём):
// пачка конвертов источника-шлюза уходит в приём; ответ — итог по каждому.
type Intake interface {
	Submit(ctx context.Context, sourceID string, envelopes [][]byte) (IntakeResult, error)
}

// IntakeResult — итог приёма пачки.
type IntakeResult struct {
	Accepted, Duplicates, Quarantined int
}

// Channel — состояние канала обмена (AD-18): ok | degraded | disabled.
type Channel struct {
	System          string
	State           string
	Endpoint        string
	Stand           bool
	ContractVersion string
	Detail          string
	CheckedAt       time.Time
	LastExchangeAt  *time.Time
}

// OutboxRow — строка очереди исходящих роли outbox: сообщение глазами
// журнала (dom.View — пишет только потребитель журнала) и транспортные поля
// (пишет только отправитель): попытки, следующая попытка, обработанное
// поколение. Разные колонки — нет потерянных обновлений между ними.
type OutboxRow struct {
	View dom.View
	// Token — поколение отправки: запрос # переотправки (меняет потребитель).
	Token string
	// Handled — поколение, по которому отправитель уже записал ответ или карантин.
	Handled string
	// RetryToken, Tries, NextAt, LastError — транспортные попытки поколения RetryToken.
	RetryToken string
	Tries      int
	NextAt     time.Time
	LastError  string
}

// TokenOf — поколение отправки сообщения: запрос и число переотправок.
func TokenOf(v dom.View) string { return fmt.Sprintf("%s#%d", v.RequestEventID, v.Generation) }

// OutboxStore — хранилище очереди исходящих и состояния каналов (схема erp,
// AD-1). Запись строк очереди — эффектами в транзакции Append (курсор
// потребителя и записи журнала атомарно, AD-45); каналы — напрямую
// (операционное состояние, не проекция журнала).
type OutboxStore interface {
	// Row — строка очереди по бизнес-ключу.
	Row(ctx context.Context, key string) (OutboxRow, bool, error)
	// Due — сообщения системы к отправке на момент now: в очереди, поколение
	// не обработано, срок следующей попытки наступил.
	Due(ctx context.Context, system string, now time.Time, limit int) ([]OutboxRow, error)
	// Counts — в очереди и в карантине по системе.
	Counts(ctx context.Context, system string) (queued, quarantined int, err error)
	// Channel, SetChannel, Channels — состояние каналов обмена.
	Channel(ctx context.Context, system string) (Channel, bool, error)
	SetChannel(ctx context.Context, c Channel) error
	Channels(ctx context.Context) ([]Channel, error)
	// NextSourceSeq — следующий source_seq шлюза входящих (AD-7).
	NextSourceSeq(ctx context.Context, sourceID string, n int) (int64, error)
}

// Эффекты транзакции Append модуля erp (применяет infrastructure/storage/erp).

// OutboxView — строка очереди глазами журнала (пишет потребитель erp.outbox).
type OutboxView struct {
	View  dom.View
	Token string
}

// EffectKind — вид эффекта.
func (OutboxView) EffectKind() string { return "erp.outbox.view" }

// OutboxTransport — транспортные поля строки очереди (пишет отправитель).
type OutboxTransport struct {
	Key        string
	Handled    string
	RetryToken string
	Tries      int
	NextAt     time.Time
	LastError  string
}

// EffectKind — вид эффекта.
func (OutboxTransport) EffectKind() string { return "erp.outbox.transport" }
