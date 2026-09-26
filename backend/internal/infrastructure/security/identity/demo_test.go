package identity

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"ant/internal/application/access"
	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
)

// repo — корень репозитория от каталога пакета.
const repo = "../../../../.."

func directory(t *testing.T) *access.Directory {
	t.Helper()
	d, err := LoadDirectory(os.DirFS(repo))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestLoadDirectory(t *testing.T) {
	d := directory(t)
	p, ok := d.Persona("INS-01")
	if !ok || p.Role.ID != "quality_inspector" || p.Role.Title == "" || p.Scope == "" {
		t.Fatalf("INS-01: %+v", p)
	}
	desk, ok := d.DeskFor("quality_inspector")
	if !ok || len(desk.Tabs) == 0 || len(desk.Tabs[0].Slots) == 0 {
		t.Fatalf("стол контролёра: %+v", desk)
	}
	// Наследник без своего файла — стол базовой роли (AD-21).
	head, ok := d.DeskFor("head_of_qc")
	if !ok || head.TitleKey != desk.TitleKey {
		t.Fatalf("стол начальника ОТК: %+v", head)
	}
	for _, p := range d.Personas {
		// У роли каждого сотрудника — свой стол или стол базовой роли (общие
		// роли employee, staff, device_source столов не имеют).
		if _, ok := d.DeskFor(p.Role.ID); !ok {
			t.Errorf("нет стола у роли %s", p.Role.ID)
		}
	}
}

func TestDemoSession(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)
	idp, err := NewDemo(directory(t), Options{Personas: true, Key: []byte("k"), Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	ps, err := idp.DemoPersonas(ctx)
	if err != nil || len(ps) == 0 {
		t.Fatal(ps, err)
	}
	p, token, err := idp.Open(ctx, access.SessionCreate{PersonaID: "INS-01"})
	if err != nil || token == "" || p.Role != "quality_inspector" || !p.Demo {
		t.Fatal(p, token, err)
	}
	got, _ := idp.Identify(ctx, access.Credentials{SessionToken: token})
	if got.PersonID != "INS-01" || got.Role != "quality_inspector" || got.Name == "" {
		t.Fatalf("%+v", got)
	}
	// Подделка — анонимный.
	if got, _ := idp.Identify(ctx, access.Credentials{SessionToken: token + "x"}); !got.Anonymous() {
		t.Fatalf("подделка: %+v", got)
	}
	// Заголовок демо-персоны.
	if got, _ := idp.Identify(ctx, access.Credentials{DemoPersona: "TEC-01"}); got.Role != "technologist" {
		t.Fatalf("заголовок: %+v", got)
	}
	// Выход — сеанс больше не узнаётся.
	if err := idp.Close(ctx, token); err != nil {
		t.Fatal(err)
	}
	if got, _ := idp.Identify(ctx, access.Credentials{SessionToken: token}); !got.Anonymous() {
		t.Fatalf("после выхода: %+v", got)
	}
	// Неизвестная персона — отказ входа.
	var pe *platform.Error
	if _, _, err := idp.Open(ctx, access.SessionCreate{PersonaID: "NOPE"}); !errors.As(err, &pe) || pe.Code != errcodes.AccessLoginFailed {
		t.Fatalf("неизвестная персона: %v", err)
	}
	// Срок истёк.
	_, token, _ = idp.Open(ctx, access.SessionCreate{PersonaID: "INS-02"})
	now = now.Add(DefaultTTL + time.Second)
	if got, _ := idp.Identify(ctx, access.Credentials{SessionToken: token}); !got.Anonymous() {
		t.Fatalf("срок: %+v", got)
	}
}

func TestDemoOutsideDemoProfiles(t *testing.T) {
	ctx := context.Background()
	idp, err := NewDemo(directory(t), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := idp.DemoPersonas(ctx); err == nil {
		t.Fatal("персоны вне демо-профиля")
	}
	if _, _, err := idp.Open(ctx, access.SessionCreate{PersonaID: "INS-01"}); err == nil {
		t.Fatal("вход персоной вне демо-профиля")
	}
	if got, _ := idp.Identify(ctx, access.Credentials{DemoPersona: "INS-01"}); !got.Anonymous() {
		t.Fatalf("заголовок вне демо-профиля: %+v", got)
	}
}
