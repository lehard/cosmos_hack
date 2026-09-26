// Команда edge-agent — агент на краю (кейс §3.2 «Edge-агенты и защищённый
// обмен», FR-39, AD-7, AD-18): ключ устройства, source_seq, буфер исходных
// подписанных конвертов на диске, досылка пачками после недоступности ядра;
// локальный вход для источников, stand-ов оборудования и систем видеофиксации
// (VisionQC, OperatorVision: адаптеры на краю, vision.go, эпик 33).
//
// Слой: точка входа отдельного процесса (граница доверия — устройство, AD-25).
//
// Подкоманды:
//
//	edge-agent run     — работа: локальный вход -listen, досылка в -core;
//	edge-agent keygen  — создать ключ устройства ГОСТ Р 34.10-2012 и показать открытый ключ для акта регистрации;
//	edge-agent send    — положить события из файла JSONL в буфер и дослать;
//	edge-agent demo    — самопоказ на ядре в памяти: пять случаев FR-29, повтор, конфликт, недоступность и восстановление.
package main

import (
	"bufio"
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
	"syscall"
	"time"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "edge-agent run | keygen | send | demo  (-h — флаги)")
		return 2
	}
	cmd, args := args[0], args[1:]
	fs := flag.NewFlagSet("edge-agent "+cmd, flag.ContinueOnError)
	fs.SetOutput(stderr)
	source := fs.String("source-id", env("EDGE_SOURCE_ID", ""), "source_id устройства (EDGE_SOURCE_ID)")
	core := fs.String("core", env("EDGE_CORE_URL", "http://127.0.0.1:8080"), "адрес ядра ant (EDGE_CORE_URL)")
	state := fs.String("state", env("EDGE_STATE_DIR", "/var/lib/edge-agent"), "каталог состояния: номер и буфер (EDGE_STATE_DIR)")
	keyFile := fs.String("key", env("EDGE_KEY_FILE", ""), "файл ключа устройства; пусто — без подписи (профиль demo)")
	keyRef := fs.String("key-ref", env("EDGE_KEY_REF", ""), "key_id@версия ключа; пусто — device-‹source-id›@1")
	listen := fs.String("listen", env("EDGE_LISTEN", "127.0.0.1:8490"), "локальный вход для источников и stand-ов")
	batch := fs.Int("batch", 100, "сообщений в пачке")
	interval := fs.Duration("interval", time.Second, "период досылки")
	kind := fs.String("source-kind", "machine", "source_kind по умолчанию (FR-140)")
	examples := fs.String("examples", "", "каталог примеров contracts/events/examples/contract-change (demo)")
	illustrations := fs.String("illustrations", env("EDGE_ILLUSTRATIONS_DIR", ""), "каталог распакованного набора иллюстраций TIG Aluminium 5083 (EDGE_ILLUSTRATIONS_DIR); пусто — иллюстрации не прикладываются")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	log := slog.New(slog.NewJSONHandler(stderr, nil)).With("module", "edge-agent", "source_id", *source)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cmd == "demo" {
		dir, err := os.MkdirTemp("", "edge-agent-demo-")
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return 1
		}
		defer func() { _ = os.RemoveAll(dir) }()
		if _, err := runDemo(ctx, stdout, *examples, dir); err != nil {
			_, _ = fmt.Fprintln(stderr, "demo:", err)
			return 1
		}
		return 0
	}
	if cmd == "keygen" {
		if *keyFile == "" {
			_, _ = fmt.Fprintln(stderr, "keygen: нужен -key ‹файл›")
			return 2
		}
		pub, err := GenGostKey(*keyFile)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "keygen:", err)
			return 1
		}
		b64, fp := PublicKeyB64(pub)
		_ = json.NewEncoder(stdout).Encode(map[string]string{"source_id": *source, "key_ref": ref(*keyRef, *source),
			"profile": "gost", "public_key_b64": b64, "fingerprint": fp})
		return 0
	}
	var signer Signer = Unsigned{Ref: ref(*keyRef, *source)}
	if *keyFile != "" {
		g, err := LoadGostKey(*keyFile, ref(*keyRef, *source))
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return 1
		}
		signer = g
	}
	a, err := NewAgent(Config{SourceID: *source, CoreURL: *core, StateDir: *state, BatchSize: *batch, FlushInterval: *interval,
		SourceKind: *kind, Reliability: "high"}, signer, nil, nil, log)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	switch cmd {
	case "send":
		n := 0
		for _, f := range fs.Args() {
			if err := sendFile(a, f, &n); err != nil {
				_, _ = fmt.Fprintln(stderr, err)
				return 1
			}
		}
		sent, err := a.Drain(ctx)
		_ = json.NewEncoder(stdout).Encode(map[string]any{"enqueued": n, "sent": sent, "status": a.Status(), "error": errString(err)})
		if err != nil {
			return 1
		}
		return 0
	case "run":
		vision, err := VisionRoutes(*core, *illustrations, nil)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return 1
		}
		srv := &http.Server{Addr: *listen, Handler: LocalHandler(a, NewExtractor(), vision...), ReadHeaderTimeout: 10 * time.Second}
		go func() {
			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Error("локальный вход", "err", err)
				stop()
			}
		}()
		log.Info("старт", "core", *core, "listen", *listen, "signed", signer.Signs(), "next_seq", a.Status().NextSeq)
		err = a.Run(ctx)
		sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(sctx)
		if err != nil {
			log.Error("остановка", "err", err)
			return 1
		}
		return 0
	}
	_, _ = fmt.Fprintln(stderr, "неизвестная подкоманда:", cmd)
	return 2
}

func sendFile(a *Agent, path string, n *int) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 16<<20)
	for sc.Scan() {
		if len(sc.Bytes()) == 0 {
			continue
		}
		var ev map[string]any
		if err := json.Unmarshal(sc.Bytes(), &ev); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if _, err := a.Enqueue(ev); err != nil {
			return err
		}
		*n++
	}
	return sc.Err()
}

func ref(r, source string) string {
	if r != "" {
		return r
	}
	return "device-" + source + "@1"
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
