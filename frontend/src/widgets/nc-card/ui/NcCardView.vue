<script setup lang="ts">
/**
 * Карточка несоответствия (FR-51, PRD §3a «Контролёр качества», UI-24) —
 * досье для решения контролёра, от смысла к технике:
 * 1. «Что случилось» — признак дефекта крупно, изделие, операция, срок, два
 *    статуса (по изделию и расследование), уверенность анализатора рядом с
 *    качеством наблюдения, кадр контроля;
 * 2. «Почему это проблема» — требование КД рядом с наблюдением; нет требования —
 *    вопрос технологу, а не брак;
 * 3. «Как было дело» — одна лента до → операция → во время → после, похожие случаи;
 * 4. «Что предлагает система» — предложение, основания, альтернативы, чего не
 *    хватает; решение принимает человек;
 * 5. «Решения людей» и итоговый статус по осям;
 * 6. «Подробности для проверки» (второй слой, свёрнут) — исходный сигнал целиком:
 *    ступени и версии анализатора, материалы, правило и режим, история зоны.
 * Исходный сигнал, анализ системы, решения людей и итоговый статус — раздельно (AD-2).
 */
import { computed, inject, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { AXIS_TEXT, STATUS_AXES_ORDER, entryText } from '@/entities/item'
import {
  BASIS_KIND_TEXT,
  INVESTIGATION_TEXT,
  NC_STATUS_TEXT,
  SEVERITY_TEXT,
  bpToFraction,
  codeText,
  conclusionVersions,
  deadlineOf,
  decisionsBeforeRevision,
  extraZoneHistory,
  type NCCard,
} from '@/entities/nonconformity'
import type { Density } from '@/shared/config/widget'
import { formatMinutes } from '@/shared/lib/duration'
import { ActionButton, KeyValue, KeyValueList, StatusTag, WIDGET_FRAME_CONTEXT } from '@/shared/ui'
import EvidenceMaterial from './EvidenceMaterial.vue'
import NcSignal from './NcSignal.vue'
import NcSystemAnalysis from './NcSystemAnalysis.vue'
import NcTimeline from './NcTimeline.vue'
import RecordLine from './RecordLine.vue'

const props = withDefaults(
  defineProps<{
    card: NCCard
    /** «Сейчас» для срока решения, мс (в воспроизведении — момент воспроизведения). */
    now: number
    density?: Density
    /** Срез `view: evidence` — подробности раскрыты сразу. */
    view?: 'evidence' | 'full'
    /** Есть экран паспорта. */
    canOpenItem?: boolean
  }>(),
  { density: 'comfortable', view: 'full', canOpenItem: true },
)
const emit = defineEmits<{ 'open-item': [itemId: string] }>()
const { t, n, d, te } = useI18n()

const deadline = computed(() => deadlineOf(props.card.to_decide.decision_due_at, props.now))
const deadlineText = computed(() => {
  const dl = deadline.value
  if (!dl) return null
  const time = formatMinutes(t, dl.minutes)
  return dl.overdue ? t('ncCard.deadline.overdueBy', { time }) : t('ncCard.deadline.countdown', { time })
})
const stale = computed(() => decisionsBeforeRevision(props.card))
const op = computed(() => props.card.happened.operation ?? null)
/** Главный сигнал — первый; остальные — в подробностях. */
const signal = computed(() => props.card.evidence.signals[0] ?? null)
const materials = computed(() => props.card.evidence.signals.flatMap((s) => s.evidence_refs))
const extraHistory = computed(() => extraZoneHistory(props.card))
const current = computed(() => conclusionVersions(props.card.system_analysis.versions).current)
const investigation = computed(() => {
  const s = props.card.investigation_status
  return s ? t(INVESTIGATION_TEXT[s]) : t('empty.noDataUnknown')
})
const dec = (bp: number) => n(bpToFraction(bp), 'decimal2')
const time = (x: string) => d(new Date(x), 'dateTime')
const mode = (m: number) => (te(`decisions.automationMode.mode${m}`) ? t(`decisions.automationMode.mode${m}`) : `UNKNOWN(${m})`)

/** Заголовок: что увидели; нет сигнала (нарушение специального процесса) — чем вызвано. */
const headline = computed(() => (signal.value ? entryText(signal.value.record) : props.card.origin === 'special_process' ? t('ncCard.origin.specialProcess') : t('ncCard.number', { number: props.card.number })))

const detailsOpen = ref(props.view === 'evidence')
watch(
  () => props.card.nc_id,
  () => (detailsOpen.value = props.view === 'evidence'),
)
/** Карточка в окне записи (Д-70): заголовок окна уже несёт номер, изделие и статус, внизу — панель решения. */
const frame = inject(WIDGET_FRAME_CONTEXT, {})
const inWindow = computed(() => !!frame.plain)
</script>

<template>
  <div class="nc-card" :class="`density-${density}`" :data-status="card.status" :data-in-window="inWindow || undefined" data-testid="nc-card">
    <!-- 1. Что случилось -->
    <section class="zone hero" data-zone="what-happened">
      <header v-if="!inWindow" class="head">
        <strong class="number ant-wrap">{{ t('ncCard.number', { number: card.number }) }}</strong>
        <button v-if="canOpenItem" type="button" class="linklike ant-wrap" data-testid="open-item" @click="emit('open-item', card.item_id)">
          {{ t('common.words.item') }} {{ card.item_label }}
        </button>
        <span v-else class="ant-wrap">{{ t('common.words.item') }} {{ card.item_label }}</span>
      </header>
      <p v-if="signal" class="kicker ant-wrap">
        {{ codeText(BASIS_KIND_TEXT, signal.basis_kind, t) }} · {{ t('common.words.severity').toLowerCase() }}: {{ codeText(SEVERITY_TEXT, signal.severity, t).toLowerCase() }}
      </p>
      <h3 class="headline ant-wrap" data-testid="headline">{{ headline }}</h3>
      <p class="meta ant-wrap">
        <span v-if="op">{{ op.label }}</span>
        <span v-if="signal">{{ time(signal.record.occurred_at) }}</span>
        <span v-if="card.group_item_ids?.length">{{ t('ncCard.groupItems', { n: card.group_item_ids.length }) }}</span>
        <span v-if="deadlineText" class="deadline" :data-overdue="deadline?.overdue || undefined" data-testid="deadline">{{ deadlineText }}</span>
      </p>
      <div class="two-statuses" data-testid="two-statuses" :title="t('ncCard.twoStatuses.note')">
        <span class="ant-wrap"><span class="muted">{{ t('ncCard.twoStatuses.byItem') }}:</span> {{ codeText(NC_STATUS_TEXT, card.status, t) }}</span>
        <span class="ant-wrap" data-testid="investigation"><span class="muted">{{ t('ncCard.twoStatuses.investigation') }}:</span> {{ investigation }}</span>
        <span class="muted ant-wrap">{{ t('ncCard.twoStatuses.note') }}</span>
      </div>

      <div v-if="signal" class="facts" data-testid="signal-facts">
        <div class="fact" :title="t('hints.analyzerConfidence')">
          <span class="fact-label ant-wrap">{{ t('inspection.analyzerConfidence') }}</span>
          <span class="fact-value" data-testid="confidence">
            <template v-if="signal.analyzer_confidence_bp != null">
              {{ dec(signal.analyzer_confidence_bp) }}
              <span class="fact-note ant-wrap">{{ t('inspection.confidenceIsNotProbability') }}</span>
            </template>
            <template v-else>{{ t('empty.noDataUnknown') }}</template>
          </span>
        </div>
        <div class="fact" :title="t('hints.observationQuality')">
          <span class="fact-label ant-wrap">{{ t('inspection.observationQuality') }}</span>
          <span class="fact-value" data-testid="observation-quality">{{ signal.observation_quality_bp != null ? dec(signal.observation_quality_bp) : t('empty.noDataUnknown') }}</span>
          <span class="fact-note ant-wrap">{{ t('ncCard.facts.qualityNote') }}</span>
        </div>
      </div>

      <div v-if="materials.length" class="materials" data-testid="materials">
        <EvidenceMaterial v-for="ref in materials" :key="ref" :address="ref" />
      </div>
      <p v-else-if="signal" class="no-material muted ant-wrap" data-testid="no-material">{{ t('ncCard.material.none') }}</p>
      <p class="muted ant-wrap">{{ t('hints.signalVsNonconformity') }}</p>
    </section>

    <!-- 2. Почему это проблема -->
    <section class="zone" data-zone="requirement">
      <h3 class="ant-wrap">{{ t('ncCard.sections.whyProblem') }}</h3>
      <div v-if="card.evidence.requirement" class="requirement" data-testid="requirement">
        <span class="fact-label">{{ t('ncCard.requirement.byDesign') }}</span>
        <p class="req-text ant-wrap">{{ card.evidence.requirement.characteristic }}</p>
        <KeyValueList>
          <KeyValue v-if="card.evidence.requirement.tolerance" :label="t('ncCard.requirement.tolerance')" :value="card.evidence.requirement.tolerance" />
          <KeyValue :label="t('ncCard.requirement.designRevision')" :value="card.evidence.requirement.kd_ref ?? t('common.words.unknown')" />
        </KeyValueList>
        <p v-if="signal" class="observed ant-wrap">
          <span class="muted">{{ t('ncCard.requirement.observed') }}:</span> {{ headline }}
        </p>
      </div>
      <p v-else class="warn ant-wrap" data-testid="no-requirement">{{ t('ncCard.requirement.noRequirement') }}</p>
    </section>

    <!-- 3. Как было дело -->
    <section class="zone" data-zone="timeline">
      <h3 class="ant-wrap">{{ t('ncCard.sections.howItWent') }}</h3>
      <NcTimeline :card="card" />
      <p v-if="card.evidence.similar_count" class="similar ant-wrap" data-testid="similar-count">{{ t('widgets.ncCard.similarCount', { n: card.evidence.similar_count }) }}</p>
    </section>

    <!-- 4. Что предлагает система -->
    <section class="zone" data-zone="what-to-decide">
      <h3 class="ant-wrap">{{ t('ncCard.sections.systemSuggests') }}</h3>
      <NcSystemAnalysis :analysis="card.system_analysis" :titled="false" :show-rule="false" />
    </section>

    <!-- 5. Решения людей и итоговый статус -->
    <section class="zone" data-zone="decisions">
      <h3 class="ant-wrap">{{ t('ncCard.layers.humanDecisions') }}</h3>
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

    <!-- 6. Подробности для проверки — второй слой -->
    <section class="zone details" data-zone="details">
      <ActionButton
        size="small"
        quaternary
        data-testid="toggle-details"
        :label="detailsOpen ? t('ncCard.details.hide') : t('ncCard.details.show')"
        @click="detailsOpen = !detailsOpen"
      />
      <div v-if="detailsOpen" class="details-body" data-testid="details">
        <h4 class="ant-wrap">{{ t('ncCard.layers.sourceSignal') }}</h4>
        <NcSignal v-for="s in card.evidence.signals" :key="s.signal_id" :signal="s" />
        <template v-if="current">
          <h4 class="ant-wrap">{{ t('ncCard.layers.systemAnalysis') }}</h4>
          <p class="muted ant-wrap" data-testid="rule">
            {{ t('widgets.ncCard.rule', { ruleId: current.rule_id }) }}<template v-if="card.rule_rev">, {{ card.rule_rev }}</template> ·
            {{ t('decisions.automationMode.title') }}: {{ mode(current.automation_mode) }}
          </p>
        </template>
        <template v-if="extraHistory.length">
          <h4 class="ant-wrap">{{ t('ncCard.whatHappened.zoneHistory') }}</h4>
          <ul class="list">
            <li v-for="r in extraHistory" :key="r.event_id"><RecordLine :record="r" technical /></li>
          </ul>
        </template>
      </div>
    </section>
  </div>
</template>

<style scoped>
/*
 * Раскладка — по ширине самой карточки (container query): в окне записи
 * (600–1080 px) разделы идут друг под другом; на широкой странице карточки —
 * слева «что случилось», требование и лента, справа предложение системы и решения.
 */
.nc-card {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: var(--ant-space-5);
  min-width: 0;
  font-size: var(--ant-fs-body);
  container: nc-card / inline-size;
}

.density-large {
  font-size: var(--ant-fs-lg);
}

.zone {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
  min-width: 0;
}

.zone + .zone {
  padding-top: var(--ant-space-5);
  border-top: 1px solid var(--ant-border);
}

@container nc-card (min-width: 1100px) {
  .nc-card {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    column-gap: var(--ant-space-8);
  }

  .hero,
  .details {
    grid-column: 1 / -1;
  }
}

h3,
h4 {
  margin: 0;
}

h3 {
  font-size: var(--ant-fs-title);
}

h4 {
  margin-top: var(--ant-space-2);
  color: var(--ant-text-2);
}

p {
  margin: 0;
}

.head,
.meta,
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

.kicker {
  color: var(--ant-text-2);
  font-size: var(--ant-fs-meta);
  font-weight: var(--ant-fw-bold);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.headline {
  font-size: var(--ant-fs-display);
  line-height: 1.2;
}

.meta {
  color: var(--ant-text-2);
}

.deadline {
  font-weight: var(--ant-fw-bold);
}

.deadline[data-overdue] {
  color: var(--ant-status-danger);
}

.facts {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: var(--ant-space-2);
  margin-top: var(--ant-space-1);
}

.fact {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  padding: var(--ant-space-2) var(--ant-space-3);
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-md);
  background: var(--ant-surface-subtle);
}

.fact-label {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.fact-value {
  font-size: var(--ant-fs-title);
  font-weight: var(--ant-fw-bold);
  font-variant-numeric: tabular-nums;
}

.fact-note {
  display: block;
  color: var(--ant-text-3);
  font-weight: normal;
  font-size: var(--ant-fs-xs);
}

.materials {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: var(--ant-space-3);
}

.no-material {
  padding: var(--ant-space-2) var(--ant-space-3);
  border: 1px dashed var(--ant-border-strong);
  border-radius: var(--ant-radius-md);
}

.requirement {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
  min-width: 0;
  padding: var(--ant-space-3) var(--ant-space-4);
  border-left: 3px solid var(--ant-status-danger);
  border-radius: 0 var(--ant-radius-md) var(--ant-radius-md) 0;
  background: var(--ant-surface-subtle);
}

.req-text {
  font-size: var(--ant-fs-title);
  font-weight: var(--ant-fw-bold);
}

.similar {
  padding: var(--ant-space-2) var(--ant-space-3);
  border-radius: var(--ant-radius-md);
  background: var(--ant-status-attention-soft);
  color: var(--ant-status-attention-text);
  font-weight: var(--ant-fw-bold);
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

.details-body {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
  min-width: 0;
}

.details {
  align-items: flex-start;
}

.details-body {
  align-self: stretch;
}

.muted {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.warn {
  padding: var(--ant-space-2) var(--ant-space-3);
  border-radius: var(--ant-radius-md);
  background: var(--ant-status-attention-soft);
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
