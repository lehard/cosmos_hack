package journal

import (
	"context"

	"ant/internal/application/platform"
)

// Queries — ведущий порт чтения модуля journal (AD-36): операции чтения API
// опираются только на него.
type Queries interface {
	// Subscribe — живые обновления (journal.stream.subscribe, AD-21): подписка
	// на изменения сущностей после seq в пределах прогона (LISTEN/NOTIFY несёт
	// только сигнал «есть новое» с seq, AD-6).
	Subscribe(ctx context.Context, afterSeq int64, runID string) (Subscription, error)
}

// Subscription — открытая подписка на изменения.
type Subscription interface {
	// Next блокирует до следующего изменения или отмены ctx.
	Next(ctx context.Context) (Change, error)
	// Close освобождает подписку.
	Close()
}

// Change — изменение сущности после записи журнала (сообщение SSE EntityChanged).
type Change struct {
	Entity platform.EntityKind
	ID     string
	Seq    int64
	RunID  string
	Mode   platform.Mode
}

// Commands — ведущий порт команд модуля journal (AD-36, AD-39): одна операция —
// одна реализация команды в модуле-владельце.
type Commands interface{}

// Unimplemented — заглушка портов journal: каждая операция отвечает 501
// api.not_implemented. Встраивается в реализации live и fixtures, чтобы новые
// операции контракта не ломали сборку (реализация переопределяет метод).
type Unimplemented struct{}

func (Unimplemented) Subscribe(context.Context, int64, string) (Subscription, error) {
	return nil, platform.NotImplemented("journal.stream.subscribe")
}

var (
	_ Queries  = Unimplemented{}
	_ Commands = Unimplemented{}
)
