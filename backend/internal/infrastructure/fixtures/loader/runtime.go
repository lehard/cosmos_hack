package loader

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"ant/internal/application/platform"
	"ant/internal/contracts/constants"
	"ant/internal/contracts/errcodes"
	"ant/internal/domain/kernel"
)

// EnvDir — переменная окружения с каталогом заготовок (scenarios/fixtures); не
// задана — встроенный мир (Builtin). Удобно при разработке заготовок: правка
// файла видна после перезапуска api без пересборки образа.
const EnvDir = "ANT_FIXTURES_DIR"

// Builtin — построитель встроенного мира заготовок: генератор
// infrastructure/fixtures/world регистрирует его при инициализации и строит в
// памяти те же сценарии, что лежат в scenarios/fixtures (равенство проверяет
// его тест). Образ ant не несёт каталога scenarios/, а заготовки должны
// отвечать сразу.
var Builtin func() (*Library, error)

// Runtime — мир заготовок процесса: библиотека сценариев и курсор (AD-36).
// Один на процесс api: модульные адаптеры берут Default().
type Runtime struct {
	lib    *Library
	mu     sync.RWMutex
	cursor platform.FixtureCursor
	// runs — сведения о прогонах, которых нет в курсоре (режим, начало, конец);
	// живут в памяти копии api — для пульта этого достаточно (см. отчёт эпика 09).
	runs     map[string]*RunInfo
	lastTick time.Time
	// sess — сессионное наложение: факты команд людей поверх мира (session.go).
	sess session
}

// RunInfo — сведения о прогоне пульта сверх положения курсора.
type RunInfo struct {
	RunID      string
	Scenario   string
	Mode       string // interactive | autocheck
	Seed       int64
	StartedAt  time.Time
	FinishedAt *time.Time
	Stopped    bool
}

// New — мир заготовок над библиотекой и курсором.
func New(lib *Library, cursor platform.FixtureCursor) *Runtime {
	return &Runtime{lib: lib, cursor: cursor, runs: map[string]*RunInfo{}}
}

var (
	defaultOnce sync.Once
	defaultRT   *Runtime
	defaultErr  error
)

// Default — мир заготовок процесса: каталог из ANT_FIXTURES_DIR или встроенный
// мир (Builtin); курсор в памяти, пока cmd/* не подаст курсор в Postgres (SetCursor).
func Default() (*Runtime, error) {
	defaultOnce.Do(func() {
		var lib *Library
		switch dir := os.Getenv(EnvDir); {
		case dir != "":
			lib, defaultErr = LoadFS(os.DirFS(dir), ".")
		case Builtin != nil:
			lib, defaultErr = Builtin()
		default:
			defaultErr = fmt.Errorf("мир заготовок: нет %s и встроенного мира (пакет infrastructure/fixtures/world не подключён)", EnvDir)
		}
		if defaultErr == nil {
			defaultRT = New(lib, NewMemoryCursor())
			go defaultRT.loop()
		}
	})
	return defaultRT, defaultErr
}

// SetCursor подаёт курсор миру заготовок процесса (адаптер
// infrastructure/storage/fixtures — общий для копий api, AD-36).
func SetCursor(c platform.FixtureCursor) error {
	rt, err := Default()
	if err != nil {
		return err
	}
	rt.mu.Lock()
	rt.cursor = c
	rt.mu.Unlock()
	return nil
}

// Library — библиотека сценариев.
func (r *Runtime) Library() *Library { return r.lib }

func (r *Runtime) cur() platform.FixtureCursor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.cursor
}

// State — положение курсора; пока пульт не запускал прогон — сценарий по
// умолчанию на его initial_step, на паузе, без прогона (ID без префикса).
func (r *Runtime) State(ctx context.Context) (platform.CursorState, *Scenario, error) {
	st, err := r.cur().Current(ctx)
	if err != nil {
		return st, nil, err
	}
	sc, ok := r.lib.Scenario(st.Scenario)
	if !ok {
		sc = r.lib.scenarios[r.lib.Default]
		n := sc.Manifest.InitialStep
		st = platform.CursorState{Scenario: sc.Manifest.ID, Step: n, Paused: true, Speed: 1, ClockAt: sc.Manifest.Steps[n].Clock}
	}
	if st.Step >= sc.Steps() {
		st.Step = sc.Steps() - 1
	}
	return st, sc, nil
}

// Respond — ответ операции op при параметрах params на момент m: шаг курсора
// или, при as_of, последний шаг с часами ≤ as_of (AD-36: «момент as_of на
// заготовках — шаг курсора»). Тело декодируется в out с префиксом прогона.
// Нет ответа для параметров — api.not_found; операция не наполнена — 501.
func (r *Runtime) Respond(ctx context.Context, op string, params map[string]string, m *platform.Moment, out any) error {
	st, sc, err := r.State(ctx)
	if err != nil {
		return err
	}
	n := st.Step
	if m != nil && m.AsOf != nil {
		n = sc.StepAt(*m.AsOf, st.Step)
	}
	clean := make(map[string]string, len(params))
	for k, v := range params {
		if v != "" {
			clean[k] = StripRun(v, st.RunID)
		}
	}
	e, res := r.lib.resolve(sc, n, op, clean)
	switch res {
	case NoOperation:
		return platform.NotImplemented(op)
	case NoParams:
		return notFound(op, clean)
	}
	if e.resp.Status >= 400 {
		code := errcodes.Code(e.resp.Code)
		if code == "" {
			code = errcodes.ApiNotFound
		}
		return platform.Fail(code, "operation_id", op)
	}
	body, err := sc.applyRun(e.body, st.RunID)
	if err != nil {
		return err
	}
	if body, err = r.lib.expandBlobs(body); err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("заготовка %s: тело не по типу ответа: %w", op, err)
	}
	return nil
}

// RespondAll — все ответы операции op на шаге курсора (или на момент as_of)
// при любых параметрах: тело каждого — в each (с префиксом прогона). Нужен
// спискам, которые адаптер собирает из ответов-записей (журнал событий
// страницами: journal.entry.read по seq; интерфейс 6).
func (r *Runtime) RespondAll(ctx context.Context, op string, m *platform.Moment, each func(body json.RawMessage) error) error {
	st, sc, err := r.State(ctx)
	if err != nil {
		return err
	}
	n := st.Step
	if m != nil && m.AsOf != nil {
		n = sc.StepAt(*m.AsOf, st.Step)
	}
	for _, e := range sc.steps[n].byOp[op] {
		if e.resp.Status >= 400 {
			continue
		}
		body, err := sc.applyRun(e.body, st.RunID)
		if err != nil {
			return err
		}
		if body, err = r.lib.expandBlobs(body); err != nil {
			return err
		}
		if err := each(body); err != nil {
			return err
		}
	}
	return nil
}

func notFound(op string, params map[string]string) error {
	id := canonParams(params)
	e := platform.Fail(errcodes.ApiNotFound, "object", op, "id", id)
	e.Detail = fmt.Sprintf("В мире заготовок нет ответа %s для %s", op, id)
	return e
}

// Start — запуск прогона сценария с шага 0 (пульт, FR-129; AD-38: run_id).
// mode: interactive — сценарий ждёт решений на столах ролей; autocheck —
// решения «подписывает» demo-signer: ожидания пропускаются сами.
func (r *Runtime) Start(ctx context.Context, scenario, runID, mode string, seed int64, speed int, now time.Time) (platform.CursorState, error) {
	sc, ok := r.lib.Scenario(scenario)
	if !ok {
		return platform.CursorState{}, platform.Fail(errcodes.ApiNotFound, "object", "сценарий", "id", scenario)
	}
	if speed < 1 {
		speed = 1
	}
	if mode == "" {
		mode = "interactive"
	}
	st := platform.CursorState{Scenario: scenario, RunID: runID, Step: 0, Speed: speed, ClockAt: sc.Manifest.Steps[0].Clock}
	if err := r.cur().Move(ctx, st); err != nil {
		return st, err
	}
	r.mu.Lock()
	r.runs[runID] = &RunInfo{RunID: runID, Scenario: scenario, Mode: mode, Seed: seed, StartedAt: now}
	r.lastTick = now
	r.mu.Unlock()
	r.resetSession(runID)
	return st, nil
}

// Run — сведения о прогоне (nil — прогон не запускался в этой копии api).
func (r *Runtime) Run(runID string) *RunInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if ri, ok := r.runs[runID]; ok {
		c := *ri
		return &c
	}
	return nil
}

// Stop — остановка прогона: курсор возвращается к сценарию по умолчанию без
// прогона (ID снова без префикса), прогон помечен остановленным.
func (r *Runtime) Stop(ctx context.Context, runID string, now time.Time) error {
	r.mu.Lock()
	if ri, ok := r.runs[runID]; ok {
		ri.Stopped = true
		t := now
		ri.FinishedAt = &t
	}
	r.mu.Unlock()
	st, _, err := r.State(ctx)
	if err != nil {
		return err
	}
	if st.RunID != runID {
		return nil
	}
	r.resetSession("")
	return r.cur().Move(ctx, platform.CursorState{})
}

// SetSpeed — ускорение доменных часов прогона ×1…×1000.
func (r *Runtime) SetSpeed(ctx context.Context, speed int) (platform.CursorState, error) {
	st, _, err := r.State(ctx)
	if err != nil {
		return st, err
	}
	if speed < 1 {
		speed = 1
	}
	st.Speed = speed
	return st, r.cur().Move(ctx, st)
}

// RunState — состояние прогона для пульта: running, paused,
// waiting_for_decision, completed, stopped.
func (r *Runtime) RunState(st platform.CursorState, sc *Scenario) string {
	ri := r.Run(st.RunID)
	switch {
	case ri != nil && ri.Stopped:
		return "stopped"
	case st.Step == sc.Steps()-1 && !st.ClockAt.Before(sc.Header(st.Step).Clock) && sc.Header(st.Step).Wait == nil:
		return "completed"
	case sc.Header(st.Step).Wait != nil && (ri == nil || ri.Mode != "autocheck"):
		return "waiting_for_decision"
	case st.Paused:
		return "paused"
	}
	return "running"
}

// Tick двигает доменные часы идущего прогона на прошедшее реальное время ×
// скорость (FR-129): шаги, чьи часы наступили, открываются по порядку; на шаге
// ожидания решения интерактивный прогон стоит, пока решение не принято.
func (r *Runtime) Tick(ctx context.Context, now time.Time) error {
	r.mu.Lock()
	last := r.lastTick
	r.lastTick = now
	r.mu.Unlock()
	st, sc, err := r.State(ctx)
	if err != nil || st.RunID == "" || st.Paused || last.IsZero() {
		return err
	}
	ri := r.Run(st.RunID)
	if ri != nil && ri.Stopped {
		return nil
	}
	interactive := ri == nil || ri.Mode != "autocheck"
	if interactive && sc.Header(st.Step).Wait != nil {
		return nil
	}
	clock := st.ClockAt.Add(now.Sub(last) * time.Duration(st.Speed))
	n := st.Step
	for n+1 < sc.Steps() && !sc.Header(n+1).Clock.After(clock) {
		n++
		if interactive && sc.Header(n).Wait != nil {
			clock = sc.Header(n).Clock
			break
		}
	}
	if n == sc.Steps()-1 && clock.After(sc.Header(n).Clock) {
		clock = sc.Header(n).Clock
		if ri != nil && ri.FinishedAt == nil {
			r.mu.Lock()
			t := now
			r.runs[st.RunID].FinishedAt = &t
			r.mu.Unlock()
		}
	}
	if n == st.Step && clock.Equal(st.ClockAt) {
		return nil
	}
	st.Step, st.ClockAt = n, clock
	return r.cur().Move(ctx, st)
}

// loop — фоновые часы прогона: раз в секунду Tick (реальные часы InfraClock
// двигают доменные часы сценария, AD-37).
func (r *Runtime) loop() {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for now := range t.C {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = r.Tick(ctx, now)
		cancel()
	}
}

// Seek ставит курсор на шаг n (переход к моменту; пауза сохраняется).
func (r *Runtime) Seek(ctx context.Context, n int) (platform.CursorState, error) {
	st, sc, err := r.State(ctx)
	if err != nil {
		return st, err
	}
	if n < 0 || n >= sc.Steps() {
		return st, platform.Fail(errcodes.ApiValidationFailed, "field", "step", "reason", fmt.Sprintf("шаг вне 0…%d", sc.Steps()-1))
	}
	st.Step = n
	st.ClockAt = sc.Manifest.Steps[n].Clock
	return st, r.cur().Move(ctx, st)
}

// Advance — шаг вперёд (на последнем шаге — без изменений).
func (r *Runtime) Advance(ctx context.Context) (platform.CursorState, error) {
	st, sc, err := r.State(ctx)
	if err != nil {
		return st, err
	}
	if st.Step+1 >= sc.Steps() {
		return st, nil
	}
	return r.Seek(ctx, st.Step+1)
}

// Pause и Resume — пауза и продолжение (FR-129): на паузе столы показывают
// состояние шага курсора.
func (r *Runtime) Pause(ctx context.Context) (platform.CursorState, error) {
	return r.setPaused(ctx, true)
}

// Resume — продолжение после паузы.
func (r *Runtime) Resume(ctx context.Context) (platform.CursorState, error) {
	return r.setPaused(ctx, false)
}

func (r *Runtime) setPaused(ctx context.Context, p bool) (platform.CursorState, error) {
	st, _, err := r.State(ctx)
	if err != nil {
		return st, err
	}
	st.Paused = p
	return st, r.cur().Move(ctx, st)
}

// Decide — команда человека на заготовках (FR-129): если шаг курсора ждёт
// именно этого решения над этим объектом, курсор уходит на следующий шаг —
// «сценарий продолжается». Иначе мир не меняется. Возвращает квитанцию с
// уникальным номером (seq шага и счётчик сессии, session.go); наложение
// факта поверх мира — Record.
func (r *Runtime) Decide(ctx context.Context, action string, obj ObjectRef, meta platform.CommandMeta) (platform.Receipt, error) {
	st, sc, err := r.State(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	if w := sc.Header(st.Step).Wait; w != nil && w.Action == action &&
		(w.Object.ID == "" || w.Object.ID == StripRun(obj.ID, st.RunID) || w.Object.ID == obj.ID) {
		if st, err = r.Advance(ctx); err != nil {
			return platform.Receipt{}, err
		}
	}
	r.sess.mu.Lock()
	seq := r.sess.nextSeq(st.Step)
	r.sess.mu.Unlock()
	id := kernel.UUIDv5(constants.NsAnt, fmt.Sprintf("fixtures/%s/%s/%d/%s", st.Scenario, st.RunID, st.Step, meta.CommandID))
	return platform.Receipt{CommandID: meta.CommandID, Seq: seq, EventIDs: []string{id}, RecordedAt: sc.Header(st.Step).Clock}, nil
}

// SeqPerStep — seq, отведённые шагу: записи журнала шага n — n·SeqPerStep+1…,
// изменения для SSE — с n·SeqPerStep+SeqChangesAt, весь шаг — StepSeq(n).
const (
	SeqPerStep   = 10000
	SeqChangesAt = 9000
)

// StepSeq — seq, «на котором» виден шаг n (basis_seq ответов и id событий
// SSE на заготовках).
func StepSeq(n int) int64 { return int64(n+1)*SeqPerStep - 1 }

// Clock — доменное «сейчас» шага курсора (AD-37).
func (r *Runtime) Clock(ctx context.Context) (time.Time, error) {
	st, sc, err := r.State(ctx)
	if err != nil {
		return time.Time{}, err
	}
	return sc.Header(st.Step).Clock, nil
}

// SeqChange — изменение сущности с seq и прогоном.
type SeqChange struct {
	Change
	Seq   int64
	RunID string
}

// ChangesBetween — изменения шагов (from, to] (или (to, from] при переходе
// назад) с префиксом прогона; плюс изменение прогона (пульт) и карты.
func (r *Runtime) ChangesBetween(sc *Scenario, from, to int, runID string) []SeqChange {
	lo, hi := from, to
	if lo > hi {
		lo, hi = hi, lo
	}
	var out []SeqChange
	for n := lo + 1; n <= hi && n < sc.Steps(); n++ {
		for i, c := range sc.Changes(n) {
			out = append(out, SeqChange{Change: Change{Entity: c.Entity, ID: sc.PrefixID(c.ID, runID)}, Seq: int64(n)*SeqPerStep + SeqChangesAt + int64(i), RunID: runID})
		}
	}
	last := StepSeq(to)
	out = append(out,
		SeqChange{Change: Change{Entity: string(platform.EntityLiveMap), ID: "global"}, Seq: last, RunID: runID},
		SeqChange{Change: Change{Entity: string(platform.EntityRun), ID: runOrScenario(runID, sc)}, Seq: last, RunID: runID})
	return out
}

func runOrScenario(runID string, sc *Scenario) string {
	if runID != "" {
		return runID
	}
	return sc.Manifest.ID
}
