<script setup lang="ts">
/**
 * Дорожка решения по наблюдению анализатора (паспорт изделия): наблюдение
 * `vision.observation.read` и паспорт допуска `vision.passport.read` — в
 * DecisionTrace. Нет ответа — честная строка ошибки, а не выдуманная цепочка.
 */
import { computed } from 'vue'
import { useAnalyzerPassport, useObservationAccount } from '@/entities/analyzer-passport'
import { useProblemText } from '@/shared/i18n/problem'
import { traceFromObservation } from '../model/trace'
import DecisionTrace from './DecisionTrace.vue'

const props = withDefaults(defineProps<{ eventId: string; zone?: string }>(), { zone: undefined })
const problemText = useProblemText()

const obsQ = useObservationAccount(() => props.eventId)
const obs = computed(() => obsQ.data.value?.data ?? null)
const passportQ = useAnalyzerPassport(() => obs.value?.passport_id ?? null)
const trace = computed(() => (obs.value ? traceFromObservation(obs.value, passportQ.data.value?.data ?? null, { zone: props.zone }) : null))
</script>

<template>
  <DecisionTrace v-if="trace" :trace="trace" />
  <p v-else-if="obsQ.error.value" class="error ant-wrap" data-testid="decision-trace-error">{{ problemText(obsQ.error.value) }}</p>
</template>

<style scoped>
.error {
  margin: 0;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}
</style>
