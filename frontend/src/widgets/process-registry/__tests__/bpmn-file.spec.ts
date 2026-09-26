/** Файлы процесса: пустой процесс, чтение загруженного .bpmn, имя файла. */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { blankProcessXml, bpmnFileName, BpmnFileError, newProcessId, readBpmnFile } from '../model/bpmn-file'

describe('файлы процесса', () => {
  it('пустой процесс — новый процесс с нашими свойствами у каждого узла', () => {
    const xml = blankProcessXml('Process_NEW', 'Корпус "датчика" <А>')
    const info = readBpmnFile(xml)
    expect(info).toEqual({ processId: 'Process_NEW', name: 'Корпус "датчика" <А>', hasAntExtension: true })
    const doc = new DOMParser().parseFromString(xml, 'application/xml')
    expect(doc.getElementsByTagNameNS('urn:ant:bpmn-ext:1', 'properties')).toHaveLength(3)
  })

  it('главный процесс фланца — не вызываемый подпроцесс «Брак»', () => {
    const xml = readFileSync(resolve(__dirname, '../../../../../normative/process/flange-process.bpmn'), 'utf8')
    expect(readBpmnFile(xml)).toMatchObject({ processId: 'Process_Flange', name: 'Фланец люка гермокорпуса в сборе', hasAntExtension: true })
  })

  it('не XML и не BPMN — понятная ошибка', () => {
    expect(() => readBpmnFile('<<<')).toThrow(BpmnFileError)
    expect(() => readBpmnFile('<a xmlns="urn:x"/>')).toThrow(expect.objectContaining({ key: 'notBpmn' }))
  })

  it('id нового процесса и имя файла', () => {
    expect(newProcessId(36 ** 3)).toBe('Process_1000')
    expect(bpmnFileName('Process_Flange', 'v 2')).toBe('Process_Flange-v_2.bpmn')
  })
})
