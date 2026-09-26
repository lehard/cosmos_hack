/**
 * Фича «Установить сдерживание» (FR-49; операция nonconformity.containment.set,
 * защитное критическое действие, подпись уровня 2) — публичный вход (FSD).
 * Ставится в окно изделия: кнопка видна тому, кому сервер разрешает действие
 * над этим изделием.
 */
export { default as ContainmentAction } from './ui/ContainmentAction.vue'
