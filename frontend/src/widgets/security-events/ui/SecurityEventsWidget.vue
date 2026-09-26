<script setup lang="ts">
/**
 * Виджет «События безопасности» стола Аудитора ИБ (только чтение; FR-106) —
 * контейнер: `security.event.list` на момент, фильтр по типу записи семейства
 * security. Срез: `event_type` — тип по умолчанию.
 */
import { computed, ref } from 'vue'
import { useSecurityEvents } from '@/entities/integrity'
import { backendModeOf } from '@/shared/api/response'
import type { WidgetProps } from '@/shared/config/widget'
import { WidgetFrame } from '@/shared/ui'
import SecurityEventsView from './SecurityEventsView.vue'

const props = defineProps<WidgetProps>()
const eventType = ref<string | null>(typeof props.slice.event_type === 'string' ? props.slice.event_type : null)
const listQ = useSecurityEvents(eventType)
const events = computed(() => listQ.data.value?.data?.items ?? null)
// Тревога в ленте — «признак», а не норма (NFR-UI-4).
const state = computed(() => (events.value?.some((e) => e.severity === 'alarm') ? 'defect_indication' : 'normal'))
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="backendModeOf(listQ.data.value)"
    :state="state"
    :loading="listQ.isPending.value && !events"
    :error="events ? undefined : listQ.error.value"
    :data-widget="widgetId"
  >
    <SecurityEventsView v-if="events" v-model:event-type="eventType" :events="events" :density="density" />
  </WidgetFrame>
</template>
