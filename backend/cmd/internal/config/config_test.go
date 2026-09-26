package config

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
	"time"
)

// repoConfig — путь к настоящему deploy/config/ant.yaml: тест проверяет, что
// файл из репозитория разбирается во всех профилях.
func repoConfig(t *testing.T) string {
	t.Helper()
	_, self, _, _ := runtime.Caller(0)
	p := filepath.Join(filepath.Dir(self), "..", "..", "..", "..", "deploy", "config", "ant.yaml")
	if _, err := os.Stat(p); err != nil {
		t.Skipf("deploy/config/ant.yaml недоступен: %v", err)
	}
	return p
}

func env(m map[string]string) func(string) (string, bool) {
	return func(k string) (string, bool) { v, ok := m[k]; return v, ok }
}

func TestRepoConfigAllProfiles(t *testing.T) {
	path := repoConfig(t)
	for _, p := range Profiles {
		cfg, err := Load(path, env(map[string]string{"ANT_PROFILE": p}))
		if err != nil {
			t.Fatalf("профиль %s: %v", p, err)
		}
		if cfg.Profile != p {
			t.Fatalf("профиль %s: получили %s", p, cfg.Profile)
		}
	}
}

func TestLayering(t *testing.T) {
	raw := []byte(`
profile: demo
defaults:
  log: {level: info}
  http: {addr: ":8080", shutdown_timeout: 10s}
  ports: {mode: live}
  engine: {partitions: 16}
  integrations: {enabled: [erp]}
profiles:
  fixtures:
    ports: {mode: fixtures}
  demo: {}
`)
	cfg, err := Parse(raw, env(map[string]string{
		"ANT_PROFILE":               "fixtures",
		"ANT_HTTP_ADDR":             "127.0.0.1:9000",
		"ANT_ENGINE_PARTITIONS":     "4",
		"ANT_HTTP_SHUTDOWN_TIMEOUT": "3s",
		"ANT_INTEGRATIONS_ENABLED":  "erp, mes",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Ports.Mode != PortsFixtures {
		t.Errorf("ports.mode = %q, ждали fixtures из профиля", cfg.Ports.Mode)
	}
	if cfg.HTTP.Addr != "127.0.0.1:9000" || cfg.Engine.Partitions != 4 || cfg.HTTP.ShutdownTimeout != 3*time.Second {
		t.Errorf("переменные окружения не наложены: %+v", cfg)
	}
	if len(cfg.Integrations.Enabled) != 2 || cfg.Integrations.Enabled[1] != "mes" {
		t.Errorf("integrations.enabled = %v", cfg.Integrations.Enabled)
	}
}

func TestUnknownProfile(t *testing.T) {
	_, err := Parse([]byte("profile: nope\n"), env(nil))
	if err == nil {
		t.Fatal("ждали ошибку неизвестного профиля")
	}
}

func TestEnvNames(t *testing.T) {
	names := EnvNames()
	for _, want := range []string{"ANT_HTTP_ADDR", "ANT_DB_PASSWORD_FILE", "ANT_PORTS_MODE", "ANT_PROFILE", "ANT_STANDS_ADDR", "ANT_STANDS_EDGE_URL"} {
		found := false
		for _, n := range names {
			found = found || n == want
		}
		if !found {
			t.Errorf("нет %s в %v", want, names)
		}
	}
}

// Эпик 43: в demo Галактика и MES установлены stand-ами, учётный обмен ведёт
// 1С; профиль накладывается на defaults, не стирая соседние ключи;
// ANT_ERP_LEDGER переключает учётный обмен на Галактику.
func TestDemoStandsGalaktikaMES(t *testing.T) {
	path := repoConfig(t)
	cfg, err := Load(path, env(map[string]string{"ANT_PROFILE": "demo"}))
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"onec", "galaktika", "mes"} {
		if !slices.Contains(cfg.Integrations.Enabled, s) {
			t.Errorf("demo: %s не установлена: %v", s, cfg.Integrations.Enabled)
		}
	}
	g := cfg.ERP.Galaktika
	if cfg.ERP.Ledger != "onec" || !g.Stand || g.Dir != "/var/lib/ant/exchange/galaktika" || g.Transport != "exchange-dir" || !cfg.MES.B2MML.Stand || cfg.MES.B2MML.RetryMax == 0 {
		t.Errorf("demo: erp %+v, mes %+v", cfg.ERP, cfg.MES)
	}
	cfg, err = Load(path, env(map[string]string{"ANT_PROFILE": "demo", "ANT_ERP_LEDGER": "galaktika"}))
	if err != nil || cfg.ERP.Ledger != "galaktika" {
		t.Fatalf("ANT_ERP_LEDGER: %+v %v", cfg.ERP.Ledger, err)
	}
	if _, err := Load(path, env(map[string]string{"ANT_ERP_LEDGER": "sap"})); err == nil {
		t.Error("erp.ledger = sap должна отвергаться")
	}
}
