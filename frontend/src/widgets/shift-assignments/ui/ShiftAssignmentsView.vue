<script setup lang="ts">
/**
 * Смена — представление (PRD §3a «Смена», FR-81, PRD §11.18, UJ-7): план и
 * факт смены по постам — кто назначен (исполнитель, контролёр), допуск и
 * квалификация, на месте ли. Исполнителя назначает мастер — только допущенного
 * по квалификации (окончательно решает сервер); контролёра — запросом с
 * согласованием начальника ОТК, назначение — по закрытому маршруту документа.
 */
import { computed, reactive } from 'vue'
import { useI18n } from 'vue-i18n'
import { NAlert, NButton, NDivider, NEllipsis, NFlex, NInput, NList, NListItem, NSelect, NTag, NText } from 'naive-ui'
import type { RefShift } from '@/entities/reference'
import { PRESENCE_TEXT, presenceTagType, type AccessAssignment, type DocumentSummary } from '@/entities/workplace'
import type { Density } from '@/shared/config/widget'
import { naiveSizeOf } from '@/shared/config/widget'
import { useProblemText } from '@/shared/i18n/problem'
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
const tagSize = computed(() => (props.density === 'large' ? 'medium' : 'small'))
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
  <NFlex vertical :size="12" data-testid="shift-view">
    <NFlex :size="8" align="center" :wrap="true">
      <NText depth="3">
        {{ workshopName ? t('widgets.shopFloor.station.workshop', { name: workshopName }) : t('widgets.shopFloor.station.noWorkshop') }}
      </NText>
      <NText depth="3">·</NText>
      <NText>{{ t('common.words.shift') }}:</NText>
      <NSelect
        v-if="shiftOptions.length"
        :value="shiftId"
        :options="shiftOptions"
        :size="size"
        :consistent-menu-width="false"
        style="min-width: 220px; max-width: 100%"
        data-testid="shift"
        @update:value="(v: string) => emit('update:shiftId', v)"
      />
      <NText v-else-if="shiftsError" type="error">{{ problemText(shiftsError) }}</NText>
      <NText v-else depth="3">{{ shiftId ?? '—' }}</NText>
    </NFlex>
    <NText depth="3">{{ t('access.onlyQualified') }} · {{ t('widgets.shopFloor.shift.controllerRule') }}</NText>
    <NAlert v-if="assignmentsError" type="warning" :bordered="false" data-testid="assignments-error">
      {{ t('widgets.shopFloor.shift.planUnavailable') }}: {{ problemText(assignmentsError) }}
    </NAlert>

    <NDivider title-placement="left">{{ t('access.shiftsAndAssignments') }}</NDivider>
    <NText v-if="rowsError && !rows" type="error">{{ problemText(rowsError) }}</NText>
    <NText v-else-if="rows && !rows.length" depth="3">{{ t('empty.noRecords') }}</NText>
    <NList v-else-if="rows" :show-divider="true">
      <NListItem v-for="r in rows" :key="r.post.workplace_id" :data-workplace="r.post.workplace_id">
        <NFlex vertical :size="6">
          <NFlex :size="8" align="center" :wrap="true">
            <NText strong><NEllipsis :tooltip="{ width: 360 }">{{ r.post.station }}</NEllipsis></NText>
            <NText depth="3">{{ t('widgets.shopFloor.shift.fact') }}:</NText>
            <NText>{{ r.post.assigned?.display ?? t('liveMap.posts.notAssigned') }}</NText>
            <NTag :size="tagSize" :bordered="false" :type="presenceTagType(r.post.presence)">{{ t(PRESENCE_TEXT[r.post.presence]) }}</NTag>
          </NFlex>

          <!-- Исполнители: план смены. -->
          <NFlex :size="6" align="center" :wrap="true" data-testid="performers">
            <NText depth="3">{{ t('widgets.shopFloor.shift.performer') }}:</NText>
            <NText v-if="!r.performers.length" depth="3">{{ t('liveMap.posts.notAssigned') }}</NText>
            <NFlex v-for="a in r.performers" :key="clearKey(a)" :size="6" align="center" :wrap="true" :data-assigned="a.person_id">
              <NText>{{ personName(a.person_id) }}</NText>
              <NTag :size="tagSize" :bordered="false" :type="a.admitted ? 'success' : 'warning'">
                {{ t(a.admitted ? 'widgets.shopFloor.people.admitted' : 'widgets.shopFloor.people.notAdmitted') }}
              </NTag>
              <NTag v-if="!a.qualification_ok" :size="tagSize" :bordered="false" type="error">{{ t('widgets.shopFloor.people.qualificationNotOk') }}</NTag>
              <template v-if="canAct">
                <NButton v-if="clearing[clearKey(a)] === undefined" :size="size" quaternary data-testid="clear" @click="clearing[clearKey(a)] = ''">
                  {{ t('widgets.shopFloor.shift.clear') }}
                </NButton>
                <form v-else @submit.prevent="confirmClear(a)">
                  <NFlex :size="6" align="center" :wrap="true">
                    <NInput v-model:value="clearReason[clearKey(a)]" :size="size" :placeholder="t('widgets.shopFloor.shift.clearReason')" data-testid="clear-reason" />
                    <NButton :size="size" type="warning" attr-type="submit" :disabled="busy || !(clearReason[clearKey(a)] ?? '').trim()" data-testid="confirm-clear">
                      {{ t('widgets.shopFloor.shift.clear') }}
                    </NButton>
                  </NFlex>
                </form>
              </template>
            </NFlex>
          </NFlex>
          <NFlex v-if="canAct" :size="6" align="center" :wrap="true">
            <NSelect
              v-model:value="pick[r.post.workplace_id]"
              :size="size"
              :options="performerOptions"
              :placeholder="t('access.assignToPost')"
              filterable
              clearable
              :consistent-menu-width="false"
              style="min-width: 220px; max-width: 100%; flex: 1"
              data-testid="performer-pick"
            />
            <NButton
              :size="size"
              type="primary"
              :disabled="busy || !pick[r.post.workplace_id] || !shiftId"
              data-testid="assign-performer"
              @click="emit('assign', r.post.workplace_id, 'performer', pick[r.post.workplace_id]!, null)"
            >
              {{ t('access.assignToPost') }}
            </NButton>
          </NFlex>

          <!-- Контролёр: запрос мастера → согласование начальника ОТК (PRD §11.18). -->
          <NFlex :size="6" align="center" :wrap="true" data-testid="inspectors">
            <NText depth="3">{{ t('widgets.shopFloor.shift.inspector') }}:</NText>
            <NText v-if="!r.inspectors.length" depth="3">{{ t('liveMap.posts.notAssigned') }}</NText>
            <NFlex v-for="a in r.inspectors" :key="clearKey(a)" :size="6" align="center" :wrap="true" :data-assigned="a.person_id">
              <NText>{{ personName(a.person_id) }}</NText>
              <NText v-if="a.approval_document_id" depth="3">{{ t('widgets.shopFloor.shift.byDocument', { doc: a.approval_document_id }) }}</NText>
            </NFlex>
          </NFlex>
          <PostApprovals :workplace-id="r.post.workplace_id" :can-act="canAct" :size="size" @use="(doc) => (approval[r.post.workplace_id] = doc)" />
          <NFlex v-if="canAct" :size="6" align="center" :wrap="true">
            <NSelect
              v-model:value="inspectorPick[r.post.workplace_id]"
              :size="size"
              :options="inspectorOptions"
              :placeholder="t('widgets.shopFloor.shift.inspectorPick')"
              filterable
              clearable
              :consistent-menu-width="false"
              style="min-width: 220px; max-width: 100%; flex: 1"
              data-testid="inspector-pick"
            />
            <template v-if="approval[r.post.workplace_id]">
              <NButton
                :size="size"
                type="primary"
                :disabled="busy || !inspectorPick[r.post.workplace_id] || !shiftId"
                data-testid="assign-inspector"
                @click="emit('assign', r.post.workplace_id, 'quality_inspector', inspectorPick[r.post.workplace_id]!, approval[r.post.workplace_id]!.document_id)"
              >
                {{ t('widgets.shopFloor.shift.assignByDocument', { doc: approval[r.post.workplace_id]!.document_id }) }}
              </NButton>
            </template>
            <template v-else>
              <NInput v-model:value="comment[r.post.workplace_id]" :size="size" :placeholder="t('common.words.comment')" style="flex: 1; min-width: 160px" data-testid="request-comment" />
              <NButton
                :size="size"
                secondary
                type="primary"
                :disabled="busy || !inspectorPick[r.post.workplace_id] || !shiftId"
                data-testid="request-inspector"
                @click="emit('requestController', r.post.workplace_id, inspectorPick[r.post.workplace_id]!, (comment[r.post.workplace_id] ?? '').trim())"
              >
                <NEllipsis>{{ t('widgets.shopFloor.shift.requestInspector') }}</NEllipsis>
              </NButton>
            </template>
          </NFlex>
        </NFlex>
      </NListItem>
    </NList>

    <NAlert v-if="result" type="success" :bordered="false" data-testid="result">{{ result }}</NAlert>
    <NAlert v-if="error" type="error" :bordered="false" data-testid="command-error">{{ problemText(error) }}</NAlert>
  </NFlex>
</template>
