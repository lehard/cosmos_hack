// Команда demo-signer — демо-персоны сценариев (AD-26, AD-33, AD-46; FR-66,
// FR-69, FR-129): код агента токена без интерфейса, вне ant, ОБЫЧНЫЙ клиент.
// Входит в ant через IdentityProvider под демо-персоной (POST /api/v1/auth/session),
// вызывает те же операции API из openapi.yaml, сам считает отпечаток и
// подписывает ключом класса scenario только запросы, совпадающие с шагом из
// scenarios/definitions (том только для чтения). Служебных обходов нет: права,
// гарды и проверка подписи — те же, что для человека с агентом токена.
//
// Слой: точка входа отдельного процесса (граница доверия — демо-персоны, AD-25).
// Контракт — contracts/internal/demo-signer.openapi.yaml. Есть только в
// профилях fixtures, demo, load; профиль prod с ним не стартует (AD-26).
//
// Подкоманды:
//
//	demo-signer init   — проверка, что ключи демо-персон есть в -keys: их
//	                     создаёт (0400) и регистрирует блоком генезиса
//	                     ant init (эпик 05, AD-33);
//	demo-signer serve  — API шагов: POST /v1/steps, GET /v1/personas, GET /healthz;
//	demo-signer step   — один шаг из командной строки (отладка, make-цели).
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "demo-signer init | serve | step  (-h — флаги)")
		return 2
	}
	cmd, args := args[0], args[1:]
	fs := flag.NewFlagSet("demo-signer "+cmd, flag.ContinueOnError)
	fs.SetOutput(stderr)
	profile := fs.String("profile", env("ANT_PROFILE", "demo"), "профиль окружения: fixtures | demo | load (prod — отказ, AD-26)")
	keys := fs.String("keys", env("DEMO_SIGNER_KEYS", "/var/lib/demo-signer/keys"), "том ключей демо-персон (0600)")
	policy := fs.String("policy", env("DEMO_SIGNER_POLICY", "/normative/policy/policy.v1.yaml"), "стартовая политика: перечень персон")
	scen := fs.String("scenarios", env("DEMO_SIGNER_SCENARIOS", "/scenarios"), "каталог scenarios (только чтение)")
	ant := fs.String("ant", env("DEMO_SIGNER_ANT", "http://ant:8080"), "адрес ant")
	listen := fs.String("listen", env("DEMO_SIGNER_LISTEN", ":8445"), "адрес API шагов")
	stepJSON := fs.String("request", "", "step: StepRequest в JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	log := slog.New(slog.NewJSONHandler(stderr, nil)).With("module", "demo-signer")
	if *profile == "prod" {
		log.Error("demo-signer не запускается в профиле prod (AD-26)")
		return 1
	}
	switch cmd {
	case "init":
		ps, err := Personas(*policy)
		if err != nil {
			log.Error("персоны", "err", err)
			return 1
		}
		if _, err := LoadKeys(*keys, ps); err != nil {
			log.Error("ключи", "err", err)
			return 1
		}
		_, _ = fmt.Fprintf(stdout, "demo-signer: ключи %d персон в %s зарегистрированы генезисом (ant init)\n", len(ps), *keys)
		return 0
	case "serve", "step":
	default:
		_, _ = fmt.Fprintln(stderr, "неизвестная подкоманда: "+cmd)
		return 2
	}
	ps, err := Personas(*policy)
	if err != nil {
		log.Error("персоны", "err", err)
		return 1
	}
	set, err := LoadKeys(*keys, ps)
	if err != nil {
		log.Error("ключи", "err", err)
		return 1
	}
	s := &Signer{Keys: set, Steps: Steps{Root: *scen}, Ant: &Client{Base: strings.TrimRight(*ant, "/"), HTTP: &http.Client{Timeout: 30 * time.Second}}}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if cmd == "step" {
		var rq StepRequest
		if err := json.Unmarshal([]byte(*stepJSON), &rq); err != nil {
			log.Error("-request", "err", err)
			return 2
		}
		res, err := s.Execute(ctx, rq)
		b, _ := json.MarshalIndent(res, "", "  ")
		_, _ = fmt.Fprintln(stdout, string(b))
		if err != nil {
			log.Error("шаг", "err", err)
			return 1
		}
		return 0
	}
	srv := &http.Server{Addr: *listen, Handler: s.Handler(), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		sc, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(sc)
	}()
	log.Info("demo-signer слушает", "addr", *listen, "personas", len(ps))
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("сервер", "err", err)
		return 1
	}
	return 0
}
