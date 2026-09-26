/**
 * Слой «нормы» живой карты (FR-156, PRD §11.17; AD-17): нормативные опоры
 * шагов — повторяемый элемент `ant:normRef` (`standard`, `clause`,
 * `<ant:check>` — что проверить, `<ant:systemAction>` — что делает система) в
 * `extensionElements` узла BPMN показанной версии. Отдельного файла и запроса
 * нет: опоры читаются из того же XML, что рисует карта (принцип одного
 * формата, FR-10). Узлы — по step_key (`ant:properties/@stepKey`).
 */

const ANT_NS = 'urn:ant:bpmn-ext:1'
const BPMN_NS = 'http://www.omg.org/spec/BPMN/20100524/MODEL'

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

const text = (el: Element, local: string): string => el.getElementsByTagNameNS(ANT_NS, local)[0]?.textContent?.trim() ?? ''

/**
 * Опоры по step_key из BPMN XML версии; узлы без опор в карту не попадают.
 * Битый XML — пустая карта (слой просто ничего не подсвечивает).
 */
export function parseNorms(xml: string): Map<string, StepNorms> {
  const out = new Map<string, StepNorms>()
  if (!xml) return out
  const doc = new DOMParser().parseFromString(xml, 'application/xml')
  if (doc.getElementsByTagName('parsererror').length) return out
  for (const ext of Array.from(doc.getElementsByTagNameNS(BPMN_NS, 'extensionElements'))) {
    const refs = Array.from(ext.children).filter((c) => c.namespaceURI === ANT_NS && c.localName === 'normRef')
    if (!refs.length) continue
    const props = Array.from(ext.children).find((c) => c.namespaceURI === ANT_NS && c.localName === 'properties')
    const stepKey = props?.getAttribute('stepKey')
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
