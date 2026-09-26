/**
 * Фича подписи решения (FR-66, FR-69, FR-139; AD-13, AD-14, AD-43) — публичный
 * вход (FSD): порт подписи (заглушка до эпика 38), окно подтверждения уровня 2,
 * бумажный путь — печать с QR и заверение скана вторым человеком.
 */
export * from './model/port'
export { default as SignConfirmPanel } from './ui/SignConfirmPanel.vue'
export { default as SignDialog } from './ui/SignDialog.vue'
export { default as PaperSignPanel } from './ui/PaperSignPanel.vue'
