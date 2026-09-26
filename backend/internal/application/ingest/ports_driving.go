package ingest

// Queries — ведущий порт чтения модуля ingest (AD-36): операции чтения API
// опираются только на него. Методы добавляются вместе с операциями контракта.
type Queries interface{}

// Commands — ведущий порт команд модуля ingest (AD-36, AD-39): одна операция —
// одна реализация команды в модуле-владельце.
type Commands interface{}

// Unimplemented — заглушка портов ingest: каждая операция отвечает 501
// api.not_implemented. Встраивается в реализации live и fixtures, чтобы новые
// операции контракта не ломали сборку (реализация переопределяет метод).
type Unimplemented struct{}

var (
	_ Queries  = Unimplemented{}
	_ Commands = Unimplemented{}
)
