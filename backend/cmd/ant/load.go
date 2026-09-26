package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/http/cookiejar"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	crossitemapp "ant/internal/application/crossitem"
	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/constants"
	jc "ant/internal/contracts/journal"
	"ant/internal/domain/engine"
	"ant/internal/domain/kernel"
	journalstore "ant/internal/infrastructure/storage/journal"
)

// Роль load (эпик 35, FR-107, AD-6, SM-3): нагрузочный прогон и две проверки
// «1 = N». Разовая роль в образе ant, её запускает make load
// (scenarios/load/load.sh) в проекте compose нагрузки:
//
//  1. (если задан ANT_LOAD_SCENARIO) запускает прогон сценария через обычный
//     API — генератор интенсивности это раннер симуляции (эпик 32): события
//     источников идут в обычный приём, решения — demo-signer;
//  2. ждёт конца прогона и того, что воркеры, стадия и проектор догнали журнал;
//  3. считает state_hash итогового состояния прогона (свёртка каждого
//     изделия, engine.StateHash — без seq, basis_seq и времени записи) и
//     rebuild_hash пересборки проекций над тем же журналом на 1 и на N
//     параллельных обработчиках (изделия делятся по партициям, как у воркеров);
//  4. пишет отчёт JSON: задержки, потери, повторы, полнота, хеши.
//
// Сравнение state_hash раздельных прогонов одного seed на 1 и N воркерах
// делает сценарий make load по двум отчётам. Роль только читает журнал и
// проекции; пересборка для rebuild_hash идёт в памяти и ничего не пишет.
func init() {
	roles["load"] = role{run: runLoad, oneShot: true}
}

// loadParams — параметры роли load (переменные окружения ANT_LOAD_*).
type loadParams struct {
	API      string
	Scenario string
	Seed     int64
	Persona  string
	RunID    string
	Workers  int
	Out      string
	Label    string
	Timeout  time.Duration
}

func loadParamsFromEnv() loadParams {
	p := loadParams{
		API:      strings.TrimRight(os.Getenv("ANT_LOAD_API"), "/"),
		Scenario: os.Getenv("ANT_LOAD_SCENARIO"),
		Persona:  envOr("ANT_LOAD_PERSONA", "ADM-01"),
		RunID:    os.Getenv("ANT_LOAD_RUN_ID"),
		Workers:  4,
		Out:      os.Getenv("ANT_LOAD_OUT"),
		Label:    os.Getenv("ANT_LOAD_LABEL"),
		Timeout:  90 * time.Minute,
	}
	if v, err := strconv.ParseInt(os.Getenv("ANT_LOAD_SEED"), 10, 64); err == nil {
		p.Seed = v
	}
	if v, err := strconv.Atoi(os.Getenv("ANT_LOAD_WORKERS")); err == nil && v > 0 {
		p.Workers = v
	}
	if v, err := time.ParseDuration(os.Getenv("ANT_LOAD_TIMEOUT")); err == nil && v > 0 {
		p.Timeout = v
	}
	return p
}

// LoadReport — отчёт нагрузочного прогона (scenarios/load/README.md).
type LoadReport struct {
	Label    string `json:"label,omitempty"`
	Scenario string `json:"scenario,omitempty"`
	RunID    string `json:"run_id"`
	Seed     int64  `json:"seed,omitempty"`
	State    string `json:"state,omitempty"`
	// DurationS — длительность прогона по часам раннера (старт → конец).
	DurationS float64 `json:"duration_s,omitempty"`
	// BoardPassed/BoardTotal, BoardHash — табло автосверки и хеш его строк
	// (assertion_id и статус): у раздельных прогонов одного seed совпадает.
	BoardPassed int    `json:"board_passed"`
	BoardTotal  int    `json:"board_total"`
	BoardHash   string `json:"board_hash,omitempty"`
	// BoardActualHash — хеш полученных значений строк табло.
	BoardActualHash string `json:"board_actual_hash,omitempty"`
	Journal         struct {
		Entries   int     `json:"entries"`
		Facts     int     `json:"facts"`
		Reactions int     `json:"reactions"`
		Decisions int     `json:"decisions"`
		Failed    int     `json:"processing_failed"`
		FactsPerS float64 `json:"facts_per_s"`
	} `json:"journal"`
	// Ingest — потери, повторы и карантин по источникам прогона.
	Ingest struct {
		Received    int `json:"received"`
		Accepted    int `json:"accepted"`
		Duplicates  int `json:"duplicates"`
		Quarantined int `json:"quarantined"`
		Conflicts   int `json:"conflicts"`
		Gaps        int `json:"gaps"`
	} `json:"ingest"`
	// Completeness — полнота: изделия прогона, из них с остановленной обработкой.
	Completeness struct {
		Items        int     `json:"items"`
		ItemsStopped int     `json:"items_stopped"`
		FactsPerItem float64 `json:"facts_per_item"`
	} `json:"completeness"`
	// LatencyMS — факт изделия → первая реакция движка на него (реальное
	// время фиксации, committed_at): приём + свёртка + запись.
	LatencyMS struct {
		N   int     `json:"n"`
		P50 float64 `json:"p50"`
		P95 float64 `json:"p95"`
		Max float64 `json:"max"`
	} `json:"latency_ms"`
	Hashes struct {
		State      string     `json:"state_hash"`
		Rebuild1   string     `json:"rebuild_hash_1"`
		RebuildN   string     `json:"rebuild_hash_n"`
		Workers    int        `json:"n"`
		RebuildEq  bool       `json:"rebuild_equal"`
		RebuildSec [2]float64 `json:"rebuild_s"`
	} `json:"hashes"`
}

func runLoad(ctx context.Context, env *environment) error {
	p := loadParamsFromEnv()
	c, err := env.readyCore(ctx)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, p.Timeout)
	defer cancel()
	rep := LoadReport{Label: p.Label, Scenario: p.Scenario, RunID: p.RunID, Seed: p.Seed}
	var api *loadAPI
	if p.API != "" {
		if api, err = newLoadAPI(ctx, p.API, p.Persona); err != nil {
			return err
		}
	}
	if p.Scenario != "" {
		if api == nil {
			return errors.New("load: ANT_LOAD_SCENARIO без ANT_LOAD_API")
		}
		if rep.RunID, err = api.start(ctx, p.Scenario, p.Seed); err != nil {
			return err
		}
		env.log.Info("load: прогон запущен", "scenario", p.Scenario, "run_id", rep.RunID, "seed", p.Seed)
	}
	if api != nil && rep.RunID != "" {
		if err := api.wait(ctx, rep.RunID, &rep, env); err != nil {
			return err
		}
	}
	// Воркеры, стадия и проектор догнали журнал — считаем итог.
	settler := &journalstore.Settler{Store: c.journal, Signal: c.listener, Partitions: env.cfg.Engine.Partitions,
		Timeout: 5 * time.Minute, Consumers: func() []string {
			names := []string{crossitemapp.ConsumerStage}
			for _, g := range c.registry.Globals() {
				names = append(names, engineapp.ConsumerName(g.Name))
			}
			return names
		}}
	// ANT_LOAD_SETTLE=0 — не ждать догонки (разбор БД остановленного стенда).
	if os.Getenv("ANT_LOAD_SETTLE") != "0" {
		if err := settler.Settle(ctx, rep.RunID); err != nil {
			return err
		}
	}
	if err := loadJournalStats(ctx, c, &rep); err != nil {
		return err
	}
	items, err := loadItems(ctx, c, rep.RunID)
	if err != nil {
		return err
	}
	if rep.Hashes.State, rep.Completeness.ItemsStopped, err = stateHash(ctx, c, items); err != nil {
		return err
	}
	rep.Completeness.Items = len(items)
	if len(items) > 0 {
		rep.Completeness.FactsPerItem = float64(rep.Journal.Facts) / float64(len(items))
	}
	rep.Hashes.Workers = p.Workers
	t0 := time.Now()
	if rep.Hashes.Rebuild1, err = rebuildHash(ctx, c, items, 1); err != nil {
		return err
	}
	t1 := time.Now()
	if rep.Hashes.RebuildN, err = rebuildHash(ctx, c, items, p.Workers); err != nil {
		return err
	}
	rep.Hashes.RebuildSec = [2]float64{t1.Sub(t0).Seconds(), time.Since(t1).Seconds()}
	rep.Hashes.RebuildEq = rep.Hashes.Rebuild1 == rep.Hashes.RebuildN
	// Отчёт — одной строкой с меткой в stdout (журнал процесса — тоже stdout,
	// scenarios/load/load.sh выбирает строку по метке) и, если задан, файлом.
	line, _ := json.Marshal(rep)
	_, _ = fmt.Fprintln(os.Stdout, "LOAD-REPORT "+string(line))
	if p.Out != "" {
		out, _ := json.MarshalIndent(rep, "", "  ")
		if err := os.WriteFile(p.Out, append(out, '\n'), 0o644); err != nil {
			return err
		}
	}
	env.log.Info("load: готово", "run_id", rep.RunID, "state_hash", rep.Hashes.State,
		"rebuild_hash_1", rep.Hashes.Rebuild1, "rebuild_hash_n", rep.Hashes.RebuildN, "n", p.Workers)
	if !rep.Hashes.RebuildEq {
		return fmt.Errorf("load: rebuild_hash на 1 и %d обработчиках разный", p.Workers)
	}
	return nil
}

// loadItems — изделия прогона (все изделия журнала, если runID пуст).
func loadItems(ctx context.Context, c *core, runID string) ([]string, error) {
	rows, err := c.pool.Query(ctx, `SELECT DISTINCT item_id FROM journal.entries
WHERE chain = 'main' AND item_id IS NOT NULL AND item_id <> '' AND ($1 = '' OR run_id = $1) ORDER BY item_id`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// stateHash — state_hash итогового состояния (AD-6): H(отсортированные
// «изделие: StateHash свёртки»); StateHash не зависит от порядка записи,
// seq и basis_seq.
func stateHash(ctx context.Context, c *core, items []string) (string, int, error) {
	q := engineapp.StateQueries{Codec: c.codec, Bundles: c.bundleSource()}
	lines := make([]string, 0, len(items))
	stopped := 0
	for _, id := range items {
		at, err := q.Item(ctx, id, platform.Moment{})
		if err != nil {
			return "", 0, fmt.Errorf("изделие %s: %w", id, err)
		}
		if at.Stopped {
			stopped++
		}
		lines = append(lines, id+":"+at.StateHash)
		if os.Getenv("ANT_LOAD_ITEMS") == "1" {
			// Хеш по изделию — сверка прогонов, прерванных на разном шаге.
			_, _ = fmt.Fprintln(os.Stdout, "LOAD-ITEM "+id+" "+at.StateHash)
		}
	}
	return engine.Hash([]byte(strings.Join(lines, "\n"))), stopped, nil
}

// rebuildHash — rebuild_hash (AD-6): все проекции изделий и вклады заново
// свёрткой, глобальные — переигрыванием журнала по seq; workers обработчиков
// параллельно, изделия делятся по партициям (kernel.PartitionOf), глобальные
// проекции — по одной на обработчик. Ничего не пишет.
func rebuildHash(ctx context.Context, c *core, items []string, workers int) (string, error) {
	q := engineapp.StateQueries{Codec: c.codec, Bundles: c.bundleSource()}
	var (
		mu   sync.Mutex
		rows []string
		errs []error
		wg   sync.WaitGroup
	)
	add := func(r []string, err error) {
		mu.Lock()
		defer mu.Unlock()
		rows = append(rows, r...)
		if err != nil {
			errs = append(errs, err)
		}
	}
	for w := range workers {
		wg.Go(func() {
			for _, id := range items {
				if kernel.PartitionOf(id, workers) != w {
					continue
				}
				at, err := q.Item(ctx, id, platform.Moment{})
				if err != nil {
					add(nil, fmt.Errorf("изделие %s: %w", id, err))
					return
				}
				if at.Stopped {
					continue
				}
				in, err := c.codec.LoadItem(ctx, id, 0)
				if err != nil {
					add(nil, err)
					return
				}
				effects, err := c.registry.ItemEffects(id, at.Snapshot, at.Reactions, in.Input)
				add(effectRows(effects), err)
			}
		})
	}
	all, err := readAll(ctx, c)
	if err != nil {
		return "", err
	}
	decoded := make([]kernel.Record, 0, len(all))
	for _, e := range all {
		d, err := c.codec.Decode(ctx, e)
		if err != nil {
			return "", fmt.Errorf("seq %d: %w", e.Seq, err)
		}
		decoded = append(decoded, d.Record)
	}
	globals := c.registry.Globals()
	for w := range workers {
		wg.Go(func() {
			for i, g := range globals {
				if i%workers != w {
					continue
				}
				state := map[string]json.RawMessage{}
				for _, r := range decoded {
					for _, k := range g.Keys(r) {
						next, err := g.Step(k, state[k], r)
						if err != nil {
							add(nil, fmt.Errorf("проекция %s, seq %d: %w", g.Name, r.Seq, err))
							return
						}
						state[k] = next
					}
				}
				var effects []appjournal.Effect
				for _, k := range slices.Sorted(maps.Keys(state)) {
					effects = append(effects, engineapp.ProjectionPut{Name: g.Name, Key: k, Value: state[k]})
				}
				add(effectRows(effects), nil)
			}
		})
	}
	wg.Wait()
	if err := errors.Join(errs...); err != nil {
		return "", err
	}
	slices.Sort(rows)
	return engine.Hash([]byte(strings.Join(rows, "\x1e"))), nil
}

// effectRows — строки хеша так же, как у ant rebuild (engine.hashEffects).
func effectRows(effects []appjournal.Effect) []string {
	var rows []string
	for _, e := range effects {
		switch x := e.(type) {
		case engineapp.ProjectionPut:
			rows = append(rows, x.Name+"\x1f"+x.Key+"\x1f"+string(x.Value))
		case engineapp.ContributionsReplace:
			b, _ := engine.Canonical(x.Rows)
			rows = append(rows, "contributions\x1f"+x.ItemID+"\x1f"+string(b))
		}
	}
	return rows
}

func readAll(ctx context.Context, c *core) ([]jc.JournalEntry, error) {
	var all []jc.JournalEntry
	after := int64(0)
	for {
		page, err := c.journal.Read(ctx, appjournal.ReadQuery{AfterSeq: after, Limit: 1000})
		if err != nil {
			return nil, err
		}
		all = append(all, page...)
		if len(page) < 1000 {
			return all, nil
		}
		after = int64(page[len(page)-1].Seq)
	}
}

// loadJournalStats — счётчики журнала и приёма, задержки по фиксации.
func loadJournalStats(ctx context.Context, c *core, rep *LoadReport) error {
	run := rep.RunID
	j := &rep.Journal
	if err := c.pool.QueryRow(ctx, `SELECT count(*),
    count(*) FILTER (WHERE entry_kind = 'fact'),
    count(*) FILTER (WHERE entry_kind = 'reaction'),
    count(*) FILTER (WHERE entry_kind = 'decision'),
    count(*) FILTER (WHERE event_type = 'ops.processing.failed'),
    COALESCE(EXTRACT(EPOCH FROM max(committed_at) FILTER (WHERE entry_kind = 'fact') - min(committed_at) FILTER (WHERE entry_kind = 'fact')), 0)
FROM journal.entries WHERE chain = 'main' AND ($1 = '' OR run_id = $1)`, run).Scan(
		&j.Entries, &j.Facts, &j.Reactions, &j.Decisions, &j.Failed, &j.FactsPerS); err != nil {
		return err
	}
	if j.FactsPerS > 0 {
		j.FactsPerS = float64(j.Facts) / j.FactsPerS
	}
	prefix := run + "/"
	in := &rep.Ingest
	if err := c.pool.QueryRow(ctx, `SELECT
    COALESCE((SELECT sum((state->>'Received')::bigint) FROM ingest.sources WHERE $1 = '/' OR starts_with(source_id, $1)), 0),
    COALESCE((SELECT sum(jsonb_array_length(COALESCE(NULLIF(state->'Gaps', 'null'::jsonb), '[]'::jsonb))) FROM ingest.sources WHERE $1 = '/' OR starts_with(source_id, $1)), 0),
    (SELECT count(*) FROM ingest.seen WHERE $1 = '/' OR starts_with(source_id, $1)),
    (SELECT count(*) FROM ingest.quarantine WHERE $1 = '/' OR starts_with(source_id, $1)),
    (SELECT count(*) FROM journal.entries WHERE chain = 'main' AND event_type = 'security.idempotency.conflict' AND ($2 = '' OR run_id = $2))`,
		prefix, run).Scan(&in.Received, &in.Gaps, &in.Accepted, &in.Quarantined, &in.Conflicts); err != nil {
		return err
	}
	in.Duplicates = max(in.Received-in.Accepted-in.Quarantined, 0)
	l := &rep.LatencyMS
	return c.pool.QueryRow(ctx, `WITH lat AS (
    SELECT EXTRACT(EPOCH FROM r.committed_at - f.committed_at) * 1000 AS ms
    FROM journal.entries f
    CROSS JOIN LATERAL (
        SELECT committed_at FROM journal.entries r
        WHERE r.chain = 'main' AND r.item_id = f.item_id AND r.entry_kind = 'reaction' AND r.seq > f.seq
        ORDER BY r.seq LIMIT 1
    ) r
    WHERE f.chain = 'main' AND f.entry_kind = 'fact' AND f.item_id IS NOT NULL AND ($1 = '' OR f.run_id = $1)
)
SELECT count(*), COALESCE(percentile_cont(0.5) WITHIN GROUP (ORDER BY ms), 0),
    COALESCE(percentile_cont(0.95) WITHIN GROUP (ORDER BY ms), 0), COALESCE(max(ms), 0) FROM lat`, run).Scan(&l.N, &l.P50, &l.P95, &l.Max)
}

// loadAPI — клиент HTTP API ant от имени демо-персоны (профили demo, load).
type loadAPI struct {
	base string
	http *http.Client
}

func newLoadAPI(ctx context.Context, base, persona string) (*loadAPI, error) {
	jar, _ := cookiejar.New(nil)
	a := &loadAPI{base: base, http: &http.Client{Jar: jar, Timeout: 30 * time.Second}}
	// API может ещё подниматься: вход с повтором до минуты.
	var last error
	for range 60 {
		if last = a.do(ctx, http.MethodPost, "/api/v1/auth/session", map[string]any{"persona_id": persona}, nil); last == nil {
			return a, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
	}
	return nil, fmt.Errorf("load: вход %s: %w", persona, last)
}

func (a *loadAPI) do(ctx context.Context, method, path string, body any, out any) error {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rd = strings.NewReader(string(b))
	}
	req, err := http.NewRequestWithContext(ctx, method, a.base+path, rd)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s: %d %s", method, path, resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if out != nil {
		return json.Unmarshal(raw, out)
	}
	return nil
}

func (a *loadAPI) start(ctx context.Context, scenario string, seed int64) (string, error) {
	body := map[string]any{"mode": "load", "speed": 1000, "basis_seq": 0, "policy_seq": 0,
		"command_id": kernel.UUIDv5(constants.NsAnt, fmt.Sprintf("ant-load|%s|%d|%d", scenario, seed, time.Now().UnixNano()))}
	if seed > 0 {
		body["seed"] = seed
	}
	var out struct {
		RunID string `json:"run_id"`
	}
	if err := a.do(ctx, http.MethodPost, "/api/v1/scenarios/"+scenario+"/runs", body, &out); err != nil {
		return "", err
	}
	return out.RunID, nil
}

// wait — ждать конца прогона (опрос состояния раз в 2 с) и снять табло.
func (a *loadAPI) wait(ctx context.Context, runID string, rep *LoadReport, env *environment) error {
	var st struct {
		State       string     `json:"state"`
		Step        int        `json:"step"`
		Steps       int        `json:"steps"`
		StartedAt   time.Time  `json:"started_at"`
		FinishedAt  *time.Time `json:"finished_at"`
		BoardPassed int        `json:"board_passed"`
		BoardTotal  int        `json:"board_total"`
		Seed        int64      `json:"seed"`
	}
	last := time.Now()
	for {
		if err := a.do(ctx, http.MethodGet, "/api/v1/runs/"+runID, nil, &st); err != nil {
			return err
		}
		if st.State != "running" {
			break
		}
		if time.Since(last) > 30*time.Second {
			env.log.Info("load: прогон идёт", "run_id", runID, "step", st.Step, "steps", st.Steps)
			last = time.Now()
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("load: прогон %s не закончился: %w", runID, ctx.Err())
		case <-time.After(2 * time.Second):
		}
	}
	rep.State, rep.BoardPassed, rep.BoardTotal, rep.Seed = st.State, st.BoardPassed, st.BoardTotal, st.Seed
	if st.FinishedAt != nil {
		rep.DurationS = st.FinishedAt.Sub(st.StartedAt).Seconds()
	}
	var board struct {
		Rows []struct {
			AssertionID string  `json:"assertion_id"`
			Status      string  `json:"status"`
			Actual      *string `json:"actual"`
		} `json:"rows"`
	}
	if err := a.do(ctx, http.MethodGet, "/api/v1/runs/"+runID+"/board", nil, &board); err == nil {
		status := make([]string, 0, len(board.Rows))
		actual := make([]string, 0, len(board.Rows))
		for _, r := range board.Rows {
			status = append(status, r.AssertionID+"="+r.Status)
			v := "null"
			if r.Actual != nil {
				v = *r.Actual
			}
			actual = append(actual, r.AssertionID+"="+v)
		}
		slices.Sort(status)
		slices.Sort(actual)
		rep.BoardHash = engine.Hash([]byte(strings.Join(status, "\n")))
		rep.BoardActualHash = engine.Hash([]byte(strings.Join(actual, "\n")))
	}
	if st.State != "completed" {
		return fmt.Errorf("load: прогон %s закончился в состоянии %s", runID, st.State)
	}
	return nil
}
