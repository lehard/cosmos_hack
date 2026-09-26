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
import { PRESENCE_TEXT, presenceTagType, type AccessAssignment, type DocumentSummary } from '@/entities/workplace'
import type { Density } from '@/shared/config/widget'
import { naiveSizeOf } from '@/shared/config/widget'
import { useProblemText } from '@/shared/i18n/problem'
import { ActionButton, EmptyState, SectionPanel, ToolBar } from '@/shared/ui'
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
    canAct?: boolean
    busy?: boolean
    error?: unknown
    /** Итог последней команды (запись журнала). */
    result?: string | null
    density?: Density
  }>(),
  { shiftsError: undefined, rowsError: undefined, assignmentsError: undefined, canAct: true, busy: false, error: undefined, result: null, density: 'comfortable' },
)
const emit = defineEmits<{
  'update:shiftId': [id: string]
  assign: [workplaceId: string, role: AssigneeRole, personId: string, approvalDocumentId: string | null]
  clear: [a: AccessAssignment, reason: string]
  requestController: [workplaceId: string, personId: string, comment: string]
}>()

const { t, d } = useI18n()
const problemText = useProblemText()
const size = computed(() => naiveSizeOf(props.density))
const time = (iso: string) => d(new Date(iso), 'time')

const shiftOptions = computed(() => (props.shifts ?? []).map((s) => ({ label: shiftLabel(s, time), value: s.shift_id })))

/** Подпись кандидата с отметкой квалификации; истекла — выбрать нельзя. */
const options = (list: readonly Candidate[]) =>
  list.map((c) => ({
    label: c.verdict === 'expired' || c.verdict === 'expiring' ? `${c.person.display_name} — ${t(`widgets.shopFloor.shift.verdict.${c.verdict}`)}` : c.person.display_name,
    value: c.person.person_id,
    disabled: c.verdict === 'expired',
  }))
const performerOptions = computed(() => options(props.performers))
const inspectorOptions = computed(() => options(props.inspectors))

/** Черновики по постам: кого назначить, кого запросить в контролёры, основания. */
const pick = reactive<Record<string, string | null>>({})
const inspectorPick = reactive<Record<string, string | null>>({})
const comment = reactive<Record<string, string>>({})
const clearing = reactive<Record<string, string>>({})
const clearReason = reactive<Record<string, string>>({})
const approval = reactive<Record<string, DocumentSummary | null>>({})

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
    <NAlert v-if="assignmentsError" type="warning" :bordered="false" data-testid="assignments-error">
      {{ t('widgets.shopFloor.shift.planUnavailable') }}: {{ problemText(assignmentsError) }}
    </NAlert>

    <SectionPanel :title="t('access.shiftsAndAssignments')" variant="plain">
      <NAlert v-if="rowsError && !rows" type="error" :bordered="false">{{ problemText(rowsError) }}</NAlert>
      <EmptyState v-else-if="rows && !rows.length" compact :title="t('empty.noRecords')" />
      <div v-else-if="rows" class="posts">
        <SectionPanel v-for="r in rows" :key="r.post.workplace_id" :title="r.post.station" variant="subtle" :data-workplace="r.post.workplace_id">
          <template #extra>
            <span class="ant-muted">{{ t('widgets.shopFloor.shift.fact') }}:</span>
            <span class="ant-wrap">{{ r.post.assigned?.display ?? t('liveMap.posts.notAssigned') }}</span>
            <NTag size="small" :bordered="false" :type="presenceTagType(r.post.presence)"><span class="ant-wrap">{{ t(PRESENCE_TEXT[r.post.presence]) }}</span></NTag>
          </template>

          <!-- Исполнители: план смены. -->
          <div class="line" data-testid="performers">
            <span class="ant-muted">{{ t('widgets.shopFloor.shift.performer') }}:</span>
            <span v-if="!r.performers.length" class="ant-muted">{{ t('liveMap.posts.notAssigned') }}</span>
            <div v-for="a in r.performers" :key="clearKey(a)" class="line" :data-assigned="a.person_id">
              <span class="ant-wrap">{{ personName(a.person_id) }}</span>
              <NTag size="small" :bordered="false" :type="a.admitted ? 'success' : 'warning'">
                {{ t(a.admitted ? 'widgets.shopFloor.people.admitted' : 'widgets.shopFloor.people.notAdmitted') }}
              </NTag>
              <NTag v-if="!a.qualification_ok" size="small" :bordered="false" type="error">{{ t('widgets.shopFloor.people.qualificationNotOk') }}</NTag>
              <template v-if="canAct">
                <ActionButton v-if="clearing[clearKey(a)] === undefined" :size="size" quaternary :label="t('widgets.shopFloor.shift.clear')" data-testid="clear" @click="clearing[clearKey(a)] = ''" />
                <form v-else class="line" @submit.prevent="confirmClear(a)">
                  <NInput v-model:value="clearReason[clearKey(a)]" class="grow" :size="size" :placeholder="t('widgets.shopFloor.shift.clearReason')" data-testid="clear-reason" />
                  <ActionButton
                    :size="size"
                    type="warning"
                    attr-type="submit"
                    :disabled="busy || !(clearReason[clearKey(a)] ?? '').trim()"
                    :label="t('widgets.shopFloor.shift.clear')"
                    data-testid="confirm-clear"
                  />
                </form>
              </template>
            </div>
          </div>
          <ToolBar v-if="canAct">
            <NSelect
              v-model:value="pick[r.post.workplace_id]"
              class="picker"
              :size="size"
              :options="performerOptions"
              :placeholder="t('access.assignToPost')"
              filterable
              clearable
              :consistent-menu-width="false"
              data-testid="performer-pick"
            />
            <ActionButton
              :size="size"
              type="primary"
              :disabled="busy || !pick[r.post.workplace_id] || !shiftId"
              :label="t('access.assignToPost')"
              data-testid="assign-performer"
              @click="emit('assign', r.post.workplace_id, 'performer', pick[r.post.workplace_id]!, null)"
            />
          </ToolBar>

          <!-- Контролёр: запрос мастера → согласование начальника ОТК (PRD §11.18). -->
          <div class="line" data-testid="inspectors">
            <span class="ant-muted">{{ t('widgets.shopFloor.shift.inspector') }}:</span>
            <span v-if="!r.inspectors.length" class="ant-muted">{{ t('liveMap.posts.notAssigned') }}</span>
            <div v-for="a in r.inspectors" :key="clearKey(a)" class="line" :data-assigned="a.person_id">
              <span class="ant-wrap">{{ personName(a.person_id) }}</span>
              <span v-if="a.approval_document_id" class="ant-muted ant-wrap">{{ t('widgets.shopFloor.shift.byDocument', { doc: a.approval_document_id }) }}</span>
            </div>
          </div>
          <PostApprovals :workplace-id="r.post.workplace_id" :can-act="canAct" :size="size" @use="(doc) => (approval[r.post.workplace_id] = doc)" />
          <ToolBar v-if="canAct">
            <NSelect
              v-model:value="inspectorPick[r.post.workplace_id]"
              class="picker"
              :size="size"
              :options="inspectorOptions"
              :placeholder="t('widgets.shopFloor.shift.inspectorPick')"
              filterable
              clearable
              :consistent-menu-width="false"
              data-testid="inspector-pick"
            />
            <ActionButton
              v-if="approval[r.post.workplace_id]"
              :size="size"
              type="primary"
              :disabled="busy || !inspectorPick[r.post.workplace_id] || !shiftId"
              :label="t('widgets.shopFloor.shift.assignByDocument', { doc: approval[r.post.workplace_id]!.document_id })"
              data-testid="assign-inspector"
              @click="emit('assign', r.post.workplace_id, 'quality_inspector', inspectorPick[r.post.workplace_id]!, approval[r.post.workplace_id]!.document_id)"
            />
            <template v-else>
              <NInput v-model:value="comment[r.post.workplace_id]" class="grow" :size="size" :placeholder="t('common.words.comment')" data-testid="request-comment" />
              <ActionButton
                :size="size"
                secondary
                type="primary"
                :disabled="busy || !inspectorPick[r.post.workplace_id] || !shiftId"
                :label="t('widgets.shopFloor.shift.requestInspector')"
                data-testid="request-inspector"
                @click="emit('requestController', r.post.workplace_id, inspectorPick[r.post.workplace_id]!, (comment[r.post.workplace_id] ?? '').trim())"
              />
            </template>
          </ToolBar>
        </SectionPanel>
      </div>
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
