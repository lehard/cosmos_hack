<script setup lang="ts">
/**
 * Журнал изменений паспорта (FR-43): отдельный журнал, связанный с глобальным
 * журналом событий номером записи; исправления — только новыми строками
 * (FR-122). Ось статуса показывается словарём статусов: было → стало.
 */
import { useI18n } from 'vue-i18n'
import { AXIS_TEXT, type ItemHistoryEntry, type ItemStatuses } from '@/entities/item'
import { StatusTag } from '@/shared/ui'

defineProps<{ changes: ItemHistoryEntry[] }>()
const { t, d } = useI18n()

const isAxis = (field: string): field is keyof ItemStatuses => Object.hasOwn(AXIS_TEXT, field)
const time = (x: string) => d(new Date(x), 'dateTime')
</script>

<template>
  <section class="changes" data-testid="passport-changes">
    <p v-if="!changes.length" class="muted">{{ t('empty.noRecords') }}</p>
    <table v-else class="table">
      <thead>
        <tr>
          <th>{{ t('widgets.passport.changes.record') }}</th>
          <th>{{ t('timeline.timeKind.recordedAt') }}</th>
          <th>{{ t('widgets.passport.changes.what') }}</th>
          <th>{{ t('widgets.passport.changes.beforeAfter') }}</th>
          <th>{{ t('common.words.author') }}</th>
          <th>{{ t('common.words.basis') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="c in changes" :key="`${c.seq}:${c.field}`" :data-field="c.field" :data-seq="c.seq">
          <td>{{ t('widgets.analysis.circumstances.journalRecord', { seq: c.seq }) }}</td>
          <td>{{ time(c.recorded_at) }}</td>
          <td>{{ isAxis(c.field) ? t(AXIS_TEXT[c.field]) : c.field }}</td>
          <td class="ba">
            <template v-if="isAxis(c.field)">
              <StatusTag v-if="c.before" :axis="c.field" :code="c.before" />
              <span v-else class="muted">—</span>
              <span aria-hidden="true">→</span>
              <StatusTag v-if="c.after" :axis="c.field" :code="c.after" />
              <span v-else class="muted">—</span>
            </template>
            <template v-else>{{ c.before ?? '—' }} → {{ c.after ?? '—' }}</template>
          </td>
          <td>{{ c.author ?? t('widgets.passport.changes.noAuthor') }}</td>
          <td>{{ c.reason ?? '—' }}</td>
        </tr>
      </tbody>
    </table>
  </section>
</template>

<style scoped>
.table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
}

.table th,
.table td {
  padding: 4px 6px;
  border-bottom: 1px solid #f3f4f6;
  text-align: left;
  vertical-align: top;
}

.table th {
  color: #6b7280;
  font-weight: 400;
}

.ba {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  align-items: center;
}

.muted {
  margin: 0;
  color: #6b7280;
}
</style>
