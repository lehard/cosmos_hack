package security

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	engineapp "ant/internal/application/engine"
	"ant/internal/application/engine/enginemem"
	appjournal "ant/internal/application/journal"
	"ant/internal/application/nonconformity/nctest"
	"ant/internal/application/security/verify"
	"ant/internal/contracts/catalog"
	dj "ant/internal/domain/journal"
	"ant/internal/domain/kernel"
	journalstore "ant/internal/infrastructure/storage/journal"
	"ant/internal/infrastructure/storage/journal/journaltest"
)

// Верификатор во время потока прогона (AD-9, AD-45): записи изделия идут
// одна за другой, воркер сворачивает их и пишет проекции вместе с курсором,
// а верификатор сверяет проекции с журналом тут же. Ложного «проекция
// расходится с журналом» быть не должно; настоящая правка проекции в обход
// системы — ловится с местом.

const raceParts = 4

// streamWorld — журнал, воркер с проекцией engine.item_state и чтение
// проекций, как у cmd/verifier.
type streamWorld struct {
	world
	t      *testing.T
	ctx    context.Context
	pool   *pgxpool.Pool
	codec  *engineapp.Codec
	worker *engineapp.WorkerService
	leases *journalstore.Leases
	at     time.Time
	mu     sync.Mutex
}

func newStreamWorld(t *testing.T) *streamWorld {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	t.Cleanup(cancel)
	w := newWorld(t)
	p := w.db.AppPool(t)
	s := &streamWorld{world: w, t: t, ctx: ctx, pool: p, leases: journalstore.NewLeases(p, journaltest.SysClock{}),
		at: time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Millisecond)}
	s.codec = &engineapp.Codec{Store: w.store, Sealer: enginemem.Sealer{KeyRef: "engine@1"}, KeyRef: "engine@1",
		DomainBuild: dj.ZeroLink.String(), Partitions: raceParts}
	s.worker = engineapp.NewWorker(engineapp.WorkerConfig{Codec: s.codec, Projections: engineapp.NewRegistry()})
	return s
}

// fact — запись входа изделия (результат контроля).
func (s *streamWorld) fact(item string) {
	s.t.Helper()
	s.mu.Lock()
	s.at = s.at.Add(time.Second)
	at := s.at
	s.mu.Unlock()
	p := nctest.RecordP(catalog.InspectionResultRecorded, item, at, map[string]any{"method": "camera", "phase": "after_operation",
		"outcome": "no_defect", "processing_state": "completed", "step_key": "welding.weld"}, raceParts)
	if _, err := s.store.Append(s.ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{p}}); err != nil {
		s.t.Error(err)
	}
}

// process — воркер сворачивает изделие до последней записи (под арендой партиции).
func (s *streamWorld) process(item string) {
	s.t.Helper()
	es, err := s.store.Read(s.ctx, appjournal.ReadQuery{ItemID: item})
	if err != nil || len(es) == 0 {
		s.t.Errorf("вход изделия: %d %v", len(es), err)
		return
	}
	last := es[len(es)-1]
	part := kernel.PartitionOf(item, raceParts)
	f, ok, err := s.leases.Acquire(s.ctx, appjournal.PartitionLease(part), "verifier-race-test", 30*time.Second)
	if err != nil || !ok {
		s.t.Errorf("аренда партиции: %v %v", ok, err)
		return
	}
	if err := s.worker.Process(s.ctx, engineapp.Partition{Number: part, Epoch: f.Epoch},
		[]engineapp.Work{{ItemID: item, UpToSeq: int64(last.Seq), Trigger: last}}); err != nil {
		s.t.Error(err)
	}
}

// dbProjections — проекции и курсор воркера из Postgres (как у cmd/verifier);
// beforeRows — вызывается перед чтением проекций (вмешательство потока).
type dbProjections struct {
	pool       *pgxpool.Pool
	beforeRows func(item string)
}

func (p dbProjections) ItemRows(ctx context.Context, itemID string) (map[string][]byte, error) {
	if p.beforeRows != nil {
		p.beforeRows(itemID)
	}
	rows, err := p.pool.Query(ctx, "SELECT name, value::text FROM engine.projections WHERE item_id = $1 AND key = $1", itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]byte{}
	for rows.Next() {
		var n, v string
		if err := rows.Scan(&n, &v); err != nil {
			return nil, err
		}
		out[n] = []byte(v)
	}
	return out, rows.Err()
}

func (p dbProjections) WorkerCursor(ctx context.Context, partition int) (int64, error) {
	var seq int64
	err := p.pool.QueryRow(ctx, "SELECT COALESCE(MAX(seq), 0) FROM journal_state.consumer_offsets WHERE name = $1 AND partition = $2",
		appjournal.WorkerConsumer, partition).Scan(&seq)
	return seq, err
}

func (s *streamWorld) verify(pr dbProjections) verify.Report {
	s.t.Helper()
	in := verify.Input{Journal: s.store, Codec: &engineapp.Codec{Store: lenient{s.store}, Partitions: raceParts},
		Registry: engineapp.NewRegistry(), Bundles: engineapp.EmptyBundles{}, Projections: pr,
		Partitions: raceParts, MaxGap: time.Hour, SettleWait: 20 * time.Millisecond}
	r, err := verify.Run(s.ctx, in)
	if err != nil {
		s.t.Fatal(err)
	}
	return r
}

func findings(r verify.Report, name string) []string {
	var out []string
	for _, c := range r.Checks {
		if string(c.Check) == name {
			for _, f := range c.Findings {
				out = append(out, f.Detail)
			}
		}
	}
	return out
}

func noViolation(t *testing.T, r verify.Report, when string) {
	t.Helper()
	for _, n := range []string{"projections", "coverage", "reactions"} {
		if got := rejected(r, n); len(got) != 0 {
			t.Fatalf("%s — ложное нарушение %s: %v\n%s", when, n, got, verify.Summary(r))
		}
	}
}

// Воркер сворачивает новую запись изделия ровно между чтением входа и
// проекций верификатором (как на стенде: basis_seq в журнале 2774, в
// проекции 2781) — верификатор не выносит «нарушено» по несошедшемуся
// снимку, а перечитывает его и даёт сверку «цело».
func TestVerifierDuringStreamNoFalseMismatch(t *testing.T) {
	s := newStreamWorld(t)
	const item = "I-07D943AE"
	for i := 0; i < 3; i++ {
		s.fact(item)
		s.process(item)
	}
	r := s.verify(dbProjections{pool: s.pool})
	noViolation(t, r, "после потока")
	if c := findings(r, "projections"); len(c) != 0 {
		t.Fatalf("проекции: %v", c)
	}
	// Поток идёт: перед чтением проекций приходит и сворачивается новая запись.
	fired := 0
	race := dbProjections{pool: s.pool, beforeRows: func(it string) {
		if it == item && fired < 2 {
			fired++
			s.fact(item)
			s.process(item)
		}
	}}
	r = s.verify(race)
	if fired == 0 {
		t.Fatal("вмешательство потока не случилось")
	}
	noViolation(t, r, "во время потока")
	if r.Verdict == "violated" {
		t.Fatalf("во время потока — «нарушено»:\n%s", verify.Summary(r))
	}
	// Запись пришла, воркер её ещё не свернул (отстаёт) — «не проверяемо»
	// (вход ещё не обработан), не нарушение.
	lag := dbProjections{pool: s.pool, beforeRows: func(it string) {
		if it == item && fired < 3 {
			fired++
			s.fact(item)
		}
	}}
	r = s.verify(lag)
	noViolation(t, r, "воркер отстаёт")
	if !strings.Contains(strings.Join(findings(r, "coverage"), "\n"), "ещё не обработан") {
		t.Fatalf("отставание воркера не отмечено:\n%s", verify.Summary(r))
	}
	s.process(item)
	if r := s.verify(dbProjections{pool: s.pool}); len(findings(r, "coverage")) != 0 || len(rejected(r, "projections")) != 0 {
		t.Fatalf("после догона:\n%s", verify.Summary(r))
	}

	// Настоящая подделка проекции в обход системы — «нарушено» с местом.
	tp := &Tamperer{Conn: s.db.AdminConn(t), KEK: s.kek, Profile: "demo"}
	if _, err := tp.ProjectionUpdate(s.ctx, item, map[string]any{"basis_seq": 1}); err != nil {
		t.Fatal(err)
	}
	r = s.verify(dbProjections{pool: s.pool})
	got := strings.Join(rejected(r, "projections"), "\n")
	if r.Verdict != "violated" || !strings.Contains(got, "проекция расходится с журналом") || !strings.Contains(got, item) ||
		!strings.Contains(got, "engine.item_state") || !strings.Contains(got, "basis_seq") {
		t.Fatalf("подделка проекции не найдена:\n%s", verify.Summary(r))
	}
}

// Живой поток нескольких изделий параллельно с проверками: ни одна проверка
// во время потока не даёт «нарушено»; после потока — сверка проекций «цело».
func TestVerifierConcurrentStream(t *testing.T) {
	s := newStreamWorld(t)
	items := []string{"I-B744627A", "I-07D943AE", "I-00000001"}
	for _, it := range items {
		s.fact(it)
		s.process(it)
	}
	done, stop := make(chan struct{}), make(chan struct{})
	var once sync.Once
	halt := func() { once.Do(func() { close(stop) }); <-done }
	t.Cleanup(halt) // поток останавливается до закрытия пулов теста
	go func() {
		defer close(done)
		for i := 0; i < 60 && s.ctx.Err() == nil; i++ {
			select {
			case <-stop:
				return
			default:
			}
			it := items[i%len(items)]
			s.fact(it)
			s.process(it)
		}
	}()
	checks := 0
	for running := true; running; checks++ {
		select {
		case <-done:
			running = false
		default:
		}
		noViolation(t, s.verify(dbProjections{pool: s.pool}), "во время потока")
	}
	r := s.verify(dbProjections{pool: s.pool})
	noViolation(t, r, "после потока")
	if len(findings(r, "coverage")) != 0 {
		t.Fatalf("после потока: %v", findings(r, "coverage"))
	}
	t.Logf("проверок во время потока: %d", checks)
}
