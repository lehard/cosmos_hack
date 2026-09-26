<script setup lang="ts">
/**
 * Виджет «Разбор обстоятельств» (FR-153) — контейнер: проекция
 * `analysis.circumstances` через сгенерированный клиент, рамка с четырьмя
 * состояниями (AD-21), связь с фокусом разбора (выбранная запись подсвечивается
 * и из доводов гипотезы).
 */
import { computed } from 'vue'
import { circumstancesState, useAnalysisFocusStore } from '@/entities/incident'
import type { WidgetProps } from '@/shared/config/widget'
import { WidgetFrame } from '@/shared/ui'
import { useCircumstancesSource } from '../model/source'
import CircumstancesView from './CircumstancesView.vue'

defineProps<WidgetProps>()

const focus = useAnalysisFocusStore()
const src = useCircumstancesSource()
const data = computed(() => src.data.value)
const state = computed(() => (data.value ? circumstancesState(data.value) : 'normal'))
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
    :empty-key="src.ncId.value ? 'empty.noRecords' : 'widgets.analysis.selectNc'"
    :data-widget="widgetId"
  >
    <CircumstancesView v-if="data" v-model:selected="focus.eventId" :model="data" :density="density" />
  </WidgetFrame>
</template>
