<script setup lang="ts">
/**
 * Раздел «Аналитика» — полный набор показателей кейса §2.4 и §5.2 (FR-86…FR-89):
 * - проверки и «оценка невозможна» — отдельной корзиной (не годно и не брак);
 * - дефекты и изделия с дефектами — двумя колонками, раздельно;
 * - повторные и незавершённые операции;
 * - раздельный учёт: входной брак, оборудование, исполнители, гипотезы (FR-87);
 * - причины установлены / не установлены; время с происхождением;
 * - сравнение сопоставимых работ — не рейтинг людей.
 * Любое число раскрывается до исходных записей (FR-7, AD-45).
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ACCOUNT_TEXT, ACCOUNTS, DIMENSION_TEXT, MetricNumber, toOverviewModel, type MetricPick, type MetricRow } from '@/entities/metric'
import type { AnalyticsOverview } from '@/shared/api/generated/model'
import MetricRowView from './MetricRowView.vue'

const props = defineProps<{ overview: AnalyticsOverview; picked: MetricPick | null }>()
const emit = defineEmits<{ pick: [p: MetricPick] }>()
const { t, d } = useI18n()

const model = computed(() => toOverviewModel(props.overview.items))
const period = computed(() => t('widgets.analytics.periodRange', { from: d(new Date(props.overview.period.from), 'dateTime'), to: d(new Date(props.overview.period.to), 'dateTime') }))

/** Простые разделы: ключ заголовка, пояснение, строки. */
const plainSections = computed(() =>
  [
    { id: 'inspection', title: 'widgets.analytics.sections.inspection', note: 'analytics.unableToAssessBucket', rows: model.value.inspection },
    { id: 'operations', title: 'widgets.analytics.sections.operations', note: null, rows: model.value.operations },
    { id: 'causes', title: 'analytics.metrics.causeEstablished.title', note: null, rows: model.value.causes },
    { id: 'time', title: 'widgets.analytics.sections.time', note: 'analytics.metrics.operationDuration.hint', rows: model.value.time },
  ].filter((s) => s.rows.length),
)

const pick = (p: MetricPick) => emit('pick', p)
const pickCell = (row: MetricRow, sliceKey: string, sliceLabel: string) => pick({ metricId: row.metric_id, title: row.title, sliceKey, sliceLabel })
const cellPicked = (row: MetricRow, key: string) => props.picked?.metricId === row.metric_id && props.picked?.sliceKey === key
</script>

<template>
  <div class="overview">
    <p class="period" data-testid="period-range">
      {{ period }} · <span :title="t('widgets.analytics.basisHint')">{{ t('widgets.analytics.basis', { seq: overview.basis_seq }) }}</span>
    </p>

    <!-- Дефекты и изделия с дефектами — раздельно (кейс §5.2) -->
    <section class="section" data-section="defects">
      <h3>{{ t('analytics.metrics.defectLadder.title') }}</h3>
      <p class="note">{{ t('analytics.metrics.defectCount.hint') }}</p>
      <div class="columns two">
        <div class="column" data-column="defects">
          <h4>{{ t('widgets.analytics.columns.defects') }}</h4>
          <MetricRowView v-for="r in model.defects" :key="r.metric_id" :row="r" :picked="picked" @pick="pick" />
          <p v-if="!model.defects.length" class="absent">{{ t('widgets.analytics.notProvided') }}</p>
        </div>
        <div class="column" data-column="items">
          <h4>{{ t('widgets.analytics.columns.items') }}</h4>
          <MetricRowView v-for="r in model.items" :key="r.metric_id" :row="r" :picked="picked" @pick="pick" />
          <p v-if="!model.items.length" class="absent">{{ t('widgets.analytics.notProvided') }}</p>
        </div>
      </div>
    </section>

    <!-- Раздельный учёт (FR-87, кейс §2.4) -->
    <section class="section" data-section="accounts">
      <h3>{{ t('widgets.analytics.sections.accounts') }}</h3>
      <p class="note">{{ t('analytics.metrics.causeMatrix.hint') }}</p>
      <div class="columns four">
        <div v-for="a in ACCOUNTS" :key="a" class="column" :data-account="a">
          <h4>{{ t(ACCOUNT_TEXT[a]) }}</h4>
          <MetricRowView v-for="r in model.accounts[a]" :key="r.metric_id" :row="r" :picked="picked" @pick="pick" />
          <p v-if="!model.accounts[a].length" class="absent" data-testid="account-absent">{{ t('widgets.analytics.notProvided') }}</p>
        </div>
      </div>
    </section>

    <section v-for="s in plainSections" :key="s.id" class="section" :data-section="s.id">
      <h3>{{ t(s.title) }}</h3>
      <p v-if="s.note" class="note">{{ t(s.note) }}</p>
      <MetricRowView v-for="r in s.rows" :key="r.metric_id" :row="r" :picked="picked" @pick="pick" />
    </section>

    <!-- Сравнение сопоставимых работ (кейс §2.4, §5.2) -->
    <section v-if="model.comparison.length" class="section" data-section="comparison">
      <h3>{{ t('widgets.analytics.sections.comparison') }}</h3>
      <p class="note">{{ t('analytics.metrics.comparableWork.hint') }}</p>
      <p class="note strong">{{ t('analytics.notARanking') }}. {{ t('analytics.participatedIsNotCause') }}</p>
      <table v-for="tbl in model.comparison" :key="tbl.dimension" class="comparison" :data-dimension="tbl.dimension">
        <thead>
          <tr>
            <th scope="col">{{ t(DIMENSION_TEXT[tbl.dimension] ?? 'widgets.analytics.dimensions.unknown') }}</th>
            <th v-for="c in tbl.columns" :key="c.metric_id" scope="col" :data-metric="c.metric_id">{{ c.title }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="r in tbl.rows" :key="r.key" :data-slice="r.key">
            <th scope="row">{{ r.label }}</th>
            <td v-for="(cell, i) in r.cells" :key="tbl.columns[i]!.metric_id">
              <button
                v-if="cell"
                type="button"
                class="num"
                :data-picked="cellPicked(tbl.columns[i]!, r.key) || undefined"
                :title="t('common.actions.drillDown')"
                @click="pickCell(tbl.columns[i]!, r.key, r.label)"
              >
                <MetricNumber :value="cell.value" />
              </button>
              <span v-else class="absent" :title="t('widgets.analytics.notProvided')">—</span>
            </td>
          </tr>
        </tbody>
      </table>
    </section>

    <section v-if="model.other.length" class="section" data-section="other">
      <h3>{{ t('widgets.analytics.sections.other') }}</h3>
      <MetricRowView v-for="r in model.other" :key="r.metric_id" :row="r" :picked="picked" @pick="pick" />
    </section>
  </div>
</template>

<style scoped>
.overview {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.period {
  margin: 0;
  color: #6b7280;
  font-size: 12px;
}

.section h3 {
  margin: 0 0 4px;
  font-size: 15px;
}

.section h4 {
  margin: 0 0 4px;
  color: #374151;
  font-size: 13px;
  font-weight: 600;
}

.note {
  margin: 0 0 8px;
  color: #6b7280;
  font-size: 12px;
}

.note.strong {
  color: #374151;
}

.columns {
  display: grid;
  gap: 12px;
}

.columns.two {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.columns.four {
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
}

.column {
  padding: 8px 10px;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
}

.absent {
  margin: 0;
  color: #9ca3af;
  font-size: 12px;
}

.comparison {
  width: 100%;
  margin-bottom: 8px;
  border-collapse: collapse;
  font-size: 13px;
}

.comparison th,
.comparison td {
  padding: 4px 8px;
  border-bottom: 1px solid #f0f1f3;
  text-align: right;
}

.comparison th:first-child {
  font-weight: 400;
  text-align: left;
}

.comparison thead th {
  color: #4b5563;
  font-weight: 500;
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
</style>
