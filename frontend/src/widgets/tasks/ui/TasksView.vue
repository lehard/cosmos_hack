<script setup lang="ts">
/**
 * Задачи — представление (FR-57, FR-8): сводка по видам (информация, тревога,
 * задача, запрос решения), открытые задачи и запросы решения с отметкой,
 * тревоги и просроченные решения с ценой задержки. Выполненные — свёрнуты.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NCollapse, NCollapseItem, NDivider, NEllipsis, NFlex, NList, NListItem, NTag, NText, NButton } from 'naive-ui'
import type { NotificationSummary } from '@/entities/notification'
import type { TaskEntry } from '@/entities/task'
import { TaskInbox } from '@/features/task-actions'
import type { DrillRef } from '@/shared/model/drill'
import type { Density } from '@/shared/config/widget'
import { naiveSizeOf } from '@/shared/config/widget'
import { useProblemText } from '@/shared/i18n/problem'
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
  <NFlex vertical :size="12" data-testid="tasks-view">
    <NFlex v-if="summary?.by_kind" :size="8" :wrap="true" data-testid="summary">
      <NTag v-for="k in KINDS" :key="k.key" :size="size" :bordered="false" :type="summary.by_kind[k.key] ? k.type : 'default'" :data-kind="k.key">
        {{ t(k.text) }}: {{ summary.by_kind[k.key] }}
      </NTag>
    </NFlex>

    <section v-if="sections.includes('task')" data-testid="section-tasks">
      <NDivider title-placement="left">{{ t('desks.tasks') }}</NDivider>
      <NText v-if="tasksError && !open" type="error">{{ problemText(tasksError) }}</NText>
      <TaskInbox v-else-if="open" :tasks="open" :basis-seq="basisSeq" :can-act="canAct" :density="density" :can-open="canOpen" @open="(r) => emit('open', r)" />
      <NCollapse v-if="closed.length" data-testid="closed-tasks">
        <NCollapseItem :title="t('widgets.shopFloor.tasks.closed', { n: closed.length })" name="closed">
          <TaskInbox :tasks="closed" :basis-seq="basisSeq" :can-act="false" :density="density" :can-open="canOpen" @open="(r) => emit('open', r)" />
        </NCollapseItem>
      </NCollapse>
    </section>

    <section v-if="sections.includes('escalation')" data-testid="section-escalations">
      <NDivider title-placement="left">{{ t('widgets.shopFloor.tasks.escalations') }}</NDivider>
      <NText v-if="escalationsError && !escalations" type="error">{{ problemText(escalationsError) }}</NText>
      <NText v-else-if="escalations && !escalations.length" depth="3">{{ t('empty.queueEmpty') }}</NText>
      <NList v-else-if="escalations" :show-divider="true">
        <NListItem v-for="r in escalations" :key="r.id" :data-escalation="r.id">
          <NFlex vertical :size="4">
            <NText :type="r.severe ? 'error' : 'warning'"><NEllipsis :line-clamp="3" :tooltip="{ width: 360 }">{{ r.text }}</NEllipsis></NText>
            <div v-if="r.ref && canOpen(r.ref)">
              <NButton text type="primary" :size="size" @click="emit('open', r.ref)">{{ t('common.actions.open') }} · {{ r.ref.id }}</NButton>
            </div>
          </NFlex>
        </NListItem>
      </NList>
    </section>

    <section v-if="sections.includes('alarm')" data-testid="section-alarms">
      <NDivider title-placement="left">{{ t('liveMap.alerts.title') }}</NDivider>
      <NText v-if="alertsError && !alerts" type="error">{{ problemText(alertsError) }}</NText>
      <NText v-else-if="alerts && !alerts.length" depth="3">{{ t('empty.noAlerts') }}</NText>
      <NList v-else-if="alerts" :show-divider="true">
        <NListItem v-for="r in alerts" :key="r.id" :data-alert="r.id">
          <NFlex vertical :size="4">
            <NFlex :size="8" align="baseline" :wrap="false">
              <NText v-if="r.time" depth="3">{{ r.time }}</NText>
              <NText :type="r.severe ? 'error' : 'warning'"><NEllipsis :line-clamp="3" :tooltip="{ width: 360 }">{{ r.text }}</NEllipsis></NText>
            </NFlex>
            <div v-if="r.ref && canOpen(r.ref)">
              <NButton text type="primary" :size="size" @click="emit('open', r.ref)">{{ t('common.actions.open') }} · {{ r.ref.id }}</NButton>
            </div>
          </NFlex>
        </NListItem>
      </NList>
    </section>
  </NFlex>
</template>
