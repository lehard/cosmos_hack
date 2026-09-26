/**
 * Образцы данных живой карты — только для тестов (PRD §11.10: в коде виджетов
 * заготовок нет, данные приходят через API).
 *
 * Схема — настоящий стартовый процесс normative/process/flange-process.bpmn.
 * Кадры главной истории «плохой день сварочного участка» (FR-155): сигнал на
 * КТ-3 → инцидент → область риска 34 → 13 → 6 → решение по изделиям.
 * Числа иллюстративные: так история выглядит на карте, а не так её считает сервер.
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import type { IncidentStatus, LiveMapData, MapItem, NodeCounters } from '@/entities/live-map'

/** Стартовый BPMN фланца из нормативного слоя. */
export const flangeXml = (): string => readFileSync(resolve(__dirname, '../../../../../normative/process/flange-process.bpmn'), 'utf8')

export const V1 = 'process-flange@1'
export const V0 = 'process-flange@0'

const item = (n: number, step_key: string, extra: Partial<MapItem> = {}): MapItem => ({
  item_id: `ENT:FL-${String(n).padStart(4, '0')}`,
  label: `ФЛ-${String(n).padStart(4, '0')}`,
  step_key,
  position: 'in_queue',
  summary: 'in_process',
  process_version_id: V1,
  ...extra,
})

const counters = (step_key: string, queue: number, in_progress: number, passed: number, defects = 0, nonconformities?: number): NodeCounters => ({
  step_key,
  queue,
  in_progress,
  passed,
  defects,
  ...(nonconformities !== undefined ? { nonconformities } : {}),
})

/** 34 изделия, сваренные в окне подозрения, разошлись по сварке, сборке и испытаниям. */
function scopeItems(statusOf: (n: number) => IncidentStatus | undefined): MapItem[] {
  const out: MapItem[] = []
  for (let n = 1; n <= 34; n++) {
    const step = n <= 12 ? 'welding.zt3_acceptance' : n <= 26 ? 'assembly.seal_install' : 'testing.leak_test'
    const incident_status = statusOf(n)
    out.push(item(n, step, { position: n <= 12 ? 'at_presentation_point' : 'in_progress', ...(incident_status ? { incident_status } : {}) }))
  }
  return out
}

const base = (): Omit<LiveMapData, 'items' | 'incident' | 'counters'> => ({
  basis_seq: 4200,
  process_version: { process_version_id: V1, label: '1', is_current: true, items: 40 },
  versions: [
    { process_version_id: V1, label: '1', is_current: true, items: 40 },
    { process_version_id: V0, label: '0', is_current: false, items: 2 },
  ],
  bpmn_xml: flangeXml(),
  bottleneck: { step_key: 'welding.zt3_acceptance', wait: '37 мин' },
  anomalies: [{ step_key: 'welding.zt3_acceptance', kind: 'queue_above_norm' }],
  data_gaps: [],
})

const commonCounters = (defects: number): NodeCounters[] => [
  counters('machining.cnc', 2, 1, 18),
  counters('welding.weld', 1, 1, 36),
  counters('welding.kt3_camera', 0, 1, 35, defects, defects ? 1 : 0),
  counters('welding.zt3_acceptance', 12, 0, 22, 0, 0),
  counters('assembly.seal_install', 14, 0, 8),
  counters('testing.leak_test', 8, 0, 3),
]

const others = (): MapItem[] => [
  item(40, 'machining.cnc', { position: 'in_progress' }),
  item(41, 'welding.weld', { position: 'in_progress' }),
  item(42, 'welding.send_to_assembly', { position: 'in_transit' }),
  // Прежняя версия — показывается только на своей карте (FR-1).
  item(90, 'welding.weld', { process_version_id: V0 }),
  item(91, 'assembly.seal_install', { process_version_id: V0 }),
]

/** Кадр 0: утро, всё идёт, ограничение линии — ЗТ-3. */
export function frameMorning(): LiveMapData {
  return { ...base(), anomalies: [], counters: commonCounters(0), items: [...scopeItems(() => undefined), ...others()] }
}

/** Кадр 1: сигнал на КТ-3 и инцидент — область риска 34, все под подозрением. */
export function frameScope34(): LiveMapData {
  const items = scopeItems((n) => (n === 1 ? 'confirmed' : 'suspect')).map((i) => (i.item_id.endsWith('0001') ? { ...i, summary: 'hold' as const } : { ...i, summary: 'suspect' as const }))
  return {
    ...base(),
    counters: commonCounters(1),
    items: [...items, ...others()],
    incident: { incident_id: 'INC-1', label: 'Пористость шва, сварочный пост 2', scope_version: 1, size_at_creation: 34, size: 34, basis: 'Сигнал КТ-3 по ФЛ-0001' },
  }
}

/** Кадр 2: сужение до 13 по журналу оборудования. */
export function frameScope13(): LiveMapData {
  const f = frameScope34()
  return {
    ...f,
    items: f.items.map((i, k) => (i.incident_status && k >= 13 ? { ...i, incident_status: 'excluded' as const, summary: 'cleared' as const } : i)),
    incident: { ...f.incident!, scope_version: 2, size: 13, basis: 'Журнал сварочного поста: сбой газа только 10:40–11:25' },
  }
}

/** Кадр 3: сужение до 6 после рентгена; у двух изделий данных нет. */
export function frameScope6(): LiveMapData {
  const f = frameScope13()
  return {
    ...f,
    items: f.items.map((i, k) =>
      i.incident_status && i.incident_status !== 'excluded' && k >= 6 ? { ...i, incident_status: 'excluded' as const, summary: 'cleared' as const } : k === 4 || k === 5 ? { ...i, incident_status: 'unknown' as const } : i,
    ),
    incident: { ...f.incident!, scope_version: 3, size: 6, basis: 'Рентген шва КТ-3: 7 изделий без пор' },
  }
}

/** Вся история по порядку. */
export const storyFrames = (): LiveMapData[] => [frameMorning(), frameScope34(), frameScope13(), frameScope6()]
