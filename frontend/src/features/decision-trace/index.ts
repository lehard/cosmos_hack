/**
 * Фича «Как машина пришла к выводу» (Ф1 сценария показа SHOW-IS2; кейс §2.2,
 * §4.5, §5.4) — публичный вход (FSD): дорожка решения анализатора и её сборка
 * из наблюдения (паспорт изделия) или карточки несоответствия.
 */
export * from './model/trace'
export { useReactionRule } from './model/queries'
export { default as DecisionTrace } from './ui/DecisionTrace.vue'
export { default as ObservationDecisionTrace } from './ui/ObservationDecisionTrace.vue'
export { default as NcDecisionTrace } from './ui/NcDecisionTrace.vue'
