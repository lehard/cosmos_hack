<script setup lang="ts">
/**
 * Строка записи журнала в карточке: время, краткое содержание от сервера,
 * вид записи, пометка источника (FR-140), автор. Клик — выбрать запись.
 */
import { useI18n } from 'vue-i18n'
import { LAYER_TEXT, SourceMark, entryText } from '@/entities/item'
import type { NCRecordRef } from '@/entities/nonconformity'

defineProps<{ record: NCRecordRef; mark?: string | null }>()
const { t, d } = useI18n()
const time = (x: string) => d(new Date(x), 'dateTime')
</script>

<template>
  <div class="record" :data-kind="record.kind" :data-event="record.event_type" :data-id="record.event_id">
    <time class="time">{{ time(record.occurred_at) }}</time>
    <span class="text ant-wrap">{{ entryText(record) }}</span>
    <span class="kind">{{ t(LAYER_TEXT[record.kind]) }}</span>
    <SourceMark :record="record" />
    <span v-if="record.author" class="muted ant-wrap">{{ record.author }}</span>
    <span v-if="record.seq != null" class="muted">{{ t('widgets.analysis.circumstances.journalRecord', { seq: record.seq }) }}</span>
    <span v-if="mark" class="mark ant-wrap" data-testid="record-mark">{{ mark }}</span>
  </div>
</template>

<style scoped>
.record {
  display: flex;
  flex-wrap: wrap;
  gap: 2px var(--ant-space-2);
  align-items: baseline;
  min-width: 0;
  font-size: var(--ant-fs-meta);
}

.time {
  color: var(--ant-text-3);
  font-variant-numeric: tabular-nums;
}

.text {
  font-weight: var(--ant-fw-bold);
}

.kind,
.muted {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-xs);
}

.mark {
  color: var(--ant-status-attention-text);
}
</style>
