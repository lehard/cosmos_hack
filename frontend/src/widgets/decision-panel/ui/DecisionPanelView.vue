<script setup lang="ts">
/**
 * Панель решений контролёра (FR-52, FR-53, FR-54, FR-146; PRD §3a).
 *
 * - Кнопки названы действием и направлением: «Подтвердить несоответствие»
 *   (изолирует отдельная кнопка), «Отклонить сигнал — изделие продолжает маршрут»…
 *   Какие решения допустимы по состоянию, сообщает сервер (`to_decide.decisions`).
 * - Причина обязательна: отклонение без причины невозможно (FR-52); в контракте
 *   основание обязательно и у остальных решений.
 * - Ремонт и «как есть» — только с выбором действующего разрешения на отклонение
 *   (FR-53, FR-54); итог — «годно по разрешению на отклонение», не «годно».
 * - «Почему вы можете / не можете»; нет полномочий — «Запросить решение» вместо
 *   подписи (FR-146). Подпись — окном уровня 2 (виджет открывает его по `sign`).
 */
import { computed, reactive, ref, watch } from 'vue'
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
  techOpen.value = false
  chosen.action = action
  draft.action = action
  draft.disposition = disposition
  draft.concession_id = null
}

const operation = computed(() => (chosen.action ? DECISION_ACTIONS[chosen.action].operation : null))
/**
 * Действие сервера (п. 15): доступность для вошедшего, почему, деловые и
 * технические последствия. Только их показываем — интерфейс ничего не досчитывает.
 */
const serverActions = computed(() => props.card.to_decide.actions ?? [])
function serverActionFor(action: DecisionAction, disposition: Disposition | null = null) {
  const op = DECISION_ACTIONS[action].operation
  return serverActions.value.find((a) => a.operation === op && (action !== 'disposition' || a.disposition === disposition)) ?? null
}
const chosenServer = computed(() => (chosen.action ? serverActionFor(chosen.action, draft.disposition) : null))
const techOpen = ref(false)
/** Кому передано исполнение решения по изделию — от сервера (стык с мастером). */
const handoff = computed(() => props.card.handoff ?? null)
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
/** Вариант коротко и его условие (UI-52): что можно сделать и что требует разрешения. */
const DISPOSITION_SHORT: Record<Disposition, string> = {
  rework: 'widgets.decisions.option.reworkShort',
  repair: 'widgets.decisions.option.repairShort',
  use_as_is: 'widgets.decisions.option.useAsIsShort',
  scrap: 'widgets.decisions.option.scrapShort',
  return_to_supplier: 'widgets.decisions.option.returnShort',
}
const DISPOSITION_TERM: Record<Disposition, string> = {
  rework: 'widgets.decisions.option.reworkTerm',
  repair: 'widgets.decisions.option.needsConcessionShort',
  use_as_is: 'widgets.decisions.option.needsConcessionShort',
  scrap: 'widgets.decisions.option.scrapTerm',
  return_to_supplier: 'widgets.decisions.option.returnTerm',
}
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
/** Состояние шага словами: пройден — что установлено, текущий — что нужно, будущий — «ещё не начато». */
function stageState(i: number): string {
  if (i < stage.value) return t(`widgets.decisions.stageDone.${STAGES[i]}`)
  if (i > stage.value) return t('widgets.decisions.stageNext')
  if (STAGES[i] === 'execution' && handoff.value) return t('widgets.decisions.handoff', { who: handoff.value.role_label, what: handoff.value.task_title, status: handoff.value.status_label })
  return t(`widgets.decisions.stageNow.${STAGES[i]}`)
}
const actionLabel = (a: DecisionAction) => t(DECISION_ACTIONS[a].labelKey, { nextStep: t('widgets.decisions.nextStep'), method: '' })
/** Коротко — до тире: «Переделка — вернуть на операцию» → «Переделка»; целиком — в подсказке. */
const head = (text: string) => text.split(' — ')[0] ?? text
/** Подсказка варианта: что это, условие и — если сервер не пускает — почему. */
function optionHint(d: Disposition): string {
  const why = serverActionFor('disposition', d)
  return [t(DISPOSITION_SHORT[d]), t(DISPOSITION_TERM[d]), why?.allowed === false ? why.why_available : null].filter(Boolean).join(' · ')
}
const reasonLabel = computed(() => (draft.action === 'reject_signal' ? t('decisions.signal.rejectReasonLabel') : t('widgets.decisions.reasonLabel')))
</script>

<template>
  <div class="panel" :class="`density-${density}`" data-testid="decision-panel">
    <p v-if="!canAct" class="muted" data-testid="replay-note">{{ t('common.modes.replayReadOnly') }}</p>
    <!-- Ход решения (UI-52): одна строка шагов; состояние шага — подсказкой, передача исполнения — строкой под шагами. -->
    <ol class="stages" data-testid="stages" :aria-label="t('widgets.decisions.stage.title')">
      <li
        v-for="(st, i) in STAGES"
        :key="st"
        class="stage"
        :data-stage="st"
        :data-state="i < stage ? 'done' : i === stage ? 'current' : 'next'"
        :title="stageState(i)"
      >
        <span class="stage-no" aria-hidden="true">{{ i < stage ? '✓' : i + 1 }}</span>
        <span class="stage-name ant-ellipsis">{{ t(`widgets.decisions.stage.${st}`) }}</span>
        <span class="sr-only" data-testid="stage-state">{{ stageState(i) }}</span>
      </li>
    </ol>
    <p v-if="stage === 2 && handoff" class="handoff ant-wrap" data-testid="handoff-line">{{ stageState(2) }}</p>
    <p v-if="!actions.length" class="muted" data-testid="nothing">{{ t('widgets.decisions.nothingToDecide') }}</p>

    <section v-if="signalActions.length" class="row" data-group="signal">
      <span v-if="stage > 0" class="row-label">{{ t('widgets.decisions.containmentTitle') }}</span>
      <div class="row-body">
        <ActionButton
          v-for="a in signalActions"
          :key="a"
          :size="stage > 0 ? 'small' : size"
          :type="chosen.action === a ? 'primary' : 'default'"
          :secondary="chosen.action !== a"
          :disabled="!canAct || busy"
          :data-action="a"
          :label="stage > 0 ? head(actionLabel(a)) : actionLabel(a)"
          :hint="stage > 0 ? actionLabel(a) : undefined"
          @click="choose(a)"
        />
      </div>
    </section>

    <section v-if="dispositionOpen" class="row" data-group="disposition">
      <span class="row-label">{{ t('decisions.disposition.title') }}</span>
      <!-- Судьба изделия — ряд пилюль: название; 🔒 — нужно разрешение на отклонение; условие и «почему нет» — подсказкой. Смысл и итог — после выбора. -->
      <div class="row-body pills" role="group" :aria-label="t('decisions.disposition.title')">
        <button
          v-for="d in DISPOSITIONS"
          :key="d"
          type="button"
          class="option"
          :data-disposition="d"
          :aria-pressed="chosen.action === 'disposition' && draft.disposition === d"
          :data-allowed="serverActionFor('disposition', d)?.allowed === false ? 'false' : undefined"
          :disabled="!canAct || busy || serverActionFor('disposition', d)?.allowed === false"
          :title="optionHint(d)"
          @click="choose('disposition', d)"
        >
          <span class="option-title">{{ head(t(DISPOSITION_SHORT[d])) }}</span>
          <span v-if="needsConcession(d)" class="lock" aria-hidden="true">🔒</span>
          <span class="option-terms sr-only" :data-concession="needsConcession(d) || undefined">{{ t(DISPOSITION_TERM[d]) }}</span>
          <span v-if="serverActionFor('disposition', d)?.allowed === false" class="option-why sr-only" data-testid="option-why">{{ serverActionFor('disposition', d)!.why_available }}</span>
        </button>
      </div>
      <p class="row-note muted ant-wrap">
        <template v-if="DISPOSITIONS.some(needsConcession)">🔒 {{ t('widgets.decisions.option.needsConcessionShort') }} · </template>{{ t('decisions.disposition.decisionDoesNotWaitForCause') }}
      </p>
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

      <template v-if="draft.action === 'disposition' && draft.disposition">
        <p class="chosen-title ant-wrap" data-testid="chosen-title">{{ t(DISPOSITION_LABEL[draft.disposition], { operation: reworkOperation }) }}</p>
        <p class="ant-wrap">{{ t(DISPOSITION_MEANING[draft.disposition]) }}</p>
        <p class="muted ant-wrap" data-testid="chosen-outcome">{{ t(DISPOSITION_OUTCOME[draft.disposition]) }}</p>
      </template>
      <!-- Что произойдёт — от сервера: деловые на виду, технические по раскрытию (UI-52). -->
      <div v-if="chosenServer" class="effects" data-testid="effects">
        <p class="muted ant-wrap" data-testid="why-available">{{ chosenServer.why_available }}</p>
        <template v-if="chosenServer.consequences.length">
          <p class="chosen-title">{{ t('widgets.presentation.afterDecision') }}</p>
          <ul class="effects-list" data-testid="consequences">
            <li v-for="c in chosenServer.consequences" :key="c" class="ant-wrap">{{ c }}</li>
          </ul>
        </template>
        <template v-if="chosenServer.technical_consequences.length">
          <button type="button" class="more" data-testid="toggle-technical" @click="techOpen = !techOpen">
            {{ techOpen ? t('widgets.presentation.technicalHide') : t('widgets.presentation.technicalShow', { n: chosenServer.technical_consequences.length }) }}
          </button>
          <ul v-if="techOpen" class="effects-list muted" data-testid="technical-consequences">
            <li v-for="c in chosenServer.technical_consequences" :key="c" class="ant-wrap">{{ c }}</li>
          </ul>
        </template>
      </div>
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
      <p v-if="handoff" class="handoff ant-wrap" data-testid="receipt-handoff">
        {{ t('widgets.decisions.handedTo', { who: handoff.role_label }) }}<template v-if="handoff.person"> ({{ handoff.person }})</template>: {{ handoff.task_title }} · {{ handoff.status_label }}
      </p>
      <p class="muted" data-testid="receipt-ref">
        {{ t('widgets.decisions.recorded', { seq: receipt.seq }) }}<template v-if="receipt.ca_ref"> · {{ receipt.ca_ref }}</template>
      </p>
    </section>
  </div>
</template>

<style scoped>
/* Нижняя панель окна: тихо и плотно — шаги строкой, ряды «подпись · кнопки»; подробности — только после выбора. */
.panel {
  container-type: inline-size;
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
  font-size: var(--ant-fs-body);
}

.density-large {
  font-size: var(--ant-fs-lg);
}

p {
  margin: 0;
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  clip: rect(0 0 0 0);
  white-space: nowrap;
}

/* Шаги — одна строка: ✓ пройдено · ● сейчас · ○ дальше, между ними тонкая линия. */
.stages {
  display: flex;
  gap: var(--ant-space-2);
  align-items: center;
  min-width: 0;
  margin: 0;
  padding: 0;
  list-style: none;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.stage {
  position: relative;
  display: flex;
  flex: 0 1 auto;
  gap: 6px;
  align-items: center;
  min-width: 0;
}

.stage + .stage::before {
  flex: none;
  width: 16px;
  height: 1px;
  margin-right: 2px;
  background: var(--ant-border-strong);
  content: '';
}

.stage-no {
  display: inline-flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  border: 1px solid var(--ant-border-strong);
  border-radius: 50%;
  font-size: 10px;
  line-height: 1;
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

.stage[data-state='done'] {
  color: var(--ant-status-success-text);
}

.stage[data-state='done'] .stage-no {
  border-color: var(--ant-status-success);
  background: var(--ant-status-success);
  color: var(--ant-surface);
}

/* Ряд: подпись слева узкой колонкой, кнопки справа; на узком окне подпись над кнопками. */
.row {
  display: grid;
  grid-template-columns: 9.5em minmax(0, 1fr);
  gap: var(--ant-space-1) var(--ant-space-3);
  align-items: center;
  min-width: 0;
}

.row-label {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
  line-height: 1.2;
}

.row-body {
  display: flex;
  flex-wrap: wrap;
  grid-column: 2;
  gap: 6px;
  min-width: 0;
}

.row[data-group='signal']:not(:has(.row-label)) .row-body {
  grid-column: 1 / -1;
}

.row-note {
  grid-column: 2;
}

@container (max-width: 480px) {
  .row {
    grid-template-columns: minmax(0, 1fr);
  }

  .row-body,
  .row-note {
    grid-column: 1;
  }
}

/* Варианты судьбы — пилюли: выбранная залита акцентом, недоступная — бледная. */
.option {
  display: inline-flex;
  gap: 4px;
  align-items: center;
  min-height: 28px;
  padding: 2px 12px;
  border: 1px solid var(--ant-border);
  border-radius: 999px;
  background: var(--ant-surface);
  color: var(--ant-text);
  font: inherit;
  font-size: var(--ant-fs-meta);
  white-space: nowrap;
  cursor: pointer;
  transition:
    background 0.12s ease,
    border-color 0.12s ease,
    color 0.12s ease;
}

.option:hover:not(:disabled) {
  border-color: var(--ant-accent);
  color: var(--ant-accent);
}

.option[aria-pressed='true'] {
  border-color: var(--ant-accent);
  background: var(--ant-accent);
  color: var(--ant-surface);
}

.option:disabled {
  border-style: dashed;
  color: var(--ant-text-3);
  cursor: not-allowed;
}

.option-title {
  font-weight: var(--ant-fw-bold);
}

.lock {
  font-size: 10px;
  opacity: 0.75;
}

.form {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
  padding: var(--ant-space-2) var(--ant-space-3);
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-md);
  background: var(--ant-surface);
}

.field {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.muted,
.warn,
.receipt p {
  font-size: var(--ant-fs-meta);
}

.muted {
  color: var(--ant-text-3);
}

.warn {
  color: var(--ant-status-attention-text);
}

.chosen-title {
  font-weight: var(--ant-fw-bold);
}

.effects {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  padding: var(--ant-space-2) var(--ant-space-3);
  border-left: 3px solid var(--ant-accent);
  border-radius: 0 var(--ant-radius-md) var(--ant-radius-md) 0;
  background: var(--ant-surface-subtle);
}

.effects-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin: 0;
  padding-left: var(--ant-space-5);
}

.more {
  align-self: flex-start;
  padding: 0;
  border: 0;
  background: none;
  color: var(--ant-accent);
  font: inherit;
  font-size: var(--ant-fs-meta);
  cursor: pointer;
}

.handoff {
  font-size: var(--ant-fs-meta);
  font-weight: var(--ant-fw-bold);
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
