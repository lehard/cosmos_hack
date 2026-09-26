// СГЕНЕРИРОВАНО contracts/scripts/gen-ts.mjs (make generate) — руками не править (AD-20).
// Источник: contracts/crypto/*.schema.json


/**
 * Содержимое контрольной точки хранителя (класс checkpoint, профиль hybrid, AD-8): головы обеих цепочек, звенья, диапазон committed_at, отпечаток предыдущей точки, время хранителя. Закрытая схема.
 * 
 * This interface was referenced by `CryptoContracts`'s JSON-Schema
 * via the `definition` "KeeperCheckpoint".
 */
export interface KeeperCheckpoint {
/**
 * Версия формата пакета.
 */
format_version: number
/**
 * Профиль контрольных точек — hybrid (AD-32).
 */
crypto_profile: "hybrid"
/**
 * Ключи хранителя (ГОСТ и ML-DSA).
 * 
 * @minItems 2
 * @maxItems 2
 */
signers: [string, string]
/**
 * Номер контрольной точки.
 */
checkpoint_no: number
/**
 * Головы цепочек main и ca.
 * 
 * @minItems 1
 * @maxItems 2
 */
heads: [{
/**
 * Цепочка.
 */
chain: ("main" | "ca")
/**
 * Номер головы.
 */
seq: number
/**
 * Звено головы.
 */
link: string
}]|[{
/**
 * Цепочка.
 */
chain: ("main" | "ca")
/**
 * Номер головы.
 */
seq: number
/**
 * Звено головы.
 */
link: string
}, {
/**
 * Цепочка.
 */
chain: ("main" | "ca")
/**
 * Номер головы.
 */
seq: number
/**
 * Звено головы.
 */
link: string
}]
/**
 * Наименьший committed_at покрытых записей.
 */
committed_at_from: string
/**
 * Наибольший committed_at покрытых записей.
 */
committed_at_to: string
/**
 * Отпечаток предыдущей точки; null у первой.
 */
previous_checkpoint_digest: (string | null)
/**
 * Отпечаток генезиса — обязателен в первой точке каждой цепочки (AD-33).
 */
genesis_digest?: string
/**
 * Время по часам хранителя.
 */
keeper_time: string
}
/**
 * Конверт DSSE v1 (AD-10): payloadType, payload (base64 канонического JSON RFC 8785) и подписи над PAE. Имена полей — как в спецификации DSSE. Закрытая схема.
 * 
 * This interface was referenced by `CryptoContracts`'s JSON-Schema
 * via the `definition` "DsseEnvelope".
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
