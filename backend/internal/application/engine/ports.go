package engine

import (
	"context"
	"encoding/json"

	jc "ant/internal/contracts/journal"
	"ant/internal/domain/engine"
	"ant/internal/domain/kernel"
)

// WorkFeed — ведомый порт подачи работы воркеру (AD-6, AD-35, ключ work_feed):
// фиксированные партиции hash(item_id) mod P в Postgres; Kafka с key = item_id —
// замена (контрактный тест + state_hash на Kafka). Сигнал «есть новое» — seq.
//
// Триггер свёртки (AD-5) — новая запись потока изделия: факт, решение,
// адресованная запись стадии, «наступил срок», «повтор обработки». Реакции и
// служебные записи самого воркера (роль-эмитент worker в каталоге) триггером
// не являются: адаптер их не подаёт, а воркер при подаче пропускает.
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
// продолжает (AD-45). Реализация — WorkerService.
type Worker interface {
	// Run обрабатывает партиции до отмены ctx.
	Run(ctx context.Context) error
}

// BundleSource — ведомый порт нормативного слоя изделия (AD-17): версия,
// закреплённая при запуске изделия, разложенная по модулям, и её ревизия
// (normative_rev записей). Реализует модуль process (эпик 17); до него —
// EmptyBundles.
type BundleSource interface {
	Bundle(ctx context.Context, itemID string, input []kernel.Record) (engine.Bundle, string, error)
}

// EmptyBundles — нормативный слой пустых модулей (волна 2).
type EmptyBundles struct{}

// Bundle возвращает пустой Bundle.
func (EmptyBundles) Bundle(context.Context, string, []kernel.Record) (engine.Bundle, string, error) {
	return engine.Bundle{}, "", nil
}

// Sealer — ведомый порт подписи записей движка (AD-3, AD-10): канонический
// конверт события → конверт DSSE, подписанный ключом «движок» (класс
// server-attested). Адаптер — SignerSealer над портом signing.Signer.
type Sealer interface {
	Seal(ctx context.Context, payload []byte) ([]byte, error)
}

// ProjectionStore — ведомый порт чтения проекций движка и каркаса проекций
// (AD-45): значение проекции name по ключу. Запись — только эффектами в
// транзакции Append (один писатель, курсор атомарно).
type ProjectionStore interface {
	Get(ctx context.Context, name, key string) (json.RawMessage, bool, error)
}

// ChangeLog — ведомый порт журнала изменений сущностей для живых обновлений
// (AD-6, AD-21): эффект Notify пишет строку в той же транзакции, что и
// проекцию; LISTEN/NOTIFY несёт только сигнал «есть новое». Позиция
// изменения (Change.Pos) растёт в порядке фиксации — по ней публикатор
// дочитывает новое, ничего не теряя, даже если воркеры разных партиций
// фиксируют изменения не в порядке seq. Копия api после переподключения
// догоняет по позиции, клиент SSE — по seq (Last-Event-ID).
type ChangeLog interface {
	// After — изменения по запросу q в порядке позиции.
	After(ctx context.Context, q ChangeQuery) ([]Change, error)
	// Tail — позиция последнего зафиксированного изменения (0 — изменений нет).
	Tail(ctx context.Context) (int64, error)
	// Wait блокирует до сигнала «есть новое» или отмены ctx.
	Wait(ctx context.Context) error
}

// ChangeQuery — выборка журнала изменений. Пустые поля не фильтруют.
type ChangeQuery struct {
	// AfterPos — только позиции после этой.
	AfterPos int64
	// UpToPos — только позиции не дальше этой (0 — без предела).
	UpToPos int64
	// AfterSeq — только изменения после seq журнала (догон подписки по Last-Event-ID).
	AfterSeq int64
	// Limit — предел числа изменений (0 — 500).
	Limit int
}
