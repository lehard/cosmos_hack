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
	// Entries — журнал событий (journal.entry.list): общий экран, фильтры по изделию, потоку, типу.
	Entries(ctx context.Context, f EntryFilter, m platform.Moment, p platform.Page) (JournalEntryList, error)
	// Entry — запись по seq (journal.entry.read).
	Entry(ctx context.Context, seq int64) (JournalEntryView, error)
	// Event — запись по event_id для окна записи (journal.event.read; интерфейс 6).
	Event(ctx context.Context, eventID string, m platform.Moment) (JournalEventView, error)
	// Head — голова журнала (journal.head.read).
	Head(ctx context.Context, m platform.Moment) (JournalHead, error)
	// Timeline — диапазон истории и метки таймлайна (journal.timeline.read, FR-4).
	Timeline(ctx context.Context, m platform.Moment) (TimelineData, error)
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

func (Unimplemented) Entries(context.Context, EntryFilter, platform.Moment, platform.Page) (JournalEntryList, error) {
	return JournalEntryList{}, platform.NotImplemented("journal.entry.list")
}

func (Unimplemented) Entry(context.Context, int64) (JournalEntryView, error) {
	return JournalEntryView{}, platform.NotImplemented("journal.entry.read")
}

func (Unimplemented) Event(context.Context, string, platform.Moment) (JournalEventView, error) {
	return JournalEventView{}, platform.NotImplemented("journal.event.read")
}

func (Unimplemented) Head(context.Context, platform.Moment) (JournalHead, error) {
	return JournalHead{}, platform.NotImplemented("journal.head.read")
}

func (Unimplemented) Timeline(context.Context, platform.Moment) (TimelineData, error) {
	return TimelineData{}, platform.NotImplemented("journal.timeline.read")
}

var (
	_ Queries  = Unimplemented{}
	_ Commands = Unimplemented{}
)
