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
	buildAPI(mux, apiOptions{mode: platform.Mode(cfg.Ports.Mode), moduleModes: moduleModes(cfg.Ports.Modules)})
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

// moduleModes — переопределение режима ведущих портов по модулю (вертикальные
// срезы, AD-36).
func moduleModes(m map[string]string) map[string]platform.Mode {
	out := make(map[string]platform.Mode, len(m))
	for k, v := range m {
		out[k] = platform.Mode(v)
	}
	return out
}
