<script setup lang="ts">
/**
 * «Разбор обстоятельств» — визуальная подпись продукта (FR-153, FR-58, AD-29).
 * Три синхронные дорожки на общей шкале времени: изделие (результаты контроля),
 * человек (действия исполнителя), оборудование (профиль выполнения операции).
 * Поверх дорожек — окно возможного возникновения и интервал операции; над
 * ними — ход событий «до / во время / после операции». Клик по событию
 * подсвечивает связанные записи на всех дорожках и показывает кадр и запись
 * журнала. Формулировки — «возможные обстоятельства», не «причина».
 *
 * Компонент только показывает переданные данные (props) — источник данных
 * подключает контейнер виджета.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NAlert } from 'naive-ui'
import {
  CIRCUMSTANCE_LANES,
  PHASES,
  describeRecord,
  inWindow,
  isIncomingDefect,
  linkedIds,
  makeTimeScale,
  phaseSummary,
  recordColor,
  sortByTime,
  stackLabels,
  type CircumstanceLane,
  type CircumstanceRecord,
  type CircumstancesModel,
  type JournalRecordRef,
  type Phase,
  type RecordTone,
} from '@/entities/incident'
import { codeToKey } from '@/shared/i18n'
import type { Density } from '@/shared/config/widget'
import { ActionButton } from '@/shared/ui'

const props = withDefaults(defineProps<{ model: CircumstancesModel; density?: Density }>(), { density: 'compact' })
/** Выбранная запись (`event_id`) — двусторонняя привязка. */
const selected = defineModel<string | null>('selected', { default: null })
const emit = defineEmits<{
  /** Открыть запись журнала. */
  'open-record': [seq: number, eventId: string]
  /** Открыть кадр или материал. */
  'open-evidence': [ref: string, eventId: string]
}>()

const { t, te, d } = useI18n()

/** Подпись записи: текст по ключу или UNKNOWN(тип). */
function label(r: JournalRecordRef): string {
  const x = describeRecord(r)
  return x.key ? t(x.key, x.params) : `UNKNOWN(${x.eventType})`
}

const records = computed(() => sortByTime(props.model.records))
const incoming = computed(() => isIncomingDefect(props.model))
const summary = computed(() => phaseSummary(props.model))
const linked = computed(() => linkedIds(records.value, selected.value))
const selectedRecord = computed(() => records.value.find((r) => r.event_id === selected.value) ?? null)

/** Общая шкала: все записи, окно и операция. */
const scale = computed(() => {
  const m = props.model
  const moments: string[] = records.value.flatMap((r) => (r.ended_at ? [r.occurred_at, r.ended_at] : [r.occurred_at]))
  if (m.window) moments.push(m.window.start, m.window.end)
  if (m.operation) moments.push(...[m.operation.started_at, m.operation.finished_at].filter((x): x is string => x !== null))
  return makeTimeScale(moments)
})

const LEVEL_PX = 22
/** Правее этой позиции (%) подпись ставится слева от отметки. */
const FLIP_AT = 70

interface Mark {
  r: CircumstanceRecord
  text: string
  tone: RecordTone
  left: number
  width: number
  level: number
  flip: boolean
}

/** Отметки дорожки с ярусами подписей. */
const lanes = computed(() =>
  CIRCUMSTANCE_LANES.map((lane) => {
    const rs = records.value.filter((r) => r.lane === lane)
    const texts = rs.map(label)
    const lefts = rs.map((r) => scale.value.pos(r.occurred_at))
    const widths = texts.map((s) => Math.min(45, s.length * 0.8 + 3))
    // У правого края подпись — слева от отметки, чтобы не уходить за шкалу.
    const flips = lefts.map((l) => l > FLIP_AT)
    const levels = stackLabels(
      lefts.map((l, i) => (flips[i] ? l - widths[i]! : l)),
      widths,
    )
    const marks: Mark[] = rs.map((r, i) => ({
      r,
      text: texts[i]!,
      tone: describeRecord(r).tone,
      left: lefts[i]!,
      width: r.ended_at ? Math.max(0.6, scale.value.pos(r.ended_at) - lefts[i]!) : 0,
      level: levels[i]!,
      flip: flips[i]!,
    }))
    const depth = marks.reduce((m, x) => Math.max(m, x.level + 1), 1)
    return { lane, marks, height: depth * LEVEL_PX + 12 }
  }),
)

const LANE_TITLE: Record<CircumstanceLane, string> = {
  item: 'ncCard.circumstances.laneItem',
  person: 'ncCard.circumstances.lanePerson',
  equipment: 'ncCard.circumstances.laneEquipment',
}

const PHASE_TITLE: Record<Phase, string> = {
  before: 'ncCard.whatHappened.beforeOperation',
  during: 'ncCard.whatHappened.operation',
  after: 'ncCard.whatHappened.afterOperation',
}

const band = (a: string, b: string | null) => {
  const l = scale.value.pos(a)
  const r = b === null ? 100 : scale.value.pos(b)
  return { left: `${l}%`, width: `${Math.max(0.4, r - l)}%` }
}

const time = (x: string | number) => d(new Date(x), 'time')
const dateTime = (x: string) => d(new Date(x), 'dateTime')

function toggle(id: string): void {
  selected.value = selected.value === id ? null : id
}

/** Действие исполнителя при входном дефекте не связывается с дефектом (FR-58). */
const mutedPerson = (lane: CircumstanceLane) => lane === 'person' && incoming.value

const sourceText = (kind: string) => {
  const key = `timeline.sourceKind.${codeToKey(kind)}`
  return te(key) ? t(key) : kind
}

const missingText = (code: string) => t(`widgets.analysis.missing.${codeToKey(code)}`)
</script>

<template>
  <div class="circumstances" :class="`density-${density}`" data-testid="circumstances">
    <h3 class="heading">{{ t('ncCard.circumstances.possibleCircumstances') }}</h3>

    <NAlert v-if="incoming" type="info" :bordered="false" :show-icon="false" class="note" data-testid="incoming-note">
      {{ t('ncCard.circumstances.incomingNotPerformer') }}
    </NAlert>
    <NAlert v-if="!model.conclusion_is_categorical" type="warning" :bordered="false" :show-icon="false" class="note" data-testid="not-categorical">
      {{ t('ncCard.circumstances.noCategoricalConclusion') }}
    </NAlert>
    <NAlert v-if="!model.window" type="warning" :bordered="false" :show-icon="false" class="note" data-testid="window-unknown">
      {{ t('widgets.analysis.circumstances.windowUnknown') }}
    </NAlert>

    <!-- Ход событий: до операции чисто → во время нештатности → после находка (FR-149). -->
    <ol v-if="model.operation" class="phases" :aria-label="t('widgets.analysis.circumstances.phaseSummary')">
      <li v-for="phase in PHASES" :key="phase" class="phase" :data-phase="phase">
        <div class="phase-title">
          {{ t(PHASE_TITLE[phase]) }}<template v-if="phase === 'during'"> · {{ model.operation.label }}</template>
        </div>
        <ul v-if="summary[phase].length" class="phase-list">
          <li v-for="r in summary[phase]" :key="r.event_id" :data-tone="describeRecord(r).tone">
            <span class="dot" :style="{ background: recordColor(describeRecord(r).tone) }" aria-hidden="true" />
            <button type="button" class="linklike" @click="toggle(r.event_id)">{{ label(r) }}</button>
          </li>
        </ul>
        <p v-else class="muted">{{ t('widgets.analysis.circumstances.nothingNotable') }}</p>
      </li>
    </ol>

    <!-- Три дорожки на общей шкале. -->
    <div class="timeline" :style="{ '--lanes': lanes.length }" role="group" :aria-label="t('desks.circumstances')">
      <div class="axis-label" />
      <div class="axis">
        <span v-for="tick in scale.ticks" :key="tick" class="tick" :style="{ left: `${scale.pos(tick)}%` }">{{ time(tick) }}</span>
      </div>

      <template v-for="(l, i) in lanes" :key="l.lane">
        <div class="lane-label" :class="{ sep: i > 0 }" :style="{ gridRow: i + 2 }">{{ t(LANE_TITLE[l.lane]) }}</div>
        <div class="lane" :class="{ sep: i > 0 }" :style="{ gridRow: i + 2, height: `${l.height}px` }" :data-lane="l.lane">
          <!-- Интервалы (цикл, отклонение) — полосой от начала до конца. -->
          <span
            v-for="m in l.marks.filter((x) => x.width)"
            :key="`${m.r.event_id}-span`"
            class="interval"
            :class="{ 'is-dim': selected !== null && !linked.has(m.r.event_id) }"
            :style="{ left: `${m.left}%`, width: `${m.width}%`, top: `${m.level * LEVEL_PX + 15}px`, '--c': recordColor(m.tone) }"
            aria-hidden="true"
          />
          <button
            v-for="m in l.marks"
            :key="m.r.event_id"
            type="button"
            class="mark"
            :class="{
              'is-selected': selected === m.r.event_id,
              'is-linked': linked.has(m.r.event_id) && selected !== m.r.event_id,
              'is-dim': selected !== null && !linked.has(m.r.event_id),
              'is-muted': mutedPerson(l.lane),
              'in-window': inWindow(m.r, model.window),
              flip: m.flip,
            }"
            :style="{ left: `${m.left}%`, top: `${m.level * LEVEL_PX + 6}px`, '--c': recordColor(m.tone) }"
            :data-event="m.r.event_id"
            :data-tone="m.tone"
            :title="mutedPerson(l.lane) ? `${m.text} — ${t('ncCard.circumstances.incomingNotPerformer')}` : `${time(m.r.occurred_at)} · ${m.text}`"
            :aria-pressed="selected === m.r.event_id"
            @click="toggle(m.r.event_id)"
          >
            <span class="shape" :data-shape="m.tone" aria-hidden="true" />
            <span class="text">{{ m.text }}</span>
          </button>
        </div>
      </template>

      <!-- Окно возможного возникновения и интервал операции — поверх всех дорожек. -->
      <div class="overlay" :style="{ gridRow: `2 / ${lanes.length + 2}` }" aria-hidden="true">
        <div v-if="model.operation" class="op-band" :style="band(model.operation.started_at, model.operation.finished_at)" data-testid="operation-band">
          <span class="band-label">{{ t('widgets.analysis.circumstances.operation', { name: model.operation.label }) }}</span>
        </div>
        <div v-if="model.window" class="window-band" :style="band(model.window.start, model.window.end)" :title="t('hints.causalWindow')" data-testid="window-band">
          <span class="band-label">{{ t('ncCard.causalWindow.title') }}</span>
        </div>
      </div>
    </div>

    <p v-if="model.window" class="window-legend" data-testid="window-legend">
      <span class="swatch" aria-hidden="true" />
      <strong>{{ t('ncCard.causalWindow.title') }}:</strong>
      {{ t('ncCard.causalWindow.range', { from: dateTime(model.window.start), to: dateTime(model.window.end) }) }}
      <span class="muted">— {{ t('ncCard.causalWindow.lowerBound') }} → {{ t('ncCard.causalWindow.upperBound') }}</span>
    </p>

    <!-- Связанный кадр и запись журнала выбранного события. -->
    <section v-if="selectedRecord" class="details" data-testid="details">
      <header class="details-title">
        <span class="dot" :style="{ background: recordColor(describeRecord(selectedRecord).tone) }" aria-hidden="true" />
        {{ label(selectedRecord) }}
      </header>
      <dl>
        <dt>{{ t('timeline.timeKind.occurredAt') }}</dt>
        <dd>
          {{ dateTime(selectedRecord.occurred_at) }}<template v-if="selectedRecord.ended_at"> — {{ time(selectedRecord.ended_at) }}</template>
          · {{ inWindow(selectedRecord, model.window) ? t('widgets.analysis.circumstances.inWindow') : t('widgets.analysis.circumstances.outsideWindow') }}
        </dd>
        <template v-if="selectedRecord.source_kind">
          <dt>{{ t('timeline.sourceKind.title') }}</dt>
          <dd>{{ sourceText(selectedRecord.source_kind) }}</dd>
        </template>
        <dt>{{ t('common.words.evidence') }}</dt>
        <dd>
          <ActionButton
            v-for="ref in selectedRecord.evidence_refs ?? []"
            :key="ref"
            size="tiny"
            secondary
            class="chip"
            data-testid="evidence"
            @click="emit('open-evidence', ref, selectedRecord.event_id)"
            :label="t('widgets.analysis.circumstances.frame', { ref })"
          />
          <span v-if="!selectedRecord.evidence_refs?.length" class="muted">{{ t('widgets.analysis.circumstances.noFrame') }}</span>
        </dd>
        <dt>{{ t('timeline.feed') }}</dt>
        <dd>
          <ActionButton
            v-if="selectedRecord.journal_seq != null"
            size="tiny"
            secondary
            class="chip"
            data-testid="journal-record"
            @click="emit('open-record', selectedRecord.journal_seq, selectedRecord.event_id)"
            :label="t('widgets.analysis.circumstances.journalRecord', { seq: selectedRecord.journal_seq })"
          />
          <span v-else class="muted">{{ t('widgets.analysis.circumstances.journalRecordUnknown') }}</span>
        </dd>
        <template v-if="linked.size > 1">
          <dt>{{ t('widgets.analysis.circumstances.linked') }}</dt>
          <dd>
            <template v-for="r in records" :key="r.event_id">
              <button v-if="linked.has(r.event_id) && r.event_id !== selectedRecord.event_id" type="button" class="linklike block" @click="toggle(r.event_id)">
                {{ time(r.occurred_at) }} · {{ label(r) }}
              </button>
            </template>
          </dd>
        </template>
      </dl>
      <p v-if="selectedRecord.lane === 'person'" class="muted">{{ t('ncCard.performerActions.circumstanceNotBlame') }}</p>
    </section>
    <p v-else class="muted hint">{{ t('widgets.analysis.circumstances.selectHint') }}</p>

    <section v-if="model.missing_information.length" class="missing" data-testid="missing">
      <strong>{{ t('widgets.analysis.missing.title') }}:</strong>
      <span v-for="m in model.missing_information" :key="m" class="chip-text">{{ missingText(m) }}</span>
    </section>
  </div>
</template>

<style scoped>
.circumstances {
  display: flex;
  flex-direction: column;
  gap: 12px;
  font-size: var(--ant-fs-body);
}

.density-large {
  font-size: var(--ant-fs-lg);
}

.heading {
  margin: 0;
  font-size: 1.05em;
}

.note {
  padding: 6px 10px;
}

.muted {
  margin: 0;
  color: var(--ant-text-3);
}

.dot {
  display: inline-block;
  flex: none;
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.linklike {
  padding: 0;
  border: 0;
  background: none;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.linklike:hover {
  text-decoration: underline;
}

.linklike.block {
  display: block;
}

/* Ход событий «до → во время → после». */
.phases {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0;
  margin: 0;
  padding: 0;
  list-style: none;
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-md);
}

.phase {
  padding: 8px 12px;
  min-width: 0;
}

.phase + .phase {
  border-left: 1px solid var(--ant-border);
}

.phase[data-phase='during'] {
  background: var(--ant-surface-subtle);
}

.phase-title {
  margin-bottom: 4px;
  font-weight: var(--ant-fw-bold);
}

.phase-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.phase-list li {
  display: flex;
  gap: 6px;
  align-items: baseline;
}

/* Дорожки: колонка подписей + колонка шкалы; оверлей окна — во всю высоту. */
.timeline {
  display: grid;
  grid-template-columns: 104px minmax(0, 1fr);
  grid-auto-rows: auto;
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-md);
  overflow: hidden;
}

.axis-label {
  grid-row: 1;
  grid-column: 1;
  border-bottom: 1px solid var(--ant-border);
}

.axis {
  position: relative;
  grid-row: 1;
  grid-column: 2;
  height: 24px;
  border-bottom: 1px solid var(--ant-border);
}

.tick {
  position: absolute;
  top: 5px;
  transform: translateX(-50%);
  color: var(--ant-text-3);
  font-family: var(--ant-font-mono);
  font-size: var(--ant-fs-xs);
  white-space: nowrap;
}

.lane-label {
  grid-column: 1;
  display: flex;
  align-items: center;
  padding: 0 10px;
  font-weight: var(--ant-fw-bold);
  background: var(--ant-surface-subtle);
  border-right: 1px solid var(--ant-border);
}

.lane {
  position: relative;
  grid-column: 2;
  z-index: 1;
}

.lane-label.sep,
.lane.sep {
  border-top: 1px dashed var(--ant-border);
}

.interval {
  position: absolute;
  height: 4px;
  border-radius: 2px;
  background: var(--c);
  opacity: 0.5;
}

.interval.is-dim {
  opacity: 0.15;
}

.overlay {
  position: relative;
  grid-column: 2;
  z-index: 0;
  pointer-events: none;
}

.op-band,
.window-band {
  position: absolute;
  top: 0;
  bottom: 0;
}

.op-band {
  background: repeating-linear-gradient(135deg, color-mix(in srgb, var(--ant-accent) 5%, transparent) 0 6px, transparent 6px 12px);
  border-left: 1px solid color-mix(in srgb, var(--ant-accent) 40%, transparent);
  border-right: 1px solid color-mix(in srgb, var(--ant-accent) 40%, transparent);
}

.window-band {
  background: color-mix(in srgb, var(--ant-status-attention) 12%, transparent);
  border-left: 2px solid color-mix(in srgb, var(--ant-status-attention) 70%, transparent);
  border-right: 2px solid color-mix(in srgb, var(--ant-status-attention) 70%, transparent);
}

.band-label {
  position: absolute;
  bottom: 2px;
  left: 4px;
  color: var(--ant-text-3);
  font-size: 10px;
  white-space: nowrap;
}

.window-band .band-label {
  top: 2px;
  bottom: auto;
  color: var(--ant-status-attention-text);
}

/* Отметка: форма по смыслу + подпись; цвет — из словаря статусов. */
.mark {
  position: absolute;
  display: flex;
  gap: 4px;
  align-items: center;
  max-width: 45%;
  height: 20px;
  padding: 0 4px 0 0;
  border: 0;
  border-radius: var(--ant-radius-sm);
  background: transparent;
  color: var(--ant-text);
  font: inherit;
  font-size: var(--ant-fs-meta);
  cursor: pointer;
  transform: translateX(-5px);
}

.mark.flip {
  flex-direction: row-reverse;
  padding: 0 0 0 4px;
  transform: translateX(calc(-100% + 5px));
}

.mark:hover,
.mark:focus-visible {
  background: color-mix(in srgb, var(--ant-surface) 90%, transparent);
  outline: 1px solid var(--ant-n-300);
  z-index: 2;
}

.mark .text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.shape {
  flex: none;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--c);
}

.shape[data-shape='finding'] {
  border-radius: 1px;
  transform: rotate(45deg);
}

.shape[data-shape='deviation'] {
  width: 0;
  height: 0;
  border-radius: 0;
  background: none;
  border-left: 6px solid transparent;
  border-right: 6px solid transparent;
  border-bottom: 10px solid var(--c);
}

.shape[data-shape='unable'] {
  background: var(--ant-surface);
  border: 2px solid var(--c);
}

.shape[data-shape='action'] {
  width: 7px;
  height: 7px;
}

.mark.is-selected {
  background: var(--ant-surface);
  outline: 2px solid var(--ant-text);
  font-weight: var(--ant-fw-bold);
  z-index: 3;
}

.mark.is-linked {
  background: var(--ant-surface);
  outline: 2px dashed var(--ant-accent);
  z-index: 3;
}

.mark.is-dim {
  opacity: 0.35;
}

.mark.is-muted {
  color: var(--ant-n-400);
}

.window-legend {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
  margin: 0;
}

.swatch {
  width: 14px;
  height: 10px;
  background: color-mix(in srgb, var(--ant-status-attention) 25%, transparent);
  border: 1px solid color-mix(in srgb, var(--ant-status-attention) 70%, transparent);
}

.details {
  padding: 8px 12px;
  border: 1px solid var(--ant-n-300);
  border-radius: var(--ant-radius-md);
  background: var(--ant-surface-subtle);
}

.details-title {
  display: flex;
  gap: 6px;
  align-items: center;
  font-weight: var(--ant-fw-bold);
}

.details dl {
  display: grid;
  grid-template-columns: max-content 1fr;
  gap: 4px 12px;
  margin: 8px 0 0;
}

.details dt {
  color: var(--ant-text-3);
}

.details dd {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  align-items: center;
  margin: 0;
}

.missing {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: baseline;
}

.chip-text {
  padding: 1px 8px;
  border-radius: var(--ant-radius-lg);
  background: var(--ant-n-100);
}
</style>
