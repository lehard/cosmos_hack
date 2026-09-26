// СГЕНЕРИРОВАНО contracts/scripts/gen-ts.mjs (make generate) — руками не править (AD-20).
// Источник: contracts/journal/entry.schema.json


/**
 * Запись журнала v1 (AD-44): открытый заголовок (маршрутизация, время, тип, изделие, правило, слот, обязательство и звено) и зашифрованный блок с исходным конвертом DSSE и salt (AD-23). Go-типы генерируются и используются всеми, включая верификатор. Формат цепочки — chain-format.v1.md.
 * 
 * This interface was referenced by `JournalContracts`'s JSON-Schema
 * via the `definition` "JournalEntry".
 */
export interface JournalEntry {
/**
 * Позиция в своей цепочке (`chain`): порядок знания (AD-37); для ca — номер CA-‹n›.
 */
seq: number
/**
 * Цепочка: основной журнал или журнал критических действий (AD-8).
 */
chain: ("main" | "ca")
/**
 * Вид записи (AD-2); совпадает с каталогом для event_type.
 */
entry_kind: ("fact" | "reaction" | "decision" | "service")
/**
 * Тип записи из catalog.yaml.
 */
event_type: string
/**
 * Мажорная версия схемы data в исходном конверте (повышатели применяются при чтении, AD-20).
 */
schema_version: number
/**
 * event_id из конверта: UUIDv7; у реакций, производных и событий сценария — UUIDv5.
 */
event_id: string
/**
 * Источник.
 */
source_id: string
/**
 * Номер у источника.
 */
source_seq?: number
/**
 * Прогон сценария (AD-38).
 */
run_id?: string
/**
 * Внутренний ID изделия — только внутренний (AD-41).
 */
item_id?: string
/**
 * Значение носителя из item_ref источника: `‹тип›:‹значение›` (AD-41).
 */
carrier_ref?: string
/**
 * Поток (AD-39): `item:‹item_id›`, `‹вид›:‹id›` или `global`.
 */
stream: string
/**
 * Партиция: hash(item_id) mod P; для записей вне изделия — партиция стадии.
 */
partition: number
/**
 * Время возникновения (из конверта; для решений и реакций ставит ядро, AD-37).
 */
occurred_at: string
/**
 * Время поступления в приём — ставит только ядро.
 */
received_at: string
/**
 * Доменное время записи (DomainClock), не убывает по seq (AD-37).
 */
recorded_at: string
/**
 * Реальное время фиксации (InfraClock), не убывает по seq, покрыто контрольными точками.
 */
committed_at: string
/**
 * Сквозная цепочка.
 */
correlation_id: string
/**
 * Непосредственная причина; null у корневой записи.
 */
causation_id: (string | null)
/**
 * Правило реакции (AD-3).
 */
rule_id?: string
/**
 * Ревизия нормативного слоя, по которой исполнено (хеш версии процесса или номер).
 */
normative_rev?: string
/**
 * Слот реакции (AD-3).
 */
reaction_slot?: {
/**
 * Правило.
 */
rule_id: string
/**
 * Субъект — поток.
 */
subject: string
/**
 * Ключ срабатывания.
 */
trigger_key: string
}
/**
 * Версия слота реакции.
 */
version?: number
/**
 * Заменяемая версия слота.
 */
supersedes?: (string | null)
/**
 * Реакции — основание пачки; решения — позиция журнала, на которой проверен гард (AD-39).
 */
basis_seq?: number
/**
 * Ревизия политики доступа, по которой проверена команда (AD-39).
 */
policy_seq?: number
/**
 * Класс происхождения подписи (AD-2).
 */
provenance_class: ("device" | "personal" | "paper" | "partner" | "server_attested" | "scenario" | "genesis")
/**
 * Хеш доменного пакета, свернувшего запись (AD-9).
 */
domain_build: string
/**
 * Обязательство: H(salt ‖ JCS(конверт DSSE)) (AD-23, AD-44).
 */
commit: string
/**
 * Звено: H(prev_link ‖ commit ‖ H(JCS(открытые поля без link))) (AD-44).
 */
link: string
sealed: SealedBlock
}
/**
 * Зашифрованный блок (AD-23): AEAD над JCS(plain_block); доп. данные — chain ‖ seq ‖ commit.
 * 
 * This interface was referenced by `JournalEntry`'s JSON-Schema
 * via the `definition` "sealed_block".
 */
export interface SealedBlock {
/**
 * Алгоритм AEAD: «Кузнечик»-MGM (ГОСТ) или AES-256-GCM.
 */
aead: ("kuznyechik_mgm" | "aes_256_gcm")
/**
 * Ключ шифрования данных; обёртки — journal.dek_wraps.
 */
dek_id: string
/**
 * Нонс.
 */
nonce_b64: string
/**
 * Шифротекст с тегом.
 */
ciphertext_b64: string
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
