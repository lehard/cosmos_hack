<script setup lang="ts">
/**
 * Виджет «Участок» (PRD §3a «Мастер участка», FR-81, UJ-7; эпик 13) — контейнер:
 * операции цеха по схеме процесса и живой карте (`process.live_map.read`),
 * показатели по шагу (`analytics.overview.read`), посты
 * (`access.workplace.list`), оборудование (`machinelogs.equipment.list`,
 * `reference.equipment.list`), текущее выполнение (`machinelogs.run_profile.read`).
 * Разделы читаются независимо: отказ одной операции не прячет остальные.
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
import { useSession } from '@/entities/session'
import { usePosts } from '@/entities/workplace'
import { useAlerts } from '@/entities/notification'
import { useTasks } from '@/entities/task'
import { useDrillDown } from '@/features/drill-down'
import { AssignPostDrawer } from '@/features/post-assignment'
import type { DrillRef } from '@/shared/model/drill'
import { backendModeOf } from '@/shared/api/response'
import type { WidgetProps } from '@/shared/config/widget'
import { useMomentStore } from '@/shared/model/moment'
import { WidgetFrame } from '@/shared/ui'
import { buildPosts, buildSteps, stationState } from '../model/station'
import { nowActions } from '../model/now'
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

// «Требует действий»: открытые задачи мастера, несостыковки изоляции, люди не на месте, посты без исполнителя.
const runParams = computed(() => (run.value ? { run_id: run.value } : {}))
const tasksQ = useTasks(runParams)
const alertsQ = useAlerts(runParams)
const actions = computed(() => nowActions(tasksQ.data.value?.data.items ?? [], alertsQ.data.value?.data ?? [], postsQ.data.value?.data ?? []))
const assignTo = ref<{ id: string; title: string } | null>(null)
const shift = computed(() => session.data.value?.data.shift ?? null)

const allFailed = computed(() => !!mapQ.error.value && !steps.value && !!postsQ.error.value && !posts.value)
const state = computed(() => (allFailed.value ? 'input_error' : stationState(steps.value ?? [], posts.value ?? [])))
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
      @item="(id) => drill.open({ entity: 'item', id })"
      :shift-title="shift?.title ?? null"
      :actions="actions"
      @assign="(id, title) => (assignTo = { id, title })"
      @open="(ref) => drill.open(ref)"
      @node="(key) => drill.open({ entity: 'operation', id: key } as unknown as DrillRef)"
      @workplace="(id) => drill.open({ entity: 'workplace', id })"
      @person="(id) => drill.open({ entity: 'person', id } as unknown as DrillRef)"
    />
    <AssignPostDrawer
      :show="!!assignTo"
      :workplace-id="assignTo?.id ?? null"
      :workplace-title="assignTo?.title ?? ''"
      :shift-id="shift?.id ?? null"
      :shift-title="shift?.title ?? null"
      :can-act="!moment.isReplay"
      @close="assignTo = null"
    />
  </WidgetFrame>
</template>
