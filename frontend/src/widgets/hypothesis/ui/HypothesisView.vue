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
import { describeRecord, isOpen, recordColor, recordLabel, sortByTime, sortHypotheses, type Hypothesis, type HypothesesModel, type JournalRecordRef, type SimilarCase } from '@/entities/incident'
import { codeToKey } from '@/shared/i18n'
import { naiveSizeOf, type Density } from '@/shared/config/widget'
import { useMomentStore } from '@/shared/model/moment'
import type { CorrectiveActionView } from '@/shared/api/generated/model'
import { ActionButton } from '@/shared/ui'
import BranchActions, { type AssignInput } from './BranchActions.vue'

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
    /** Меры инцидента (обе причины); null — мер не показывать (нет инцидента). */
    actions?: readonly CorrectiveActionView[] | null
    /** Право назначить меру (`analysis.action.assign`). */
    canAssign?: boolean
    /** Ответственный за новую меру — имя из сеанса. */
    ownerName?: string | null
  }>(),
  { density: 'compact', fromFactor: null, canConfirm: true, canReject: true, canMeasure: true, busy: false, actions: null, canAssign: false, ownerName: null },
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
  /** Назначить меру по причине: направление — по ветке. */
  assign: [direction: 'prevent_occurrence' | 'improve_detection', input: AssignInput]
}>()

const { t, n, d } = useI18n()
const moment = useMomentStore()

const hypotheses = computed(() => sortHypotheses(props.model.hypotheses))
const size = computed(() => naiveSizeOf(props.density))

/**
 * Две причины значимой проблемы (FR-64): почему дефект возник и почему контроль
 * не остановил его раньше. Гипотеза без ветки — к первой.
 */
type Branch = 'why_made' | 'why_missed'
const BRANCHES: readonly Branch[] = ['why_made', 'why_missed']
const BRANCH_TITLE: Record<Branch, string> = {
  why_made: 'widgets.analysis.hypothesis.branchWhyMade',
  why_missed: 'widgets.analysis.hypothesis.branchWhyMissed',
}
const BRANCH_EMPTY: Record<Branch, string> = {
  why_made: 'empty.noHypotheses',
  why_missed: 'widgets.analysis.hypothesis.whyMissedEmpty',
}
const byBranch = computed(() => {
  const out: Record<Branch, Hypothesis[]> = { why_made: [], why_missed: [] }
  for (const h of hypotheses.value) out[h.branch === 'why_missed' ? 'why_missed' : 'why_made'].push(h)
  return out
})
/** Уверенность вывода 0…1 для шкалы; null — сервер не дал. */
/** Направление мер причины: почему возник → предотвратить появление, почему пропустили → улучшить обнаружение. */
const DIRECTION: Record<Branch, 'prevent_occurrence' | 'improve_detection'> = { why_made: 'prevent_occurrence', why_missed: 'improve_detection' }
const actionsOf = (b: Branch) => (props.actions ?? []).filter((a) => a.direction === DIRECTION[b])
const confidence = (h: Hypothesis) => (h.confidence_bp == null ? null : Math.min(Math.max(h.confidence_bp / 10_000, 0), 1))

const label = (r: JournalRecordRef) => recordLabel(r, t)

/** Что проверить следующим: из ответа сервера (next_check), иначе — подсказка измерения. */
const nextText = (h: Hypothesis) => h.next_check?.text || h.measurement_hint || null
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
  first.value = kind === 'measure' ? (nextText(h) ?? '') : ''
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
    <p v-if="fromFactor" class="meta" data-testid="from-factor">{{ t('widgets.analysis.hypothesis.fromFactor', { factor: fromFactor }) }}</p>

    <NAlert v-if="!model.conclusion_is_categorical" type="warning" :bordered="false" :show-icon="false" class="note" data-testid="not-categorical">
      {{ t('ncCard.circumstances.noCategoricalConclusion') }}
    </NAlert>
    <p v-if="model.missing_information.length" class="missing">
      <strong>{{ t('widgets.analysis.missing.title') }}:</strong>
      <span v-for="m in model.missing_information" :key="m" class="chip-text">{{ t(`widgets.analysis.missing.${codeToKey(m)}`) }}</span>
    </p>

    <section v-for="b in BRANCHES" :key="b" class="branch" :data-branch="b" data-testid="branch">
      <h4 class="branch-title">{{ t(BRANCH_TITLE[b]) }}</h4>
      <p v-if="!byBranch[b].length" class="branch-empty ant-wrap" data-testid="branch-empty">{{ t(BRANCH_EMPTY[b]) }}</p>

      <article v-for="h in byBranch[b]" :key="h.hypothesis_id" class="card" :data-status="h.status" :data-category="h.category">
        <header class="card-head">
          <p class="statement ant-wrap">{{ h.statement || category(h.category) }}</p>
          <span class="status">{{ status(h) }}</span>
        </header>
        <p v-if="h.statement" class="kind muted ant-wrap">{{ t('common.words.hypothesis') }} · {{ category(h.category) }}</p>

        <p v-if="h.category === 'incoming'" class="muted">{{ t('hints.incomingDefect') }}</p>

        <p class="support ant-wrap" data-testid="support">
          {{ t('widgets.analysis.hypothesis.supportLine', { pro: h.supporting.length, con: h.contradicting.length }) }}
        </p>
        <details class="more" data-testid="more">
          <summary>{{ t('widgets.analysis.hypothesis.moreTitle') }}</summary>
          <div v-if="confidence(h) != null" class="confidence" :title="t('hints.analyzerConfidence')" data-testid="confidence">
          <span class="meter" aria-hidden="true"><span class="meter-fill" :style="{ width: `${confidence(h)! * 100}%` }" /></span>
          <span class="muted ant-wrap">{{ t('widgets.analysis.hypothesis.confidence', { value: n(confidence(h)!, 'decimal2') }) }}</span>
        </div>
        <div class="args">
          <section class="arg" data-side="for">
            <h5>{{ t('ncCard.commonFactors.argumentsFor') }} · {{ h.supporting.length }}</h5>
            <ul v-if="h.supporting.length">
              <li v-for="r in sortByTime(h.supporting)" :key="r.event_id">
                <span class="dot" :style="{ background: recordColor(describeRecord(r).tone) }" aria-hidden="true" />
                <button type="button" class="linklike ant-wrap" @click="emit('select-record', r.event_id)">{{ time(r.occurred_at) }} · {{ label(r) }}</button>
              </li>
            </ul>
            <p v-else class="muted">{{ t('widgets.analysis.hypothesis.noArguments') }}</p>
          </section>
          <section class="arg" data-side="against">
            <h5>{{ t('ncCard.commonFactors.argumentsAgainst') }} · {{ h.contradicting.length }}</h5>
            <ul v-if="h.contradicting.length">
              <li v-for="r in sortByTime(h.contradicting)" :key="r.event_id">
                <span class="dot" :style="{ background: recordColor(describeRecord(r).tone) }" aria-hidden="true" />
                <button type="button" class="linklike ant-wrap" @click="emit('select-record', r.event_id)">{{ time(r.occurred_at) }} · {{ label(r) }}</button>
              </li>
            </ul>
            <p v-else class="muted">{{ t('widgets.analysis.hypothesis.noArguments') }}</p>
          </section>
        </div>

        <details v-if="h.history?.length" class="history" data-testid="history">
          <summary>{{ t('widgets.analysis.hypothesis.historyTitle', { n: h.history.length }) }}</summary>
          <ol>
            <li v-for="(c, i) in h.history" :key="i" class="ant-wrap">
              <span class="muted">{{ time(c.at) }}</span> · {{ c.text }}
              <template v-if="c.confidence_bp != null"> · {{ t('widgets.analysis.hypothesis.confidenceNow', { value: n(c.confidence_bp / 10_000, 'decimal2') }) }}</template>
            </li>
          </ol>
        </details>
        </details>

        <!-- Что проверить следующим: проверка, которая подтвердит или ослабит гипотезу. -->
        <div v-if="isOpen(h) && nextText(h)" class="next" data-testid="next-check">
          <p class="next-title">{{ t('widgets.analysis.hypothesis.nextCheck') }}</p>
          <p class="next-text ant-wrap">{{ nextText(h) }}</p>
          <p v-if="h.next_check?.could_exclude" class="next-gain ant-wrap" data-testid="next-gain">
            {{ t('widgets.analysis.hypothesis.couldExclude', { n: h.next_check.could_exclude, of: h.next_check.scope_size }) }}
          </p>
          <p v-if="h.next_check?.unlocks_text" class="next-unlocks ant-wrap">{{ h.next_check.unlocks_text }}</p>
          <ActionButton overflow="wrap" :size="size" type="primary" :disabled="!canMeasure || busy || moment.isReplay" data-testid="request-measurement" @click="openForm(h, 'measure')" :label="t('widgets.analysis.hypothesis.requestCheck')" />
        </div>


        <footer v-if="isOpen(h)" class="card-actions">
          <ActionButton overflow="wrap" :size="size" type="primary" secondary :disabled="!canConfirm || busy || moment.isReplay" data-testid="confirm" @click="openForm(h, 'confirm')" :label="t('decisions.cause.confirmCause')" />
          <ActionButton overflow="wrap" :size="size" :disabled="!canReject || busy || moment.isReplay" data-testid="reject" @click="openForm(h, 'reject')" :label="t('decisions.cause.rejectHypothesis')" />
          <ActionButton
            v-if="!nextText(h)"
            overflow="wrap"
            :size="size"
            :disabled="!canMeasure || busy || moment.isReplay"
            data-testid="request-measurement"
            @click="openForm(h, 'measure')"
            :label="t('decisions.cause.requestMeasurement', { what: category(h.category) })"
          />
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

      <BranchActions
        v-if="actions"
        :direction="DIRECTION[b]"
        :actions="actionsOf(b)"
        :can-assign="canAssign"
        :busy="busy"
        :owner-name="ownerName"
        :density="density"
        @assign="(input) => emit('assign', DIRECTION[b], input)"
      />
    </section>

    <section class="similar" data-testid="similar-cases">
      <h4>{{ t('ncCard.similarCases.title') }}</h4>
      <ul v-if="model.similar_cases.length">
        <li v-for="c in model.similar_cases" :key="c.nc_id">
          <button type="button" class="linklike ant-wrap" @click="emit('open-case', c)">{{ caseLine(c) }}</button>
        </li>
      </ul>
      <p v-else class="muted">{{ t('empty.noSimilarCases') }}</p>
    </section>
    <p class="meta" :title="t('hints.signalVsNonconformity')">{{ t('ncCard.hypotheses.version', { version: model.version }) }}</p>
  </div>
</template>

<style scoped>
.hypotheses {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-3);
  min-width: 0;
  font-size: var(--ant-fs-body);
}

.density-large {
  font-size: var(--ant-fs-lg);
}

p {
  margin: 0;
}

.meta,
.muted {
  color: var(--ant-text-3);
}

.meta {
  font-size: var(--ant-fs-meta);
}

.note {
  padding: var(--ant-space-2) var(--ant-space-3);
}

.missing {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-2);
  align-items: baseline;
}

.chip-text {
  padding: 1px var(--ant-space-2);
  border-radius: var(--ant-radius-pill);
  background: var(--ant-surface-subtle);
}

/* Две причины — две секции. */
.branch {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
  min-width: 0;
}

.branch-title,
.similar h4 {
  margin: 0;
  font-size: 1em;
}

.branch-empty {
  padding: var(--ant-space-2) var(--ant-space-3);
  border: 1px dashed var(--ant-status-attention);
  border-radius: var(--ant-radius-md);
  background: var(--ant-status-attention-soft);
  color: var(--ant-status-attention-text);
}

.card {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
  min-width: 0;
  padding: var(--ant-space-3);
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-md);
  background: var(--ant-surface);
}

.card[data-status='rejected'] {
  opacity: 0.6;
}

.card[data-status='confirmed'] {
  border-color: var(--ant-status-success);
}

.card-head {
  display: flex;
  gap: var(--ant-space-2);
  align-items: flex-start;
  justify-content: space-between;
  min-width: 0;
}

.statement {
  min-width: 0;
  font-size: var(--ant-fs-title);
  font-weight: var(--ant-fw-bold);
}

.status {
  flex: none;
  padding: 0 var(--ant-space-2);
  border-radius: var(--ant-radius-pill);
  background: var(--ant-surface-subtle);
  font-size: var(--ant-fs-xs);
}

.kind {
  font-size: var(--ant-fs-meta);
}

.confidence {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-2);
  align-items: center;
}

.meter {
  flex: 0 1 160px;
  height: 8px;
  overflow: hidden;
  border-radius: var(--ant-radius-pill);
  background: var(--ant-surface-subtle);
}

.meter-fill {
  display: block;
  height: 100%;
  background: var(--ant-accent);
}

/* Что проверить следующим — главный шаг технолога по гипотезе. */
.next {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  align-items: flex-start;
  padding: var(--ant-space-2) var(--ant-space-3);
  border-left: 4px solid var(--ant-accent);
  border-radius: var(--ant-radius-md);
  background: var(--ant-accent-soft);
}

.next-title {
  color: var(--ant-text-2);
  font-size: var(--ant-fs-meta);
  font-weight: var(--ant-fw-bold);
}

.next-text {
  font-weight: var(--ant-fw-bold);
}

.next-gain {
  color: var(--ant-status-success-text);
  font-weight: var(--ant-fw-bold);
}

.next-unlocks {
  color: var(--ant-text-2);
  font-size: var(--ant-fs-meta);
}

.support {
  color: var(--ant-text-2);
}

.more summary {
  color: var(--ant-accent);
  font-size: var(--ant-fs-meta);
  cursor: pointer;
}

.more {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
}

.history summary {
  color: var(--ant-text-2);
  font-size: var(--ant-fs-meta);
  cursor: pointer;
}

.history ol {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin: var(--ant-space-1) 0 0;
  padding-left: var(--ant-space-5);
  font-size: var(--ant-fs-meta);
}

.args {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: var(--ant-space-3);
}

.arg h5 {
  margin: 0 0 var(--ant-space-1);
  font-size: 1em;
}

.arg[data-side='against'] h5 {
  color: var(--ant-text-2);
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
  gap: var(--ant-space-2);
  align-items: baseline;
  min-width: 0;
}

.dot {
  flex: none;
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.linklike {
  min-width: 0;
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
  gap: var(--ant-space-2);
  align-items: center;
}

.form {
  display: flex;
  flex-basis: 100%;
  flex-direction: column;
  gap: var(--ant-space-2);
  padding: var(--ant-space-2);
  border: 1px solid var(--ant-border-strong);
  border-radius: var(--ant-radius-md);
  background: var(--ant-surface-subtle);
}

.form label {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.form textarea {
  padding: var(--ant-space-1) var(--ant-space-2);
  border: 1px solid var(--ant-border-strong);
  border-radius: var(--ant-radius-sm);
  font: inherit;
  resize: vertical;
}

.form-actions {
  display: flex;
  gap: var(--ant-space-2);
}

.legal {
  flex-basis: 100%;
  font-size: var(--ant-fs-meta);
}
</style>
