// Пакет permissive — разрешающие заглушки ведомых портов прав и входа для
// волны 1 (эпик 02): AccessControl разрешает всё, IdentityProvider не знает
// сеансов (субъект анонимный). Общий декоратор (application/access.Gate) уже
// вызывается для каждой операции; эпик 08 заменяет адаптеры на Casbin и
// локальных пользователей с демо-персонами — без правки операций (AD-15, AD-35).
// Signer — конверт DSSE без подписи для записей движка до эпика 05.
//
// Слой: infrastructure/security — технический механизм (AD-1); реализует
// порты application/access и application/signing.Signer; ключ конфигурации
// access_control = permissive, identity_provider = none.
package permissive

import (
	"context"

	"ant/internal/application/access"
	"ant/internal/application/platform"
)

// AccessControl — всё разрешено; версия политики 0.
type AccessControl struct{}

// Enforce разрешает любое действие.
func (AccessControl) Enforce(context.Context, access.Request) (access.Decision, error) {
	return access.Decision{Allowed: true}, nil
}

// PolicySeq — политики ещё нет.
func (AccessControl) PolicySeq(context.Context) (int64, error) { return 0, nil }

// Identity — сеансов нет: субъект всегда анонимный; вход — 501.
type Identity struct{}

// Identify — анонимный субъект.
func (Identity) Identify(context.Context, access.Credentials) (platform.Principal, error) {
	return platform.Principal{}, nil
}

// Open — вход ещё не реализован (эпик 08).
func (Identity) Open(context.Context, access.SessionCreate) (platform.Principal, string, error) {
	return platform.Principal{}, "", platform.NotImplemented("access.session.create")
}

// Close — выход ещё не реализован (эпик 08).
func (Identity) Close(context.Context, string) error {
	return platform.NotImplemented("access.session.delete")
}

// DemoPersonas — ещё не реализовано (эпик 08).
func (Identity) DemoPersonas(context.Context) ([]access.DemoPersona, error) {
	return nil, platform.NotImplemented("access.persona.list")
}

var (
	_ access.AccessControl    = AccessControl{}
	_ access.IdentityProvider = Identity{}
)
