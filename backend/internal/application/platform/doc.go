// Пакет platform — общие соглашения операций и ведомые порты платформы слоя
// application (AD-35, AD-36, AD-39, AD-21, AD-22, AD-37).
//
// Слой: application. Импортирует только domain/kernel, сгенерированные
// контракты и stdlib; HTTP-фреймворка и драйверов БД здесь нет (AD-1).
//
// Что здесь:
//   - соглашения операций: момент чтения (axis, as_of, run_id), метаданные
//     команды (command_id, basis_seq, policy_seq), квитанция команды, режим
//     fixtures | live, описание операции x-ant-action (id, класс, критичность,
//     модуль-владелец, гарды, эмитируемые типы), ошибки портов;
//   - субъект запроса (Principal) в контексте;
//   - ведомые порты платформы без модуля-владельца: Publisher, Telemetry,
//     FixtureCursor — с ключами конфигурации (AD-35, таблица PortKeys).
//
// Порты модулей-владельцев объявлены в их пакетах: JournalStore, LeaseStore,
// Consumer, DomainClock, InfraClock — application/journal; MaterialStore —
// application/materials; WorkFeed — application/engine; Signer, Verifier,
// Cipher — application/signing; AccessControl, IdentityProvider —
// application/access.
package platform
