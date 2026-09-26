package casbin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"go.yaml.in/yaml/v3"

	"ant/internal/application/access"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	accessdom "ant/internal/domain/access"
	"ant/internal/infrastructure/security/identity"
)

// repo — корень репозитория от каталога пакета.
const repo = "../../../../.."

func seed(t *testing.T) accessdom.Policy {
	t.Helper()
	s, err := identity.LoadSeed(os.DirFS(repo))
	if err != nil {
		t.Fatal(err)
	}
	return accessdom.FromSeed(s)
}

// catalogActions — каталог операций из contracts/openapi.yaml (x-ant-action).
func catalogActions(t *testing.T) []platform.Action {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repo, "contracts/openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var o struct {
		Paths map[string]map[string]struct {
			X *struct {
				ID    string `yaml:"id"`
				Class string `yaml:"class"`
			} `yaml:"x-ant-action"`
		} `yaml:"paths"`
	}
	if err := yaml.Unmarshal(b, &o); err != nil {
		t.Fatal(err)
	}
	var out []platform.Action
	for _, ms := range o.Paths {
		for _, op := range ms {
			if op.X != nil {
				out = append(out, platform.Action{ID: op.X.ID, Class: platform.Class(op.X.Class), Owner: strings.SplitN(op.X.ID, ".", 2)[0]})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	if len(out) < 100 {
		t.Fatalf("каталог операций: %d", len(out))
	}
	return out
}

func allowed(t *testing.T, ac *AccessControl, person, scope string, act platform.Action, at time.Time) bool {
	t.Helper()
	d, err := ac.Enforce(context.Background(), access.Request{Principal: platform.Principal{PersonID: person}, Action: act, Scope: scope, At: at})
	if err != nil {
		t.Fatal(err)
	}
	return d.Allowed
}

var cmd = func(id string) platform.Action { return platform.Action{ID: id, Class: platform.ClassRecord} }

// AD-15: встроенные timeMatch, условия связей, eval и Explain (внешний ИИ)
// запрещены — ни в модели, ни в коде адаптера.
func TestForbiddenBuiltins(t *testing.T) {
	for _, bad := range []string{"timeMatch", "eval(", "keyMatch", "regexMatch"} {
		if strings.Contains(Model, bad) {
			t.Errorf("модель использует %s", bad)
		}
	}
	files, _ := filepath.Glob("*.go")
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		// Код без комментариев.
		var code strings.Builder
		for _, line := range strings.Split(string(b), "\n") {
			if i := strings.Index(line, "//"); i >= 0 {
				line = line[:i]
			}
			code.WriteString(line + "\n")
		}
		for _, bad := range []string{"LinkConditionFunc", "timeMatch", "TimeMatch", "Explain(", "EnableAutoSave(true)", "\"eval\""} {
			if strings.Contains(code.String(), bad) {
				t.Errorf("%s: %s запрещено (AD-15)", f, bad)
			}
		}
	}
	// Casbin не меняет политику: только записями журнала.
	e, err := NewEnforcer(seed(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.AddPolicy("performer", "access.policy.grant"); err == nil {
		t.Fatal("запись политики через Casbin")
	}
}

// FR-78, FR-85: исполнитель поста А не может выполнить действие поста Б.
func TestPostAIsNotPostB(t *testing.T) {
	pol := seed(t)
	places, err := identity.LoadPlaces(os.DirFS(repo))
	if err != nil {
		t.Fatal(err)
	}
	// Сварщик назначен на пост сварки 1 записью политики (роль в области рабочего места).
	d, _ := json.Marshal(ev.PolicyRoleAssignedV1{PersonID: "W-POST1", RoleID: "performer", Scope: places["WP-WELD-1"]})
	_ = pol.Apply(accessdom.Record{Seq: 1, Type: string(catalog.PolicyRoleAssigned), Data: d})
	pol.Persons = append(pol.Persons, accessdom.Person{ID: "W-POST1", Name: "Сварщик поста 1"})
	ac := New(access.StaticPolicy(pol))
	confirm := cmd("access.operator.confirm_step")
	at := time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)
	if !allowed(t, ac, "W-POST1", places["WP-WELD-1"], confirm, at) {
		t.Fatal("свой пост")
	}
	for _, other := range []string{"WP-WELD-2", "WP-CNC-1", "WP-ASM-1"} {
		if allowed(t, ac, "W-POST1", places[other], confirm, at) {
			t.Errorf("чужой пост %s разрешён", other)
		}
	}
	// W21 затравки — роль в области сварочного цеха: оба поста сварки, но не ЧПУ.
	if !allowed(t, ac, "W21", places["WP-WELD-2"], confirm, at) || allowed(t, ac, "W21", places["WP-CNC-1"], confirm, at) {
		t.Fatal("область цеха")
	}
	// Неизвестное рабочее место не входит ни в одну область.
	if allowed(t, ac, "W21", "?", confirm, at) {
		t.Fatal("неизвестное место")
	}
}

// Сроки полномочий — атрибут: доменное время запроса против срока назначения (AD-15, AD-37).
func TestValidity(t *testing.T) {
	pol := seed(t)
	from := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	until := ev.Timestamp(from.AddDate(0, 1, 0))
	d, _ := json.Marshal(ev.PolicyRoleAssignedV1{PersonID: "TMP-1", RoleID: "technologist", Scope: "ent01", ValidFrom: ev.Timestamp(from), ValidUntil: &until})
	_ = pol.Apply(accessdom.Record{Seq: 5, Type: string(catalog.PolicyRoleAssigned), Data: d})
	ac := New(access.StaticPolicy(pol))
	conclude := platform.Action{ID: "analysis.cause.conclude", Class: platform.ClassIrreversible}
	if allowed(t, ac, "TMP-1", "", conclude, from.Add(-time.Hour)) || !allowed(t, ac, "TMP-1", "", conclude, from.Add(time.Hour)) ||
		allowed(t, ac, "TMP-1", "", conclude, until.Time()) {
		t.Fatal("срок полномочия")
	}
	if seq, _ := ac.PolicySeq(context.Background()); seq != 5 {
		t.Fatalf("policy_seq %d", seq)
	}
}

// Генерируемый тест «чужая операция → отказ» (FR-85, AD-15): для каждой роли
// политики и каждой операции каталога решение Casbin совпадает с тем, что
// роль (с наследованием) явно разрешает шаблонами политики; каждая роль
// получает отказ хотя бы на одну команду.
func TestForeignOperationDenied(t *testing.T) {
	pol := seed(t)
	acts := catalogActions(t)
	h := pol.Hierarchy()
	for i, r := range pol.Roles {
		person := "GEN-" + r.ID
		pol.Persons = append(pol.Persons, accessdom.Person{ID: person})
		pol.Assignments = append(pol.Assignments, accessdom.Assignment{PersonID: person, RoleID: r.ID, Scope: "ent01"})
		_ = i
	}
	ac := New(access.StaticPolicy(pol))
	at := time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)
	for _, r := range pol.Roles {
		var patterns []string
		for _, x := range h.Closure(r.ID) {
			role, _ := pol.Role(x)
			patterns = append(patterns, role.Actions...)
		}
		denied := 0
		for _, act := range acts {
			want := false
			for _, p := range patterns {
				if accessdom.ActionMatches(act.ID, p) || accessdom.ActionMatches(accessdom.ReadAlias(act.ID, string(act.Class)), p) {
					want = true
				}
			}
			got := allowed(t, ac, "GEN-"+r.ID, "", act, at)
			if got != want {
				t.Errorf("роль %s, операция %s: Casbin %v, политика %v", r.ID, act.ID, got, want)
			}
			if !got && act.Class != platform.ClassRead {
				denied++
			}
		}
		if denied == 0 {
			t.Errorf("роль %s: нет ни одной чужой команды", r.ID)
		}
	}
	// Без сеанса — только приём фактов устройств.
	for _, act := range acts {
		got := allowed(t, ac, "", "", act, at)
		want := act.ID == "ingest.batch.submit" || act.ID == "ingest.event.submit" || act.ID == "materials.material.upload"
		if got != want {
			t.Errorf("без сеанса %s: %v", act.ID, got)
		}
	}
}

// Явные отказы ролей кейса (не выводятся из той же функции сопоставления).
func TestCaseRoleRefusals(t *testing.T) {
	ac := New(access.StaticPolicy(seed(t)))
	at := time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)
	for _, c := range []struct {
		person, action string
		ok             bool
	}{
		{"INS-01", "nonconformity.presentation.resolve", true},
		{"INS-01", "analysis.scope.narrow", false}, // сценарий S05: сужать круг может только технолог
		{"INS-01", "process.version.activate", false},
		{"TEC-01", "analysis.scope.narrow", true},
		{"TEC-01", "access.policy.grant", false},
		{"PM-01", "process.version.activate", true},
		{"PM-01", "nonconformity.presentation.resolve", false},
		{"ADM-01", "access.account.activate", true},
		{"ADM-01", "nonconformity.disposition.set", false},
		{"AUD-01", "access.audit.set_parameters", true},
		{"AUD-01", "simulation.run.start", false},
		{"HQC-01", "nonconformity.presentation.resolve", true}, // наследник контролёра
		{"HQC-01", "vision.passport.reinstate", true},
		{"INS-01", "vision.passport.reinstate", false},
		{"W21", "process.operation.start", true},
		{"W21", "nonconformity.presentation.resolve", false},
		{"CR-71", "documents.document.sign", true},
		{"CR-71", "process.operation.start", false},
	} {
		if got := allowed(t, ac, c.person, "", cmd(c.action), at); got != c.ok {
			t.Errorf("%s %s: %v, ожидалось %v", c.person, c.action, got, c.ok)
		}
	}
	read := platform.Action{ID: "quality.signal.list", Class: platform.ClassRead}
	if !allowed(t, ac, "W21", "", read, at) || allowed(t, ac, "CR-71", "", read, at) {
		t.Fatal("чтение по второму имени ‹модуль›.*.read")
	}
}
