/** Панель наших свойств (FR-25): чтение значений узла и правка через модельер. */
import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { describe, expect, it, vi } from 'vitest'
import { i18n } from '@/shared/i18n'
import type { EditorElement, ElementEditor } from '../model/editor'
import { createEditor } from '../model/editor'
import PropertiesPanel from '../ui/PropertiesPanel.vue'

const element = (): EditorElement => ({
  id: 'W2',
  type: 'bpmn:Task',
  businessObject: {
    $type: 'bpmn:Task',
    name: 'Сварка',
    documentation: [{ text: 'Сварка фланца с патрубком' }],
    extensionElements: {
      values: [
        { $type: 'ant:Properties', stepKey: 'welding.weld', stepKind: 'operation', specialProcess: true },
        { $type: 'ant:NormRef', standard: 'ГОСТ Р 58633', clause: '3.17', check: 'Режимы' },
      ],
    },
  },
})

const editorSpy = (): ElementEditor => ({ setName: vi.fn(), setDocumentation: vi.fn(), setField: vi.fn(), addEntry: vi.fn(), removeEntry: vi.fn() })
const mountPanel = (editor: ElementEditor | null) =>
  mount(PropertiesPanel, { props: { element: element(), editor, revision: 0 }, global: { plugins: [createPinia(), i18n] } })

describe('панель наших свойств', () => {
  it('черновик: все группы дескриптора, значения узла, правка — командой модельера', async () => {
    const ed = editorSpy()
    const w = mountPanel(ed)
    expect(w.findAll('[data-group]')).toHaveLength(10)
    expect((w.find('[data-field="documentation"] textarea').element as HTMLTextAreaElement).value).toBe('Сварка фланца с патрубком')
    const key = w.find('[data-group="Properties"] [data-field="stepKey"] input')
    expect((key.element as HTMLInputElement).value).toBe('welding.weld')
    await key.setValue('welding.weld_v2')
    await key.trigger('change')
    expect(ed.setField).toHaveBeenCalledWith(expect.objectContaining({ id: 'W2' }), 'Properties', 0, 'stepKey', 'welding.weld_v2')
    await w.find('[data-add="NormRef"]').trigger('click')
    expect(ed.addEntry).toHaveBeenCalledWith(expect.objectContaining({ id: 'W2' }), 'NormRef')
  })

  it('не черновик: только заполненные группы и только просмотр', () => {
    const w = mountPanel(null)
    expect(w.findAll('[data-group]').map((g) => g.attributes('data-group'))).toEqual(['Properties', 'NormRef'])
    expect(w.text()).toContain('Только просмотр')
  })
})

describe('правка через службы модельера', () => {
  it('нет элемента типа — создаётся в extensionElements, затем свойство', () => {
    const calls: Array<[string, Record<string, unknown>]> = []
    const created: Array<Record<string, unknown>> = []
    const svc = {
      modeling: {
        updateModdleProperties: (_el: unknown, target: { $type: string }, props: Record<string, unknown>) => {
          calls.push([target.$type, props])
          Object.assign(target, props)
        },
        updateProperties: vi.fn(),
      },
      moddle: {
        create: (type: string, attrs: Record<string, unknown> = {}) => {
          const x = { $type: type, ...attrs }
          created.push(x)
          return x
        },
      },
    }
    const el: EditorElement = { id: 'T1', type: 'bpmn:Task', businessObject: { $type: 'bpmn:Task' } }
    createEditor(svc).setField(el, 'Norm', 0, 'timeMinutes', 30)
    expect(created.map((c) => c.$type)).toEqual(['bpmn:ExtensionElements', 'ant:Norm'])
    expect(calls.at(-1)).toEqual(['ant:Norm', { timeMinutes: 30 }])
    createEditor(svc).setDocumentation(el, 'Описание')
    expect(calls.at(-1)?.[1]).toHaveProperty('documentation')
  })
})
