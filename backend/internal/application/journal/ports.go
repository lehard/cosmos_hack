package journal

import (
	"context"
	"errors"
	"fmt"
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

// Pending — запись, подготовленная к Append: открытые поля (без seq, chain,
// committed_at, commit, link — их ставит Append) и конверт DSSE. recorded_at
// пуст — Append ставит его равным committed_at (часы system, AD-37); задан
// (режим scenario — доменное «сейчас» сценария) — не может убывать по seq.
type Pending struct {
	Entry    jc.JournalEntry
	Envelope []byte
}

// AppendRequest — пачка записи.
type AppendRequest struct {
	// Batch — записи основной цепочки (не больше K, journal.batch_max).
	Batch []Pending
	// Critical — записи журнала критических действий (цепочка ca) в той же
	// транзакции, что и основная (AD-8, AD-28).
	Critical []Pending
	// Fence — аренда партиции с эпохой; запись от копии, потерявшей аренду, — ErrFenced (AD-6).
	Fence *Fence
	// Checks — проверки AD-39: потоки и их basis_seq, policy_seq субъекта, расход разрешения.
	Checks []Check
	// ConcessionGrants — лимиты разрешений на отклонение, открываемые этой
	// пачкой (запись разрешения): от них считается остаток для Check.Consume.
	ConcessionGrants []ConcessionGrant
	// Consumer — курсор потребителя, обновляемый в той же транзакции (AD-45).
	Consumer *CursorAdvance
	// Cursors — ещё курсоры в той же транзакции: группа потребителей одной
	// копии (GroupConsumer, эпик 35 — проектор сдвигает курсоры всех
	// глобальных проекций одной фиксацией).
	Cursors []CursorAdvance
	// Effects — выход потребителя в таблицы модулей-писателей проекций в той
	// же транзакции (AD-44, AD-45): проекции изделия, вклады показателей,
	// журнал изменений для SSE. Виды эффектов объявляет модуль-писатель
	// (application/‹модуль›), применяет адаптер хранения этого модуля, которого
	// адаптер журнала вызывает внутри транзакции Append; эффект, который
	// никто не применил, — ошибка записи. Пачка может быть пустой: проектор
	// без выходных записей сдвигает курсор вместе с проекцией.
	Effects []Effect
	// Project — выход потребителя, которому нужна сама транзакция (адаптеры
	// хранения, а не application): вызывается после Effects в той же
	// транзакции (AD-45). Транзакцию адаптер хранения кладёт в ctx; адаптеры
	// проекций берут её оттуда (infrastructure/storage/journal.Tx). res — уже
	// назначенные seq. Сценарии application пишут проекции через Effects.
	Project func(ctx context.Context, res AppendResult) error
}

// CriticalBuilder — построитель записей журнала критических действий (AD-8,
// AD-28): Append вызывает его в своей транзакции после того, как записи
// основной пачки получили seq и commit, и дописывает возвращённые записи в
// цепочку ca той же транзакцией. Реализация — application/security
// (domain/security.BuildCA); модули записей CA сами не создают.
type CriticalBuilder interface {
	// Critical — записи CA для записей основной пачки batch. next — номер
	// следующей записи цепочки ca (CA-‹n›): первый вызов берёт блокировку
	// головы ca (порядок main → ca сохраняется) и читает её; каждый вызов
	// резервирует следующий номер. Записей не нужно — nil.
	Critical(ctx context.Context, batch []Sealed, next func() (int64, error)) ([]Pending, error)
}

// Sealed — запись основной пачки после вычисления звена: открытые поля с
// seq, commit и link и канонический конверт (для CriticalBuilder).
type Sealed struct {
	Entry    jc.JournalEntry
	Envelope []byte
}

// Effect — запись проекции в транзакции Append (AD-45: один писатель на
// проекцию, курсор и проекция атомарно).
type Effect interface {
	// EffectKind — вид эффекта: по нему адаптер журнала выбирает применяющего.
	EffectKind() string
}

// Check — проверка конкурентности команды (AD-39). Пустые поля не проверяются.
type Check struct {
	// Stream — поток, проверенный гардом; после BasisSeq в нём не должно быть
	// записей guard_relevant (иначе ErrStaleState).
	Stream   string
	BasisSeq int64
	// ItemProcessed — у изделия потока Stream (`item:‹id›`) нет необработанного
	// входа: после курсора воркера (WorkerConsumer) его партиции нет
	// записей-триггеров свёртки (иначе ErrStaleState).
	ItemProcessed bool
	// PolicyStream, PolicySeq — политика субъекта (поток `policy:‹область›`) не
	// менялась после PolicySeq (иначе ErrStalePolicy).
	PolicyStream string
	PolicySeq    int64
	// ConcessionID, Consume — расход лимита разрешения на отклонение атомарно с
	// решением (иначе ErrConcessionExhausted).
	ConcessionID string
	Consume      int64
}

// ConcessionGrant — открытие лимита разрешения на отклонение.
type ConcessionGrant struct {
	ConcessionID string
	Limit        int64
}

// AppendResult — позиции записанных записей.
type AppendResult struct {
	// Seqs — seq записей Batch в основной цепочке по порядку.
	Seqs []int64
	// CARefs — номера записей Critical: `CA-‹n›`.
	CARefs []string
	// Committed — committed_at пачки (InfraClock).
	Committed time.Time
}

// ReadQuery — чтение журнала (AD-22). Пустые поля не фильтруют.
type ReadQuery struct {
	// Chain — main (по умолчанию) или ca.
	Chain string
	// Stream — поток `item:‹id›`, `‹вид›:‹id›`, `global`.
	Stream string
	// ItemID — все записи изделия (item_id), в каком бы потоке они ни были:
	// вход свёртки изделия и записанные реакции (AD-5). Вместе с Partition —
	// «вся партиция изделия»; без Stream и ItemID — вся партиция или журнал.
	ItemID string
	// Partition — партиция; nil — все (вся партиция — Partition без Stream и
	// ItemID; записи вне изделия лежат в партиции 0).
	Partition *int
	// EventType — тип записи.
	EventType string
	// AfterSeq — только записи с seq > AfterSeq.
	AfterSeq int64
	// Limit — предел числа записей (0 — 1000).
	Limit int
	// Backward — по убыванию seq (последние записи: «сейчас» сценария, головы).
	// AfterSeq при этом не учитывается.
	Backward bool
	// Moment — ось и момент: recorded — «что мы знали» (префикс журнала по seq
	// с recorded_at ≤ T), occurred — «как было» (occurred_at ≤ T по всему
	// известному). AsOf пуст — всё записанное.
	Moment platform.Moment
	// RunID — прогон сценария (AD-38).
	RunID string
}

// Heads — головы двух цепочек.
type Heads struct {
	MainSeq  int64
	MainLink string
	CASeq    int64
	CALink   string
}

// Envelope — расшифрованный конверт записи: исходные подписанные байты DSSE
// (JCS) и соль; commit сверен.
type Envelope struct {
	Raw  []byte
	Salt []byte
}

// Fence — аренда с эпохой (AD-6).
type Fence struct {
	Lease string
	Epoch int64
}

// CursorAdvance — продвижение курсора потребителя в транзакции записи.
// Курсор не убывает (GREATEST): повтор той же пачки после сбоя безопасен,
// единственность писателя даёт Fence.
type CursorAdvance struct {
	Name      string
	Partition int
	Seq       int64
}

// GlobalPartition — «партиция» курсора глобального потребителя.
const GlobalPartition = -1

// WorkerConsumer — имя курсора воркера по партициям (AD-5, AD-45): по нему
// WorkFeed отдаёт изделия с необработанным входом, а Check.ItemProcessed
// проверяет, что вход изделия обработан.
const WorkerConsumer = "engine.worker"

// PartitionLease — имя аренды партиции воркера (AD-6).
func PartitionLease(p int) string { return fmt.Sprintf("partition:%d", p) }

// Ошибки записи (коды journal.* в contracts/errors.yaml).
var (
	ErrFenced              = errors.New("journal.fenced")
	ErrStaleState          = errors.New("journal.stale_state")
	ErrStalePolicy         = errors.New("journal.stale_policy")
	ErrConcessionExhausted = errors.New("journal.concession_exhausted")
	ErrTimeRegression      = errors.New("journal.time_regression")
	// ErrDuplicate — event_id уже записан в этой цепочке (защита от повторной
	// записи; дедупликацию с ответом источнику делает приём, AD-7).
	ErrDuplicate = errors.New("journal.duplicate")
	// ErrBatchTooLarge — в пачке больше K записей (journal.batch_max, AD-44).
	ErrBatchTooLarge = errors.New("journal.batch_too_large")
	// ErrInvalidEntry — заголовок записи не проходит проверку перед записью.
	ErrInvalidEntry = errors.New("journal.invalid_entry")
	// ErrSealed — блок записи зашифрован, KEK нет (эпик 29).
	ErrSealed = errors.New("journal.sealed")
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
	// записи для Append (выход потребителя) и Project для своих таблиц — курсор
	// сдвигается атомарно с ними. Копия-лидер — по аренде `consumer:‹имя›`
	// (глобальный) или `consumer:‹имя›:‹партиция›`; Fence ставит адаптер.
	// Возвращается при отмене ctx или ошибке handle.
	Consume(ctx context.Context, name string, scope Scope, handle func(ctx context.Context, batch []jc.JournalEntry) (AppendRequest, error)) error
}

// GroupConsumer — необязательное расширение Consumer (эпик 35): несколько
// глобальных потребителей одной копии читают журнал одним проходом, их
// выходы и курсоры фиксируются одной транзакцией Append. У каждого имени —
// свой курсор и своя аренда `consumer:‹имя›`, как у Consume: смысл и
// единственность писателя те же, меньше чтений и фиксаций. handle получает
// пачку после наименьшего курсора группы и курсоры имён (from): записи с
// seq ≤ from[имя] этому имени уже отданы.
type GroupConsumer interface {
	ConsumeGroup(ctx context.Context, names []string, handle func(ctx context.Context, batch []jc.JournalEntry, from map[string]int64) (AppendRequest, error)) error
}

// Signal — сигнал «есть новое» (AD-6): LISTEN/NOTIFY несёт только seq головы
// основной цепочки, данных в нём нет; после переподключения копия догоняет по seq.
type Signal interface {
	// Head — последний известный seq основной цепочки.
	Head() int64
	// Wait блокирует, пока seq головы не станет больше afterSeq, или до отмены ctx.
	Wait(ctx context.Context, afterSeq int64) (int64, error)
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

type runKey struct{}

// WithRun кладёт прогон сценария в контекст: доменное «сейчас» в режиме
// scenario — последняя запись time.clock.ticked этого прогона (AD-37, AD-38).
func WithRun(ctx context.Context, runID string) context.Context {
	return context.WithValue(ctx, runKey{}, runID)
}

// RunFrom — прогон из контекста или пусто.
func RunFrom(ctx context.Context) string {
	s, _ := ctx.Value(runKey{}).(string)
	return s
}

// InfraClock — инфраструктурные монотонные часы (AD-37, ключ infra_clock):
// аренды, сеансы, частота, повторы, контрольные точки, TLS.
type InfraClock interface {
	Now() time.Time
}
