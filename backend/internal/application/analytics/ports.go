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
