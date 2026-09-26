// Пакет engine — производственные сценарии движка слоя application (эпик 07):
//
//   - сценарий воркера WorkerService (AD-5, AD-6, AD-40, AD-45): вход изделия
//     из партиции (порт WorkFeed) → пересвёртка всего входа (domain/engine.Fold)
//     → сравнение реакций со слотами записанных (domain/engine.Diff) →
//     journal.Append разницы с basis_seq, проекциями изделия, вкладами
//     показателей и курсором партиции в одной транзакции; ошибка на записи
//     изделия → ops.processing.failed, «обработка остановлена», партиция
//     продолжает;
//   - каркас проекций (Registry, ItemProjection, GlobalProjection, Projector):
//     один писатель на проекцию, проекция и курсор атомарно; роль projector —
//     копия-лидер по аренде (Leader);
//   - `ant rebuild` и `ant rebuild --item` (Rebuilder, FR-115, FR-124);
//   - запросы состояния на момент (StateQueries, AD-22) — свёртка префикса без записи;
//   - публикатор живых обновлений SSE (LiveUpdates, AD-6, AD-21, FR-2) с
//     метрикой ant_event_to_sse_seconds;
//   - Codec — записи журнала ↔ представление домена (AD-20, AD-44).
//
// Слой: application (AD-1). Ведомые порты: WorkFeed, BundleSource, Sealer,
// ProjectionStore, ChangeLog (здесь) и JournalStore, LeaseStore, Consumer
// (application/journal). Адаптеры Postgres — infrastructure/storage/engine
// (проекции, вклады, журнал изменений, LISTEN/NOTIFY) и эпик 04 (журнал);
// фейки в памяти для тестов — enginemem.
//
// Требования: FR-2, FR-32, FR-40, FR-112, FR-115, FR-124, NFR-DET-1; AD-3,
// AD-4, AD-5, AD-6, AD-22, AD-39, AD-40, AD-42, AD-45.
// Владелец: эпик 07 (движок, воркер, проекции).
package engine
