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
 *
 * Вёрстка: разделы — ровная сетка карточек (MetricCard): название сверху,
 * крупное число ниже, срезы — таблицей на всю ширину карточки. Плашек у чисел
 * нет — оговорки времени в подсказке значка у заголовка.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ACCOUNT_TEXT, ACCOUNTS, DIMENSION_TEXT, MetricNotes, MetricNumber, toOverviewModel, valueNotes, type MetricPick, type MetricRow } from '@/entities/metric'
import type { AnalyticsOverview } from '@/shared/api/generated/model'
import { DataTable } from '@/shared/ui'
import MetricCard from './MetricCard.vue'

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
/** Оговорки столбца сравнения (время: происхождение, интервал) — в подсказке шапки, не у каждой ячейки. */
const columnNotes = (row: MetricRow) => valueNotes(t, row.total)
const pickCell = (row: MetricRow, sliceKey: string, sliceLabel: string) => pick({ metricId: row.metric_id, title: row.title, sliceKey, sliceLabel })
const cellPicked = (row: MetricRow, key: string) => props.picked?.metricId === row.metric_id && props.picked?.sliceKey === key
</script>

<template>
  <div class="overview">
    <p class="period ant-wrap" data-testid="period-range">
      {{ period }} · <span :title="t('widgets.analytics.basisHint')">{{ t('widgets.analytics.basis', { seq: overview.basis_seq }) }}</span>
    </p>

    <!-- Дефекты и изделия с дефектами — раздельно (кейс §5.2) -->
    <section class="section" data-section="defects">
      <h3 class="ant-wrap">{{ t('analytics.metrics.defectLadder.title') }}</h3>
      <p class="note ant-wrap">{{ t('analytics.metrics.defectCount.hint') }}</p>
      <div class="cards">
        <MetricCard :title="t('widgets.analytics.columns.defects')" :rows="model.defects" :picked="picked" data-column="defects" @pick="pick" />
        <MetricCard :title="t('widgets.analytics.columns.items')" :rows="model.items" :picked="picked" data-column="items" @pick="pick" />
      </div>
    </section>

    <!-- Раздельный учёт (FR-87, кейс §2.4) -->
    <section class="section" data-section="accounts">
      <h3 class="ant-wrap">{{ t('widgets.analytics.sections.accounts') }}</h3>
      <p class="note ant-wrap">{{ t('analytics.metrics.causeMatrix.hint') }}</p>
      <div class="cards">
        <MetricCard v-for="a in ACCOUNTS" :key="a" :title="t(ACCOUNT_TEXT[a])" :rows="model.accounts[a]" :picked="picked" :data-account="a" @pick="pick" />
      </div>
    </section>

    <!-- Простые разделы: карточка на показатель, заголовок карточки — название показателя -->
    <section v-for="s in plainSections" :key="s.id" class="section" :data-section="s.id">
      <h3 class="ant-wrap">{{ t(s.title) }}</h3>
      <p v-if="s.note" class="note ant-wrap">{{ t(s.note) }}</p>
      <div class="cards">
        <MetricCard v-for="r in s.rows" :key="r.metric_id" :rows="[r]" :picked="picked" @pick="pick" />
      </div>
    </section>

    <!-- Сравнение сопоставимых работ (кейс §2.4, §5.2) -->
    <section v-if="model.comparison.length" class="section" data-section="comparison">
      <h3 class="ant-wrap">{{ t('widgets.analytics.sections.comparison') }}</h3>
      <p class="note ant-wrap">{{ t('analytics.metrics.comparableWork.hint') }}</p>
      <p class="note strong ant-wrap">{{ t('analytics.notARanking') }}. {{ t('analytics.participatedIsNotCause') }}</p>
      <DataTable v-for="tbl in model.comparison" :key="tbl.dimension" class="comparison" :data-dimension="tbl.dimension">
        <thead>
          <tr>
            <th scope="col"><span class="ant-clamp-2">{{ t(DIMENSION_TEXT[tbl.dimension] ?? 'widgets.analytics.dimensions.unknown') }}</span></th>
            <th v-for="c in tbl.columns" :key="c.metric_id" scope="col" :data-metric="c.metric_id">
              <span class="col-head">
                <span class="ant-clamp-2" :title="c.title">{{ c.title }}</span>
                <MetricNotes :notes="columnNotes(c)" />
              </span>
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="r in tbl.rows" :key="r.key" :data-slice="r.key">
            <th scope="row"><span class="ant-wrap">{{ r.label }}</span></th>
            <td v-for="(cell, i) in r.cells" :key="tbl.columns[i]!.metric_id">
              <button
                v-if="cell"
                type="button"
                class="num"
                :data-picked="cellPicked(tbl.columns[i]!, r.key) || undefined"
                :title="t('common.actions.drillDown')"
                @click="pickCell(tbl.columns[i]!, r.key, r.label)"
              >
                <MetricNumber :value="cell.value" :show-origin="false" />
              </button>
              <span v-else class="absent" :title="t('widgets.analytics.notProvided')">—</span>
            </td>
          </tr>
        </tbody>
      </DataTable>
    </section>

    <section v-if="model.other.length" class="section" data-section="other">
      <h3 class="ant-wrap">{{ t('widgets.analytics.sections.other') }}</h3>
      <div class="cards">
        <MetricCard v-for="r in model.other" :key="r.metric_id" :rows="[r]" :picked="picked" @pick="pick" />
      </div>
    </section>
  </div>
</template>

<style scoped>
.overview {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-5);
  min-width: 0;
}

.period {
  margin: 0;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.section {
  min-width: 0;
}

.section h3 {
  margin: 0 0 var(--ant-space-1);
  font-size: var(--ant-fs-title);
}

.note {
  margin: 0 0 var(--ant-space-2);
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.note.strong {
  color: var(--ant-n-700);
}

/* Ровная сетка карточек: колонки одной ширины, в узкой области — одна колонка. */
.cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(100%, calc(var(--ant-space-10) * 6)), 1fr));
  gap: var(--ant-space-3);
  margin-top: var(--ant-space-2);
}

.absent {
  margin: 0;
  color: var(--ant-n-400);
  font-size: var(--ant-fs-meta);
}

.comparison {
  margin-bottom: var(--ant-space-2);
}

/* Шапка сравнения — названия показателей переносятся, а не распирают таблицу. */
.overview .comparison thead th {
  white-space: normal;
  vertical-align: bottom;
}

.col-head {
  display: inline-flex;
  gap: var(--ant-space-1);
  align-items: flex-start;
  justify-content: flex-end;
  max-width: 100%;
}

.comparison td,
.comparison thead th:not(:first-child) {
  text-align: right;
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
</style>
