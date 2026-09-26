/**
 * Провайдер панели сверяется с дескриптором (FR-25, AD-20): копия дескриптора
 * у модельера совпадает с контрактом, у каждого типа дескриптора — группа
 * панели с теми же свойствами тех же видов, перечисления — из rules.yaml.
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { parse } from 'yaml'
import copy from '../model/ant-moddle.json'
import { coerce, entriesOf, fieldValue, PANEL_GROUPS, stepKeyOf, type BusinessObjectLike, type FieldKind } from '../model/properties'

const REPO = resolve(__dirname, '../../../../..')
const contract = JSON.parse(readFileSync(resolve(REPO, 'contracts/bpmn-ext/ant.json'), 'utf8')) as typeof copy
const rules = parse(readFileSync(resolve(REPO, 'contracts/bpmn-ext/rules.yaml'), 'utf8')) as {
  enums: Record<string, string[]>
  step_kind: { values: string[] }
}

const kindOf = (p: { type: string; isAttr?: boolean }): FieldKind[] => {
  switch (p.type) {
    case 'Boolean':
      return ['boolean']
    case 'Integer':
      return ['integer']
    default:
      // Строка-атрибут — однострочное поле; строка-элемент (<ant:check>) — текст.
      return p.isAttr ? ['string'] : ['text']
  }
}

describe('панель наших свойств ↔ дескриптор urn:ant:bpmn-ext:1', () => {
  it('копия дескриптора у модельера совпадает с contracts/bpmn-ext/ant.json', () => {
    expect(copy).toEqual(contract)
    expect(copy.uri).toBe('urn:ant:bpmn-ext:1')
  })

  it('у каждого типа дескриптора — группа с теми же свойствами и видами', () => {
    const groups = new Map(PANEL_GROUPS.map((g) => [g.type, g]))
    expect([...groups.keys()].sort()).toEqual(contract.types.map((t) => t.name).sort())
    for (const type of contract.types) {
      const g = groups.get(type.name)!
      expect(g.fields.map((f) => f.name).sort(), type.name).toEqual(type.properties.map((p) => p.name).sort())
      for (const p of type.properties) {
        const f = g.fields.find((x) => x.name === p.name)!
        expect(kindOf(p), `${type.name}.${p.name}`).toContain(f.kind)
      }
    }
  })

  it('списки значений — из rules.yaml', () => {
    const byField: Record<string, string[]> = {
      'Properties.stepKind': rules.step_kind.values,
      'Properties.reworkLimitScope': rules.enums['properties.reworkLimitScope']!,
      'Properties.erpAction': rules.enums['properties.erpAction']!,
      'Properties.timerScope': rules.enums['properties.timerScope']!,
      'Properties.outcome': rules.enums['properties.outcome']!,
      'Properties.closingPoint': rules.enums['properties.closingPoint']!,
      'Inspection.method': rules.enums['inspection.method']!,
      'Inspection.phase': rules.enums['inspection.phase']!,
      'Precondition.kind': rules.enums['precondition.kind']!,
      'Precondition.mode': rules.enums['precondition.mode']!,
    }
    const withOptions = PANEL_GROUPS.flatMap((g) => g.fields.filter((f) => f.options).map((f) => [`${g.type}.${f.name}`, f.options!] as const))
    expect(withOptions.map(([k]) => k).sort()).toEqual(Object.keys(byField).sort())
    for (const [k, opts] of withOptions) expect([...opts], k).toEqual(byField[k])
  })
})

describe('чтение и приведение значений', () => {
  const bo: BusinessObjectLike = {
    $type: 'bpmn:Task',
    extensionElements: {
      values: [
        { $type: 'ant:Properties', stepKey: 'welding.weld', specialProcess: true, reworkLimit: 2 },
        { $type: 'ant:NormRef', standard: 'ГОСТ Р 58633', clause: '5.4', check: 'Режимы сварки' },
        { $type: 'ant:NormRef', standard: 'ГОСТ Р 58876', clause: '8.5.1' },
      ],
    },
  }

  it('читает группы и поля узла', () => {
    expect(stepKeyOf(bo)).toBe('welding.weld')
    expect(entriesOf(bo, 'NormRef')).toHaveLength(2)
    const props = PANEL_GROUPS[0]!
    const el = entriesOf(bo, 'Properties')[0]!
    expect(fieldValue(el, props.fields.find((f) => f.name === 'specialProcess')!)).toBe(true)
    expect(fieldValue(el, props.fields.find((f) => f.name === 'reworkLimit')!)).toBe(2)
    expect(fieldValue(el, props.fields.find((f) => f.name === 'workshop')!)).toBeNull()
  })

  it('пустой ввод снимает свойство, число — только целое', () => {
    const int = { name: 'reworkLimit', kind: 'integer' } as const
    expect(coerce(int, '3')).toBe(3)
    expect(coerce(int, 'abc')).toBeUndefined()
    expect(coerce({ name: 'stepKey', kind: 'string' }, '  ')).toBeUndefined()
    expect(coerce({ name: 'specialProcess', kind: 'boolean' }, false)).toBeUndefined()
    expect(coerce({ name: 'specialProcess', kind: 'boolean' }, true)).toBe(true)
  })
})
