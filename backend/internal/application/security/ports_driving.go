package security

import (
	"context"

	"ant/internal/application/platform"
)

// Queries — ведущий порт чтения модуля security (AD-36): операции чтения API
// опираются только на него. Методы добавляются вместе с операциями контракта.
type Queries interface {
	// Integrity — состояние целостности журнала «по данным сервера»
	// (security.integrity.read, AD-46): последний подписанный отчёт верификатора.
	Integrity(ctx context.Context) (IntegrityStatus, error)
}

// Commands — ведущий порт команд модуля security (AD-36, AD-39): одна операция —
// одна реализация команды в модуле-владельце.
type Commands interface{}

// Unimplemented — заглушка портов security: каждая операция отвечает 501
// api.not_implemented. Встраивается в реализации live и fixtures, чтобы новые
// операции контракта не ломали сборку (реализация переопределяет метод).
type Unimplemented struct{}

func (Unimplemented) Integrity(context.Context) (IntegrityStatus, error) {
	return IntegrityStatus{}, platform.NotImplemented("security.integrity.read")
}

var (
	_ Queries  = Unimplemented{}
	_ Commands = Unimplemented{}
)
