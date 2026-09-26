/**
 * Живая карта процесса (FR-1…5, 9, 130, 154, 155; AD-21, AD-22): состояние
 * производства на схеме действующей версии и таймлайн. Слой entities (FSD):
 * ключи кэша, форма данных и обёртки запросов — виджеты live-map и map-timeline
 * читают сервер только отсюда.
 *
 * Операций живой карты ещё нет в contracts/openapi.yaml (эпик 02), поэтому
 * запросы стоят на заглушке `api.not_implemented` (shared/api/pending.ts), а
 * форма данных описана здесь по спайну и PRD — она же предложение для
 * контракта. Когда операции появятся, типы заменяются сгенерированными, а
 * функции запроса — вызовами клиента; ключи и виджеты не меняются.
 *
 * Счётчики узлов и ограничение линии считает сервер (AD-21): здесь только
 * показываем, ничего не пересчитываем.
 */
import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { useQuery } from '@tanstack/vue-query'
import { statusAxes, statusDictionaries } from '@/shared/api/generated/statuses'
import { entityKeys } from '@/shared/api/keys'
import { pendingOperation, type Envelope } from '@/shared/api/pending'

export type { Envelope }
import type { DrillRef } from '@/shared/model/drill'
import { useMomentStore } from '@/shared/model/moment'

export type { DrillRef }

export const liveMapKeys = entityKeys('live_map')

// ─────────────────────────────── форма данных ───────────────────────────────

/** Период счётчиков узлов (FR-3). */
export type CounterPeriod = 'shift' | 'day' | 'week' | 'month' | 'custom'

/** Счётчик узла (FR-2): очередь / в работе / прошло / дефекты за период. */
export type CounterKind = 'queue' | 'in_progress' | 'passed' | 'defects'

/** Все счётчики по порядку показа на узле. */
export const COUNTER_KINDS: readonly CounterKind[] = ['queue', 'in_progress', 'passed', 'defects']

/** Вид аномалии узла (FR-5). */
export type AnomalyKind =
  | 'queue_above_norm'
  | 'wait_above_norm'
  | 'downtime_over_threshold'
  | 'output_spike'
  | 'defect_rate_out_of_control'

/** Статус изделия относительно инцидента (ось `incident` словаря статусов, FR-9). */
export type IncidentStatus = keyof typeof statusAxes.incident.values

/** Сводный статус изделия для списков и карты (словарь `item_summary`). */
export type ItemSummaryStatus = keyof typeof statusDictionaries.item_summary.values

/** Положение изделия в процессе (ось `position`). */
export type PositionStatus = keyof typeof statusAxes.position.values

/** Счётчики одного узла по `step_key` (FR-2, FR-3). */
export interface NodeCounters {
  step_key: string
  queue: number
  in_progress: number
  passed: number
  /** Дефекты за период: физические дефекты, не наблюдения (соглашение «Дефект»). */
  defects: number
  /** Открытые несоответствия узла — для перехода из карточки узла (FR-154). */
  nonconformities?: number
}

/** Аномалия узла (FR-5); `threshold` — порог простоя текстом с единицей. */
export interface NodeAnomaly {
  step_key: string
  kind: AnomalyKind
  threshold?: string
}

/** Изделие-точка в текущем узле (FR-2). */
export interface MapItem {
  item_id: string
  /** Номер детали для подписи точки. */
  label: string
  step_key: string
  position: PositionStatus
  summary: ItemSummaryStatus
  /** Версия процесса, по которой изделие запущено (FR-1: прежние версии — на своей карте). */
  process_version_id: string
  /** Статус относительно выбранного инцидента; нет — изделие вне области (FR-9). */
  incident_status?: IncidentStatus
}

/** Версия процесса на карте. */
export interface ProcessVersionRef {
  process_version_id: string
  /** Номер версии для подписи. */
  label: string
  /** Действующая версия. */
  is_current: boolean
  /** Изделий в работе по этой версии. */
  items: number
}

/** Узел-ограничение линии (FR-5): наибольшее ожидание при наибольшей загрузке. */
export interface Bottleneck {
  step_key: string
  /** Среднее ожидание текстом с единицей («37 мин»). */
  wait?: string
}

/** Выбранный инцидент на карте (FR-9, FR-61). */
export interface MapIncident {
  incident_id: string
  label: string
  /** Версия области риска. */
  scope_version: number
  /** Размер области при создании. */
  size_at_creation: number
  /** Размер области сейчас (34 → 13 → 6). */
  size: number
  /** Основание последнего изменения области. */
  basis?: string
}

/** Состояние живой карты на момент (AD-22). */
export interface LiveMapData {
  /** Версия, схема которой показана. */
  process_version: ProcessVersionRef
  /** Все версии, по которым есть изделия в работе. */
  versions: ProcessVersionRef[]
  /** BPMN 2.0 XML показанной версии: раскладка BPMNDI, `documentation`, `ant:properties/@stepKey`. */
  bpmn_xml: string
  counters: NodeCounters[]
  items: MapItem[]
  bottleneck: Bottleneck | null
  anomalies: NodeAnomaly[]
  /** Узлы, где оценка невозможна: нет данных источника (не «норма»). */
  data_gaps: string[]
  incident: MapIncident | null
}

/** Параметры запроса живой карты (помимо момента). */
export interface LiveMapParams {
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

/** Вид метки таймлайна (FR-4). */
export type TimelineMarkKind = 'escalation' | 'process_stop' | 'spike' | 'revision'

/** Значимое событие на таймлайне. */
export interface TimelineMark {
  mark_id: string
  /** Момент события на выбранной оси (RFC 3339 UTC). */
  at: string
  kind: TimelineMarkKind
  /** Короткая подпись от сервера (узел, изделие). */
  title?: string
  ref?: DrillRef
}

/** Таймлайн: диапазон истории и метки. */
export interface TimelineData {
  /** Начало доступной истории (или прогона). */
  from: string
  /** Конец — «сейчас» на сервере. */
  to: string
  marks: TimelineMark[]
}

// ────────────────────────────────── запросы ─────────────────────────────────

/**
 * Состояние живой карты на момент из useMomentStore (FR-1…5, 9).
 * Ожидаемая операция — `process.live_map.read` (GET /api/v1/live-map).
 */
export function useLiveMap(params: MaybeRefOrGetter<LiveMapParams>) {
  const moment = useMomentStore()
  return useQuery({
    queryKey: computed(() => liveMapKeys.list('view', toValue(params), moment.params)),
    queryFn: () => pendingOperation<Envelope<LiveMapData>>('process.live_map.read')(),
    // Смена момента или периода не мигает пустой картой — старое видно до нового ответа.
    placeholderData: (prev: Envelope<LiveMapData> | undefined) => prev,
  })
}

/**
 * Диапазон истории и метки таймлайна (FR-4). Ожидаемая операция —
 * `journal.timeline.read` (GET /api/v1/timeline); ось — из момента.
 */
export function useTimeline(params: MaybeRefOrGetter<{ run_id?: string }>) {
  const moment = useMomentStore()
  return useQuery({
    queryKey: computed(() => liveMapKeys.list('timeline', toValue(params), { axis: moment.axis })),
    queryFn: () => pendingOperation<Envelope<TimelineData>>('journal.timeline.read')(),
  })
}
