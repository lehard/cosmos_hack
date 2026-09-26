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
import { NAlert, NButton, NInput, NSelect } from 'naive-ui'
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
const reasonLabel = computed(() => (draft.action === 'reject_signal' ? t('decisions.signal.rejectReasonLabel') : t('widgets.decisions.reasonLabel')))
</script>

<template>
  <div class="panel" :class="`density-${density}`" data-testid="decision-panel">
    <p v-if="!canAct" class="muted" data-testid="replay-note">{{ t('common.modes.replayReadOnly') }}</p>
    <p v-if="!actions.length" class="muted" data-testid="nothing">{{ t('widgets.decisions.nothingToDecide') }}</p>

    <section v-if="signalActions.length" class="group" data-group="signal">
      <NButton
        v-for="a in signalActions"
        :key="a"
        :size="size"
        :type="chosen.action === a ? 'primary' : 'default'"
        :secondary="chosen.action !== a"
        :disabled="!canAct || busy"
        :data-action="a"
        @click="choose(a)"
      >
        {{ t(DECISION_ACTIONS[a].labelKey, { nextStep: t('widgets.decisions.nextStep'), method: '' }) }}
      </NButton>
    </section>

    <section v-if="dispositionOpen" class="group" data-group="disposition">
      <h4>{{ t('decisions.disposition.title') }}</h4>
      <NButton
        v-for="d in DISPOSITIONS"
        :key="d"
        :size="size"
        :type="chosen.action === 'disposition' && draft.disposition === d ? 'primary' : 'default'"
        :secondary="!(chosen.action === 'disposition' && draft.disposition === d)"
        :disabled="!canAct || busy"
        :data-disposition="d"
        @click="choose('disposition', d)"
      >
        {{ t(DISPOSITION_LABEL[d], { operation: reworkOperation }) }}
      </NButton>
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
          <NButton v-if="!usable.length" :size="size" secondary :disabled="!canAct || busy" data-testid="create-concession" @click="emit('create-concession')">
            {{ t('decisions.concession.create') }}
          </NButton>
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

      <NButton
        v-if="permitted !== false"
        type="primary"
        :size="size"
        :disabled="!canSign"
        :loading="busy"
        data-testid="sign"
        @click="emit('sign', { ...draft })"
      >
        {{ t('decisions.signature.signClosingDecision') }}
      </NButton>
    </section>

    <NAlert v-if="error" type="error" :bordered="false" :show-icon="false" data-testid="command-error">{{ problemText(error) }}</NAlert>
    <p v-if="receipt" class="receipt" data-testid="receipt">
      {{ t('widgets.decisions.recorded', { seq: receipt.seq }) }}<template v-if="receipt.ca_ref"> · {{ receipt.ca_ref }}</template>
    </p>
  </div>
</template>

<style scoped>
.panel {
  display: flex;
  flex-direction: column;
  gap: 10px;
  font-size: 13px;
}

.density-large {
  font-size: 16px;
}

.group {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}

.group h4 {
  flex-basis: 100%;
  margin: 0;
}

.form {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 8px 10px;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.muted,
.warn,
.receipt {
  margin: 0;
  font-size: 12px;
}

.muted {
  color: #6b7280;
}

.warn {
  color: #b45309;
}

.problems {
  margin: 0;
  padding-left: 16px;
  color: #b45309;
  font-size: 12px;
}

.receipt {
  color: #2e9e5b;
}
</style>
