<script setup lang="ts">
/**
 * Виджет «Задачи» (FR-57, FR-8, FR-55; PRD §3a «Мастер участка — Задачи»,
 * терминал исполнителя FR-137) — контейнер: задачи и запросы решения
 * (`notifications.task.list`), отметка (`notifications.task.acknowledge`),
 * подтверждение перемещения в изолятор (`process.movement.receive`), тревоги
 * (`notifications.alert.list`), просроченные решения с ценой задержки
 * (`notifications.attention.list`), сводка по видам (`notifications.summary.read`).
 * Разделы читаются независимо: отказ одной операции не прячет остальные.
 * Срез — model/slice.ts.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { alertText, attentionText, useAlerts, useAttention, useNotificationSummary } from '@/entities/notification'
import { useSession } from '@/entities/session'
import { useTasks } from '@/entities/task'
import { useDrillDown } from '@/features/drill-down'
import { backendModeOf } from '@/shared/api/response'
import type { WidgetProps } from '@/shared/config/widget'
import { useMomentStore } from '@/shared/model/moment'
import { WidgetFrame } from '@/shared/ui'
import { ownWorkplaceTasks, runOf, sectionsOf, splitTasks, type NoticeRow } from '../model/slice'
import TasksView from './TasksView.vue'

const props = defineProps<WidgetProps>()
const { t, te, d } = useI18n()
const drill = useDrillDown()
const moment = useMomentStore()
const session = useSession()

const sections = computed(() => sectionsOf(props.slice))
const run = computed(() => runOf(props.slice))
const runParams = computed(() => (run.value ? { run_id: run.value } : {}))

const tasksQ = useTasks(runParams)
const alertsQ = useAlerts(runParams)
const attentionQ = useAttention(runParams)
const summaryQ = useNotificationSummary()

const workplaceId = computed(() => (props.slice.scope === 'own_workplace' ? (session.data.value?.data.workplace?.id ?? null) : null))
const tasks = computed(() => {
  const items = tasksQ.data.value?.data.items
  return items ? splitTasks(ownWorkplaceTasks(items, workplaceId.value)) : null
})

const x = { t, te }
const alerts = computed<NoticeRow[] | null>(() => {
  const list = alertsQ.data.value?.data
  if (!list) return null
  return list.map((a) => ({
    id: a.alert_id,
    text: alertText(x, a),
    time: d(new Date(a.at), 'dateTime'),
    severe: a.kind !== 'anomaly' && a.kind !== 'gate_overdue',
    ref: a.ref ?? null,
  }))
})
const escalations = computed<NoticeRow[] | null>(() => {
  const list = attentionQ.data.value?.data
  if (!list) return null
  return list.map((e) => ({ id: e.entry_id, text: attentionText(x, e), severe: e.kind === 'overdue_decision', ref: e.ref ?? null }))
})

/** Все включённые разделы не прочитались — «ошибка входа». */
const allFailed = computed(() => {
  const failed = { task: !!tasksQ.error.value && !tasks.value, alarm: !!alertsQ.error.value && !alerts.value, escalation: !!attentionQ.error.value && !escalations.value }
  return sections.value.every((s) => failed[s])
})
const mode = computed(() => backendModeOf(tasksQ.data.value ?? alertsQ.data.value ?? attentionQ.data.value))
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="mode"
    :state="allFailed ? 'input_error' : 'normal'"
    :loading="tasksQ.isPending.value && alertsQ.isPending.value && attentionQ.isPending.value"
    :data-widget="widgetId"
  >
    <TasksView
      :sections="sections"
      :summary="summaryQ.data.value?.data ?? null"
      :open="tasks?.open ?? null"
      :closed="tasks?.closed ?? []"
      :basis-seq="tasksQ.data.value?.data.basis_seq ?? 0"
      :tasks-error="tasksQ.error.value ?? undefined"
      :alerts="alerts"
      :alerts-error="alertsQ.error.value ?? undefined"
      :escalations="escalations"
      :escalations-error="attentionQ.error.value ?? undefined"
      :can-act="!moment.isReplay"
      :density="density"
      :can-open="drill.canOpen"
      @open="drill.open"
    />
  </WidgetFrame>
</template>
