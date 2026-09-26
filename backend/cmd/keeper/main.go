// Команда keeper — хранитель контрольных точек (AD-8, AD-46, FR-72): отдельный
// процесс по границе доверия со своим томом и mTLS. Принимает от ant головы
// обеих цепочек журнала со звеньями и выдаёт подписанные контрольные точки
// профиля hybrid (ГОСТ Р 34.10-2012 + ML-DSA-65); сам поднимает тревогу, если
// головы не приходили дольше 2N секунд; хранит подписанные отчёты верификатора
// — первичный канал вердикта (Аудитор ИБ и ВП получают его без ant).
//
// Протокол — contracts/internal/keeper.openapi.yaml. В демо — контейнер
// keeper на том же хосте; в промышленной эксплуатации — хост службы ИБ или ВП
// в другом административном домене.
//
// Режимы:
//
//	keeper -init …   — разовая подготовка доверия (до эпика 05 «ant init»):
//	                   ключи хранителя и верификатора, trust-anchors, демо-УЦ
//	                   mTLS и сертификаты участников, KEK шифрования при хранении;
//	keeper           — служба (https :8444, mTLS; живость — http 127.0.0.1:8445);
//	keeper -healthcheck — проверка живости для HEALTHCHECK контейнера.
//
// N (интервал голов) и предельный разрыв — policy.audit.*: до записи
// policy.audit.parameters_set, подписанной Аудитором ИБ (эпик 26), — флаги
// -interval (ANT_KEEPER_INTERVAL) и конфигурация стенда. TODO(26).
//
// Слой: точка входа (cmd/*). Владелец: эпик 29 (доверие).
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	app "ant/internal/application/security"
	"ant/internal/infrastructure/observability/logging"
	"ant/internal/infrastructure/security/atrest"
	"ant/internal/infrastructure/security/hybrid"
	"ant/internal/infrastructure/security/mtls"
)

var version = "dev"

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("keeper", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dataDir := fs.String("data", envOr("ANT_KEEPER_DATA", "/var/lib/keeper"), "том хранителя: ключи, trust-anchors, УЦ, контрольные точки")
	addr := fs.String("addr", envOr("ANT_KEEPER_ADDR", ":8444"), "адрес mTLS")
	healthAddr := fs.String("health-addr", envOr("ANT_KEEPER_HEALTH_ADDR", "127.0.0.1:8445"), "адрес живости (http, только внутри контейнера)")
	interval := fs.Duration("interval", durEnv("ANT_KEEPER_INTERVAL", 10*time.Second), "N — интервал голов (policy.audit.checkpoint_interval; TODO(26): из журнала)")
	initMode := fs.Bool("init", false, "подготовить доверие и выйти")
	verifierDir := fs.String("init-verifier", "", "-init: том верификатора (ключи, сертификат, копия trust-anchors)")
	antDir := fs.String("init-ant", "", "-init: каталог сертификата mTLS ant")
	tamperDir := fs.String("init-tamper", "", "-init: каталог сертификата демо-инструмента (профили fixtures, demo)")
	kekFile := fs.String("init-kek", "", "-init: файл KEK шифрования при хранении (AD-23)")
	hosts := fs.String("hosts", envOr("ANT_KEEPER_HOSTS", "keeper,localhost"), "-init: имена хранителя в сертификате")
	health := fs.Bool("healthcheck", false, "проверить живость и выйти")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	log := logging.New(stdout, "info", "keeper")
	if *health {
		c := http.Client{Timeout: 2 * time.Second}
		resp, err := c.Get("http://" + *healthAddr + "/healthz")
		if err != nil || resp.StatusCode != http.StatusOK {
			_, _ = fmt.Fprintln(stderr, "keeper: не жив", err)
			return 1
		}
		_ = resp.Body.Close()
		return 0
	}
	if *initMode {
		if err := initTrust(*dataDir, *verifierDir, *antDir, *tamperDir, *kekFile, strings.Split(*hosts, ",")); err != nil {
			_, _ = fmt.Fprintln(stderr, "keeper -init:", err)
			return 1
		}
		_, _ = fmt.Fprintln(stdout, "keeper -init: доверие подготовлено в", *dataDir)
		return 0
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := serve(ctx, log, *dataDir, *addr, *healthAddr, *interval); err != nil {
		log.Error("остановка с ошибкой", "err", err)
		return 1
	}
	return 0
}

func durEnv(k string, d time.Duration) time.Duration {
	if v, err := time.ParseDuration(os.Getenv(k)); err == nil && v > 0 {
		return v
	}
	return d
}

// AnchorsFile — имя файла trust-anchors (AD-33).
const AnchorsFile = "trust-anchors.json"

// initTrust — разовая подготовка доверия (до эпика 05: ant init и генезис
// регистрируют ключи хранителя и верификатора в журнале). Повтор ничего не
// перезаписывает.
func initTrust(data, verifierDir, antDir, tamperDir, kekFile string, hosts []string) error {
	keeperKeys, err := hybrid.Generate(filepath.Join(data, "keys"), "keeper")
	if err != nil {
		return err
	}
	a := hybrid.Anchors{FormatVersion: 1, Keys: keeperKeys,
		Profiles: map[string]string{"checkpoint": "hybrid", "verifier-report": "hybrid", "genesis": "hybrid", "key-act": "hybrid", "event": "gost"},
		Note:     "Демо: ключи создал keeper -init одного хоста; в промышленной эксплуатации файл подписан Аудитором ИБ и передаётся на отчуждаемом носителе (AD-33)."}
	clients := []string{}
	if verifierDir != "" {
		vk, err := hybrid.Generate(filepath.Join(verifierDir, "keys"), "verifier")
		if err != nil {
			return err
		}
		a.Keys = append(a.Keys, vk...)
		clients = append(clients, "verifier")
	}
	if antDir != "" {
		clients = append(clients, "ant")
	}
	if tamperDir != "" {
		clients = append(clients, "tamper")
	}
	pki := filepath.Join(data, "pki")
	if err := mtls.Init(pki, hosts, clients); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(a, "", "  ")
	anchors := filepath.Join(data, AnchorsFile)
	if _, err := os.Stat(anchors); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(anchors, b, 0o444); err != nil {
			return err
		}
	} else if b, err = os.ReadFile(anchors); err != nil {
		return err
	}
	copyTo := func(dir, name string) error {
		if dir == "" {
			return nil
		}
		if err := os.MkdirAll(filepath.Join(dir, "pki"), 0o700); err != nil {
			return err
		}
		for _, f := range []string{"ca.crt", name + ".crt", name + ".key"} {
			dst := filepath.Join(dir, "pki", f)
			if _, err := os.Stat(dst); err == nil {
				continue
			}
			src, err := os.ReadFile(filepath.Join(pki, f))
			if err != nil {
				return fmt.Errorf("%s: %w (повторная подготовка без тома участника — удалите тома доверия и повторите)", f, err)
			}
			mode := os.FileMode(0o444)
			if strings.HasSuffix(f, ".key") {
				mode = 0o400
			}
			if err := os.WriteFile(dst, src, mode); err != nil {
				return err
			}
		}
		return nil
	}
	if err := copyTo(verifierDir, "verifier"); err != nil {
		return err
	}
	if verifierDir != "" {
		dst := filepath.Join(verifierDir, AnchorsFile)
		if _, err := os.Stat(dst); errors.Is(err, os.ErrNotExist) {
			if err := os.WriteFile(dst, b, 0o444); err != nil {
				return err
			}
		}
	}
	if err := copyTo(antDir, "ant"); err != nil {
		return err
	}
	if err := copyTo(tamperDir, "tamper"); err != nil {
		return err
	}
	if kekFile != "" {
		if _, err := atrest.Generate(kekFile); err != nil {
			return err
		}
	}
	// Закрытые ключи участников не остаются в томе хранителя: только у
	// владельцев (ant, verifier, tamper) — сертификаты и УЦ остаются.
	for _, c := range clients {
		_ = os.Remove(filepath.Join(pki, c+".key"))
	}
	return nil
}

func serve(ctx context.Context, log *slog.Logger, data, addr, healthAddr string, interval time.Duration) error {
	signer, err := hybrid.Load(filepath.Join(data, "keys"), "keeper")
	if err != nil {
		return fmt.Errorf("ключи хранителя (сначала keeper -init): %w", err)
	}
	anchors, _, err := hybrid.LoadAnchors(filepath.Join(data, AnchorsFile))
	if err != nil {
		return err
	}
	st, err := OpenStore(data, signer, anchors, interval)
	if err != nil {
		return err
	}
	tlsCfg, err := mtls.Server(mtls.Files{Dir: filepath.Join(data, "pki"), Name: "keeper"})
	if err != nil {
		return err
	}
	srv := &http.Server{Addr: addr, Handler: Handler(st, log), TLSConfig: tlsCfg, ReadHeaderTimeout: 5 * time.Second,
		BaseContext: func(net.Listener) context.Context { return ctx }}
	hmux := http.NewServeMux()
	hmux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	hsrv := &http.Server{Addr: healthAddr, Handler: hmux, ReadHeaderTimeout: 2 * time.Second}
	go func() { _ = hsrv.ListenAndServe() }()
	go func() {
		t := time.NewTicker(max(interval/2, time.Second))
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				before := len(st.Status().Alarms)
				st.Watch()
				if s := st.Status(); len(s.Alarms) > before {
					log.Warn("тревога хранителя", "alert", s.Alarms[len(s.Alarms)-1].Alert, "detail", s.Alarms[len(s.Alarms)-1].Detail)
				}
			}
		}
	}()
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServeTLS("", "") }()
	log.Info("хранитель слушает (mTLS)", "addr", addr, "interval", interval.String(), "checkpoints", st.Status().Checkpoints, "version", version)
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = hsrv.Shutdown(sctx)
	return srv.Shutdown(sctx)
}

// Handler — операции keeper.openapi.yaml.
func Handler(st *Store, log *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /v1/heads", func(w http.ResponseWriter, r *http.Request) {
		var sub app.HeadsSubmission
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<20)).Decode(&sub); err != nil {
			problem(w, http.StatusBadRequest, "invalid", err.Error())
			return
		}
		cp, err := st.Submit(sub)
		var rej *Reject
		switch {
		case errors.As(err, &rej):
			log.Warn("головы отвергнуты", "kind", rej.Kind.Error(), "detail", rej.Detail, "client", mtls.ClientName(r.TLS))
			code := http.StatusConflict
			if errors.Is(err, ErrInvalid) {
				code = http.StatusBadRequest
			}
			problem(w, code, rej.Kind.Error(), rej.Detail)
		case err != nil:
			problem(w, http.StatusInternalServerError, "internal", err.Error())
		default:
			raw(w, http.StatusOK, cp.Envelope)
		}
	})
	mux.HandleFunc("GET /v1/checkpoints/latest", func(w http.ResponseWriter, _ *http.Request) {
		st.mu.Lock()
		cp, ok := st.last()
		st.mu.Unlock()
		if !ok {
			problem(w, http.StatusNotFound, "not_found", "контрольных точек ещё нет")
			return
		}
		raw(w, http.StatusOK, cp.Envelope)
	})
	mux.HandleFunc("GET /v1/checkpoints", func(w http.ResponseWriter, r *http.Request) {
		after, _ := strconv.ParseInt(r.URL.Query().Get("after_no"), 10, 64)
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit <= 0 || limit > 1000 {
			limit = 1000
		}
		st.mu.Lock()
		out := []json.RawMessage{}
		for _, c := range st.checkpoints {
			if int64(c.Payload.CheckpointNo) > after && len(out) < limit {
				out = append(out, c.Envelope)
			}
		}
		st.mu.Unlock()
		writeJSON(w, http.StatusOK, out)
	})
	mux.HandleFunc("POST /v1/verifier-reports", func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 16<<20))
		if err != nil {
			problem(w, http.StatusBadRequest, "invalid", err.Error())
			return
		}
		rp, err := st.SubmitReport(b)
		if err != nil {
			log.Warn("отчёт верификатора отвергнут", "err", err, "client", mtls.ClientName(r.TLS))
			problem(w, http.StatusUnprocessableEntity, "report_rejected", err.Error())
			return
		}
		log.Info("отчёт верификатора принят", "digest", rp.Digest, "verdict", rp.Payload.Verdict)
		writeJSON(w, http.StatusCreated, map[string]string{"report_digest": rp.Digest})
	})
	mux.HandleFunc("GET /v1/verifier-reports/latest", func(w http.ResponseWriter, _ *http.Request) {
		st.mu.Lock()
		defer st.mu.Unlock()
		if len(st.reports) == 0 {
			problem(w, http.StatusNotFound, "not_found", "отчётов ещё нет")
			return
		}
		raw(w, http.StatusOK, st.reports[len(st.reports)-1].Envelope)
	})
	mux.HandleFunc("GET /v1/verifier-reports", func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit <= 0 || limit > 1000 {
			limit = 100
		}
		st.mu.Lock()
		out := []json.RawMessage{}
		for i := len(st.reports) - 1; i >= 0 && len(out) < limit; i-- {
			out = append(out, st.reports[i].Envelope)
		}
		st.mu.Unlock()
		writeJSON(w, http.StatusOK, out)
	})
	mux.HandleFunc("GET /v1/verifier-reports/{digest}", func(w http.ResponseWriter, r *http.Request) {
		d := r.PathValue("digest")
		st.mu.Lock()
		defer st.mu.Unlock()
		for _, rp := range st.reports {
			if rp.Digest == d {
				raw(w, http.StatusOK, rp.Envelope)
				return
			}
		}
		problem(w, http.StatusNotFound, "not_found", "отчёта "+d+" нет")
	})
	mux.HandleFunc("GET /v1/links", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		after, _ := strconv.ParseInt(q.Get("after_seq"), 10, 64)
		limit, _ := strconv.Atoi(q.Get("limit"))
		if limit <= 0 || limit > 10000 {
			limit = 10000
		}
		writeJSON(w, http.StatusOK, st.Links(q.Get("chain"), after, limit))
	})
	mux.HandleFunc("GET /v1/status", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, http.StatusOK, st.Status()) })
	return mux
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	b, _ := json.Marshal(v)
	raw(w, code, b)
}

func raw(w http.ResponseWriter, code int, b []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write(b)
}

func problem(w http.ResponseWriter, code int, c, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]any{"type": "urn:ant:problem:keeper." + c, "title": "Хранитель: " + c, "status": code, "code": c, "detail": detail})
}
