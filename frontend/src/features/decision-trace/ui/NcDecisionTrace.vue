<script setup lang="ts">
/**
 * Дорожка решения в карточке несоответствия: сигнал и анализ системы из
 * карточки (`NCCard`), то же наблюдение анализатора (`vision.observation.read`,
 * если сигнал — результат контроля), паспорт допуска и правило карты реакций
 * по номеру из вывода системы — в DecisionTrace.
 *
 * Встраивание — одной строкой: `<NcDecisionTrace :card="card" />`.
 */
import { computed } from 'vue'
import { useAnalyzerPassport, useObservationAccount } from '@/entities/analyzer-passport'
import type { NCCard } from '@/shared/api/generated/model'
import { useReactionRule } from '../model/queries'
import { observationEventOf, traceFromNc } from '../model/trace'
import DecisionTrace from './DecisionTrace.vue'

const props = defineProps<{ card: NCCard }>()

const obsQ = useObservationAccount(() => observationEventOf(props.card))
const obs = computed(() => obsQ.data.value?.data ?? null)
const passportQ = useAnalyzerPassport(() => obs.value?.passport_id ?? null)
const { rule } = useReactionRule(() => props.card.system_analysis.versions.at(-1)?.rule_id)
const trace = computed(() => (props.card.evidence.signals.length ? traceFromNc(props.card, { observation: obs.value, passport: passportQ.data.value?.data ?? null, rule: rule.value }) : null))
</script>

<template>
  <DecisionTrace v-if="trace" :trace="trace" />
</template>
