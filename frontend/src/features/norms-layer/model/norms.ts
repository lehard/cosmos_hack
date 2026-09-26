/**
 * Слой «нормы» живой карты (FR-156, PRD §11.17; AD-17): нормативные опоры
 * шагов — повторяемый элемент `ant:normRef` (`standard`, `clause`,
 * `<ant:check>` — что проверить, `<ant:systemAction>` — что делает система) в
 * `extensionElements` узла BPMN показанной версии. Отдельного файла и запроса
 * нет: опоры читаются из того же XML, что рисует карта (принцип одного
 * формата, FR-10). Узлы — по step_key (`ant:properties/@stepKey`).
 */

/** Нормативная опора шага. */
export interface NormAnchor {
  standard: string
  clause: string
  /** Что проверить на шаге. */
  check: string
  /** Что делает система на шаге. */
  systemAction: string
}

/** Опоры шага и его название. */
export interface StepNorms {
  stepKey: string
  elementId: string
  name: string
  anchors: NormAnchor[]
}

// Элементы — по локальному имени (префиксы в файлах разные, а разбор XML в
// среде тестов не везде знает пространства имён), как entities/live-map/model/steps.ts.
const kids = (el: Element, local: string): Element[] => Array.from(el.children).filter((c) => c.localName === local)
const text = (el: Element, local: string): string => kids(el, local)[0]?.textContent?.trim() ?? ''

/**
 * Опоры по step_key из BPMN XML версии; узлы без опор в карту не попадают.
 * Битый XML — пустая карта (слой просто ничего не подсвечивает).
 */
export function parseNorms(xml: string): Map<string, StepNorms> {
  const out = new Map<string, StepNorms>()
  if (!xml) return out
  let doc: Document
  try {
    doc = new DOMParser().parseFromString(xml, 'application/xml')
  } catch {
    return out
  }
  if (doc.getElementsByTagName('parsererror').length) return out
  for (const ext of Array.from(doc.getElementsByTagName('*')).filter((e) => e.localName === 'extensionElements')) {
    const refs = kids(ext, 'normRef')
    if (!refs.length) continue
    const stepKey = kids(ext, 'properties')[0]?.getAttribute('stepKey')
    const node = ext.parentElement
    if (!stepKey || !node) continue
    out.set(stepKey, {
      stepKey,
      elementId: node.getAttribute('id') ?? '',
      name: node.getAttribute('name') ?? stepKey,
      anchors: refs.map((r) => ({
        standard: r.getAttribute('standard') ?? '',
        clause: r.getAttribute('clause') ?? '',
        check: text(r, 'check'),
        systemAction: text(r, 'systemAction'),
      })),
    })
  }
  return out
}
