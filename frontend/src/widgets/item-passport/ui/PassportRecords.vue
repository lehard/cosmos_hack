<script setup lang="ts">
/**
 * Записи паспорта «кто · что · когда» (FR-42, FR-140): факты, выводы системы,
 * решения людей, документы — у каждой пометка источника, автор, время,
 * подпись и итог её проверки. Исправление — отдельная запись со ссылкой на
 * исправляемую (FR-122); пересмотренный вывод помечен причиной (FR-32).
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NRadioButton, NRadioGroup } from 'naive-ui'
import {
  LAYER_TEXT,
  RECORD_LAYERS,
  SignatureMark,
  SourceMark,
  isCriticalType,
  recordLabel,
  recordLayer,
  sortRecords,
  type PassportRecord,
  type RecordLayer,
} from '@/entities/item'

const props = withDefaults(
  defineProps<{
    records: PassportRecord[]
    /** Показать только последние N (компактный паспорт); 0 — все. */
    limit?: number
    /** Фильтр по слоям. */
    filterable?: boolean
  }>(),
  { limit: 0, filterable: true },
)

const { t, d } = useI18n()
const layer = ref<RecordLayer | 'all'>('all')

const rows = computed(() => {
  const sorted = sortRecords(props.records)
  const filtered = layer.value === 'all' ? sorted : sorted.filter((r) => recordLayer(r.event_type) === layer.value)
  const list = props.limit > 0 ? filtered.slice(-props.limit) : filtered
  return list.map((r) => ({ r, layer: recordLayer(r.event_type), text: recordLabel(r, t), critical: isCriticalType(r.event_type) }))
})
const time = (x: string) => d(new Date(x), 'dateTime')
</script>

<template>
  <section class="records" data-testid="passport-records">
    <h4>{{ t('passport.whoWhatWhen') }}</h4>
    <NRadioGroup v-if="filterable" v-model:value="layer" size="small" class="filter">
      <NRadioButton value="all">{{ t('timeline.layer.all') }}</NRadioButton>
      <NRadioButton v-for="l in RECORD_LAYERS" :key="l" :value="l">{{ t(LAYER_TEXT[l]) }}</NRadioButton>
    </NRadioGroup>
    <p v-if="!rows.length" class="muted">{{ t('empty.noRecords') }}</p>
    <ol class="list">
      <li v-for="{ r, layer: l, text, critical } in rows" :key="r.event_id" class="row" :data-layer="l" :data-event="r.event_type" :data-id="r.event_id">
        <div class="line">
          <time class="time" :title="`${t('timeline.timeKind.recordedAt')}: ${time(r.recorded_at)}`">{{ time(r.occurred_at) }}</time>
          <span class="text">{{ text }}</span>
          <span class="layer">{{ t(LAYER_TEXT[l]) }}</span>
          <SourceMark :record="r" />
        </div>
        <div class="line meta">
          <span v-if="r.author">{{ t('common.words.author') }}: {{ r.author }}</span>
          <SignatureMark :signature="r.signature" />
          <span v-if="r.ca_id" class="ca" :title="t('hints.criticalAction')">{{ r.ca_id }}</span>
          <span v-else-if="critical" class="ca" :title="t('hints.criticalAction')">{{ t('widgets.passport.critical') }}</span>
          <span class="seq">{{ t('widgets.analysis.circumstances.journalRecord', { seq: r.seq }) }}</span>
        </div>
        <p v-if="r.corrects" class="mark" data-testid="corrects">
          {{ t('timeline.marks.corrects', { eventId: r.corrects.event_id }) }} · {{ t('common.words.basis') }}: {{ r.corrects.reason }}
        </p>
        <p v-if="r.corrected_by" class="mark" data-testid="corrected-by">{{ t('timeline.marks.correctedBy', { eventId: r.corrected_by }) }}</p>
        <p v-if="r.revised_due_to" class="mark" data-testid="revised">{{ t('timeline.marks.revised', { eventId: r.revised_due_to }) }}</p>
      </li>
    </ol>
  </section>
</template>

<style scoped>
.records {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

h4 {
  margin: 0;
}

.filter {
  flex-wrap: wrap;
}

.list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.row {
  padding: 4px 0;
  border-bottom: 1px solid #f3f4f6;
}

.line {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 8px;
  align-items: baseline;
}

.time {
  color: #6b7280;
  font-variant-numeric: tabular-nums;
}

.text {
  font-weight: 600;
}

.layer,
.ca,
.seq {
  color: #6b7280;
  font-size: 11px;
}

.ca {
  font-weight: 700;
}

.meta {
  padding-left: 12px;
  font-size: 12px;
}

.mark,
.muted {
  margin: 0;
  padding-left: 12px;
  color: #6b7280;
  font-size: 12px;
}
</style>
