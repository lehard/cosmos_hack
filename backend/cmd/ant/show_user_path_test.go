package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"uuid"

	"ant/cmd/internal/config"
	"ant/internal/application/platform"
	"ant/internal/infrastructure/storage/journal/journaltest"
	"ant/internal/infrastructure/transport/httpapi"
)

// Сквозной тест показа SHOW-IS2 «как пользователь» (решение пользователя через
// сессию «Процесс»): весь процесс ant профиля demo на своей БД (make dev-db) —
// роль init (ключи, генезис), затем роли api, worker, crossitem, projector,
// scheduler, outbox и stands в одном процессе, как в compose. Пульт запускает
// SHOW-IS2 в интерактивном режиме; историю до live.from проигрывает demo-signer.
// Дальше на каждой остановке прогона (simulation.run.plan → waiting_for) тест
// делает ровно то, что человек на своём столе: берёт задачу своей роли из
// «Задач» (notifications.task.list, как выдвижное окно) с operation_id ожидаемого
// действия и выполняет его над объектом ЗАДАЧИ; решения, которые интерфейс
// ведёт не через «Задачи» (очередь контролёра, карточка инцидента, окно поста),
// — теми же чтениями, что экран, и командой над объектом из чтения. После
// нажатия прогон должен уйти с этой остановки, а объект задачи — совпасть с
// объектом ожидания. Падение называет остановку и причину.
//
// Долгий (минуты) и требует БД — поэтому только по явному запросу:
//
//	ANT_SHOW_USER_PATH=1 go test -run TestShowIS2UserPath -v ./cmd/ant/
//
// (с make dev-db: контейнер Go из Makefile подхватывает .dev/db.env).
func TestShowIS2UserPath(t *testing.T) {
	if os.Getenv("ANT_SHOW_USER_PATH") == "" || os.Getenv("ANT_DB_HOST") == "" {
		t.Skip("сквозной тест показа SHOW-IS2 — только по ANT_SHOW_USER_PATH=1 и со своей БД (make dev-db)")
	}
	s := startShowSystem(t)
	s.startRun()
	s.waitLive()
	s.checkAfterHistory()
	s.walk()
}

// showSystem — процесс ant на своей БД и клиент HTTP «как интерфейс».
type showSystem struct {
	t      *testing.T
	ctx    context.Context
	base   string
	client *http.Client
	routes map[string]showRoute
	runID  string
	// stops — пройденные остановки (для отчёта).
	stops []string
	// runs — изделие → выполнение, начатое с терминала (startedHere).
	runs map[string]string
	// ncs — фланец → несоответствие, открытое контролёром в карточке.
	ncs map[string]string
	// note — пояснение к ожиданию после нажатия (для сообщения о падении).
	note string
	// w21Checked — задачи сварщика после приёмки партии проверены.
	w21Checked bool
}

type showRoute struct{ method, path string }

// activeRunSkip — пути, которым фронт не подставляет run_id активного прогона (SKIP в active-run.ts).
var activeRunSkip = regexp.MustCompile(`^/api/v1/(runs|scenarios|auth|tasks)(/|$)`)

func freeAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	a := ln.Addr().String()
	_ = ln.Close()
	return a
}

func startShowSystem(t *testing.T) *showSystem {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Minute)
	db := journaltest.NewDB(t)
	dir := t.TempDir()
	for k, v := range map[string]string{"ANT_KEYS_DIR": "ant-keys", "ANT_KEEPER_DIR": "keeper", "ANT_VERIFIER_DIR": "verifier",
		"ANT_PKI_DIR": "ant-pki", "ANT_DEMO_SIGNER_KEYS": "demo-signer", "ANT_DEVICE_KEYS": "edge", "ANT_PARTNER_KEYS": "partner",
		"ANT_TOKEN_AGENT_KEYS": "token-agent"} {
		t.Setenv(k, filepath.Join(dir, v))
	}
	t.Setenv("ANT_DB_NAME", db.Name)
	t.Setenv("ANT_PROFILE", config.ProfileDemo)
	cfg, err := config.Load("../../../deploy/config/ant.yaml", os.LookupEnv)
	if err != nil {
		t.Fatal(err)
	}
	// Свои порты и каталоги: порты стенда пользователя (8480–8491) не берём.
	httpAddr, standsAddr := freeAddr(t), freeAddr(t)
	cfg.HTTP.Addr = httpAddr
	cfg.Stands.Addr = standsAddr
	cfg.Stands.Scenarios = "../../../scenarios"
	cfg.ERP.OneC.BaseURL = "http://" + standsAddr + "/stand/1c/erp"
	cfg.ERP.Galaktika.Dir = filepath.Join(dir, "galaktika")
	cfg.Materials.Dir = filepath.Join(dir, "materials")
	cfg.Security.KEKFile = filepath.Join(dir, "kek")
	cfg.Security.ExportFile = ""
	// Память машины: меньше партиций и соединений, чем у стенда.
	cfg.Engine.Partitions = 4
	cfg.DB.MaxConns = 12
	var logw io.Writer = io.Discard
	if p := os.Getenv("ANT_SHOW_LOG"); p != "" {
		f, err := os.Create(p)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = f.Close() })
		logw = f
	}
	log := slog.New(slog.NewTextHandler(logw, &slog.HandlerOptions{Level: slog.LevelInfo}))
	roles := []string{"api", "worker", "crossitem", "projector", "scheduler", "outbox", "stands"}
	env := &environment{cfg: cfg, log: log, ctx: ctx, selectedRoles: roles}
	if err := runInit(ctx, env); err != nil {
		cancel()
		t.Fatalf("init: %v", err)
	}
	// Ядро init — своё; роли откроют его заново.
	env.coreH = coreHolder{}
	done := make(chan error, 1)
	go func() { done <- runRoles(ctx, roles, env) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(30 * time.Second):
		}
		env.closeCore()
	})
	s := &showSystem{t: t, ctx: ctx, base: "http://" + httpAddr, client: &http.Client{Timeout: 60 * time.Second}}
	a := buildAPI(http.NewServeMux(), apiOptions{mode: platform.ModeLive})
	s.routes = map[string]showRoute{}
	for path, item := range a.Huma().OpenAPI().Paths {
		if item.Get != nil {
			s.routes[item.Get.OperationID] = showRoute{http.MethodGet, path}
		}
		if item.Post != nil {
			s.routes[item.Post.OperationID] = showRoute{http.MethodPost, path}
		}
	}
	// Процесс готов: /readyz (самопроверка) и ответ операций.
	s.until("процесс ant не стал готов (/readyz)", 3*time.Minute, func() bool {
		select {
		case err := <-done:
			t.Fatalf("роли остановились: %v", err)
		default:
		}
		resp, err := s.client.Get(s.base + "/readyz")
		if err != nil {
			return false
		}
		_ = resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	})
	return s
}

// until — ждать условие (опрос раз в 300 мс) не дольше d; иначе — падение с what.
func (s *showSystem) until(what string, d time.Duration, cond func() bool) {
	s.t.Helper()
	deadline := time.Now().Add(d)
	for !cond() {
		if time.Now().After(deadline) {
			s.t.Fatalf("%s (ждали %s)", what, d)
		}
		time.Sleep(300 * time.Millisecond)
	}
}

// call — операция API от имени демо-персоны (заголовок Ant-Demo-Persona —
// демо-подпись профиля demo, как у demo-signer и стола под персоной).
func (s *showSystem) call(persona, op string, params map[string]string, body any) (int, map[string]any) {
	s.t.Helper()
	r, ok := s.routes[op]
	if !ok {
		s.t.Fatalf("операции %s нет в API", op)
	}
	path := r.path
	q := url.Values{}
	for k, v := range params {
		if ph := "{" + k + "}"; strings.Contains(path, ph) {
			path = strings.ReplaceAll(path, ph, url.PathEscape(v))
			continue
		}
		q.Set(k, v)
	}
	if strings.Contains(path, "{") {
		s.t.Fatalf("%s: не хватает параметров пути в %s", op, path)
	}
	// Активный прогон (frontend/src/shared/api/active-run.ts): пока на пульте идёт
	// прогон, все чтения /api/v1/* без run_id получают run_id активного прогона,
	// кроме прогонов, сценариев, входа и задач.
	if r.method == http.MethodGet && s.runID != "" && !q.Has("run_id") && !activeRunSkip.MatchString(r.path) {
		q.Set("run_id", s.runID)
	}
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			s.t.Fatal(err)
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(s.ctx, r.method, s.base+path, rd)
	if err != nil {
		s.t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if persona != "" {
		req.Header.Set(httpapi.DemoPersonaHeader, persona)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		s.t.Fatalf("%s %s: %v", op, path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	var out map[string]any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	_ = dec.Decode(&out)
	if out == nil {
		out = map[string]any{"_raw": string(raw)}
	}
	return resp.StatusCode, out
}

// read — чтение, которое должно ответить 200.
func (s *showSystem) read(persona, op string, params map[string]string) map[string]any {
	s.t.Helper()
	code, out := s.call(persona, op, params, nil)
	if code != http.StatusOK {
		s.t.Fatalf("%s от %s %v: HTTP %d %v", op, persona, params, code, out)
	}
	return out
}

func str(m map[string]any, k string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[k]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}

func list(m map[string]any, k string) []map[string]any {
	var out []map[string]any
	if m == nil {
		return nil
	}
	xs, _ := m[k].([]any)
	for _, x := range xs {
		if e, ok := x.(map[string]any); ok {
			out = append(out, e)
		}
	}
	return out
}

func (s *showSystem) startRun() {
	s.t.Helper()
	code, out := s.call("ADM-01", "simulation.run.start", map[string]string{"scenario_id": "SHOW-IS2"},
		map[string]any{"command_id": newID(), "basis_seq": 0, "policy_seq": 0, "mode": "interactive", "speed": 600})
	if code != http.StatusOK || str(out, "run_id") == "" {
		s.t.Fatalf("запуск SHOW-IS2: HTTP %d %v", code, out)
	}
	s.runID = str(out, "run_id")
	s.t.Logf("прогон %s", s.runID)
}

// run — состояние прогона на пульте (simulation.run.read).
func (s *showSystem) run() map[string]any {
	s.t.Helper()
	return s.read("ADM-01", "simulation.run.read", map[string]string{"run_id": s.runID})
}

// plan — план прогона с пульта (simulation.run.plan).
func (s *showSystem) plan() map[string]any {
	s.t.Helper()
	return s.read("ADM-01", "simulation.run.plan", map[string]string{"run_id": s.runID, "limit": "200"})
}

func (s *showSystem) waitLive() {
	s.t.Helper()
	var last map[string]any
	s.until("история не проиграна: прогон не дошёл до первой остановки живой части", 20*time.Minute, func() bool {
		last = s.read("ADM-01", "simulation.run.read", map[string]string{"run_id": s.runID})
		switch str(last, "state") {
		case "failed", "stopped", "completed":
			s.t.Fatalf("прогон остановился на истории: %v", last)
		}
		return str(last, "state") == "waiting_for_decision"
	})
	s.t.Logf("живая часть: %v", last["waiting_for"])
}

func newID() string { return uuid.NewV7().String() }

// tasks — открытые задачи персоны в выдвижном окне «Задачи» шапки
// (NotificationsPanel → useTasks({state: 'open'}): limit 200; задачи фронт
// читает без run_id — active-run.ts их не трогает).
func (s *showSystem) tasks(persona string) []map[string]any {
	s.t.Helper()
	return list(s.read(persona, "notifications.task.list", map[string]string{"state": "open", "limit": "200"}), "items")
}

// ─────────────────────────────── остановки ───────────────────────────────

// showStop — остановка прогона: строка плана пульта с waiting=true и
// waiting_for («сейчас ждём: ‹роль› — ‹действие›»).
type showStop struct {
	N                                             int
	Label, Op, Role, Persona, Object, Item, Title string
}

func (st *showStop) String() string {
	return fmt.Sprintf("остановка %d «%s» (%s, %s %s, объект %s)", st.N, st.Label, st.Persona, st.Role, st.Op, st.Object)
}

var flangeRe = regexp.MustCompile(`[FФ]-\d{3}`)

// flangeOf — номер фланца в тексте (F-001, Ф-001) в латинице: F-001.
func flangeOf(text string) string {
	if m := flangeRe.FindString(text); m != "" {
		return "F-" + m[strings.Index(m, "-")+1:]
	}
	return ""
}

// flange — фланец остановки, как его знает человек (F-001 / Ф-001): из метки
// шага карточки или заголовка ожидания.
func (st *showStop) flange() string {
	if f := flangeOf(st.Label); f != "" {
		return f
	}
	return flangeOf(st.Title)
}

// sameFlange — метка изделия для людей (Ф-001, DM:F-001, «… Ф-001») — этот фланец.
func sameFlange(label, flange string) bool {
	if flange == "" {
		return false
	}
	return strings.Contains(label, flange) || strings.Contains(label, "Ф-"+strings.TrimPrefix(flange, "F-"))
}

// current — состояние прогона и остановка, на которой он стоит (nil — не стоит).
func (s *showSystem) current() (string, *showStop) {
	s.t.Helper()
	p := s.plan()
	state := str(p, "state")
	if state != "waiting_for_decision" {
		return state, nil
	}
	w, _ := p["waiting_for"].(map[string]any)
	for _, e := range list(p, "items") {
		if e["waiting"] == true {
			return state, &showStop{Label: str(e, "label"), Op: str(w, "action"), Role: str(w, "role"), Persona: str(e, "persona"),
				Object: str(w, "object_id"), Item: str(e, "item_id"), Title: str(w, "title")}
		}
	}
	s.t.Fatalf("прогон ждёт %v, но в плане нет строки waiting", w)
	return state, nil
}

// waitFor — ждать условие (опрос раз в 300 мс) не дольше d; иначе падение с
// what и последним пояснением условия.
func (s *showSystem) waitFor(what string, d time.Duration, cond func() (bool, string)) {
	s.t.Helper()
	deadline := time.Now().Add(d)
	for {
		ok, detail := cond()
		if ok {
			return
		}
		if time.Now().After(deadline) {
			s.t.Fatalf("%s (ждали %s): %s", what, d, detail)
		}
		time.Sleep(300 * time.Millisecond)
	}
}

// walk — остановки живой части по очереди: действие человека, затем прогон
// должен уйти с этой остановки.
func (s *showSystem) walk() {
	s.t.Helper()
	s.runs = map[string]string{}
	s.ncs = map[string]string{}
	defer func() {
		s.t.Logf("пройдено остановок: %d\n%s", len(s.stops), strings.Join(s.stops, "\n"))
	}()
	// Состояние — simulation.run.read (шаг прогона); план (simulation.run.plan)
	// опрашивается и на ходу, как пульт: чтение плана не должно мешать раннеру
	// (было: concurrent map write в IDMap плана ронял ant).
	prev := ""
	for n := 1; n <= 60; n++ {
		var r map[string]any
		s.waitFor(fmt.Sprintf("после «%s» прогон не дошёл до следующей остановки", prev), 5*time.Minute, func() (bool, string) {
			s.plan()
			r = s.run()
			switch str(r, "state") {
			case "failed", "stopped":
				s.t.Fatalf("прогон остановился после «%s»: %v", prev, r)
			case "completed":
				return true, ""
			}
			return str(r, "state") == "waiting_for_decision", "состояние " + str(r, "state")
		})
		if str(r, "state") == "completed" {
			break
		}
		_, st := s.current()
		if st == nil {
			s.t.Fatalf("после «%s»: прогон ждёт, а план без остановки: %v", prev, r)
		}
		st.N = n
		step := str(r, "step")
		s.t.Logf("%s — %s", st, st.Title)
		how := s.act(st)
		s.stops = append(s.stops, fmt.Sprintf("%2d. %-7s %-24s %s", n, st.Persona, st.Label, how))
		s.waitFor(fmt.Sprintf("%s: после нажатия прогон стоит — решение человека не засчитано", st), 2*time.Minute, func() (bool, string) {
			s.plan()
			r := s.run()
			if str(r, "state") != "waiting_for_decision" || str(r, "step") != step {
				return true, ""
			}
			return false, fmt.Sprintf("прогон ждёт %v%s", r["waiting_for"], s.note)
		})
		s.note = ""
		prev = st.Label
	}
	if r := s.run(); str(r, "state") != "completed" {
		s.t.Fatalf("показ не дошёл до конца: %s", str(r, "state"))
	}
	if len(s.stops) != 27 {
		s.t.Errorf("остановок %d, по карточке 27", len(s.stops))
	}
}

// act — действие человека на остановке тем путём, каким его делает интерфейс;
// возвращает путь для отчёта («задачи» или экран).
func (s *showSystem) act(st *showStop) string {
	s.t.Helper()
	switch st.Op {
	case "process.movement.receive":
		if strings.Contains(st.Label, "isolator") {
			return s.isolatorMove(st)
		}
		return s.receive(st)
	case "access.workplace.admit":
		return s.admit(st)
	case "process.operation.start":
		return s.start(st)
	case "process.operation.finish":
		return s.finish(st)
	case "nonconformity.presentation.resolve":
		return s.resolve(st)
	case "nonconformity.nonconformity.confirm":
		return s.confirm(st)
	case "nonconformity.item.isolate":
		return s.isolate(st)
	case "nonconformity.recheck.request":
		return s.recheck(st)
	case "nonconformity.process_hold.set":
		return s.hold(st)
	case "analysis.scope.narrow":
		return s.narrow(st)
	case "analysis.measurement.request":
		return s.measure(st)
	case "analysis.cause.conclude":
		return s.cause(st)
	case "nonconformity.disposition.set":
		return s.disposition(st)
	}
	s.t.Fatalf("%s: нет пути интерфейса для %s", st, st.Op)
	return ""
}

// ─────────────────────────────── чтения стола ───────────────────────────────

type showSession struct {
	user, workplace string
	policySeq       any
}

func (s *showSystem) session(persona string) showSession {
	s.t.Helper()
	m := s.read(persona, "access.session.read", nil)
	u, _ := m["user"].(map[string]any)
	w, _ := m["workplace"].(map[string]any)
	return showSession{user: str(u, "id"), workplace: str(w, "id"), policySeq: m["policy_seq"]}
}

// plainMeta — поля команды технолога (useAnalysisCommands): без рабочего места.
func (ss showSession) plainMeta(basis any) map[string]any {
	return map[string]any{"command_id": newID(), "basis_seq": basis, "policy_seq": ss.policySeq}
}

// meta — общие поля команды, как у интерфейса (AD-7, AD-39).
func (ss showSession) meta(basis any) map[string]any {
	m := map[string]any{"command_id": newID(), "basis_seq": basis, "policy_seq": ss.policySeq}
	if ss.workplace != "" {
		m["workplace_id"] = ss.workplace
	}
	return m
}

func (s *showSystem) head(persona string) any {
	s.t.Helper()
	return s.read(persona, "journal.head.read", nil)["seq"]
}

// passport — паспорт изделия (basis_seq команд исполнителя — из него).
func (s *showSystem) passport(persona, item string) map[string]any {
	s.t.Helper()
	return s.read(persona, "item.passport.read", map[string]string{"item_id": item})
}

// command — команда, которая должна быть принята.
func (s *showSystem) command(st *showStop, persona, op string, params map[string]string, body map[string]any) {
	s.t.Helper()
	code, out := s.call(persona, op, params, body)
	if code < 200 || code >= 300 {
		s.t.Fatalf("%s: команда %s от %s отклонена: HTTP %d %v\nтело: %v", st, op, persona, code, out, body)
	}
}

// object — объект, который человек взял из задачи или экрана, — тот, которого ждёт прогон.
func (s *showSystem) object(st *showStop, got, from string) {
	s.t.Helper()
	if st.Object != "" && got != st.Object {
		s.t.Fatalf("%s: %s ведёт на %s, а прогон ждёт решения над %s", st, from, got, st.Object)
	}
}

// task — единственная открытая задача персоны по фланцу остановки с operation_id
// (и видом kind, если задан), как её находит человек в «Задачах».
func (s *showSystem) task(st *showStop, op, kind string) map[string]any {
	s.t.Helper()
	all := s.tasks(st.Persona)
	var hits []map[string]any
	for _, x := range all {
		if (op == "" || str(x, "operation_id") == op) && (kind == "" || str(x, "kind") == kind) && sameFlange(str(x, "item_label")+" "+str(x, "title"), st.flange()) {
			hits = append(hits, x)
		}
	}
	switch {
	case len(hits) == 0:
		s.t.Fatalf("%s: в «Задачах» %s нет задачи %s%s по %s; открытые: %s", st, st.Persona, op, kind, st.flange(), taskSummary(all))
	case len(hits) > 1:
		s.t.Fatalf("%s: в «Задачах» %s %d задач %s по %s — какую нажать? %s", st, st.Persona, len(hits), op, st.flange(), taskSummary(hits))
	}
	return hits[0]
}

func taskSummary(ts []map[string]any) string {
	var out []string
	for i, x := range ts {
		if i == 12 {
			out = append(out, fmt.Sprintf("… ещё %d", len(ts)-i))
			break
		}
		out = append(out, fmt.Sprintf("[%s %s %s «%s»]", str(x, "kind"), str(x, "operation_id"), str(x, "item_id"), str(x, "title")))
	}
	return fmt.Sprintf("%d: %s", len(ts), strings.Join(out, " "))
}

// ─────────────────────────────── действия ───────────────────────────────

// receive — мастер: «Задачи» → «Принять в цех» (форма в задаче, ReceiveAction).
func (s *showSystem) receive(st *showStop) string {
	s.t.Helper()
	s.noForeignTasks(st)
	t := s.task(st, st.Op, "")
	item := str(t, "item_id")
	s.object(st, item, fmt.Sprintf("задача «%s»", str(t, "title")))
	ss := s.session(st.Persona)
	body := ss.meta(s.head(st.Persona))
	body["to_location_id"] = str(t, "location_id")
	body["destination_kind"] = "workshop"
	body["inspection_on_receipt"] = "no_damage"
	if k := str(t, "step_key"); k != "" {
		body["step_key"] = k // шаг процесса из задачи (ReceiveAction :step-key)
	}
	s.command(st, st.Persona, st.Op, map[string]string{"item_id": item}, body)
	return "задачи: «" + str(t, "title") + "» → форма «Принять»"
}

// isolatorMove — мастер: задача «в изолятор» → «Принять в изоляторе» (IsolatorMoveConfirm).
func (s *showSystem) isolatorMove(st *showStop) string {
	s.t.Helper()
	s.noForeignTasks(st)
	t := s.task(st, "", "isolate_move")
	item := str(t, "item_id")
	if item == "" && t["ref"] != nil {
		item = str(t["ref"].(map[string]any), "id")
	}
	s.object(st, item, fmt.Sprintf("задача «%s»", str(t, "title")))
	pp := s.passport(st.Persona, item)
	ncs, _ := pp["nonconformities"].([]any)
	if len(ncs) == 0 {
		s.t.Fatalf("%s: в паспорте %s нет несоответствия — изоляции по решению нет", st, item)
	}
	card := s.read(st.Persona, "nonconformity.card.read", map[string]string{"nc_id": fmt.Sprint(ncs[len(ncs)-1])})
	// Изолятор по умолчанию — из решения «изолировать», иначе первый изолятор
	// цеха рабочего места (IsolatorMoveConfirm, isolatorsFor).
	iso, _ := card["isolation"].(map[string]any)
	to := str(iso, "isolator_location_id")
	if to == "" {
		to = s.firstIsolator(st.Persona)
	}
	if to == "" {
		s.t.Fatalf("%s: изолятора нет ни в решении «изолировать» (%v), ни в справочнике мест", st, card["isolation"])
	}
	body := s.session(st.Persona).meta(pp["basis_seq"])
	body["destination_kind"] = "isolator"
	body["to_location_id"] = to
	body["inspection_on_receipt"] = "no_damage"
	s.command(st, st.Persona, st.Op, map[string]string{"item_id": item}, body)
	return "задачи: «" + str(t, "title") + "» → «Принять в изоляторе»"
}

// firstIsolator — первый изолятор цеха рабочего места сеанса (isolatorsFor):
// без рабочего места — первый изолятор справочника.
func (s *showSystem) firstIsolator(persona string) string {
	s.t.Helper()
	locs := list(s.read(persona, "reference.location.list", nil), "items")
	wp := s.session(persona).workplace
	shop := ""
	for _, l := range locs {
		if str(l, "location_id") == wp {
			shop = str(l, "parent_id") // пост → участок → цех
			for _, x := range locs {
				if str(x, "location_id") == shop && str(x, "kind") != "workshop" {
					shop = str(x, "parent_id")
				}
			}
		}
	}
	first := ""
	for _, l := range locs {
		if str(l, "kind") != "isolator" {
			continue
		}
		if shop != "" && str(l, "parent_id") == shop {
			return str(l, "location_id")
		}
		if first == "" {
			first = str(l, "location_id")
		}
	}
	return first
}

// admit — сварщик: терминал исполнителя → «Допуск к посту» (WorkplaceAdmission):
// свои посты из панели постов, ключ w21@1, открыт PIN-ом (как расширение).
func (s *showSystem) admit(st *showStop) string {
	s.t.Helper()
	for _, x := range s.tasks(st.Persona) {
		if str(x, "operation_id") == st.Op {
			s.t.Logf("%s: есть задача допуска «%s»", st, str(x, "title"))
		}
	}
	if !s.w21Checked {
		s.w21Checked = true
		s.checkW21AfterReceive(st)
	}
	ss := s.session(st.Persona)
	var mine []string
	for _, p := range list(s.read(st.Persona, "access.workplace.list", nil), "items") {
		if a, _ := p["assigned"].(map[string]any); str(a, "person_id") == ss.user {
			mine = append(mine, str(p, "workplace_id"))
		}
	}
	if !slices.Contains(mine, st.Object) {
		s.t.Fatalf("%s: на терминале %s среди своих постов нет %s: %v", st, st.Persona, st.Object, mine)
	}
	body := ss.meta(0)
	body["workplace_id"] = st.Object
	body["key_ref"] = strings.ToLower(st.Persona) + "@1"
	body["pin_verified"] = true
	s.command(st, st.Persona, st.Op, map[string]string{"workplace_id": st.Object}, body)
	// Оборудование поста на терминале и в окне поста — по station_id == пост.
	var eqs []string
	for _, e := range list(s.read(st.Persona, "machinelogs.equipment.list", nil), "items") {
		eqs = append(eqs, str(e, "equipment_id")+"@"+str(e, "station_id"))
	}
	if eq := list(s.read(st.Persona, "machinelogs.equipment.list", map[string]string{"station_id": st.Object}), "items"); len(eq) == 0 {
		s.t.Logf("%s: у поста %s нет оборудования (machinelogs.equipment.list?station_id): терминал «Начать» уйдёт без equipment_id, в окне поста нет «Остановить пост»; всё оборудование: %v", st, st.Object, eqs)
	}
	return "экран: терминал исполнителя → «Допуск к посту» " + st.Object
}

var opCodeRe = regexp.MustCompile(`<ant:properties [^>]*>`)

// operationCode — код операции шага из схемы живой карты (как parseProcessSteps).
func (s *showSystem) operationCode(persona, step string) string {
	s.t.Helper()
	xml := str(s.read(persona, "process.live_map.read", map[string]string{"period": "shift"}), "bpmn_xml")
	for _, tag := range opCodeRe.FindAllString(xml, -1) {
		if strings.Contains(tag, `stepKey="`+step+`"`) {
			if m := regexp.MustCompile(`operationCode="([^"]*)"`).FindStringSubmatch(tag); m != nil && m[1] != "" {
				return m[1]
			}
		}
	}
	return step
}

// start — сварщик: «Задачи» «Начать: …» → терминал, «Начать» по изделию задачи
// (PerformerTerminalWidget.onStart: новый operation_run_id, оборудование поста).
func (s *showSystem) start(st *showStop) string {
	s.t.Helper()
	t := s.task(st, st.Op, "")
	item, step := str(t, "item_id"), str(t, "step_key")
	s.object(st, item, fmt.Sprintf("задача «%s»", str(t, "title")))
	ss := s.session(st.Persona)
	if ss.workplace == "" {
		s.t.Fatalf("%s: в сеансе %s нет рабочего места — терминал без допуска", st, st.Persona)
	}
	// Очередь терминала (PerformerTerminalWidget.candidates): изделия открытых
	// задач «Начать» этого шага; нет таких задач — изделия шага (item.item.list).
	var cands []string
	for _, x := range list(s.read(st.Persona, "notifications.task.list", map[string]string{"limit": "200"}), "items") {
		if str(x, "state") == "open" && str(x, "operation_id") == "process.operation.start" && str(x, "step_key") == step && str(x, "item_id") != "" {
			cands = append(cands, str(x, "item_id"))
		}
	}
	if len(cands) == 0 {
		for _, r := range list(s.read(st.Persona, "item.item.list", map[string]string{"step_key": step, "limit": "100"}), "items") {
			cands = append(cands, str(r, "item_id"))
		}
	}
	pp := s.passport(st.Persona, item)
	if !slices.Contains(cands, item) {
		// Нет изделия в очереди терминала — у сварщика нет кнопки «Начать», хотя задача есть.
		s.t.Errorf("%s: на терминале нет кнопки «Начать» по %s — в очереди шага %s его нет (%d); в паспорте шаг %s, состояние %v",
			st, item, step, len(cands), str(pp, "step_key"), pp["status"])
	}
	eq := list(s.read(st.Persona, "machinelogs.equipment.list", map[string]string{"station_id": ss.workplace}), "items")
	run := newID()
	body := ss.meta(pp["basis_seq"])
	body["operation_run_id"] = run
	body["operation_code"] = s.operationCode(st.Persona, step)
	body["step_key"] = step
	body["station_id"] = ss.workplace
	if len(eq) > 0 {
		body["equipment_id"] = str(eq[0], "equipment_id")
	}
	s.command(st, st.Persona, st.Op, map[string]string{"item_id": item}, body)
	s.runs[item] = run
	return "задачи: «" + str(t, "title") + "» → терминал «Начать»"
}

// finish — сварщик: «Задачи» «Завершить…» → терминал «Выполнено» по текущему
// выполнению поста (начатому с терминала).
func (s *showSystem) finish(st *showStop) string {
	s.t.Helper()
	t := s.task(st, st.Op, "")
	item := str(t, "item_id")
	if st.Item != "" && item != st.Item {
		s.t.Fatalf("%s: задача «%s» ведёт на %s, а прогон ждёт сварку изделия %s", st, str(t, "title"), item, st.Item)
	}
	ss := s.session(st.Persona)
	// Как currentRunId терминала: текущее выполнение оборудования поста, иначе начатое здесь.
	run := s.runs[item]
	for _, e := range list(s.read(st.Persona, "machinelogs.equipment.list", map[string]string{"station_id": ss.workplace}), "items") {
		if r := str(e, "current_run_id"); r != "" {
			if r != run {
				s.t.Logf("%s: оборудование %s показывает текущее выполнение %s, начато с терминала %s", st, str(e, "equipment_id"), r, run)
			}
			run = r
			break
		}
	}
	if run == "" {
		s.t.Fatalf("%s: на терминале нет текущего выполнения по %s", st, item)
	}
	if run != st.Object {
		s.note = fmt.Sprintf("; терминал завершил выполнение %s (id от «Начать»), а остановка ждёт %s — выполнения с таким id нет", run, st.Object)
	}
	body := ss.meta(s.passport(st.Persona, item)["basis_seq"])
	body["completion"] = "completed"
	s.command(st, st.Persona, st.Op, map[string]string{"run_id": run}, body)
	return "задачи: «" + str(t, "title") + "» → терминал «Выполнено»"
}

// queueRow — строка очереди «Ждут моего решения» по фланцу остановки
// (DecisionQueueWidget → useDecisionQueue: sort=risk, run_id активного прогона).
func (s *showSystem) queueRow(st *showStop, kinds ...string) map[string]any {
	s.t.Helper()
	all := list(s.read(st.Persona, "nonconformity.queue.list", map[string]string{"sort": "risk"}), "items")
	var hits []map[string]any
	for _, r := range all {
		if (len(kinds) == 0 || slices.Contains(kinds, str(r, "kind"))) && sameFlange(str(r, "item_label")+" "+str(r, "title"), st.flange()) {
			hits = append(hits, r)
		}
	}
	if len(hits) == 0 {
		var rows []string
		for _, r := range all {
			rows = append(rows, fmt.Sprintf("[%s %s %s «%s»]", str(r, "kind"), str(r, "item_label"), str(r, "item_id"), str(r, "title")))
		}
		s.t.Fatalf("%s: в очереди %s нет строки %v по %s; очередь (%d): %s\n%s", st, st.Persona, kinds, st.flange(), len(all), strings.Join(rows, " "), s.itemDiag(st.Item))
	}
	if len(hits) > 1 {
		var rows []string
		for _, r := range hits {
			rows = append(rows, fmt.Sprintf("[%s «%s»]", str(r, "kind"), str(r, "title")))
		}
		s.t.Logf("%s: в очереди %d строк по %s, беру первую: %s", st, len(hits), st.flange(), strings.Join(rows, " "))
	}
	return hits[0]
}

// resolve — контролёр: очередь «Ждут моего решения» → ЗТ-3 → «Принять и передать»
// (PresentationRecord: nonconformity.presentation.read и действие с resolution accept).
func (s *showSystem) resolve(st *showStop) string {
	s.t.Helper()
	s.logTasks(st)
	row := s.queueRow(st, "presentation")
	item := str(row, "item_id")
	s.object(st, item, fmt.Sprintf("строка очереди «%s»", str(row, "title")))
	v := s.read(st.Persona, "nonconformity.presentation.read", map[string]string{"item_id": item})
	var act map[string]any
	for _, a := range list(v, "actions") {
		if str(a, "operation") == st.Op && str(a, "resolution") == "accept" && a["allowed"] == true {
			act = a
		}
	}
	if act == nil {
		s.t.Fatalf("%s: в окне предъявления нет доступного «Принять»: %v", st, v["actions"])
	}
	p, _ := v["presentation"].(map[string]any)
	body := s.session(st.Persona).meta(v["basis_seq"])
	body["step_key"] = p["step_key"]
	body["closing_point"] = p["closing_point"]
	body["presentation_no"] = p["presentation_no"]
	body["method_event_ids"] = p["method_event_ids"]
	body["resolution"] = "accept"
	s.command(st, st.Persona, st.Op, map[string]string{"item_id": item}, body)
	return "экран: очередь контролёра → ЗТ-3 «" + str(act, "label") + "»"
}

// card — карточка несоответствия, в которой доступно решение op.
func (s *showSystem) card(st *showStop, nc string) map[string]any {
	s.t.Helper()
	c := s.read(st.Persona, "nonconformity.card.read", map[string]string{"nc_id": nc})
	td, _ := c["to_decide"].(map[string]any)
	ds, _ := td["decisions"].([]any)
	if !slices.ContainsFunc(ds, func(x any) bool { return fmt.Sprint(x) == st.Op }) {
		s.t.Fatalf("%s: в карточке НС %s (%s) нет решения %s; доступны %v", st, nc, str(c, "status"), st.Op, ds)
	}
	return c
}

// decisionBody — тело решения из карточки (buildDecisionRequest).
func (s *showSystem) decisionBody(st *showStop, c map[string]any, reason string) map[string]any {
	b := s.session(st.Persona).meta(c["basis_seq"])
	b["reason"] = map[string]any{"text": reason}
	return b
}

// confirm — контролёр: очередь → сигнал по Ф-003 → карточка НС → «Подтвердить несоответствие».
func (s *showSystem) confirm(st *showStop) string {
	s.t.Helper()
	s.logTasks(st)
	row := s.queueRow(st, "signal")
	nc := str(row, "nc_id")
	s.object(st, nc, fmt.Sprintf("строка очереди «%s»", str(row, "title")))
	c := s.card(st, nc)
	ev, _ := c["evidence"].(map[string]any)
	sigs := list(ev, "signals")
	var ids []string
	for _, x := range sigs {
		ids = append(ids, str(x, "signal_id"))
	}
	b := s.decisionBody(st, c, "Прожог У2 подтверждён по двум ракурсам КТ-3")
	b["signal_ids"] = ids
	b["severity"] = "unknown"
	if len(sigs) > 0 {
		if v := str(sigs[0], "severity"); v != "" {
			b["severity"] = v
		}
		if v := str(sigs[0], "defect_type_code"); v != "" {
			b["defect_type_code"] = v
		}
	}
	if r, _ := ev["requirement"].(map[string]any); str(r, "kd_ref") != "" {
		b["requirement_ref"] = str(r, "kd_ref")
	}
	s.command(st, st.Persona, st.Op, map[string]string{"nc_id": nc}, b)
	s.ncs[st.flange()] = nc
	return "экран: очередь контролёра → карточка НС «Подтвердить несоответствие»"
}

// ncOf — несоответствие фланца, открытое у человека: то, что он только что
// подтвердил, иначе строка очереди, иначе последнее в паспорте.
func (s *showSystem) ncOf(st *showStop, item string) string {
	s.t.Helper()
	if nc := s.ncs[st.flange()]; nc != "" {
		return nc
	}
	pp := s.passport(st.Persona, item)
	ncs, _ := pp["nonconformities"].([]any)
	if len(ncs) == 0 {
		s.t.Fatalf("%s: у %s нет несоответствия — карточку открыть не из чего", st, item)
	}
	return fmt.Sprint(ncs[len(ncs)-1])
}

// isolate — контролёр: та же карточка НС → «Изолировать до решения».
func (s *showSystem) isolate(st *showStop) string {
	s.t.Helper()
	c := s.card(st, s.ncOf(st, st.Item))
	item := str(c, "item_id")
	s.object(st, item, "карточка НС "+str(c, "number"))
	b := s.decisionBody(st, c, "Прожог У2 — изолировать до решения по изделию")
	s.command(st, st.Persona, st.Op, map[string]string{"item_id": item}, b)
	return "экран: карточка НС → «Изолировать»"
}

// recheck — контролёр: «Задачи» «Доп. проверка Ф-002» → форма в задаче
// (RecheckRequestForm: метод по умолчанию рентген, основание — заголовок задачи,
// basis_seq — голова журнала списка задач).
func (s *showSystem) recheck(st *showStop) string {
	s.t.Helper()
	t := s.task(st, st.Op, "")
	item := str(t, "item_id")
	s.object(st, item, fmt.Sprintf("задача «%s»", str(t, "title")))
	b := s.session(st.Persona).plainMeta(s.head(st.Persona))
	b["method"] = "radiography"
	b["reason"] = map[string]any{"text": str(t, "title")}
	s.command(st, st.Persona, st.Op, map[string]string{"item_id": item}, b)
	return "задачи: «" + str(t, "title") + "» → форма «Доп. проверка»"
}

// hold — мастер: окно поста ИС-2 (WorkplaceRecord) → оборудование поста →
// «Остановить пост» (ProcessHoldAction).
func (s *showSystem) hold(st *showStop) string {
	s.t.Helper()
	const post = "WP-WELD-2" // пост ИС-2 (как в панели постов мастера)
	var eqs []map[string]any
	for _, e := range list(s.read(st.Persona, "machinelogs.equipment.list", nil), "items") {
		if str(e, "station_id") == post {
			eqs = append(eqs, e)
		}
	}
	if len(eqs) != 1 {
		s.t.Fatalf("%s: в окне поста %s оборудования %d: %v", st, post, len(eqs), eqs)
	}
	eq := str(eqs[0], "equipment_id")
	s.object(st, eq, "окно поста "+post)
	b := s.session(st.Persona).meta(s.head(st.Persona))
	b["hold_id"] = fmt.Sprintf("HOLD-%s-%s", eq, str(b, "command_id")[:8])
	b["equipment_id"] = eq
	b["level"] = "process_point_stop"
	b["reason"] = map[string]any{"text": "Ток ИС-2 176–182 А при уставке 160 ± 10; прожог на Ф-003"}
	b["release_condition"] = "Ремонт регулятора тока и проверка контрольным образцом"
	s.command(st, st.Persona, st.Op, nil, b)
	return "экран: окно поста " + post + " → «Остановить пост»"
}

// incident — инцидент в разборе технолога: первый открытый (useFocusedIncident).
func (s *showSystem) incident(st *showStop) map[string]any {
	s.t.Helper()
	all := list(s.read(st.Persona, "analysis.incident.list", nil), "items")
	for _, x := range all {
		if str(x, "status") == "open" {
			return x
		}
	}
	if len(all) == 0 {
		s.t.Fatalf("%s: у технолога нет инцидентов", st)
	}
	return all[0]
}

// narrow — технолог: «Расследование» → «Область риска» → предложение сужения
// (narrow_options) — «Сузить» с его изделиями, основаниями и текстом.
func (s *showSystem) narrow(st *showStop) string {
	s.t.Helper()
	inc := str(s.incident(st), "incident_id")
	s.object(st, inc, "инцидент в разборе")
	rs := s.read(st.Persona, "analysis.risk_scope.read", map[string]string{"incident_id": inc})
	want := "до выхода"
	if strings.Contains(st.Label, "IS1") {
		want = "IS-1"
	}
	var opt map[string]any
	var labels []string
	for _, o := range list(rs, "narrow_options") {
		labels = append(labels, str(o, "label"))
		if opt == nil && (strings.Contains(str(o, "label"), want) || strings.Contains(str(o, "label"), strings.ReplaceAll(want, "IS-", "ИС-"))) {
			opt = o
		}
	}
	if opt == nil {
		s.t.Fatalf("%s: в области риска нет предложения сужения «%s»; есть %v", st, want, labels)
	}
	var ev []string
	for _, e := range list(opt, "evidence") {
		ev = append(ev, str(e, "event_id"))
	}
	b := s.session(st.Persona).plainMeta(rs["basis_seq"])
	b["item_ids"] = opt["item_ids"]
	b["evidence_event_ids"] = ev
	b["reason"] = map[string]any{"text": str(opt, "reason_text")}
	s.command(st, st.Persona, st.Op, map[string]string{"incident_id": inc}, b)
	return "экран: «Расследование» → «Область риска» → предложение «" + str(opt, "label") + "»"
}

// hypothesis — гипотеза «оборудование» у несоответствия расследования.
func (s *showSystem) hypothesis(st *showStop) (inc, nc string, h map[string]any, basis any) {
	s.t.Helper()
	i := s.incident(st)
	inc, nc = str(i, "incident_id"), str(i, "primary_nc_id")
	hs := s.read(st.Persona, "analysis.hypothesis.list", map[string]string{"nc_id": nc})
	var cats []string
	for _, x := range list(hs, "hypotheses") {
		cats = append(cats, str(x, "category"))
		if str(x, "category") == "equipment" && h == nil {
			h = x
		}
	}
	if h == nil {
		s.t.Fatalf("%s: у НС %s нет гипотезы «оборудование»; есть %v", st, nc, cats)
	}
	return inc, nc, h, hs["basis_seq"]
}

// measure — технолог: «Расследование» → «Что проверить следующим» → «Запросить проверку».
func (s *showSystem) measure(st *showStop) string {
	s.t.Helper()
	_, nc, h, basis := s.hypothesis(st)
	s.object(st, nc, "несоответствие расследования")
	what := str(h, "measurement_hint")
	if nc2, _ := h["next_check"].(map[string]any); str(nc2, "text") != "" {
		what = str(nc2, "text")
	}
	b := s.session(st.Persona).plainMeta(basis)
	b["hypothesis_id"] = str(h, "hypothesis_id")
	b["what"] = what
	s.command(st, st.Persona, st.Op, map[string]string{"nc_id": nc}, b)
	return "экран: «Расследование» → гипотеза «оборудование» → «Запросить проверку»"
}

// cause — технолог: гипотеза «оборудование» → «Подтвердить причину».
func (s *showSystem) cause(st *showStop) string {
	s.t.Helper()
	inc, nc, h, basis := s.hypothesis(st)
	s.object(st, inc, "инцидент в разборе")
	b := s.session(st.Persona).plainMeta(basis)
	b["nc_ids"] = []string{nc}
	b["conclusion"] = "confirmed"
	b["category"] = str(h, "category")
	b["hypothesis_id"] = str(h, "hypothesis_id")
	b["verification"] = "журнал тока ИС-2, КТ-3"
	b["reason"] = map[string]any{"text": "Ток ИС-2 вне уставки на сварках Ф-002 и Ф-003; на Ф-001 ток в уставке"}
	s.command(st, st.Persona, st.Op, map[string]string{"incident_id": inc}, b)
	return "экран: «Расследование» → гипотеза «оборудование» → «Подтвердить причину»"
}

// disposition — технолог: окно НС Ф-003 (из расследования) → «Переделка».
func (s *showSystem) disposition(st *showStop) string {
	s.t.Helper()
	nc := str(s.incident(st), "primary_nc_id")
	s.object(st, nc, "НС расследования")
	c := s.card(st, nc)
	b := s.decisionBody(st, c, "Переварка участка У2 по ТП на ИС-1")
	b["disposition"] = "rework"
	s.command(st, st.Persona, st.Op, map[string]string{"nc_id": nc}, b)
	return "экран: «Расследование» → окно НС → «Переделка»"
}

// logTasks — есть ли у человека задача на это решение (для отчёта: какие
// решения приходят в «Задачи»).
func (s *showSystem) logTasks(st *showStop) {
	s.t.Helper()
	for _, x := range s.tasks(st.Persona) {
		if str(x, "operation_id") == st.Op && sameFlange(str(x, "item_label")+" "+str(x, "title"), st.flange()) {
			s.t.Logf("%s: есть и задача «%s» (%s)", st, str(x, "title"), str(x, "item_id"))
		}
	}
}

// ─────────────────────────────── дополнительные проверки ───────────────────────────────

// noForeignTasks — у мастера сварочного цеха FOR-WC нет задач чужого цеха WS-MC.
func (s *showSystem) noForeignTasks(st *showStop) {
	s.t.Helper()
	var bad []map[string]any
	for _, x := range s.tasks("FOR-WC") {
		if str(x, "location_id") == "WS-MC" {
			bad = append(bad, x)
		}
	}
	if len(bad) > 0 {
		s.t.Errorf("%s: у мастера FOR-WC задачи чужого цеха WS-MC: %s", st, taskSummary(bad))
	}
}

// checkAfterHistory — после истории: у каждой Ф-00x ровно одна регистрация;
// у W21 нет задач по шагам, выполненным в истории; у FOR-WC — только свой цех.
func (s *showSystem) checkAfterHistory() {
	s.t.Helper()
	// Регистрации изделий прогона: метка для людей — из паспорта (бирка, DM).
	regs := list(s.read("ADM-01", "journal.entry.list", map[string]string{"event_type": "item.item.registered", "run_id": s.runID, "limit": "500"}), "items")
	byFlange := map[string][]string{}
	for _, e := range regs {
		item := str(e, "item_id")
		pp := s.passport("ADM-01", item)
		label := str(pp, "label")
		var carriers []string
		for _, c := range list(pp, "carriers") {
			carriers = append(carriers, str(c, "value"))
		}
		f := flangeOf(label + " " + strings.Join(carriers, " "))
		byFlange[f] = append(byFlange[f], item)
	}
	s.t.Logf("регистраций изделий в прогоне: %d, из них без номера фланца в метке и носителях: %d", len(regs), len(byFlange[""]))
	for _, f := range []string{"F-001", "F-002", "F-003"} {
		if len(byFlange[f]) != 1 {
			s.t.Errorf("после истории у %s регистраций изделия %d (ждали 1): %v", f, len(byFlange[f]), byFlange[f])
		}
	}
	// У сварщика до приёмки партии — ни одной задачи процесса: всё, что было
	// в истории, сварено (Д-85).
	if ts := s.tasks("W21"); len(ts) > 0 {
		s.t.Errorf("после истории у W21 открытых задач %d — по шагам, выполненным в истории: %s", len(ts), taskSummary(ts))
	}
	for _, x := range s.tasks("FOR-WC") {
		if str(x, "location_id") == "WS-MC" {
			s.t.Errorf("после истории у мастера FOR-WC задача чужого цеха WS-MC: «%s»", str(x, "title"))
		}
	}
}

// checkW21AfterReceive — после приёмки партии у W21 открыты только «Начать» по
// Ф-001…Ф-003 (задачи истории не висят).
func (s *showSystem) checkW21AfterReceive(st *showStop) {
	s.t.Helper()
	var extra []map[string]any
	got := map[string]bool{}
	for _, x := range s.tasks("W21") {
		f := flangeOf(str(x, "item_label") + " " + str(x, "title"))
		if str(x, "operation_id") == "process.operation.start" && slices.Contains([]string{"F-001", "F-002", "F-003"}, f) && !got[f] {
			got[f] = true
			continue
		}
		extra = append(extra, x)
	}
	if len(got) != 3 || len(extra) > 0 {
		s.t.Errorf("%s: после приёмки партии у W21 ждали только «Начать» Ф-001…Ф-003; есть %v, лишние %s", st, got, taskSummary(extra))
	}
}

// itemDiag — что знает система об изделии остановки (для сообщения о падении):
// паспорт, текущее предъявление, последние записи журнала изделия.
func (s *showSystem) itemDiag(item string) string {
	s.t.Helper()
	if item == "" {
		return ""
	}
	var b strings.Builder
	pp := s.passport("ADM-01", item)
	fmt.Fprintf(&b, "изделие %s «%s»: шаг %s, состояние %v, НС %v", item, str(pp, "label"), str(pp, "step_key"), pp["status"], pp["nonconformities"])
	code, pr := s.call("INS-01", "nonconformity.presentation.read", map[string]string{"item_id": item}, nil)
	fmt.Fprintf(&b, "\n  предъявление (INS-01): HTTP %d %v", code, pr["presentation"])
	if code != 200 {
		fmt.Fprintf(&b, " %v", pr)
	}
	es := list(s.read("ADM-01", "journal.entry.list", map[string]string{"item_id": item, "order": "desc", "limit": "40"}), "items")
	for _, e := range es {
		d, _ := e["data"].(map[string]any)
		fmt.Fprintf(&b, "\n  %s %s %s %s %s", str(e, "seq"), str(e, "occurred_at"), str(e, "event_type"), str(d, "step_key"), str(d, "outcome"))
		for _, k := range []string{"operation_run_id", "operation_code", "equipment_id", "station_id", "operator_id", "completion", "phase", "precondition", "reason", "method", "observation_quality_bp", "processing_state", "kind", "title", "rule_id", "status"} {
			if v := str(d, k); v != "" {
				fmt.Fprintf(&b, " %s=%s", k, v)
			}
		}
	}
	return b.String()
}
