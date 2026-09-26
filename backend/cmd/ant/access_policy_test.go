package main

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"testing"
	"time"

	accessapp "ant/internal/application/access"
	engineapp "ant/internal/application/engine"
	"ant/internal/application/engine/enginemem"
	"ant/internal/application/platform"
	signingapp "ant/internal/application/signing"
	accessdom "ant/internal/domain/access"
	domdocs "ant/internal/domain/documents"
	dj "ant/internal/domain/journal"
	"ant/internal/infrastructure/security/casbin"
	"ant/internal/infrastructure/security/identity"
	accessstore "ant/internal/infrastructure/storage/access"
	storagedocs "ant/internal/infrastructure/storage/documents"
)

// Эпик 26 на HTTP API с настоящими барьерами (вход демо-персоной, Casbin над
// проекцией политики, декоратор с проверкой policy_seq): выдача себе, редкие
// подписанты, команда по устаревшей политике на двух копиях api.

// Порт полномочий signing реализует access (вместо статической таблицы эпика 27).
var _ signingapp.Authorities = accessapp.PolicyAuthorities{}

var now26 = time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)

// Администратор не может расширить себе права без Аудитора ИБ (AD-11, FR-85,
// PRD §11.16); Аудитор ИБ сам прав не выдаёт (домен 3 — только чтение).
func TestAdminCannotGrantSelfHTTP(t *testing.T) {
	h, _, _ := testAPI(t, platform.ModeLive)
	grant := func(persona string, body map[string]any) (int, map[string]any) {
		code, out, _ := do(t, h, call{method: "POST", path: "/api/v1/grants", persona: persona, body: header(body)})
		return code, out
	}
	code, out := grant("ADM-01", map[string]any{"person_id": "ADM-01", "kind": "role", "role_id": "technologist", "scope": "ent01", "valid_from": "2026-09-26T08:00:00Z"})
	if code != http.StatusForbidden || out["code"] != "access.self_grant" {
		t.Fatalf("выдача себе: %d %v", code, out)
	}
	if acts, _ := out["allowed_actions"].([]any); !slices.Contains(acts, any("documents.version.request")) {
		t.Fatalf("«Запросить решение» в отказе: %v", out)
	}
	// Полномочие второй подписи Аудитора себе — тоже нет.
	code, out = grant("ADM-01", map[string]any{"person_id": "ADM-01", "kind": "authority", "authority_id": "second_signature_audit", "scope": "ent01", "valid_from": "2026-09-26T08:00:00Z"})
	if code != http.StatusForbidden || out["code"] != "access.self_grant" {
		t.Fatalf("полномочие себе: %d %v", code, out)
	}
	// Роль администратора другому — привилегия: нужна подпись Аудитора ИБ.
	code, out = grant("ADM-01", map[string]any{"person_id": "TEC-01", "kind": "role", "role_id": "administrator", "scope": "ent01", "valid_from": "2026-09-26T08:00:00Z"})
	if code != http.StatusForbidden || out["code"] != "access.signature_required" {
		t.Fatalf("привилегия другому: %d %v", code, out)
	}
	// Аудитор ИБ выдавать права не может.
	code, out = grant("AUD-01", map[string]any{"person_id": "AUD-01", "kind": "role", "role_id": "administrator", "scope": "ent01", "valid_from": "2026-09-26T08:00:00Z"})
	if code != http.StatusForbidden || out["code"] != "access.forbidden" {
		t.Fatalf("Аудитор ИБ: %d %v", code, out)
	}
	// Обычная роль другому — одной подписью.
	code, out = grant("ADM-01", map[string]any{"person_id": "W22", "kind": "role", "role_id": "technologist", "scope": "ent01/b1/wc", "valid_from": "2026-09-26T08:00:00Z"})
	if code != http.StatusOK {
		t.Fatalf("обычная выдача: %d %v", code, out)
	}
	// Чья подпись нужна — для «Запросить решение».
	code, out, _ = do(t, h, call{method: "GET", path: "/api/v1/grants/assessment?person_id=ADM-01&kind=role&subject_id=technologist", persona: "ADM-01"})
	stages, _ := out["stages"].([]any)
	if code != http.StatusOK || out["self_grant"] != true || out["second_signer"] != "Аудитор ИБ" || len(stages) != 1 || out["template"] != "policy-grant@1" {
		t.Fatalf("оценка выдачи: %d %v", code, out)
	}
	if c := stages[0].(map[string]any)["candidates"].([]any); len(c) != 1 || c[0] != "AUD-01" {
		t.Fatalf("кандидаты второй подписи: %v", c)
	}
	// Объяснение прав своим кодом.
	code, out, _ = do(t, h, call{method: "GET", path: "/api/v1/permissions/explain?action=access.policy.grant", persona: "AUD-01"})
	if code != http.StatusOK || out["allowed"] != false || out["who_can"] == nil {
		t.Fatalf("объяснение: %d %v", code, out)
	}
}

// Представитель заказчика подписывает разрешение на отклонение с карточки без
// доступа к остальным экранам (FR-136): в его правах — карточка и подпись
// документа, прочих экранов нет; в маршруте разрешения на отклонение (каталог
// документов спайна) он — кандидат этапа представителя заказчика.
func TestCustomerRepresentativeCardOnlyHTTP(t *testing.T) {
	h, _, b := testAPI(t, platform.ModeLive)
	list := permissions(t, h, "CR-71", "")
	for _, a := range []string{"documents.decision_card.read", "documents.request.list", "documents.signature.record", "documents.document.sign", "access.desk.read"} {
		if !slices.Contains(list, a) {
			t.Errorf("нет действия карточки %s", a)
		}
	}
	for _, a := range []string{"nonconformity.queue.list", "nonconformity.card.read", "process.live_map.read", "analysis.incident.list", "analytics.tile.list",
		"item.item.list", "journal.timeline.read", "access.person.list", "documents.document.list", "quality.signal.list"} {
		if slices.Contains(list, a) {
			t.Errorf("редкому подписанту доступен чужой экран %s", a)
		}
	}
	if code, _, _ := do(t, h, call{method: "GET", path: "/api/v1/decision-queue", persona: "CR-71"}); code != http.StatusForbidden {
		t.Fatalf("очередь решений: %d", code)
	}
	if code, desk, _ := do(t, h, call{method: "GET", path: "/api/v1/desk", persona: "CR-71"}); code != http.StatusOK || desk["role"] != "customer_representative" {
		t.Fatalf("стол карточки: %d %v", code, desk)
	}
	// Порт полномочий signing над проекцией: вторая подпись Аудитора ИБ и сферы сотрудников.
	auth, ok := b.authorities()
	if !ok {
		t.Fatal("порт полномочий")
	}
	if has, err := auth.Has(context.Background(), "AUD-01", "second_signature_audit", 0); err != nil || !has || auth.Domain(context.Background(), "HQC-01") != "qc" ||
		auth.Domain(context.Background(), "ADM-01") != "admin" || auth.Domain(context.Background(), "W21") != "production" {
		t.Fatal("полномочия и сферы", err)
	}
	pol, err := b.policy.Policy(context.Background())
	if err != nil || !pol.CardOnly("CR-71", now26) || pol.CardOnly("HQC-01", now26) {
		t.Fatal("редкий подписант по политике", err)
	}
	yes := true
	route := []accessdom.RouteStage{
		{Stage: 1, Title: "Мастер цеха-виновника", Role: "site_foreman", AuthorityID: "site_foreman", Quorum: "one", SignatureLevel: 2},
		{Stage: 2, Title: "Технолог", Role: "technologist", AuthorityID: "concession_approval", Quorum: "one", SignatureLevel: 2},
		{Stage: 3, Title: "Начальник ОТК", Role: "head_of_qc", AuthorityID: "concession_approval", Quorum: "one", SignatureLevel: 2},
		{Stage: 4, Title: "Представитель заказчика", Role: "customer_representative", AuthorityID: "concession_approval", Quorum: "one", SignatureLevel: 2,
			ExternalParty: "customer_representative", When: &accessdom.StageCondition{CustomerAcceptance: &yes}},
		{Stage: 5, Title: "Держатель КД", Role: "design_authority", AuthorityID: "concession_approval", Quorum: "one", SignatureLevel: 2},
		{Stage: 6, Title: "Руководитель производства", Role: "production_manager", AuthorityID: "production_manager", Quorum: "one", SignatureLevel: 2},
	}
	st := accessdom.RequiredApprovals(route, accessdom.ApprovalContext{Decision: "use_as_is", CustomerAcceptance: true, Scope: "ent01/b1/wc", At: now26}, pol)
	if len(st) != 6 || !slices.Equal(st[3].Candidates, []string{"CR-71"}) || !slices.Equal(st[0].Candidates, []string{"FOR-WC", "HWS-WC"}) ||
		!slices.Equal(st[4].Candidates, []string{"DA-81"}) {
		t.Fatalf("маршрут разрешения на отклонение: %+v", st)
	}
}

// journalAPI — копия api над общим журналом в памяти: запись решений через
// journal.Append (проверки AD-39), проекция политики — из журнала с
// перечитыванием не чаще every.
func journalAPI(t *testing.T, j *enginemem.Journal, every time.Duration) (http.Handler, *accessapp.Projection) {
	t.Helper()
	dir, seed, places, err := seedFS()
	if err != nil {
		t.Fatal(err)
	}
	codec := &engineapp.Codec{Store: j, Sealer: enginemem.Sealer{KeyRef: "engine@1"}, KeyRef: "engine@1", DomainBuild: dj.ZeroLink.String(), Partitions: 1}
	proj := accessapp.NewProjection(seed, accessstore.PolicyLog{Journal: j, Codec: codec})
	proj.Every = every
	proj.Now = func() time.Time { return time.Now() }
	hasher := identity.Argon2id{Memory: 64, Time: 1, Threads: 1, KeyLen: 32, SaltLen: 16}
	creds := &memCreds{m: map[string]accessapp.Credential{}}
	idp, err := identity.NewLocal(dir, proj, creds, hasher, noEvents{}, nil, identity.LocalOptions{Personas: true,
		At: func(context.Context) time.Time { return now26 }})
	if err != nil {
		t.Fatal(err)
	}
	b := &accessBundle{directory: dir, identity: idp, control: casbin.New(proj), policy: proj, places: places, events: noEvents{},
		creds: creds, hasher: hasher, decisions: accessapp.JournalDecisions{Journal: j, DomainBuild: dj.ZeroLink.String()},
		now: func(context.Context) (time.Time, error) { return now26, nil }}
	mux := http.NewServeMux()
	buildAPI(mux, apiOptions{mode: platform.ModeLive, access: b})
	return mux, proj
}

// Команда по устаревшей политике — 409 journal.stale_policy (AD-39, AD-15):
// две копии api над одним журналом; копия B ещё не перечитала политику после
// отзыва роли на копии A и разрешает команду по старой версии — journal.Append
// отвергает запись в той же транзакции. Клиент с устаревшим policy_seq — тоже 409.
func TestStalePolicyCommandHTTP(t *testing.T) {
	ctx := context.Background()
	j := enginemem.New(nil)
	a, projA := journalAPI(t, j, 0)
	b, projB := journalAPI(t, j, time.Hour)
	// Журнал политики не пуст: сотрудник заведён (seq 1).
	if code, out, _ := do(t, a, call{method: "POST", path: "/api/v1/persons", persona: "ADM-01", body: header(map[string]any{"person_id": "X-1", "display_name": "Новый"})}); code != http.StatusOK {
		t.Fatalf("завести сотрудника: %d %v", code, out)
	}
	if err := projB.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if p, _ := projB.Policy(ctx); p.Seq == 0 {
		t.Fatal("копия B не прочитала журнал")
	}
	audit := header(map[string]any{"checkpoint_interval_s": 60, "checkpoint_max_gap_s": 300,
		"keeper_key_fingerprint": "streebog256:0000000000000000000000000000000000000000000000000000000000000000"})
	// Отзыв роли Аудитора ИБ на копии A.
	if code, out, _ := do(t, a, call{method: "POST", path: "/api/v1/grants/revocations", persona: "ADM-01", body: header(map[string]any{
		"person_id": "AUD-01", "kind": "role", "subject_id": "security_auditor", "effective_from": "2026-09-26T08:00:00Z", "reason": map[string]any{"text": "проверка отзыва"}})}); code != http.StatusOK {
		t.Fatalf("отзыв: %d %v", code, out)
	}
	// Копия B (устаревшая проекция) разрешает, журнал отвергает: 409.
	code, out, _ := do(t, b, call{method: "POST", path: "/api/v1/audit/parameters", persona: "AUD-01", body: audit})
	if code != http.StatusConflict || out["code"] != "journal.stale_policy" {
		t.Fatalf("устаревшая копия api: %d %v", code, out)
	}
	// Копия A (актуальная проекция) — отказ правами.
	if code, out, _ := do(t, a, call{method: "POST", path: "/api/v1/audit/parameters", persona: "AUD-01", body: audit}); code != http.StatusForbidden {
		t.Fatalf("актуальная копия: %d %v", code, out)
	}
	// Клиент с устаревшим policy_seq: выдача полномочия руководителю
	// производства меняет политику его области — команда с прежним policy_seq отвергается.
	pa, _ := projA.Policy(ctx)
	old := pa.Seq
	if code, out, _ := do(t, a, call{method: "POST", path: "/api/v1/grants", persona: "ADM-01", body: header(map[string]any{
		"person_id": "TEC-01", "kind": "role", "role_id": "production_manager", "scope": "ent01", "valid_from": "2026-09-26T08:00:00Z"})}); code != http.StatusOK {
		t.Fatalf("выдача роли: %d %v", code, out)
	}
	stale := header(map[string]any{"person_id": "X-2", "display_name": "Ещё"})
	stale["policy_seq"] = old
	if code, out, _ := do(t, a, call{method: "POST", path: "/api/v1/persons", persona: "ADM-01", body: stale}); code != http.StatusConflict || out["code"] != "journal.stale_policy" {
		t.Fatalf("устаревший policy_seq клиента: %d %v", code, out)
	}
	pa, _ = projA.Policy(ctx)
	fresh := header(map[string]any{"person_id": "X-2", "display_name": "Ещё"})
	fresh["policy_seq"] = pa.Seq
	if code, out, _ := do(t, a, call{method: "POST", path: "/api/v1/persons", persona: "ADM-01", body: fresh}); code != http.StatusOK {
		t.Fatalf("актуальный policy_seq: %d %v", code, out)
	}
	_ = accessdom.PolicyStream
}

// Маршрут документа «Выдача ролей, полномочий, клейм» — один: grant_route
// политики (оценка выдачи в access) совпадает с маршрутом шаблона policy-grant
// нормативного слоя documents (черновик документа).
func TestGrantRouteMatchesDocumentTemplate(t *testing.T) {
	_, seed, _, err := seedFS()
	if err != nil {
		t.Fatal(err)
	}
	env, err := storagedocs.SeedEnv(domdocs.VerificationDemo)
	if err != nil {
		t.Fatal(err)
	}
	tpl, ok := env.Templates.ByRef(seed.Catalog.GrantTemplate)
	if !ok {
		t.Fatalf("шаблон %s не найден в normative/documents", seed.Catalog.GrantTemplate)
	}
	a, _ := json.Marshal(seed.Catalog.GrantRoute)
	b, _ := json.Marshal(tpl.Route)
	if string(a) != string(b) || len(seed.Catalog.GrantRoute) != 3 {
		t.Fatalf("маршрут выдачи расходится:\n политика  %s\n документы %s", a, b)
	}
}
