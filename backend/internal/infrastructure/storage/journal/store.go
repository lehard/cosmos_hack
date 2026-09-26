package journal

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	app "ant/internal/application/journal"
	jc "ant/internal/contracts/journal"
	dj "ant/internal/domain/journal"
	enginestore "ant/internal/infrastructure/storage/engine"
)

// Канал LISTEN/NOTIFY «есть новое» (AD-6): полезная нагрузка — только seq
// головы основной цепочки.
const NotifyChannel = "ant_journal"

// ProgressChannel — канал LISTEN/NOTIFY «курсор потребителя сдвинулся»
// (эпик 16): полезной нагрузки нет. Listener его слушает, но Append в него
// больше не пишет (эпик 35): сдвиг курсора сообщается в процессе
// (Store.OnProgress), а Settler перепроверяет условие по таймеру для
// потребителей других процессов.
const ProgressChannel = "ant_journal_progress"

// Ключи pg_advisory_xact_lock голов цепочек (AD-44): порядок фиксирован —
// сначала основная, затем ca, поэтому две цепочки не блокируют друг друга
// взаимно. Значения — «antjmain» и «antjca» в ASCII.
const (
	lockMain int64 = 0x616e746a6d61696e
	lockCA   int64 = 0x616e746a6361
)

// DefaultBatchMax — K по умолчанию (deploy/config: journal.batch_max).
const DefaultBatchMax = 256

// DefaultSkew — допустимый откат InfraClock между копиями: committed_at
// следующей записи берётся не меньше головы, если часы копии отстают не
// больше чем на DefaultSkew; больше — journal.time_regression (AD-37).
const DefaultSkew = 2 * time.Second

// Store — адаптер postgres порта JournalStore (AD-35, ключ journal_store):
// единственная функция записи Append (AD-44) и чтение по оси и моменту (AD-22).
type Store struct {
	pool     *pgxpool.Pool
	clock    app.InfraClock
	batchMax int
	skew     time.Duration
	effects  []EffectApplier
	// critical — построитель записей журнала критических действий (AD-28);
	// nil — только записи Critical из запроса.
	critical app.CriticalBuilder
	// cipher — шифрование блока при хранении (AD-23); nil — блок открыт.
	cipher Cipher
	deks   dekCache
	// scenario — журнал в режиме часов scenario (AD-37): recorded_at записи
	// без доменного времени — доменное «сейчас» журнала (recorded_at головы).
	scenario bool
	// progress — подписчики «курсор потребителя сдвинулся» в этом процессе
	// (Settler); вызываются после фиксации (эпик 35).
	progressMu sync.Mutex
	progress   []func()
}

// OnProgress подписывает fn на сдвиг курсора потребителя этим процессом:
// fn вызывается после фиксации Append с CursorAdvance (эпик 35: вместо
// pg_notify в каждой фиксации потребителя).
func (s *Store) OnProgress(fn func()) {
	s.progressMu.Lock()
	defer s.progressMu.Unlock()
	s.progress = append(s.progress, fn)
}

func (s *Store) notifyProgress() {
	s.progressMu.Lock()
	fns := slices.Clone(s.progress)
	s.progressMu.Unlock()
	for _, fn := range fns {
		fn()
	}
}

// EffectApplier — применяющий эффекты модуля-писателя проекций внутри
// транзакции Append (AD-44, AD-45): handled = false — эффект не этого
// модуля. Эффект, который не взял ни один применяющий, — ошибка записи:
// курсор без своей проекции не сдвигается.
type EffectApplier func(ctx context.Context, tx pgx.Tx, e app.Effect) (handled bool, err error)

// EngineEffects — применяющий эффекты движка (проекции каркаса, вклады,
// журнал изменений для SSE; storage/engine.ApplyEffect). Подключён всегда.
func EngineEffects(ctx context.Context, tx pgx.Tx, e app.Effect) (bool, error) {
	return enginestore.ApplyEffect(ctx, tx, e)
}

// Option — настройка Store.
type Option func(*Store)

// WithBatchMax — K: предел записей в пачке.
func WithBatchMax(k int) Option { return func(s *Store) { s.batchMax = k } }

// WithSkew — допустимое отставание часов копии от головы цепочки.
func WithSkew(d time.Duration) Option { return func(s *Store) { s.skew = d } }

// WithScenarioClock — журнал в режиме часов scenario (AD-37, эпик 16):
// записи, которым писатель не поставил recorded_at (реакции движка, решения
// модулей без своего флага часов), получают recorded_at головы — последнее
// доменное «сейчас» сценария, а не реальное committed_at. Иначе реальное
// время обгоняет виртуальное, и следующий тик прогона отвергается как
// убывание recorded_at.
func WithScenarioClock(on bool) Option { return func(s *Store) { s.scenario = on } }

// WithEffects — применяющие эффекты модулей со своими проекциями (сверх
// движка), по порядку: эффект применяет первый, кто его взял.
func WithEffects(a ...EffectApplier) Option {
	return func(s *Store) { s.effects = append(s.effects, a...) }
}

// WithCritical — построитель записей журнала критических действий, который
// Append вызывает в своей транзакции для каждой пачки (AD-8, AD-28):
// критическое действие не записывается без записи CA — ни решением модуля, ни
// реакцией движка, ни фактом приёма.
func WithCritical(b app.CriticalBuilder) Option { return func(s *Store) { s.critical = b } }

// NewStore создаёт хранилище журнала над пулом pgx. Часы — InfraClock
// (committed_at, проверка аренд; AD-37).
func NewStore(pool *pgxpool.Pool, clock app.InfraClock, opts ...Option) *Store {
	s := &Store{pool: pool, clock: clock, batchMax: DefaultBatchMax, skew: DefaultSkew, effects: []EffectApplier{EngineEffects}}
	for _, o := range opts {
		o(s)
	}
	return s
}

var _ app.JournalStore = (*Store)(nil)

type txKey struct{}

// Tx — транзакция Append из контекста AppendRequest.Project и применяющих
// эффекты (EffectApplier): адаптеры
// проекций модулей пишут свои таблицы в той же транзакции, что и записи
// журнала и курсор потребителя (AD-45). Вне Project — nil, false.
func Tx(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txKey{}).(pgx.Tx)
	return tx, ok
}

type head struct {
	seq       int64
	link      dj.Digest
	committed time.Time
	recorded  time.Time
}

// Append — единственная функция записи журнала (AD-44, FR-71). Одна
// транзакция: проверка аренды с эпохой (ErrFenced) → pg_advisory_xact_lock
// основной цепочки, затем ca → проверки AD-39 → чтение голов, звенья, INSERT
// → леджер разрешений, курсор потребителя, выход потребителя (Project) →
// NOTIFY. Выход потребителя — эффекты (Effects, применяют адаптеры хранения
// модулей-писателей) и Project. Блокировка головы держится до фиксации, поэтому видимость по seq
// монотонна и вилка цепочки невозможна при любом числе копий.
func (s *Store) Append(ctx context.Context, rq app.AppendRequest) (app.AppendResult, error) {
	var res app.AppendResult
	// Политика субъекта команды (AD-39): проверки, положенные в контекст
	// общим декоратором прав, — в той же транзакции.
	rq = app.WithContextChecks(ctx, rq)
	if len(rq.Batch) > s.batchMax || len(rq.Critical) > s.batchMax {
		return res, fmt.Errorf("%w: %d / %d > %d", app.ErrBatchTooLarge, len(rq.Batch), len(rq.Critical), s.batchMax)
	}
	for _, p := range append(append([]app.Pending(nil), rq.Batch...), rq.Critical...) {
		if err := dj.ValidatePending(p.Entry); err != nil {
			return res, fmt.Errorf("%w: %v", app.ErrInvalidEntry, err)
		}
	}
	err := pgx.BeginTxFunc(ctx, s.pool, pgx.TxOptions{IsoLevel: pgx.ReadCommitted}, func(tx pgx.Tx) error {
		var err error
		res, err = s.append(ctx, tx, rq)
		return err
	})
	if err != nil {
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pe.Code == "23505" {
			return app.AppendResult{}, fmt.Errorf("%w: %s", app.ErrDuplicate, pe.Detail)
		}
		return app.AppendResult{}, err
	}
	if rq.Consumer != nil || len(rq.Cursors) > 0 {
		s.notifyProgress()
	}
	return res, nil
}

func (s *Store) append(ctx context.Context, tx pgx.Tx, rq app.AppendRequest) (app.AppendResult, error) {
	var res app.AppendResult
	now := s.clock.Now().UTC().Truncate(time.Millisecond)

	// 1. Аренда с эпохой (AD-6): строка аренды под FOR SHARE до фиксации —
	// перехват аренды другой копией ждёт конца этой транзакции.
	if rq.Fence != nil {
		if err := checkFence(ctx, tx, *rq.Fence, now); err != nil {
			return res, err
		}
	}
	// Пачка без записей и проверок — только курсор и выход потребителя: голова
	// цепочки не нужна.
	chainless := len(rq.Batch) == 0 && len(rq.Critical) == 0 && len(rq.Checks) == 0 && len(rq.ConcessionGrants) == 0
	if chainless {
		// Эпик 35: фиксация без записей журнала (курсор и проекции —
		// пересобираемый кэш, AD-45) не ждёт сброса WAL на диск. После сбоя
		// БД теряется разве что хвост таких фиксаций — курсор вместе со своим
		// выходом (одна транзакция), и потребитель повторяет их; записи
		// журнала и курсоры в транзакциях с записями фиксируются синхронно.
		if _, err := tx.Exec(ctx, "SET LOCAL synchronous_commit = off"); err != nil {
			return res, err
		}
		return res, s.finish(ctx, tx, rq, &res, now)
	}
	// 2. Головы цепочек: основная, затем ca (порядок фиксирован, AD-44).
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", lockMain); err != nil {
		return res, err
	}
	if len(rq.Critical) > 0 {
		if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", lockCA); err != nil {
			return res, err
		}
	}
	// 3. Проверки конкурентности команды (AD-39) — после блокировки: между
	// проверкой и записью в журнал ничего не вклинится.
	for _, c := range rq.Checks {
		if err := check(ctx, tx, c); err != nil {
			return res, err
		}
	}
	// 4. Головы, время, звенья.
	mainHead, err := readHead(ctx, tx, "main")
	if err != nil {
		return res, err
	}
	committed, err := s.committedAt(now, mainHead.committed)
	if err != nil {
		return res, err
	}
	var caHead head
	if len(rq.Critical) > 0 {
		if caHead, err = readHead(ctx, tx, "ca"); err != nil {
			return res, err
		}
		if committed, err = s.committedAt(committed, caHead.committed); err != nil {
			return res, err
		}
	}
	res.Committed = committed
	var rows, wraps [][]any
	mainRows, seqs, sealed, mainWraps, err := s.sealChain(jc.JournalEntryChainMain, rq.Batch, mainHead, committed)
	if err != nil {
		return res, err
	}
	rows, wraps = append(rows, mainRows...), append(wraps, mainWraps...)
	res.Seqs = seqs
	// Записи CA (AD-8, AD-28): переданные запросом и построенные
	// CriticalBuilder по записям основной пачки — та же транзакция. Голова ca
	// блокируется после основной (порядок фиксирован, AD-44).
	critical := rq.Critical
	caLocked := len(rq.Critical) > 0
	caNext := caHead.seq + int64(len(rq.Critical))
	if s.critical != nil && len(sealed) > 0 {
		next := func() (int64, error) {
			if !caLocked {
				if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", lockCA); err != nil {
					return 0, err
				}
				if caHead, err = readHead(ctx, tx, "ca"); err != nil {
					return 0, err
				}
				caLocked, caNext = true, caHead.seq
			}
			caNext++
			return caNext, nil
		}
		built, err := s.critical.Critical(ctx, sealed, next)
		if err != nil {
			return res, fmt.Errorf("журнал критических действий: %w", err)
		}
		for _, p := range built {
			if err := dj.ValidatePending(p.Entry); err != nil {
				return res, fmt.Errorf("%w: запись CA: %v", app.ErrInvalidEntry, err)
			}
		}
		critical = append(slices.Clone(critical), built...)
	}
	if len(critical) > 0 {
		if caHead.committed.After(committed) {
			return res, app.Reject(app.ErrTimeRegression, 0, "committed_at", dj.FormatTime(committed), "head", dj.FormatTime(caHead.committed))
		}
		caRows, caSeqs, _, caWraps, err := s.sealChain(jc.JournalEntryChainCa, critical, caHead, committed)
		if err != nil {
			return res, err
		}
		rows, wraps = append(rows, caRows...), append(wraps, caWraps...)
		for _, n := range caSeqs {
			res.CARefs = append(res.CARefs, "CA-"+strconv.FormatInt(n, 10))
		}
	}
	if len(rows) > 0 {
		if _, err := tx.CopyFrom(ctx, pgx.Identifier{"journal", "entries"}, entryColumns, pgx.CopyFromRows(rows)); err != nil {
			return res, err
		}
	}
	if len(wraps) > 0 {
		if _, err := tx.CopyFrom(ctx, pgx.Identifier{"journal", "dek_wraps"}, []string{"dek_id", "kek_id", "wrapped", "created_at"}, pgx.CopyFromRows(wraps)); err != nil {
			return res, err
		}
	}
	// 5. Леджер разрешений на отклонение: открытие и расход атомарно с решением.
	anchor := mainHead.seq + 1
	if len(res.Seqs) > 0 {
		anchor = res.Seqs[0]
	}
	for _, g := range rq.ConcessionGrants {
		if g.ConcessionID == "" || g.Limit <= 0 {
			return res, fmt.Errorf("%w: лимит разрешения %q = %d", app.ErrInvalidEntry, g.ConcessionID, g.Limit)
		}
		if _, err := tx.Exec(ctx, "INSERT INTO journal.concession_ledger (concession_id, seq, delta) VALUES ($1, $2, $3)", g.ConcessionID, anchor, g.Limit); err != nil {
			return res, err
		}
	}
	for _, c := range rq.Checks {
		if c.ConcessionID != "" && c.Consume > 0 {
			if _, err := tx.Exec(ctx, "INSERT INTO journal.concession_ledger (concession_id, seq, delta) VALUES ($1, $2, $3)", c.ConcessionID, anchor, -c.Consume); err != nil {
				return res, err
			}
		}
	}
	if err := s.finish(ctx, tx, rq, &res, now); err != nil {
		return res, err
	}
	// 8. Сигнал «есть новое» — доставляется при фиксации (AD-6).
	if len(res.Seqs) > 0 {
		if _, err := tx.Exec(ctx, "SELECT pg_notify($1, $2)", NotifyChannel, strconv.FormatInt(res.Seqs[len(res.Seqs)-1], 10)); err != nil {
			return res, err
		}
	}
	return res, nil
}

// finish — курсор потребителя и выход потребителя в той же транзакции (AD-45).
func (s *Store) finish(ctx context.Context, tx pgx.Tx, rq app.AppendRequest, res *app.AppendResult, now time.Time) error {
	// 6. Курсор потребителя не убывает; единственность писателя — Fence.
	// Сигнал «курсор сдвинулся» (Settler, эпик 16) — после фиксации в
	// процессе (Append → progress), а не pg_notify: NOTIFY берёт общую на
	// кластер блокировку очереди уведомлений до сброса WAL на диск, и фиксации
	// всех потребителей выстраивались в одну очередь (эпик 35, главное узкое
	// место MS-1).
	cursors := rq.Cursors
	if rq.Consumer != nil {
		cursors = append([]app.CursorAdvance{*rq.Consumer}, cursors...)
	}
	if len(cursors) > 0 {
		names, parts, seqs := make([]string, len(cursors)), make([]int32, len(cursors)), make([]int64, len(cursors))
		for i, c := range cursors {
			if c.Name == "" || c.Partition < app.GlobalPartition || c.Seq < 0 {
				return fmt.Errorf("%w: курсор %q/%d/%d", app.ErrInvalidEntry, c.Name, c.Partition, c.Seq)
			}
			names[i], parts[i], seqs[i] = c.Name, int32(c.Partition), c.Seq
		}
		// Все курсоры — одним запросом (группа проектора — десятки курсоров).
		if _, err := tx.Exec(ctx, `INSERT INTO journal_state.consumer_offsets (name, partition, seq, updated_at)
SELECT n, p, s, $4 FROM unnest($1::text[], $2::int[], $3::bigint[]) AS c(n, p, s)
ON CONFLICT (name, partition) DO UPDATE SET seq = GREATEST(consumer_offsets.seq, EXCLUDED.seq), updated_at = EXCLUDED.updated_at`,
			names, parts, seqs, now); err != nil {
			return err
		}
	}
	// 7. Выход потребителя в таблицы модулей — в той же транзакции: эффекты
	// (проекции, вклады, журнал изменений), затем Project.
	tctx := context.WithValue(ctx, txKey{}, tx)
	for _, e := range rq.Effects {
		if err := s.applyEffect(tctx, tx, e); err != nil {
			return err
		}
	}
	if rq.Project != nil {
		return rq.Project(tctx, *res)
	}
	return nil
}

func (s *Store) applyEffect(ctx context.Context, tx pgx.Tx, e app.Effect) error {
	if e == nil {
		return fmt.Errorf("%w: пустой эффект", app.ErrInvalidEntry)
	}
	for _, apply := range s.effects {
		handled, err := apply(ctx, tx, e)
		if err != nil {
			return err
		}
		if handled {
			return nil
		}
	}
	return fmt.Errorf("%w: эффект %s никто не применяет (Store: WithEffects)", app.ErrInvalidEntry, e.EffectKind())
}

// committedAt — committed_at не убывает по seq (AD-37): отставание часов
// копии в пределах skew поглощается, больше — отказ.
func (s *Store) committedAt(now, prev time.Time) (time.Time, error) {
	if !now.Before(prev) {
		return now, nil
	}
	if prev.Sub(now) <= s.skew {
		return prev, nil
	}
	return now, app.Reject(app.ErrTimeRegression, 0, "committed_at", dj.FormatTime(now), "head", dj.FormatTime(prev))
}

var entryColumns = []string{
	"chain", "seq", "event_id", "event_type", "entry_kind", "stream", "partition", "item_id", "run_id", "source_id",
	"occurred_at", "recorded_at", "committed_at", "guard_relevant", "is_trigger", "commit", "link", "header", "salt", "envelope",
	"dek_id", "aead", "nonce",
}

// sealChain ставит пачке одной цепочки seq, committed_at, recorded_at, commit
// и link по формуле AD-44 и готовит строки вставки. Соль — 128 случайных бит
// на запись (AD-23); с Cipher блок {соль, конверт} шифруется DEK записи, а
// обёртка DEK идёт в journal.dek_wraps (wraps).
func (s *Store) sealChain(chain jc.JournalEntryChain, batch []app.Pending, h head, committed time.Time) (rows [][]any, seqs []int64, sealed []app.Sealed, wraps [][]any, err error) {
	rows = make([][]any, 0, len(batch))
	seqs = make([]int64, 0, len(batch))
	prev, seq, recorded := h.link, h.seq, h.recorded
	committedS := dj.FormatTime(committed)
	for _, p := range batch {
		e := p.Entry
		seq++
		e.Seq = int(seq)
		e.Chain = chain
		e.CommittedAt = committedS
		if e.RecordedAt == "" {
			// Часы system: recorded_at = committed_at (AD-37). Часы scenario:
			// доменное «сейчас» — recorded_at головы (последний тик прогона).
			e.RecordedAt = committedS
			if s.scenario && !recorded.IsZero() {
				e.RecordedAt = dj.FormatTime(recorded)
			}
		}
		rec, err := dj.ParseTime(e.RecordedAt)
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("%w: recorded_at %q", app.ErrInvalidEntry, e.RecordedAt)
		}
		if rec.Before(recorded) {
			return nil, nil, nil, nil, app.Reject(app.ErrTimeRegression, 0, "recorded_at", e.RecordedAt, "head", dj.FormatTime(recorded))
		}
		recorded = rec
		occurred, _ := dj.ParseTime(e.OccurredAt)
		// AD-23: соль — 128 случайных бит на запись; хранится только внутри
		// блока, поэтому открытый commit не выдаёт содержимое перебором.
		salt := make([]byte, dj.SaltSize)
		if _, err := rand.Read(salt); err != nil {
			return nil, nil, nil, nil, err
		}
		envelope, err := dj.Canonical(p.Envelope)
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("%w: конверт %s: %v", app.ErrInvalidEntry, e.EventID, err)
		}
		link, header, err := dj.Seal(&e, prev, salt, envelope)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		commit, _ := dj.ParseDigest(e.Commit)
		cls := dj.Classify(e)
		eventUUID, err := parseUUID(e.EventID)
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("%w: event_id %q", app.ErrInvalidEntry, e.EventID)
		}
		storedSalt, stored := salt, envelope
		var dekID, aead *string
		var nonce []byte
		if s.cipher != nil {
			id, n, ct, wrapped, err := s.sealBlock(e, salt, envelope)
			if err != nil {
				return nil, nil, nil, nil, fmt.Errorf("шифрование записи %s: %w", e.EventID, err)
			}
			alg := s.cipher.AEAD()
			dekID, aead, nonce, storedSalt, stored = &id, &alg, n, []byte{}, ct
			wraps = append(wraps, []any{id, s.cipher.KEKID(), wrapped, committed})
		}
		rows = append(rows, []any{
			string(chain), seq, eventUUID, e.EventType, string(e.EntryKind), e.Stream, e.Partition, e.ItemID, e.RunID, e.SourceID,
			occurred, rec, committed, cls.GuardRelevant, cls.Trigger, commit.Bytes(), link.Bytes(), string(header), storedSalt, stored,
			dekID, aead, nonce,
		})
		seqs = append(seqs, seq)
		sealed = append(sealed, app.Sealed{Entry: e, Envelope: envelope})
		prev = link
	}
	return rows, seqs, sealed, wraps, nil
}

func readHead(ctx context.Context, tx pgx.Tx, chain string) (head, error) {
	var h head
	var link []byte
	err := tx.QueryRow(ctx, `SELECT seq, link, committed_at, recorded_at FROM journal.entries
WHERE chain = $1 ORDER BY seq DESC LIMIT 1`, chain).Scan(&h.seq, &link, &h.committed, &h.recorded)
	if errors.Is(err, pgx.ErrNoRows) {
		return head{link: dj.ZeroLink}, nil
	}
	if err != nil {
		return h, err
	}
	h.link, err = dj.DigestFromBytes(link)
	return h, err
}

func checkFence(ctx context.Context, tx pgx.Tx, f app.Fence, now time.Time) error {
	var epoch int64
	var expires time.Time
	err := tx.QueryRow(ctx, "SELECT epoch, expires_at FROM journal_state.leases WHERE name = $1 FOR SHARE", f.Lease).Scan(&epoch, &expires)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return app.Reject(app.ErrFenced, 0, "partition", f.Lease)
	case err != nil:
		return err
	case epoch != f.Epoch || !now.Before(expires):
		return app.Reject(app.ErrFenced, 0, "partition", f.Lease, "epoch", strconv.FormatInt(f.Epoch, 10), "current_epoch", strconv.FormatInt(epoch, 10))
	}
	return nil
}

func check(ctx context.Context, tx pgx.Tx, c app.Check) error {
	var stale bool
	if c.Stream != "" {
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM journal.entries
WHERE chain = 'main' AND stream = $1 AND seq > $2 AND guard_relevant)`, c.Stream, c.BasisSeq).Scan(&stale); err != nil {
			return err
		}
		if stale {
			return app.Reject(app.ErrStaleState, c.BasisSeq, "stream", c.Stream, "basis_seq", strconv.FormatInt(c.BasisSeq, 10))
		}
	}
	if c.ItemProcessed && strings.HasPrefix(c.Stream, "item:") {
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM journal.entries e
WHERE e.chain = 'main' AND e.stream = $1 AND e.is_trigger
  AND e.seq > COALESCE((SELECT o.seq FROM journal_state.consumer_offsets o WHERE o.name = $2 AND o.partition = e.partition), 0))`,
			c.Stream, app.WorkerConsumer).Scan(&stale); err != nil {
			return err
		}
		if stale {
			return app.Reject(app.ErrStaleState, c.BasisSeq, "stream", c.Stream, "basis_seq", strconv.FormatInt(c.BasisSeq, 10), "reason", "unprocessed_input")
		}
	}
	if c.PolicyStream != "" {
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM journal.entries
WHERE chain = 'main' AND stream = $1 AND seq > $2)`, c.PolicyStream, c.PolicySeq).Scan(&stale); err != nil {
			return err
		}
		if stale {
			return app.Reject(app.ErrStalePolicy, c.PolicySeq, "policy_seq", strconv.FormatInt(c.PolicySeq, 10))
		}
	}
	if c.ConcessionID != "" && c.Consume > 0 {
		var remaining, limit int64
		if err := tx.QueryRow(ctx, `SELECT COALESCE(SUM(delta), 0), COALESCE(SUM(delta) FILTER (WHERE delta > 0), 0)
FROM journal.concession_ledger WHERE concession_id = $1`, c.ConcessionID).Scan(&remaining, &limit); err != nil {
			return err
		}
		if remaining < c.Consume {
			return app.Reject(app.ErrConcessionExhausted, 0, "concession_id", c.ConcessionID,
				"remaining", strconv.FormatInt(remaining, 10), "limit", strconv.FormatInt(limit, 10))
		}
	}
	return nil
}

func parseUUID(s string) ([16]byte, error) {
	var u [16]byte
	b, err := hex.DecodeString(strings.ReplaceAll(s, "-", ""))
	if err != nil || len(b) != 16 {
		return u, fmt.Errorf("uuid %q", s)
	}
	copy(u[:], b)
	return u, nil
}
