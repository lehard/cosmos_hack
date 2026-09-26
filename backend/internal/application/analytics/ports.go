package analytics

import (
	"context"
	"time"

	domain "ant/internal/domain/analytics"
)

// Store — ведомый порт чтения показателей (AD-45): строки вклада изделий и
// глобальные проекции analytics. Пишет их только движок эффектами в
// транзакции journal.Append (ContributionsReplace, ProjectionPut).
// Реализации: infrastructure/storage/analytics (Postgres), MemStore (тесты).
type Store interface {
	// Rows — строки вклада показателей metrics (пусто — все).
	Rows(ctx context.Context, metrics ...string) ([]domain.Row, error)
	// Equipment — проекции оборудования и остановок точек процесса.
	Equipment(ctx context.Context) ([]domain.Equipment, error)
	// Incidents — проекции инцидентов.
	Incidents(ctx context.Context) ([]domain.Incident, error)
}

// Clock — доменное «сейчас» (AD-37): конец периода по умолчанию.
type Clock interface {
	Now(ctx context.Context) (time.Time, error)
}

// Norms — нормы узлов (FR-5, FR-12: свойства элемента процесса). До модуля
// process — domain.DefaultNorm для всех узлов.
type Norms interface {
	Norm(ctx context.Context, stepKey string) domain.Norm
}

// DefaultNorms — нормы по умолчанию.
type DefaultNorms struct{}

// Norm — domain.DefaultNorm.
func (DefaultNorms) Norm(context.Context, string) domain.Norm { return domain.DefaultNorm }

// Shifts — график смен (справочник reference, эпик 19; FR-81): границы
// периода «смена» и срез «смена» (кейс §2.4: длительность операций по
// исполнителю и смене). nil — смены по 8 ч с 00:00, 08:00, 16:00 местного времени.
type Shifts interface {
	// Schedule — график смен прогона run (AD-38) на момент запроса.
	Schedule(ctx context.Context, run string) (ShiftLookup, error)
}

// ShiftLookup — смена на момент t: идущая или, если t вне смен, последняя
// начавшаяся до t (Active=false). ok=false — смен до t не было.
type ShiftLookup func(t time.Time) (s ShiftSpan, ok bool)

// ShiftSpan — смена-экземпляр графика.
type ShiftSpan struct {
	ID     string
	Label  string
	From   time.Time
	To     time.Time
	Active bool
}
