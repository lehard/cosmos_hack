<script setup lang="ts">
/**
 * Уведомления в шапке (FR-57): непрочитанное по видам — информация, тревога,
 * задача, запрос решения — и открытые задачи с отметкой (фича task-actions).
 * Монтируется только при открытии всплывающего окна — список задач не читается зря.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { NAlert, NScrollbar, NTag } from 'naive-ui'
import type { NotificationSummary } from '@/entities/notification'
import { useTasks } from '@/entities/task'
import { useDrillDown } from '@/features/drill-down'
import { TaskInbox } from '@/features/task-actions'
import { useProblemText } from '@/shared/i18n/problem'
import type { DrillRef } from '@/shared/model/drill'
import { ActionButton } from '@/shared/ui'
import { useMomentStore } from '@/shared/model/moment'

defineProps<{ summary: NotificationSummary | null }>()
const emit = defineEmits<{ close: [] }>()
const { t } = useI18n()
const router = useRouter()
const drill = useDrillDown()
const moment = useMomentStore()
const problemText = useProblemText()
const tasksQ = useTasks({ state: 'open' })
const tasks = computed(() => tasksQ.data.value?.data.items ?? null)

const KINDS = [
  { key: 'info', text: 'common.notifications.info', type: 'default' },
  { key: 'alarm', text: 'common.notifications.alarm', type: 'error' },
  { key: 'task', text: 'common.notifications.task', type: 'warning' },
  { key: 'decision_request', text: 'common.notifications.decisionRequest', type: 'info' },
] as const

function open(ref: DrillRef): void {
  if (drill.open(ref)) emit('close')
}
</script>

<template>
  <div class="notifications-panel" data-testid="notifications-panel">
    <strong>{{ t('common.header.notifications') }}</strong>
    <div v-if="summary?.by_kind" class="line">
      <NTag v-for="k in KINDS" :key="k.key" size="small" :bordered="false" :type="summary.by_kind[k.key] ? k.type : 'default'" :data-kind="k.key">
        {{ t(k.text) }}: {{ summary.by_kind[k.key] }}
      </NTag>
    </div>
    <NScrollbar class="scroll">
      <NAlert v-if="tasksQ.error.value && !tasks" type="error" :bordered="false">{{ problemText(tasksQ.error.value) }}</NAlert>
      <TaskInbox
        v-else-if="tasks"
        :tasks="tasks"
        :basis-seq="tasksQ.data.value?.data.basis_seq ?? 0"
        :can-act="!moment.isReplay"
        density="compact"
        :can-open="drill.canOpen"
        @open="open"
      />
    </NScrollbar>
    <div>
      <ActionButton text type="primary" size="small" :label="t('widgets.shopFloor.tasks.toDesk')" @click="router.push('/desk').then(() => emit('close'))" />
    </div>
  </div>
</template>

<style scoped>
.notifications-panel {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
  width: min(var(--ant-w-side), 90vw);
  min-width: 0;
}

.line {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-1) var(--ant-space-2);
}

.scroll {
  max-height: 60vh;
}
</style>
