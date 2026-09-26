/**
 * Что рисовать поверх схемы (FR-2, FR-5, FR-9, FR-130) — чистые функции над
 * данными живой карты, без bpmn-js: их проверяют юнит-тесты.
 *
 * Цвета — только тона словаря статусов (contracts/statuses.yaml, AD-30).
 * В режиме инцидента цвет точки — статус изделия относительно этого инцидента,
 * а не общий статус качества (FR-9, NFR-UI-4).
 */
import type { IncidentStatus, LiveMapData, MapIncident, MapItem } from '@/entities/live-map'
import { statusAxes, statusDictionaries, statusPalette, type StatusTone } from '@/shared/api/generated/statuses'
import type { WidgetDataState } from '@/shared/config/widget'
import { codeToKey } from '@/shared/i18n'
import type { DiagramIndex } from './bpmn'

/** Сколько точек рисовать на узле; остальные — «+N». */
export const DOTS_PER_NODE = 6

/** Изделия показанной версии по step_key (FR-1: прежние версии — на своей карте). */
export function itemsByStep(items: readonly MapItem[], versionId: string): Map<string, MapItem[]> {
  const out = new Map<string, MapItem[]>()
  for (const it of items) {
    if (it.process_version_id !== versionId) continue
    const list = out.get(it.step_key)
    if (list) list.push(it)
    else out.set(it.step_key, [it])
  }
  return out
}

/** Изделий по прежним (не показанным) версиям. */
export const itemsOfOtherVersions = (items: readonly MapItem[], versionId: string): number =>
  items.filter((it) => it.process_version_id !== versionId).length

/** Вид точки изделия. */
export interface DotLook {
  color: string
  tone: StatusTone
  /** Изделие вне области инцидента — приглушено, чтобы не спутать с «неизвестно». */
  dimmed: boolean
  /** Ключ текста статуса для подсказки. */
  statusKey: string
}

/**
 * Вид точки: в режиме инцидента — по оси `incident`, иначе — по сводному статусу.
 * @param incidentMode — выбран инцидент (FR-9)
 */
export function dotLook(item: MapItem, incidentMode: boolean): DotLook {
  if (incidentMode) {
    if (!item.incident_status) {
      return { color: statusPalette.muted, tone: 'muted', dimmed: true, statusKey: '' }
    }
    const tone = statusAxes.incident.values[item.incident_status].tone as StatusTone
    return { color: statusPalette[tone], tone, dimmed: false, statusKey: `statuses.incident.${item.incident_status}` }
  }
  const entry = (statusDictionaries.item_summary.values as Record<string, { tone: string } | undefined>)[item.summary]
  const tone = (entry?.tone ?? 'neutral') as StatusTone
  return { color: statusPalette[tone], tone, dimmed: false, statusKey: `statuses.itemSummary.${codeToKey(item.summary)}` }
}

/** Порядок важности в режиме инцидента: сначала то, что требует решения. */
const INCIDENT_ORDER: Record<string, number> = { confirmed: 0, suspect: 1, unknown: 2, excluded: 3 }

/**
 * Точки узла по важности: в инциденте — подтверждённые, под подозрением,
 * неизвестные, исключённые, вне области; иначе — сначала признаки дефекта.
 * Порядок внутри группы — как пришло с сервера (сортировка устойчивая).
 */
export function sortForDots(items: readonly MapItem[], incidentMode: boolean): MapItem[] {
  const rank = (i: MapItem) =>
    incidentMode ? (i.incident_status ? INCIDENT_ORDER[i.incident_status]! : 4) : DEFECT_SIGNS.has(i.summary) ? 0 : 1
  return [...items].sort((a, b) => rank(a) - rank(b))
}

/** Легенда режима инцидента: каждый цвет с подписью (FR-9). */
export const INCIDENT_LEGEND: ReadonlyArray<{ status: IncidentStatus; textKey: string; color: string }> = (
  [
    ['confirmed', 'liveMap.incident.legendConfirmed'],
    ['suspect', 'liveMap.incident.legendSuspect'],
    ['excluded', 'liveMap.incident.legendExcluded'],
    ['unknown', 'liveMap.incident.legendUnknown'],
  ] as const
).map(([status, textKey]) => ({ status, textKey, color: statusPalette[statusAxes.incident.values[status].tone as StatusTone] }))

/** Сокращение области риска (FR-9, FR-61): 34 → 6, на 82 %. */
export function scopeReduction(incident: MapIncident): { from: number; to: number; fraction: number } {
  const from = incident.size_at_creation
  const to = incident.size
  return { from, to, fraction: from > 0 ? Math.max(0, (from - to) / from) : 0 }
}

/** Изделий в каждой дорожке-цехе показанной версии (FR-130). */
export function itemsPerLane(byStep: ReadonlyMap<string, MapItem[]>, index: DiagramIndex): Map<string, number> {
  const out = new Map<string, number>()
  for (const lane of index.lanes) out.set(lane.bpmnId, 0)
  for (const [stepKey, items] of byStep) {
    const laneId = index.byStepKey.get(stepKey)?.laneId
    if (laneId) out.set(laneId, (out.get(laneId) ?? 0) + items.length)
  }
  return out
}

/** Сводные статусы, которые означают признак дефекта или подозрение (не «брак»). */
const DEFECT_SIGNS = new Set<string>(['suspect', 'reinspection_required', 'hold', 'pending_decision', 'nonconforming'])

/**
 * Состояние данных карты для рамки (NFR-UI-4): признак дефекта — есть дефекты за
 * период, изделия под подозрением или инцидент; «оценка невозможна» — есть узлы
 * без данных источника и признаков дефекта нет; иначе — норма.
 */
export function mapDataState(data: LiveMapData | null | undefined): WidgetDataState {
  if (!data) return 'normal'
  const defects = data.counters.some((c) => c.defects > 0) || data.items.some((i) => DEFECT_SIGNS.has(i.summary))
  if (defects || data.incident) return 'defect_indication'
  if (data.data_gaps.length) return 'unable_to_assess'
  return 'normal'
}
