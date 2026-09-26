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
import { isIsolatorMoveTask, isOpenTask, taskNotificationKind, useAcknowledgeTask, type AcknowledgeTaskOutcome, type TaskEntry } from '@/entities/task'
import type { DrillRef } from '@/shared/model/drill'
import { naiveSizeOf, type Density } from '@/shared/config/widget'
import { useProblemText } from '@/shared/i18n/problem'
import { newCommandId } from '@/shared/lib/command-id'
import { ActionButton, EmptyState } from '@/shared/ui'
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
const acked = reactive<Record<string, number>>({})

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
    acked[task.task_id] = res.data.seq
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
      <div v-if="task.ref && canOpen(task.ref)" class="line">
        <ActionButton
          text
          type="primary"
          :size="size"
          :label="`${task.ref.entity === 'item' ? t('common.actions.openPassport') : t('common.actions.open')} · ${task.ref.id}`"
          data-testid="open-ref"
          @click="emit('open', task.ref)"
        />
      </div>

      <template v-if="isOpenTask(task) && canAct">
        <template v-if="isIsolatorMoveTask(task)">
          <div v-if="moving !== task.task_id" class="line">
            <ActionButton :size="size" type="primary" secondary :label="t('decisions.containment.confirmIsolatorMove')" data-testid="open-isolator-move" @click="moving = task.task_id" />
          </div>
          <IsolatorMoveConfirm v-else :item-id="task.ref!.id" :density="density" :can-act="canAct" verbose />
        </template>
        <div v-else class="line">
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
      <p v-if="acked[task.task_id]" class="ok ant-wrap" data-testid="acked">{{ t('widgets.shopFloor.recorded', { seq: acked[task.task_id] }) }}</p>
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
  color: var(--ant-status-success-text);
}
</style>
