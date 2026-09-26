<script setup lang="ts">
/**
 * Смена — представление (PRD §3a «Смена», FR-81, PRD §11.18, UJ-7): план и
 * факт смены по постам — кто назначен (исполнитель, контролёр), допуск и
 * квалификация, на месте ли. Исполнителя назначает мастер — только допущенного
 * по квалификации (окончательно решает сервер); контролёра — запросом с
 * согласованием начальника ОТК, назначение — по закрытому маршруту документа.
 * Элементы и токены «Главного»: посты — панели секций, действия — ToolBar и
 * ActionButton.
 */
import { computed, reactive } from 'vue'
import { useI18n } from 'vue-i18n'
import { NAlert, NInput, NSelect, NTag } from 'naive-ui'
import type { RefShift } from '@/entities/reference'
import { PRESENCE_TEXT, presenceTagType, type AccessAssignment } from '@/entities/workplace'
import type { Density } from '@/shared/config/widget'
import { naiveSizeOf } from '@/shared/config/widget'
import { useProblemText } from '@/shared/i18n/problem'
import { ActionButton, DataTable, EmptyState, SectionPanel, ToolBar } from '@/shared/ui'
import { shiftLabel, type AssigneeRole, type Candidate, type ShiftPostRow } from '../model/shift'
import PostApprovals from './PostApprovals.vue'

const props = withDefaults(
  defineProps<{
    workshopName: string | null
    shifts: readonly RefShift[] | null
    shiftsError?: unknown
    shiftId: string | null
    rows: readonly ShiftPostRow[] | null
    rowsError?: unknown
    assignmentsError?: unknown
    performers: readonly Candidate[]
    inspectors: readonly Candidate[]
    personName: (id: string) => string
    /** Почему назначить нельзя — словами (нет доступа к сотрудникам, смена не выбрана, нет допущенных); null — можно. */
    performerBlocked?: string | null
    inspectorBlocked?: string | null
    canAct?: boolean
    busy?: boolean
    error?: unknown
    /** Итог последней команды (запись журнала). */
    result?: string | null
    density?: Density
  }>(),
  { shiftsError: undefined, rowsError: undefined, assignmentsError: undefined, performerBlocked: null, inspectorBlocked: null, canAct: true, busy: false, error: undefined, result: null, density: 'comfortable' },
)
const emit = defineEmits<{
  'update:shiftId': [id: string]
  assign: [workplaceId: string, role: AssigneeRole, personId: string, approvalDocumentId: string | null]
  clear: [a: AccessAssignment, reason: string]
  requestController: [workplaceId: string, personId: string, comment: string]
  /** Настроить пост — правое окно назначения; документ согласования, если выбран. */
  configure: [workplaceId: string, station: string, approvalDocumentId: string | null]
}>()

const { t, d } = useI18n()
/** Почему назначать нельзя — одна причина на экран (исполнителей и контролёров). */
const blockedReason = computed(() => props.performerBlocked ?? props.inspectorBlocked ?? null)
const problemText = useProblemText()
const size = computed(() => naiveSizeOf(props.density))
const time = (iso: string) => d(new Date(iso), 'time')

const shiftOptions = computed(() => (props.shifts ?? []).map((s) => ({ label: shiftLabel(s, time), value: s.shift_id })))

/** Черновики снятия с поста: основания. */
const clearing = reactive<Record<string, string>>({})
const clearReason = reactive<Record<string, string>>({})

const clearKey = (a: AccessAssignment) => `${a.workplace_id}/${a.person_id}/${a.assignee_role}`

function confirmClear(a: AccessAssignment): void {
  const reason = (clearReason[clearKey(a)] ?? '').trim()
  if (!reason) return
  emit('clear', a, reason)
  delete clearing[clearKey(a)]
}
</script>

<template>
  <div class="shift" data-testid="shift-view">
    <ToolBar>
      <span class="ant-muted ant-wrap">
        {{ workshopName ? t('widgets.shopFloor.station.workshop', { name: workshopName }) : t('widgets.shopFloor.station.noWorkshop') }}
      </span>
      <template #end>
        <span>{{ t('common.words.shift') }}:</span>
        <NSelect
          v-if="shiftOptions.length"
          class="picker"
          :value="shiftId"
          :options="shiftOptions"
          :size="size"
          :consistent-menu-width="false"
          data-testid="shift"
          @update:value="(v: string) => emit('update:shiftId', v)"
        />
        <span v-else-if="shiftsError" class="error ant-wrap">{{ problemText(shiftsError) }}</span>
        <span v-else class="ant-muted">{{ shiftId ?? '—' }}</span>
      </template>
    </ToolBar>
    <p class="ant-muted ant-wrap">{{ t('access.onlyQualified') }} · {{ t('widgets.shopFloor.shift.controllerRule') }}</p>
    <!-- Одна плашка сверху: почему назначить нельзя и что делать (UI-48) — вместо повтора под каждым постом. -->
    <NAlert v-if="canAct && blockedReason" type="warning" :bordered="false" :title="t('widgets.shopFloor.shift.cannotAssignTitle')" data-testid="shift-blocked">
      {{ blockedReason }}
    </NAlert>
    <NAlert v-else-if="assignmentsError" type="warning" :bordered="false" data-testid="assignments-error">
      {{ t('widgets.shopFloor.shift.planUnavailable') }}: {{ problemText(assignmentsError) }}
    </NAlert>

    <SectionPanel :title="t('access.shiftsAndAssignments')" variant="plain">
      <NAlert v-if="rowsError && !rows" type="error" :bordered="false">{{ problemText(rowsError) }}</NAlert>
      <EmptyState v-else-if="rows && !rows.length" compact :title="t('empty.noRecords')" />
      <!-- Смена одной таблицей: пост · исполнитель · факт · контролёр · «Настроить» (правое окно). -->
      <DataTable v-else-if="rows" :caption="t('access.shiftsAndAssignments')" data-testid="shift-table">
        <thead>
          <tr>
            <th>{{ t('liveMap.posts.station') }}</th>
            <th>{{ t('widgets.shopFloor.shift.performer') }}</th>
            <th>{{ t('widgets.shopFloor.shift.fact') }}</th>
            <th>{{ t('widgets.shopFloor.shift.inspector') }}</th>
            <th />
          </tr>
        </thead>
        <tbody>
          <tr v-for="r in rows" :key="r.post.workplace_id" :data-workplace="r.post.workplace_id">
            <th scope="row"><span class="ant-wrap">{{ r.post.station }}</span></th>
            <td data-testid="performers">
              <span v-if="!r.performers.length" class="ant-muted">{{ t('liveMap.posts.notAssigned') }}</span>
              <div v-for="a in r.performers" :key="clearKey(a)" class="line" :data-assigned="a.person_id">
                <span class="ant-wrap">{{ personName(a.person_id) }}</span>
                <NTag v-if="!a.qualification_ok" size="small" :bordered="false" type="error">{{ t('widgets.shopFloor.people.qualificationNotOk') }}</NTag>
                <NTag v-else-if="!a.admitted" size="small" :bordered="false" type="warning">{{ t('widgets.shopFloor.people.notAdmitted') }}</NTag>
                <template v-if="canAct">
                  <ActionButton v-if="clearing[clearKey(a)] === undefined" size="tiny" quaternary :label="t('widgets.shopFloor.shift.clear')" data-testid="clear" @click="clearing[clearKey(a)] = ''" />
                  <form v-else class="line" @submit.prevent="confirmClear(a)">
                    <NInput v-model:value="clearReason[clearKey(a)]" class="grow" size="small" :placeholder="t('widgets.shopFloor.shift.clearReason')" data-testid="clear-reason" />
                    <ActionButton size="small" type="warning" attr-type="submit" :disabled="busy || !(clearReason[clearKey(a)] ?? '').trim()" :label="t('widgets.shopFloor.shift.clear')" data-testid="confirm-clear" />
                  </form>
                </template>
              </div>
            </td>
            <td>
              <div class="line">
                <span v-if="r.post.assigned" class="ant-wrap">{{ r.post.assigned.display }}</span>
                <NTag size="small" :bordered="false" :type="presenceTagType(r.post.presence)"><span class="ant-wrap">{{ t(PRESENCE_TEXT[r.post.presence]) }}</span></NTag>
              </div>
            </td>
            <td data-testid="inspectors">
              <span v-if="!r.inspectors.length" class="ant-muted">{{ t('liveMap.posts.notAssigned') }}</span>
              <span v-for="a in r.inspectors" :key="clearKey(a)" class="ant-wrap" :data-assigned="a.person_id">{{ personName(a.person_id) }}</span>
              <PostApprovals :workplace-id="r.post.workplace_id" :can-act="canAct" size="small" @use="(doc) => emit('configure', r.post.workplace_id, r.post.station, doc.document_id)" />
            </td>
            <td>
              <ActionButton v-if="canAct" size="small" secondary :label="t('widgets.shopFloor.shift.configure')" data-testid="configure" @click="emit('configure', r.post.workplace_id, r.post.station, null)" />
            </td>
          </tr>
        </tbody>
      </DataTable>
    </SectionPanel>

    <NAlert v-if="result" type="success" :bordered="false" data-testid="result">{{ result }}</NAlert>
    <NAlert v-if="error" type="error" :bordered="false" data-testid="command-error">{{ problemText(error) }}</NAlert>
  </div>
</template>

<style scoped>
.shift,
.posts {
  display: flex;
  flex-direction: column;
  gap: var(--ant-gap);
  min-width: 0;
}

.line {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-1) var(--ant-space-2);
  align-items: center;
  min-width: 0;
}

/* Список выбора растягивается по строке и не уже боковой панели стола. */
.picker {
  flex: 1 1 var(--ant-w-side-min);
  min-width: 0;
  max-width: var(--ant-w-side);
}

.grow {
  flex: 1 1 var(--ant-w-queue-min);
  min-width: 0;
}

.error {
  color: var(--ant-status-danger-text);
}
</style>
