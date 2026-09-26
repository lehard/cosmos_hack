<script setup lang="ts">
/**
 * Виджет «Терминал исполнителя» (FR-137, FR-55, FR-17; PRD §3a; эпик 13) —
 * контейнер. Только своё рабочее место (из сеанса): оборудование поста
 * (`machinelogs.equipment.list?station_id`, `reference.equipment.list`),
 * текущее выполнение (`machinelogs.run_profile.read`), операции цеха по схеме
 * (`process.live_map.read`), изделия шага (`item.item.list`), пост
 * (`access.workplace.list`).
 *
 * Команды — с рабочим местом сеанса (барьер 2, AD-15), command_id — UUIDv7
 * (AD-7), basis_seq — паспорта изделия, policy_seq — сеанса (AD-39):
 * начать и остановить операцию (`process.operation.start|finish`), сообщить об
 * отклонении и запросить контроль (`access.operator.*`), подтвердить
 * перемещение в изолятор (`process.movement.receive` — фича task-actions).
 * Подпись уровня 1 — агент токена (эпик 38); в демо команда уходит без подписи
 * (Д-30).
 */
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useEquipmentRegistry, useEquipmentStates, useRunProfile } from '@/entities/equipment'
import { elapsedMinutes, normText, operationsOf, overNorm, parseProcessSteps, useLiveMap } from '@/entities/live-map'
import { useItemBasis, useItemsAtStep, useOperationCommand, type FinishOperationCompletion } from '@/entities/operation'
import { useLocations, workshopOf } from '@/entities/reference'
import { useSession } from '@/entities/session'
import { useTasks } from '@/entities/task'
import { useOperatorCommand, usePosts } from '@/entities/workplace'
import { useDrillDown } from '@/features/drill-down'
import { WorkplaceAdmission } from '@/features/workplace-admission'
import { backendModeOf } from '@/shared/api/response'
import type { WidgetProps } from '@/shared/config/widget'
import { newCommandId } from '@/shared/lib/command-id'
import { useMomentStore } from '@/shared/model/moment'
import { WidgetFrame } from '@/shared/ui'
import { currentRunId, postItems, startCandidates, terminalState, terminalWarnings } from '../model/terminal'
import PerformerTerminalView from './PerformerTerminalView.vue'

const props = defineProps<WidgetProps>()
const { t } = useI18n()
const drill = useDrillDown()
const moment = useMomentStore()
const session = useSession()
const locationsQ = useLocations()

const run = computed(() => (typeof props.slice.run_id === 'string' && props.slice.run_id ? props.slice.run_id : undefined))
const workplace = computed(() => session.data.value?.data.workplace ?? null)
const workplaceId = computed(() => workplace.value?.id ?? null)

// Оборудование поста стоит на участке (IS-1/IS-2 → ST-WELD), а не на самом посту: читаем по участку,
// иначе список пуст, «Выполнено» держится только на startedHere и пропадает после перезагрузки страницы.
const stationId = computed(() => {
  const wp = workplaceId.value
  if (!wp) return null
  const loc = (locationsQ.data.value?.data ?? []).find((l) => l.location_id === wp)
  return loc?.parent_id ?? wp
})
const equipmentQ = useEquipmentStates(stationId)
const registryQ = useEquipmentRegistry()
const postsQ = usePosts(computed(() => (run.value ? { run_id: run.value } : {})))
const mapQ = useLiveMap(computed(() => ({ period: 'shift' as const, ...(run.value ? { run_id: run.value } : {}) })))

// Изделие поста и цех рабочего места: из панели постов, иначе по справочнику мест.
const post = computed(() => postsQ.data.value?.data.find((p) => p.workplace_id === workplaceId.value) ?? null)
const workshopId = computed(() => {
  if (post.value?.workshop) return post.value.workshop
  const wp = workplaceId.value
  return wp ? (workshopOf(locationsQ.data.value?.data ?? [], wp)?.location_id ?? null) : null
})
const parsed = computed(() => parseProcessSteps(mapQ.data.value?.data.bpmn_xml))
const operations = computed(() => operationsOf(parsed.value, workshopId.value))

// Без поста в сеансе оборудования «своего места» нет: чужие операции и
// предупреждения не показываем (UI-40) — запрос без station_id вернул бы весь завод.
// Своё оборудование: источник, привязанный к посту по номеру (WP-WELD-1 → IS-1), иначе всё оборудование участка.
const equipment = computed(() => {
  const wp = workplaceId.value
  if (!wp) return []
  const all = equipmentQ.data.value?.data ?? []
  const n = /-(\d+)$/.exec(wp)?.[1]
  const own = n ? all.filter((e) => new RegExp(`-${n}$`).test(e.equipment_id)) : []
  return own.length ? own : all
})
/** Выполнение, начатое с этого терминала, — пока оборудование его не показало. */
const startedHere = ref<{ runId: string; itemId: string } | null>(null)
const runId = computed(() => currentRunId(equipment.value, startedHere.value?.runId ?? null))
const profileQ = useRunProfile(runId)
const runProfile = computed(() => (profileQ.data.value?.data && !profileQ.data.value.data.finished_at ? profileQ.data.value.data : null))
const runItemId = computed(() => runProfile.value?.item_id ?? startedHere.value?.itemId ?? null)

const stepKey = ref<string | null>(null)
/** Операция, по которой у исполнителя есть открытая задача «Начать», — её и показываем. */
const tasksQ = useTasks(computed(() => ({})))
const taskStep = computed(() => {
  const open = (tasksQ.data.value?.data.items ?? []).filter((t) => t.state === 'open' && t.operation_id === 'process.operation.start')
  return open.find((t) => t.step_key)?.step_key ?? null
})
watch(
  [() => runProfile.value?.step_key, operations, taskStep],
  ([running, ops, fromTask]) => {
    if (running) stepKey.value = running
    else if (fromTask && ops.some((o) => o.stepKey === fromTask)) stepKey.value = fromTask
    else if (!stepKey.value || !ops.some((o) => o.stepKey === stepKey.value)) stepKey.value = ops[0]?.stepKey ?? null
  },
  { immediate: true },
)
const itemsQ = useItemsAtStep(stepKey)
const items = computed(() => postItems(post.value, runProfile.value, itemsQ.data.value?.data))
/** Очередь шага + изделия из открытых задач «Начать» этого шага (шаг изделия меняется только при старте операции). */
const candidates = computed(() => {
  const base = startCandidates(itemsQ.data.value?.data)
  const fromTasks = (tasksQ.data.value?.data.items ?? [])
    .filter((t) => t.state === 'open' && t.operation_id === 'process.operation.start' && t.step_key === stepKey.value && t.item_id)
    .map((t) => ({ row: { item_id: t.item_id!, label: t.item_label ?? t.item_id! } as (typeof base)[number]['row'], blocked: false }))
  // Пока операция начата отсюда и сервер её ещё не показал — очередь скрыта: второй «Начать»
  // до ответа сервера ушёл бы на другое изделие тем же выполнением.
  if (startedHere.value && !runProfile.value) return []
  // Очередь — только изделия с открытой задачей «Начать». Список шага (items?step_key) не годится:
  // шаг изделия в паспорте отстаёт от процесса, там висит уже сваренное из истории (Ф-101, Ф-121…),
  // и «Начать» по нему даёт отказ «изделие на шаге welding.weld (сейчас: …)».
  void base
  return fromTasks
})

// «Идёт N мин» против нормы шага.
const now = ref(new Date())
const timer = setInterval(() => (now.value = new Date()), 30_000)
onBeforeUnmount(() => clearInterval(timer))
const at = computed(() => (moment.asOf ? new Date(moment.asOf) : now.value))
const runStep = computed(() => (runProfile.value ? (parsed.value.byKey.get(runProfile.value.step_key) ?? null) : null))
const runMinutes = computed(() => (runProfile.value ? elapsedMinutes(runProfile.value.started_at, at.value) : null))
const runOver = computed(() => (runStep.value ? overNorm(runMinutes.value, runStep.value.norm) : null))
const runNorm = computed(() => normText(t, runStep.value?.norm))

// ── команды ──
const opCmd = useOperationCommand()
const operatorCmd = useOperatorCommand()
const result = ref<string | null>(null)
const busy = computed(() => opCmd.isPending.value || operatorCmd.isPending.value)
const error = computed(() => opCmd.error.value ?? operatorCmd.error.value ?? undefined)

/** seq паспорта изделия для basis_seq команды (AD-39). */
const basisOf = useItemBasis()

/** Общие поля команды. */
function meta(basisSeq: number) {
  const s = session.data.value?.data
  return { command_id: newCommandId(), basis_seq: basisSeq, policy_seq: s?.policy_seq ?? 0, ...(workplaceId.value ? { workplace_id: workplaceId.value } : {}) }
}

function reset(): void {
  result.value = null
  opCmd.reset()
  operatorCmd.reset()
}

async function onStart(itemId: string): Promise<void> {
  const step = operations.value.find((o) => o.stepKey === stepKey.value)
  if (!step) return
  reset()
  const runIdNew = newCommandId()
  try {
    const res = await opCmd.mutateAsync({
      kind: 'start',
      item_id: itemId,
      body: {
        ...meta(await basisOf(itemId)),
        operation_run_id: runIdNew,
        operation_code: step.operationCode ?? step.stepKey,
        step_key: step.stepKey,
        ...(workplaceId.value ? { station_id: workplaceId.value } : {}),
        ...(equipment.value[0] ? { equipment_id: equipment.value[0].equipment_id } : {}),
      },
    })
    startedHere.value = { runId: runIdNew, itemId }
    result.value = t('widgets.shopFloor.terminal.started', { seq: res.data.seq })
  } catch {
    // Текст отказа (предусловия, лимит доработок, точка предъявления) — из opCmd.error.
  }
}

async function onFinish(completion: FinishOperationCompletion): Promise<void> {
  const id = runId.value
  const item = runItemId.value
  if (!id || !item) return
  reset()
  try {
    const res = await opCmd.mutateAsync({ kind: 'finish', item_id: item, run_id: id, body: { ...meta(await basisOf(item)), completion } })
    startedHere.value = null
    result.value = t('widgets.shopFloor.terminal.finished', { seq: res.data.seq })
  } catch {
    // Текст ошибки — из opCmd.error.
  }
}

async function onDeviation(description: string, itemId: string | null): Promise<void> {
  const wp = workplaceId.value
  if (!wp) return
  reset()
  const item = itemId ?? runItemId.value
  try {
    const res = await operatorCmd.mutateAsync({
      kind: 'report_deviation',
      workplace_id: wp,
      body: { ...meta(await basisOf(item)), description, ...(item ? { item_id: item } : {}), ...(runId.value ? { operation_run_id: runId.value } : {}) },
    })
    result.value = t('widgets.shopFloor.terminal.deviationSent', { seq: res.data.seq })
  } catch {
    // Текст ошибки — из operatorCmd.error.
  }
}

async function onInspection(itemId: string | null): Promise<void> {
  const wp = workplaceId.value
  const step = stepKey.value
  if (!wp || !step) return
  reset()
  const item = itemId ?? runItemId.value
  try {
    const res = await operatorCmd.mutateAsync({
      kind: 'request_inspection',
      workplace_id: wp,
      body: { ...meta(await basisOf(item)), step_key: step, ...(item ? { item_id: item } : {}), ...(runId.value ? { operation_run_id: runId.value } : {}) },
    })
    result.value = t('widgets.shopFloor.terminal.inspectionSent', { seq: res.data.seq })
  } catch {
    // Текст ошибки — из operatorCmd.error.
  }
}

const warnings = computed(() => terminalWarnings(equipment.value, registryQ.data.value?.data))
const state = computed(() => (!session.data.value && session.error.value ? 'input_error' : terminalState(equipment.value)))
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="backendModeOf(equipmentQ.data.value ?? mapQ.data.value)"
    :state="state"
    :error="session.data.value ? undefined : (session.error.value ?? undefined)"
    :loading="session.isPending.value"
    :data-widget="widgetId"
  >
    <!-- Эпик 37: допуск к рабочему месту (барьер 2) — рабочее место сеанса. -->
    <WorkplaceAdmission :run-id="run" :can-act="!moment.isReplay" :density="density" />
    <PerformerTerminalView
      v-model:step-key="stepKey"
      :workplace="workplace"
      :shift-title="session.data.value?.data.shift?.title ?? null"
      :warnings="warnings"
      :run="runProfile"
      :run-step="runStep"
      :run-minutes="runMinutes"
      :run-over="runOver"
      :run-norm="runNorm"
      :operations="operations"
      :candidates="candidates"
      :items-error="itemsQ.error.value ?? undefined"
      :items="items"
      :can-act="!moment.isReplay"
      :busy="busy"
      :error="error"
      :result="result"
      :density="density"
      @start="onStart"
      @finish="onFinish"
      @deviation="onDeviation"
      @inspection="onInspection"
      @open="(id) => drill.open({ entity: 'item', id })"
    />
  </WidgetFrame>
</template>
