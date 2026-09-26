package signing

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	app "ant/internal/application/signing"
	dom "ant/internal/domain/signing"
	"ant/internal/infrastructure/security/profiles"
	journalstore "ant/internal/infrastructure/storage/journal"
	"ant/internal/infrastructure/storage/journal/journaltest"
)

// demoRegs — открытые ключи демо-персон по key_ref из ключей генезиса
// (без журнала: только provisionKeys над томами v).
func demoRegs(t *testing.T, v Volumes, profile string, log *slog.Logger) map[string]string {
	t.Helper()
	seed, err := app.LoadSeed(Seed())
	if err != nil {
		t.Fatal(err)
	}
	cfg := provisionConfig(v)
	cfg.Profile = profile
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	regs, _, _, err := provisionKeys(cfg, seed, DemoValidFrom, log)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, r := range regs {
		if r.SubjectKind == dom.SubjectDemoPersona {
			out[r.KeyRef] = r.PublicKeyB64
		}
	}
	return out
}

// Д-82: два генезиса на пустых каталогах (другая машина, сброс стенда) дают
// одинаковые открытые ключи всех демо-персон — и агента токена, и demo-signer.
func TestDemoPersonaKeysDeterministic(t *testing.T) {
	for _, profile := range []string{"demo", "fixtures"} {
		a, b := demoRegs(t, volumes(t), profile, nil), demoRegs(t, volumes(t), profile, nil)
		// 26 людей × 2 ключа агента токена + персоны политики × 2 ключа demo-signer.
		if len(a) < 2*len(InteractivePersonas) || len(a) != len(b) {
			t.Fatalf("%s: ключей %d и %d", profile, len(a), len(b))
		}
		for ref, pub := range a {
			if b[ref] != pub {
				t.Errorf("%s: ключ %s различается между генезисами", profile, ref)
			}
		}
		for _, ref := range []string{"ins-01-ta@1", "ins-01-ta-pq@1", "met-82-ta@1", "tec-01@1", "tec-01-pq@1"} {
			if a[ref] == "" {
				t.Errorf("%s: ключа %s нет", profile, ref)
			}
		}
	}
	// Профиль входит в ввод KDF только через гард: demo и fixtures дают одни ключи.
	if a, b := demoRegs(t, volumes(t), "demo", nil), demoRegs(t, volumes(t), "fixtures", nil); a["ins-01-ta@1"] != b["ins-01-ta@1"] {
		t.Error("demo и fixtures: разные ключи одной персоны")
	}
}

// Файл, записанный генезисом, совпадает с выводом из демо-зерна и читается
// так, как его загрузит расширение.
func TestDemoPersonaKeyFile(t *testing.T) {
	v := volumes(t)
	regs := demoRegs(t, v, "demo", nil)
	for _, x := range interactiveRefs("HQC-01") {
		k, err := profiles.Load(filepath.Join(v.TokenAgent, profiles.FileName(x.ref)))
		if err != nil {
			t.Fatal(err)
		}
		want, err := DemoPersonaKey("demo", "HQC-01", x.ref, x.profile)
		if err != nil || k.PublicB64() != want.PublicB64() || regs[x.ref] != want.PublicB64() || k.Profile != x.profile {
			t.Fatalf("%s: файл и вывод различаются (%v)", x.ref, err)
		}
		// Ключ подписывает и проверяется своим профилем.
		sig, err := k.SignPAE("application/vnd.ant.event+json", []byte("pae"))
		if err != nil || !profiles.Verify(x.profile, k.Public(), "application/vnd.ant.event+json", []byte("pae"), sig) {
			t.Fatalf("%s: подпись: %v", x.ref, err)
		}
	}
	// Разные персоны, версии и профили — разные ключи.
	a, _ := DemoPersonaKey("demo", "INS-01", "ins-01-ta@1", dom.ProfileGost)
	b, _ := DemoPersonaKey("demo", "INS-02", "ins-02-ta@1", dom.ProfileGost)
	c, _ := DemoPersonaKey("demo", "INS-01", "ins-01-ta@2", dom.ProfileGost)
	if a.PublicB64() == b.PublicB64() || a.PublicB64() == c.PublicB64() {
		t.Error("ключи разных персон или версий совпали")
	}
}

// Существующий файл главнее: старый случайный ключ (уже загружен человеком в
// расширение) не перезаписывается и регистрируется как есть; предупреждение в
// логе; недостающие ключи — детерминированные.
func TestDemoPersonaExistingKeyKept(t *testing.T) {
	v := volumes(t)
	old, err := profiles.Generate("ins-01-ta@1", dom.ProfileGost)
	if err != nil {
		t.Fatal(err)
	}
	if err := profiles.Save(v.TokenAgent, old); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(v.TokenAgent, profiles.FileName("ins-01-ta@1"))
	before, _ := os.ReadFile(path)
	var buf bytes.Buffer
	regs := demoRegs(t, v, "demo", slog.New(slog.NewTextHandler(&buf, nil)))
	if after, _ := os.ReadFile(path); !bytes.Equal(before, after) {
		t.Fatal("существующий файл ключа перезаписан")
	}
	if regs["ins-01-ta@1"] != old.PublicB64() {
		t.Fatal("генезис зарегистрировал не существующий ключ")
	}
	if !strings.Contains(buf.String(), "level=WARN") || !strings.Contains(buf.String(), "ins-01-ta@1") {
		t.Fatalf("нет предупреждения о случайном ключе: %s", buf.String())
	}
	if strings.Contains(buf.String(), "ins-02-ta@1") {
		t.Error("детерминированный ключ попал в предупреждение")
	}
	want, _ := DemoPersonaKey("demo", "INS-02", "ins-02-ta@1", dom.ProfileGost)
	if regs["ins-02-ta@1"] != want.PublicB64() {
		t.Error("недостающий ключ не детерминирован")
	}
}

// prod: демо-персон нет, вывод из демо-зерна отказывает; load — случайные.
func TestDemoPersonaKeysNotInProd(t *testing.T) {
	for _, p := range []string{"prod", "load", ""} {
		if _, err := DemoPersonaKey(p, "INS-01", "ins-01-ta@1", dom.ProfileGost); !errors.Is(err, ErrNotDemoProfile) {
			t.Fatalf("%q: ожидался отказ, %v", p, err)
		}
	}
	v := volumes(t)
	if regs := demoRegs(t, v, "prod", nil); len(regs) != 0 {
		t.Fatalf("prod: ключей демо-персон %d", len(regs))
	}
	if ents, _ := os.ReadDir(v.TokenAgent); len(ents) != 0 {
		t.Fatal("prod: ключи агента токена записаны")
	}
	if a, b := demoRegs(t, volumes(t), "load", nil), demoRegs(t, volumes(t), "load", nil); a["ins-01-ta@1"] == "" || a["ins-01-ta@1"] == b["ins-01-ta@1"] {
		t.Fatal("load: ключи должны быть случайными")
	}
}

// Генезис уже есть, каталог агента токена очищен: повтор ant init
// восстанавливает те же файлы (открытые ключи совпадают с генезисом).
func TestDemoPersonaKeysRestoredOnDB(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	db := journaltest.NewDB(t)
	j := journalstore.NewStore(db.AppPool(t), journaltest.SysClock{}, journalstore.WithBatchMax(4096))
	v := volumes(t)
	if _, err := Provision(ctx, j, provisionConfig(v)); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(v.TokenAgent, profiles.FileName("cr-71-ta-pq@1"))
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(v.TokenAgent); err != nil {
		t.Fatal(err)
	}
	again, err := Provision(ctx, j, provisionConfig(v))
	if err != nil || again.Written {
		t.Fatalf("повтор: %+v %v", again, err)
	}
	if after, err := os.ReadFile(path); err != nil || !bytes.Equal(before, after) {
		t.Fatalf("ключ не восстановлен тем же: %v", err)
	}
	if ents, _ := os.ReadDir(v.TokenAgent); len(ents) != 2*len(InteractivePersonas) {
		t.Fatalf("восстановлено %d файлов", len(ents))
	}
}
