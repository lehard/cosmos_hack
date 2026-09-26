<script setup lang="ts">
/**
 * Виджет «Живая карта» (FR-1…9, 130, 154, 155; AD-21, AD-22) — контейнер.
 *
 * Читает состояние карты через entities/live-map на момент из useMomentStore
 * (таймлайн меняет момент — карта перечитывается той же операцией, AD-22),
 * держит состояние интерфейса (период, версия) и ведёт в детали (FR-7).
 * Срез со стола: `period` — период по умолчанию, `incident_id`, `run_id`.
 * Инцидент и прогон можно задать и адресом страницы: `?incident=…&run=…`
 * (ссылки из ленты тревог, области риска и пульта сценариев).
 * Живые обновления: SSE `live_map` инвалидирует ключ, карта перечитывается (FR-2, SM-6).
 */
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useLiveMap, useProcesses, type CounterPeriod, type LiveMapParams } from '@/entities/live-map'
import { useDrillDown } from '@/features/drill-down'
import { backendModeOf } from '@/shared/api/response'
import type { WidgetProps } from '@/shared/config/widget'
import { WidgetFrame } from '@/shared/ui'
import { mapDataState } from '../model/overlays'
import LiveMapView from './LiveMapView.vue'

const props = defineProps<WidgetProps>()
const route = useRoute()
const drill = useDrillDown()

const PERIODS = new Set<string>(['shift', 'day', 'week', 'month', 'custom'])
const str = (v: unknown): string | undefined => (typeof v === 'string' && v ? v : undefined)

const period = ref<CounterPeriod>(PERIODS.has(String(props.slice.period)) ? (props.slice.period as CounterPeriod) : 'shift')
const range = ref<[number, number] | null>(null)
const versionId = ref<string | undefined>(undefined)
/** Процесс (UI-11); не выбран — основной. Другой процесс — его действующая версия. */
const processId = ref<string | undefined>(undefined)
function selectProcess(id: string): void {
  processId.value = id
  versionId.value = undefined
}
const processes = useProcesses()

const params = computed<LiveMapParams>(() => {
  const p: LiveMapParams = { period: period.value }
  if (period.value === 'custom' && range.value) {
    p.from = new Date(range.value[0]).toISOString()
    p.to = new Date(range.value[1]).toISOString()
  }
  if (processId.value) p.process_id = processId.value
  if (versionId.value) p.process_version_id = versionId.value
  const incident = str(route?.query.incident) ?? str(props.slice.incident_id)
  if (incident) p.incident_id = incident
  const run = str(route?.query.run) ?? str(props.slice.run_id)
  if (run) p.run_id = run
  return p
})

const query = useLiveMap(params)
const data = computed(() => query.data.value?.data ?? null)
const mode = computed(() => backendModeOf(query.data.value))
const state = computed(() => mapDataState(data.value))
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="mode"
    :state="state"
    :loading="query.isPending.value && !data"
    :error="data ? undefined : query.error.value"
    :data-widget="widgetId"
  >
    <LiveMapView
      v-if="data"
      v-model:period="period"
      v-model:range="range"
      :data="data"
      :processes="processes.data.value ?? []"
      :density="density"
      @select-process="selectProcess"
      @select-version="(id) => (versionId = id)"
      @open-item="(id) => drill.open({ entity: 'item', id })"
      @open-node="(key) => drill.open({ entity: 'live_map', id: key })"
    />
  </WidgetFrame>
</template>
