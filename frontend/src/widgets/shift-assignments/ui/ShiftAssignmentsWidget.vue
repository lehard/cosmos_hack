<script setup lang="ts">
/**
 * Виджет «Смена» (PRD §3a «Мастер участка — Смена», FR-81, PRD §11.18, UJ-7;
 * эпик 13) — контейнер: смены цеха (`reference.shift.list`), посты и факт
 * присутствия (`access.workplace.list`), план — назначения смены
 * (`access.assignment.list`), сотрудники и роли (`access.person.list`,
 * `access.role.list`), квалификации (`access.qualification.list`).
 * Команды: назначить и снять (`access.assignment.set|clear`), запросить
 * назначение контролёра (`documents.document.request`, шаблон
 * «назначение контролёра», маршрут — согласование начальника ОТК).
 * command_id — UUIDv7 (AD-7); basis_seq — из ответа назначений, policy_seq —
 * сеанса (AD-39).
 *
 * Срез: цех — `resolveWorkshop` (`workshop`, `scope`); `run_id`.
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { usePersons, useRoles } from '@/entities/policy'
import { resolveWorkshop, useLocations, useShifts } from '@/entities/reference'
import { useSession } from '@/entities/session'
import {
  CONTROLLER_ASSIGNMENT_TEMPLATE,
  useAssignmentCommand,
  useAssignments,
  useControllerAssignmentRequest,
  usePosts,
  useQualifications,
  type AccessAssignment,
} from '@/entities/workplace'
import { backendModeOf } from '@/shared/api/response'
import type { WidgetProps } from '@/shared/config/widget'
import { newCommandId } from '@/shared/lib/command-id'
import { useMomentStore } from '@/shared/model/moment'
import { WidgetFrame } from '@/shared/ui'
import { candidates, controllerDecision, rolesInheriting, shiftRows, type AssigneeRole } from '../model/shift'
import ShiftAssignmentsView from './ShiftAssignmentsView.vue'

const props = defineProps<WidgetProps>()
const { t } = useI18n()
const moment = useMomentStore()
const session = useSession()
const locationsQ = useLocations()

const run = computed(() => (typeof props.slice.run_id === 'string' && props.slice.run_id ? props.slice.run_id : undefined))
const workshop = computed(() => resolveWorkshop(props.slice, locationsQ.data.value?.data ?? [], session.data.value?.data.scope))
const workshopScope = computed(() => (workshop.value ? (locationsQ.data.value?.data.find((l) => l.location_id === workshop.value!.id)?.scope ?? null) : null))

const shiftsQ = useShifts(computed(() => workshop.value?.id ?? null))
/** Выбранная смена: по умолчанию — смена сеанса, иначе первая смена цеха. */
const shiftId = ref<string | null>(null)
watch(
  [() => session.data.value?.data.shift?.id, () => shiftsQ.data.value?.data],
  ([own, list]) => {
    if (!shiftId.value) shiftId.value = own ?? list?.[0]?.shift_id ?? null
  },
  { immediate: true },
)

const scoped = computed(() => ({ ...(workshop.value ? { workshop: workshop.value.id } : {}), ...(run.value ? { run_id: run.value } : {}) }))
const postsQ = usePosts(scoped)
const assignmentsQ = useAssignments(computed(() => ({ ...scoped.value, ...(shiftId.value ? { shift_id: shiftId.value } : {}) })))
const personsQ = usePersons()
const rolesQ = useRoles()
const qualificationsQ = useQualifications()

const rows = computed(() => {
  const posts = postsQ.data.value?.data
  return posts ? shiftRows(posts, assignmentsQ.data.value?.data.items) : null
})
const persons = computed(() => personsQ.data.value?.data?.items ?? [])
const roles = computed(() => rolesQ.data.value?.data?.items ?? [])
const performers = computed(() => candidates(persons.value, rolesInheriting(roles.value, 'performer'), workshopScope.value, qualificationsQ.data.value?.data))
const inspectors = computed(() => candidates(persons.value, rolesInheriting(roles.value, 'quality_inspector'), workshopScope.value, qualificationsQ.data.value?.data))
/**
 * Почему назначить нельзя — словами, а не пустым списком «нет данных» (UI-46):
 * нет права читать сотрудников цеха, смена не выбрана, нет допущенных по роли и области.
 */
function blocked(list: readonly unknown[]): string | null {
  if (personsQ.error.value || rolesQ.error.value) return t('widgets.shopFloor.shift.noPeopleAccess')
  if (shiftsQ.data.value && !(shiftsQ.data.value.data ?? []).length) return t('widgets.shopFloor.shift.noShifts')
  if (!shiftId.value) return t('widgets.shopFloor.shift.pickShiftFirst')
  if (personsQ.data.value && !list.length) return t('widgets.shopFloor.shift.noCandidates')
  return null
}
const personName = (id: string) => persons.value.find((p) => p.person_id === id)?.display_name ?? id

const assignCmd = useAssignmentCommand()
const requestCmd = useControllerAssignmentRequest()
const result = ref<string | null>(null)
const busy = computed(() => assignCmd.isPending.value || requestCmd.isPending.value)
const error = computed(() => assignCmd.error.value ?? requestCmd.error.value ?? undefined)

/** Общие поля команды: версия политики сеанса, рабочее место (AD-39, AD-15). */
function meta() {
  const s = session.data.value?.data
  return { command_id: newCommandId(), policy_seq: s?.policy_seq ?? 0, ...(s?.workplace?.id ? { workplace_id: s.workplace.id } : {}) }
}
const basisSeq = () => assignmentsQ.data.value?.data.basis_seq ?? 0

function reset(): void {
  result.value = null
  assignCmd.reset()
  requestCmd.reset()
}

async function onAssign(workplaceId: string, role: AssigneeRole, personId: string, approvalDocumentId: string | null): Promise<void> {
  if (!shiftId.value) return
  reset()
  try {
    const res = await assignCmd.mutateAsync({
      kind: 'set',
      body: {
        ...meta(),
        basis_seq: basisSeq(),
        assignee_role: role,
        person_id: personId,
        workplace_id: workplaceId,
        shift_id: shiftId.value,
        ...(approvalDocumentId ? { approval_document_id: approvalDocumentId } : {}),
      },
    })
    result.value = t('widgets.shopFloor.shift.assigned', { person: personName(personId), seq: res.data.seq })
  } catch {
    // Текст ошибки — из assignCmd.error (например, квалификация истекла).
  }
}

async function onClear(a: AccessAssignment, reason: string): Promise<void> {
  reset()
  try {
    const res = await assignCmd.mutateAsync({
      kind: 'clear',
      body: { ...meta(), basis_seq: basisSeq(), person_id: a.person_id, workplace_id: a.workplace_id, shift_id: a.shift_id, reason: { text: reason } },
    })
    result.value = t('widgets.shopFloor.shift.cleared', { person: personName(a.person_id), seq: res.data.seq })
  } catch {
    // Текст ошибки — из assignCmd.error.
  }
}

async function onRequestController(workplaceId: string, personId: string, comment: string): Promise<void> {
  if (!shiftId.value) return
  reset()
  try {
    const res = await requestCmd.mutateAsync({
      ...meta(),
      basis_seq: basisSeq(),
      template: CONTROLLER_ASSIGNMENT_TEMPLATE,
      subject: { entity: 'workplace', id: workplaceId },
      decision: controllerDecision(personId, shiftId.value),
      ...(comment ? { comment } : {}),
    })
    result.value = t('widgets.shopFloor.shift.requested', { person: personName(personId), seq: res.data.seq })
  } catch {
    // Текст ошибки — из requestCmd.error.
  }
}

const allFailed = computed(() => !rows.value && !!postsQ.error.value && !!assignmentsQ.error.value)
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="backendModeOf(postsQ.data.value ?? assignmentsQ.data.value)"
    :state="allFailed ? 'input_error' : 'normal'"
    :loading="postsQ.isPending.value && assignmentsQ.isPending.value"
    :data-widget="widgetId"
  >
    <ShiftAssignmentsView
      v-model:shift-id="shiftId"
      :workshop-name="workshop?.name ?? null"
      :shifts="shiftsQ.data.value?.data ?? null"
      :shifts-error="shiftsQ.error.value ?? undefined"
      :rows="rows"
      :rows-error="postsQ.error.value ?? undefined"
      :assignments-error="assignmentsQ.error.value ?? undefined"
      :performers="performers"
      :inspectors="inspectors"
      :person-name="personName"
      :performer-blocked="blocked(performers)"
      :inspector-blocked="blocked(inspectors)"
      :can-act="!moment.isReplay"
      :busy="busy"
      :error="error"
      :result="result"
      :density="density"
      @assign="onAssign"
      @clear="onClear"
      @request-controller="onRequestController"
    />
  </WidgetFrame>
</template>
