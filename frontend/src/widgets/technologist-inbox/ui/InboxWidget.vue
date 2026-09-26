<script setup lang="ts">
/**
 * Виджет «Рабочий стол» технолога — контейнер: задачи из данных сервера;
 * «перейти» — раздел стола (расследование открывается сразу нужное).
 */
import { computed, inject } from 'vue'
import { routerKey } from 'vue-router'
import { useAnalysisFocusStore } from '@/entities/incident'
import type { WidgetProps } from '@/shared/config/widget'
import { WidgetFrame } from '@/shared/ui'
import { useInboxSource } from '../model/source'
import { buildTasks, type InboxTask } from '../model/tasks'
import InboxView from './InboxView.vue'

defineProps<WidgetProps>()

const router = inject(routerKey, null)
const focus = useAnalysisFocusStore()
const src = useInboxSource()
const tasks = computed(() => buildTasks(src.input.value))

function go(x: InboxTask): void {
  if (x.query?.incident) {
    focus.incidentId = x.query.incident
    focus.ncId = null
  }
  if (router?.hasRoute('desk')) void router.push({ name: 'desk', params: { tab: x.tab }, query: x.query ?? {} })
}
</script>

<template>
  <WidgetFrame :title-key="titleKey" :density="density" :loading="src.loading.value" :error="src.error.value" :data-widget="widgetId">
    <InboxView :tasks="tasks" @go="go" />
  </WidgetFrame>
</template>
