<script setup lang="ts">
/**
 * Исходные записи вклада изделия (FR-7, AD-45, кейс §7.2): id записей из строки
 * вклада → записи журнала изделия (`item.passport.read`). У каждой — вид записи
 * (факт источника / вывод системы / решение человека / служебная), тип, время,
 * номер в журнале и пометка источника (FR-140). Запись, которой нет в паспорте,
 * не прячется — показан её id.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NSpin } from 'naive-ui'
import { entryText, LAYER_TEXT, SourceMark, sortEntries, usePassport } from '@/entities/item'
import { useProblemText } from '@/shared/i18n/problem'

const props = defineProps<{ itemId: string; eventIds: string[] }>()
const { t, d } = useI18n()
const problemText = useProblemText()
const passport = usePassport(() => props.itemId)

const entries = computed(() => {
  const all = passport.data.value?.data.entries ?? []
  return sortEntries(all.filter((e) => props.eventIds.includes(e.event_id)))
})
const missing = computed(() => (passport.data.value ? props.eventIds.filter((id) => !entries.value.some((e) => e.event_id === id)) : []))
</script>

<template>
  <div class="records" data-testid="records">
    <NSpin v-if="passport.isLoading.value" size="small" />
    <p v-else-if="passport.error.value" class="error">{{ problemText(passport.error.value) }}</p>
    <template v-else>
      <ol class="list">
        <li v-for="e in entries" :key="e.event_id" class="record" :data-event="e.event_id" :data-kind="e.kind">
          <span class="layer">{{ t(LAYER_TEXT[e.kind]) }}</span>
          <span class="text">{{ entryText(e) }}</span>
          <span class="meta">
            <time :datetime="e.occurred_at">{{ d(new Date(e.occurred_at), 'dateTime') }}</time>
            · {{ t('widgets.analytics.drilldown.seq', { seq: e.seq }) }}
            · <code>{{ e.event_type }}</code>
          </span>
          <SourceMark :record="e" />
        </li>
      </ol>
      <p v-for="id in missing" :key="id" class="missing" data-testid="missing-record">{{ t('widgets.analytics.drilldown.notInPassport', { id }) }}</p>
    </template>
  </div>
</template>

<style scoped>
.records {
  margin: 4px 0 0;
  padding: 6px 8px;
  border-left: 2px solid #e5e7eb;
  font-size: 12px;
}

.list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.record {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 8px;
  align-items: baseline;
}

.layer {
  color: #374151;
  font-weight: 600;
}

.record[data-kind='decision'] .layer {
  color: #1f2937;
}

.meta {
  color: #6b7280;
}

.missing,
.error {
  margin: 4px 0 0;
  color: #7a5a00;
}
</style>
