<script setup lang="ts">
/**
 * Виджет «Мои расследования» — контейнер: инциденты (`analysis.incident.list`);
 * выбранный уходит в фокус разбора — его читают область риска и гипотезы
 * раздела «Расследование».
 */
import { computed } from 'vue'
import { useAnalysisFocusStore, useFocusedIncident } from '@/entities/incident'
import type { WidgetProps } from '@/shared/config/widget'
import { WidgetFrame } from '@/shared/ui'
import InvestigationsView from './InvestigationsView.vue'

defineProps<WidgetProps>()

const focus = useAnalysisFocusStore()
const { incidents, list, id } = useFocusedIncident()
const items = computed(() => list.value)
const select = (incidentId: string) => (focus.incidentId = incidentId)
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :loading="incidents.isPending.value"
    :error="incidents.error.value"
    :empty="!items.length"
    empty-key="empty.noRiskScope"
    :data-widget="widgetId"
  >
    <InvestigationsView v-if="items.length" :incidents="items" :selected="id" @select="select" />
  </WidgetFrame>
</template>
