/**
 * Файлы процесса (эпик 39; FR-10, FR-25): пустой процесс «старт → контроль → конец» для
 * «Создать процесс», чтение загруженного .bpmn (какой процесс в нём) и
 * скачивание версии как файла. Процесс определяется id главного
 * `bpmn:process` (UI-11): новый id — новый процесс, тот же — новая версия.
 * Проверку наших расширений `urn:ant:bpmn-ext:1` делает сервер при загрузке
 * черновика (FR-13) — отказ с кодом и id элемента.
 */

const BPMN_NS = 'http://www.omg.org/spec/BPMN/20100524/MODEL'
export const ANT_NS = 'urn:ant:bpmn-ext:1'

const esc = (s: string): string => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')

/**
 * id нового процесса: `Process_‹метка времени base36›` — латиница, как у
 * id элементов BPMN; уникален в пределах предприятия на практике.
 */
export const newProcessId = (now: number = Date.now()): string => `Process_${now.toString(36).toUpperCase()}`

/**
 * Пустой процесс «старт → контроль ОТК → конец» с нашими свойствами и раскладкой BPMNDI
 * (проверяется сервером так же, как любой черновик; копия — в Go-тесте
 * application/process/approval_test.go).
 */
export function blankProcessXml(processId: string, name: string): string {
  const id = esc(processId)
  return `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="${BPMN_NS}" xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI" xmlns:dc="http://www.omg.org/spec/DD/20100524/DC" xmlns:di="http://www.omg.org/spec/DD/20100524/DI" xmlns:ant="${ANT_NS}" id="Definitions_${id}" targetNamespace="urn:ant:process:${id}" expressionLanguage="urn:ant:expr:1">
  <bpmn:process id="${id}" name="${esc(name)}" isExecutable="true">
    <bpmn:startEvent id="Start" name="Начало">
      <bpmn:extensionElements>
        <ant:properties stepKey="start" />
      </bpmn:extensionElements>
      <bpmn:outgoing>Flow_Start_Check</bpmn:outgoing>
    </bpmn:startEvent>
    <bpmn:userTask id="Check" name="Контроль ОТК">
      <bpmn:documentation>Контроль человеком перед выпуском: без него путь к конечному событию не проходит проверку (FR-13).</bpmn:documentation>
      <bpmn:extensionElements>
        <ant:properties stepKey="inspection.final" stepKind="human_inspection" />
      </bpmn:extensionElements>
      <bpmn:incoming>Flow_Start_Check</bpmn:incoming>
      <bpmn:outgoing>Flow_Check_End</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:endEvent id="End" name="Конец">
      <bpmn:extensionElements>
        <ant:properties stepKey="end" />
      </bpmn:extensionElements>
      <bpmn:incoming>Flow_Check_End</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:sequenceFlow id="Flow_Start_Check" sourceRef="Start" targetRef="Check" />
    <bpmn:sequenceFlow id="Flow_Check_End" sourceRef="Check" targetRef="End" />
  </bpmn:process>
  <bpmndi:BPMNDiagram id="BPMNDiagram_${id}">
    <bpmndi:BPMNPlane id="BPMNPlane_${id}" bpmnElement="${id}">
      <bpmndi:BPMNShape id="Start_di" bpmnElement="Start">
        <dc:Bounds x="180" y="160" width="36" height="36" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Check_di" bpmnElement="Check">
        <dc:Bounds x="280" y="138" width="120" height="80" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="End_di" bpmnElement="End">
        <dc:Bounds x="480" y="160" width="36" height="36" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNEdge id="Flow_Start_Check_di" bpmnElement="Flow_Start_Check">
        <di:waypoint x="216" y="178" />
        <di:waypoint x="280" y="178" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_Check_End_di" bpmnElement="Flow_Check_End">
        <di:waypoint x="400" y="178" />
        <di:waypoint x="480" y="178" />
      </bpmndi:BPMNEdge>
    </bpmndi:BPMNPlane>
  </bpmndi:BPMNDiagram>
</bpmn:definitions>
`
}

/** Что в загруженном файле. */
export interface BpmnFileInfo {
  /** id главного bpmn:process (исполняемый и не вызываемый callActivity). */
  processId: string
  name: string
  /** Есть ли у элементов наши свойства (пространство urn:ant:bpmn-ext:1). */
  hasAntExtension: boolean
}

/** Ошибка чтения файла: ключ текста интерфейса. */
export class BpmnFileError extends Error {
  constructor(readonly key: 'notXml' | 'notBpmn' | 'noProcess') {
    super(key)
  }
}

/**
 * Разобрать .bpmn: главный процесс и наличие наших свойств. Полную проверку
 * (FR-13) делает сервер; здесь — только чтобы сказать «это новый процесс»
 * или «новая версия процесса …» до отправки.
 */
export function readBpmnFile(xml: string): BpmnFileInfo {
  const doc = new DOMParser().parseFromString(xml, 'application/xml')
  if (doc.getElementsByTagName('parsererror').length) throw new BpmnFileError('notXml')
  const root = doc.documentElement
  if (root.localName !== 'definitions' || root.namespaceURI !== BPMN_NS) throw new BpmnFileError('notBpmn')
  const called = new Set<string>()
  for (const c of Array.from(doc.getElementsByTagNameNS(BPMN_NS, 'callActivity'))) called.add(c.getAttribute('calledElement') ?? '')
  const processes = Array.from(doc.getElementsByTagNameNS(BPMN_NS, 'process'))
  const main = processes.find((p) => p.getAttribute('isExecutable') !== 'false' && !called.has(p.getAttribute('id') ?? '')) ?? processes[0]
  const id = main?.getAttribute('id')
  if (!main || !id) throw new BpmnFileError('noProcess')
  return { processId: id, name: main.getAttribute('name') ?? id, hasAntExtension: doc.getElementsByTagNameNS(ANT_NS, 'properties').length > 0 }
}

/** Имя файла версии: `‹процесс›-‹метка›.bpmn`. */
export const bpmnFileName = (processId: string, label: string): string => `${processId}-${label}`.replace(/[^\p{L}\p{N}._-]+/gu, '_') + '.bpmn'

/** Скачать текст как файл (браузер). */
export function downloadText(text: string, fileName: string): void {
  const url = URL.createObjectURL(new Blob([text], { type: 'application/xml' }))
  const a = document.createElement('a')
  a.href = url
  a.download = fileName
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
}
