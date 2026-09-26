/* eslint-disable */
// СГЕНЕРИРОВАНО contracts/scripts/gen-ts.mjs (make generate) — руками не править (AD-20).
// Источник: contracts/internal/**/*.json


/**
 * Сообщение stand-а оборудования (станок ЧПУ, сварочный источник, стенд) → edge-агенту (AD-18, AD-46, FR-149): отсчёты и события по классификации MTConnect. Сырые отсчёты остаются на краю; в ant уходят события и сводки на окно цикла (`equipment.*`). Закрытая схема.
 * 
 * This interface was referenced by `ProcsContracts`'s JSON-Schema
 * via the `definition` "StandTelemetryV1".
 */
export interface StandTelemetryV1 {
/**
 * Идентификатор stand-а.
 */
stand_id: string
/**
 * Оборудование.
 */
equipment_id: string
/**
 * Прогон сценария.
 */
run_id?: string
/**
 * Номер сообщения stand-а.
 */
seq: number
/**
 * Доменное время сценария.
 */
sent_at: string
/**
 * Отсчёты.
 * 
 * Items: Отсчёт SAMPLE.
 */
samples?: {
/**
 * Параметр.
 */
parameter: string
/**
 * Время.
 */
at: string
/**
 * Мантисса.
 */
value: number
/**
 * Масштаб.
 */
scale: number
/**
 * Единица UCUM.
 */
unit: string
}[]
/**
 * События.
 * 
 * Items: Событие EVENT/CONDITION.
 */
events?: {
/**
 * Категория.
 */
category: ("execution" | "controller_mode" | "condition" | "program" | "tool" | "cycle" | "alarm")
/**
 * Значение (например, running, manual, warning, программа, инструмент).
 */
value: string
/**
 * Время.
 */
at: string
/**
 * Код.
 */
code?: string
/**
 * Ресурс инструмента: сделано.
 */
tool_life_used?: number
/**
 * Ресурс инструмента: допустимо.
 */
tool_life_limit?: number
}[]
/**
 * Маркер цикла.
 */
cycle?: {
/**
 * Цикл.
 */
cycle_ref: string
/**
 * Фаза.
 */
phase: ("start" | "end")
}
}
/**
 * Сообщение браузерного расширения → агенту токена по Native Messaging (AD-14, AD-46). Вид — поле `type`; для каждого вида обязателен свой блок (без oneOf, диалект AD-20). Хост Native Messaging — тонкий посредник; PIN вводится только в окне агента, никогда на странице.
 * 
 * This interface was referenced by `ProcsContracts`'s JSON-Schema
 * via the `definition` "TokenAgentRequestV1".
 */
export interface TokenAgentRequestV1 {
/**
 * Версия протокола.
 */
protocol_version: number
/**
 * Вид запроса.
 */
type: ("hello" | "status" | "sign" | "sign_batch" | "shift_report" | "local_journal")
/**
 * Идентификатор запроса расширения.
 */
request_id: string
/**
 * Origin страницы ant, от которой пришёл запрос (проверяет расширение).
 */
origin: string
sign?: SignBlock
/**
 * Пачка однородных подписей уровня 2: «подписать: N изделий, операция X, годен» — одно окно, одно касание токена (AD-13).
 * 
 * @minItems 1
 * @maxItems 500
 */
sign_batch?: [SignBlock, ...(SignBlock)[]]
/**
 * Запрос сменного рапорта (уровень 3).
 */
shift_report?: {
/**
 * Смена.
 */
shift_id: string
/**
 * Начало окна.
 */
window_from: string
/**
 * Конец окна.
 */
window_to: string
}
/**
 * Запрос выписки локального журнала подписанного.
 */
local_journal?: {
/**
 * С какого номера локального журнала.
 */
since_seq: number
}
}
/**
 * Запрос одной подписи: агент сам разбирает содержимое, сам считает отпечаток пакетом domain/documents и сам вычисляет поля сводки (AD-14).
 * 
 * This interface was referenced by `TokenAgentRequestV1`'s JSON-Schema
 * via the `definition` "sign_block".
 */
export interface SignBlock {
/**
 * Уровень подписи 1–2; уровень 1 — только типы из level1-actions.yaml, иначе агент отклоняет сам.
 */
level: number
/**
 * payloadType пакета.
 */
payload_type: string
/**
 * Каноническое содержимое (base64): событие-команда или содержимое документа.
 */
payload_b64: string
/**
 * Тип события-команды (для проверки перечня уровня 1).
 */
event_type?: string
/**
 * Шаблон документа `‹id›@‹версия›`, если подписывается документ.
 */
template_ref?: string
/**
 * Версия формата документа; неизвестную агент отвергает (signing.unknown_doc_format).
 */
doc_format_version?: number
/**
 * Отпечаток, который ожидает сервер; агент сверяет со своим.
 */
expected_doc_digest?: string
/**
 * Ключ, которым просят подписать.
 */
key_ref?: string
}
/**
 * Ответ агента токена расширению по Native Messaging (AD-14, AD-46). Вид — поле `type`.
 * 
 * This interface was referenced by `ProcsContracts`'s JSON-Schema
 * via the `definition` "TokenAgentResponseV1".
 */
export interface TokenAgentResponseV1 {
/**
 * Версия протокола.
 */
protocol_version: number
/**
 * Вид ответа.
 */
type: ("hello" | "status" | "signed" | "shift_report" | "local_journal" | "error")
/**
 * Запрос, на который ответ.
 */
request_id: string
/**
 * Приветствие агента.
 */
hello?: {
/**
 * Версия агента.
 */
agent_version: string
/**
 * Хеш сборки: агент, сервер и demo-signer — из одного коммита (AD-12).
 */
build_digest: string
/**
 * Поддерживаемые версии формата документа.
 * 
 * @minItems 1
 */
doc_format_versions: [number, ...(number)[]]
/**
 * Ключи.
 * 
 * Items: Ключ на токене.
 */
keys: {
/**
 * Ключ.
 */
key_ref: string
/**
 * Профиль.
 */
profile: ("gost" | "pq")
/**
 * Субъект.
 */
person_id: string
}[]
}
/**
 * Состояние токена.
 */
status?: {
/**
 * Токен вставлен.
 */
token_present: boolean
/**
 * PIN введён в этой смене.
 */
pin_unlocked: boolean
/**
 * Владелец.
 */
person_id?: string
/**
 * Рабочее место из конфигурации агента.
 */
workplace_id?: string
}
/**
 * Подписанные пакеты (один или пачка).
 * 
 * @minItems 1
 * 
 * Items: Результат подписи.
 */
signed?: [{
envelope: DsseEnvelope
/**
 * Отпечаток, посчитанный агентом.
 */
doc_digest?: string
/**
 * Поля сводки уровня 2 (3–7), которые агент вычислил сам из содержимого.
 * 
 * @maxItems 7
 * 
 * Items: Поле сводки.
 */
summary?: []|[{
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}]|[{
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}]|[{
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}]|[{
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}]|[{
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}]|[{
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}]|[{
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}]
/**
 * Номер в локальном журнале подписанного.
 */
local_journal_seq: number
/**
 * Время подписи по часам клиента — справочно, в порядок не входит (AD-37).
 */
client_signed_at: string
}, ...({
envelope: DsseEnvelope
/**
 * Отпечаток, посчитанный агентом.
 */
doc_digest?: string
/**
 * Поля сводки уровня 2 (3–7), которые агент вычислил сам из содержимого.
 * 
 * @maxItems 7
 * 
 * Items: Поле сводки.
 */
summary?: []|[{
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}]|[{
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}]|[{
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}]|[{
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}]|[{
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}]|[{
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}]|[{
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}, {
/**
 * Подпись поля.
 */
label: string
/**
 * Значение.
 */
value: string
}]
/**
 * Номер в локальном журнале подписанного.
 */
local_journal_seq: number
/**
 * Время подписи по часам клиента — справочно, в порядок не входит (AD-37).
 */
client_signed_at: string
})[]]
shift_report?: ShiftReportV1
/**
 * Локальный журнал подписанного.
 * 
 * Items: Запись локального журнала подписанного.
 */
local_journal?: {
/**
 * Номер.
 */
seq: number
/**
 * Тип.
 */
event_type?: string
/**
 * Отпечаток подписанного.
 */
digest: string
/**
 * Когда.
 */
signed_at: string
/**
 * seq журнала сервера из квитанции приёма, если получена.
 */
server_seq?: number
}[]
/**
 * Отказ агента.
 */
error?: {
/**
 * Код из contracts/errors.yaml (например, signing.level_not_allowed, signing.token_missing).
 */
code: string
/**
 * Пояснение.
 */
message?: string
}
}
/**
 * Конверт DSSE v1 (AD-10): payloadType, payload (base64 канонического JSON RFC 8785) и подписи над PAE. Имена полей — как в спецификации DSSE. Закрытая схема.
 */
export interface DsseEnvelope {
/**
 * Тип содержимого: application/vnd.ant.‹класс›+json; v=‹версия›.
 */
payloadType: string
/**
 * Base64 (стандартный алфавит, с выравниванием) канонических байтов содержимого.
 */
payload: string
/**
 * Подписи над PAE(payloadType, payload); для hybrid — две (ГОСТ и ML-DSA-65). Лишние и повторные не засчитываются.
 * 
 * @minItems 1
 * @maxItems 8
 */
signatures: [{
/**
 * Ключ подписанта: key_id@версия.
 */
keyid: string
/**
 * Подпись в base64; кодирование по профилю — contracts/crypto/README.md.
 */
sig: string
}]|[{
/**
 * Ключ подписанта: key_id@версия.
 */
keyid: string
/**
 * Подпись в base64; кодирование по профилю — contracts/crypto/README.md.
 */
sig: string
}, {
/**
 * Ключ подписанта: key_id@версия.
 */
keyid: string
/**
 * Подпись в base64; кодирование по профилю — contracts/crypto/README.md.
 */
sig: string
}]|[{
/**
 * Ключ подписанта: key_id@версия.
 */
keyid: string
/**
 * Подпись в base64; кодирование по профилю — contracts/crypto/README.md.
 */
sig: string
}, {
/**
 * Ключ подписанта: key_id@версия.
 */
keyid: string
/**
 * Подпись в base64; кодирование по профилю — contracts/crypto/README.md.
 */
sig: string
}, {
/**
 * Ключ подписанта: key_id@версия.
 */
keyid: string
/**
 * Подпись в base64; кодирование по профилю — contracts/crypto/README.md.
 */
sig: string
}]|[{
/**
 * Ключ подписанта: key_id@версия.
 */
keyid: string
/**
 * Подпись в base64; кодирование по профилю — contracts/crypto/README.md.
 */
sig: string
}, {
/**
 * Ключ подписанта: key_id@версия.
 */
keyid: string
/**
 * Подпись в base64; кодирование по профилю — contracts/crypto/README.md.
 */
sig: string
}, {
/**
 * Ключ подписанта: key_id@версия.
 */
keyid: string
/**
 * Подпись в base64; кодирование по профилю — contracts/crypto/README.md.
 */
sig: string
}, {
/**
 * Ключ подписанта: key_id@версия.
 */
keyid: string
/**
 * Подпись в base64; кодирование по профилю — contracts/crypto/README.md.
 */
sig: string
}]|[{
/**
 * Ключ подписанта: key_id@версия.
 */
keyid: string
/**
 * Подпись в base64; кодирование по профилю — contracts/crypto/README.md.
 */
sig: string
}, {
/**
 * Ключ подписанта: key_id@версия.
 */
keyid: string
/**
 * Подпись в base64; кодирование по профилю — contracts/crypto/README.md.
 */
sig: string
}, {
/**
 * Ключ подписанта: key_id@версия.
 */
keyid: string
/**
 * Подпись в base64; кодирование по профилю — contracts/crypto/README.md.
 */
sig: string
}, {
/**
 * Ключ подписанта: key_id@версия.
 */
keyid: string
/**
 * Подпись в base64; кодирование по профилю — contracts/crypto/README.md.
 */
sig: string
}, {
/**
 * Ключ подписанта: key_id@версия.
 */
keyid: string
/**
 * Подпись в base64; кодирование по профилю — contracts/crypto/README.md.
 */
sig: string
}]|[{
/**
 * Ключ подписанта: key_id@версия.
 */
keyid: string
/**
 * Подпись в base64; кодирование по профилю — contracts/crypto/README.md.
 */
sig: string
}, {
/**
 * Ключ подписанта: key_id@версия.
 */
keyid: string
/**
 * Подпись в base64; кодирование по профилю — contracts/crypto/README.md.
 */
sig: string
}, {
/**
 * Ключ подписанта: key_id@версия.
 */
keyid: string
/**
 * Подпись в base64; кодирование по профилю — contracts/crypto/README.md.
 */
sig: string
}, {
/**
 * Ключ подписанта: key_id@версия.
 */
keyid: string
/**
 * Подпись в base64; кодирование по профилю — contracts/crypto/README.md.
 */
sig: string
}, {
/**
 * Ключ подписанта: key_id@версия.
 */
keyid: string
/**
 * Подпись в base64; кодирование по профилю — contracts/crypto/README.md.
 */
sig: string
}, {
/**
 * Ключ подписанта: key_id@версия.
 */
keyid: string
/**
 * Подпись в base64; кодирование по профилю — contracts/crypto/README.md.
 */
sig: string
}]|[{
/**
 * Ключ подписанта: key_id@версия.
 */
keyid: string
/**
 * Подпись в base64; кодирование по профилю — contracts/crypto/README.md.
 */
sig: string
}, {
/**
 * Ключ подписанта: key_id@версия.
 */
keyid: string
/**
 * Подпись в base64; кодирование по профилю — contracts/crypto/README.md.
 */
sig: string
}, {
/**
 * Ключ подписанта: key_id@версия.
 */
keyid: string
/**
 * Подпись в base64; кодирование по профилю — contracts/crypto/README.md.
 */
sig: string
}, {
/**
 * Ключ подписанта: key_id@версия.
 */
keyid: string
/**
 * Подпись в base64; кодирование по профилю — contracts/crypto/README.md.
 */
sig: string
}, {
/**
 * Ключ подписанта: key_id@версия.
 */
keyid: string
/**
 * Подпись в base64; кодирование по профилю — contracts/crypto/README.md.
 */
sig: string
}, {
/**
 * Ключ подписанта: key_id@версия.
 */
keyid: string
/**
 * Подпись в base64; кодирование по профилю — contracts/crypto/README.md.
 */
sig: string
}, {
/**
 * Ключ подписанта: key_id@версия.
 */
keyid: string
/**
 * Подпись в base64; кодирование по профилю — contracts/crypto/README.md.
 */
sig: string
}]|[{
/**
 * Ключ подписанта: key_id@версия.
 */
keyid: string
/**
 * Подпись в base64; кодирование по профилю — contracts/crypto/README.md.
 */
sig: string
}, {
/**
 * Ключ подписанта: key_id@версия.
 */
keyid: string
/**
 * Подпись в base64; кодирование по профилю — contracts/crypto/README.md.
 */
sig: string
}, {
/**
 * Ключ подписанта: key_id@версия.
 */
keyid: string
/**
 * Подпись в base64; кодирование по профилю — contracts/crypto/README.md.
 */
sig: string
}, {
/**
 * Ключ подписанта: key_id@версия.
 */
keyid: string
/**
 * Подпись в base64; кодирование по профилю — contracts/crypto/README.md.
 */
sig: string
}, {
/**
 * Ключ подписанта: key_id@версия.
 */
keyid: string
/**
 * Подпись в base64; кодирование по профилю — contracts/crypto/README.md.
 */
sig: string
}, {
/**
 * Ключ подписанта: key_id@версия.
 */
keyid: string
/**
 * Подпись в base64; кодирование по профилю — contracts/crypto/README.md.
 */
sig: string
}, {
/**
 * Ключ подписанта: key_id@версия.
 */
keyid: string
/**
 * Подпись в base64; кодирование по профилю — contracts/crypto/README.md.
 */
sig: string
}, {
/**
 * Ключ подписанта: key_id@версия.
 */
keyid: string
/**
 * Подпись в base64; кодирование по профилю — contracts/crypto/README.md.
 */
sig: string
}]
}
/**
 * Сменный рапорт — уровень подписи 3 (AD-12, AD-13): одна подпись над корнем дерева Меркла RFC 6962 по отпечаткам подписей смены из локального журнала агента; счётчики по типам. Класс пакета `shift-report`. Закрытая схема.
 * 
 * This interface was referenced by `ProcsContracts`'s JSON-Schema
 * via the `definition` "ShiftReportV1".
 */
export interface ShiftReportV1 {
/**
 * Версия формата.
 */
format_version: number
/**
 * Профиль.
 */
crypto_profile: ("gost" | "hybrid")
/**
 * Подписанты.
 * 
 * @minItems 1
 * @maxItems 2
 * 
 * Items: Ключ.
 */
signers: [string]|[string, string]
/**
 * Подписант.
 */
person_id: string
/**
 * Смена.
 */
shift_id: string
/**
 * Рабочее место.
 */
workplace_id?: string
/**
 * Начало окна.
 */
window_from: string
/**
 * Конец окна.
 */
window_to: string
/**
 * Корень: лист H(0x00‖отпечаток подписи), узел H(0x01‖l‖r), листья в порядке подписания.
 */
merkle_root: string
/**
 * Число листьев.
 */
leaf_count: number
/**
 * Сколько подписей каждого типа.
 */
counts_by_type: {
[k: string]: number | undefined
}
/**
 * Первый номер локального журнала в окне.
 */
first_local_seq?: number
/**
 * Последний номер.
 */
last_local_seq?: number
/**
 * Последняя известная агенту контрольная точка.
 */
seen_checkpoint?: number
}
/**
 * Подписанный отчёт независимого верификатора (AD-9, AD-46): класс `verifier-report`, профиль hybrid, ключ из тома verifier. Сдаётся хранителю; ant журналирует только ссылку (`security.integrity.checked`). Закрытая схема.
 * 
 * This interface was referenced by `ProcsContracts`'s JSON-Schema
 * via the `definition` "VerifierReportV1".
 */
export interface VerifierReportV1 {
/**
 * Версия формата.
 */
format_version: number
/**
 * Профиль.
 */
crypto_profile: "hybrid"
/**
 * Подписанты.
 * 
 * @minItems 2
 * @maxItems 2
 * 
 * Items: Ключ верификатора.
 */
signers: [string, string]
/**
 * Хеш бинарника верификатора.
 */
verifier_build: string
/**
 * Перечень допустимых сборок из нормативного слоя.
 * 
 * Items: Хеш допустимой сборки доменного пакета.
 */
domain_builds_allowed?: string[]
/**
 * Когда сформирован (часы верификатора).
 */
generated_at: string
/**
 * Отпечаток файла trust-anchors.
 */
trust_anchors_digest: string
/**
 * Проверенный диапазон.
 */
range: {
/**
 * С.
 */
main_from_seq: number
/**
 * По.
 */
main_to_seq: number
/**
 * Цепочка CA по.
 */
ca_to_seq: number
/**
 * Последняя контрольная точка.
 */
last_checkpoint_no: number
}
/**
 * Прогон сценария, если проверялся он.
 */
run_id?: string
/**
 * Временные проверки — по виртуальному времени сценария.
 */
virtual_time: boolean
/**
 * Общий вердикт: «цело» только при нуле «не проверяемо».
 */
verdict: ("intact" | "intact_with_reservations" | "violated")
/**
 * Проверки.
 * 
 * @minItems 1
 * 
 * Items: Итог одной проверки.
 */
checks: [{
/**
 * Проверка.
 */
check: ("chains" | "checkpoints" | "signatures" | "signing_moment" | "authority" | "bpmn_quorum" | "reactions" | "coverage" | "source_seq" | "rendering" | "late_write" | "build" | "projections" | "genesis")
/**
 * Статус.
 */
status: ("intact" | "rejected" | "not_verifiable")
/**
 * Сколько объектов проверено.
 */
checked: number
/**
 * Находки.
 * 
 * Items: Нарушение или оговорка.
 */
findings: {
/**
 * Запись.
 */
seq?: number
/**
 * Цепочка.
 */
chain?: ("main" | "ca")
/**
 * Номер критического действия CA-‹n›.
 */
ca_ref?: string
/**
 * Код.
 */
code: string
/**
 * Что и где — по-русски.
 */
detail: string
}[]
}, ...({
/**
 * Проверка.
 */
check: ("chains" | "checkpoints" | "signatures" | "signing_moment" | "authority" | "bpmn_quorum" | "reactions" | "coverage" | "source_seq" | "rendering" | "late_write" | "build" | "projections" | "genesis")
/**
 * Статус.
 */
status: ("intact" | "rejected" | "not_verifiable")
/**
 * Сколько объектов проверено.
 */
checked: number
/**
 * Находки.
 * 
 * Items: Нарушение или оговорка.
 */
findings: {
/**
 * Запись.
 */
seq?: number
/**
 * Цепочка.
 */
chain?: ("main" | "ca")
/**
 * Номер критического действия CA-‹n›.
 */
ca_ref?: string
/**
 * Код.
 */
code: string
/**
 * Что и где — по-русски.
 */
detail: string
}[]
})[]]
/**
 * Подписи по классам происхождения — различаются в отчёте (AD-9).
 */
signature_classes: {
/**
 * Подписи устройств.
 */
device: number
/**
 * Личные подписи людей (агент токена).
 */
personal: number
/**
 * Бумажные подписи с заверением.
 */
paper: number
/**
 * Подписи партнёра (федерация).
 */
partner: number
/**
 * Подписи ключами сервера (движок, шлюзы).
 */
server_attested: number
/**
 * Подписи демо-персон сценариев.
 */
scenario: number
/**
 * Подписи блока генезиса.
 */
genesis: number
}
/**
 * Реестр бумажных решений.
 * 
 * Items: Бумажное решение для сверки с оригиналом.
 */
paper_register: {
/**
 * Документ.
 */
document_id: string
/**
 * Отпечаток.
 */
doc_digest: string
/**
 * Учётный номер оригинала.
 */
paper_original_no?: string
/**
 * Подписант.
 */
signer_person_id: string
/**
 * Заверитель.
 */
attested_by: string
/**
 * QR со скана перечитан и совпал.
 */
scan_qr_ok: boolean
}[]
}
