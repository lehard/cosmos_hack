package main

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"ant/cmd/internal/db"
	opsapp "ant/internal/application/ops"
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

	// Самопроверка после старта (FR-109, эпик 34): процесс жив сразу; готов —
	// когда ответила БД и самопроверка (журнал, генезис, миграции, роли) не
	// нашла критических ошибок. До итога и при критических находках /readyz — 503.
	var selfcheck selfCheckState
	var opsSvc *opsapp.Service
	opsReady := make(chan struct{})
	go func() {
		if err := db.WaitReady(ctx, pool, 2*time.Minute); err != nil {
			log.Error("самопроверка: БД не ответила", "err", err)
			return
		}
		select {
		case <-opsReady:
		case <-ctx.Done():
			return
		}
		v := runSelfCheck(ctx, env, pool)
		selfcheck.set(v)
		if opsSvc != nil {
			opsSvc.SetSelfCheck(v)
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": version})
	})
	mux.HandleFunc("GET /readyz", readyHandler(pool, &selfcheck))
	// Операции всех модулей (AD-20, AD-36); неизвестный путь /api/ — 404 problem+json.
	opts := apiOptions{mode: platform.Mode(cfg.Ports.Mode), moduleModes: moduleModes(cfg.Ports.Modules)}
	// Вход, сеансы и права (эпик 08): порт входа, проекция политики, Casbin.
	if opts.access, err = accessLive(ctx, env); err != nil {
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
	if modeOf(opts, "item") == platform.ModeLive {
		// Паспорт, носители, генеалогия (эпик 18).
		if opts.item, err = itemLive(ctx, env); err != nil {
			return err
		}
	}
	if modeOf(opts, "crossitem") == platform.ModeLive {
		// Партии, садки, привязка событий без изделия (эпик 18).
		if opts.crossitem, err = crossitemLive(ctx, env); err != nil {
			return err
		}
	}
	if modeOf(opts, "erp") == platform.ModeLive {
		// Эпик 30: операции erp.* над проекциями erp.* и каналами обмена.
		if opts.erp, err = erpLive(ctx, env); err != nil {
			return err
		}
	}
	if modeOf(opts, "mes") == platform.ModeLive {
		// Эпик 31: блоки и задания MES — проекции mes.*.
		if opts.mes, err = mesLive(ctx, env); err != nil {
			return err
		}
	}
	if modeOf(opts, "cad") == platform.ModeLive {
		// Эпик 31: импорт условной сборки КОМПАС через приём, проекция cad.assembly.
		if opts.cad, err = cadLive(ctx, env, opts.ingest); err != nil {
			return err
		}
	}
	if modeOf(opts, "analysis") == platform.ModeLive {
		if opts.analysis, err = analysisLive(ctx, env); err != nil {
			return err
		}
		// Эпик 42: вход генератора «ограничение линии» — счётчики узлов analytics.
		if opts.analytics != nil {
			opts.analysis.ConnectLine(analyticsLine{opts.analytics})
		}
	}
	if modeOf(opts, "nonconformity") == platform.ModeLive {
		if opts.nonconformity, err = nonconformityLive(ctx, env, opts.accessDirectory(), opts.policyAuthorities); err != nil {
			return err
		}
	}
	if modeOf(opts, "documents") == platform.ModeLive {
		// Документы-проекции журнала и маршруты подписей (эпик 28).
		if opts.documents, err = documentsLive(ctx, env, opts.access.stampRegistry()); err != nil {
			return err
		}
	}
	if modeOf(opts, "signing") == platform.ModeLive || modeOf(opts, "nonconformity") == platform.ModeLive {
		// Подписи и ключи (эпик 27) и проверка подписи команд уровня ≥ 1 в
		// декораторе (пачка стыков Д-59).
		if opts.signing, err = signingLive(ctx, env, opts.access); err != nil {
			return err
		}
	}
	if modeOf(opts, "vision") == platform.ModeLive {
		// Паспорта допуска анализаторов — над журналом ядра (эпик 33).
		if opts.vision, err = visionLive(ctx, env); err != nil {
			return err
		}
	}
	if modeOf(opts, "process") == platform.ModeLive {
		// Живая карта, версии и команды исполнителя (эпик 17).
		if opts.process, err = processLive(ctx, env, opts.ingest, opts.analytics, opts.documents); err != nil {
			return err
		}
	}
	if modeOf(opts, "reference") == platform.ModeLive {
		// Справочники, календарь и смены из журнала ядра (эпик 19).
		if opts.reference, err = referenceLive(ctx, env); err != nil {
			return err
		}
	}
	if modeOf(opts, "notifications") == platform.ModeLive {
		// Сроки, задачи, тревоги — проекции notifications.* (эпик 24).
		if opts.notifications, err = notificationsLive(ctx, env); err != nil {
			return err
		}
	}
	if modeOf(opts, "security") == platform.ModeLive {
		// Журнал CA, шина безопасности, индикатор целостности (эпик 29).
		if opts.security, err = securityLive(ctx, env); err != nil {
			return err
		}
	}
	var simProxy *portProxy
	if modeOf(opts, "simulation") == platform.ModeLive {
		// Пульт тестовых сценариев (эпики 32, 16): раннер — в роли stands процесса.
		if opts.simulation, simProxy, err = simulationLive(ctx, env, opts.ingest); err != nil {
			return err
		}
	}
	if modeOf(opts, "ops") == platform.ModeLive {
		// Эпик 34: состояние компонентов, остановленные изделия, настройки.
		if opts.ops, err = opsLive(ctx, env, pool, opts.ingest); err != nil {
			close(opsReady)
			return err
		}
		opsSvc = opts.ops
	}
	close(opsReady)
	api := buildAPI(mux, opts)
	// Метрики процесса (FR-41, FR-113, AD-35): приём, воркер, живые обновления, ops.
	mux.Handle("GET /metrics", env.metricsHandler(opts.ops))
	attachSimulation(env, opts.simulation, simProxy, api, mux)
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		if what, ok := appendOnly(r); ok {
			// Д-62, AD-28: попытка правки журнала или журнала критических
			// действий через API — 405 (код семейства api), а не 404.
			w.Header().Set("Allow", "GET")
			w.WriteHeader(http.StatusMethodNotAllowed)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type":   "urn:ant:problem:api.method_not_allowed",
				"title":  "Метод не разрешён",
				"status": http.StatusMethodNotAllowed,
				"code":   "api.method_not_allowed",
				"detail": what + " только дописывается: " + r.Method + " " + r.URL.Path + " не поддерживается — записи не правятся и не удаляются (AD-2, AD-28)",
				"params": map[string]string{"object": what, "method": r.Method},
			})
			return
		}
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

// readyHandler — готовность: итог самопроверки после старта и ответ БД сейчас.
func readyHandler(pool *pgxpool.Pool, sc *selfCheckState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()
		code, body := readyView(sc.get(), pool.Ping(ctx))
		writeJSON(w, code, body)
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

// appendOnly — запрос на изменение журнала или журнала критических действий
// (методы, кроме чтения, по путям, у которых есть только чтение): такие
// журналы только дописываются системой (AD-2, AD-28, Д-62).
func appendOnly(r *http.Request) (string, bool) {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		return "", false
	}
	p := r.URL.Path
	switch {
	case p == "/api/v1/journal" || strings.HasPrefix(p, "/api/v1/journal/"):
		return "Журнал событий", true
	case p == "/api/v1/critical-actions" || strings.HasPrefix(p, "/api/v1/critical-actions/"):
		return "Журнал критических действий", true
	}
	return "", false
}
