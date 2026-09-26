<script setup lang="ts">
/**
 * Задачи и запросы решения (FR-57): владелец, срок, подтверждение. Открытая
 * задача отмечается «выполнено», «принято» или «отклонено с примечанием»
 * (`notifications.task.acknowledge`); задачу «перенести в изолятор» закрывает
 * не отметка, а подтверждённая приёмка в изоляторе (FR-55) — её основание
 * снимается само, задача уходит из открытых.
 */
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NAlert, NInput, NTag } from 'naive-ui'
import { useSession } from '@/entities/session'
import { isOpenTask, taskNotificationKind, useAcknowledgeTask, type AcknowledgeTaskOutcome, type TaskEntry } from '@/entities/task'
import { ReceiveAction } from '@/features/item-receive'
import type { DrillRef } from '@/shared/model/drill'
import { naiveSizeOf, type Density } from '@/shared/config/widget'
import { useProblemText } from '@/shared/i18n/problem'
import { newCommandId } from '@/shared/lib/command-id'
import { ActionButton, EmptyState } from '@/shared/ui'
import { taskActionOf, taskItemLabel, type ProcessTask, type TaskAction } from '../model/actions'
import IsolatorMoveConfirm from './IsolatorMoveConfirm.vue'

const props = withDefaults(
  defineProps<{
    tasks: readonly TaskEntry[]
    /** `basis_seq` отметки — seq, на котором прочитан список (AD-39). */
    basisSeq: number
    canAct?: boolean
    density?: Density
    /** Можно ли открыть объект задачи (есть экран). */
    canOpen?: (ref: DrillRef) => boolean
  }>(),
  { canAct: true, density: 'comfortable', canOpen: () => true },
)
const emit = defineEmits<{ open: [ref: DrillRef] }>()

const { t, d } = useI18n()
const problemText = useProblemText()
const session = useSession()
const ack = useAcknowledgeTask()
const size = computed(() => naiveSizeOf(props.density))

/** Раскрыто подтверждение перемещения у задачи. */
const moving = ref<string | null>(null)
/** Отклонение — с примечанием: id задачи → текст. */
const declining = ref<string | null>(null)
const notes = reactive<Record<string, string>>({})
/** id команды на задачу — один на намерение (AD-7). */
const commandIds = new Map<string, string>()
const lastError = ref<{ task: string; error: unknown } | null>(null)
/** Отмеченные в этом сеансе: исход и номер записи — кнопки у задачи прячутся, итог виден (UI-37). */
const acked = reactive<Record<string, { outcome: AcknowledgeTaskOutcome; seq: number }>>({})

/** Действие каждой задачи — по operationId из реестра (model/actions.ts). */
const actions = computed(() => new Map<string, TaskAction>(props.tasks.map((task) => [task.task_id, taskActionOf(task as ProcessTask)])))
const actionOf = (task: TaskEntry): TaskAction => actions.value.get(task.task_id) ?? { kind: 'ack', ref: task.ref ?? null }
const itemLabel = (task: TaskEntry) => taskItemLabel(task as ProcessTask)
const opOf = (task: TaskEntry) => (task as ProcessTask).operation ?? task.kind

/**
 * Действие исполнителя — на его терминале: если терминал на этом столе, кнопка
 * ведёт к нему; иначе открывается окно изделия.
 */
function goTerminal(a: Extract<TaskAction, { kind: 'terminal' }>): void {
  const el = typeof document !== 'undefined' ? document.querySelector<HTMLElement>('[data-widget="performer-terminal"]') : null
  if (el) {
    el.scrollIntoView({ behavior: 'smooth', block: 'start' })
    return
  }
  if (a.ref) emit('open', a.ref)
}

const time = (iso: string | null | undefined) => (iso ? d(new Date(iso), 'dateTime') : '')

async function acknowledge(task: TaskEntry, outcome: AcknowledgeTaskOutcome): Promise<void> {
  const note = (notes[task.task_id] ?? '').trim()
  if (outcome === 'declined' && !note) return
  const s = session.data.value?.data
  const key = `${task.task_id}/${outcome}`
  const commandId = commandIds.get(key) ?? newCommandId()
  commandIds.set(key, commandId)
  lastError.value = null
  try {
    const res = await ack.mutateAsync({
      task_id: task.task_id,
      body: {
        command_id: commandId,
        basis_seq: props.basisSeq,
        policy_seq: s?.policy_seq ?? 0,
        outcome,
        ...(note ? { note } : {}),
        ...(s?.workplace?.id ? { workplace_id: s.workplace.id } : {}),
      },
    })
    acked[task.task_id] = { outcome, seq: res.data.seq }
    declining.value = null
    commandIds.delete(key)
  } catch (error) {
    lastError.value = { task: task.task_id, error }
  }
}

const stateKey = (task: TaskEntry) => (task.state === 'open' && task.overdue ? 'statuses.task.overdue' : `widgets.shopFloor.tasks.state.${task.state}`)
const kindKey = (task: TaskEntry) => (taskNotificationKind(task.kind) === 'decision_request' ? 'common.notifications.decisionRequest' : 'common.notifications.task')
</script>

<template>
  <EmptyState v-if="!tasks.length" compact :title="t('empty.noTasks')" data-testid="task-inbox" />
  <ul v-else class="task-inbox" data-testid="task-inbox">
    <li
      v-for="task in tasks"
      :key="task.task_id"
      class="task"
      :data-task="task.task_id"
      :data-kind="task.kind"
      :data-state="task.state"
      :data-overdue="task.overdue || undefined"
    >
      <div class="line">
        <NTag size="small" :bordered="false" :type="taskNotificationKind(task.kind) === 'decision_request' ? 'info' : 'default'">{{ t(kindKey(task)) }}</NTag>
        <NTag size="small" :bordered="false" :type="task.state === 'open' ? (task.overdue ? 'error' : 'warning') : 'default'">{{ t(stateKey(task)) }}</NTag>
        <span v-if="task.due_at" class="ant-muted">{{ t('common.words.deadline') }}: {{ time(task.due_at) }}</span>
      </div>
      <p class="ant-clamp-2" :title="task.title" data-testid="task-title">{{ task.title }}</p>
      <p v-if="itemLabel(task) && !task.title.includes(itemLabel(task)!)" class="ant-muted ant-ellipsis" data-testid="task-item">
        {{ t('common.words.item') }}: {{ itemLabel(task) }}
      </p>

      <!-- Своё действие задачи — глаголом (реестр operationId → форма или окно). -->
      <template v-if="isOpenTask(task) && canAct && !acked[task.task_id] && actionOf(task).kind !== 'ack'">
        <template v-for="a in [actionOf(task)]" :key="a.kind">
          <ReceiveAction v-if="a.kind === 'form' && a.form === 'receive'" :item-id="a.itemId" :basis-seq="basisSeq" data-testid="task-receive" />
          <template v-else-if="a.kind === 'form' && a.form === 'isolator_move'">
            <div v-if="moving !== task.task_id" class="line">
              <ActionButton :size="size" type="primary" secondary :label="t(a.verbKey)" data-testid="open-isolator-move" @click="moving = task.task_id" />
            </div>
            <IsolatorMoveConfirm v-else :item-id="a.itemId" :density="density" :can-act="canAct" verbose />
          </template>
          <div v-else-if="a.kind === 'window' && canOpen(a.ref)" class="line">
            <ActionButton :size="size" type="primary" :label="t(a.verbKey)" data-testid="task-action" :data-action="opOf(task)" @click="emit('open', a.ref)" />
          </div>
          <div v-else-if="a.kind === 'terminal'" class="line">
            <ActionButton :size="size" type="primary" :label="t(a.verbKey)" data-testid="task-action" :data-action="opOf(task)" @click="goTerminal(a)" />
          </div>
        </template>
      </template>

      <!-- Паспорт изделия — ссылкой, если действие само окно не открывает. -->
      <div v-if="task.ref && canOpen(task.ref) && (actionOf(task).kind === 'ack' || actionOf(task).kind === 'form')" class="line">
        <ActionButton
          text
          type="primary"
          :size="size"
          :label="task.ref.entity === 'item' ? t('common.actions.openPassport') : t('taskActions.verb.goTo')"
          data-testid="open-ref"
          @click="emit('open', task.ref)"
        />
      </div>

      <template v-if="isOpenTask(task) && canAct && !acked[task.task_id] && actionOf(task).kind === 'ack'">
        <div class="line">
          <ActionButton
            :size="size"
            type="primary"
            secondary
            :loading="ack.isPending.value && ack.variables.value?.task_id === task.task_id"
            :label="t('widgets.shopFloor.tasks.outcome.done')"
            data-testid="ack-done"
            @click="acknowledge(task, 'done')"
          />
          <ActionButton :size="size" secondary :label="t('widgets.shopFloor.tasks.outcome.accepted')" data-testid="ack-accepted" @click="acknowledge(task, 'accepted')" />
          <ActionButton
            :size="size"
            quaternary
            :label="t('widgets.shopFloor.tasks.outcome.declined')"
            data-testid="ack-decline"
            @click="declining = declining === task.task_id ? null : task.task_id"
          />
        </div>
        <form v-if="declining === task.task_id" class="line" @submit.prevent="acknowledge(task, 'declined')">
          <NInput
            v-model:value="notes[task.task_id]"
            class="grow"
            :size="size"
            :placeholder="t('widgets.shopFloor.tasks.declineNote')"
            :aria-label="t('widgets.shopFloor.tasks.declineNote')"
            data-testid="decline-note"
          />
          <ActionButton
            :size="size"
            type="warning"
            attr-type="submit"
            :disabled="!(notes[task.task_id] ?? '').trim()"
            :label="t('widgets.shopFloor.tasks.outcome.declined')"
            data-testid="confirm-decline"
          />
        </form>
      </template>
      <!-- Итог отметки виден сразу (UI-37): что вы отметили; номер записи — мелко. -->
      <p v-if="acked[task.task_id]" class="ok ant-wrap" data-testid="acked">
        {{ t('widgets.shopFloor.tasks.ackedAs', { outcome: t(`widgets.shopFloor.tasks.state.${acked[task.task_id]!.outcome}`) }) }}
        <span class="ant-muted">· {{ t('widgets.shopFloor.recorded', { seq: acked[task.task_id]!.seq }) }}</span>
      </p>
      <NAlert v-if="lastError?.task === task.task_id" type="error" :bordered="false" data-testid="ack-error">{{ problemText(lastError.error) }}</NAlert>
    </li>
  </ul>
</template>

<style scoped>
.task-inbox {
  display: flex;
  flex-direction: column;
  min-width: 0;
  margin: 0;
  padding: 0;
  list-style: none;
}

.task {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  min-width: 0;
  padding: var(--ant-space-3) 0;
  border-bottom: 1px solid var(--ant-border);
}

.task:first-child {
  padding-top: 0;
}

.task:last-child {
  border-bottom: 0;
}

.line {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-1) var(--ant-space-2);
  align-items: center;
  min-width: 0;
}

.grow {
  flex: 1 1 var(--ant-w-queue-min);
  min-width: 0;
}

.ok {
  padding: var(--ant-space-2) var(--ant-space-3);
  border-radius: var(--ant-radius-md);
  background: var(--ant-status-success-soft);
  color: var(--ant-status-success-text);
  font-weight: var(--ant-fw-bold);
}
</style>
