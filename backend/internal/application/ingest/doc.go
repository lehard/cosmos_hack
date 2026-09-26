// Пакет ingest — производственные сценарии приёма событий (кейс §3.2, §4.3–4.7):
// конвейер «подпись → схема теми же JSON Schema → пять случаев изменения
// контракта → идемпотентность → source_seq → карантин с записью факта в журнал
// → флаги часов и последовательности → привязка к изделию → запись в журнал»;
// ручной ввод и импорт CSV — такие же источники; переобработка карантина;
// метрики приёма; учёт разрывов source_seq для сигнала «потеря данных источника».
//
// Слой: application (AD-1). Объявляет ведущие порты Queries и Commands
// (реализации: live — Service здесь, fixtures — infrastructure/fixtures/ingest,
// AD-36) и ведомые порты модуля (Registry, QuarantineStore, KeyRegistry,
// SecurityBus, StandControl); пользуется портами платформы JournalStore,
// DomainClock, InfraClock (application/journal), MaterialStore
// (application/materials), Verifier (application/signing), Telemetry
// (application/platform); правила — domain/ingest, разрешение носителя —
// domain/crossitem.ResolveCarrier. HTTP-фреймворка и драйверов БД здесь нет.
//
// Проверка схем — santhosh-tekuri/jsonschema над теми же файлами contracts/,
// что встроены в бинарник (AD-20, internal/contracts/schemas).
//
// Требования: FR-26…FR-41, FR-123, FR-140, FR-141, AD-2, AD-7, AD-18, AD-20, AD-41.
// Владелец: эпик 06 (приём и edge-агент).
package ingest
