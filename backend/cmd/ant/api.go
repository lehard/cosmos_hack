package main

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"ant/cmd/internal/db"
	"ant/internal/application/platform"
	"ant/internal/infrastructure/fixtures/loader"
	storagefx "ant/internal/infrastructure/storage/fixtures"
	"ant/internal/infrastructure/transport/webui"
)

// runAPI — роль api: HTTP API (операции Huma всех модулей под /api/v1/),
// проверки живости и готовности, встроенный интерфейс (AD-25).
func runAPI(ctx context.Context, env *environment) error {
	cfg, log := env.cfg, env.log

	pool, err := db.Open(ctx, cfg.DB, "ant-api")
	if err != nil {
		return err
	}
	defer pool.Close()
	// Отмена раньше pool.Close: Close ждёт возврата всех соединений, в том
	// числе занятых самопроверкой.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Самопроверка после старта (FR-109): процесс жив сразу, готов — когда
	// ответила БД. Пока БД не ответила, /readyz отдаёт 503.
	var started atomic.Bool
	go func() {
		if err := db.WaitReady(ctx, pool, 2*time.Minute); err != nil {
			log.Error("самопроверка: БД не ответила", "err", err)
			return
		}
		started.Store(true)
		log.Info("самопроверка пройдена", "check", "db")
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": version})
	})
	mux.HandleFunc("GET /readyz", readyHandler(pool, &started))
	// Операции всех модулей (AD-20, AD-36); неизвестный путь /api/ — 404 problem+json.
	opts := apiOptions{mode: platform.Mode(cfg.Ports.Mode), moduleModes: moduleModes(cfg.Ports.Modules)}
	// Вход демо-персоной, сеанс и стол роли (демо-трек эпика 08).
	if opts.identity, opts.directory, err = demoIdentity(cfg); err != nil {
		return err
	}
	if modeOf(opts, "journal") == platform.ModeLive || modeOf(opts, "ingest") == platform.ModeLive {
		// Живые обновления (SSE) и журнал — на ядре процесса (эпики 04, 07);
		// приём — в журнал ядра (эпик 06). В профиле fixtures с живым приёмом
		// SSE сливает смену шага курсора и изменения движка (hybrid.go).
		if opts.journal, err = journalLive(ctx, env); err != nil {
			return err
		}
		if opts.ingest, err = ingestLive(ctx, env); err != nil {
			return err
		}
	}
	if modeOf(opts, "quality") == platform.ModeLive {
		// Операции quality — над проекциями движка (эпик 20).
		if opts.quality, err = qualityLive(ctx, env); err != nil {
			return err
		}
	}
	if modeOf(opts, "analytics") == platform.ModeLive {
		// Показатели — строки вклада движка на ядре процесса (эпик 25).
		if opts.analytics, err = analyticsLive(ctx, env); err != nil {
			return err
		}
	}
	// Курсор мира заготовок — в Postgres, общий для копий api (AD-36, эпик 09).
	// Схему создаёт migrate; EnsureSchema — для запуска без migrate (make run).
	// Без модулей на заготовках мир не строится (память, AD-25).
	if usesFixtures(opts) {
		cursor := storagefx.New(pool)
		if err := cursor.EnsureSchema(ctx); err != nil {
			log.Warn("курсор заготовок: схема не создана — жду migrate", "err", err)
		}
		if err := loader.SetCursor(cursor); err != nil {
			return err
		}
	}
	if modeOf(opts, "machinelogs") == platform.ModeLive {
		if opts.machinelogs, err = machinelogsLive(ctx, env); err != nil {
			return err
		}
	}
	if modeOf(opts, "analysis") == platform.ModeLive {
		if opts.analysis, err = analysisLive(ctx, env); err != nil {
			return err
		}
	}
	buildAPI(mux, opts)
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type":   "urn:ant:problem:api.not_found",
			"title":  "Объект не найден",
			"status": http.StatusNotFound,
			"code":   "api.not_found",
			"detail": "Операции " + r.Method + " " + r.URL.Path + " нет в контракте",
		})
	})
	mux.Handle("/", webui.Handler())

	var handler http.Handler = mux
	handler = http.MaxBytesHandler(handler, cfg.HTTP.MaxBodyBytes)
	handler = http.NewCrossOriginProtection().Handler(handler)

	srv := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           handler,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}
	errc := make(chan error, 1)
	ln, err := net.Listen("tcp", cfg.HTTP.Addr)
	if err != nil {
		return err
	}
	log.Info("HTTP слушает", "addr", ln.Addr().String())
	go func() { errc <- srv.Serve(ln) }()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	sctx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(sctx); err != nil {
		return err
	}
	if err := <-errc; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// readyHandler — готовность: самопроверка пройдена и БД отвечает сейчас.
func readyHandler(pool *pgxpool.Pool, started *atomic.Bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		checks := map[string]string{"selfcheck": "ok", "db": "ok"}
		code := http.StatusOK
		if !started.Load() {
			checks["selfcheck"] = "pending"
			code = http.StatusServiceUnavailable
		}
		ctx, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			checks["db"] = "unavailable"
			code = http.StatusServiceUnavailable
		}
		status := "ready"
		if code != http.StatusOK {
			status = "not_ready"
		}
		writeJSON(w, code, map[string]any{"status": status, "checks": checks})
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// modeOf — режим ведущих портов модуля с учётом переопределения (AD-36).
func modeOf(o apiOptions, module string) platform.Mode {
	if m, ok := o.moduleModes[module]; ok {
		return m
	}
	return o.mode
}

// usesFixtures — есть ли модули на заготовках (режим по умолчанию или переопределение).
func usesFixtures(o apiOptions) bool {
	if o.mode == platform.ModeFixtures {
		return true
	}
	for _, m := range o.moduleModes {
		if m == platform.ModeFixtures {
			return true
		}
	}
	return false
}

// moduleModes — переопределение режима ведущих портов по модулю (вертикальные
// срезы, AD-36).
func moduleModes(m map[string]string) map[string]platform.Mode {
	out := make(map[string]platform.Mode, len(m))
	for k, v := range m {
		out[k] = platform.Mode(v)
	}
	return out
}
