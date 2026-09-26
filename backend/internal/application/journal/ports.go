package journal

import (
	"context"
	"errors"
	"time"

	"ant/internal/application/platform"
	jc "ant/internal/contracts/journal"
)

// Append — единственная функция записи журнала (AD-44) — метод порта
// JournalStore. В одной транзакции: проверка аренды с эпохой (ErrFenced),
// блокировка головы основной цепочки, затем цепочки критических действий,
// проверки конкурентности AD-39, вычисление звеньев, вставка; проекции и
// курсор потребителя — в той же транзакции (AD-45).

// JournalStore — ведомый порт журнала (AD-35, ключ journal_store): Postgres
// в MVP, BFT-реестр — замена (описание). Проверка — верификатор.
type JournalStore interface {
	// Append записывает пачку (не больше K записей) и, если есть, записи
	// критических действий одной транзакцией (AD-8, AD-28, AD-44).
	Append(ctx context.Context, rq AppendRequest) (AppendResult, error)
	// Read — записи потока или партиции в порядке seq, с фильтром момента
	// (AD-22): «что мы знали» — префикс по seq до recorded_at ≤ T.
	Read(ctx context.Context, q ReadQuery) ([]jc.JournalEntry, error)
	// Head — головы цепочек (seq, звено) для хранителя (AD-8).
	Head(ctx context.Context) (Heads, error)
	// Open — расшифрованный конверт записи (AD-23); нет KEK — ошибка.
	Open(ctx context.Context, e jc.JournalEntry) (Envelope, error)
}

// Pending — запись, подготовленная к Append: открытые поля (без seq,
// committed_at, commit, link — их ставит Append) и канонический конверт DSSE.
type Pending struct {
	Entry    jc.JournalEntry
	Envelope []byte
}

// AppendRequest — пачка записи.
type AppendRequest struct {
	Batch    []Pending
	Critical []Pending
	// Fence — аренда партиции с эпохой; запись от копии, потерявшей аренду, — ErrFenced (AD-6).
	Fence *Fence
	// Checks — проверки AD-39: потоки и их basis_seq, policy_seq субъекта, расход разрешения.
	Checks []Check
	// Consumer — курсор потребителя, обновляемый в той же транзакции (AD-45).
	Consumer *CursorAdvance
	// Effects — записи проекций потребителя в той же транзакции (AD-44,
	// AD-45): проекции изделия, вклады показателей, журнал изменений для SSE.
	// Пачка может быть пустой: проектор без выходных записей сдвигает курсор
	// вместе с проекцией. Виды эффектов объявляет модуль-писатель проекции
	// (application/engine), применяет их адаптер хранения модуля внутри
	// транзакции Append.
	Effects []Effect
}

// Effect — запись проекции в транзакции Append (AD-45: один писатель на
// проекцию, курсор и проекция атомарно).
type Effect interface {
	// EffectKind — вид эффекта: по нему адаптер журнала выбирает применяющего.
	EffectKind() string
}

// Check — проверка конкурентности команды (AD-39).
type Check struct {
	// Stream — поток, проверенный гардом; после BasisSeq в нём не должно быть guard_relevant.
	Stream   string
	BasisSeq int64
	// PolicySubject, PolicySeq — политика субъекта не менялась после PolicySeq.
	PolicySubject string
	PolicySeq     int64
	// ConcessionID, Consume — расход лимита разрешения на отклонение атомарно с решением.
	ConcessionID string
	Consume      int64
}

// AppendResult — позиции записанных записей.
type AppendResult struct {
	Seqs      []int64
	CARefs    []string
	Committed time.Time
}

// ReadQuery — чтение журнала.
type ReadQuery struct {
	Stream    string
	Partition int
	AfterSeq  int64
	Limit     int
	Moment    platform.Moment
	RunID     string
}

// Heads — головы двух цепочек.
type Heads struct {
	MainSeq  int64
	MainLink string
	CASeq    int64
	CALink   string
}

// Envelope — расшифрованный конверт записи: исходные подписанные байты DSSE.
type Envelope struct {
	Raw []byte
}

// Fence — аренда с эпохой (AD-6).
type Fence struct {
	Lease string
	Epoch int64
}

// CursorAdvance — продвижение курсора потребителя в транзакции записи.
type CursorAdvance struct {
	Name      string
	Partition int
	Seq       int64
}

// Ошибки записи (коды journal.* в contracts/errors.yaml).
var (
	ErrFenced              = errors.New("journal.fenced")
	ErrStaleState          = errors.New("journal.stale_state")
	ErrStalePolicy         = errors.New("journal.stale_policy")
	ErrConcessionExhausted = errors.New("journal.concession_exhausted")
	ErrTimeRegression      = errors.New("journal.time_regression")
)

// LeaseStore — ведомый порт аренд партиций и ролей-лидеров (AD-6, AD-35, ключ
// lease_store): Postgres в MVP, k8s Lease — замена. Время — только InfraClock (AD-37).
type LeaseStore interface {
	// Acquire — взять или продлить аренду name до ttl; возвращает эпоху.
	Acquire(ctx context.Context, name, holder string, ttl time.Duration) (Fence, bool, error)
	// Release — отдать аренду.
	Release(ctx context.Context, f Fence) error
}

// Consumer — порт потребителя журнала (AD-45): имя, охват partition | global,
// курсор consumer_offsets в той же транзакции, что и выход. Адаптер —
// infrastructure/storage/journal/feed.
type Consumer interface {
	// Consume отдаёт записи после курсора в handle пачками; handle возвращает
	// записи для Append (выход потребителя) — курсор сдвигается атомарно с ними.
	Consume(ctx context.Context, name string, scope Scope, handle func(ctx context.Context, batch []jc.JournalEntry) (AppendRequest, error)) error
}

// Scope — охват потребителя.
type Scope struct {
	Global    bool
	Partition int
}

// DomainClock — доменные часы (AD-37, ключ domain_clock): системные или
// «сейчас» сценария из журнала (последняя запись time.clock.ticked). Читает
// только application при приёме команды; transport и аренды — никогда.
type DomainClock interface {
	Now(ctx context.Context) (time.Time, error)
}

// InfraClock — инфраструктурные монотонные часы (AD-37, ключ infra_clock):
// аренды, сеансы, частота, повторы, контрольные точки, TLS.
type InfraClock interface {
	Now() time.Time
}
