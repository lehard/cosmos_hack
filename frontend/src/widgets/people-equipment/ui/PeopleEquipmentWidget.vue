<script setup lang="ts">
/**
 * Виджет «Люди и оборудование» (PRD §3a «Мастер участка», FR-6, FR-17, FR-80,
 * FR-83, FR-84, FR-149; эпик 13) — контейнер: посты и присутствие
 * (`access.workplace.list`), назначения смены (`access.assignment.list`),
 * квалификации (`access.qualification.list`), состояние оборудования
 * (`machinelogs.equipment.list`) и поверка (`reference.equipment.list`).
 * Разделы читаются независимо: отказ одной операции не прячет остальные.
 *
 * Срез: цех — как у «Участка» (`resolveWorkshop`: `workshop`, `scope`); `run_id`.
 */
import { computed } from 'vue'
import { useEquipmentRegistry, useEquipmentStates } from '@/entities/equipment'
import { resolveWorkshop, useLocations } from '@/entities/reference'
import { useSession } from '@/entities/session'
import { useAssignments, usePosts, useQualifications } from '@/entities/workplace'
import { backendModeOf } from '@/shared/api/response'
import type { WidgetProps } from '@/shared/config/widget'
import { WidgetFrame } from '@/shared/ui'
import { buildEquipment, buildPeople, peopleEquipmentState } from '../model/people'
import PeopleEquipmentView from './PeopleEquipmentView.vue'

const props = defineProps<WidgetProps>()
const session = useSession()
const locationsQ = useLocations()

const run = computed(() => (typeof props.slice.run_id === 'string' && props.slice.run_id ? props.slice.run_id : undefined))
const workshop = computed(() => resolveWorkshop(props.slice, locationsQ.data.value?.data ?? [], session.data.value?.data.scope))
const scoped = computed(() => ({ ...(workshop.value ? { workshop: workshop.value.id } : {}), ...(run.value ? { run_id: run.value } : {}) }))

const postsQ = usePosts(scoped)
const assignmentsQ = useAssignments(computed(() => {
  const shift = session.data.value?.data.shift?.id
  return { ...scoped.value, ...(shift ? { shift_id: shift } : {}) }
}))
const qualificationsQ = useQualifications()
const statesQ = useEquipmentStates()
const registryQ = useEquipmentRegistry()

const people = computed(() => {
  const rows = postsQ.data.value?.data
  return rows ? buildPeople(rows, assignmentsQ.data.value?.data.items, qualificationsQ.data.value?.data) : null
})
const equipment = computed(() => {
  const states = statesQ.data.value?.data
  const registry = registryQ.data.value?.data
  if (!states && !registry) return null
  const posts = postsQ.data.value?.data
  return buildEquipment(states, registry, {
    postIds: posts ? new Set(posts.map((p) => p.workplace_id)) : null,
    locations: locationsQ.data.value?.data ?? null,
    workshopId: workshop.value?.id ?? null,
  })
})
const equipmentError = computed(() => (statesQ.error.value && registryQ.error.value ? statesQ.error.value : undefined))

const allFailed = computed(() => !people.value && !!postsQ.error.value && !equipment.value && !!equipmentError.value)
const state = computed(() => (allFailed.value ? 'input_error' : peopleEquipmentState(people.value ?? [], equipment.value ?? [])))
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="backendModeOf(postsQ.data.value ?? statesQ.data.value ?? registryQ.data.value)"
    :state="state"
    :loading="postsQ.isPending.value && statesQ.isPending.value && registryQ.isPending.value"
    :data-widget="widgetId"
  >
    <PeopleEquipmentView
      :workshop-name="workshop?.name ?? null"
      :people="people"
      :people-error="postsQ.error.value ?? undefined"
      :equipment="equipment"
      :equipment-error="equipmentError"
      :density="density"
    />
  </WidgetFrame>
</template>
