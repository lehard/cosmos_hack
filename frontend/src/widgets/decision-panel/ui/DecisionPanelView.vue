<script setup lang="ts">
/**
 * Панель решений контролёра (FR-52, FR-53, FR-54, FR-146; PRD §3a).
 *
 * - Кнопки названы действием и направлением: «Подтвердить несоответствие —
 *   изолировать до решения», «Отклонить сигнал — изделие продолжает маршрут»…
 *   Какие решения допустимы по состоянию, сообщает сервер (`to_decide.decisions`).
 * - Причина обязательна: отклонение без причины невозможно (FR-52); в контракте
 *   основание обязательно и у остальных решений.
 * - Ремонт и «как есть» — только с выбором действующего разрешения на отклонение
 *   (FR-53, FR-54); итог — «годно по разрешению на отклонение», не «годно».
 * - «Почему вы можете / не можете»; нет полномочий — «Запросить решение» вместо
 *   подписи (FR-146). Подпись — окном уровня 2 (виджет открывает его по `sign`).
 */
import { computed, reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NAlert, NInput, NSelect } from 'naive-ui'
import {
  DECISION_ACTIONS,
  DISPOSITIONS,
  DISPOSITION_LABEL,
  METHOD_TEXT,
  availableActions,
  draftProblems,
  needsConcession,
  usableConcessions,
  type Concession,
  type DecisionAction,
  type DecisionDraft,
  type Disposition,
  type InspectionMethod,
  type NCCard,
  type Receipt,
} from '@/entities/nonconformity'
import { AuthorityNote, type Explanation } from '@/features/decision-authority'
import { naiveSizeOf, type Density } from '@/shared/config/widget'
import { useProblemText } from '@/shared/i18n/problem'
import { ActionButton, StatusTag } from '@/shared/ui'

const props = withDefaults(
  defineProps<{
    card: NCCard
    /** Разрешённые операции (x-ant-action id) по изделию и несоответствию; null — список прав не пришёл. */
    allowed: ReadonlySet<string> | null
    /** Разрешения на отклонение по изделию. */
    concessions?: Concession[]
    /** Действия доступны (не воспроизведение). */
    canAct?: boolean
    /** Идёт отправка. */
    busy?: boolean
    /** Ошибка последней команды или запроса решения. */
    error?: unknown
    /** Квитанция последней записанной команды. */
    receipt?: Receipt | null
    /** Объяснение прав по выбранному действию. */
    explanation?: Explanation | null
    explanationOpen?: boolean
    explanationLoading?: boolean
    explanationError?: unknown
    density?: Density
  }>(),
  {
    concessions: () => [],
    canAct: true,
    busy: false,
    error: undefined,
    receipt: null,
    explanation: null,
    explanationOpen: false,
    explanationLoading: false,
    explanationError: undefined,
    density: 'comfortable',
  },
)
const emit = defineEmits<{
  /** Черновик проверен — открыть окно подписи. */
  sign: [draft: DecisionDraft]
  /** Раскрыть или свернуть «почему вы можете / не можете» по операции. */
  explain: [operation: string]
  /** Нет полномочий — запросить решение у уполномоченного. */
  'request-decision': [operation: string]
  /** Оформить новое разрешение на отклонение — отправить на подписи. */
  'create-concession': []
}>()

const { t } = useI18n()
const problemText = useProblemText()
const size = computed(() => naiveSizeOf(props.density))

const actions = computed(() => availableActions(props.card))
const signalActions = computed(() => actions.value.filter((a) => a !== 'disposition'))
const dispositionOpen = computed(() => actions.value.includes('disposition'))

const draft = reactive<DecisionDraft>({ action: 'confirm_nc', disposition: null, concession_id: null, method: null, reason: '', claim_basis: '' })
const chosen = reactive<{ action: DecisionAction | null }>({ action: null })

// Новая карточка — черновик заново: решение по одной карточке не переносится на другую.
watch(
  () => props.card.nc_id,
  () => {
    chosen.action = null
    Object.assign(draft, { action: 'confirm_nc', disposition: null, concession_id: null, method: null, reason: '', claim_basis: '' })
  },
)

function choose(action: DecisionAction, disposition: Disposition | null = null): void {
  chosen.action = action
  draft.action = action
  draft.disposition = disposition
  draft.concession_id = null
}

const operation = computed(() => (chosen.action ? DECISION_ACTIONS[chosen.action].operation : null))
/** Есть ли право на выбранное действие; null — неизвестно (список прав не пришёл). */
const permitted = computed<boolean | null>(() => (operation.value && props.allowed ? props.allowed.has(operation.value) : null))
const usable = computed(() => usableConcessions(props.concessions, draft.disposition))
const problems = computed(() => (chosen.action ? draftProblems(draft, props.concessions) : []))
const canSign = computed(() => props.canAct && !props.busy && permitted.value === true && !problems.value.length)

const methodOptions = computed(() => (Object.keys(METHOD_TEXT) as InspectionMethod[]).map((m) => ({ label: t(METHOD_TEXT[m]), value: m })))
const concessionOptions = computed(() =>
  usable.value.map((c) => ({
    label: `${c.title} · ${t('decisions.concession.usage', { used: c.used, limit: c.limit ?? '∞' })}`,
    value: c.concession_id,
  })),
)
const actionLabels = computed(() => Object.fromEntries(Object.values(DECISION_ACTIONS).map((d) => [d.operation, t(d.labelKey, { method: '', nextStep: '', operation: '' })])))
/** Операция, на которую вернуть при переделке, — операция несоответствия. */
const reworkOperation = computed(() => props.card.happened.operation?.label ?? t('widgets.decisions.reworkOperation'))
/**
 * Этап решения по несоответствию (UI-26): 1 — что это (подтвердить, отклонить,
 * доп. контроль), 2 — что делать с изделием, 3 — исполнение и закрытие. Этап —
 * по статусу карточки «по изделию», кнопки — по допустимым решениям сервера.
 */
const STAGES = ['identify', 'disposition', 'execution'] as const
const stage = computed(() => {
  const s = props.card.status
  return s === 'draft' ? 0 : s === 'confirmed' ? 1 : 2
})
/** Что означает вариант и чем закончится (UI-26) — тексты, а не вычисления: условия проверяет сервер (гарды). */
const DISPOSITION_MEANING: Record<Disposition, string> = {
  rework: 'widgets.decisions.option.reworkMeaning',
  repair: 'widgets.decisions.option.repairMeaning',
  use_as_is: 'widgets.decisions.option.useAsIsMeaning',
  scrap: 'widgets.decisions.option.scrapMeaning',
  return_to_supplier: 'widgets.decisions.option.returnMeaning',
}
const DISPOSITION_OUTCOME: Record<Disposition, string> = {
  rework: 'widgets.decisions.option.reworkOutcome',
  repair: 'widgets.decisions.option.concessionOutcome',
  use_as_is: 'widgets.decisions.option.concessionOutcome',
  scrap: 'widgets.decisions.option.scrapOutcome',
  return_to_supplier: 'widgets.decisions.option.returnOutcome',
}
const reasonLabel = computed(() => (draft.action === 'reject_signal' ? t('decisions.signal.rejectReasonLabel') : t('widgets.decisions.reasonLabel')))
</script>

<template>
  <div class="panel" :class="`density-${density}`" data-testid="decision-panel">
    <p v-if="!canAct" class="muted" data-testid="replay-note">{{ t('common.modes.replayReadOnly') }}</p>
    <ol class="stages" data-testid="stages" :aria-label="t('widgets.decisions.stage.title')">
      <li v-for="(st, i) in STAGES" :key="st" class="stage" :data-stage="st" :data-state="i < stage ? 'done' : i === stage ? 'current' : 'next'">
        <span class="stage-no">{{ i + 1 }}</span>
        <span class="ant-ellipsis">{{ t(`widgets.decisions.stage.${st}`) }}</span>
      </li>
    </ol>
    <p v-if="!actions.length" class="muted" data-testid="nothing">{{ t('widgets.decisions.nothingToDecide') }}</p>

    <section v-if="signalActions.length" class="group" data-group="signal">
      <h4 v-if="stage > 0">{{ t('widgets.decisions.containmentTitle') }}</h4>
      <ActionButton overflow="wrap"
        v-for="a in signalActions"
        :key="a"
        :size="size"
        :type="chosen.action === a ? 'primary' : 'default'"
        :secondary="chosen.action !== a"
        :disabled="!canAct || busy"
        :data-action="a"
        @click="choose(a)"
        :label="t(DECISION_ACTIONS[a].labelKey, { nextStep: t('widgets.decisions.nextStep'), method: '' })"
      />
    </section>

    <section v-if="dispositionOpen" class="options" data-group="disposition">
      <h4>{{ t('decisions.disposition.title') }}</h4>
      <button
        v-for="d in DISPOSITIONS"
        :key="d"
        type="button"
        class="option"
        :data-disposition="d"
        :aria-pressed="chosen.action === 'disposition' && draft.disposition === d"
        :disabled="!canAct || busy"
        @click="choose('disposition', d)"
      >
        <span class="option-title ant-wrap">{{ t(DISPOSITION_LABEL[d], { operation: reworkOperation }) }}</span>
        <span class="option-meaning ant-wrap">{{ t(DISPOSITION_MEANING[d]) }}</span>
        <span class="option-terms ant-wrap" :data-concession="needsConcession(d) || undefined">
          {{ needsConcession(d) ? t('widgets.decisions.option.needsConcession') : t('widgets.decisions.option.noConcession') }} · {{ t(DISPOSITION_OUTCOME[d]) }}
        </span>
      </button>
      <p class="muted">{{ t('decisions.disposition.decisionDoesNotWaitForCause') }}</p>
    </section>

    <section v-if="chosen.action" class="form" :data-chosen="chosen.action" data-testid="decision-form">
      <p v-if="draft.action === 'reject_signal'" class="muted">{{ t('decisions.signal.originalSignalKept') }}</p>
      <template v-if="draft.action === 'request_recheck'">
        <label class="field">
          <span>{{ t('inspection.method.title') }}</span>
          <NSelect v-model:value="draft.method" :options="methodOptions" :size="size" data-testid="method" />
        </label>
        <p class="muted">{{ t('decisions.signal.requestRecheckWaiting') }}</p>
      </template>

      <template v-if="draft.action === 'disposition'">
        <p v-if="draft.disposition === 'rework' || draft.disposition === 'repair'" class="muted">{{ t('decisions.disposition.afterReworkNote') }}</p>
        <p v-if="draft.disposition === 'repair'" class="muted">{{ t('hints.reworkVsRepair') }}</p>
        <template v-if="needsConcession(draft.disposition)">
          <label class="field">
            <span>{{ t('decisions.disposition.selectConcession') }}</span>
            <NSelect v-if="usable.length" v-model:value="draft.concession_id" :options="concessionOptions" :size="size" data-testid="concession" />
          </label>
          <p v-if="!usable.length" class="warn" data-testid="no-concession">{{ t('decisions.disposition.noActiveConcession') }}</p>
          <ActionButton overflow="wrap" v-if="!usable.length" :size="size" secondary :disabled="!canAct || busy" data-testid="create-concession" @click="emit('create-concession')" :label="t('decisions.concession.create')" />
          <p class="muted" data-testid="concession-result">{{ t('decisions.concession.resultStatus') }}</p>
          <p class="muted">{{ t('decisions.concession.notDeviationPermit') }}</p>
        </template>
        <template v-if="draft.disposition === 'return_to_supplier'">
          <p class="muted" data-testid="return-only-unprocessed">{{ t('decisions.disposition.returnOnlyUnprocessed') }}</p>
          <label class="field">
            <span>{{ t('widgets.decisions.claimBasis') }}</span>
            <NInput v-model:value="draft.claim_basis" type="textarea" :autosize="{ minRows: 1, maxRows: 3 }" :size="size" />
          </label>
        </template>
        <p v-if="draft.disposition === 'scrap'" class="muted">{{ t('documents.scrapActNote') }}</p>
      </template>

      <label class="field">
        <span>{{ reasonLabel }}</span>
        <NInput
          v-model:value="draft.reason"
          type="textarea"
          :autosize="{ minRows: 2, maxRows: 5 }"
          :placeholder="draft.action === 'reject_signal' ? t('decisions.signal.rejectReasonPlaceholder') : ''"
          :size="size"
          :disabled="!canAct || busy"
          data-testid="reason"
        />
      </label>

      <ul v-if="problems.length" class="problems" data-testid="problems">
        <li v-for="p in problems" :key="p">{{ t(p, { decision: '', who: '' }) }}</li>
      </ul>

      <AuthorityNote
        :allowed="permitted"
        :explanation="explanation"
        :open="explanationOpen"
        :loading="explanationLoading"
        :error="explanationError"
        :can-request="true"
        :action-labels="actionLabels"
        :disabled="!canAct || busy"
        :size="size"
        @toggle="operation && emit('explain', operation)"
        @request-decision="operation && emit('request-decision', operation)"
      />

      <ActionButton overflow="wrap"
        v-if="permitted !== false"
        type="primary"
        :size="size"
        :disabled="!canSign"
        :loading="busy"
        data-testid="sign"
        @click="emit('sign', { ...draft })"
        :label="t('decisions.signature.signClosingDecision')"
      />
    </section>

    <NAlert v-if="error" type="error" :bordered="false" :show-icon="false" data-testid="command-error">{{ problemText(error) }}</NAlert>
    <section v-if="receipt" class="receipt" data-testid="receipt">
      <p class="receipt-title">{{ t('widgets.decisions.recordedTitle') }}</p>
      <p class="receipt-now" data-testid="receipt-now">
        <span class="muted">{{ t('widgets.decisions.recordedNow') }}:</span>
        <StatusTag axis="position" :code="card.axes.position" />
        <StatusTag axis="containment" :code="card.axes.containment" />
      </p>
      <p class="muted" data-testid="receipt-ref">
        {{ t('widgets.decisions.recorded', { seq: receipt.seq }) }}<template v-if="receipt.ca_ref"> · {{ receipt.ca_ref }}</template>
      </p>
    </section>
  </div>
</template>

<style scoped>
.panel {
  display: flex;
  flex-direction: column;
  gap: 10px;
  font-size: var(--ant-fs-body);
}

.density-large {
  font-size: var(--ant-fs-lg);
}

.group {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}

.group h4,
.options h4 {
  flex-basis: 100%;
  margin: 0;
}

/* Варианты решения по изделию — карточки: название, смысл, условия и итог. */
.options {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: var(--ant-space-2);
}

.options h4,
.options > .muted {
  grid-column: 1 / -1;
}

.option {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  padding: var(--ant-space-2) var(--ant-space-3);
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-md);
  background: var(--ant-surface);
  color: var(--ant-text);
  font: inherit;
  text-align: start;
  cursor: pointer;
}

.option:hover:not(:disabled) {
  border-color: var(--ant-border-strong);
  background: var(--ant-surface-hover);
}

.option[aria-pressed='true'] {
  border-color: var(--ant-accent);
  background: var(--ant-accent-soft);
  box-shadow: inset 0 0 0 1px var(--ant-accent);
}

.option:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

.option-title {
  font-weight: var(--ant-fw-bold);
}

.option-meaning {
  color: var(--ant-text-2);
  font-size: var(--ant-fs-meta);
}

.option-terms {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-xs);
}

.option-terms[data-concession] {
  color: var(--ant-status-attention-text);
}

.form {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 8px 10px;
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-md);
}

.field {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.muted,
.warn,
.receipt p {
  margin: 0;
  font-size: var(--ant-fs-meta);
}

/* Этапы: номер в кружке, текущий — акцентом, пройденные — приглушены. */
.stages {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-1) var(--ant-space-4);
  margin: 0;
  padding: 0;
  list-style: none;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.stage {
  display: flex;
  gap: var(--ant-space-1);
  align-items: center;
  min-width: 0;
}

.stage-no {
  display: inline-flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border: 1px solid var(--ant-border-strong);
  border-radius: 50%;
  font-size: var(--ant-fs-xs);
}

.stage[data-state='current'] {
  color: var(--ant-text);
  font-weight: var(--ant-fw-bold);
}

.stage[data-state='current'] .stage-no {
  border-color: var(--ant-accent);
  background: var(--ant-accent);
  color: var(--ant-surface);
}

.stage[data-state='done'] .stage-no {
  border-color: var(--ant-status-success);
  color: var(--ant-status-success);
}

.muted {
  color: var(--ant-text-3);
}

.warn {
  color: var(--ant-status-attention-text);
}

.problems {
  margin: 0;
  padding-left: 16px;
  color: var(--ant-status-attention-text);
  font-size: var(--ant-fs-meta);
}

.receipt {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  padding: var(--ant-space-2) var(--ant-space-3);
  border-left: 3px solid var(--ant-status-success);
  border-radius: 0 var(--ant-radius-md) var(--ant-radius-md) 0;
  background: var(--ant-status-success-soft);
}

.receipt-title {
  color: var(--ant-status-success-text);
  font-weight: var(--ant-fw-bold);
}

.receipt-now {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-1) var(--ant-space-2);
  align-items: center;
}
</style>
