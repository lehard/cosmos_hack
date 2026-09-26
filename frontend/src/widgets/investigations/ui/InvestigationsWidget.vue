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
import { useRecordLink } from '@/shared/model/record'
import InvestigationsView from './InvestigationsView.vue'
import NcLink from './NcLink.vue'

defineProps<WidgetProps>()

const focus = useAnalysisFocusStore()
const { incidents, list, id } = useFocusedIncident()
const items = computed(() => list.value)
/** Выбрать расследование: его несоответствие (primary_nc_id) становится входом гипотез и дорожек. */
/** Несоответствие расследования → окно несоответствия (Д-70), выбор — вход гипотез и дорожек. */
const record = useRecordLink()
function openNc(incidentId: string, ncId: string): void {
  select(incidentId)
  focus.ncId = ncId
  record.open({ entity: 'nonconformity', id: ncId })
}

function select(incidentId: string): void {
  focus.incidentId = incidentId
  focus.ncId = null
  focus.eventId = null
}
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :loading="incidents.isPending.value"
    :error="incidents.error.value"
    :empty="!items.length"
    empty-key="widgets.analysis.investigations.empty"
    :data-widget="widgetId"
  >
    <InvestigationsView v-if="items.length" :incidents="items" :selected="id" @select="select">
      <template #ncs="{ incident }">
        <NcLink v-for="nc in incident.nc_ids" :key="nc" :id="nc" @open="(ncId) => openNc(incident.incident_id, ncId)" />
      </template>
    </InvestigationsView>
  </WidgetFrame>
</template>
