<script setup lang="ts">
/**
 * Показатель в карточке раздела «Аналитика»: название (от сервера) сверху,
 * крупное число ниже, пояснение мелко под числом, срезы по измерениям —
 * таблицей на всю ширину карточки. Каждое число — кнопка раскрытия до исходных
 * записей (FR-7, AD-45).
 *
 * Плашек у чисел нет: оговорки времени (происхождение, смысл интервала — кейс
 * §5.2, FR-88) — в подсказке значка у названия показателя, одна на показатель,
 * а не у каждой строки среза. Если показатель в карточке один, название и
 * значок несёт заголовок карточки (`show-title=false`) — без второго заголовка.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { DIMENSION_TEXT, metricHintKey, MetricNotes, MetricNumber, originOf, slicesByDimension, valueNotes, type MetricPick, type MetricRow } from '@/entities/metric'
import { DataTable } from '@/shared/ui'

const props = withDefaults(defineProps<{ row: MetricRow; picked: MetricPick | null; showTitle?: boolean }>(), { showTitle: true })
const emit = defineEmits<{ pick: [p: MetricPick] }>()
const { t } = useI18n()

const hint = computed(() => {
  const key = metricHintKey(props.row.metric_id)
  return key ? t(key) : null
})
const notes = computed(() => valueNotes(t, props.row.total))
const warn = computed(() => Boolean(originOf(props.row.total)?.warn))
const groups = computed(() => slicesByDimension(props.row))
const isPicked = (sliceKey?: string) => props.picked?.metricId === props.row.metric_id && props.picked?.sliceKey === sliceKey
</script>

<template>
  <div class="metric-row" :data-metric="row.metric_id" :data-group="row.group">
    <div v-if="showTitle" class="head">
      <span class="title ant-clamp-2" data-testid="metric-title" :title="row.title">{{ row.title }}</span>
      <MetricNotes :notes="notes" :warn="warn" />
    </div>
    <button
      type="button"
      class="num total"
      data-testid="total"
      :data-picked="isPicked() || undefined"
      :title="t('common.actions.drillDown')"
      @click="emit('pick', { metricId: row.metric_id, title: row.title })"
    >
      <MetricNumber :value="row.total" :unknown="row.unknown" :show-origin="false" />
    </button>
    <p v-if="hint" class="hint ant-wrap" data-testid="metric-hint">{{ hint }}</p>
    <DataTable v-for="g in groups" :key="g.dimension" class="slices" :data-dimension="g.dimension">
      <thead>
        <tr>
          <th scope="col">{{ t(DIMENSION_TEXT[g.dimension] ?? 'widgets.analytics.dimensions.unknown') }}</th>
          <th scope="col" class="num-col">{{ t('widgets.analytics.sliceValue') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="s in g.slices" :key="s.key" :data-slice="s.key">
          <!-- Название среза переносится по словам; число — в узкой колонке по своей ширине. -->
          <th scope="row"><span class="label ant-wrap">{{ s.label }}</span></th>
          <td class="num-col">
            <button
              type="button"
              class="num"
              data-testid="slice"
              :data-picked="isPicked(s.key) || undefined"
              :title="t('common.actions.drillDown')"
              @click="emit('pick', { metricId: row.metric_id, title: row.title, sliceKey: s.key, sliceLabel: s.label })"
            >
              <MetricNumber :value="s.value" :show-origin="false" />
            </button>
          </td>
        </tr>
      </tbody>
    </DataTable>
  </div>
</template>

<style scoped>
/* Столбик: название, число, пояснение, срезы — одинаково во всех карточках. */
.metric-row {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  min-width: 0;
}

.head {
  display: flex;
  gap: var(--ant-space-1);
  align-items: flex-start;
  min-width: 0;
}

.title {
  flex: 1 1 auto;
  color: var(--ant-text-2);
  font-size: var(--ant-fs-meta);
  line-height: var(--ant-lh-tight);
}

.num {
  padding: 1px var(--ant-space-1);
  border: 1px solid transparent;
  border-radius: var(--ant-radius-sm);
  background: none;
  color: inherit;
  font: inherit;
  cursor: pointer;
}

.num:hover,
.num:focus-visible {
  border-color: var(--ant-n-400);
}

.num[data-picked] {
  border-color: var(--ant-text);
  background: var(--ant-n-100);
}

/* Итог — крупно, от левого края, как в плитках. */
.total {
  align-self: flex-start;
  max-width: 100%;
  margin-left: calc(-1 * var(--ant-space-1));
  font-size: var(--ant-fs-xl);
  font-weight: var(--ant-fw-bold);
  line-height: var(--ant-lh-tight);
  text-align: left;
}

.hint {
  margin: 0;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
  line-height: var(--ant-lh-tight);
}

.slices {
  margin-top: var(--ant-space-2);
}

/* Шапка среза в узкой карточке переносится, а не распирает таблицу. */
.metric-row .slices thead th {
  white-space: normal;
}

/* Колонка числа — по ширине числа, остальное — названию среза. */
.metric-row .slices .num-col {
  width: 1%;
  text-align: right;
  white-space: nowrap;
}

.label {
  display: block;
  word-break: normal;
}
</style>
