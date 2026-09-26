package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ant/cmd/internal/config"
	processapp "ant/internal/application/process"
	"ant/internal/infrastructure/storage/journal/journaltest"
)

// Роль init на своей БД (make dev-db), эпик 05: на чистой базе — миграции,
// ключи и генезис; повтор (и migrate после него) журнал не трогает; ядро
// берёт из генезиса стартовый процесс и ключ шлюза приёма.
func TestInitRole(t *testing.T) {
	if os.Getenv("ANT_DB_HOST") == "" {
		t.Skip("нет ANT_DB_HOST — интеграционный тест роли init пропущен (make dev-db)")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
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
	env := &environment{cfg: cfg, log: slog.New(slog.DiscardHandler), ctx: ctx}
	if err := runInit(ctx, env); err != nil {
		t.Fatalf("init: %v", err)
	}
	c, err := env.core(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// Фоновые циклы ядра живут до отмены ctx: отменить, затем закрыть ядро.
	defer func() { cancel(); env.closeCore() }()
	head := func() int64 {
		h, err := c.journal.Head(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return h.MainSeq
	}
	n := head()
	if n < 100 {
		t.Fatalf("блок генезиса: %d записей", n)
	}
	if err := runInit(ctx, env); err != nil || head() != n {
		t.Fatalf("повтор init: %v, записей %d → %d", err, n, head())
	}
	if err := runMigrate(ctx, env); err != nil || head() != n {
		t.Fatalf("migrate после генезиса: %v, записей %d → %d", err, n, head())
	}
	xml, err := genesisProcessXML(ctx, c, env, func() ([]byte, error) {
		t.Fatal("стартовый процесс не из генезиса")
		return nil, nil
	})
	if err != nil || len(xml) == 0 {
		t.Fatalf("процесс: %v", err)
	}
	c.ensureProcessSeed(ctx, env)
	if v, ok, err := c.versions.ByID(ctx, processapp.SeedVersionID); err != nil || !ok || !v.Genesis {
		t.Fatalf("стартовая версия: %+v %v %v", v, ok, err)
	}
	if v, k, s := genesisIngest(env, c); v == nil || k == nil || s == nil {
		t.Fatal("приём без ключей генезиса")
	}
}
