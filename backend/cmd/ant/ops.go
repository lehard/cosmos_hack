package main

import (
	"context"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	erpapp "ant/internal/application/erp"
	ingestapp "ant/internal/application/ingest"
	opsapp "ant/internal/application/ops"
	"ant/internal/application/platform"
	dom "ant/internal/domain/ops"
	"ant/internal/infrastructure/observability/logging"
	"ant/internal/infrastructure/observability/telemetry"
	erpstore "ant/internal/infrastructure/storage/erp"
	journalstore "ant/internal/infrastructure/storage/journal"
	"ant/internal/infrastructure/storage/journal/migrator"
	opsstore "ant/internal/infrastructure/storage/ops"
)

// Эпик 34 — эксплуатация и наблюдаемость: телеметрия процесса (порт
// Telemetry, адаптер prometheus, /metrics роли api), модуль ops вживую
// (состояние ролей, очередей, интеграций, остановленные изделия, настройки),
// порт ops для записей ops.integration.degraded (эпик 30) и самопроверка
// после старта (FR-109, FR-113, FR-127; AD-25, AD-35, AD-45).

// observability — телеметрия процесса: одна на процесс, общая для его ролей
// (роли одного процесса делят registry — /metrics роли api видит и приём, и
// воркер, и живые обновления).
type observability struct {
	once sync.Once
	tel  platform.Telemetry
	prom *telemetry.Prometheus
}

// telemetry — порт Telemetry процесса по ключу ports.adapters.telemetry.
// Неизвестный адаптер — пустышка с ошибкой в логе (метрики не должны
// останавливать процесс).
func (e *environment) telemetry() platform.Telemetry {
	e.obs.once.Do(func() {
		tel, prom, err := telemetry.New(e.cfg.Ports.Adapters[string(platform.PortTelemetry)], e.moduleLog("ops"))
		if err != nil {
			e.log.Error("телеметрия: адаптер не собран — метрики не выгружаются", logging.KeyModule, "ops", "err", err)
			tel = telemetry.Nop{}
		}
		e.obs.tel, e.obs.prom = tel, prom
	})
	return e.obs.tel
}

// moduleLog — логгер модуля: поле module по соглашению спайна («Логи»).
func (e *environment) moduleLog(module string) *slog.Logger {
	return e.log.With(logging.KeyModule, module)
}

// metricsHandler — /metrics (FR-41, FR-113): перед выдачей ops снимает свои
// показатели (остановленные изделия, отставание потребителей, интеграции).
func (e *environment) metricsHandler(ops *opsapp.Service) http.Handler {
	_ = e.telemetry()
	if e.obs.prom == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "метрики не выгружаются: ports.adapters.telemetry не prometheus", http.StatusNotFound)
		})
	}
	h := e.obs.prom.Handler()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ops != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			if err := ops.Collect(ctx); err != nil {
				e.log.Warn("метрики ops не сняты", logging.KeyModule, "ops", "err", err)
			}
			cancel()
		}
		h.ServeHTTP(w, r)
	})
}

// opsLive — live-реализация модуля ops для роли api на ядре процесса;
// pool — пул роли api (пользователь конфигурации: читает pg_roles и таблицы
// версий миграций), ingest — живой приём (карантин), nil — без него.
func opsLive(ctx context.Context, env *environment, pool *pgxpool.Pool, ingest *ingestapp.Service) (*opsapp.Service, error) {
	c, err := env.core(ctx)
	if err != nil {
		return nil, err
	}
	cfg := env.cfg
	oc := opsapp.Config{
		Journal: c.journal, Codec: c.codec,
		Runtime:  opsRuntime{store: c.journal, leases: c.leases},
		Database: opsDatabase{Probe: opsstore.Probe{Pool: pool}, pool: pool},
		Exchanges: []opsapp.Exchange{
			// Эпик 30: очередь исходящих и канал 1С (схема erp). TODO(31):
			// Галактика и MES — их модули добавляют сюда свой Exchange.
			erpExchange{store: erpstore.NewStore(c.pool), systems: []string{"onec"}},
		},
		Partitions: cfg.Engine.Partitions, Enabled: cfg.Integrations.Enabled,
		Profile: cfg.Profile, Version: version, Mode: platform.Mode(cfg.Ports.Mode),
		Adapters:         cfg.Ports.Adapters,
		VerifierInterval: cfg.Security.VerifierInterval,
		Now:              c.codec.Now, Clock: c.domainClock(), DomainBuild: c.codec.DomainBuild,
		Telemetry: env.telemetry(), Log: env.moduleLog("ops"),
	}
	if ingest != nil {
		oc.Quarantine = ingestQuarantine{ingest}
	}
	return opsapp.NewLive(oc), nil
}

// opsReporter — порт ops для других модулей (ops.processing.failed,
// ops.integration.degraded) на ядре процесса.
func opsReporter(env *environment, c *core) *opsapp.Reporter {
	return &opsapp.Reporter{Journal: c.journal, Codec: c.codec, Clock: c.domainClock(), Now: c.codec.Now,
		Log: env.moduleLog("ops")}
}

// opsRuntime — аренды и курсоры из хранения журнала (схема journal_state
// принадлежит модулю journal; ops читает её только через его адаптер).
type opsRuntime struct {
	store  *journalstore.Store
	leases *journalstore.Leases
}

func (r opsRuntime) Leases(ctx context.Context) ([]dom.Lease, error) {
	ls, err := r.leases.All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]dom.Lease, 0, len(ls))
	for _, l := range ls {
		out = append(out, dom.Lease{Name: l.Name, Holder: l.Holder, Epoch: l.Epoch, ExpiresAt: l.ExpiresAt})
	}
	return out, nil
}

func (r opsRuntime) Backlogs(ctx context.Context, partitions int) ([]opsapp.Backlog, error) {
	w, err := r.store.WorkerBacklog(ctx, partitions)
	if err != nil {
		return nil, err
	}
	g, err := r.store.GlobalBacklog(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]opsapp.Backlog, 0, len(w)+len(g))
	for _, b := range append(w, g...) {
		out = append(out, opsapp.Backlog{Consumer: b.Name, Partition: b.Partition, Seq: b.Seq, Head: b.Head, Pending: b.Pending})
	}
	return out, nil
}

// opsDatabase — проверки базы: ответ, роли БД и права (storage/ops),
// миграции модулей сборки (migrationSets, migrator.Status).
type opsDatabase struct {
	opsstore.Probe
	pool *pgxpool.Pool
}

func (d opsDatabase) Migrations(ctx context.Context) []dom.Migration {
	var out []dom.Migration
	for _, s := range migrator.Status(ctx, d.pool, migrationSets...) {
		m := dom.Migration{Module: s.Module, Current: s.Current, Target: s.Target, Pending: len(s.Pending)}
		if s.Err != nil {
			m.Err = s.Err.Error()
		}
		out = append(out, m)
	}
	return out
}

// erpExchange — очередь исходящих и каналы обмена модуля erp (роль outbox).
type erpExchange struct {
	store   *erpstore.Store
	systems []string
}

func (x erpExchange) Exchange(ctx context.Context) ([]opsapp.ExchangeQueue, []dom.Channel, error) {
	var qs []opsapp.ExchangeQueue
	for _, sys := range x.systems {
		q, quar, err := x.store.Counts(ctx, sys)
		if err != nil {
			return nil, nil, err
		}
		qs = append(qs, opsapp.ExchangeQueue{System: sys, Queued: int64(q), Quarantined: int64(quar), Consumer: erpapp.ConsumerOutbox})
	}
	chs, err := x.store.Channels(ctx)
	if err != nil {
		return nil, nil, err
	}
	out := make([]dom.Channel, 0, len(chs))
	for _, c := range chs {
		out = append(out, dom.Channel{System: c.System, State: c.State, Detail: c.Detail, CheckedAt: c.CheckedAt})
	}
	return qs, out, nil
}

// ingestQuarantine — открытый карантин живого приёма.
type ingestQuarantine struct{ s *ingestapp.Service }

func (q ingestQuarantine) QuarantineOpen(ctx context.Context) (int64, error) {
	st, err := q.s.Stats(ctx)
	return st.QuarantineOpen, err
}

// channelWatch — хранилище очереди исходящих erp, которое сообщает ops о
// смене состояния канала (эпик 30 просил ops.integration.degraded через порт
// ops: erp не эмитирует чужой тип, AD-40). Переход пишет Reporter; ошибка
// записи перехода не мешает обмену.
type channelWatch struct {
	erpapp.OutboxStore
	rep *opsapp.Reporter
	log *slog.Logger
}

func (w channelWatch) SetChannel(ctx context.Context, c erpapp.Channel) error {
	if err := w.OutboxStore.SetChannel(ctx, c); err != nil {
		return err
	}
	if _, err := w.rep.ReportIntegration(ctx, opsapp.IntegrationReport{System: c.System, State: c.State, Detail: c.Detail}); err != nil {
		w.log.Warn("ops.integration.degraded не записана", logging.KeyModule, "ops", "system", c.System, "state", c.State, "err", err)
	}
	return nil
}

// Самопроверка после старта (FR-109, AD-25).

// selfCheckState — итог самопроверки для /readyz: пока не выполнена — nil.
type selfCheckState struct {
	mu sync.RWMutex
	v  *opsapp.SelfCheckView
}

func (s *selfCheckState) set(v opsapp.SelfCheckView) {
	s.mu.Lock()
	s.v = &v
	s.mu.Unlock()
}

func (s *selfCheckState) get() *opsapp.SelfCheckView {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.v
}

// runSelfCheck — самопроверка процесса: журнал, генезис, миграции, роли БД
// и роли процесса. Генезис обязателен, когда роль init собрана (эпик 05);
// до того его отсутствие — предупреждение.
func runSelfCheck(ctx context.Context, env *environment, pool *pgxpool.Pool) opsapp.SelfCheckView {
	c, err := env.core(ctx)
	if err != nil {
		v := opsapp.SelfCheckView{Summary: dom.MsgFailed + ": ядро процесса не собрано: " + err.Error(), CheckedAt: time.Now().UTC(),
			Findings: []opsapp.SelfCheckFinding{{Check: dom.CheckJournal, Severity: string(dom.Critical), Text: "ядро процесса не собрано: " + err.Error()}}}
		env.log.Error(v.Summary, logging.KeyModule, "ops")
		return v
	}
	roles := env.selectedRoles
	if len(roles) == 0 {
		roles = []string{"api"}
	}
	sc := &opsapp.SelfCheck{
		Journal: c.journal, Database: opsDatabase{Probe: opsstore.Probe{Pool: pool}, pool: pool},
		RequireGenesis: !isPending("init"),
		Runtime:        opsRuntime{store: c.journal, leases: c.leases},
		Roles:          roles, Pending: pendingOf(roles), Idle: idleLeaders(env),
		LeaderWait: 2 * c.ttl, Now: c.codec.Now, Log: env.moduleLog("ops"),
	}
	return sc.Run(ctx)
}

// isPending — роль зарегистрирована заглушкой (pendingRole) до своего эпика.
// Реестр читается при вызове (registry): роли подменяют тела в init() своих
// файлов, а прямая ссылка на реестр из тела роли дала бы цикл инициализации.
func isPending(name string) bool { return registry()[name].pending }

var registry func() map[string]role

func init() { registry = func() map[string]role { return roles } }

// idleLeaders — роли-лидеры, которые в этой конфигурации аренду не берут:
// outbox без 1С в integrations.enabled, security без хранителя.
func idleLeaders(env *environment) []string {
	var out []string
	if !onecEnabled(env) {
		out = append(out, "outbox")
	}
	if env.cfg.Security.KeeperURL == "" {
		out = append(out, "security")
	}
	return out
}

func pendingOf(roles []string) []string {
	var out []string
	for _, r := range roles {
		if isPending(r) {
			out = append(out, r)
		}
	}
	return out
}

// readyView — ответ /readyz: итог самопроверки и ответ базы сейчас.
func readyView(sc *opsapp.SelfCheckView, dbErr error) (int, map[string]any) {
	checks := map[string]string{"selfcheck": "ok", "db": "ok"}
	body := map[string]any{"checks": checks}
	code := http.StatusOK
	switch {
	case sc == nil:
		checks["selfcheck"] = "pending"
		code = http.StatusServiceUnavailable
	case !sc.OK:
		checks["selfcheck"] = "failed"
		code = http.StatusServiceUnavailable
	}
	if sc != nil {
		body["selfcheck"] = sc
		body["summary"] = sc.Summary
	}
	if dbErr != nil {
		checks["db"] = "unavailable"
		code = http.StatusServiceUnavailable
	}
	body["status"] = "ready"
	if code != http.StatusOK {
		body["status"] = "not_ready"
	}
	return code, body
}

// moduleModesOf — режимы ведущих портов модулей API (для ops.setting.list).
func moduleModesOf(actions []platform.Action, modeFor func(string) platform.Mode) []opsapp.ModuleMode {
	seen := map[string]bool{}
	var out []opsapp.ModuleMode
	for _, a := range actions {
		if a.Owner == "" || seen[a.Owner] {
			continue
		}
		seen[a.Owner] = true
		out = append(out, opsapp.ModuleMode{Module: a.Owner, Mode: modeFor(a.Owner)})
	}
	slices.SortFunc(out, func(a, b opsapp.ModuleMode) int { return strings.Compare(a.Module, b.Module) })
	return out
}
