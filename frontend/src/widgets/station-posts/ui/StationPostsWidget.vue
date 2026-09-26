<script setup lang="ts">
/**
 * Виджет «Участок» (PRD §3a «Мастер участка», FR-81, UJ-7; эпик 13) — контейнер:
 * операции цеха по схеме процесса и живой карте (`process.live_map.read`),
 * показатели по шагу (`analytics.overview.read`), посты
 * (`access.workplace.list`), оборудование (`machinelogs.equipment.list`,
 * `reference.equipment.list`), текущее выполнение (`machinelogs.run_profile.read`).
 * Разделы читаются независимо: отказ одной операции не прячет остальные.
 *
 * Наверху — что пришло и ждёт: задачи процесса «Принять в цех» (форма прямо
 * здесь, `notifications.task.list`) и изделия на входе участка (очередь первой
 * операции, `item.item.list`); затем посты; затем операции. Нехватка данных
 * источника в шапке не показывается — коротко у операции.
 *
 * Срез: `scope: own_station` — цех по области роли сеанса (справочник мест,
 * `reference.location.list`); `workshop: WS-…` — цех явно; иначе весь завод;
 * `run_id` — прогон сценария (AD-38).
 */
import { computed, onBeforeUnmount, ref } from 'vue'
import { useEquipmentRegistry, useEquipmentStates } from '@/entities/equipment'
import { operationsOf, parseProcessSteps, useLiveMap } from '@/entities/live-map'
import { useAnalyticsOverview } from '@/entities/metric'
import { resolveWorkshop, useLocations } from '@/entities/reference'
import { useItemsAtStep } from '@/entities/operation'
import { useSession } from '@/entities/session'
import { useTasks } from '@/entities/task'
import { taskActionOf } from '@/features/task-actions'
import { usePosts } from '@/entities/workplace'
import { useDrillDown } from '@/features/drill-down'
import type { DrillRef } from '@/shared/model/drill'
import { backendModeOf } from '@/shared/api/response'
import type { WidgetProps } from '@/shared/config/widget'
import { useMomentStore } from '@/shared/model/moment'
import { WidgetFrame } from '@/shared/ui'
import { buildPosts, buildSteps, type EntryItem, type IncomingTask } from '../model/station'
import StationPostsView from './StationPostsView.vue'

const props = defineProps<WidgetProps>()
const drill = useDrillDown()
const moment = useMomentStore()
const session = useSession()
const locationsQ = useLocations()

const str = (v: unknown) => (typeof v === 'string' && v ? v : undefined)
const run = computed(() => str(props.slice.run_id))

/** Цех участка: явно из среза или по области роли сеанса. */
const workshop = computed(() => resolveWorkshop(props.slice, locationsQ.data.value?.data ?? [], session.data.value?.data.scope))

const mapQ = useLiveMap(computed(() => ({ period: 'shift' as const, ...(run.value ? { run_id: run.value } : {}) })))
const overview = useAnalyticsOverview()
const postsQ = usePosts(computed(() => ({ ...(workshop.value ? { workshop: workshop.value.id } : {}), ...(run.value ? { run_id: run.value } : {}) })))
const equipmentQ = useEquipmentStates()
const registryQ = useEquipmentRegistry()

const parsed = computed(() => parseProcessSteps(mapQ.data.value?.data.bpmn_xml))
const steps = computed(() => {
  const map = mapQ.data.value?.data
  if (!map) return null
  return buildSteps(operationsOf(parsed.value, workshop.value?.id ?? null), map, overview.data.value?.items)
})
const posts = computed(() => {
  const rows = postsQ.data.value?.data
  return rows ? buildPosts(rows, equipmentQ.data.value?.data) : null
})

// «Сейчас» для «идёт N мин» — раз в полминуты; в воспроизведении — момент просмотра.
const now = ref(new Date())
const timer = setInterval(() => (now.value = new Date()), 30_000)
onBeforeUnmount(() => clearInterval(timer))
const at = computed(() => (moment.asOf ? new Date(moment.asOf) : now.value))

// Что пришло и ждёт: открытые задачи приёмки (действие process.movement.receive).
const tasksQ = useTasks(computed(() => (run.value ? { run_id: run.value } : {})))
const incoming = computed<IncomingTask[]>(() =>
  (tasksQ.data.value?.data.items ?? [])
    .filter((t) => t.state === 'open')
    .flatMap((t) => {
      const a = taskActionOf(t)
      return a.kind === 'form' && a.form === 'receive' ? [{ taskId: t.task_id, itemId: a.itemId, title: t.title }] : []
    }),
)
// Изделия на входе участка — очередь первой операции цеха.
const entryStep = computed(() => {
  const first = steps.value?.[0]
  return first && (first.counters?.queue ?? 0) > 0 ? first.step.stepKey : null
})
const entryQ = useItemsAtStep(entryStep)
const entryItems = computed<EntryItem[]>(() => {
  if (!entryStep.value) return []
  const waiting = new Set(incoming.value.map((r) => r.itemId))
  return (entryQ.data.value?.data ?? []).filter((i) => !waiting.has(i.item_id)).map((i) => ({ item_id: i.item_id, label: i.label }))
})

const allFailed = computed(() => !!mapQ.error.value && !steps.value && !!postsQ.error.value && !posts.value)
const state = computed(() => (allFailed.value ? 'input_error' : 'normal'))
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="backendModeOf(mapQ.data.value ?? postsQ.data.value)"
    :state="state"
    :loading="mapQ.isPending.value && postsQ.isPending.value"
    :data-widget="widgetId"
  >
    <StationPostsView
      :workshop-name="workshop?.name ?? null"
      :steps="steps"
      :steps-error="mapQ.error.value ?? undefined"
      :posts="posts"
      :posts-error="postsQ.error.value ?? undefined"
      :registry="registryQ.data.value?.data ?? null"
      :step-index="parsed.byKey"
      :at="at"
      :density="density"
      :incoming="incoming"
      :entry-items="entryItems"
      :basis-seq="tasksQ.data.value?.data.basis_seq ?? 0"
      :can-act="!moment.isReplay"
      @item="(id) => drill.open({ entity: 'item', id })"
      @node="(key) => drill.open({ entity: 'operation', id: key } as unknown as DrillRef)"
      @workplace="(id) => drill.open({ entity: 'workplace', id })"
      @person="(id) => drill.open({ entity: 'person', id } as unknown as DrillRef)"
    />
  </WidgetFrame>
</template>
