package engine

import (
	"context"

	jc "ant/internal/contracts/journal"
)

// WorkFeed — ведомый порт подачи работы воркеру (AD-6, AD-35, ключ work_feed):
// фиксированные партиции hash(item_id) mod P в Postgres; Kafka с key = item_id —
// замена (контрактный тест + state_hash на Kafka). Сигнал «есть новое» — seq.
type WorkFeed interface {
	// Partitions — партиции, аренду которых держит эта копия (с эпохами).
	Partitions(ctx context.Context) ([]Partition, error)
	// Next — изделия партиции с необработанным входом после курсора; блокирует
	// до сигнала «есть новое» или отмены ctx.
	Next(ctx context.Context, p Partition) ([]Work, error)
}

// Partition — партиция с эпохой аренды.
type Partition struct {
	Number int
	Epoch  int64
}

// Work — изделие с необработанным входом: вход целиком читается из журнала до
// basis_seq и сворачивается заново (AD-5).
type Work struct {
	ItemID  string
	UpToSeq int64
	Trigger jc.JournalEntry
}

// Worker — сценарий воркера (AD-5, AD-40): вход изделия из партиции →
// пересвёртка целиком (domain/engine.Fold) → сравнение реакций со слотами
// записанных (новые, пересмотренные «из-за записи ‹id›», исчезнувшие по AD-3)
// → journal.Append с basis_seq, вкладами показателей и курсором. Ошибка на
// записи → ops.processing.failed, изделие «обработка остановлена», партиция
// продолжает (AD-45). Реализация — эпик 07.
type Worker interface {
	// Run обрабатывает партиции до отмены ctx.
	Run(ctx context.Context) error
}
