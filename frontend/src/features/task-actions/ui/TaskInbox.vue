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
import { NAlert, NButton, NEllipsis, NFlex, NInput, NList, NListItem, NTag, NText } from 'naive-ui'
import { useSession } from '@/entities/session'
import { isIsolatorMoveTask, isOpenTask, taskNotificationKind, useAcknowledgeTask, type AcknowledgeTaskOutcome, type TaskEntry } from '@/entities/task'
import type { DrillRef } from '@/shared/model/drill'
import { naiveSizeOf, type Density } from '@/shared/config/widget'
import { useProblemText } from '@/shared/i18n/problem'
import { newCommandId } from '@/shared/lib/command-id'
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
const small = computed(() => (props.density === 'large' ? 'medium' : 'small'))

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
  <NText v-if="!tasks.length" depth="3" data-testid="task-inbox">{{ t('empty.noTasks') }}</NText>
  <NList v-else :show-divider="true" data-testid="task-inbox">
    <NListItem
      v-for="task in tasks"
      :key="task.task_id"
      :data-task="task.task_id"
      :data-kind="task.kind"
      :data-state="task.state"
      :data-overdue="task.overdue || undefined"
    >
      <NFlex vertical :size="4">
        <NFlex :size="8" align="center" :wrap="true">
          <NTag :size="small" :bordered="false" :type="taskNotificationKind(task.kind) === 'decision_request' ? 'info' : 'default'">{{ t(kindKey(task)) }}</NTag>
          <NTag :size="small" :bordered="false" :type="task.state === 'open' ? (task.overdue ? 'error' : 'warning') : 'default'">{{ t(stateKey(task)) }}</NTag>
          <NText v-if="task.due_at" depth="3">{{ t('common.words.deadline') }}: {{ time(task.due_at) }}</NText>
        </NFlex>
        <NEllipsis :line-clamp="2" :tooltip="{ width: 360 }" data-testid="task-title">{{ task.title }}</NEllipsis>
        <div v-if="task.ref && canOpen(task.ref)">
          <NButton text type="primary" :size="size" data-testid="open-ref" @click="emit('open', task.ref)">
            {{ task.ref.entity === 'item' ? t('common.actions.openPassport') : t('common.actions.open') }} · {{ task.ref.id }}
          </NButton>
        </div>

        <template v-if="isOpenTask(task) && canAct">
          <template v-if="isIsolatorMoveTask(task)">
            <NButton v-if="moving !== task.task_id" :size="size" type="primary" secondary data-testid="open-isolator-move" @click="moving = task.task_id">
              <NEllipsis>{{ t('decisions.containment.confirmIsolatorMove') }}</NEllipsis>
            </NButton>
            <IsolatorMoveConfirm v-else :item-id="task.ref!.id" :density="density" :can-act="canAct" verbose />
          </template>
          <NFlex v-else :size="8" :wrap="true">
            <NButton :size="size" type="primary" secondary :loading="ack.isPending.value && ack.variables.value?.task_id === task.task_id" data-testid="ack-done" @click="acknowledge(task, 'done')">
              {{ t('widgets.shopFloor.tasks.outcome.done') }}
            </NButton>
            <NButton :size="size" secondary data-testid="ack-accepted" @click="acknowledge(task, 'accepted')">{{ t('widgets.shopFloor.tasks.outcome.accepted') }}</NButton>
            <NButton :size="size" quaternary data-testid="ack-decline" @click="declining = declining === task.task_id ? null : task.task_id">
              {{ t('widgets.shopFloor.tasks.outcome.declined') }}
            </NButton>
          </NFlex>
          <form v-if="declining === task.task_id" @submit.prevent="acknowledge(task, 'declined')">
            <NFlex :size="8" align="center" :wrap="true">
              <NInput
                v-model:value="notes[task.task_id]"
                :size="size"
                :placeholder="t('widgets.shopFloor.tasks.declineNote')"
                :aria-label="t('widgets.shopFloor.tasks.declineNote')"
                data-testid="decline-note"
              />
              <NButton :size="size" type="warning" attr-type="submit" :disabled="!(notes[task.task_id] ?? '').trim()" data-testid="confirm-decline">
                {{ t('widgets.shopFloor.tasks.outcome.declined') }}
              </NButton>
            </NFlex>
          </form>
        </template>
        <NText v-if="acked[task.task_id]" type="success" data-testid="acked">{{ t('widgets.shopFloor.recorded', { seq: acked[task.task_id] }) }}</NText>
        <NAlert v-if="lastError?.task === task.task_id" type="error" :bordered="false" data-testid="ack-error">{{ problemText(lastError.error) }}</NAlert>
      </NFlex>
    </NListItem>
  </NList>
</template>
