// Пакет analytics — сценарии модуля analytics слоя application: показатели
// кейса как агрегаты строк вклада изделий, счётчики узлов, ограничение линии,
// аномалии и контрольные карты (FR-2, FR-3, FR-5, FR-86…FR-89; кейс §2.4, §5.2).
//
// Слой: application (AD-1). Ведущие порты Queries и Commands (live — Service,
// fixtures — infrastructure/fixtures/analytics, AD-36); ведомые порты Store,
// Clock, Norms. Подключение к движку — Register (AddContributor и глобальные
// проекции analytics.equipment, analytics.incident; вызывает engineRegistry в
// cmd/ant): воркер при каждой пересвёртке изделия заменяет его строки вклада
// целиком в той же транзакции, что курсор (AD-45); агрегат — сумма строк.
// HTTP-фреймворка и драйверов БД здесь нет: операции регистрирует
// infrastructure/transport/analytics, чтение строк — infrastructure/storage/analytics.
//
// Требования: FR-2, FR-3, FR-5, FR-7, FR-86…FR-89, FR-140, AD-22, AD-45.
// Владелец: эпик 25 (аналитика).
package analytics
