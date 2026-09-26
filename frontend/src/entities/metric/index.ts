/**
 * Показатели (эпик 15; FR-5, FR-7, FR-86…FR-89, FR-143; кейс §2.4, §5.2):
 * чтение операций модуля analytics, фокус аналитики (период и выбранное
 * число), правила раскладки и форматирования, общие элементы числа.
 *
 * Отдельного вида сущности «показатель» в контракте SSE нет: ключи кэша —
 * под `live_map` (объект прав операций analytics), см. api.ts.
 */
export * from './model/value'
export * from './model/catalog'
export * from './model/overview'
export * from './model/chart'
export * from './model/focus'
export * from './api'
export { default as MetricNumber } from './ui/MetricNumber.vue'
export { default as OriginMark } from './ui/OriginMark.vue'
export { default as PeriodPicker } from './ui/PeriodPicker.vue'
