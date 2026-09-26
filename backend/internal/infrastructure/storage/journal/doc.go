// Пакет journal — хранение модуля journal (зона storage, AD-1): журнал —
// единственный источник истины (AD-2) на Postgres.
//
// Что здесь:
//   - Store — адаптер JournalStore: единственная функция записи Append
//     (AD-44): аренда с эпохой, две хеш-цепочки (main и ca) в одной
//     транзакции, звено по формуле AD-44 на Стрибоге-256 (domain/journal),
//     проверки конкурентности AD-39, монотонность committed_at и recorded_at
//     (AD-37), курсор и выход потребителя в той же транзакции (AD-45):
//     эффекты проекций применяют адаптеры хранения модулей-писателей
//     (EngineEffects — storage/engine.ApplyEffect, остальные — WithEffects);
//     чтение по оси и моменту (AD-22), головы цепочек для хранителя (AD-8);
//   - Leases — адаптер LeaseStore: аренды с эпохой по InfraClock (AD-6);
//   - Listener — адаптер Signal: LISTEN/NOTIFY «есть новое» с seq (AD-6);
//   - Migrations — миграции goose схем journal (неизменяемая) и journal_state
//     (аренды, курсоры); роли БД и запуск миграций — пакет migrator.
//
// Подпакеты: feed — потребители (Consumer) и подача работы воркерам
// (WorkFeed); clock — InfraClock и DomainClock; migrator — роли БД и goose.
//
// Слой: infrastructure/storage; реализует ведомые порты application/journal
// и application/engine.WorkFeed; из своей зоны импортирует только
// storage/engine (применение эффектов движка в транзакции Append); другие
// зоны не импортирует. Своя схема
// Postgres — journal (+ journal_state); чужих таблиц не читает. Запросы
// написаны вручную на pgx (sqlc не используется: вставка пачки идёт COPY,
// фильтры чтения составные).
//
// Требования: FR-40, FR-71, FR-122, AD-2, AD-6, AD-8, AD-22, AD-37, AD-39,
// AD-44, AD-45. Кейс §3.4 «Хранение и защита», критерий Т3.
// Владелец после волны 1: эпик 04 (журнал и хранение), 29 (хранитель).
package journal
