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
import { useI18n } from 'vue-i18n'
import { useIncidents, useRiskScope } from '@/entities/incident'
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
import ProductionFlow from './ProductionFlow.vue'

const props = defineProps<WidgetProps>()
const { t } = useI18n()

/** Вид карты: «Карта производства» по участкам (по умолчанию) или схема процесса (BPMN) — уровень глубже. */
const view = ref<'flow' | 'bpmn'>(props.slice.view === 'bpmn' ? 'bpmn' : 'flow')

/** Инцидент по умолчанию — первый открытый: карта сразу показывает, где изделия его области. */
const incidents = useIncidents()
const openIncident = computed(() => incidents.data.value?.data.items.find((i) => i.status === 'open')?.incident_id)
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
  const incident = str(route?.query.incident) ?? str(props.slice.incident_id) ?? openIncident.value
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

/** Путь области риска инцидента карты (34 → 13 → 6) — на тот же момент и ось, что и карта. */
const scope = useRiskScope(computed(() => data.value?.incident?.incident_id ?? null))
const scopePath = computed(() => scope.data.value?.versions.map((v) => v.size) ?? null)
/** Текущий момент для ленты «что сейчас произошло». */
const now = computed(() => (moment.asOf ? Date.parse(moment.asOf) : Date.now()))
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
    <nav v-if="data" class="views" :aria-label="t('widgets.liveMap.flow.views')">
      <button type="button" :aria-pressed="view === 'flow'" data-view="flow" @click="view = 'flow'">{{ t('widgets.liveMap.flow.title') }}</button>
      <button type="button" :aria-pressed="view === 'bpmn'" data-view="bpmn" @click="view = 'bpmn'">{{ t('widgets.liveMap.flow.scheme') }}</button>
    </nav>
    <div v-if="data && view === 'flow'" class="flow-wrap">
      <ProductionFlow
        :map="data"
        :scope-path="scopePath"
        :marks="timelineData?.marks ?? []"
        :now="now"
        @open-section="view = 'bpmn'"
        @open-item="(id) => drill.open({ entity: 'item', id })"
      />
      <TimelineBar
        compact
        class="flow-timeline"
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
    </div>
    <LiveMapView
      v-if="data && view === 'bpmn'"
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

<style scoped>
.views {
  display: flex;
  gap: var(--ant-space-5);
  margin-bottom: var(--ant-space-4);
}

.views button {
  padding: 0 0 var(--ant-space-1);
  border: 0;
  border-bottom: 2px solid transparent;
  background: none;
  color: var(--ant-text-2);
  font: inherit;
  cursor: pointer;
}

.views button[aria-pressed='true'] {
  border-bottom-color: var(--ant-accent);
  color: var(--ant-text);
  font-weight: var(--ant-fw-bold);
}

.flow-wrap {
  height: 100%;
  min-height: 0;
  overflow-y: auto;
}

.flow-timeline {
  margin-top: var(--ant-space-6);
}
</style>
