/**
 * Фича «Остановить пост» (FR-49; nonconformity.process_hold.set / release —
 * сдерживание процесса на оборудовании, подпись уровня 2) — публичный вход (FSD).
 * Ставится в окно поста для каждого оборудования поста; кнопка — по праву
 * сервера над оборудованием.
 */
export { default as ProcessHoldAction } from './ui/ProcessHoldAction.vue'
