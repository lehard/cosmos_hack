package process_test

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	engineapp "ant/internal/application/engine"
	"ant/internal/application/engine/enginemem"
	"ant/internal/application/platform"
	app "ant/internal/application/process"
	"ant/internal/application/process/proctest"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	dj "ant/internal/domain/journal"
	dp "ant/internal/domain/process"
	enginestore "ant/internal/infrastructure/storage/engine"
	journalstore "ant/internal/infrastructure/storage/journal"
	"ant/internal/infrastructure/storage/journal/feed"
	"ant/internal/infrastructure/storage/journal/journaltest"
	"ant/internal/infrastructure/storage/journal/migrator"
	processstore "ant/internal/infrastructure/storage/process"
)

// Сквозной путь process на своей БД (make dev-db): журнал эпика 04,
// хранение движка эпика 07, хранилище версий process в Postgres, воркер с
// Bundles и проектор; операции чтения и гарды команд — живая реализация.
// Без ANT_DB_HOST пропускается.

const partitions = 4

var t0 = time.Date(2026, 9, 22, 6, 0, 0, 0, time.UTC)

type core struct {
	db       *journaltest.DB
	journal  *journalstore.Store
	engine   *enginestore.Store
	codec    *engineapp.Codec
	reg      *engineapp.Registry
	versions *processstore.Versions
	bundles  *app.Bundles
	svc      *app.LiveService
	hash     string
	xml      []byte
	proj     int
	pool     *pgxpool.Pool
}

func newCore(t *testing.T) *core {
	t.Helper()
	if os.Getenv("ANT_DB_HOST") == "" {
		t.Skip("нет своей БД (make dev-db)")
	}
	ctx := context.Background()
	db := journaltest.NewDB(t)
	set := migrator.Set{Module: "process", FS: processstore.Migrations, Dir: processstore.MigrationsDir}
	if a, err := migrator.Up(ctx, db.Admin.ConnConfig, nil, set); err != nil || len(a) != 1 {
		t.Fatalf("миграция process: %v %v", a, err)
	}
	if a, err := migrator.Up(ctx, db.Admin.ConnConfig, nil, set); err != nil || len(a) != 0 {
		t.Fatalf("повтор миграции: %v %v", a, err)
	}
	p := db.AppPool(t)
	clock := journaltest.SysClock{}
	c := &core{db: db, journal: journalstore.NewStore(p, clock), engine: &enginestore.Store{Pool: p}, reg: engineapp.NewRegistry(),
		versions: &processstore.Versions{Pool: p}, pool: p}
	t.Cleanup(c.engine.Close)
	c.codec = &engineapp.Codec{Store: c.journal, Sealer: enginemem.Sealer{KeyRef: "engine@1"}, KeyRef: "engine@1",
		DomainBuild: dj.ZeroLink.String(), Partitions: partitions}
	if err := app.RegisterProjections(c.reg); err != nil {
		t.Fatal(err)
	}
	xml, err := os.ReadFile("../../../../../normative/process/flange-process.bpmn")
	if err != nil {
		t.Fatal(err)
	}
	c.xml = xml
	// FR-10: стартовая версия на чистой базе без ручных шагов; повтор — без изменений.
	seed, err := app.EnsureSeed(ctx, c.versions, xml, t0.Add(-24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if again, err := app.EnsureSeed(ctx, c.versions, xml, t0); err != nil || !again.CreatedAt.Equal(seed.CreatedAt) {
		t.Fatalf("повтор загрузки стартовой версии: %v", err)
	}
	c.hash = seed.Hash
	c.bundles = &app.Bundles{Store: c.versions, TTL: time.Nanosecond}
	c.svc = &app.LiveService{Store: c.engine, States: engineapp.StateQueries{Codec: c.codec, Bundles: c.bundles}, Library: c.versions,
		Bundles: c.bundles, Clock: func(context.Context) (time.Time, error) { return t0.Add(100 * time.Hour), nil }}
	return c
}

// sync — записи в журнал, воркер до свёртки всех записей, проектор по новым записям.
func (c *core) sync(t *testing.T, js ...*proctest.Journey) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	for _, j := range js {
		if err := proctest.Append(ctx, c.codec, c.journal, j.Facts); err != nil {
			t.Fatal(err)
		}
		j.Facts = nil
	}
	all := journaltest.ReadAll(t, c.journal, "main")
	last := map[string]int{}
	for _, e := range all {
		info, _ := catalog.Lookup(catalog.Type(e.EventType))
		if e.ItemID != nil && info.Role != engineapp.RoleWorker {
			last[*e.ItemID] = e.Seq
		}
	}
	leases, listener := journalstore.NewLeases(c.pool, journaltest.SysClock{}), journalstore.NewListener(c.pool, nil)
	lctx, lcancel := context.WithCancel(ctx)
	defer lcancel()
	go func() { _ = listener.Run(lctx) }()
	wf := feed.NewWorkFeed(c.journal, leases, listener, partitions, feed.Options{Holder: "process-test", TTL: 3 * time.Second})
	wk := engineapp.NewWorker(engineapp.WorkerConfig{Feed: wf, Codec: c.codec, Projections: c.reg, Bundles: c.bundles, Refresh: 200 * time.Millisecond})
	wctx, wcancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() { _ = wk.Run(wctx); close(done) }()
	for len(last) > 0 {
		for it, seq := range last {
			raw, ok, err := c.engine.Get(ctx, engineapp.ItemStateProjection, it)
			var st engineapp.ItemState
			if err == nil && ok && json.Unmarshal(raw, &st) == nil && st.BasisSeq >= int64(seq) {
				delete(last, it)
			}
		}
		select {
		case <-ctx.Done():
			t.Fatalf("воркер не свернул изделия: %v", last)
		case <-time.After(50 * time.Millisecond):
		}
	}
	wcancel()
	<-done
	all = journaltest.ReadAll(t, c.journal, "main")
	var fresh = all[:0:0]
	for _, e := range all {
		if e.Seq > c.proj {
			fresh = append(fresh, e)
			c.proj = e.Seq
		}
	}
	pr := &engineapp.Projector{Codec: c.codec, Store: c.engine, Registry: c.reg}
	for _, g := range c.reg.Globals() {
		rq, err := pr.Apply(ctx, g, fresh)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := c.journal.Append(ctx, rq); err != nil {
			t.Fatal(err)
		}
	}
}

func (c *core) item(t *testing.T, id string) app.ItemView {
	t.Helper()
	raw, ok, err := c.engine.Get(context.Background(), app.ProjectionItem, id)
	if err != nil || !ok {
		t.Fatalf("нет проекции изделия %s: %v", id, err)
	}
	var v app.ItemView
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func (c *core) count(t *testing.T, tp catalog.Type, item string) int {
	n := 0
	for _, e := range journaltest.ReadAll(t, c.journal, "main") {
		if e.EventType == string(tp) && e.ItemID != nil && *e.ItemID == item {
			n++
		}
	}
	return n
}

func onStep(v app.ItemView, step string) bool {
	for _, tk := range v.Tokens {
		if tk.StepKey == step {
			return true
		}
	}
	return false
}

func code(err error) errcodes.Code {
	if pe, ok := platform.AsError(err); ok {
		return pe.Code
	}
	return ""
}

// Готово эпика 17 на своей БД: окно кромок, подпроцесс брака с возвратом,
// 4-я доработка, точка предъявления без подписи, подделка XML.
func TestProcessOnPostgres(t *testing.T) {
	c := newCore(t)
	ctx := context.Background()

	// 1. Истечение окна «кромки → сварка» уводит на повторную подготовку.
	a := proctest.New("ent01:FL-A", c.hash, t0)
	a.ToWelding()
	a.Run("RUN-A-W1-1", "welding.edge_prep", 6, 7)
	a.Run("RUN-A-W2-1", "welding.weld", 15.5, 0)
	// 2. Подпроцесс брака: вызов из сварки, возврат в сварку.
	b := proctest.New("ent01:FL-B", c.hash, t0)
	b.ToWelding()
	b.Run("RUN-B-W1-1", "welding.edge_prep", 6, 6.5)
	b.Run("RUN-B-W2-1", "welding.weld", 7, 7.5)
	b.Inspect("welding.kt3_camera", "defect_indicated", 8, "W-1.U3")
	b.Inspect("welding.kt3_radiography", "no_defect_indicated", 8.5, "W-1.U3")
	b.Decide("welding.zt3_acceptance", "reject", 9)
	b.Dispose("rework", 9.5)
	// 3. Четыре цикла доработки зоны W-1.U3.
	r := proctest.New("ent01:FL-R", c.hash, t0)
	r.ToWelding()
	h := 6.0
	for i := 1; i <= 4; i++ {
		r.Run("RUN-R-W1-"+itoa(i), "welding.edge_prep", h, h+0.5)
		r.Run("RUN-R-W2-"+itoa(i), "welding.weld", h+1, h+1.5)
		r.Inspect("welding.kt3_camera", "defect_indicated", h+2, "W-1.U3")
		r.Inspect("welding.kt3_radiography", "no_defect_indicated", h+2.5, "W-1.U3")
		r.Decide("welding.zt3_acceptance", "reject", h+3)
		r.Dispose("rework", h+3.5)
		h += 4
	}
	r.Add(catalog.DecisionReworkLimitWaived, h-0.1, map[string]any{"zone_id": "welding.edge_prep", "used": 4, "limit": 3, "extra_allowed": 1,
		"reason": map[string]string{"code": "x", "text": "x"}})
	r.Run("RUN-R-W1-5", "welding.edge_prep", h, h+0.5)
	// 4. Точка предъявления ЗТ-2 без решения: факты дальше не двигают изделие.
	g := proctest.New("ent01:FL-G", c.hash, t0)
	g.Register(0)
	g.Decide("incoming.zt1_lot_acceptance", "accept", 1)
	g.Run("RUN-G-M1-1", "machining.cnc", 2, 3)
	g.Inspect("machining.kt2_camera", "no_defect_indicated", 3.5, "F-FACE")
	g.Inspect("machining.kt2_cmm", "no_defect_indicated", 3.7, "F-FACE")
	g.Run("RUN-G-M5-1", "machining.send_to_welding", 5, 6)
	c.sync(t, a, b, r, g)

	if v := c.item(t, a.Item); !onStep(v, "welding.edge_prep") || v.Visits == nil {
		t.Fatalf("A: окно кромок — изделие на %+v", v.Tokens)
	}
	if c.count(t, catalog.OperationPreconditionFailed, a.Item) != 1 {
		t.Fatal("A: нет реакции нарушения окна")
	}
	if v := c.item(t, b.Item); !onStep(v, "welding.edge_prep") || len(v.Tokens) != 2 {
		t.Fatalf("B: подпроцесс брака не вернул в сварку: %+v", v.Tokens)
	}
	if c.count(t, catalog.OperationMessageThrown, b.Item) < 4 {
		t.Fatalf("B: сообщения «в 1С» (принято в работу ×2, перемещение, перевод в брак): %d", c.count(t, catalog.OperationMessageThrown, b.Item))
	}
	// Команда сварщика через час после кромок — в окне 8 ч.
	c.svc.Clock = func(context.Context) (time.Time, error) { return r.At(h + 1.5), nil }
	_, err := c.svc.StartOperation(ctx, r.Item, app.StartOperation{OperationRunID: "RUN-R-W2-5", OperationCode: "030", StepKey: "welding.weld"})
	if code(err) != errcodes.ProcessReworkLimitExceeded {
		t.Fatalf("R: 4-я доработка зоны не заблокирована: %v", err)
	}
	if pe, _ := platform.AsError(err); pe.Params["zone"] != "W-1.U3" || pe.Params["used"] != "4" || pe.Params["limit"] != "3" {
		t.Fatalf("R: параметры отказа %+v", pe.Params)
	}
	if v := c.item(t, g.Item); !onStep(v, "machining.zt2_acceptance") {
		t.Fatalf("G: изделие прошло ЗТ-2 без решения: %+v", v.Tokens)
	}
	_, err = c.svc.StartOperation(ctx, g.Item, app.StartOperation{OperationRunID: "RUN-G-W1-1", OperationCode: "020", StepKey: "welding.edge_prep"})
	if code(err) != errcodes.NonconformityGateWithoutSignature {
		t.Fatalf("G: гард пропустил за ЗТ-2 без подписи: %v", err)
	}

	// Живая карта по проекциям в Postgres.
	lm, err := c.svc.LiveMap(ctx, app.LiveMapQuery{Period: "day"}, platform.Moment{})
	if err != nil {
		t.Fatal(err)
	}
	if len(lm.Items) != 4 || lm.ProcessVersion.Items != 4 {
		t.Fatalf("живая карта: %d изделий, версия %+v", len(lm.Items), lm.ProcessVersion)
	}

	// 5. Подделка XML действующей версии в обход системы (прямой UPDATE):
	// следующая свёртка изделия его не исполняет, гард отказывает.
	tampered := strings.Replace(string(c.xml), `reworkLimit="3" reworkLimitScope="zone"`, `reworkLimit="99" reworkLimitScope="zone"`, 1)
	if _, err := c.db.AdminConn(t).Exec(ctx, `UPDATE process.versions SET xml = $1 WHERE version_id = $2`, []byte(tampered), app.SeedVersionID); err != nil {
		t.Fatal(err)
	}
	g.Decide("machining.zt2_acceptance", "accept", 7)
	c.sync(t, g)
	v := c.item(t, g.Item)
	if v.Refused != string(errcodes.ProcessVersionTampered) || len(v.Tokens) != 0 {
		t.Fatalf("G: изменённый XML исполнен: refused=%q токены %+v", v.Refused, v.Tokens)
	}
	_, err = c.svc.StartOperation(ctx, r.Item, app.StartOperation{OperationRunID: "RUN-R-W2-5", OperationCode: "030", StepKey: "welding.weld"})
	if code(err) != errcodes.ProcessVersionTampered {
		t.Fatalf("R: команда по изменённой версии: %v", err)
	}
	if bp, _ := c.svc.Bpmn(ctx, app.SeedVersionID); dp.VersionHash([]byte(bp.BpmnXML)) == bp.Hash {
		t.Fatal("подделка не видна: хеш совпал")
	}
}

func itoa(i int) string { return string(rune('0' + i)) }
