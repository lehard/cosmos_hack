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
 * Полоса времени (FR-4, FR-155) — часть карты, внизу одной строкой (UI-19):
 * фича playback, момент — useMomentStore, как у виджета таймлайна.
 */
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useLiveMap, useProcesses, useTimeline, type CounterPeriod, type LiveMapParams } from '@/entities/live-map'
import { TimelineBar, usePlayback, type PlaybackRange } from '@/features/playback'
import { useMomentStore } from '@/shared/model/moment'
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

const moment = useMomentStore()
const runId = computed(() => str(route?.query.run) ?? str(props.slice.run_id))
const timeline = useTimeline(computed(() => (runId.value ? { run_id: runId.value } : {})))
const timelineData = computed(() => timeline.data.value?.data ?? null)
const timeRange = computed<PlaybackRange | null>(() =>
  timelineData.value ? { from: Date.parse(timelineData.value.from), to: Date.parse(timelineData.value.to) } : null,
)
const playback = usePlayback(() => timeRange.value)
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
      :replay="moment.asOf !== null"
      :density="density"
      @select-process="selectProcess"
      @select-version="(id) => (versionId = id)"
      @open-item="(id) => drill.open({ entity: 'item', id })"
      @open-node="(key) => drill.open({ entity: 'live_map', id: key })"
    >
      <template #timeline>
        <TimelineBar
          compact
          :range="timeRange"
          :marks="timelineData?.marks"
          :as-of="moment.asOf"
          :axis="moment.axis"
          :playing="playback.playing.value"
          :speed="playback.speed.value"
          :density="density"
          @play="playback.play()"
          @pause="playback.pause()"
          @live="playback.goLive()"
          @jump="(at) => playback.jump(at)"
          @speed="(s) => (playback.speed.value = s)"
          @axis="(a) => (moment.asOf ? moment.travel(moment.asOf, a) : (moment.axis = a))"
        />
      </template>
    </LiveMapView>
  </WidgetFrame>
</template>
