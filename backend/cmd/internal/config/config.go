// Пакет config — конфигурация процессов ant: файл deploy/config/ant.yaml,
// профиль и переопределения переменными окружения ANT_* (AD-25).
//
// Слой: сборка зависимостей (cmd/*), не домен и не приложение.
// Связи: читается всеми точками входа из backend/cmd; другие пакеты получают
// уже разобранные значения аргументами.
//
// Порядок наложения значений: секция defaults → секция profiles.<профиль> →
// переменные окружения ANT_<ПУТЬ>. Путь — имена ключей yaml через «_» в верхнем
// регистре: http.addr → ANT_HTTP_ADDR, db.password_file → ANT_DB_PASSWORD_FILE.
// Профиль выбирает ANT_PROFILE, иначе ключ profile в файле.
//
// Секретов в ANT_* нет (AD-25): пароль БД и ключи читаются только из файлов,
// переменная указывает путь к файлу.
package config

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

// Профили конфигурации (AD-25).
const (
	ProfileFixtures = "fixtures"
	ProfileDemo     = "demo"
	ProfileLoad     = "load"
	ProfileProd     = "prod"
)

// Profiles — допустимые профили в порядке описания.
var Profiles = []string{ProfileFixtures, ProfileDemo, ProfileLoad, ProfileProd}

// Режимы ведущих портов приложения (AD-36, FR-150).
const (
	PortsFixtures = "fixtures"
	PortsLive     = "live"
)

// EnvPrefix — префикс переменных окружения.
const EnvPrefix = "ANT"

// Config — полная конфигурация процесса ant.
type Config struct {
	// Profile — выбранный профиль; заполняется загрузчиком.
	Profile string `yaml:"-"`

	Log struct {
		// Level — debug | info | warn | error.
		Level string `yaml:"level"`
	} `yaml:"log"`

	HTTP struct {
		// Addr — адрес прослушивания API и встроенного интерфейса.
		Addr              string        `yaml:"addr"`
		ReadHeaderTimeout time.Duration `yaml:"read_header_timeout"`
		ShutdownTimeout   time.Duration `yaml:"shutdown_timeout"`
		// MaxBodyBytes — предельный размер тела запроса.
		MaxBodyBytes int64 `yaml:"max_body_bytes"`
	} `yaml:"http"`

	DB DB `yaml:"db"`

	Ports struct {
		// Mode — fixtures | live: реализация ведущих портов Queries/Commands (AD-36).
		Mode string `yaml:"mode"`
	} `yaml:"ports"`

	Engine struct {
		// Partitions — число фиксированных партиций P (AD-6).
		Partitions int `yaml:"partitions"`
		// FoldParallelism — параллелизм свёртки изделий внутри воркера.
		FoldParallelism int `yaml:"fold_parallelism"`
	} `yaml:"engine"`

	Journal struct {
		// BatchMax — предельное число записей в пачке journal.Append (AD-44).
		BatchMax int `yaml:"batch_max"`
		// BatchWait — предельное ожидание добора пачки (AD-44: 50 мс).
		BatchWait time.Duration `yaml:"batch_wait"`
	} `yaml:"journal"`

	Integrations struct {
		// Enabled — включённые внешние системы (stand-ы или настоящие адаптеры).
		Enabled []string `yaml:"enabled"`
	} `yaml:"integrations"`
}

// DB — подключение к PostgreSQL. Пароль — только файлом.
type DB struct {
	Host           string        `yaml:"host"`
	Port           int           `yaml:"port"`
	Name           string        `yaml:"name"`
	User           string        `yaml:"user"`
	PasswordFile   string        `yaml:"password_file"`
	SSLMode        string        `yaml:"sslmode"`
	MaxConns       int           `yaml:"max_conns"`
	ConnectTimeout time.Duration `yaml:"connect_timeout"`
}

// file — формат deploy/config/ant.yaml.
type file struct {
	Profile  string               `yaml:"profile"`
	Defaults yaml.Node            `yaml:"defaults"`
	Profiles map[string]yaml.Node `yaml:"profiles"`
}

// Load читает файл path, накладывает профиль и переменные окружения из lookup
// (обычно os.LookupEnv) и проверяет результат.
func Load(path string, lookup func(string) (string, bool)) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("конфигурация: %w", err)
	}
	return Parse(raw, lookup)
}

// Parse — то же, что Load, но из байтов.
func Parse(raw []byte, lookup func(string) (string, bool)) (*Config, error) {
	var f file
	if err := yaml.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("конфигурация: разбор yaml: %w", err)
	}
	cfg := &Config{}
	if !f.Defaults.IsZero() {
		if err := f.Defaults.Decode(cfg); err != nil {
			return nil, fmt.Errorf("конфигурация: defaults: %w", err)
		}
	}
	profile := f.Profile
	if v, ok := lookup(EnvPrefix + "_PROFILE"); ok && v != "" {
		profile = v
	}
	if !slices.Contains(Profiles, profile) {
		return nil, fmt.Errorf("конфигурация: неизвестный профиль %q (допустимы: %s)", profile, strings.Join(Profiles, ", "))
	}
	if node, ok := f.Profiles[profile]; ok && !node.IsZero() {
		if err := node.Decode(cfg); err != nil {
			return nil, fmt.Errorf("конфигурация: профиль %s: %w", profile, err)
		}
	}
	cfg.Profile = profile
	if err := applyEnv(reflect.ValueOf(cfg).Elem(), EnvPrefix, lookup); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Validate проверяет значения, без которых процесс не должен стартовать.
func (c *Config) Validate() error {
	var errs []error
	if c.HTTP.Addr == "" {
		errs = append(errs, errors.New("http.addr пуст"))
	}
	if c.Ports.Mode != PortsFixtures && c.Ports.Mode != PortsLive {
		errs = append(errs, fmt.Errorf("ports.mode = %q, допустимы fixtures | live", c.Ports.Mode))
	}
	if c.Engine.Partitions <= 0 {
		errs = append(errs, errors.New("engine.partitions должно быть > 0"))
	}
	switch c.Log.Level {
	case "debug", "info", "warn", "error":
	default:
		errs = append(errs, fmt.Errorf("log.level = %q, допустимы debug | info | warn | error", c.Log.Level))
	}
	if len(errs) > 0 {
		return fmt.Errorf("конфигурация: %w", errors.Join(errs...))
	}
	return nil
}

// EnvNames возвращает имена всех переменных ANT_*, которые понимает загрузчик
// (для документации и проверки опечаток).
func EnvNames() []string {
	var names []string
	walk(reflect.TypeFor[Config](), EnvPrefix, func(name string, _ reflect.Type) { names = append(names, name) })
	slices.Sort(names)
	return append(names, EnvPrefix+"_PROFILE")
}

func walk(t reflect.Type, prefix string, fn func(string, reflect.Type)) {
	for i := range t.NumField() {
		f := t.Field(i)
		tag := strings.Split(f.Tag.Get("yaml"), ",")[0]
		if tag == "" || tag == "-" {
			continue
		}
		name := prefix + "_" + strings.ToUpper(tag)
		if f.Type.Kind() == reflect.Struct && f.Type != reflect.TypeFor[time.Duration]() {
			walk(f.Type, name, fn)
			continue
		}
		fn(name, f.Type)
	}
}

var durationType = reflect.TypeFor[time.Duration]()

// applyEnv накладывает переменные окружения на листья структуры v.
func applyEnv(v reflect.Value, prefix string, lookup func(string) (string, bool)) error {
	t := v.Type()
	for i := range t.NumField() {
		f := t.Field(i)
		tag := strings.Split(f.Tag.Get("yaml"), ",")[0]
		if tag == "" || tag == "-" {
			continue
		}
		name := prefix + "_" + strings.ToUpper(tag)
		fv := v.Field(i)
		if f.Type.Kind() == reflect.Struct && f.Type != durationType {
			if err := applyEnv(fv, name, lookup); err != nil {
				return err
			}
			continue
		}
		s, ok := lookup(name)
		if !ok {
			continue
		}
		if err := setValue(fv, s); err != nil {
			return fmt.Errorf("конфигурация: %s: %w", name, err)
		}
	}
	return nil
}

func setValue(fv reflect.Value, s string) error {
	if fv.Type() == durationType {
		d, err := time.ParseDuration(s)
		if err != nil {
			return err
		}
		fv.SetInt(int64(d))
		return nil
	}
	switch fv.Kind() {
	case reflect.String:
		fv.SetString(s)
	case reflect.Int, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(s, 10, fv.Type().Bits())
		if err != nil {
			return err
		}
		fv.SetInt(n)
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return err
		}
		fv.SetBool(b)
	case reflect.Slice:
		if fv.Type().Elem().Kind() != reflect.String {
			return fmt.Errorf("тип %s не поддержан", fv.Type())
		}
		var items []string
		for p := range strings.SplitSeq(s, ",") {
			if p = strings.TrimSpace(p); p != "" {
				items = append(items, p)
			}
		}
		fv.Set(reflect.ValueOf(items))
	default:
		return fmt.Errorf("тип %s не поддержан", fv.Type())
	}
	return nil
}
