<script setup lang="ts">
/**
 * Виджет «Разбор причин» — контейнер: группы несоответствий; выбранная группа
 * уходит в фокус разбора — её читают общие факторы и разбор обстоятельств.
 */
import { computed } from 'vue'
import { useAnalysisFocusStore } from '@/entities/incident'
import type { WidgetProps } from '@/shared/config/widget'
import { WidgetFrame } from '@/shared/ui'
import { useNcGroupsSource } from '../model/source'
import CauseAnalysisView from './CauseAnalysisView.vue'

defineProps<WidgetProps>()

const focus = useAnalysisFocusStore()
const src = useNcGroupsSource()
const data = computed(() => src.data.value)
const selected = computed({ get: () => focus.groupKey, set: (k: string | null) => focus.selectGroup(k) })
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="src.mode.value"
    :loading="src.isPending.value"
    :error="src.error.value"
    :empty="!data?.length"
    empty-key="empty.noNonconformities"
    :data-widget="widgetId"
  >
    <CauseAnalysisView v-if="data?.length" v-model:selected="selected" :groups="data" :density="density" />
  </WidgetFrame>
</template>
