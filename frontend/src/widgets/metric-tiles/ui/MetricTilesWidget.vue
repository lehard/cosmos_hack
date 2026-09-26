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
import { useI18n } from 'vue-i18n'
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
const { t, d } = useI18n()

const tiles = computed(() => src.data.value?.items ?? [])
/** Все плитки «оценка невозможна» — так и говорим, а не «норма». */
const state = computed<WidgetDataState>(() => (tiles.value.length && tiles.value.every((x) => x.unknown) ? 'unable_to_assess' : 'normal'))

/** Сервер отдал числа за другой период, чем выбран: подпись и точные границы. */
const otherPeriod = computed(() => {
  const p = src.data.value?.period
  if (!p || p.kind === focus.period) return null
  return {
    text: t('widgets.analytics.tiles.otherPeriod', { period: t(`widgets.analytics.tiles.periodFor.${p.kind}`) }),
    range: t('widgets.analytics.periodRange', { from: d(new Date(p.from), 'dateTime'), to: d(new Date(p.to), 'dateTime') }),
  }
})

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
        <span v-if="otherPeriod" class="period-note ant-ellipsis" data-testid="other-period" :title="`${otherPeriod.text}. ${otherPeriod.range}`">
          {{ otherPeriod.text }}
        </span>
      </ToolBar>
    </div>
    <MetricTilesView :tiles="tiles" :density="density" @open="open" />
  </WidgetFrame>
</template>

<style scoped>
.head {
  margin-bottom: var(--ant-space-2);
}

.period-note {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}
</style>
