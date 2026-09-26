// Пакет security — производственные сценарии модуля security слоя application: сервис доверенных решений (CriticalActions), журнал критических действий, шина безопасности, индикатор целостности.
//
// Слой: application (AD-1). Объявляет ведущие порты Queries и Commands
// (реализации: live — Service здесь, fixtures — infrastructure/fixtures/security,
// AD-36) и ведомые порты модуля; вызывает domain/security. HTTP-фреймворка и
// драйверов БД здесь нет: операции регистрирует infrastructure/transport/security,
// хранение — infrastructure/storage/security.
//
// Что здесь (эпик 29):
//   - critical.go — сервис доверенных решений: Execute (единственный путь
//     критического действия) и CriticalHook — запись цепочки ca в транзакции
//     journal.Append (domain/security.BuildCA, AD-28);
//   - bus.go, events.go — шина безопасности поверх журнала: IngestBus (порт
//     ingest.SecurityBus), Emitter, подписчики со своими курсорами, экспорт
//     JSON-строк (AD-24);
//   - keeper.go, trust.go — протокол хранителя, передача голов (HeadsSender),
//     отчёты верификатора и тревоги хранителя в журнал (IntegrityPoller, AD-8, AD-46);
//   - service.go — живые операции security.* (журнал CA, шина, индикатор, отчёты);
//   - verify — независимый верификатор (AD-9), его исполняет cmd/verifier.
//
// Требования: FR-72…FR-77, FR-118, FR-146, AD-8, AD-9, AD-24, AD-28, AD-46.
// Владелец после волны 1: эпик 29 (доверие).
package security
