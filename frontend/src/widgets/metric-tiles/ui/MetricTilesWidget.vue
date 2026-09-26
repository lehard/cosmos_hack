<script setup lang="ts">
/**
 * Виджет «Показатели» — контейнер: плитки `analytics.tile.list` за период из
 * фокуса аналитики. Нажатие на плитку выбирает число для раскрытия и ведёт на
 * вкладку «Аналитика» стола (FR-7: по показателю → исходные записи).
 *
 * Период уходит в запрос (`period`); если сервер вернул числа за другой
 * период, чем выбран (`MetricTileList.period.kind`), об этом сказано словами
 * рядом с переключателем — иначе кажется, что переключатель не работает.
 */
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import type { MetricTile } from '@/shared/api/generated/model'
import { PeriodPicker, useMetricFocusStore, useMetricTiles } from '@/entities/metric'
import { naiveSizeOf, type WidgetDataState, type WidgetProps } from '@/shared/config/widget'
import { ToolBar, WidgetFrame } from '@/shared/ui'
import MetricTilesView from './MetricTilesView.vue'

/** Вкладка стола, где стоят раздел «Аналитика» и раскрытие. */
const ANALYTICS_TAB = 'analytics'

const props = defineProps<WidgetProps>()
const src = useMetricTiles()
const focus = useMetricFocusStore()
const router = useRouter()

const tiles = computed(() => src.data.value?.items ?? [])
/** Все плитки «оценка невозможна» — так и говорим, а не «норма». */
const state = computed<WidgetDataState>(() => (tiles.value.length && tiles.value.every((x) => x.unknown) ? 'unable_to_assess' : 'normal'))

function open(tile: MetricTile): void {
  focus.select({ metricId: tile.metric_id, title: tile.title })
  const tab = typeof props.slice.drill_tab === 'string' ? props.slice.drill_tab : ANALYTICS_TAB
  if (router?.hasRoute('desk')) void router.push({ name: 'desk', params: { tab } })
}
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="src.mode.value"
    :state="state"
    :loading="src.isPending.value"
    :error="src.error.value"
    :empty="!tiles.length"
    :data-widget="widgetId"
  >
    <div class="head">
      <ToolBar>
        <PeriodPicker :size="naiveSizeOf(density)" />
      </ToolBar>
    </div>
    <MetricTilesView :tiles="tiles" :density="density" @open="open" />
  </WidgetFrame>
</template>

<style scoped>
.head {
  margin-bottom: var(--ant-space-2);
}

</style>
