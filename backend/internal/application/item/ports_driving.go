package item

import (
	"context"

	"ant/internal/application/platform"
)

// Queries — ведущий порт чтения модуля item (AD-36): операции чтения API
// опираются только на него. Методы добавляются вместе с операциями контракта.
type Queries interface {
	// Lookup — изделие по номеру детали или содержимому DataMatrix (item.item.lookup):
	// разрешение носителя на момент (AD-41); не найдено — api.not_found.
	Lookup(ctx context.Context, q string, m platform.Moment) (ItemLookup, error)
}

// Commands — ведущий порт команд модуля item (AD-36, AD-39): одна операция —
// одна реализация команды в модуле-владельце.
type Commands interface{}

// Unimplemented — заглушка портов item: каждая операция отвечает 501
// api.not_implemented. Встраивается в реализации live и fixtures, чтобы новые
// операции контракта не ломали сборку (реализация переопределяет метод).
type Unimplemented struct{}

func (Unimplemented) Lookup(context.Context, string, platform.Moment) (ItemLookup, error) {
	return ItemLookup{}, platform.NotImplemented("item.item.lookup")
}

var (
	_ Queries  = Unimplemented{}
	_ Commands = Unimplemented{}
)
