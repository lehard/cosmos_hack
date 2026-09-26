package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
	"uuid"

	accessapp "ant/internal/application/access"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	accessdom "ant/internal/domain/access"
	"ant/internal/infrastructure/security/casbin"
	"ant/internal/infrastructure/security/identity"
	"ant/internal/infrastructure/transport/httpapi"
)

// Эпик 08, сквозные проверки барьеров на HTTP API в памяти: журнал политики,
// учётные данные и сеансы — поддельные порты, вычислитель — настоящий Casbin
// над проекцией политики, вход — настоящий identity.Local.

// memJournal — журнал политики в памяти: DecisionWriter пишет, PolicyLog читает.
type memJournal struct {
	mu   sync.Mutex
	recs []accessdom.Record
}

func (j *memJournal) Write(_ context.Context, b accessapp.Batch) (platform.Receipt, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	for _, r := range b.Records {
		data, err := json.Marshal(r.Data)
		if err != nil {
			return platform.Receipt{}, err
		}
		j.recs = append(j.recs, accessdom.Record{Seq: int64(100 + len(j.recs)), Type: string(r.Type), Data: data, OccurredAt: b.OccurredAt})
	}
	return platform.Receipt{CommandID: b.Meta.CommandID, Seq: int64(99 + len(j.recs))}, nil
}

func (j *memJournal) Since(_ context.Context, after int64) ([]accessdom.Record, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	var out []accessdom.Record
	for _, r := range j.recs {
		if r.Seq > after {
			out = append(out, r)
		}
	}
	return out, nil
}

func (j *memJournal) Head() int64 { return 0 }

type memCreds struct {
	mu sync.Mutex
	m  map[string]accessapp.Credential
}

func (s *memCreds) Get(_ context.Context, l string) (accessapp.Credential, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.m[l]
	return c, ok, nil
}

func (s *memCreds) Create(_ context.Context, c accessapp.Credential) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.m[c.Login]; ok {
		return accessapp.ErrLoginTaken
	}
	s.m[c.Login] = c
	return nil
}

func (s *memCreds) Activate(_ context.Context, l, p string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.m[l]
	c.Status, c.PersonID = accessapp.AccountActive, p
	s.m[l] = c
	return nil
}

func (s *memCreds) Failed(_ context.Context, l string, _ time.Time, _ int, _ time.Duration) (accessapp.Credential, error) {
	return s.m[l], nil
}

func (s *memCreds) Succeeded(context.Context, string) error { return nil }

func (s *memCreds) List(context.Context) ([]accessapp.Credential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []accessapp.Credential
	for _, c := range s.m {
		out = append(out, c)
	}
	slices.SortFunc(out, func(a, b accessapp.Credential) int { return map[bool]int{true: -1, false: 1}[a.Login < b.Login] })
	return out, nil
}

type noEvents struct{}

func (noEvents) AuthFailed(context.Context, accessapp.AuthFailure) error { return nil }
func (noEvents) AccessDenied(context.Context, accessapp.Denial) error    { return nil }

// testAPI — API с настоящими барьерами (вход, Casbin) в режиме mode.
func testAPI(t *testing.T, mode platform.Mode) (http.Handler, *httpapi.API, *accessBundle) {
	t.Helper()
	dir, seed, places, err := seedFS()
	if err != nil {
		t.Fatal(err)
	}
	j := &memJournal{}
	proj := accessapp.NewProjection(seed, j)
	proj.Every = 0
	hasher := identity.Argon2id{Memory: 64, Time: 1, Threads: 1, KeyLen: 32, SaltLen: 16}
	creds := &memCreds{m: map[string]accessapp.Credential{}}
	now := time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)
	idp, err := identity.NewLocal(dir, proj, creds, hasher, noEvents{}, nil, identity.LocalOptions{Personas: true,
		At: func(context.Context) time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	b := &accessBundle{directory: dir, identity: idp, control: casbin.New(proj), policy: proj, places: places, events: noEvents{},
		creds: creds, hasher: hasher, decisions: j, now: func(context.Context) (time.Time, error) { return now, nil }}
	mux := http.NewServeMux()
	a := buildAPI(mux, apiOptions{mode: mode, access: b})
	return mux, a, b
}

type call struct {
	method, path, persona, cookie string
	body                          any
}

func do(t *testing.T, h http.Handler, c call) (int, map[string]any, *http.Response) {
	t.Helper()
	var rd *bytes.Reader
	if c.body != nil {
		b, _ := json.Marshal(c.body)
		rd = bytes.NewReader(b)
	} else {
		rd = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(c.method, c.path, rd)
	if c.body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.persona != "" {
		req.Header.Set(httpapi.DemoPersonaHeader, c.persona)
	}
	if c.cookie != "" {
		req.AddCookie(&http.Cookie{Name: httpapi.SessionCookie, Value: c.cookie})
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out, rec.Result()
}

func header(extra map[string]any) map[string]any {
	m := map[string]any{"command_id": uuid.NewV7().String(), "basis_seq": 0, "policy_seq": 0}
	for k, v := range extra {
		m[k] = v
	}
	return m
}

func permissions(t *testing.T, h http.Handler, persona, cookie string) []string {
	t.Helper()
	code, out, _ := do(t, h, call{method: "GET", path: "/api/v1/permissions", persona: persona, cookie: cookie})
	if code != http.StatusOK {
		t.Fatalf("%s: /permissions %d %v", persona, code, out)
	}
	var ids []string
	for _, it := range out["items"].([]any) {
		ids = append(ids, it.(map[string]any)["action"].(string))
	}
	return ids
}

// AD-15 «Для фронтенда»: действие в списке ⇔ сервер его разрешит — для
// персон демо и каждой операции каталога тем же декоратором.
func TestPermissionListEqualsServerDecision(t *testing.T) {
	h, a, b := testAPI(t, platform.ModeLive)
	gate := accessapp.NewGate(b.control, nil, a.Actions)
	gate.Places = b.places
	ctx := context.Background()
	for _, persona := range []string{"INS-01", "PM-01", "TEC-01", "ADM-01", "AUD-01", "HQC-01", "W21", "FOR-WC", "CR-71"} {
		list := permissions(t, h, persona, "")
		p, err := b.identity.Identify(ctx, accessapp.Credentials{DemoPersona: persona})
		if err != nil || p.Anonymous() {
			t.Fatal(persona, err)
		}
		for _, act := range a.Actions() {
			if act.Anonymous {
				continue
			}
			ok := gate.Authorize(ctx, p, act, platform.ObjectRef{}, nil) == nil
			if ok != slices.Contains(list, act.ID) {
				t.Errorf("%s %s: сервер %v, в списке %v", persona, act.ID, ok, !ok)
			}
		}
	}
}

// Экран входа — только персоны показа (demo_login политики, решение
// пользователя); скрытые остаются в политике и действуют заголовком.
func TestPersonaLoginList(t *testing.T) {
	h, _, _ := testAPI(t, platform.ModeFixtures)
	code, out, _ := do(t, h, call{method: "GET", path: "/api/v1/auth/personas"})
	if code != http.StatusOK {
		t.Fatalf("персоны: %d %v", code, out)
	}
	var ids []string
	items, _ := out["items"].([]any)
	for _, it := range items {
		ids = append(ids, it.(map[string]any)["id"].(string))
	}
	if want := "INS-01 HQC-01 FOR-WC TEC-01 CWL-01 PM-01 ADM-01 AUD-01 W21"; strings.Join(ids, " ") != want {
		t.Fatalf("персоны экрана входа: %v, want %s", ids, want)
	}
	if code, desk, _ := do(t, h, call{method: "GET", path: "/api/v1/desk", persona: "W22"}); code != http.StatusOK || desk["role"] == "" {
		t.Fatalf("скрытая персона W22: стол %d %v", code, desk)
	}
}

// Столы и кнопки демо-персон: чтения своего стола и свои команды — в списке прав.
func TestPersonaDesksAndButtons(t *testing.T) {
	h, _, _ := testAPI(t, platform.ModeFixtures)
	shell := []string{"access.session.read", "access.desk.read", "access.permission.list", "notifications.summary.read", "security.integrity.read", "journal.stream.subscribe", "item.item.lookup"}
	want := map[string][]string{
		"INS-01": {"nonconformity.queue.list", "item.passport.read", "nonconformity.card.read", "nonconformity.presentation.resolve", "nonconformity.signal.reject", "nonconformity.item.isolate"},
		"PM-01": {"process.live_map.read", "analytics.tile.list", "notifications.attention.list", "access.workplace.list", "process.version.list", "process.version.activate", "journal.timeline.read",
			"access.workplace.read", "access.workplace.history", "access.person.card"},
		"TEC-01": {"analysis.incident.list", "analysis.risk_scope.read", "analysis.scope.narrow", "analysis.cause.conclude", "analysis.hypothesis.list", "process.version.diff"},
		"ADM-01": {"simulation.run.start", "simulation.scenario.list", "ingest.quarantine.list", "ops.health.read", "access.person.list", "access.account.activate", "access.person.register", "journal.entry.list"},
		"AUD-01": {"security.critical_action.list", "access.grant.list", "security.event.list", "security.verifier_report.list", "access.audit.set_parameters"},
	}
	for persona, acts := range want {
		code, desk, _ := do(t, h, call{method: "GET", path: "/api/v1/desk", persona: persona})
		if code != http.StatusOK || desk["role"] == "" {
			t.Fatalf("%s: стол %d %v", persona, code, desk)
		}
		list := permissions(t, h, persona, "")
		for _, a := range append(slices.Clone(shell), acts...) {
			if !slices.Contains(list, a) {
				t.Errorf("%s: нет действия своего стола %s", persona, a)
			}
		}
	}
	// UI-16: окна «Пост» и «Сотрудник» у руководителя — карточки без учётной записи;
	// мастер видит посты только своей области (барьер 3).
	for _, path := range []string{"/api/v1/workplaces/WP-WELD-2", "/api/v1/workplaces/WP-WELD-2/history", "/api/v1/persons/W21/card"} {
		code, out, _ := do(t, h, call{method: "GET", path: path, persona: "PM-01"})
		if code != http.StatusOK {
			t.Fatalf("руководитель %s: %d %v", path, code, out)
		}
		if _, ok := out["login"]; ok {
			t.Fatalf("%s: логин в карточке: %v", path, out)
		}
	}
	if code, out, _ := do(t, h, call{method: "GET", path: "/api/v1/workplaces/WP-WELD-2", persona: "FOR-WC"}); code != http.StatusOK || out["workplace_id"] != "WP-WELD-2" {
		t.Fatalf("мастер — свой пост: %d %v", code, out)
	}
	if code, out, _ := do(t, h, call{method: "GET", path: "/api/v1/workplaces/WP-CNC-1", persona: "FOR-WC"}); code != http.StatusForbidden {
		t.Fatalf("мастер — пост чужого цеха: %d %v", code, out)
	}
	// Заготовки отвечают персонам по их правам: очередь контролёра — 200, чужая команда — 403.
	if code, out, _ := do(t, h, call{method: "GET", path: "/api/v1/decision-queue", persona: "INS-01"}); code != http.StatusOK {
		t.Fatalf("очередь контролёра: %d %v", code, out)
	}
	if code, out, _ := do(t, h, call{method: "POST", path: "/api/v1/process/versions/v1/activate", persona: "INS-01", body: header(map[string]any{"route_closed_event_id": uuid.NewV7().String()})}); code != http.StatusForbidden || out["code"] != "access.forbidden" {
		t.Fatalf("чужая команда: %d %v", code, out)
	}
}

// FR-78, FR-85: исполнитель поста А не может действие поста Б; без сеанса — 401.
func TestWorkplaceScopeHTTP(t *testing.T) {
	h, _, _ := testAPI(t, platform.ModeLive)
	body := header(map[string]any{"step_key": "welding.weld"})
	if code, out, _ := do(t, h, call{method: "POST", path: "/api/v1/workplaces/WP-CNC-1/steps/confirm", persona: "W21", body: body}); code != http.StatusForbidden || out["code"] != "access.wrong_workplace" {
		t.Fatalf("чужой пост: %d %v", code, out)
	}
	// Свой пост проходит барьер (дальше — реализация порта: в live без эпика 37 — 501).
	if code, out, _ := do(t, h, call{method: "POST", path: "/api/v1/workplaces/WP-WELD-1/steps/confirm", persona: "W21", body: body}); code == http.StatusForbidden || code == http.StatusUnauthorized {
		t.Fatalf("свой пост: %d %v", code, out)
	}
	if code, out, _ := do(t, h, call{method: "POST", path: "/api/v1/workplaces/WP-WELD-1/steps/confirm", body: body}); code != http.StatusUnauthorized || out["code"] != "access.unauthenticated" {
		t.Fatalf("без сеанса: %d %v", code, out)
	}
	if code, out, _ := do(t, h, call{method: "GET", path: "/api/v1/items"}); code != http.StatusUnauthorized {
		t.Fatalf("чтение без сеанса: %d %v", code, out)
	}
}

// FR-128: пользователь, добавленный администратором, входит по логину и паролю
// и видит стол своей роли; заявка на регистрацию — после активации.
func TestAdminAddsUserWhoLogsIn(t *testing.T) {
	h, _, _ := testAPI(t, platform.ModeLive)
	login := func(l, p string) (int, string, map[string]any) {
		code, out, res := do(t, h, call{method: "POST", path: "/api/v1/auth/session", body: map[string]any{"login": l, "password": p}})
		for _, c := range res.Cookies() {
			if c.Name == httpapi.SessionCookie {
				return code, c.Value, out
			}
		}
		return code, "", out
	}
	// Не администратор не заводит сотрудников.
	if code, out, _ := do(t, h, call{method: "POST", path: "/api/v1/persons", persona: "INS-01", body: header(map[string]any{"person_id": "U-sidorov", "display_name": "Сидоров"})}); code != http.StatusForbidden {
		t.Fatalf("контролёр заводит сотрудника: %d %v", code, out)
	}
	if code, out, _ := do(t, h, call{method: "POST", path: "/api/v1/persons", persona: "ADM-01", body: header(map[string]any{"person_id": "U-sidorov", "display_name": "Сидоров"})}); code != http.StatusOK {
		t.Fatalf("заведение сотрудника: %d %v", code, out)
	}
	if code, out, _ := do(t, h, call{method: "POST", path: "/api/v1/persons/U-sidorov/account", persona: "ADM-01",
		body: header(map[string]any{"login": "sidorov", "password": "пароль-сидорова", "initial_role_id": "technologist"})}); code != http.StatusOK {
		t.Fatalf("активация: %d %v", code, out)
	}
	if code, _, out := login("sidorov", "не-тот"); code != http.StatusUnauthorized || out["code"] != "access.login_failed" {
		t.Fatalf("неверный пароль: %d %v", code, out)
	}
	code, cookie, out := login("sidorov", "пароль-сидорова")
	if code != http.StatusCreated || cookie == "" || out["role"].(map[string]any)["id"] != "technologist" || out["demo"] != false {
		t.Fatalf("вход: %d %q %v", code, cookie, out)
	}
	if code, desk, _ := do(t, h, call{method: "GET", path: "/api/v1/desk", cookie: cookie}); code != http.StatusOK || desk["role"] != "technologist" {
		t.Fatalf("стол технолога: %d %v", code, desk)
	}
	if list := permissions(t, h, "", cookie); !slices.Contains(list, "analysis.cause.conclude") || slices.Contains(list, "access.account.activate") {
		t.Fatalf("права технолога: %v", list)
	}
	if code, _, _ := do(t, h, call{method: "DELETE", path: "/api/v1/auth/session", cookie: cookie}); code >= 300 {
		t.Fatalf("выход: %d", code)
	}
	if code, _, _ := do(t, h, call{method: "GET", path: "/api/v1/desk", cookie: cookie}); code != http.StatusUnauthorized {
		t.Fatalf("после выхода: %d", code)
	}

	// Заявка на регистрацию → вход до активации — account_pending → активация администратором.
	if code, out, _ := do(t, h, call{method: "POST", path: "/api/v1/auth/registration", body: map[string]any{"login": "kuznetsov", "password": "пароль-кузнецова", "display_name": "Кузнецов"}}); code != http.StatusAccepted || out["person_id"] != "U-kuznetsov" {
		t.Fatalf("заявка: %d %v", code, out)
	}
	if code, _, out := login("kuznetsov", "пароль-кузнецова"); code != http.StatusForbidden || out["code"] != "access.account_pending" {
		t.Fatalf("до активации: %d %v", code, out)
	}
	code, list, _ := do(t, h, call{method: "GET", path: "/api/v1/persons", persona: "ADM-01"})
	if code != http.StatusOK || !pending(list, "U-kuznetsov") {
		t.Fatalf("заявка в списке сотрудников: %d %v", code, list)
	}
	if code, out, _ := do(t, h, call{method: "POST", path: "/api/v1/persons/U-kuznetsov/account", persona: "ADM-01",
		body: header(map[string]any{"login": "kuznetsov", "initial_role_id": "quality_inspector", "scope": "ent01/b1/wc"})}); code != http.StatusOK {
		t.Fatalf("активация заявки: %d %v", code, out)
	}
	code, cookie, out = login("kuznetsov", "пароль-кузнецова")
	if code != http.StatusCreated || out["scope"] != "ent01/b1/wc" {
		t.Fatalf("вход после активации: %d %v", code, out)
	}
	if code, desk, _ := do(t, h, call{method: "GET", path: "/api/v1/desk", cookie: cookie}); code != http.StatusOK || desk["role"] != "quality_inspector" {
		t.Fatalf("стол контролёра: %d %v", code, desk)
	}
	// Выдать роль себе нельзя (вторая подпись — эпик 26).
	if code, out, _ := do(t, h, call{method: "POST", path: "/api/v1/persons/ADM-01/account", persona: "ADM-01",
		body: header(map[string]any{"login": "adm01", "password": "пароль-админа", "initial_role_id": "security_auditor"})}); code != http.StatusForbidden || out["code"] != "access.self_grant" {
		t.Fatalf("выдача себе: %d %v", code, out)
	}
	_ = catalog.PolicyRoleAssigned
}

func pending(list map[string]any, person string) bool {
	for _, it := range list["items"].([]any) {
		m := it.(map[string]any)
		if m["person_id"] == person && m["account_status"] == "pending" {
			return true
		}
	}
	return false
}
