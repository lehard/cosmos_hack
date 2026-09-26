<script setup lang="ts">
/**
 * Виджет «История выдачи прав» стола Аудитора ИБ (только чтение; FR-79;
 * AD-11) — контейнер: `access.grant.list` на момент, фильтр по сотруднику.
 * Срез: `person_id` — сотрудник по умолчанию.
 */
import { computed, ref } from 'vue'
import { useGrantHistory } from '@/entities/policy'
import { backendModeOf } from '@/shared/api/response'
import type { WidgetProps } from '@/shared/config/widget'
import { WidgetFrame } from '@/shared/ui'
import GrantsHistoryView from './GrantsHistoryView.vue'

const props = defineProps<WidgetProps>()
const personId = ref<string | null>(typeof props.slice.person_id === 'string' ? props.slice.person_id : null)
const listQ = useGrantHistory(personId)
const entries = computed(() => listQ.data.value?.data?.items ?? null)
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="backendModeOf(listQ.data.value)"
    :loading="listQ.isPending.value && !entries"
    :error="entries ? undefined : listQ.error.value"
    :data-widget="widgetId"
  >
    <GrantsHistoryView v-if="entries" v-model:person-id="personId" :entries="entries" :density="density" />
  </WidgetFrame>
</template>
