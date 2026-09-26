/**
 * Живая карта процесса (FR-1…5, 9, 130, 154, 155; AD-21, AD-22): состояние
 * производства на схеме действующей версии и таймлайн. Слой entities (FSD):
 * ключи кэша, типы контракта и обёртки запросов — виджеты live-map и
 * map-timeline читают сервер только отсюда.
 *
 * Операции — `process.live_map.read` и `journal.timeline.read` из
 * contracts/openapi.yaml, вызов — сгенерированным клиентом; ключи — по
 * соглашению shared/api/keys.ts, чтобы SSE `live_map` их инвалидировал.
 *
 * Счётчики узлов и ограничение линии считает сервер (AD-21): здесь только
 * показываем, ничего не пересчитываем.
 */
import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { keepPreviousData, useQuery } from '@tanstack/vue-query'
import { journalTimelineRead, processLiveMapRead, processProcessList } from '@/shared/api/generated/client'
import type {
  LiveMap,
  MapBottleneck,
  MapIncident,
  MapItem,
  MapItemIncidentStatus,
  MapItemPosition,
  MapItemSummary,
  MapNodeAnomaly,
  MapNodeAnomalyKind,
  MapNodeCounters,
  MapVersionRef,
  ProcessLiveMapReadPeriod,
  TimelineData,
  TimelineMark,
  TimelineMarkKind,
} from '@/shared/api/generated/model'
import { entityKeys } from '@/shared/api/keys'
import type { Envelope } from '@/shared/api/response'
import type { DrillRef } from '@/shared/model/drill'
import { useMomentStore } from '@/shared/model/moment'

export type { DrillRef, Envelope, MapIncident, MapItem, TimelineData, TimelineMark, TimelineMarkKind }

export * from './model/steps'

export const liveMapKeys = entityKeys('live_map')

// ───────────────────── типы контракта под именами виджетов ─────────────────────

/** Состояние живой карты на момент (AD-22). `bottleneck`, `incident` могут не прийти. */
export type LiveMapData = LiveMap
/** Период счётчиков узлов (FR-3). */
export type CounterPeriod = ProcessLiveMapReadPeriod
/** Счётчики одного узла по `step_key` (FR-2, FR-3). */
export type NodeCounters = MapNodeCounters
/** Аномалия узла (FR-5). */
export type NodeAnomaly = MapNodeAnomaly
export type AnomalyKind = MapNodeAnomalyKind
/** Узел-ограничение линии (FR-5). */
export type Bottleneck = MapBottleneck
/** Версия процесса на карте. */
export type ProcessVersionRef = MapVersionRef
/** Статус изделия относительно инцидента (FR-9). */
export type IncidentStatus = MapItemIncidentStatus
/** Сводный статус изделия для карты. */
export type ItemSummaryStatus = MapItemSummary
/** Положение изделия в процессе. */
export type PositionStatus = MapItemPosition

/** Параметры запроса живой карты (помимо момента). */
export interface LiveMapParams {
  /** Процесс (UI-11); не задан — основной. */
  process_id?: string
  /** Версия процесса; не задана — действующая. */
  process_version_id?: string
  period: CounterPeriod
  /** Границы произвольного периода (RFC 3339 UTC). */
  from?: string
  to?: string
  /** Режим инцидента: изделия окрашены по статусу в этом инциденте. */
  incident_id?: string
  /** Прогон сценария (AD-38): карта в пределах прогона. */
  run_id?: string
}

// ────────────────────────────────── запросы ─────────────────────────────────

/** Состояние живой карты на момент из useMomentStore (FR-1…5, 9) — `process.live_map.read`. */
export function useLiveMap(params: MaybeRefOrGetter<LiveMapParams>) {
  const moment = useMomentStore()
  return useQuery({
    queryKey: computed(() => liveMapKeys.list('view', toValue(params), moment.params)),
    queryFn: async ({ signal }): Promise<Envelope<LiveMapData>> => {
      const res = await processLiveMapRead({ ...toValue(params), ...moment.params }, { signal })
      return { data: res.data, headers: res.headers }
    },
    // Смена момента или периода не мигает пустой картой — старое видно до нового ответа.
    placeholderData: keepPreviousData,
  })
}

/** Процессы предприятия для выбора на карте (UI-11) — `process.process.list`. */
export function useProcesses() {
  const moment = useMomentStore()
  return useQuery({
    queryKey: computed(() => liveMapKeys.list('processes', {}, moment.params)),
    queryFn: async ({ signal }) => {
      const res = await processProcessList(moment.params, { signal })
      return res.data.items
    },
  })
}

/**
 * Диапазон истории и метки таймлайна (FR-4) — `journal.timeline.read`. Зависит
 * только от оси: момент воспроизведения диапазон не меняет.
 */
export function useTimeline(params: MaybeRefOrGetter<{ run_id?: string }>) {
  const moment = useMomentStore()
  return useQuery({
    queryKey: computed(() => liveMapKeys.list('timeline', toValue(params), { axis: moment.axis })),
    queryFn: async ({ signal }): Promise<Envelope<TimelineData>> => {
      const res = await journalTimelineRead({ ...toValue(params), axis: moment.axis }, { signal })
      return { data: res.data, headers: res.headers }
    },
  })
}
