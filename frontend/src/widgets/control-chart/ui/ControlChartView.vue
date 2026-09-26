<script setup lang="ts">
/**
 * Контрольная карта узла (FR-5, FR-89; ГОСТ Р ИСО 7870-2): значения по порядку
 * выполнений, центральная линия, верхняя и нижняя контрольные границы — как
 * их прислал сервер. Выход за границы (`out_of_control` — решает сервер)
 * отмечен формой, цветом тревоги и строкой в списке: не только цветом.
 * Точка ведёт к своему объекту (изделию) — раскрытие до записей (FR-7).
 * Рисунок — SVG без библиотек графиков; рядом таблица точек.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { statusPalette } from '@/shared/api/generated/statuses'
import { chartGeometry, formatValue, type ControlChart, type ControlChartPoint, type MetricValue } from '@/entities/metric'
import type { DrillRef } from '@/shared/model/drill'
import { DataTable } from '@/shared/ui'

const props = defineProps<{ chart: ControlChart }>()
const emit = defineEmits<{ open: [ref: DrillRef] }>()
const { t, n, d } = useI18n()

const g = computed(() => chartGeometry(props.chart))
const fmt = (v: MetricValue) => formatValue({ t, n: (x, f) => n(x, f) }, v)
const outside = computed(() => props.chart.points.filter((p) => p.out_of_control))
const DANGER = statusPalette.danger

/** Показатели карт с названием в словаре; прочие — код как есть. */
const CHART_METRIC_TEXT: Record<string, string> = {
  current_a: 'widgets.analytics.chart.metrics.currentA',
  defect_rate: 'widgets.analytics.chart.metrics.defectRate',
}
const metricName = computed(() => {
  if (props.chart.title) return props.chart.title
  const key = CHART_METRIC_TEXT[props.chart.metric_id]
  return key ? t(key) : props.chart.metric_id
})

const pointTitle = (p: ControlChartPoint) =>
  `${d(new Date(p.at), 'dateTime')}: ${fmt(p.value)}${p.out_of_control ? ` — ${t('widgets.analytics.chart.outOfControl')}` : ''}`

function openPoint(p: ControlChartPoint): void {
  if (p.ref) emit('open', p.ref)
}
</script>

<template>
  <div class="chart">
    <p class="caption">
      <code>{{ chart.step_key }}</code> · {{ metricName }}
    </p>
    <dl class="limits">
      <div data-limit="upper"><dt>{{ t('widgets.analytics.chart.upper') }}</dt><dd>{{ fmt(chart.upper) }}</dd></div>
      <div data-limit="center"><dt>{{ t('widgets.analytics.chart.center') }}</dt><dd>{{ fmt(chart.center) }}</dd></div>
      <div data-limit="lower"><dt>{{ t('widgets.analytics.chart.lower') }}</dt><dd>{{ fmt(chart.lower) }}</dd></div>
    </dl>

    <p v-if="!chart.points.length" class="empty">{{ t('widgets.analytics.chart.noPoints') }}</p>
    <svg
      v-else
      class="plot"
      :viewBox="`0 0 ${g.box.width} ${g.box.height}`"
      role="img"
      :aria-label="t('widgets.analytics.chart.aria', { n: chart.points.length, out: outside.length })"
      preserveAspectRatio="none"
    >
      <line class="limit" data-line="upper" :x1="g.box.left" :x2="g.box.width - g.box.right" :y1="g.upper" :y2="g.upper" />
      <line class="center" data-line="center" :x1="g.box.left" :x2="g.box.width - g.box.right" :y1="g.center" :y2="g.center" />
      <line class="limit" data-line="lower" :x1="g.box.left" :x2="g.box.width - g.box.right" :y1="g.lower" :y2="g.lower" />
      <text class="axis" :x="g.box.left - 6" :y="g.upper + 4" text-anchor="end">{{ fmt(chart.upper) }}</text>
      <text class="axis" :x="g.box.left - 6" :y="g.center + 4" text-anchor="end">{{ fmt(chart.center) }}</text>
      <text class="axis" :x="g.box.left - 6" :y="g.lower + 4" text-anchor="end">{{ fmt(chart.lower) }}</text>
      <polyline class="series" :points="g.path" />
      <g
        v-for="p in g.points"
        :key="p.index"
        class="point"
        data-testid="point"
        :data-out="p.point.out_of_control || undefined"
        :data-ref="p.point.ref?.id"
        :tabindex="p.point.ref ? 0 : undefined"
        @click="openPoint(p.point)"
        @keydown.enter="openPoint(p.point)"
      >
        <title>{{ pointTitle(p.point) }}</title>
        <circle class="hit" :cx="p.x" :cy="p.y" r="10" />
        <rect v-if="p.point.out_of_control" :x="p.x - 5" :y="p.y - 5" width="10" height="10" :fill="DANGER" style="stroke: var(--ant-surface)" stroke-width="2" :transform="`rotate(45 ${p.x} ${p.y})`" />
        <circle v-else :cx="p.x" :cy="p.y" r="4" class="dot" />
      </g>
    </svg>

    <div v-if="outside.length" class="outside" data-testid="outside">
      <p class="outside-title">{{ t('widgets.analytics.chart.outsideList', { n: outside.length }) }}</p>
      <ul>
        <li v-for="(p, i) in outside" :key="i">
          <span class="mark" aria-hidden="true">◆</span>
          {{ d(new Date(p.at), 'dateTime') }} — {{ fmt(p.value) }}
          <button v-if="p.ref" type="button" class="link" @click="emit('open', p.ref)">{{ t('common.actions.open') }}</button>
        </li>
      </ul>
    </div>
    <p v-else-if="chart.points.length" class="inside" data-testid="inside">{{ t('widgets.analytics.chart.allInside') }}</p>

    <details class="table-view">
      <summary>{{ t('widgets.analytics.chart.table') }}</summary>
      <DataTable>
        <thead>
          <tr>
            <th scope="col">{{ t('common.words.time') }}</th>
            <th scope="col">{{ t('widgets.analytics.chart.value') }}</th>
            <th scope="col">{{ t('common.words.status') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(p, i) in chart.points" :key="i" :data-out="p.out_of_control || undefined">
            <td>{{ d(new Date(p.at), 'dateTime') }}</td>
            <td>{{ fmt(p.value) }}</td>
            <td>{{ p.out_of_control ? t('widgets.analytics.chart.outOfControl') : t('widgets.analytics.chart.inControl') }}</td>
          </tr>
        </tbody>
      </DataTable>
    </details>
  </div>
</template>

<style scoped>
.chart {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.caption,
.empty,
.inside {
  margin: 0;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.limits {
  display: flex;
  gap: 16px;
  margin: 0;
  font-size: var(--ant-fs-meta);
}

.limits div {
  display: flex;
  gap: 4px;
}

.limits dt {
  color: var(--ant-text-3);
}

.limits dd {
  margin: 0;
  font-variant-numeric: tabular-nums;
}

.plot {
  width: 100%;
  height: 220px;
}

.limit {
  stroke: var(--ant-n-400);
  stroke-width: 1;
  stroke-dasharray: 4 3;
}

.center {
  stroke: var(--ant-text-3);
  stroke-width: 1;
}

.axis {
  fill: var(--ant-text-3);
  font-size: 10px;
}

.series {
  fill: none;
  stroke: var(--ant-text-2);
  stroke-width: 2;
  stroke-linejoin: round;
  vector-effect: non-scaling-stroke;
}

.dot {
  fill: var(--ant-text-2);
  stroke: var(--ant-surface);
  stroke-width: 2;
}

.hit {
  fill: transparent;
}

.point[tabindex] {
  cursor: pointer;
}

.outside {
  font-size: var(--ant-fs-body);
}

.outside-title {
  margin: 0 0 4px;
  font-weight: var(--ant-fw-bold);
}

.outside ul {
  margin: 0;
  padding: 0;
  list-style: none;
}

.mark {
  color: var(--ant-status-danger);
}

.link {
  padding: 0;
  border: 0;
  background: none;
  color: var(--ant-accent);
  font: inherit;
  cursor: pointer;
}

.table-view {
  font-size: var(--ant-fs-meta);
}

.table-view tr[data-out] td {
  font-weight: var(--ant-fw-bold);
}
</style>
