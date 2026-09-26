<script setup lang="ts">
/**
 * Записи паспорта «кто · что · когда» (FR-42, FR-140, FR-68): факты, выводы
 * системы, решения людей, служебные записи — различимы; у каждой — пометка
 * источника, автор, время, подписи и итог их проверки, номер критического
 * действия. Исправление — отдельная запись со ссылкой на исправляемую (FR-122).
 * У наблюдения анализатора — «Как машина пришла к выводу» (Ф1 SHOW-IS2): дорожка
 * решения раскрывается по кнопке и читает наблюдение только тогда.
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NRadioButton, NRadioGroup } from 'naive-ui'
import {
  LAYER_TEXT,
  RECORD_LAYERS,
  SignatureMark,
  SourceMark,
  entryText,
  isCriticalType,
  sortEntries,
  type PassportEntry,
  type RecordLayer,
} from '@/entities/item'
import { ObservationDecisionTrace } from '@/features/decision-trace'
import { ActionButton, EmptyState } from '@/shared/ui'

const props = withDefaults(
  defineProps<{
    entries: PassportEntry[]
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
  const sorted = sortEntries(props.entries)
  const filtered = layer.value === 'all' ? sorted : sorted.filter((e) => e.kind === layer.value)
  const list = props.limit > 0 ? filtered.slice(-props.limit) : filtered
  return list.map((e) => ({ e, text: entryText(e), critical: isCriticalType(e.event_type) }))
})
const time = (x: string) => d(new Date(x), 'dateTime')

/** Наблюдение анализатора (не ручной ввод) — у него есть дорожка решения. */
const isObservation = (e: PassportEntry) => e.event_type === 'inspection.result.recorded' && e.source_kind !== 'manual_entry'
/** Раскрытые дорожки решения (event_id). */
const traces = ref<Set<string>>(new Set())
function toggleTrace(id: string): void {
  const next = new Set(traces.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  traces.value = next
}
</script>

<template>
  <section class="entries" data-testid="passport-entries">
    <h4>{{ t('passport.whoWhatWhen') }}</h4>
    <NRadioGroup v-if="filterable" v-model:value="layer" size="small" class="filter">
      <NRadioButton value="all">{{ t('timeline.layer.all') }}</NRadioButton>
      <NRadioButton v-for="l in RECORD_LAYERS" :key="l" :value="l">{{ t(LAYER_TEXT[l]) }}</NRadioButton>
    </NRadioGroup>
    <EmptyState v-if="!rows.length" compact :title="t('empty.noRecords')" />
    <ol class="list">
      <li v-for="{ e, text, critical } in rows" :key="e.event_id" class="row" :data-kind="e.kind" :data-event="e.event_type" :data-id="e.event_id">
        <div class="line">
          <time class="time" :title="`${t('timeline.timeKind.recordedAt')}: ${time(e.recorded_at)}`">{{ time(e.occurred_at) }}</time>
          <span class="text">{{ text }}</span>
          <span class="layer">{{ t(LAYER_TEXT[e.kind]) }}</span>
          <SourceMark :record="e" />
        </div>
        <div class="line meta">
          <span v-if="e.author">{{ t('common.words.author') }}: {{ e.author }}</span>
          <span v-if="e.ca_ref" class="ca" :title="t('hints.criticalAction')">{{ e.ca_ref }}</span>
          <span v-else-if="critical" class="ca" :title="t('hints.criticalAction')">{{ t('widgets.passport.critical') }}</span>
          <span class="seq">{{ t('widgets.analysis.circumstances.journalRecord', { seq: e.seq }) }}</span>
        </div>
        <ul class="signatures">
          <li v-for="(s, i) in e.signatures" :key="i"><SignatureMark :signature="s" /></li>
          <li v-if="!e.signatures.length" class="muted" data-testid="no-signature">{{ t('widgets.passport.signature.none') }}</li>
        </ul>
        <p v-if="e.corrects" class="mark" data-testid="corrects">{{ t('timeline.marks.corrects', { eventId: e.corrects }) }}</p>
        <template v-if="isObservation(e)">
          <ActionButton
            size="small"
            quaternary
            class="trace-toggle"
            data-testid="trace-toggle"
            :aria-expanded="traces.has(e.event_id)"
            :label="traces.has(e.event_id) ? t('decisionTrace.hide') : t('decisionTrace.title')"
            @click="toggleTrace(e.event_id)"
          />
          <ObservationDecisionTrace v-if="traces.has(e.event_id)" :event-id="e.event_id" class="trace" />
        </template>
      </li>
    </ol>
  </section>
</template>

<style scoped>
.trace-toggle {
  align-self: flex-start;
}

.trace {
  margin: var(--ant-space-2) 0 var(--ant-space-2) var(--ant-space-4);
}

.entries {
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

.list,
.signatures {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.row {
  padding: 4px 0;
  border-bottom: 1px solid var(--ant-n-100);
}

.line {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 8px;
  align-items: baseline;
}

.time {
  color: var(--ant-text-3);
  font-variant-numeric: tabular-nums;
}

.text {
  font-weight: var(--ant-fw-bold);
}

.layer,
.ca,
.seq {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-xs);
}

.ca {
  font-weight: var(--ant-fw-bold);
}

.meta,
.signatures {
  padding-left: 12px;
  font-size: var(--ant-fs-meta);
}

.mark,
.muted {
  margin: 0;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.mark {
  padding-left: 12px;
}
</style>
