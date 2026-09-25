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
	"ant/internal/infrastructure/transport/webui"
)

// runAPI — роль api: HTTP API, проверки живости и готовности, встроенный
// интерфейс (AD-25). Операции Huma подключает эпик 02 под префиксом /api/.
func runAPI(ctx context.Context, env *environment) error {
	cfg, log := env.cfg, env.log

	pool, err := db.Open(ctx, cfg.DB, "ant-api")
	if err != nil {
		return err
	}
	defer pool.Close()

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
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		// Заглушка до эпика 02: ответ в формате RFC 9457.
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type":   "about:blank",
			"title":  "Операция ещё не реализована",
			"status": http.StatusNotImplemented,
			"detail": r.Method + " " + r.URL.Path,
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
	go func() {
		log.Info("HTTP слушает", "addr", cfg.HTTP.Addr)
		errc <- srv.ListenAndServe()
	}()

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
