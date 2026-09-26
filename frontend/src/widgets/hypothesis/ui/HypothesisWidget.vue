<script setup lang="ts">
/**
 * Виджет «Гипотезы причины» (FR-59, FR-60, FR-135) — контейнер. Довод «за» или
 * «против» по клику подсвечивается на дорожках разбора обстоятельств.
 *
 * Команды «подтвердить причину / отклонить / запросить измерение» — решения
 * человека с подписью (`incident.cause.concluded`, группа критических действий
 * `cause`); пока их операций нет в API, кнопки выключены (canAct=false).
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { FACTOR_TEXT, hypothesesState, useAnalysisFocusStore } from '@/entities/incident'
import type { WidgetProps } from '@/shared/config/widget'
import { WidgetFrame } from '@/shared/ui'
import { useHypothesesSource } from '../model/source'
import HypothesisView from './HypothesisView.vue'

defineProps<WidgetProps>()

const { t } = useI18n()
const focus = useAnalysisFocusStore()
const src = useHypothesesSource()
const data = computed(() => src.data.value)
const state = computed(() => (data.value ? hypothesesState(data.value) : 'normal'))

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
    empty-key="empty.noHypotheses"
    :data-widget="widgetId"
  >
    <HypothesisView
      v-if="data"
      :model="data"
      :density="density"
      :from-factor="fromFactor"
      :can-act="false"
      @select-record="(id) => (focus.eventId = id)"
    />
  </WidgetFrame>
</template>
