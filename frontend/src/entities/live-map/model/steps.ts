/**
 * Шаги процесса по цехам (эпик 13: стол мастера и терминал исполнителя;
 * AD-17, FR-12, FR-18, FR-130): нормативный слой
 * говорит, какие операции идут в цехе (дорожка BPMN с `ant:properties/@workshop`),
 * их нормы времени и очереди (`ant:norm`) и лимит доработок
 * (`ant:properties/@reworkLimit`). Схема приходит в ответе живой карты
 * (`bpmn_xml`); узел связан с данными по `step_key`, а не по id элемента.
 *
 * Чистый разбор XML (DOMParser), без пространства имён: расширения читаются по
 * локальным именам элементов `properties` и `norm` внутри `extensionElements`.
 */
import { formatMinutes } from '@/shared/lib/duration'

/** Нормы шага (минуты); нет в схеме — null (норма не задана, а не ноль). */
export interface StepNorm {
  timeMinMinutes: number | null
  timeMaxMinutes: number | null
  /** Типовое время операции, если задано одним числом. */
  timeMinutes: number | null
  queueNormMinutes: number | null
}

/** Шаг процесса со свойствами нормативного слоя. */
export interface ProcessStep {
  stepKey: string
  bpmnId: string
  name: string
  /** operation, movement, storage… (`ant:properties/@stepKind`). */
  stepKind: string | null
  operationCode: string | null
  /** Лимит доработок (FR-18) и его область: на изделие, зону, цикл. */
  reworkLimit: number | null
  reworkLimitScope: string | null
  specialProcess: boolean
  norm: StepNorm
  /** Цех дорожки (`WS-…`), если узел в дорожке. */
  workshop: string | null
}

/** Разобранная схема: шаги в порядке документа. */
export interface ProcessSteps {
  steps: ProcessStep[]
  byKey: Map<string, ProcessStep>
}

const EMPTY: ProcessSteps = { steps: [], byKey: new Map() }

const num = (v: string | null): number | null => {
  if (v === null || v.trim() === '') return null
  const n = Number(v)
  return Number.isFinite(n) ? n : null
}

/** Прямой потомок `extensionElements` с локальным именем `name`. */
function extension(el: Element, name: string): Element | null {
  for (const ext of Array.from(el.children)) {
    if (ext.localName !== 'extensionElements') continue
    for (const c of Array.from(ext.children)) if (c.localName === name) return c
  }
  return null
}

/**
 * Разобрать BPMN XML версии процесса.
 * @param xml — `bpmn_xml` ответа `process.live_map.read`
 */
export function parseProcessSteps(xml: string | null | undefined): ProcessSteps {
  if (!xml) return EMPTY
  let doc: Document
  try {
    doc = new DOMParser().parseFromString(xml, 'application/xml')
  } catch {
    return EMPTY
  }
  if (doc.getElementsByTagName('parsererror').length) return EMPTY
  const all = Array.from(doc.getElementsByTagName('*'))

  // Дорожки-цеха: id элемента → цех.
  const workshopOfNode = new Map<string, string>()
  for (const lane of all.filter((e) => e.localName === 'lane')) {
    const ws = extension(lane, 'properties')?.getAttribute('workshop') ?? null
    if (!ws) continue
    for (const ref of Array.from(lane.children).filter((c) => c.localName === 'flowNodeRef')) {
      const id = ref.textContent?.trim()
      if (id) workshopOfNode.set(id, ws)
    }
  }

  const steps: ProcessStep[] = []
  const byKey = new Map<string, ProcessStep>()
  for (const el of all) {
    const props = extension(el, 'properties')
    const stepKey = props?.getAttribute('stepKey')
    if (!props || !stepKey || byKey.has(stepKey)) continue
    const norm = extension(el, 'norm')
    const id = el.getAttribute('id') ?? ''
    const step: ProcessStep = {
      stepKey,
      bpmnId: id,
      name: el.getAttribute('name') ?? stepKey,
      stepKind: props.getAttribute('stepKind'),
      operationCode: props.getAttribute('operationCode'),
      reworkLimit: num(props.getAttribute('reworkLimit')),
      reworkLimitScope: props.getAttribute('reworkLimitScope'),
      specialProcess: props.getAttribute('specialProcess') === 'true',
      norm: {
        timeMinMinutes: num(norm?.getAttribute('timeMinMinutes') ?? null),
        timeMaxMinutes: num(norm?.getAttribute('timeMaxMinutes') ?? null),
        timeMinutes: num(norm?.getAttribute('timeMinutes') ?? null),
        queueNormMinutes: num(norm?.getAttribute('queueNormMinutes') ?? null),
      },
      workshop: workshopOfNode.get(id) ?? null,
    }
    steps.push(step)
    byKey.set(stepKey, step)
  }
  return { steps, byKey }
}

/** Операции цеха (шаги вида `operation`); цех не задан — все операции. */
export function operationsOf(parsed: ProcessSteps, workshop: string | null): ProcessStep[] {
  return parsed.steps.filter((s) => s.stepKind === 'operation' && (!workshop || s.workshop === workshop))
}

/** Верхняя граница нормы времени операции: max, иначе типовое время. */
export const normCeiling = (n: StepNorm): number | null => n.timeMaxMinutes ?? n.timeMinutes

/** Выше верхней границы нормы; нормы нет или длительность неизвестна — null. */
export function overNorm(minutes: number | null, norm: StepNorm): boolean | null {
  const ceiling = normCeiling(norm)
  if (minutes === null || ceiling === null) return null
  return minutes > ceiling
}

/**
 * Сколько минут идёт операция к моменту просмотра (вычислено интерфейсом от
 * записи «операция начата»; смысл интервала — с начала выполнения).
 * @param startedAt — начало выполнения (RFC 3339)
 * @param at — момент просмотра (воспроизведение) или «сейчас»
 */
export function elapsedMinutes(startedAt: string, at: Date): number | null {
  const start = Date.parse(startedAt)
  if (Number.isNaN(start)) return null
  return Math.max(0, Math.round((at.getTime() - start) / 60000))
}

/** «норма 40–90 мин», «норма 75 мин» или «норма не задана» (тексты widgets.shopFloor.station). */
export function normText(t: (key: string, params?: Record<string, unknown>) => string, n: StepNorm | null | undefined): string {
  if (!n) return t('widgets.shopFloor.station.noNorm')
  if (n.timeMinMinutes !== null && n.timeMaxMinutes !== null) {
    return t('widgets.shopFloor.station.normRange', { min: formatMinutes(t, n.timeMinMinutes), max: formatMinutes(t, n.timeMaxMinutes) })
  }
  const one = n.timeMaxMinutes ?? n.timeMinutes
  return one !== null ? t('widgets.shopFloor.station.norm', { value: formatMinutes(t, one) }) : t('widgets.shopFloor.station.noNorm')
}
