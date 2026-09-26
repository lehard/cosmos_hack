<script setup lang="ts">
/**
 * Строка показателя раздела «Аналитика»: название (от сервера), итог и срезы по
 * измерениям. Каждое число — кнопка раскрытия до исходных записей (FR-7,
 * AD-45); у времени видно происхождение (кейс §5.2).
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { DIMENSION_TEXT, metricHintKey, MetricNumber, slicesByDimension, type MetricPick, type MetricRow } from '@/entities/metric'

const props = defineProps<{ row: MetricRow; picked: MetricPick | null }>()
const emit = defineEmits<{ pick: [p: MetricPick] }>()
const { t } = useI18n()

const hint = computed(() => {
  const key = metricHintKey(props.row.metric_id)
  return key ? t(key) : null
})
const groups = computed(() => slicesByDimension(props.row))
const isPicked = (sliceKey?: string) => props.picked?.metricId === props.row.metric_id && props.picked?.sliceKey === sliceKey
</script>

<template>
  <div class="metric-row" :data-metric="row.metric_id" :data-group="row.group">
    <div class="head">
      <span class="title">{{ row.title }}</span>
      <button
        type="button"
        class="num total"
        data-testid="total"
        :data-picked="isPicked() || undefined"
        :title="t('common.actions.drillDown')"
        @click="emit('pick', { metricId: row.metric_id, title: row.title })"
      >
        <MetricNumber :value="row.total" :unknown="row.unknown" />
      </button>
    </div>
    <p v-if="hint" class="hint">{{ hint }}</p>
    <table v-for="g in groups" :key="g.dimension" class="slices" :data-dimension="g.dimension">
      <caption>{{ t(DIMENSION_TEXT[g.dimension] ?? 'widgets.analytics.dimensions.unknown') }}</caption>
      <tbody>
        <tr v-for="s in g.slices" :key="s.key" :data-slice="s.key">
          <th scope="row">{{ s.label }}</th>
          <td>
            <button
              type="button"
              class="num"
              data-testid="slice"
              :data-picked="isPicked(s.key) || undefined"
              :title="t('common.actions.drillDown')"
              @click="emit('pick', { metricId: row.metric_id, title: row.title, sliceKey: s.key, sliceLabel: s.label })"
            >
              <MetricNumber :value="s.value" />
            </button>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<style scoped>
.metric-row {
  padding: 8px 0;
  border-bottom: 1px solid #f0f1f3;
}

.metric-row:last-child {
  border-bottom: 0;
}

.head {
  display: flex;
  gap: 12px;
  align-items: baseline;
  justify-content: space-between;
}

.title {
  font-weight: 500;
}

.hint {
  margin: 2px 0 0;
  color: #6b7280;
  font-size: 12px;
}

.num {
  padding: 1px 6px;
  border: 1px solid transparent;
  border-radius: 4px;
  background: none;
  color: inherit;
  font: inherit;
  cursor: pointer;
}

.num:hover,
.num:focus-visible {
  border-color: #9ca3af;
}

.num[data-picked] {
  border-color: #1f2937;
  background: #f3f4f6;
}

.total {
  font-size: 16px;
  font-weight: 600;
  white-space: nowrap;
}

.slices {
  margin: 6px 0 0 12px;
  border-collapse: collapse;
  font-size: 13px;
}

.slices caption {
  color: #6b7280;
  font-size: 11px;
  text-align: left;
}

.slices th {
  padding: 1px 12px 1px 0;
  font-weight: 400;
  text-align: left;
}

.slices td {
  text-align: right;
}
</style>
