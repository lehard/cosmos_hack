package access

import (
	"context"
	"errors"
	"time"

	"ant/internal/application/platform"
	accessdom "ant/internal/domain/access"
)

// PolicySource — действующая политика (AD-15: источник правды — журнал):
// затравка или генезис плюс записи policy.* и access.* через проекцию.
// Реализация — Projection; её читают вычислитель Casbin, порт входа и сценарии access.
type PolicySource interface {
	// Policy — снимок политики; вызывающий не меняет его (изменения — Clone).
	Policy(ctx context.Context) (accessdom.Policy, error)
}

// StaticPolicy — политика без журнала (тесты, выгрузка OpenAPI).
type StaticPolicy accessdom.Policy

// Policy возвращает саму политику.
func (s StaticPolicy) Policy(context.Context) (accessdom.Policy, error) {
	return accessdom.Policy(s), nil
}

// PolicyLog — ведомый порт чтения записей политики из журнала (AD-45):
// адаптер — infrastructure/storage/access над JournalStore и кодеком записей.
type PolicyLog interface {
	// Since — записи типов accessdom.Types после seq, по возрастанию seq, data —
	// в текущей версии схемы.
	Since(ctx context.Context, afterSeq int64) ([]accessdom.Record, error)
	// Head — seq головы журнала по сигналу «есть новое» (LISTEN/NOTIFY, без
	// запроса к БД); 0 — сигнала нет, проекция перечитывает по интервалу.
	Head() int64
}

// Places — ведомый порт мест: рабочее место → путь области (справочник мест,
// normative/reference; эпик 19 — живой справочник). Место операции — атрибут
// запроса Casbin (барьер 3, AD-15).
type Places interface {
	// ScopeOf — область рабочего места или места (id из справочника мест).
	ScopeOf(id string) (string, bool)
}

// SecurityEvents — ведомый порт шины безопасности (AD-24): события семейства
// security пишет в журнал модуль security; access сообщает о неудачном входе
// и отказе в доступе, не зная подписчиков.
type SecurityEvents interface {
	AuthFailed(ctx context.Context, f AuthFailure) error
	AccessDenied(ctx context.Context, d Denial) error
}

// AuthFailure — неудачный вход (security.auth.failed).
type AuthFailure struct {
	Login    string
	ClientIP string
	// Reason — bad_credentials | account_locked | account_pending | rate_limited.
	Reason string
	At     time.Time
}

// Denial — отказ в доступе (security.access.denied).
type Denial struct {
	PersonID string
	ActionID string
	Object   platform.ObjectRef
	Code     string
	At       time.Time
}

// Причины неудачного входа (security.auth.failed.reason).
const (
	AuthBadCredentials = "bad_credentials"
	AuthAccountLocked  = "account_locked"
	AuthAccountPending = "account_pending"
	AuthRateLimited    = "rate_limited"
)

// Состояния учётной записи (AccessPerson.account_status).
const (
	AccountNone    = "none"
	AccountPending = "pending"
	AccountActive  = "active"
	AccountBlocked = "blocked"
)

// Credential — учётные данные входа по логину (FR-128). Хеш пароля — только
// argon2id и только в схеме access, не в журнале: журнал хранит факт
// активации (access.account.activated), а не секрет.
type Credential struct {
	Login    string
	PersonID string
	// Hash — хеш пароля в формате PHC ($argon2id$…).
	Hash string
	// Status — pending (заявка) | active | blocked.
	Status string
	// DisplayName — имя из заявки на регистрацию.
	DisplayName string
	// Failures — неудачных попыток подряд; LockedUntil — блокировка после N неудач.
	Failures    int
	LockedUntil time.Time
	CreatedAt   time.Time
}

// ErrLoginTaken — логин уже занят.
var ErrLoginTaken = errors.New("access: логин занят")

// CredentialStore — ведомый порт учётных данных (схема access, адаптер —
// infrastructure/storage/access).
type CredentialStore interface {
	// Get — учётные данные по логину.
	Get(ctx context.Context, login string) (Credential, bool, error)
	// Create — новая заявка или учётная запись; логин занят — ErrLoginTaken.
	Create(ctx context.Context, c Credential) error
	// Activate — учётная запись действует и привязана к сотруднику.
	Activate(ctx context.Context, login, personID string) error
	// Failed — неудачная попытка: счётчик +1; достиг max — блокировка до now+lockFor.
	Failed(ctx context.Context, login string, now time.Time, max int, lockFor time.Duration) (Credential, error)
	// Succeeded — сброс счётчика неудач.
	Succeeded(ctx context.Context, login string) error
	// List — все учётные данные (заявки и учётные записи) по логину.
	List(ctx context.Context) ([]Credential, error)
}

// PasswordHasher — ведомый порт хеша паролей (argon2id, infrastructure/security/identity).
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(hash, password string) (bool, error)
}
