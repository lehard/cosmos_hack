<script setup lang="ts">
/**
 * Карточка несоответствия (FR-51, PRD §3a «Контролёр качества») в три зоны:
 * «что произошло» (до операции / операция: станок, инструмент, программа,
 * исполнитель / после), «доказательства» (исходный сигнал, требование, история
 * зоны), «что решить» (почему система это предлагает, решения людей, итоговый
 * статус, срок). Исходный сигнал, анализ системы, решения людей и итоговый
 * статус показаны раздельно; у несоответствия два статуса — по изделию и
 * системное расследование.
 */
import { computed, inject } from 'vue'
import { useI18n } from 'vue-i18n'
import { AXIS_TEXT, STATUS_AXES_ORDER } from '@/entities/item'
import { NC_STATUS_TEXT, codeText, deadlineOf, decisionsBeforeRevision, type NCCard } from '@/entities/nonconformity'
import type { Density } from '@/shared/config/widget'
import { formatMinutes } from '@/shared/lib/duration'
import { KeyValue, KeyValueList, StatusTag, WIDGET_FRAME_CONTEXT } from '@/shared/ui'
import NcSignal from './NcSignal.vue'
import NcSystemAnalysis from './NcSystemAnalysis.vue'
import RecordLine from './RecordLine.vue'

const props = withDefaults(
  defineProps<{
    card: NCCard
    /** «Сейчас» для срока решения, мс (в воспроизведении — момент воспроизведения). */
    now: number
    density?: Density
    /** Срез `view: evidence` (стол контролёра) — доказательства раскрыты первыми. */
    view?: 'evidence' | 'full'
    /** Есть экран паспорта. */
    canOpenItem?: boolean
  }>(),
  { density: 'comfortable', view: 'full', canOpenItem: true },
)
const emit = defineEmits<{ 'open-item': [itemId: string] }>()
const { t } = useI18n()

const deadline = computed(() => deadlineOf(props.card.to_decide.decision_due_at, props.now))
const deadlineText = computed(() => {
  const dl = deadline.value
  if (!dl) return null
  const time = formatMinutes(t, dl.minutes)
  return dl.overdue ? t('ncCard.deadline.overdueBy', { time }) : t('ncCard.deadline.countdown', { time })
})
const stale = computed(() => decisionsBeforeRevision(props.card))
const op = computed(() => props.card.happened.operation ?? null)
/** Карточка в окне записи (Д-70): заголовок окна уже несёт номер, изделие и статус, внизу — панель решения. */
const frame = inject(WIDGET_FRAME_CONTEXT, {})
const inWindow = computed(() => !!frame.plain)
</script>

<template>
  <div class="nc-card" :class="`density-${density}`" :data-status="card.status" :data-in-window="inWindow || undefined" data-testid="nc-card">
    <!-- В окне записи номер и изделие уже в заголовке окна, паспорт — на вкладке: здесь только срок. -->
    <header v-if="!inWindow || deadlineText" class="head">
      <template v-if="!inWindow">
        <strong class="number ant-wrap">{{ t('ncCard.number', { number: card.number }) }}</strong>
        <button v-if="canOpenItem" type="button" class="linklike ant-wrap" data-testid="open-item" @click="emit('open-item', card.item_id)">
          {{ t('common.words.item') }} {{ card.item_label }}
        </button>
        <span v-else class="ant-wrap">{{ t('common.words.item') }} {{ card.item_label }}</span>
      </template>
      <span v-if="deadlineText" class="deadline" :data-overdue="deadline?.overdue || undefined" data-testid="deadline">{{ deadlineText }}</span>
    </header>
    <div class="two-statuses" data-testid="two-statuses">
      <span class="ant-wrap"><span class="muted">{{ t('ncCard.twoStatuses.byItem') }}:</span> {{ codeText(NC_STATUS_TEXT, card.status, t) }}</span>
      <span class="ant-wrap"><span class="muted">{{ t('ncCard.twoStatuses.investigation') }}:</span> {{ t('empty.noDataUnknown') }}</span>
      <span class="muted ant-wrap">{{ t('ncCard.twoStatuses.note') }}</span>
    </div>
    <p class="muted ant-wrap">{{ t('hints.signalVsNonconformity') }}</p>

    <div class="zones" :data-view="view">
      <section class="zone" data-zone="what-happened">
        <h3 class="ant-wrap">{{ t('ncCard.zones.whatHappened') }}</h3>
        <h5 class="ant-wrap">{{ t('ncCard.whatHappened.beforeOperation') }}</h5>
        <ul class="list">
          <li v-for="r in card.happened.before" :key="r.event_id"><RecordLine :record="r" /></li>
          <li v-if="!card.happened.before.length" class="muted">{{ t('empty.noRecords') }}</li>
        </ul>
        <h5 class="ant-wrap">{{ t('ncCard.whatHappened.operation') }}</h5>
        <KeyValueList v-if="op" data-testid="operation">
          <KeyValue :label="t('common.words.operation')" :value="op.label" />
          <KeyValue :label="t('ncCard.whatHappened.machine')" :value="op.equipment_id ?? t('common.words.unknown')" />
          <KeyValue :label="t('ncCard.whatHappened.tool')" :value="op.tool_id ?? t('common.words.unknown')" />
          <KeyValue :label="t('ncCard.whatHappened.program')" :value="op.program_ref ?? t('common.words.unknown')" />
          <KeyValue :label="t('ncCard.whatHappened.performer')">
            <span data-testid="performer">{{ op.performer_id ?? t('empty.performerUnknown') }}</span>
          </KeyValue>
        </KeyValueList>
        <p v-else class="muted">{{ t('empty.noDataUnknown') }}</p>
        <h5 class="ant-wrap">{{ t('ncCard.equipment.title') }} · {{ t('ncCard.performerActions.title') }}</h5>
        <p class="muted ant-wrap">{{ t('ncCard.performerActions.circumstanceNotBlame') }}</p>
        <ul class="list">
          <li v-for="r in card.happened.during" :key="r.event_id"><RecordLine :record="r" /></li>
          <li v-if="!card.happened.during.length" class="muted">{{ t('empty.noRecords') }}</li>
        </ul>
        <h5 class="ant-wrap">{{ t('ncCard.whatHappened.afterOperation') }}</h5>
        <ul class="list">
          <li v-for="r in card.happened.after" :key="r.event_id"><RecordLine :record="r" /></li>
          <li v-if="!card.happened.after.length" class="muted">{{ t('empty.noRecords') }}</li>
        </ul>
      </section>

      <section class="zone" data-zone="evidence">
        <h3 class="ant-wrap">{{ t('ncCard.zones.evidence') }}</h3>
        <h4 class="ant-wrap">{{ t('ncCard.layers.sourceSignal') }}</h4>
        <NcSignal v-for="s in card.evidence.signals" :key="s.signal_id" :signal="s" />
        <h4 class="ant-wrap">{{ t('ncCard.requirement.title') }}</h4>
        <KeyValueList v-if="card.evidence.requirement" data-testid="requirement">
          <KeyValue :label="t('ncCard.requirement.characteristic')" :value="card.evidence.requirement.characteristic" />
          <KeyValue :label="t('ncCard.requirement.tolerance')" :value="card.evidence.requirement.tolerance ?? t('common.words.unknown')" />
          <KeyValue :label="t('ncCard.requirement.designRevision')" :value="card.evidence.requirement.kd_ref ?? t('common.words.unknown')" mono />
        </KeyValueList>
        <p v-else class="warn ant-wrap" data-testid="no-requirement">{{ t('ncCard.requirement.noRequirement') }}</p>
        <h4 class="ant-wrap">{{ t('ncCard.whatHappened.zoneHistory') }}</h4>
        <ul class="list">
          <li v-for="r in card.evidence.zone_history" :key="r.event_id"><RecordLine :record="r" /></li>
          <li v-if="!card.evidence.zone_history.length" class="muted">{{ t('empty.noRecords') }}</li>
        </ul>
        <p class="muted ant-wrap" data-testid="similar-count">{{ t('widgets.ncCard.similarCount', { n: card.evidence.similar_count }) }}</p>
      </section>

      <section class="zone" data-zone="what-to-decide">
        <!-- В окне кнопки решения — в нижней панели «Что решить»; здесь — основания для него. -->
        <h3 class="ant-wrap">{{ inWindow ? t('ncCard.zones.basisForDecision') : t('ncCard.zones.whatToDecide') }}</h3>
        <h4 class="ant-wrap">{{ t('ncCard.layers.systemAnalysis') }}</h4>
        <NcSystemAnalysis :analysis="card.system_analysis" />
        <h4 class="ant-wrap">{{ t('ncCard.layers.humanDecisions') }}</h4>
        <ul class="list" data-testid="human-decisions">
          <li v-for="r in card.human_decisions" :key="r.event_id">
            <RecordLine :record="r" :mark="stale.has(r.event_id) ? t('timeline.marks.decisionBeforeNewData') : null" />
          </li>
          <li v-if="!card.human_decisions.length" class="muted">{{ t('empty.noDecisionIsNotNoData') }}</li>
        </ul>
        <h4 class="ant-wrap">{{ t('ncCard.layers.finalStatus') }}</h4>
        <KeyValueList data-testid="final-status">
          <KeyValue v-for="axis in STATUS_AXES_ORDER" :key="axis" :label="t(AXIS_TEXT[axis])">
            <StatusTag :axis="axis" :code="card.axes[axis]" />
          </KeyValue>
        </KeyValueList>
        <p v-if="card.to_decide.concession_required" class="muted ant-wrap" data-testid="concession-required">{{ t('hints.concession') }}</p>
        <p class="muted ant-wrap">{{ t('decisions.disposition.decisionDoesNotWaitForCause') }}</p>
      </section>
    </div>
  </div>
</template>

<style scoped>
/*
 * Раскладка — по ширине самой карточки, а не экрана (container query): в окне
 * записи (600–1080 px) зоны идут друг под другом, на широкой странице — в три колонки.
 */
.nc-card {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
  min-width: 0;
  font-size: var(--ant-fs-body);
  container: nc-card / inline-size;
}

.density-large {
  font-size: var(--ant-fs-lg);
}

.head,
.two-statuses {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-1) var(--ant-space-4);
  align-items: baseline;
  min-width: 0;
}

.number {
  font-size: 1.15em;
}

.deadline {
  color: var(--ant-text-3);
}

.deadline[data-overdue] {
  color: var(--ant-status-danger);
  font-weight: var(--ant-fw-bold);
}

/* Узко (окно записи): одна колонка, зоны разделены линией. */
.zones {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: var(--ant-space-5);
  margin-top: var(--ant-space-2);
}

.zone + .zone {
  padding-top: var(--ant-space-5);
  border-top: 1px solid var(--ant-border);
}

.zones[data-view='evidence'] [data-zone='evidence'] {
  order: -1;
}

/* Порядок «доказательства первыми»: линия — у зон после первой по порядку показа. */
.zones[data-view='evidence'] [data-zone='evidence'] {
  padding-top: 0;
  border-top: 0;
}

.zones[data-view='evidence'] [data-zone='what-happened'] {
  padding-top: var(--ant-space-5);
  border-top: 1px solid var(--ant-border);
}

/* Широко (страница карточки, стол): три колонки, как раньше. */
@container nc-card (min-width: 1100px) {
  .zones {
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: var(--ant-space-6);
  }

  .zones[data-view] .zone {
    padding-top: 0;
    border-top: 0;
  }
}

.zone {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
  min-width: 0;
}

h3,
h4,
h5 {
  margin: 0;
}

h3 {
  font-size: var(--ant-fs-title);
}

h4 {
  margin-top: var(--ant-space-2);
}

h3 + h4,
h3 + h5 {
  margin-top: 0;
}

h5 {
  margin-top: var(--ant-space-1);
  color: var(--ant-text-2);
}

.list {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  min-width: 0;
  margin: 0;
  padding: 0;
  list-style: none;
}

.muted {
  margin: 0;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.warn {
  margin: 0;
  color: var(--ant-status-attention-text);
}

.linklike {
  padding: 0;
  border: 0;
  background: none;
  color: var(--ant-accent);
  font: inherit;
  text-align: start;
  cursor: pointer;
}
</style>
