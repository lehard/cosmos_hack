<script setup lang="ts">
/**
 * Виджет «Раскрытие показателя до исходных записей» — контейнер: число,
 * выбранное в плитках или разделе «Аналитика» (фокус аналитики), →
 * `analytics.metric.drilldown` (FR-7, AD-45). Изделие — переход в паспорт.
 */
import { computed } from 'vue'
import { useDrillDown } from '@/features/drill-down'
import { useMetricDrilldown, useMetricFocusStore } from '@/entities/metric'
import type { WidgetProps } from '@/shared/config/widget'
import { WidgetFrame } from '@/shared/ui'
import MetricDrilldownView from './MetricDrilldownView.vue'

defineProps<WidgetProps>()
const focus = useMetricFocusStore()
const drill = useDrillDown()
const src = useMetricDrilldown(
  () => focus.pick?.metricId ?? null,
  () => focus.pick?.sliceKey,
)
const data = computed(() => (focus.pick ? src.data.value : null))
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="src.mode.value"
    :loading="!!focus.pick && src.isPending.value"
    :error="focus.pick ? src.error.value : undefined"
    :empty="!focus.pick"
    empty-key="widgets.analytics.drilldown.pickHint"
    :data-widget="widgetId"
  >
    <MetricDrilldownView
      v-if="focus.pick && data"
      :pick="focus.pick"
      :drilldown="data"
      :has-more="src.hasMore.value"
      :loading-more="src.loadingMore.value"
      @open-item="(id) => drill.open({ entity: 'item', id })"
      @open-ref="(ref) => drill.open(ref)"
      @more="src.loadMore"
    />
  </WidgetFrame>
</template>
