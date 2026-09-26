<script setup lang="ts">
/**
 * Гипотезы причины с доводами «за» и «против» (FR-59, FR-135) и похожие случаи
 * (FR-60). Гипотеза — предположение; причину подтверждает только человек
 * кнопкой «подтвердить причину». Ошибка исполнителя — юридически значимый факт:
 * только после расследования, письменного объяснения и решения уполномоченного.
 * Уверенность вывода — не вероятность вины (NFR-UI-4).
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NAlert } from 'naive-ui'
import { describeRecord, isOpen, recordColor, sortByTime, sortHypotheses, type Hypothesis, type HypothesesModel, type JournalRecordRef, type SimilarCase } from '@/entities/incident'
import { codeToKey } from '@/shared/i18n'
import { naiveSizeOf, type Density } from '@/shared/config/widget'
import { useMomentStore } from '@/shared/model/moment'
import { ActionButton } from '@/shared/ui'

const props = withDefaults(
  defineProps<{
    model: HypothesesModel
    density?: Density
    /** Подпись общего фактора, из которого технолог вошёл в гипотезу. */
    fromFactor?: string | null
    /** Доступно ли «подтвердить причину» (право `analysis.cause.conclude`). */
    canConfirm?: boolean
    /** Доступно ли «отклонить» (право `analysis.hypothesis.reject`). */
    canReject?: boolean
    /** Доступно ли «запросить измерение» (право `analysis.measurement.request`). */
    canMeasure?: boolean
    /** Команда отправляется — формы выключены. */
    busy?: boolean
  }>(),
  { density: 'compact', fromFactor: null, canConfirm: true, canReject: true, canMeasure: true, busy: false },
)
const emit = defineEmits<{
  /** Подтвердить причину: чем проверили и основание. */
  confirm: [h: Hypothesis, input: { verification: string; reason: string }]
  /** Отклонить гипотезу с основанием. */
  reject: [h: Hypothesis, input: { reason: string }]
  /** Запросить измерение: что измерить. */
  'request-measurement': [h: Hypothesis, input: { what: string }]
  /** Показать запись-довод на дорожках. */
  'select-record': [eventId: string]
  /** Открыть похожий случай. */
  'open-case': [c: SimilarCase]
}>()

const { t, n, d } = useI18n()
const moment = useMomentStore()

const hypotheses = computed(() => sortHypotheses(props.model.hypotheses))
const size = computed(() => naiveSizeOf(props.density))

const label = (r: JournalRecordRef) => {
  const x = describeRecord(r)
  return x.key ? t(x.key, x.params) : `UNKNOWN(${x.eventType})`
}
const category = (c: string | null) => (c ? t(`statuses.causeCategory.${codeToKey(c)}`) : t('statuses.causeCategory.notEstablished'))
const status = (h: Hypothesis) => t(`statuses.hypothesis.${codeToKey(h.status)}`)
const time = (x: string) => d(new Date(x), 'dateTime')

/** Открытая форма решения по гипотезе: у каждого решения — обязательные поля. */
type FormKind = 'confirm' | 'reject' | 'measure'
const form = ref<{ id: string; kind: FormKind } | null>(null)
const first = ref('')
const second = ref('')

function openForm(h: Hypothesis, kind: FormKind): void {
  form.value = { id: h.hypothesis_id, kind }
  first.value = kind === 'measure' ? (h.measurement_hint ?? '') : ''
  second.value = ''
}

const formReady = computed(() => {
  if (!form.value) return false
  if (form.value.kind === 'confirm') return first.value.trim() !== '' && second.value.trim() !== ''
  return first.value.trim() !== ''
})

function submit(h: Hypothesis): void {
  if (!form.value || !formReady.value) return
  if (form.value.kind === 'confirm') emit('confirm', h, { verification: first.value.trim(), reason: second.value.trim() })
  else if (form.value.kind === 'reject') emit('reject', h, { reason: first.value.trim() })
  else emit('request-measurement', h, { what: first.value.trim() })
  form.value = null
}

const FIRST_LABEL: Record<FormKind, string> = {
  confirm: 'decisions.cause.verifiedBy',
  reject: 'common.words.basis',
  measure: 'widgets.analysis.hypothesis.measureWhat',
}

function caseLine(c: SimilarCase): string {
  return t('ncCard.similarCases.line', {
    case: c.number,
    causeStatus: c.cause_confirmed ? t('widgets.analysis.hypothesis.similarCauseConfirmed') : t('widgets.analysis.hypothesis.similarCauseHypothesis'),
    cause: category(c.cause_category),
    measure: c.measure ?? t('widgets.analysis.hypothesis.noMeasure'),
    result: c.result ? t(`statuses.correctiveAction.${codeToKey(c.result)}`) : t('common.words.unknown'),
  })
}
</script>

<template>
  <div class="hypotheses" :class="`density-${density}`" data-testid="hypotheses">
    <p class="meta" :title="t('hints.signalVsNonconformity')">{{ t('ncCard.hypotheses.version', { version: model.version }) }}</p>
    <p v-if="fromFactor" class="meta" data-testid="from-factor">{{ t('widgets.analysis.hypothesis.fromFactor', { factor: fromFactor }) }}</p>

    <NAlert v-if="!model.conclusion_is_categorical" type="warning" :bordered="false" :show-icon="false" class="note" data-testid="not-categorical">
      {{ t('ncCard.circumstances.noCategoricalConclusion') }}
    </NAlert>
    <p v-if="model.missing_information.length" class="missing">
      <strong>{{ t('widgets.analysis.missing.title') }}:</strong>
      <span v-for="m in model.missing_information" :key="m" class="chip-text">{{ t(`widgets.analysis.missing.${codeToKey(m)}`) }}</span>
    </p>

    <p v-if="!hypotheses.length" class="muted">{{ t('empty.noHypotheses') }}</p>
    <article v-for="h in hypotheses" :key="h.hypothesis_id" class="card" :data-status="h.status" :data-category="h.category">
      <header class="card-head">
        <span class="category">{{ t('common.words.hypothesis') }}: {{ category(h.category) }}</span>
        <span class="status">{{ status(h) }}</span>
        <span v-if="h.branch" class="branch">{{ t(h.branch === 'why_made' ? 'ncCard.investigation.whyMade' : 'ncCard.investigation.whyMissed') }}</span>
      </header>
      <p v-if="h.statement" class="statement">{{ h.statement }}</p>
      <p v-if="h.confidence_bp != null" class="muted" :title="t('hints.analyzerConfidence')">
        {{ t('widgets.analysis.hypothesis.confidence', { value: n(h.confidence_bp / 10_000, 'decimal2') }) }}
      </p>
      <p v-if="h.category === 'incoming'" class="muted">{{ t('hints.incomingDefect') }}</p>

      <div class="args">
        <section class="arg" data-side="for">
          <h4>{{ t('ncCard.commonFactors.argumentsFor') }}</h4>
          <ul v-if="h.supporting.length">
            <li v-for="r in sortByTime(h.supporting)" :key="r.event_id">
              <span class="dot" :style="{ background: recordColor(describeRecord(r).tone) }" aria-hidden="true" />
              <button type="button" class="linklike" @click="emit('select-record', r.event_id)">{{ time(r.occurred_at) }} · {{ label(r) }}</button>
            </li>
          </ul>
          <p v-else class="muted">{{ t('widgets.analysis.hypothesis.noArguments') }}</p>
        </section>
        <section class="arg" data-side="against">
          <h4>{{ t('ncCard.commonFactors.argumentsAgainst') }}</h4>
          <ul v-if="h.contradicting.length">
            <li v-for="r in sortByTime(h.contradicting)" :key="r.event_id">
              <span class="dot" :style="{ background: recordColor(describeRecord(r).tone) }" aria-hidden="true" />
              <button type="button" class="linklike" @click="emit('select-record', r.event_id)">{{ time(r.occurred_at) }} · {{ label(r) }}</button>
            </li>
          </ul>
          <p v-else class="muted">{{ t('widgets.analysis.hypothesis.noArguments') }}</p>
        </section>
      </div>

      <footer v-if="isOpen(h)" class="card-actions">
        <ActionButton overflow="wrap" :size="size" type="primary" secondary :disabled="!canConfirm || busy || moment.isReplay" data-testid="confirm" @click="openForm(h, 'confirm')" :label="t('decisions.cause.confirmCause')" />
        <ActionButton overflow="wrap" :size="size" :disabled="!canReject || busy || moment.isReplay" data-testid="reject" @click="openForm(h, 'reject')" :label="t('decisions.cause.rejectHypothesis')" />
        <ActionButton overflow="wrap" :size="size" :disabled="!canMeasure || busy || moment.isReplay" data-testid="request-measurement" @click="openForm(h, 'measure')" :label="t('decisions.cause.requestMeasurement', { what: h.measurement_hint ?? category(h.category) })" />
        <p v-if="h.category === 'performer'" class="muted legal" data-testid="performer-note">{{ t('decisions.cause.performerErrorPrerequisites') }}</p>

        <form v-if="form?.id === h.hypothesis_id" class="form" :data-form="form.kind" @submit.prevent="submit(h)">
          <label>
            <span>{{ t(FIRST_LABEL[form.kind]) }}</span>
            <textarea v-model="first" rows="2" required data-testid="form-first" />
          </label>
          <label v-if="form.kind === 'confirm'">
            <span>{{ t('common.words.basis') }}</span>
            <textarea v-model="second" rows="2" required data-testid="form-second" />
          </label>
          <div class="form-actions">
            <ActionButton overflow="wrap" :size="size" type="primary" attr-type="submit" :disabled="!formReady || busy || moment.isReplay" data-testid="form-submit" :label="t('common.actions.send')" />
            <ActionButton overflow="wrap" :size="size" quaternary data-testid="form-cancel" @click="form = null" :label="t('common.actions.cancel')" />
          </div>
        </form>
      </footer>
    </article>

    <section class="similar" data-testid="similar-cases">
      <h4>{{ t('ncCard.similarCases.title') }}</h4>
      <ul v-if="model.similar_cases.length">
        <li v-for="c in model.similar_cases" :key="c.nc_id">
          <button type="button" class="linklike" @click="emit('open-case', c)">{{ caseLine(c) }}</button>
        </li>
      </ul>
      <p v-else class="muted">{{ t('empty.noSimilarCases') }}</p>
    </section>
  </div>
</template>

<style scoped>
.hypotheses {
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: var(--ant-fs-body);
}

.density-large {
  font-size: var(--ant-fs-lg);
}

.meta,
.muted,
.missing,
.statement {
  margin: 0;
}

.meta,
.muted {
  color: var(--ant-text-3);
}

.note {
  padding: 6px 10px;
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

.card {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 8px 10px;
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-md);
}

.card[data-status='rejected'] {
  opacity: 0.6;
}

.card[data-status='confirmed'] {
  border-color: var(--ant-text);
}

.card-head {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: baseline;
}

.category {
  font-weight: var(--ant-fw-bold);
}

.status,
.branch {
  padding: 0 6px;
  border-radius: var(--ant-radius-lg);
  background: var(--ant-n-100);
  font-size: var(--ant-fs-xs);
}

.args {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}

.arg h4,
.similar h4 {
  margin: 0 0 4px;
  font-size: 1em;
}

.arg[data-side='against'] h4 {
  color: var(--ant-text-3);
}

.arg ul,
.similar ul {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.arg li {
  display: flex;
  gap: 6px;
  align-items: baseline;
}

.dot {
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

.card-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}

.form {
  display: flex;
  flex-basis: 100%;
  flex-direction: column;
  gap: 6px;
  padding: 8px;
  border: 1px solid var(--ant-n-300);
  border-radius: var(--ant-radius-md);
  background: var(--ant-surface-subtle);
}

.form label {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.form textarea {
  font: inherit;
  padding: 4px 6px;
  border: 1px solid var(--ant-n-300);
  border-radius: var(--ant-radius-sm);
  resize: vertical;
}

.form-actions {
  display: flex;
  gap: 6px;
}

.legal {
  flex-basis: 100%;
  font-size: var(--ant-fs-meta);
}
</style>
