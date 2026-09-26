<script setup lang="ts">
/**
 * Уведомления в шапке (FR-57): непрочитанное по видам — информация, тревога,
 * задача, запрос решения — и открытые задачи с отметкой (фича task-actions).
 * Монтируется только при открытии всплывающего окна — список задач не читается зря.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { NButton, NFlex, NScrollbar, NTag, NText } from 'naive-ui'
import type { NotificationSummary } from '@/entities/notification'
import { useTasks } from '@/entities/task'
import { useDrillDown } from '@/features/drill-down'
import { TaskInbox } from '@/features/task-actions'
import { useProblemText } from '@/shared/i18n/problem'
import type { DrillRef } from '@/shared/model/drill'
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
  <NFlex vertical :size="8" class="notifications-panel" data-testid="notifications-panel">
    <NText strong>{{ t('common.header.notifications') }}</NText>
    <NFlex v-if="summary?.by_kind" :size="6" :wrap="true">
      <NTag v-for="k in KINDS" :key="k.key" size="small" :bordered="false" :type="summary.by_kind[k.key] ? k.type : 'default'" :data-kind="k.key">
        {{ t(k.text) }}: {{ summary.by_kind[k.key] }}
      </NTag>
    </NFlex>
    <NScrollbar style="max-height: 420px">
      <NText v-if="tasksQ.error.value && !tasks" type="error">{{ problemText(tasksQ.error.value) }}</NText>
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
    <NButton text type="primary" size="small" @click="router.push('/desk').then(() => emit('close'))">{{ t('widgets.shopFloor.tasks.toDesk') }}</NButton>
  </NFlex>
</template>

<style scoped>
.notifications-panel {
  width: min(420px, 90vw);
}
</style>
