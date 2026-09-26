/**
 * «Карта производства» (режим руководителя, по умолчанию): шаги процесса сведены
 * в участки — дорожки BPMN версии («Склад и входной контроль», «Механический
 * цех», «Сварочный цех»…), в порядке схемы. У участка — сколько в работе, в
 * очереди, прошло, дефектов; изделия инцидента на участке и сколько из них
 * локализовано (в изоляторе); отклонения от норм и узкое место; статус участка.
 * Всё — из ответа живой карты (те же counters, items, anomalies, bottleneck);
 * BPMN разбирается без bpmn-js — только дорожки и ant:properties/@stepKey.
 */
import type { LiveMap, MapItem } from '@/shared/api/generated/model'

/** Участок — дорожка BPMN и её шаги. */
export interface FlowSection {
  id: string
  name: string
  stepKeys: string[]
}

/** Статус участка: норма / отклонение / инцидент или ограничение / нет данных. */
export type SectionStatus = 'normal' | 'attention' | 'danger' | 'unknown'

export interface SectionState extends FlowSection {
  queue: number
  inProgress: number
  passed: number
  defects: number
  nonconformities: number
  /** Изделия, которые сейчас на участке. */
  items: MapItem[]
  /** Изделия области риска на участке (подтверждено / под подозрением / нет данных). */
  inScope: MapItem[]
  /** Из них физически локализованы (в изоляторе). */
  isolated: number
  /** Отклонения от норм на шагах участка словами (имя шага · порог). */
  anomalies: string[]
  /** Участок — узкое место линии (среднее ожидание). */
  bottleneck: string | null
  status: SectionStatus
}

/** Элементы по локальному имени тега (без префикса пространства имён) — одинаково в браузере и в тестовой среде. */
const byLocal = (root: Document | Element, name: string): Element[] =>
  Array.from(root.getElementsByTagName('*')).filter((e) => (e.tagName.split(':').pop() ?? '') === name)

/** Участки процесса из BPMN: дорожки по порядку, шаги — по flowNodeRef и stepKey узла. */
export function sectionsOf(bpmnXml: string): FlowSection[] {
  if (!bpmnXml || typeof DOMParser === 'undefined') return []
  const doc = new DOMParser().parseFromString(bpmnXml, 'application/xml')
  if (doc.getElementsByTagName('parsererror').length) return []
  // stepKey узла: ant:properties внутри extensionElements узла.
  const stepOf = new Map<string, string>()
  for (const p of byLocal(doc, 'properties')) {
    const key = p.getAttribute('stepKey')
    const node = p.parentElement?.parentElement
    const id = node?.getAttribute('id')
    if (key && id) stepOf.set(id, key)
  }
  return byLocal(doc, 'lane')
    .map((lane) => {
      const refs = byLocal(lane, 'flowNodeRef').map((r) => (r.textContent ?? '').trim())
      const stepKeys = [...new Set(refs.map((id) => stepOf.get(id)).filter((k): k is string => Boolean(k)))]
      return { id: lane.getAttribute('id') ?? '', name: lane.getAttribute('name') ?? '', stepKeys }
    })
    .filter((s) => s.stepKeys.length > 0)
}

const IN_SCOPE = new Set(['confirmed', 'suspect', 'unknown'])

/** Состояние участков по ответу живой карты. */
export function sectionStates(map: LiveMap, sections: readonly FlowSection[]): SectionState[] {
  return sections.map((s) => {
    const keys = new Set(s.stepKeys)
    const counters = map.counters.filter((c) => keys.has(c.step_key))
    const sum = (f: (c: (typeof counters)[number]) => number | undefined) => counters.reduce((a, c) => a + (f(c) ?? 0), 0)
    const items = map.items.filter((i) => keys.has(i.step_key) && i.position !== 'completed')
    const inScope = map.items.filter((i) => keys.has(i.step_key) && i.incident_status && IN_SCOPE.has(i.incident_status))
    const isolated = inScope.filter((i) => i.position === 'isolated').length
    const anomalies = map.anomalies.filter((a) => keys.has(a.step_key)).map((a) => [a.step_name || a.step_key, a.threshold].filter(Boolean).join(' · '))
    const bottleneck = map.bottleneck && keys.has(map.bottleneck.step_key) ? (map.bottleneck.wait ?? '') : null
    const defects = sum((c) => c.defects)
    const notContained = inScope.length - isolated
    const status: SectionStatus =
      notContained > 0 || inScope.some((i) => i.incident_status === 'confirmed' && i.position !== 'isolated')
        ? 'danger'
        : anomalies.length || bottleneck !== null || defects > 0 || inScope.length > 0
          ? 'attention'
          : 'normal'
    return {
      ...s,
      queue: sum((c) => c.queue),
      inProgress: sum((c) => c.in_progress),
      passed: sum((c) => c.passed),
      defects,
      nonconformities: sum((c) => c.nonconformities),
      items,
      inScope,
      isolated,
      anomalies,
      bottleneck,
      status,
    }
  })
}

/** Итог по инциденту: сколько в области на карте и сколько локализовано. */
export function containment(states: readonly SectionState[]): { inScope: number; isolated: number } {
  return states.reduce((a, s) => ({ inScope: a.inScope + s.inScope.length, isolated: a.isolated + s.isolated }), { inScope: 0, isolated: 0 })
}
