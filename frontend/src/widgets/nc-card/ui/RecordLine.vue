<script setup lang="ts">
/**
 * Строка записи журнала в карточке: время, краткое содержание от сервера,
 * пометка источника (FR-140; она же называет вид записи — «вывод системы»,
 * «решение человека»), автор. Номер записи журнала — только во втором слое
 * (`technical`).
 */
import { useI18n } from 'vue-i18n'
import { SourceMark, entryText } from '@/entities/item'
import type { NCRecordRef } from '@/entities/nonconformity'

defineProps<{ record: NCRecordRef; mark?: string | null; technical?: boolean }>()
const { t, d } = useI18n()
const time = (x: string) => d(new Date(x), 'dateTime')
</script>

<template>
  <div class="record" :data-kind="record.kind" :data-event="record.event_type" :data-id="record.event_id">
    <time class="time">{{ time(record.occurred_at) }}</time>
    <span class="text ant-wrap">{{ entryText(record) }}</span>
    <SourceMark :record="record" />
    <span v-if="record.author" class="muted ant-wrap">{{ record.author }}</span>
    <span v-if="technical && record.seq != null" class="muted">{{ t('widgets.analysis.circumstances.journalRecord', { seq: record.seq }) }}</span>
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

.muted {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-xs);
}

.mark {
  color: var(--ant-status-attention-text);
}
</style>
