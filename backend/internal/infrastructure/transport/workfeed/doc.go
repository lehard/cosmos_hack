// Пакет workfeed — адаптер подачи работы воркеру (порт application/engine.WorkFeed,
// ключ work_feed = postgres; AD-6, AD-35): фиксированные партиции hash(item_id)
// mod P с арендой в Postgres. Замена — Kafka с key = item_id (описание,
// docs/observability-kafka-otel.md): тот же порт, контрактный тест + state_hash.
//
// Слой: infrastructure/transport. Владелец: эпик 07 (движок, воркер).
package workfeed
