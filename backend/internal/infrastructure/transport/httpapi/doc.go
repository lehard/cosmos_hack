// Пакет httpapi — общая рамка операций HTTP API на Huma (AD-20, AD-21, AD-36,
// AD-40; FR-110, FR-111; NFR-DEV-2): создание API, регистрация операций с
// x-ant-action, соглашения чтения (axis, as_of, run_id) и команд (command_id,
// basis_seq, policy_seq), режим fixtures | live в заголовке Ant-Backend,
// ошибки RFC 9457 problem+json с кодами contracts/errors.yaml, SSE-операция,
// выгрузка contracts/openapi.yaml.
//
// Слой: infrastructure/transport. Опирается только на порты application;
// другие зоны не импортирует (AD-1). Операции модулей регистрирует
// transport/‹модуль› через Register; общий декоратор приложения
// (application/access.Gate — права, место сеанса, гарды, допустимые действия)
// вызывается здесь для каждой операции одинаково — и на заготовках, и вживую.
//
// Ответы об ошибках описаны одним default: Problem (коды — в errors.yaml).
//
// Проверки контракта (make check): Register паникует на операции без класса,
// с чужим эмитируемым типом или id не того модуля — выгрузка openapi.yaml
// падает; то же проверяет contracts/scripts/check-openapi.mjs по файлу.
package httpapi
