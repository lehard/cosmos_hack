/**
 * Разбор схемы BPMN для живой карты (FR-1, FR-130, FR-154; AD-17, AD-21).
 *
 * Узел карты связан с данными по `step_key` из `ant:properties/@stepKey`
 * (contracts/bpmn-ext/ant.json), а не по id элемента BPMN: изделия прежних версий
 * показываются на узлах своего step_key (normative/process/flange-process.steps.yaml).
 * Дескриптор moddle в сборку интерфейса не входит (в образ копируется только
 * frontend/), поэтому свойства читаются из «общих» элементов расширения, которые
 * bpmn-moddle сохраняет для незнакомого пространства имён, — без предупреждений.
 */

/** Минимум бизнес-объекта bpmn-moddle, который нужен карте. */
export interface BusinessObjectLike {
  $type: string
  id?: string
  name?: string
  documentation?: Array<{ text?: string }>
  extensionElements?: { values?: Array<Record<string, unknown> & { $type?: string }> }
  flowNodeRef?: Array<{ id?: string }>
}

/** Узел схемы с шагом процесса. */
export interface StepNode {
  /** id элемента BPMN (на него ставятся наложения и метки). */
  bpmnId: string
  stepKey: string
  name: string
  bpmnType: string
  /** Описание шага из `bpmn:documentation` (FR-154). */
  documentation: string
  /** Цех (`ant:properties/@workshop` дорожки), если узел в дорожке. */
  workshop: string | null
  /** id дорожки-цеха и её название. */
  laneId: string | null
  laneName: string | null
}

/** Дорожка-цех (FR-130). */
export interface LaneInfo {
  bpmnId: string
  name: string
  workshop: string | null
}

/** Свойство `ant:properties/@attr` элемента или null. */
export function antProperty(bo: BusinessObjectLike | undefined, attr: string): string | null {
  for (const ext of bo?.extensionElements?.values ?? []) {
    if (ext.$type !== 'ant:properties') continue
    const v = ext[attr] ?? (ext.$attrs as Record<string, unknown> | undefined)?.[attr]
    if (typeof v === 'string' && v) return v
  }
  return null
}

/** Текст `documentation` (все блоки через пустую строку), обрезанный по краям. */
export function documentationOf(bo: BusinessObjectLike | undefined): string {
  return (bo?.documentation ?? [])
    .map((d) => (d.text ?? '').trim())
    .filter(Boolean)
    .join('\n\n')
}

/** Индекс схемы: узлы по id элемента и по step_key, дорожки. */
export interface DiagramIndex {
  byBpmnId: Map<string, StepNode>
  byStepKey: Map<string, StepNode>
  lanes: LaneInfo[]
}

/**
 * Построить индекс по бизнес-объектам элементов холста.
 * @param elements — бизнес-объекты всех элементов (elementRegistry → businessObject)
 */
export function buildIndex(elements: Iterable<BusinessObjectLike>): DiagramIndex {
  const all = [...elements]
  const lanes: LaneInfo[] = []
  const laneOf = new Map<string, LaneInfo>()
  for (const bo of all) {
    if (bo.$type !== 'bpmn:Lane' || !bo.id) continue
    const lane: LaneInfo = { bpmnId: bo.id, name: bo.name ?? '', workshop: antProperty(bo, 'workshop') }
    lanes.push(lane)
    for (const ref of bo.flowNodeRef ?? []) if (ref.id) laneOf.set(ref.id, lane)
  }
  const byBpmnId = new Map<string, StepNode>()
  const byStepKey = new Map<string, StepNode>()
  for (const bo of all) {
    const stepKey = antProperty(bo, 'stepKey')
    if (!stepKey || !bo.id || byBpmnId.has(bo.id)) continue
    const lane = laneOf.get(bo.id) ?? null
    const node: StepNode = {
      bpmnId: bo.id,
      stepKey,
      name: bo.name ?? '',
      bpmnType: bo.$type,
      documentation: documentationOf(bo),
      workshop: antProperty(bo, 'workshop') ?? lane?.workshop ?? null,
      laneId: lane?.bpmnId ?? null,
      laneName: lane?.name ?? null,
    }
    byBpmnId.set(bo.id, node)
    // step_key уникален в версии (AD-17); если нет — первый выигрывает, карта не падает.
    if (!byStepKey.has(stepKey)) byStepKey.set(stepKey, node)
  }
  return { byBpmnId, byStepKey, lanes }
}
