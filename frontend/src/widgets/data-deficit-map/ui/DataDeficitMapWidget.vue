<script setup lang="ts">
/**
 * Виджет «Карта дефицита данных» — каркас страницы (FR-143; наполнение — эпик
 * 42). Узлы без данных источника — из `analytics.node_counters.read`.
 */
import { computed } from 'vue'
import { useNodeCounterSet } from '@/entities/metric'
import type { WidgetDataState, WidgetProps } from '@/shared/config/widget'
import { WidgetFrame } from '@/shared/ui'
import DataDeficitMapView from './DataDeficitMapView.vue'

defineProps<WidgetProps>()
const nodes = useNodeCounterSet()
const gaps = computed(() => nodes.data.value?.data_gaps ?? [])
/** Есть узлы без данных — «оценка невозможна», а не «норма». */
const state = computed<WidgetDataState>(() => (gaps.value.length ? 'unable_to_assess' : 'normal'))
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="nodes.mode.value"
    :state="state"
    :loading="nodes.isPending.value"
    :error="nodes.error.value"
    :data-widget="widgetId"
  >
    <DataDeficitMapView :data-gaps="gaps" />
  </WidgetFrame>
</template>
