<script setup lang="ts">
/**
 * Виджет «Гипотезы причины» (FR-59, FR-60, FR-135) — контейнер. Довод «за» или
 * «против» по клику подсвечивается на дорожках разбора обстоятельств.
 *
 * Команды — решения человека (AD-39: `basis_seq` ответа + `policy_seq` политики):
 * «подтвердить причину» — `analysis.cause.conclude` (критическое действие по
 * инциденту), «отклонить» — `analysis.hypothesis.reject`, «запросить измерение» —
 * `analysis.measurement.request`. Кнопка доступна, только если действие есть в
 * списке прав сервера (AD-15).
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NAlert } from 'naive-ui'
import { FACTOR_TEXT, hypothesesState, useAnalysisCommands, useAnalysisFocusStore, useCan, type Hypothesis } from '@/entities/incident'
import type { WidgetProps } from '@/shared/config/widget'
import { useProblemText } from '@/shared/i18n/problem'
import { WidgetFrame } from '@/shared/ui'
import { useHypothesesSource } from '../model/source'
import HypothesisView from './HypothesisView.vue'

defineProps<WidgetProps>()

const { t } = useI18n()
const problemText = useProblemText()
const focus = useAnalysisFocusStore()
const src = useHypothesesSource()
const data = computed(() => src.data.value)
const state = computed(() => (data.value ? hypothesesState(data.value) : 'normal'))

const can = useCan()
const cmd = useAnalysisCommands()
const mutations = [cmd.concludeCause, cmd.rejectHypothesis, cmd.requestMeasurement]
const busy = computed(() => mutations.some((m) => m.isPending.value))
const lastError = computed(() => mutations.map((m) => m.error.value).find(Boolean) ?? null)
const sent = computed(() => mutations.some((m) => m.isSuccess.value))

const canConfirm = computed(() => Boolean(src.incidentId.value) && can('analysis.cause.conclude', 'incident', src.incidentId.value))
const canReject = computed(() => can('analysis.hypothesis.reject', 'nonconformity', src.ncId.value))
const canMeasure = computed(() => can('analysis.measurement.request', 'nonconformity', src.ncId.value))

const basisSeq = () => src.basisSeq.value ?? 0
const reset = () => mutations.forEach((m) => m.reset())

function confirm(h: Hypothesis, input: { verification: string; reason: string }): void {
  if (!src.incidentId.value || !src.ncId.value) return
  reset()
  cmd.concludeCause.mutate({
    incidentId: src.incidentId.value,
    body: {
      nc_ids: [src.ncId.value],
      conclusion: 'confirmed',
      category: h.category,
      hypothesis_id: h.hypothesis_id,
      verification: input.verification,
      reason: { text: input.reason },
      basis_seq: basisSeq(),
    },
  })
}

function reject(h: Hypothesis, input: { reason: string }): void {
  if (!src.ncId.value) return
  reset()
  cmd.rejectHypothesis.mutate({ ncId: src.ncId.value, body: { hypothesis_id: h.hypothesis_id, reason: { text: input.reason }, basis_seq: basisSeq() } })
}

function measure(h: Hypothesis, input: { what: string }): void {
  if (!src.ncId.value) return
  reset()
  cmd.requestMeasurement.mutate({ ncId: src.ncId.value, body: { hypothesis_id: h.hypothesis_id, what: input.what, basis_seq: basisSeq() } })
}

/** Фактор, из которого технолог вошёл в гипотезу (FR-135). */
const fromFactor = computed(() => {
  const f = focus.factor
  if (!f || f.intent !== 'hypothesis') return null
  return `${t(FACTOR_TEXT[f.row.factor])}: ${f.row.value ?? t('common.words.unknown')}`
})
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="src.mode.value"
    :state="state"
    :loading="src.isPending.value"
    :error="src.error.value"
    :empty="!data"
    :empty-key="src.ncId.value ? 'empty.noHypotheses' : 'widgets.analysis.selectNc'"
    :data-widget="widgetId"
  >
    <NAlert v-if="lastError" type="error" :bordered="false" :show-icon="false" class="cmd-note" data-testid="command-error">{{ problemText(lastError) }}</NAlert>
    <NAlert v-else-if="sent" type="success" :bordered="false" :show-icon="false" class="cmd-note" data-testid="command-sent">
      {{ t('widgets.analysis.commandSent') }}
    </NAlert>
    <HypothesisView
      v-if="data"
      :model="data"
      :density="density"
      :from-factor="fromFactor"
      :can-confirm="canConfirm"
      :can-reject="canReject"
      :can-measure="canMeasure"
      :busy="busy"
      @confirm="confirm"
      @reject="reject"
      @request-measurement="measure"
      @select-record="(id) => (focus.eventId = id)"
    />
  </WidgetFrame>
</template>

<style scoped>
.cmd-note {
  margin-bottom: 8px;
  padding: 6px 10px;
}
</style>
