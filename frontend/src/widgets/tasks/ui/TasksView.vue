<script setup lang="ts">
/**
 * Задачи — представление (FR-57, FR-8): сводка по видам (информация, тревога,
 * задача, запрос решения), открытые задачи и запросы решения с отметкой,
 * тревоги и просроченные решения с ценой задержки. Выполненные — свёрнуты.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NAlert, NCollapse, NCollapseItem, NTag } from 'naive-ui'
import type { NotificationSummary } from '@/entities/notification'
import type { TaskEntry } from '@/entities/task'
import { TaskInbox } from '@/features/task-actions'
import type { DrillRef } from '@/shared/model/drill'
import type { Density } from '@/shared/config/widget'
import { naiveSizeOf } from '@/shared/config/widget'
import { useProblemText } from '@/shared/i18n/problem'
import { ActionButton, EmptyState, SectionPanel } from '@/shared/ui'
import type { NoticeRow, TaskSection } from '../model/slice'

const props = withDefaults(
  defineProps<{
    sections: readonly TaskSection[]
    summary?: NotificationSummary | null
    open: readonly TaskEntry[] | null
    closed: readonly TaskEntry[]
    basisSeq: number
    tasksError?: unknown
    alerts: readonly NoticeRow[] | null
    alertsError?: unknown
    escalations: readonly NoticeRow[] | null
    escalationsError?: unknown
    canAct?: boolean
    density?: Density
    canOpen?: (ref: DrillRef) => boolean
  }>(),
  { summary: null, tasksError: undefined, alertsError: undefined, escalationsError: undefined, canAct: true, density: 'comfortable', canOpen: () => true },
)
const emit = defineEmits<{ open: [ref: DrillRef] }>()
const { t } = useI18n()
const problemText = useProblemText()
const size = computed(() => naiveSizeOf(props.density))

const KINDS = [
  { key: 'info', text: 'common.notifications.info', type: 'default' },
  { key: 'alarm', text: 'common.notifications.alarm', type: 'error' },
  { key: 'task', text: 'common.notifications.task', type: 'warning' },
  { key: 'decision_request', text: 'common.notifications.decisionRequest', type: 'info' },
] as const
</script>

<template>
  <div class="tasks" data-testid="tasks-view">
    <div v-if="summary?.by_kind" class="line" data-testid="summary">
      <NTag v-for="k in KINDS" :key="k.key" size="small" :bordered="false" :type="summary.by_kind[k.key] ? k.type : 'default'" :data-kind="k.key">
        {{ t(k.text) }}: {{ summary.by_kind[k.key] }}
      </NTag>
    </div>

    <SectionPanel v-if="sections.includes('task')" :title="t('desks.tasks')" variant="plain" data-testid="section-tasks">
      <NAlert v-if="tasksError && !open" type="error" :bordered="false">{{ problemText(tasksError) }}</NAlert>
      <TaskInbox v-else-if="open" :tasks="open" :basis-seq="basisSeq" :can-act="canAct" :density="density" :can-open="canOpen" @open="(r) => emit('open', r)" />
      <NCollapse v-if="closed.length" data-testid="closed-tasks">
        <NCollapseItem :title="t('widgets.shopFloor.tasks.closed', { n: closed.length })" name="closed">
          <TaskInbox :tasks="closed" :basis-seq="basisSeq" :can-act="false" :density="density" :can-open="canOpen" @open="(r) => emit('open', r)" />
        </NCollapseItem>
      </NCollapse>
    </SectionPanel>

    <SectionPanel v-if="sections.includes('escalation')" :title="t('widgets.shopFloor.tasks.escalations')" variant="plain" data-testid="section-escalations">
      <NAlert v-if="escalationsError && !escalations" type="error" :bordered="false">{{ problemText(escalationsError) }}</NAlert>
      <EmptyState v-else-if="escalations && !escalations.length" compact :title="t('empty.queueEmpty')" />
      <ul v-else-if="escalations" class="notices">
        <li v-for="r in escalations" :key="r.id" class="notice" :data-escalation="r.id">
          <p class="ant-clamp-2" :class="r.severe ? 'severe' : 'mild'" :title="r.text">{{ r.text }}</p>
          <div v-if="r.ref && canOpen(r.ref)">
            <ActionButton text type="primary" :size="size" :label="`${t('common.actions.open')} · ${r.ref.id}`" @click="emit('open', r.ref)" />
          </div>
        </li>
      </ul>
    </SectionPanel>

    <SectionPanel v-if="sections.includes('alarm')" :title="t('liveMap.alerts.title')" variant="plain" data-testid="section-alarms">
      <NAlert v-if="alertsError && !alerts" type="error" :bordered="false">{{ problemText(alertsError) }}</NAlert>
      <EmptyState v-else-if="alerts && !alerts.length" compact :title="t('empty.noAlerts')" />
      <ul v-else-if="alerts" class="notices">
        <li v-for="r in alerts" :key="r.id" class="notice" :data-alert="r.id">
          <p class="ant-clamp-2" :title="r.text">
            <span v-if="r.time" class="ant-muted">{{ r.time }} · </span><span :class="r.severe ? 'severe' : 'mild'">{{ r.text }}</span>
          </p>
          <div v-if="r.ref && canOpen(r.ref)">
            <ActionButton text type="primary" :size="size" :label="`${t('common.actions.open')} · ${r.ref.id}`" @click="emit('open', r.ref)" />
          </div>
        </li>
      </ul>
    </SectionPanel>
  </div>
</template>

<style scoped>
.tasks {
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

.notices {
  display: flex;
  flex-direction: column;
  min-width: 0;
  margin: 0;
  padding: 0;
  list-style: none;
}

.notice {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  min-width: 0;
  padding: var(--ant-space-2) 0;
  border-bottom: 1px solid var(--ant-border);
}

.notice:last-child {
  border-bottom: 0;
}

.severe {
  color: var(--ant-status-danger-text);
}

.mild {
  color: var(--ant-status-attention-text);
}
</style>
