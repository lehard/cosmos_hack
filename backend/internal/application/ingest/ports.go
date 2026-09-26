package ingest

import (
	"context"
	"errors"
	"time"

	"ant/internal/application/journal"
	"ant/internal/contracts/errcodes"
	dom "ant/internal/domain/ingest"
)

// Registry — ведомый порт реестра приёма (схема Postgres ingest): ключи
// идемпотентности source_id + event_id с отпечатком содержимого (AD-7, FR-31) и
// учёт source_seq по источникам (FR-39). Адаптеры: infrastructure/storage/ingest
// (Postgres), inmem (память — тесты, демо edge-агента).
type Registry interface {
	// Seen — запись реестра по ключу; nil — ключ не встречался.
	Seen(ctx context.Context, sourceID, eventID string) (*dom.Seen, error)
	// SourceState — учёт номеров источника (пустой — источник новый).
	SourceState(ctx context.Context, sourceID string) (dom.SourceState, error)
	// Commit — атомарно запомнить принятое сообщение и новое состояние источника.
	Commit(ctx context.Context, seen dom.Seen, state dom.SourceState) error
	// SaveSourceState — сохранить состояние источника (досылка в карантин, пометка разрывов).
	SaveSourceState(ctx context.Context, state dom.SourceState) error
	// Sources — состояния всех источников, отсортированные по source_id.
	Sources(ctx context.Context) ([]dom.SourceState, error)
}

// QuarantineStatus — состояние записи карантина.
type QuarantineStatus string

// Состояния записи карантина (FR-30).
const (
	QuarantineOpen        QuarantineStatus = "open"
	QuarantineReprocessed QuarantineStatus = "reprocessed"
	QuarantineDiscarded   QuarantineStatus = "discarded"
)

// QuarantineRecord — сообщение в карантине (FR-30, AD-2): содержимое — в
// MaterialStore по адресу H(байты), здесь — метаданные и ссылка на служебную
// запись журнала ingest.message.quarantined.
type QuarantineRecord struct {
	// ID — event_id служебной записи ingest.message.quarantined.
	ID string
	// JournalSeq — её позиция в журнале.
	JournalSeq int64
	SourceID   string
	EventID    string
	SourceSeq  int64
	EventType  string
	// Fingerprint — отпечаток исходных байтов; MaterialAddress — адрес содержимого.
	Fingerprint     string
	MaterialAddress string
	Code            errcodes.Code
	Field           string
	Detail          string
	Status          QuarantineStatus
	ReceivedAt      time.Time
	// Raw — исходные байты, если хранилища материалов нет (упрощённый режим).
	Raw []byte
	// ResolvedBy — event_id записи ingest.message.reprocessed.
	ResolvedBy string
}

// QuarantineFilter — отбор карантина.
type QuarantineFilter struct {
	Status   QuarantineStatus
	SourceID string
	Code     errcodes.Code
	Limit    int
}

// QuarantineStore — ведомый порт хранилища карантина (FR-30): повтор тех же
// байтов с тем же кодом не создаёт второй записи.
type QuarantineStore interface {
	// Find — запись по отпечатку исходных байтов и коду; nil — нет.
	Find(ctx context.Context, fingerprint string, code errcodes.Code) (*QuarantineRecord, error)
	// Put — сохранить новую запись.
	Put(ctx context.Context, r QuarantineRecord) error
	// Get — запись по ID; ErrNotFound — нет.
	Get(ctx context.Context, id string) (QuarantineRecord, error)
	// List — записи по фильтру, новые сначала.
	List(ctx context.Context, f QuarantineFilter) ([]QuarantineRecord, error)
	// Count — число записей в состоянии (пусто — все).
	Count(ctx context.Context, status QuarantineStatus) (int64, error)
	// Resolve — отметить итог переобработки.
	Resolve(ctx context.Context, id string, status QuarantineStatus, resolvedBy string) error
}

// ErrNotFound — записи нет.
var ErrNotFound = errors.New("не найдено")

// KeyInfo — ключ источника в реестре ключей (AD-11, FR-26, FR-70).
type KeyInfo struct {
	// KeyRef — key_id@версия.
	KeyRef string
	// SourceID — источник, которому выдан ключ (устройство edge-агента, шлюз, терминал).
	SourceID string
	// Provenance — класс происхождения подписи этим ключом (AD-2): device, personal, server_attested, partner, scenario.
	Provenance string
	// Revoked — ключ отозван (акт отзыва).
	Revoked bool
}

// Ошибки реестра ключей.
var (
	ErrUnknownSource = errors.New("источник не зарегистрирован")
	ErrUnknownKey    = errors.New("ключ не зарегистрирован")
)

// KeyRegistry — ведомый порт реестра источников и ключей (FR-26): доверие ключу —
// от акта регистрации (AD-11). Реализация — модуль signing (эпики 05, 27); до
// них — адаптер в памяти из генезиса демо.
type KeyRegistry interface {
	// Source — зарегистрирован ли источник; ErrUnknownSource — нет.
	Source(ctx context.Context, sourceID string) error
	// Key — ключ по key_id@версия; ErrUnknownKey — нет.
	Key(ctx context.Context, keyRef string) (KeyInfo, error)
}

// SignatureFailure — отказ проверки подписи для шины безопасности (AD-2, AD-24).
type SignatureFailure struct {
	SourceID     string
	EventID      string
	KeyRef       string
	Failure      string // значение перечисления security.signature.invalid.failure
	QuarantineID string
	OccurredAt   time.Time
}

// IdempotencyConflict — конфликт целостности (AD-7, FR-31).
type IdempotencyConflict struct {
	SourceID        string
	EventID         string
	FirstDigest     string
	ConflictDigest  string
	FirstSeq        int64
	QuarantineID    string
	OccurredAt      time.Time
	FirstJournalRef string
}

// SecurityBus — ведомый порт шины безопасности (AD-24): записи security.*
// (эмитент — модуль security, AD-40) и записи цепочки критических действий
// добавляются в ту же пачку Append, что и факт карантина. Реализация — модуль
// security (эпик 29); до него — мост JournalSecurityBus (security_bridge.go).
type SecurityBus interface {
	SignatureInvalid(ctx context.Context, f SignatureFailure) (main, critical []journal.Pending, err error)
	IdempotencyConflict(ctx context.Context, c IdempotencyConflict) (main, critical []journal.Pending, err error)
}
