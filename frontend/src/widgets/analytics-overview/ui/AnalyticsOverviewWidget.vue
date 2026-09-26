<script setup lang="ts">
/**
 * Виджет «Аналитика» — контейнер: `analytics.overview.read` за период из фокуса
 * аналитики; выбранное число уходит в фокус — его раскрывает виджет
 * «Раскрытие показателя» (FR-7).
 */
import { computed } from 'vue'
import { PeriodPicker, useAnalyticsOverview, useMetricFocusStore } from '@/entities/metric'
import { naiveSizeOf, type WidgetDataState, type WidgetProps } from '@/shared/config/widget'
import { WidgetFrame } from '@/shared/ui'
import AnalyticsOverviewView from './AnalyticsOverviewView.vue'

defineProps<WidgetProps>()
const src = useAnalyticsOverview()
const focus = useMetricFocusStore()

const data = computed(() => src.data.value)
/** Все показатели «оценка невозможна» — так и говорим, а не «норма». */
const state = computed<WidgetDataState>(() => (data.value?.items.length && data.value.items.every((r) => r.unknown) ? 'unable_to_assess' : 'normal'))
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="src.mode.value"
    :state="state"
    :loading="src.isPending.value"
    :error="src.error.value"
    :empty="!data?.items.length"
    :data-widget="widgetId"
  >
    <div class="head">
      <PeriodPicker :size="naiveSizeOf(density)" />
    </div>
    <AnalyticsOverviewView v-if="data" :overview="data" :picked="focus.pick" @pick="focus.select" />
  </WidgetFrame>
</template>

<style scoped>
.head {
  margin-bottom: 12px;
}
</style>
