/**
 * Читаемая разница версии процесса с действующей (FR-24): «добавлена точка
 * предъявления после сварки», «порог уверенности для вида X изменён».
 * Элементы сопоставляются по смысловому ключу шага, без ключа — по id BPMN.
 */
import type { ProcessDiffEntry, ProcessElement, ProcessPropertyKey, ProcessPropertyValue, ProcessVersion } from './types'

const keyOf = (e: ProcessElement): string => e.step_key ?? e.id

const same = (a: ProcessPropertyValue | undefined, b: ProcessPropertyValue | undefined): boolean => (a ?? null) === (b ?? null)

/** Предшественник элемента по переходам версии. */
const predecessorOf = (v: ProcessVersion, id: string): ProcessElement | undefined => v.elements.find((e) => e.next?.includes(id))

/**
 * Отличия `next` от `base` в порядке элементов маршрута: сначала изменения и
 * добавления по порядку `next`, затем удалённые.
 */
export function diffVersions(base: ProcessVersion, next: ProcessVersion): ProcessDiffEntry[] {
  const out: ProcessDiffEntry[] = []
  const baseByKey = new Map(base.elements.map((e) => [keyOf(e), e]))
  const nextKeys = new Set(next.elements.map(keyOf))

  for (const e of next.elements) {
    const old = baseByKey.get(keyOf(e))
    if (!old) {
      // Новая точка предъявления — «после» предыдущего шага маршрута.
      if (e.properties.presentationPoint) out.push({ kind: 'presentationPointAdded', step: predecessorOf(next, e.id)?.name ?? e.name })
      else out.push({ kind: 'elementAdded', element: e.name })
      continue
    }
    const props = new Set([...Object.keys(old.properties), ...Object.keys(e.properties)] as ProcessPropertyKey[])
    for (const p of props) {
      const from = old.properties[p]
      const to = e.properties[p]
      if (same(from, to)) continue
      if (p === 'presentationPoint' && to === true && !from) out.push({ kind: 'presentationPointAdded', step: e.name })
      else out.push({ kind: 'propertyChanged', element: e.name, property: p, from: from ?? null, to: to ?? null })
    }
    const types = new Set([...Object.keys(old.thresholds ?? {}), ...Object.keys(e.thresholds ?? {})])
    for (const t of [...types].sort()) {
      const from = old.thresholds?.[t] ?? null
      const to = e.thresholds?.[t] ?? null
      if (from !== to) out.push({ kind: 'thresholdChanged', element: e.name, defectType: t, from, to })
    }
  }
  for (const e of base.elements) if (!nextKeys.has(keyOf(e))) out.push({ kind: 'elementRemoved', element: e.name })
  return out
}

/** Действующая версия списка; undefined — нет. */
export const activeVersion = (vs: readonly ProcessVersion[]): ProcessVersion | undefined => vs.find((v) => v.status === 'active')
