<script setup lang="ts">
/**
 * Раскрытие числа (FR-7, AD-45, соглашение «Показатели»): итог → строки вклада
 * изделий → исходные записи журнала. Для штучных показателей видна сверка
 * «сумма вкладов = итог»: число проверяемо по сохранённой истории (кейс §7.2).
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton } from 'naive-ui'
import { codeText, LAYER_TEXT, SOURCE_KIND_TEXT } from '@/entities/item'
import { MetricNumber, sumCheck, type MetricPick } from '@/entities/metric'
import type { ContributionRow, MetricDrilldown } from '@/shared/api/generated/model'
import SourceRecords from './SourceRecords.vue'

const props = defineProps<{ pick: MetricPick; drilldown: MetricDrilldown; hasMore: boolean; loadingMore: boolean }>()
const emit = defineEmits<{ openItem: [itemId: string]; more: [] }>()
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

/** Происхождение вклада: вид источника (FR-140) или вид записи (AD-2); чужой код — UNKNOWN(код). */
const KIND_TEXT: Record<string, string> = { ...LAYER_TEXT, ...SOURCE_KIND_TEXT }
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
          <button type="button" class="item" :title="t('common.actions.openPassport')" @click="emit('openItem', r.item_id)">{{ r.label }}</button>
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
        <SourceRecords v-if="open.has(rowKey(r))" :item-id="r.item_id" :event-ids="r.source_event_ids" />
      </li>
    </ul>
    <NButton v-if="hasMore" size="small" :loading="loadingMore" data-testid="more" @click="emit('more')">{{ t('common.actions.showAll') }}</NButton>
  </div>
</template>

<style scoped>
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
  font-weight: 600;
}

.slice,
.period {
  margin: 0;
  color: #6b7280;
  font-size: 12px;
}

.total {
  font-size: 18px;
  font-weight: 600;
}

.check {
  margin: 0;
  font-size: 12px;
}

.check[data-check='mismatch'] {
  color: #d64545;
  font-weight: 600;
}

.empty {
  margin: 0;
  color: #6b7280;
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
  border-bottom: 1px solid #f0f1f3;
}

.line {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: baseline;
}

.line.sub {
  margin-top: 2px;
  font-size: 12px;
}

.item {
  padding: 0;
  border: 0;
  background: none;
  color: #1f2937;
  font: inherit;
  text-decoration: underline;
  cursor: pointer;
}

.slice-key {
  color: #6b7280;
  font-size: 12px;
}

.value {
  margin-left: auto;
}

.kind {
  padding: 0 6px;
  border: 1px solid #d1d5db;
  border-radius: 8px;
  color: #4b5563;
  font-size: 11px;
}

.toggle {
  padding: 0;
  border: 0;
  background: none;
  color: #2f6fdb;
  font: inherit;
  cursor: pointer;
}

.toggle:disabled {
  color: #9ca3af;
  cursor: default;
}
</style>
