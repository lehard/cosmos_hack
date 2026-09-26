/**
 * Правка наших свойств и описания элемента в модельере bpmn-js (FR-25):
 * все изменения — командами `modeling` (отмена и повтор модельера работают),
 * наши элементы — типы дескриптора `ant:*` (model/properties.ts).
 */
import { moddleType, type BusinessObjectLike, type ModdleLike } from './properties'

/** Элемент схемы, выбранный в модельере. */
export interface EditorElement {
  id: string
  type: string
  businessObject: BusinessObjectLike
}

/** Что панель свойств может сделать с выбранным элементом. */
export interface ElementEditor {
  setName(el: EditorElement, name: string): void
  setDocumentation(el: EditorElement, text: string): void
  /** Свойство `name` у `index`-го элемента типа `type`; нет элемента — создаётся. */
  setField(el: EditorElement, type: string, index: number, name: string, value: string | number | boolean | undefined): void
  addEntry(el: EditorElement, type: string): void
  removeEntry(el: EditorElement, type: string, index: number): void
}

/** Минимум служб bpmn-js, нужный правке. */
export interface ModelerServices {
  modeling: {
    updateModdleProperties(element: EditorElement, target: ModdleLike, props: Record<string, unknown>): void
    updateProperties(element: EditorElement, props: Record<string, unknown>): void
  }
  moddle: { create(type: string, attrs?: Record<string, unknown>): ModdleLike }
}

type Ext = ModdleLike & { values?: ModdleLike[] }

/** Правка над службами модельера. */
export function createEditor(svc: ModelerServices): ElementEditor {
  const { modeling, moddle } = svc

  const ext = (el: EditorElement): Ext => {
    const bo = el.businessObject
    let e = bo.extensionElements as Ext | null | undefined
    if (!e) {
      e = moddle.create('bpmn:ExtensionElements', { values: [] }) as Ext
      modeling.updateModdleProperties(el, bo, { extensionElements: e })
    }
    return e
  }
  const entries = (e: Ext, type: string) => (e.values ?? []).filter((v) => v.$type === moddleType(type))
  const add = (el: EditorElement, e: Ext, type: string): ModdleLike => {
    const x = moddle.create(moddleType(type), {})
    x.$parent = e
    modeling.updateModdleProperties(el, e, { values: [...(e.values ?? []), x] })
    return x
  }

  return {
    setName(el, name) {
      modeling.updateProperties(el, { name: name === '' ? undefined : name })
    },
    setDocumentation(el, text) {
      const docs = text.trim() === '' ? [] : [moddle.create('bpmn:Documentation', { text })]
      modeling.updateModdleProperties(el, el.businessObject, { documentation: docs })
    },
    setField(el, type, index, name, value) {
      const e = ext(el)
      const target = entries(e, type)[index] ?? add(el, e, type)
      modeling.updateModdleProperties(el, target, { [name]: value })
    },
    addEntry(el, type) {
      add(el, ext(el), type)
    },
    removeEntry(el, type, index) {
      const e = ext(el)
      const target = entries(e, type)[index]
      if (!target) return
      modeling.updateModdleProperties(el, e, { values: (e.values ?? []).filter((v) => v !== target) })
    },
  }
}
