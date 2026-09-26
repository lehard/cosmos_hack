<script setup lang="ts">
/**
 * Виджет «Центр управления» — контейнер: живая карта смены (с открытым
 * инцидентом — для «локализовано N из M»), внимание и тревоги, инциденты и путь
 * области риска. Щелчок по строке — к объекту (FR-7); «Открыть на карте» —
 * раздел «Карта производства» с этим инцидентом.
 */
import { computed, inject } from 'vue'
import { routerKey } from 'vue-router'
import { useIncidents, useRiskScope } from '@/entities/incident'
import { useAlerts, useAttention } from '@/entities/notification'
import { useDrillDown } from '@/features/drill-down'
import type { WidgetProps } from '@/shared/config/widget'
import { WidgetFrame } from '@/shared/ui'
import { containment, sectionStates, sectionsOf, useLiveMap } from '@/entities/live-map'
import { decisionRows, stateTiles } from '../model/center'
import ControlCenterView from './ControlCenterView.vue'

defineProps<WidgetProps>()
const router = inject(routerKey, null)
const drill = useDrillDown()

const incidents = useIncidents()
const incident = computed(() => incidents.data.value?.data.items.find((i) => i.status === 'open') ?? null)
const map = useLiveMap(computed(() => ({ period: 'shift' as const, ...(incident.value ? { incident_id: incident.value.incident_id } : {}) })))
const mapData = computed(() => map.data.value?.data ?? null)
const attention = useAttention({})
const alerts = useAlerts({})
const decisions = computed(() => decisionRows(attention.data.value?.data ?? [], alerts.data.value?.data ?? []))
const tiles = computed(() => stateTiles(mapData.value, incident.value, decisions.value))
const scope = useRiskScope(computed(() => incident.value?.incident_id ?? null))
const scopePath = computed(() => scope.data.value?.versions.map((v) => v.size) ?? null)
const contained = computed(() => (mapData.value && incident.value ? containment(sectionStates(mapData.value, sectionsOf(mapData.value.bpmn_xml))) : null))

function openMap(incidentId: string): void {
  if (router?.hasRoute('desk')) void router.push({ name: 'desk', params: { tab: 'processes' }, query: { incident: incidentId } })
}
</script>

<template>
  <WidgetFrame :title-key="titleKey" :density="density" :loading="incidents.isPending.value && map.isPending.value" :data-widget="widgetId">
    <ControlCenterView
      :tiles="tiles"
      :decisions="decisions"
      :incident="incident"
      :scope-path="scopePath"
      :contained="contained"
      :can-open="(ref) => drill.canOpen(ref)"
      @open="(ref) => drill.open(ref)"
      @open-map="openMap"
    />
  </WidgetFrame>
</template>
