// Пакет reference — производственные сценарии модуля reference слоя application: справочники с версиями: номенклатура, места, оборудование и поверка, календарь, смены, соответствия внешних ID.
//
// Слой: application (AD-1). Объявляет ведущие порты Queries и Commands
// (реализации: live — Service здесь, fixtures — infrastructure/fixtures/reference,
// AD-36) и ведомые порты BookSource (справочник из журнала — JournalSource)
// и Writer (решения в журнал — JournalWriter); вызывает domain/reference.
// HTTP-фреймворка и драйверов БД здесь нет: операции регистрирует
// infrastructure/transport/reference.
//
// Подключение к потребителям (AD-31): Bundles — внешний слой нормативного
// слоя изделия (срез на basis_seq: поверка и квалификации для предусловий
// процесса, производственный календарь сроков notifications); WorkingCalendar
// — порт календаря nonconformity; затравка генезисом — SeedRecords и Seed.
//
// Требования: FR-17, FR-55, FR-80, FR-81, FR-95, AD-31, AD-18.
// Владелец после волны 1: эпик 19 (справочники, календарь, смены).
package reference
