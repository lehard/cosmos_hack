<script setup lang="ts">
/**
 * «Как было дело» (FR-51, PRD §3a): одна вертикальная лента — до операции →
 * операция (станок, программа, исполнитель, время) → во время → после. У
 * каждой записи время, отметка смысла (норма / обстоятельство / признак
 * дефекта), текст сервера и источник факта (FR-140); у записи с числами режима —
 * полоса «уставка / наблюдалось» (один раз на параметр). Действия исполнителя —
 * обстоятельство, а не вина. Узкое окно записи: без горизонтальной прокрутки.
 * Коротко (UI-53): в фазе до 3 главных событий — сначала признаки дефекта, потом
 * обстоятельства, потом нормы (по времени внутри); остальное — «Показать всю историю».
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { SourceMark, entryText } from '@/entities/item'
import { evidencePhases, recordTone, type NCCard, type RecordTone } from '@/entities/nonconformity'
import RegimeBar from './RegimeBar.vue'

const props = defineProps<{ card: NCCard }>()
const { t, d } = useI18n()

const PER_PHASE = 3
const RANK: Record<RecordTone, number> = { danger: 0, warn: 1, ok: 2, neutral: 3 }
const all = ref(false)
const full = computed(() => evidencePhases(props.card))
/** Главные события фазы: не больше трёх, по важности, внутри — по времени. */
const phases = computed(() =>
  full.value.map((b) => {
    if (all.value || b.records.length <= PER_PHASE) return b
    const keep = new Set(
      [...b.records]
        .map((r, i) => ({ r, i }))
        .sort((a, b2) => RANK[recordTone(a.r)] - RANK[recordTone(b2.r)] || a.i - b2.i)
        .slice(0, PER_PHASE)
        .map((x) => x.r.event_id),
    )
    return { ...b, records: b.records.filter((r) => keep.has(r.event_id)) }
  }),
)
const hidden = computed(() => full.value.reduce((n, b) => n + b.records.length, 0) - phases.value.reduce((n, b) => n + b.records.length, 0))
/** Полоса режима — один раз на параметр: у последней по времени записи с числами. */
const barAt = computed(() => {
  const last = new Map<string, string>()
  for (const b of phases.value) for (const r of b.records) if (r.reading) last.set(r.reading.parameter, r.event_id)
  return new Set(last.values())
})
const op = computed(() => props.card.happened.operation ?? null)
const time = (x: string) => d(new Date(x), 'dateTime')
const clock = (x: string) => d(new Date(x), 'time')
const PHASE_TITLE = { before: 'ncCard.whatHappened.beforeOperation', during: 'ncCard.timeline.during', after: 'ncCard.whatHappened.afterOperation' } as const
const TONE_TEXT: Record<RecordTone, string> = {
  ok: 'ncCard.timeline.tone.ok',
  warn: 'ncCard.timeline.tone.warn',
  danger: 'ncCard.timeline.tone.danger',
  neutral: 'ncCard.timeline.tone.neutral',
}
</script>

<template>
  <ol class="timeline" data-testid="evidence-timeline">
    <template v-for="block in phases" :key="block.phase">
      <li v-if="block.phase === 'during'" class="operation" data-testid="operation">
        <span class="marker" data-tone="operation" aria-hidden="true" />
        <div class="body">
          <p class="phase ant-wrap">{{ t('ncCard.whatHappened.operation') }}</p>
          <template v-if="op">
            <p class="op-title ant-wrap">{{ op.label }}</p>
            <p class="op-meta ant-wrap">
              <span v-if="op.started_at">{{ time(op.started_at) }}<template v-if="op.finished_at">–{{ clock(op.finished_at) }}</template></span>
              <span>{{ t('ncCard.whatHappened.machine') }}: {{ op.equipment_label ?? op.equipment_id ?? t('common.words.unknown') }}</span>
              <span>{{ t('ncCard.whatHappened.program') }}: {{ op.program_ref ?? t('common.words.unknown') }}</span>
              <span v-if="op.tool_id">{{ t('ncCard.whatHappened.tool') }}: {{ op.tool_id }}</span>
              <span>{{ t('ncCard.whatHappened.performer') }}: <span data-testid="performer">{{ op.performer_id ?? t('empty.performerUnknown') }}</span></span>
            </p>
          </template>
          <p v-else class="muted">{{ t('empty.noDataUnknown') }}</p>
        </div>
      </li>
      <li class="phase-row" :data-phase="block.phase">
        <p class="phase ant-wrap">{{ t(PHASE_TITLE[block.phase]) }}</p>
        <p v-if="block.phase === 'during'" class="muted ant-wrap">{{ t('ncCard.performerActions.circumstanceNotBlame') }}</p>
      </li>
      <li v-for="r in block.records" :key="r.event_id" class="entry" :data-tone="recordTone(r)" :data-event="r.event_type" :data-id="r.event_id">
        <span class="marker" :data-tone="recordTone(r)" :title="t(TONE_TEXT[recordTone(r)])" aria-hidden="true" />
        <div class="body">
          <p class="line">
            <time class="time">{{ time(r.occurred_at) }}</time>
            <SourceMark :record="r" />
            <span v-if="r.author && r.kind === 'decision'" class="muted ant-wrap">{{ r.author }}</span>
          </p>
          <p class="text ant-wrap">{{ entryText(r) }}</p>
          <RegimeBar v-if="r.reading && barAt.has(r.event_id)" :reading="r.reading" />
        </div>
      </li>
      <li v-if="!block.records.length" class="entry empty">
        <span class="marker" data-tone="neutral" aria-hidden="true" />
        <p class="muted body">{{ t('empty.noRecords') }}</p>
      </li>
    </template>
  </ol>
  <button v-if="hidden > 0 || all" type="button" class="more" data-testid="toggle-history" @click="all = !all">
    {{ all ? t('ncCard.timeline.showMain') : t('ncCard.timeline.showAll', { n: hidden }) }}
  </button>
</template>

<style scoped>
/* Лента: слева вертикальная линия с отметками, справа текст. */
.timeline {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
  min-width: 0;
  margin: 0;
  padding: 0 0 0 var(--ant-space-5);
  list-style: none;
}

.timeline::before {
  position: absolute;
  top: var(--ant-space-2);
  bottom: var(--ant-space-2);
  left: 5px;
  width: 2px;
  background: var(--ant-border);
  content: '';
}

li {
  position: relative;
  min-width: 0;
}

.marker {
  position: absolute;
  top: 5px;
  left: calc(-1 * var(--ant-space-5));
  width: 12px;
  height: 12px;
  border: 2px solid var(--ant-surface);
  border-radius: 50%;
  background: var(--ant-status-neutral);
}

.marker[data-tone='ok'] {
  background: var(--ant-status-success);
}

.marker[data-tone='warn'] {
  background: var(--ant-status-attention);
}

.marker[data-tone='danger'] {
  background: var(--ant-status-danger);
}

.marker[data-tone='operation'] {
  border-radius: var(--ant-radius-sm);
  background: var(--ant-accent);
}

.body {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

p {
  margin: 0;
}

.phase-row {
  margin-top: var(--ant-space-1);
}

.phase {
  color: var(--ant-text-2);
  font-size: var(--ant-fs-meta);
  font-weight: var(--ant-fw-bold);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.operation {
  padding: var(--ant-space-2) var(--ant-space-3);
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-md);
  background: var(--ant-surface-subtle);
}

.operation .marker {
  left: calc(-1 * var(--ant-space-5) - 1px);
  top: calc(var(--ant-space-2) + 5px);
}

.op-title {
  font-weight: var(--ant-fw-bold);
}

.op-meta,
.line {
  display: flex;
  flex-wrap: wrap;
  gap: 2px var(--ant-space-3);
  align-items: baseline;
  min-width: 0;
  color: var(--ant-text-2);
  font-size: var(--ant-fs-meta);
}

.time {
  color: var(--ant-text-3);
  font-variant-numeric: tabular-nums;
}

.entry[data-tone='danger'] .text {
  color: var(--ant-status-danger-text);
  font-weight: var(--ant-fw-bold);
}

.entry[data-tone='warn'] .text {
  color: var(--ant-status-attention-text);
}

.muted {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}
.more {
  align-self: flex-start;
  margin-top: var(--ant-space-2);
  padding: 0;
  border: 0;
  background: none;
  color: var(--ant-accent);
  font: inherit;
  font-size: var(--ant-fs-meta);
  cursor: pointer;
}
</style>
