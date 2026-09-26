/** Слой «нормы» (FR-156): опоры стартового процесса фланца по step_key. */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { parseNorms } from '../model/norms'

const flange = readFileSync(resolve(__dirname, '../../../../../normative/process/flange-process.bpmn'), 'utf8')

describe('опоры шагов из BPMN', () => {
  it('у фланца 33 узла с опорами, 53 опоры (вход процессной сессии)', () => {
    const m = parseNorms(flange)
    expect(m.size).toBe(33)
    expect([...m.values()].reduce((n, s) => n + s.anchors.length, 0)).toBe(53)
  })

  it('у опоры — стандарт, пункт, что проверить и что делает система', () => {
    const m = parseNorms(flange)
    const weld = m.get('welding.weld')
    expect(weld).toBeDefined()
    for (const a of weld!.anchors) {
      expect(a.standard).toMatch(/ГОСТ/)
      expect(a.clause).not.toBe('')
      expect(a.check.length + a.systemAction.length).toBeGreaterThan(0)
    }
  })

  it('битый XML — пустой слой', () => {
    expect(parseNorms('<<<').size).toBe(0)
    expect(parseNorms('').size).toBe(0)
  })
})
