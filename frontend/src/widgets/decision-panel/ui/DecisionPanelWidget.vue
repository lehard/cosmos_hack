<script setup lang="ts">
/**
 * Виджет «Что решить» — панель решений (FR-52, FR-53, FR-146) — контейнер.
 *
 * Данные: карточка (`nonconformity.card.read`, тот же кэш, что у карточки),
 * права по изделию и несоответствию (`access.permission.list`), объяснение
 * (`access.permission.explain`), разрешения на отклонение
 * (`nonconformity.concession.list`), сеанс (policy_seq, рабочее место).
 *
 * Подпись (FR-66, AD-13): черновик → окно уровня 2 со сводкой → агент токена
 * через порт подписи (features/sign-decision) → команда с конвертом DSSE.
 * Агента нет: в демонстрационном профиле — команда без подписи агента (в
 * паспорте подпись «не проверялась»); иначе подписать нечем (бумажного пути для
 * команд решений в контракте пока нет). `command_id` один на решение: повтор
 * после сбоя не создаёт второе решение (AD-7).
 */
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useRequestDecision } from '@/entities/document'
import {
  DECISION_ACTIONS,
  buildDecisionRequest,
  decisionSummary,
  useConcessions,
  useDecisionCommand,
  useNcCard,
  uuidv7,
  type DecisionDraft,
  type Receipt,
} from '@/entities/nonconformity'
import { useSession } from '@/entities/session'
import { useExplanation, useObjectActions } from '@/features/decision-authority'
import { SignDialog, payloadTypeOf, toPayloadB64, useSigningPort } from '@/features/sign-decision'
import { backendModeOf } from '@/shared/api/response'
import type { WidgetProps } from '@/shared/config/widget'
import { useMomentStore } from '@/shared/model/moment'
import { WidgetFrame } from '@/shared/ui'
import DecisionPanelView from './DecisionPanelView.vue'

const props = defineProps<WidgetProps>()
const route = useRoute()
const moment = useMomentStore()
const port = useSigningPort()

const str = (v: unknown): string | undefined => (typeof v === 'string' && v ? v : undefined)
// Несоответствие — из среза: окно записи (Д-70) или страница карточки.
const ncId = computed(() => str(props.slice.nc_id) ?? null)
const runId = computed(() => str(route?.query.run) ?? str(props.slice.run_id))

const cardQ = useNcCard(ncId, runId)
const card = computed(() => cardQ.data.value?.data ?? null)
const itemId = computed(() => card.value?.item_id ?? '')
const session = useSession()

// Права: действия над изделием (отклонить, изолировать, доп. проверка) и над несоответствием.
const itemActions = useObjectActions('item', itemId)
const ncActions = useObjectActions('nonconformity', () => ncId.value ?? '')
const allowed = computed<Set<string> | null>(() => {
  const a = itemActions.allowed.value
  const b = ncActions.allowed.value
  return a && b ? new Set([...a, ...b]) : null
})

const needConcessions = computed(() => !!card.value?.to_decide.decisions.includes(DECISION_ACTIONS.disposition.operation))
const concessionsQ = useConcessions(itemId, needConcessions)
const concessions = computed(() => concessionsQ.data.value?.data?.items ?? [])

// «Почему вы можете / не можете» — по выбранной операции.
const explainOp = ref<string | null>(null)
const explainSubject = computed(() => {
  const def = Object.values(DECISION_ACTIONS).find((d) => d.operation === explainOp.value)
  return def?.subject ?? 'nonconformity'
})
const explainId = computed(() => (explainSubject.value === 'item' ? itemId.value : (ncId.value ?? '')))
const explanationQ = useExplanation(explainOp, explainSubject, explainId)
function toggleExplain(op: string): void {
  explainOp.value = explainOp.value === op ? null : op
}

// Подпись и команда.
const command = useDecisionCommand()
const requestDecision = useRequestDecision()
const pending = ref<{ draft: DecisionDraft; commandId: string } | null>(null)
const dialogError = ref<unknown>(undefined)
const receipt = ref<Receipt | null>(null)
const signing = ref(false)

watch(ncId, () => {
  pending.value = null
  receipt.value = null
  explainOp.value = null
})

const summary = computed(() => (pending.value && card.value ? decisionSummary(pending.value.draft, card.value, concessions.value) : []))
const demoUnsigned = computed(() => session.data.value?.data?.demo === true)

function openSign(draft: DecisionDraft): void {
  dialogError.value = undefined
  pending.value = { draft, commandId: uuidv7() }
}

async function submit(withAgent: boolean): Promise<void> {
  const p = pending.value
  const c = card.value
  const s = session.data.value?.data
  if (!p || !c || !s) return
  dialogError.value = undefined
  signing.value = true
  try {
    const meta = { command_id: p.commandId, policy_seq: s.policy_seq, ...(s.workplace?.id ? { workplace_id: s.workplace.id } : {}) }
    let req = buildDecisionRequest(p.draft, c, meta)
    if (withAgent) {
      // Уровень 2: агент показывает доверенную сводку и ждёт касания токена (AD-14).
      // Событие-команду, отпечаток и сводку расширение собирает само из
      // операции, параметров пути и тела — тем же пакетом, что сервер (AD-12).
      const def = DECISION_ACTIONS[p.draft.action]
      const params = def.subject === 'item' ? { item_id: req.item_id } : { nc_id: req.nc_id }
      const signature = await port.sign({
        level: 2,
        payload_type: payloadTypeOf('event'),
        payload_b64: toPayloadB64(req.body),
        event_type: def.eventType,
        command_request: { operation: def.operation, params, item_id: req.item_id },
      })
      req = buildDecisionRequest(p.draft, c, { ...meta, signature })
    }
    const res = await command.mutateAsync(req)
    receipt.value = res.data as Receipt
    pending.value = null
  } catch (err) {
    // Отказ в окне подтверждения — не ошибка: окно решения остаётся открытым.
    if ((err as { info?: { code?: string } } | null)?.info?.code !== 'signing.cancelled') dialogError.value = err
  } finally {
    signing.value = false
  }
}

function askForDecision(operation: string): void {
  const subject = Object.values(DECISION_ACTIONS).find((d) => d.operation === operation)?.subject ?? 'nonconformity'
  const id = subject === 'item' ? itemId.value : (ncId.value ?? '')
  requestDecision.mutate({ action: operation, subject_ref: `${subject}:${id}` })
}
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="backendModeOf(cardQ.data.value)"
    :loading="!!ncId && cardQ.isPending.value && !card"
    :error="card ? undefined : cardQ.error.value"
    :empty="!ncId"
    empty-key="widgets.ncCard.noSelection"
    :data-widget="widgetId"
  >
    <DecisionPanelView
      v-if="card"
      :card="card"
      :allowed="allowed"
      :concessions="concessions"
      :can-act="!moment.isReplay"
      :busy="signing || command.isPending.value"
      :error="requestDecision.error.value ?? undefined"
      :receipt="receipt"
      :explanation="explanationQ.data.value?.data ?? null"
      :explanation-open="!!explainOp"
      :explanation-loading="!!explainOp && explanationQ.isPending.value"
      :explanation-error="explanationQ.error.value ?? undefined"
      :density="density"
      @sign="openSign"
      @explain="toggleExplain"
      @request-decision="askForDecision"
      @create-concession="askForDecision('nonconformity.concession.create')"
    />
    <SignDialog
      :show="!!pending"
      stage="confirm"
      :summary="summary"
      :token-status="port.status.value"
      :paper-allowed="false"
      :demo-unsigned="demoUnsigned"
      :busy="signing"
      :error="dialogError"
      @confirm-token="submit(true)"
      @confirm-unsigned="submit(false)"
      @close="pending = null"
    />
  </WidgetFrame>
</template>
