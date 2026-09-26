// Пакет ingest — хранение модуля ingest (зона storage, AD-1): своя схема
// Postgres «ingest» (реестр идемпотентности source_id + event_id с отпечатком
// payload, учёт source_seq по источникам, карантин сообщений) и свои миграции
// goose (migrations/, версии — метки времени). Запросы — pgx, как у журнала
// эпика 04 (sqlc не понадобился: пять простых таблиц).
//
// Строки пишутся в транзакции journal.Append (AppendRequest.Project → Tx из
// контекста, AD-45): факт в журнале и ключ в реестре фиксируются вместе.
// Чужих таблиц не читает; в журнал пишет только через порт JournalStore (AD-44).
//
// Слой: infrastructure/storage; реализует ведомые порты application/ingest
// Registry и QuarantineStore.
// Владелец: эпик 06 (приём и edge-агент).
package ingest
