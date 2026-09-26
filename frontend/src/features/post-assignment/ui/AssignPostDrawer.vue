<script setup lang="ts">
/**
 * Назначение на пост в смену — правое окно (Д-70; разбор стола мастера):
 * подходящие по роли и области сотрудники с вердиктом квалификации на дату смены
 * (действует / истекает / истекла / нет в области) — от сервера
 * (`access.candidate.list`), допущенные сверху; назначить исполнителя
 * (`access.assignment.set`); контролёр ОТК — по согласованию начальника ОТК
 * (запрос документом, PRD §11.18). Обойти квалификацию нельзя: сервер скажет «почему».
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NAlert, NInput } from 'naive-ui'
import { useSession } from '@/entities/session'
import { CONTROLLER_ASSIGNMENT_TEMPLATE, controllerDecision, useAssignmentCommand, useCandidates, useControllerAssignmentRequest } from '@/entities/workplace'
import type { WorkplaceCandidate } from '@/shared/api/generated/model'
import { useProblemText } from '@/shared/i18n/problem'
import { newCommandId } from '@/shared/lib/command-id'
import { ActionButton, RecordDrawer } from '@/shared/ui'

const props = defineProps<{
  show: boolean
  workplaceId: string | null
  workplaceTitle: string
  shiftId: string | null
  shiftTitle?: string | null
  canAct?: boolean
  /** Закрытый документ согласования начальника ОТК — контролёра можно назначить по нему. */
  approvalDocumentId?: string | null
}>()
const emit = defineEmits<{ close: [] }>()
const { t, d } = useI18n()
const problemText = useProblemText()
const session = useSession()

const listQ = useCandidates(() => (props.show ? props.workplaceId : null), () => props.shiftId)
const list = computed(() => (listQ.data.value?.status === 200 ? listQ.data.value.data : null))
const order = (c: WorkplaceCandidate) => (c.assigned_here ? 0 : c.allowed ? 1 : 2)
const byRole = (role: string) => [...(list.value?.items ?? [])].filter((c) => c.role === role).sort((a, b) => order(a) - order(b))
const performers = computed(() => byRole('performer'))
const inspectors = computed(() => byRole('quality_inspector'))

const pick = ref<string | null>(null)
const inspectorPick = ref<string | null>(null)
const comment = ref('')
const result = ref<string | null>(null)
watch(
  () => [props.workplaceId, props.show],
  () => {
    pick.value = null
    inspectorPick.value = null
    comment.value = ''
    result.value = null
  },
)

const assignCmd = useAssignmentCommand()
const requestCmd = useControllerAssignmentRequest()
const busy = computed(() => assignCmd.isPending.value || requestCmd.isPending.value)
const error = computed(() => assignCmd.error.value ?? requestCmd.error.value ?? null)
function meta() {
  const s = session.data.value?.data
  return { command_id: newCommandId(), policy_seq: s?.policy_seq ?? 0, basis_seq: list.value?.basis_seq ?? 0, ...(s?.workplace?.id ? { workplace_id: s.workplace.id } : {}) }
}
const nameOf = (id: string | null) => list.value?.items.find((c) => c.person_id === id)?.display ?? id ?? ''

async function assign(): Promise<void> {
  if (!pick.value || !props.workplaceId || !props.shiftId) return
  assignCmd.reset()
  const res = await assignCmd
    .mutateAsync({ kind: 'set', body: { ...meta(), assignee_role: 'performer', person_id: pick.value, workplace_id: props.workplaceId, shift_id: props.shiftId } })
    .catch(() => null)
  if (res) result.value = t('features.postAssignment.assigned', { person: nameOf(pick.value), post: props.workplaceTitle })
}

async function assignInspector(): Promise<void> {
  if (!inspectorPick.value || !props.workplaceId || !props.shiftId || !props.approvalDocumentId) return
  assignCmd.reset()
  const res = await assignCmd
    .mutateAsync({
      kind: 'set',
      body: { ...meta(), assignee_role: 'quality_inspector', person_id: inspectorPick.value, workplace_id: props.workplaceId, shift_id: props.shiftId, approval_document_id: props.approvalDocumentId },
    })
    .catch(() => null)
  if (res) result.value = t('features.postAssignment.assigned', { person: nameOf(inspectorPick.value), post: props.workplaceTitle })
}

async function requestInspector(): Promise<void> {
  if (!inspectorPick.value || !props.workplaceId || !props.shiftId) return
  requestCmd.reset()
  const res = await requestCmd
    .mutateAsync({
      ...meta(),
      template: CONTROLLER_ASSIGNMENT_TEMPLATE,
      subject: { entity: 'workplace', id: props.workplaceId },
      decision: controllerDecision(inspectorPick.value, props.shiftId),
      ...(comment.value.trim() ? { comment: comment.value.trim() } : {}),
    })
    .catch(() => null)
  if (res) result.value = t('features.postAssignment.requested', { person: nameOf(inspectorPick.value) })
}

const VERDICT: Record<string, { mark: string; tone: string }> = {
  ok: { mark: '✓', tone: 'ok' },
  expiring: { mark: '!', tone: 'warn' },
  expired: { mark: '⛔', tone: 'bad' },
  missing: { mark: '—', tone: 'bad' },
}
const date = (x?: string) => (x ? d(new Date(x), 'date') : '')
</script>

<template>
  <RecordDrawer
    :show="show"
    :kind-label="t('features.postAssignment.kind')"
    :number="workplaceTitle"
    :subtitle="shiftTitle ? t('features.postAssignment.shift', { shift: shiftTitle }) : ''"
    :loading="listQ.isPending.value && !list"
    data-record="post-assignment"
    @close="emit('close')"
  >
    <div class="assign" data-testid="assign-post">
      <NAlert v-if="listQ.error.value && !list" type="warning" :bordered="false">{{ problemText(listQ.error.value) }}</NAlert>
      <p v-if="!shiftId" class="warn ant-wrap">{{ t('widgets.shopFloor.shift.pickShiftFirst') }}</p>

      <section class="block" data-role="performer">
        <h4>{{ t('features.postAssignment.performer') }}</h4>
        <p v-if="list && !performers.length" class="muted ant-wrap">{{ t('features.postAssignment.noneFit') }}</p>
        <ul class="people">
          <li v-for="c in performers" :key="c.person_id">
            <label class="person" :data-allowed="c.allowed || undefined" :data-testid="`cand-${c.person_id}`">
              <input v-model="pick" type="radio" name="performer" :value="c.person_id" :disabled="!c.allowed || !canAct || busy" />
              <span class="mark" :data-tone="VERDICT[c.qualification_verdict]?.tone">{{ VERDICT[c.qualification_verdict]?.mark }}</span>
              <span class="who ant-wrap">
                <strong>{{ c.display }}</strong>
                <span v-if="c.assigned_here" class="muted"> · {{ t('features.postAssignment.here') }}</span>
                <span v-else-if="c.assigned_elsewhere" class="muted"> · {{ t('features.postAssignment.elsewhere', { post: c.assigned_elsewhere }) }}</span>
                <span v-if="c.valid_until" class="muted"> · {{ t('features.postAssignment.validUntil', { date: date(c.valid_until) }) }}</span>
                <span class="why">{{ c.why }}</span>
              </span>
            </label>
          </li>
        </ul>
      </section>

      <section v-if="inspectors.length" class="block" data-role="inspector">
        <h4>{{ t('features.postAssignment.inspector') }}</h4>
        <p class="muted ant-wrap">{{ t('widgets.shopFloor.shift.controllerRule') }}</p>
        <ul class="people">
          <li v-for="c in inspectors" :key="c.person_id">
            <label class="person" :data-allowed="c.allowed || c.needs_approval || undefined">
              <input v-model="inspectorPick" type="radio" name="inspector" :value="c.person_id" :disabled="!canAct || busy || c.assigned_here" />
              <span class="who ant-wrap"><strong>{{ c.display }}</strong><span class="why">{{ c.why }}</span></span>
            </label>
          </li>
        </ul>
        <NInput v-if="inspectorPick" v-model:value="comment" :placeholder="t('common.words.comment')" />
      </section>

      <NAlert v-if="error" type="error" :bordered="false" data-testid="assign-error">{{ problemText(error) }}</NAlert>
      <NAlert v-else-if="result" type="success" :bordered="false" data-testid="assign-result">{{ result }}</NAlert>
    </div>
    <template #actions>
      <ActionButton type="primary" :disabled="!pick || !shiftId || !canAct" :loading="assignCmd.isPending.value" data-testid="assign-confirm" :label="t('features.postAssignment.assign')" @click="assign" />
      <ActionButton v-if="inspectorPick && approvalDocumentId" secondary :disabled="!shiftId || !canAct" :loading="assignCmd.isPending.value" data-testid="assign-inspector" :label="t('features.postAssignment.assignByDocument', { doc: approvalDocumentId })" @click="assignInspector" />
      <ActionButton v-else-if="inspectorPick" secondary :disabled="!shiftId || !canAct" :loading="requestCmd.isPending.value" data-testid="request-inspector" :label="t('features.postAssignment.requestInspector')" @click="requestInspector" />
    </template>
  </RecordDrawer>
</template>

<style scoped>
.assign,
.block {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-3);
  min-width: 0;
}

h4,
p {
  margin: 0;
}

.people {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  margin: 0;
  padding: 0;
  list-style: none;
}

.person {
  display: flex;
  gap: var(--ant-space-2);
  align-items: flex-start;
  padding: var(--ant-space-2) var(--ant-space-3);
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-md);
  cursor: pointer;
}

.person:not([data-allowed]) {
  cursor: not-allowed;
  opacity: 0.7;
}

.person:has(input:checked) {
  border-color: var(--ant-accent);
  background: var(--ant-accent-soft);
}

.mark {
  flex: none;
  width: 1.4em;
  text-align: center;
  font-weight: var(--ant-fw-bold);
}

.mark[data-tone='ok'] {
  color: var(--ant-status-success);
}

.mark[data-tone='warn'] {
  color: var(--ant-status-attention-text);
}

.mark[data-tone='bad'] {
  color: var(--ant-status-danger);
}

.who {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.why,
.muted {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.warn {
  color: var(--ant-status-attention-text);
}
</style>
