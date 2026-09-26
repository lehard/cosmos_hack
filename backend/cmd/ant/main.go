// Команда ant — единый бинарник модульного монолита (AD-25).
//
// Слой: точка входа и сборка зависимостей (cmd/*) — единственное место, где
// зоны инфраструктуры соединяются с приложением (AD-1).
//
// Роль процесса задаётся флагом -role (несколько — через запятую). Все роли
// спайна зарегистрированы заранее (эпик 02). Работают: api, migrate, worker,
// crossitem, projector, rebuild (-item ‹id› — одно изделие), stands (каркас
// эпика 06, stands.go); остальные — заглушки до своих эпиков (scheduler — 24;
// outbox — 30; init — 05).
// Роли одного процесса делят ядро (пул ant_app, журнал, LISTEN) — core.go.
//
// Флаг -openapi ‹файл› — выгрузить спецификацию HTTP API (contracts/openapi.yaml)
// из операций Huma и выйти (make generate, AD-20); БД и конфигурация не нужны.
//
// Флаг -healthcheck — проверка живости для HEALTHCHECK контейнера: в образе
// нет оболочки и curl, поэтому проверку делает сам бинарник.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"

	"ant/cmd/internal/config"
	"ant/internal/infrastructure/observability/logging"
)

// version проставляется при сборке: -ldflags "-X main.version=…".
var version = "dev"

// roleFunc — тело роли; возвращается при остановке или ошибке.
type roleFunc func(ctx context.Context, env *environment) error

// role — описание роли процесса.
type role struct {
	run roleFunc
	// oneShot — роль выполняется и завершается (migrate, init, rebuild);
	// такие роли не совмещаются с долгоживущими в одном процессе.
	oneShot bool
}

// roles — реестр ролей процесса.
var roles = map[string]role{
	"api":       {run: runAPI},
	"migrate":   {run: runMigrate, oneShot: true},
	"worker":    {run: runWorker},
	"crossitem": {run: runCrossItem},
	"projector": {run: runProjector},
	"scheduler": pendingRole("scheduler", "эпик 24: сроки и «наступил срок»", false),
	"outbox":    pendingRole("outbox", "эпик 30: исходящие сообщения и квитанции", false),
	"stands":    pendingRole("stands", "эпики 06, 30–33: stand-ы внешних систем и прогоны", false),
	"init":      pendingRole("init", "эпик 05: ключи, миграции, генезис", true),
	"rebuild":   {run: runRebuild, oneShot: true},
	"security":  {run: runSecurity}, // эпик 29: хранитель, отчёты верификатора, шина безопасности
}

// environment — то, что роль получает от точки входа.
type environment struct {
	cfg *config.Config
	log *slog.Logger
	// item, reason — `ant rebuild -item ‹id› [-reason ‹текст›]`.
	item   string
	reason string
	// coreH — ядро процесса, общее для ролей (core.go).
	coreH coreHolder
	// ctx — общий контекст ролей процесса (runRoles): фоновые циклы ядра.
	ctx context.Context
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("ant", flag.ContinueOnError)
	fs.SetOutput(stderr)
	roleFlag := fs.String("role", "api", "роли процесса через запятую: "+strings.Join(roleNames(), ", "))
	cfgPath := fs.String("config", envOr("ANT_CONFIG", "/etc/ant/ant.yaml"), "путь к ant.yaml (или ANT_CONFIG)")
	healthcheck := fs.Bool("healthcheck", false, "проверить /healthz запущенного процесса и выйти (для HEALTHCHECK)")
	showVersion := fs.Bool("version", false, "показать версию и выйти")
	openapiOut := fs.String("openapi", "", "выгрузить спецификацию HTTP API в файл и выйти (make generate)")
	item := fs.String("item", "", "роль rebuild: только это изделие (повтор после «обработка остановлена» или пересборка его проекций)")
	reason := fs.String("reason", "", "роль rebuild -item: причина повтора обработки (в журнал)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *showVersion {
		_, _ = fmt.Fprintln(stdout, version)
		return 0
	}
	if *openapiOut != "" {
		if err := dumpOpenAPI(*openapiOut); err != nil {
			_, _ = fmt.Fprintln(stderr, "openapi:", err)
			return 1
		}
		return 0
	}

	cfg, err := config.Load(*cfgPath, os.LookupEnv)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	if *healthcheck {
		return probe(cfg.HTTP.Addr, stderr)
	}

	selected, err := parseRoles(*roleFlag)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	log := logging.New(stdout, cfg.Log.Level, "ant").With(logging.KeyRole, strings.Join(selected, ","))
	log.Info("старт", "version", version, "profile", cfg.Profile, "ports_mode", cfg.Ports.Mode)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if *item != "" && !slices.Equal(selected, []string{"rebuild"}) {
		_, _ = fmt.Fprintln(stderr, "-item — только для -role=rebuild")
		return 2
	}
	env := &environment{cfg: cfg, log: log, item: *item, reason: *reason}
	err = runRoles(ctx, selected, env)
	env.closeCore()
	if err != nil {
		log.Error("остановка с ошибкой", "err", err)
		return 1
	}
	log.Info("остановка")
	return 0
}

func roleNames() []string {
	names := make([]string, 0, len(roles))
	for n := range roles {
		names = append(names, n)
	}
	slices.Sort(names)
	return names
}

func parseRoles(s string) ([]string, error) {
	var out []string
	oneShot := false
	for p := range strings.SplitSeq(s, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		r, ok := roles[p]
		if !ok {
			return nil, fmt.Errorf("неизвестная роль %q; доступны: %s", p, strings.Join(roleNames(), ", "))
		}
		if !slices.Contains(out, p) {
			out = append(out, p)
		}
		oneShot = oneShot || r.oneShot
	}
	if len(out) == 0 {
		return nil, errors.New("не задана роль: -role")
	}
	if oneShot && len(out) > 1 {
		return nil, fmt.Errorf("разовые роли выполняются отдельным процессом: %s", strings.Join(out, ","))
	}
	return out, nil
}

// runRoles запускает роли параллельно; первая ошибка останавливает остальные.
func runRoles(ctx context.Context, names []string, env *environment) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	env.ctx = ctx
	var (
		wg   sync.WaitGroup
		once sync.Once
		fail error
	)
	for _, n := range names {
		wg.Go(func() {
			if err := roles[n].run(ctx, env); err != nil {
				once.Do(func() { fail = fmt.Errorf("роль %s: %w", n, err); cancel() })
			}
		})
	}
	wg.Wait()
	return fail
}

// probe обращается к /healthz процесса на этой же машине.
func probe(addr string, stderr io.Writer) int {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "http.addr:", err)
		return 1
	}
	c := http.Client{Timeout: 2 * time.Second}
	resp, err := c.Get("http://" + net.JoinHostPort("127.0.0.1", port) + "/healthz")
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = fmt.Fprintln(stderr, "healthz:", resp.Status)
		return 1
	}
	return 0
}

func envOr(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}
