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
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { AXIS_TEXT, STATUS_AXES_ORDER } from '@/entities/item'
import { NC_STATUS_TEXT, codeText, deadlineOf, decisionsBeforeRevision, type NCCard } from '@/entities/nonconformity'
import type { Density } from '@/shared/config/widget'
import { formatMinutes } from '@/shared/lib/duration'
import { StatusTag } from '@/shared/ui'
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
</script>

<template>
  <div class="nc-card" :class="`density-${density}`" :data-status="card.status" data-testid="nc-card">
    <header class="head">
      <strong class="number">{{ t('ncCard.number', { number: card.number }) }}</strong>
      <button v-if="canOpenItem" type="button" class="linklike" data-testid="open-item" @click="emit('open-item', card.item_id)">
        {{ t('common.words.item') }} {{ card.item_label }}
      </button>
      <span v-else>{{ t('common.words.item') }} {{ card.item_label }}</span>
      <span v-if="deadlineText" class="deadline" :data-overdue="deadline?.overdue || undefined" data-testid="deadline">{{ deadlineText }}</span>
    </header>
    <div class="two-statuses" data-testid="two-statuses">
      <span><span class="muted">{{ t('ncCard.twoStatuses.byItem') }}:</span> {{ codeText(NC_STATUS_TEXT, card.status, t) }}</span>
      <span><span class="muted">{{ t('ncCard.twoStatuses.investigation') }}:</span> {{ t('empty.noDataUnknown') }}</span>
      <span class="muted">{{ t('ncCard.twoStatuses.note') }}</span>
    </div>
    <p class="muted" :title="t('hints.signalVsNonconformity')">{{ t('hints.signalVsNonconformity') }}</p>

    <div class="zones" :data-view="view">
      <section class="zone" data-zone="what-happened">
        <h3>{{ t('ncCard.zones.whatHappened') }}</h3>
        <h5>{{ t('ncCard.whatHappened.beforeOperation') }}</h5>
        <ul class="list">
          <li v-for="r in card.happened.before" :key="r.event_id"><RecordLine :record="r" /></li>
          <li v-if="!card.happened.before.length" class="muted">{{ t('empty.noRecords') }}</li>
        </ul>
        <h5>{{ t('ncCard.whatHappened.operation') }}</h5>
        <dl v-if="op" class="facts" data-testid="operation">
          <dt>{{ t('common.words.operation') }}</dt>
          <dd>{{ op.label }}</dd>
          <dt>{{ t('ncCard.whatHappened.machine') }}</dt>
          <dd>{{ op.equipment_id ?? t('common.words.unknown') }}</dd>
          <dt>{{ t('ncCard.whatHappened.tool') }}</dt>
          <dd>{{ op.tool_id ?? t('common.words.unknown') }}</dd>
          <dt>{{ t('ncCard.whatHappened.program') }}</dt>
          <dd>{{ op.program_ref ?? t('common.words.unknown') }}</dd>
          <dt>{{ t('ncCard.whatHappened.performer') }}</dt>
          <dd data-testid="performer">{{ op.performer_id ?? t('empty.performerUnknown') }}</dd>
        </dl>
        <p v-else class="muted">{{ t('empty.noDataUnknown') }}</p>
        <h5>{{ t('ncCard.equipment.title') }} · {{ t('ncCard.performerActions.title') }}</h5>
        <p class="muted">{{ t('ncCard.performerActions.circumstanceNotBlame') }}</p>
        <ul class="list">
          <li v-for="r in card.happened.during" :key="r.event_id"><RecordLine :record="r" /></li>
          <li v-if="!card.happened.during.length" class="muted">{{ t('empty.noRecords') }}</li>
        </ul>
        <h5>{{ t('ncCard.whatHappened.afterOperation') }}</h5>
        <ul class="list">
          <li v-for="r in card.happened.after" :key="r.event_id"><RecordLine :record="r" /></li>
          <li v-if="!card.happened.after.length" class="muted">{{ t('empty.noRecords') }}</li>
        </ul>
      </section>

      <section class="zone" data-zone="evidence">
        <h3>{{ t('ncCard.zones.evidence') }}</h3>
        <h4>{{ t('ncCard.layers.sourceSignal') }}</h4>
        <NcSignal v-for="s in card.evidence.signals" :key="s.signal_id" :signal="s" />
        <h4>{{ t('ncCard.requirement.title') }}</h4>
        <dl v-if="card.evidence.requirement" class="facts" data-testid="requirement">
          <dt>{{ t('ncCard.requirement.characteristic') }}</dt>
          <dd>{{ card.evidence.requirement.characteristic }}</dd>
          <dt>{{ t('ncCard.requirement.tolerance') }}</dt>
          <dd>{{ card.evidence.requirement.tolerance ?? t('common.words.unknown') }}</dd>
          <dt>{{ t('ncCard.requirement.designRevision') }}</dt>
          <dd>{{ card.evidence.requirement.kd_ref ?? t('common.words.unknown') }}</dd>
        </dl>
        <p v-else class="warn" data-testid="no-requirement">{{ t('ncCard.requirement.noRequirement') }}</p>
        <h4>{{ t('ncCard.whatHappened.zoneHistory') }}</h4>
        <ul class="list">
          <li v-for="r in card.evidence.zone_history" :key="r.event_id"><RecordLine :record="r" /></li>
          <li v-if="!card.evidence.zone_history.length" class="muted">{{ t('empty.noRecords') }}</li>
        </ul>
        <p class="muted" data-testid="similar-count">{{ t('widgets.ncCard.similarCount', { n: card.evidence.similar_count }) }}</p>
      </section>

      <section class="zone" data-zone="what-to-decide">
        <h3>{{ t('ncCard.zones.whatToDecide') }}</h3>
        <h4>{{ t('ncCard.layers.systemAnalysis') }}</h4>
        <NcSystemAnalysis :analysis="card.system_analysis" />
        <h4>{{ t('ncCard.layers.humanDecisions') }}</h4>
        <ul class="list" data-testid="human-decisions">
          <li v-for="r in card.human_decisions" :key="r.event_id">
            <RecordLine :record="r" :mark="stale.has(r.event_id) ? t('timeline.marks.decisionBeforeNewData') : null" />
          </li>
          <li v-if="!card.human_decisions.length" class="muted">{{ t('empty.noDecisionIsNotNoData') }}</li>
        </ul>
        <h4>{{ t('ncCard.layers.finalStatus') }}</h4>
        <dl class="facts" data-testid="final-status">
          <template v-for="axis in STATUS_AXES_ORDER" :key="axis">
            <dt>{{ t(AXIS_TEXT[axis]) }}</dt>
            <dd><StatusTag :axis="axis" :code="card.axes[axis]" /></dd>
          </template>
        </dl>
        <p v-if="card.to_decide.concession_required" class="muted" data-testid="concession-required">{{ t('hints.concession') }}</p>
        <p class="muted">{{ t('decisions.disposition.decisionDoesNotWaitForCause') }}</p>
      </section>
    </div>
  </div>
</template>

<style scoped>
.nc-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: var(--ant-fs-body);
}

.density-large {
  font-size: var(--ant-fs-lg);
}

.head,
.two-statuses {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 14px;
  align-items: baseline;
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

.zones {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.zones[data-view='evidence'] [data-zone='evidence'] {
  order: -1;
}

@media (max-width: 1279px) {
  .zones {
    grid-template-columns: minmax(0, 1fr);
  }
}

.zone {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}

h3,
h4,
h5 {
  margin: 0;
}

h3 {
  font-size: 1.05em;
}

.list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.facts {
  display: grid;
  grid-template-columns: max-content minmax(0, 1fr);
  gap: 2px 12px;
  margin: 0;
}

.facts dt {
  color: var(--ant-text-3);
}

.facts dd {
  margin: 0;
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
  cursor: pointer;
}
</style>
