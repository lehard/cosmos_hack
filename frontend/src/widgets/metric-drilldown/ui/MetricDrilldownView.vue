<script setup lang="ts">
/**
 * Раскрытие числа (FR-7, AD-45, соглашение «Показатели»): итог → строки вклада
 * изделий → исходные записи журнала. Для штучных показателей видна сверка
 * «сумма вкладов = итог»: число проверяемо по сохранённой истории (кейс §7.2).
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { codeText, LAYER_TEXT, SOURCE_KIND_TEXT } from '@/entities/item'
import { MetricNumber, sumCheck, type MetricPick } from '@/entities/metric'
import type { ContributionRow, DrillRef, MetricDrilldown } from '@/shared/api/generated/model'
import SourceRecords from './SourceRecords.vue'
import { ActionButton } from '@/shared/ui'

const props = defineProps<{ pick: MetricPick; drilldown: MetricDrilldown; hasMore: boolean; loadingMore: boolean }>()
const emit = defineEmits<{ openItem: [itemId: string]; openRef: [ref: DrillRef]; more: [] }>()
const { t, d } = useI18n()

/** Раскрытые строки: item_id + срез. */
const open = ref<Set<string>>(new Set())
const rowKey = (r: ContributionRow) => `${r.item_id}|${r.slice_key}`
function toggle(r: ContributionRow): void {
  const next = new Set(open.value)
  if (next.has(rowKey(r))) next.delete(rowKey(r))
  else next.add(rowKey(r))
  open.value = next
}

const check = computed(() => sumCheck(props.drilldown.total, props.drilldown.items, !props.hasMore))
const period = computed(() =>
  t('widgets.analytics.periodRange', { from: d(new Date(props.drilldown.period.from), 'dateTime'), to: d(new Date(props.drilldown.period.to), 'dateTime') }),
)

/** Происхождение вклада: вид источника (FR-140), «вывод системы» или вид записи (AD-2); чужой код — UNKNOWN(код). */
const KIND_TEXT: Record<string, string> = { ...LAYER_TEXT, ...SOURCE_KIND_TEXT, system: 'timeline.sourceKind.system' }

/** Строка вне изделия (оборудование, несоответствие, инцидент): открывается её объект. */
const outside = (r: ContributionRow) => !!r.ref && r.ref.entity !== 'item'
function openRow(r: ContributionRow): void {
  if (r.ref && outside(r)) emit('openRef', r.ref)
  else emit('openItem', r.item_id)
}
const kindText = (code: string) => codeText(KIND_TEXT, code, t)
</script>

<template>
  <div class="drilldown">
    <header class="head">
      <div class="what">
        <span class="title" data-testid="drill-title">{{ pick.title }}</span>
        <span v-if="pick.sliceLabel" class="slice" data-testid="drill-slice">{{ pick.sliceLabel }}</span>
      </div>
      <MetricNumber class="total" :value="drilldown.total" />
    </header>
    <p class="period">{{ period }}</p>
    <p v-if="check" class="check" :data-check="check" data-testid="sum-check">
      {{ t(check === 'match' ? 'widgets.analytics.drilldown.sumMatch' : 'widgets.analytics.drilldown.sumMismatch', { n: drilldown.items.length }) }}
    </p>

    <p v-if="!drilldown.items.length" class="empty">{{ t('widgets.analytics.drilldown.noContributions') }}</p>
    <ul v-else class="rows">
      <li v-for="r in drilldown.items" :key="rowKey(r)" class="row" :data-item="r.item_id">
        <div class="line">
          <button type="button" class="item" :title="t('common.actions.openPassport')" @click="openRow(r)">{{ r.label }}</button>
          <span v-if="r.slice_key" class="slice-key">{{ r.slice_key }}</span>
          <MetricNumber class="value" :value="r.value" />
        </div>
        <div class="line sub">
          <span v-for="k in r.source_kinds" :key="k" class="kind" data-testid="source-kind">{{ kindText(k) }}</span>
          <button
            type="button"
            class="toggle"
            data-testid="toggle-records"
            :aria-expanded="open.has(rowKey(r))"
            :disabled="!r.source_event_ids.length"
            @click="toggle(r)"
          >
            {{ t('widgets.analytics.drilldown.records', { n: r.source_event_ids.length }) }}
          </button>
        </div>
        <template v-if="open.has(rowKey(r))">
          <ol v-if="outside(r)" class="ids" data-testid="record-ids">
            <li v-for="id in r.source_event_ids" :key="id"><code>{{ id }}</code></li>
          </ol>
          <SourceRecords v-else :item-id="r.item_id" :event-ids="r.source_event_ids" />
        </template>
      </li>
    </ul>
    <ActionButton v-if="hasMore" size="small" :loading="loadingMore" data-testid="more" @click="emit('more')" :label="t('common.actions.showAll')" />
  </div>
</template>

<style scoped>
.ids {
  margin: 0;
  padding-left: 18px;
  font-size: var(--ant-fs-meta);
}

.drilldown {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.head {
  display: flex;
  gap: 8px;
  align-items: baseline;
  justify-content: space-between;
}

.what {
  display: flex;
  flex-direction: column;
}

.title {
  font-weight: var(--ant-fw-bold);
}

.slice,
.period {
  margin: 0;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.total {
  font-size: var(--ant-fs-xl);
  font-weight: var(--ant-fw-bold);
}

.check {
  margin: 0;
  font-size: var(--ant-fs-meta);
}

.check[data-check='mismatch'] {
  color: var(--ant-status-danger);
  font-weight: var(--ant-fw-bold);
}

.empty {
  margin: 0;
  color: var(--ant-text-3);
}

.rows {
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 0;
  list-style: none;
}

.row {
  padding: 6px 0;
  border-bottom: 1px solid var(--ant-n-100);
}

.line {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: baseline;
}

.line.sub {
  margin-top: 2px;
  font-size: var(--ant-fs-meta);
}

.item {
  padding: 0;
  border: 0;
  background: none;
  color: var(--ant-text);
  font: inherit;
  text-decoration: underline;
  cursor: pointer;
}

.slice-key {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.value {
  margin-left: auto;
}

.kind {
  padding: 0 6px;
  border: 1px solid var(--ant-border-strong);
  border-radius: var(--ant-radius-lg);
  color: var(--ant-text-2);
  font-size: var(--ant-fs-xs);
}

.toggle {
  padding: 0;
  border: 0;
  background: none;
  color: var(--ant-accent);
  font: inherit;
  cursor: pointer;
}

.toggle:disabled {
  color: var(--ant-n-400);
  cursor: default;
}
</style>
