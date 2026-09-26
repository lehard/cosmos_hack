// Пакет telemetry — адаптер порта platform.Telemetry (ключ telemetry =
// prometheus; AD-35): /metrics Prometheus в MVP — операционные метрики (задержка
// приёма, дубли, карантин, ant_event_to_sse_seconds), не проекции (AD-7). Замена —
// OTLP (OpenTelemetry), тест — экспортёр в память.
//
// Слой: infrastructure/observability — технический механизм. Владелец: эпик 34.
package telemetry
