package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"ant/cmd/token-agent/internal/agent"
	dom "ant/internal/domain/signing"
)

// cmdImport — положить ключ на «токен»: зашифровать под PIN (класс
// hardware_token). PIN спрашивает эта программа.
func cmdImport(args []string) int {
	fs := newFlags("import")
	var keys multi
	fs.Var(&keys, "key", "файл ключа ‹key_ref›.key.json (повторяемый: gost и pq)")
	token := fs.String("token", "", "каталог токена (съёмный носитель); по умолчанию — из конфигурации")
	wp := fs.String("workplace", "", "рабочее место (workplace_id)")
	person := fs.String("person", "", "субъект ключа; по умолчанию — по имени ключа")
	if err := fs.Parse(args); err != nil || len(keys) == 0 {
		fmt.Fprintln(os.Stderr, "нужен хотя бы один -key")
		return 2
	}
	cfg := defaultConfig()
	if *token != "" {
		cfg.TokenDir = *token
	}
	if *wp != "" {
		cfg.WorkplaceID = *wp
	}
	var files [][]byte
	for _, k := range keys {
		b, err := os.ReadFile(k)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		files = append(files, b)
	}
	b, err := agent.NewBundle(files, *person)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	pin, err := askPINTerminal("Новый PIN токена: ")
	if err != nil {
		return 1
	}
	again, err := askPINTerminal("Повторите PIN: ")
	if err != nil || again != pin {
		fmt.Fprintln(os.Stderr, "PIN не совпал")
		return 1
	}
	s, err := agent.Seal(b, pin, agent.SealOptions{KeyStorage: dom.StorageHardwareToken}, rand.Reader)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := os.MkdirAll(cfg.TokenDir, 0o700); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	raw, _ := json.MarshalIndent(s, "", "  ")
	if err := os.WriteFile(cfg.sealedPath(), append(raw, '\n'), 0o600); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := cfg.save(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Printf("Ключ %v (%s) на токене %s, класс hardware_token.\n", s.KeyRefs(), s.PersonID, cfg.sealedPath())
	return 0
}

// manifestDirs — каталоги манифестов Native Messaging браузеров семейства Chromium.
func manifestDirs(browser string) []string {
	home, _ := os.UserHomeDir()
	var m map[string]string
	switch runtime.GOOS {
	case "darwin":
		as := filepath.Join(home, "Library", "Application Support")
		m = map[string]string{"chrome": filepath.Join(as, "Google", "Chrome", "NativeMessagingHosts"),
			"chromium": filepath.Join(as, "Chromium", "NativeMessagingHosts"),
			"yandex":   filepath.Join(as, "Yandex", "YandexBrowser", "NativeMessagingHosts")}
	default:
		cf := filepath.Join(home, ".config")
		m = map[string]string{"chrome": filepath.Join(cf, "google-chrome", "NativeMessagingHosts"),
			"chromium": filepath.Join(cf, "chromium", "NativeMessagingHosts"),
			"yandex":   filepath.Join(cf, "yandex-browser", "NativeMessagingHosts")}
	}
	if browser == "all" || browser == "" {
		return []string{m["chrome"], m["chromium"], m["yandex"]}
	}
	return []string{m[browser]}
}

// Manifest — манифест хоста Native Messaging: в allowed_origins — только
// расширение «Главный — подпись» с фиксированным ID (поле key, AD-14).
type Manifest struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Path           string   `json:"path"`
	Type           string   `json:"type"`
	AllowedOrigins []string `json:"allowed_origins"`
}

func cmdInstall(args []string) int {
	fs := newFlags("install")
	id := fs.String("extension-id", "", "ID расширения «Главный — подпись»")
	browser := fs.String("browser", "all", "chrome | chromium | yandex | all")
	if err := fs.Parse(args); err != nil || *id == "" {
		fmt.Fprintln(os.Stderr, "нужен -extension-id")
		return 2
	}
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	exe, _ = filepath.Abs(exe)
	m := Manifest{Name: HostName, Description: "Главный — агент токена (подпись физическим ключом)", Path: exe, Type: "stdio",
		AllowedOrigins: []string{"chrome-extension://" + *id + "/"}}
	raw, _ := json.MarshalIndent(m, "", "  ")
	for _, dir := range manifestDirs(*browser) {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		p := filepath.Join(dir, HostName+".json")
		if err := os.WriteFile(p, append(raw, '\n'), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		fmt.Println("манифест:", p)
	}
	return 0
}

func cmdStatus([]string) int {
	cfg := defaultConfig()
	fmt.Println("токен:", cfg.sealedPath())
	b, err := os.ReadFile(cfg.sealedPath())
	if err != nil {
		fmt.Println("  не вставлен")
	} else {
		var s agent.Sealed
		_ = json.Unmarshal(b, &s)
		fmt.Printf("  %s, ключи %v, класс %s\n", s.PersonID, s.KeyRefs(), s.KeyStorage)
	}
	fmt.Println("рабочее место:", cfg.WorkplaceID)
	for _, d := range manifestDirs("all") {
		if _, err := os.Stat(filepath.Join(d, HostName+".json")); err == nil {
			fmt.Println("манифест:", filepath.Join(d, HostName+".json"))
		}
	}
	return 0
}
