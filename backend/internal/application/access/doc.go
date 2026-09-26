// Пакет access — сценарии модуля access слоя application: вход и сеанс,
// демо-персоны, заявка на регистрацию и активация учётной записи (FR-128),
// сотрудники и роли (FR-78), столы ролей, права «кто — что — над чем — где —
// когда» (FR-85), допустимые действия и объяснение прав.
//
// Состав:
//   - Gate — общий декоратор над ведущими портами всех модулей (AD-36): права
//     по x-ant-action, место операции (рабочее место команды, объекта или
//     сеанса), гарды; плоский список прав фронтенда тем же Enforce;
//   - Projection — проекция политики из журнала (policy.*, access.*) поверх
//     затравки; PrincipalOf — субъект сеанса по политике с наследованием ролей
//     (одна функция — domain/access.Roles.Closure);
//   - Service — живые операции access; JournalDecisions — запись решений.
//
// Ведомые порты: AccessControl (Casbin — infrastructure/security/casbin),
// IdentityProvider (infrastructure/security/identity), PolicyLog,
// CredentialStore, SecurityEvents (infrastructure/storage/access), Places,
// PasswordHasher.
//
// Слой: application (AD-1). HTTP-фреймворка и Casbin здесь нет: операции
// регистрирует transport/access.
//
// Требования: FR-78, FR-85, FR-128, FR-136, FR-146; AD-15, AD-24, AD-36, AD-39.
// Владелец после волны 1: эпик 08 (ядро), 26 (политика), 37 (СКУД, допуск).
package access
