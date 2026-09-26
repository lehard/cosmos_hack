package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config — конфигурация агента (AD-14): где «токен», где журнал, рабочее
// место. Файл config.json в каталоге состояния; переменные окружения —
// ANT_TOKEN_DIR, ANT_WORKPLACE_ID.
type Config struct {
	// TokenDir — каталог «токена»: key.sealed (хранилище под PIN). На
	// съёмном носителе извлечение носителя = извлечение токена: PIN
	// сбрасывается.
	TokenDir string `json:"token_dir"`
	// StateDir — локальный журнал подписанного и конфигурация.
	StateDir string `json:"-"`
	// WorkplaceID — рабочее место (в MVP — из конфигурации; цель — ключ
	// рабочего места по акту).
	WorkplaceID string `json:"workplace_id,omitempty"`
}

// SealedFile — имя хранилища на токене.
const SealedFile = "key.sealed"

func stateDir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "Glavny", "token-agent")
}

func defaultConfig() Config {
	c := Config{StateDir: stateDir()}
	if b, err := os.ReadFile(filepath.Join(c.StateDir, "config.json")); err == nil {
		_ = json.Unmarshal(b, &c)
	}
	if v := os.Getenv("ANT_TOKEN_DIR"); v != "" {
		c.TokenDir = v
	}
	if v := os.Getenv("ANT_WORKPLACE_ID"); v != "" {
		c.WorkplaceID = v
	}
	if c.TokenDir == "" {
		c.TokenDir = filepath.Join(c.StateDir, "token")
	}
	return c
}

func (c Config) save() error {
	if err := os.MkdirAll(c.StateDir, 0o700); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(c, "", "  ")
	return os.WriteFile(filepath.Join(c.StateDir, "config.json"), append(b, '\n'), 0o600)
}

func (c Config) sealedPath() string  { return filepath.Join(c.TokenDir, SealedFile) }
func (c Config) journalPath() string { return filepath.Join(c.StateDir, "journal.json") }
