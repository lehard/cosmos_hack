// Модель живой карты: разбор схемы по step_key и что рисовать поверх неё
// (FR-1, FR-2, FR-5, FR-9, FR-130, FR-154; NFR-UI-4).
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import NavigatedViewer from 'bpmn-js/lib/NavigatedViewer'
import { parse } from 'yaml'
import { describe, expect, it } from 'vitest'
import { statusPalette } from '@/shared/api/generated/statuses'
import { buildIndex, type BusinessObjectLike } from '../model/bpmn'
import { dotLook, INCIDENT_LEGEND, itemsByStep, itemsOfOtherVersions, itemsPerLane, mapDataState, scopeReduction } from '../model/overlays'
import { flangeXml, frameMorning, frameScope34, frameScope6, V0, V1 } from './fixtures'
import { installSvgStubs } from './svg-env'

installSvgStubs()

/** Индекс схемы так же, как его строит просмотрщик: по бизнес-объектам элементов холста. */
async function flangeIndex() {
  const viewer = new NavigatedViewer({ container: document.createElement('div') })
  await viewer.importXML(flangeXml())
  const all = (viewer.get('elementRegistry') as { getAll(): Array<{ businessObject: BusinessObjectLike }> }).getAll()
  const idx = buildIndex(all.map((e) => e.businessObject))
  viewer.destroy()
  return idx
}

describe('разбор схемы', () => {
  it('каждый узел из списка шагов находится по step_key — без дескриптора moddle', async () => {
    const steps = parse(readFileSync(resolve(__dirname, '../../../../../normative/process/flange-process.steps.yaml'), 'utf8')).steps as Array<{ step_key: string; bpmn_id: string }>
    const idx = await flangeIndex()
    expect(steps.length).toBeGreaterThan(80)
    for (const s of steps) expect(idx.byStepKey.get(s.step_key)?.bpmnId, s.step_key).toBe(s.bpmn_id)
    expect(idx.byBpmnId.size).toBe(steps.length)
  })

  it('описание шага из documentation и цех из дорожки (FR-154, FR-130)', async () => {
    const idx = await flangeIndex()
    const weld = idx.byStepKey.get('welding.weld')!
    expect(weld.name).toBe('Сварка фланца с патрубком')
    expect(weld.documentation.length).toBeGreaterThan(40)
    expect(weld.workshop).toBe('WS-WC')
    expect(weld.laneName).toBe('Сварочный цех')
    expect(idx.lanes.map((l) => l.workshop)).toEqual(['WS-SK', 'WS-MC', 'WS-WC', 'WS-AC', 'WS-QA'])
  })
})

describe('наложения', () => {
  it('изделия прежней версии — только на карте своей версии (FR-1)', () => {
    const f = frameMorning()
    const v1 = itemsByStep(f.items, V1)
    expect(v1.get('welding.weld')?.map((i) => i.label)).toEqual(['ФЛ-0041'])
    expect(itemsByStep(f.items, V0).get('welding.weld')?.map((i) => i.label)).toEqual(['ФЛ-0090'])
    expect(itemsOfOtherVersions(f.items, V1)).toBe(2)
  })

  it('обычный режим — тон сводного статуса; режим инцидента — статус в инциденте (FR-9)', () => {
    const f = frameScope34()
    const confirmed = f.items.find((i) => i.incident_status === 'confirmed')!
    const outside = f.items.find((i) => !i.incident_status)!
    expect(dotLook(confirmed, false)).toMatchObject({ tone: 'danger', statusKey: 'statuses.itemSummary.hold' })
    expect(dotLook(confirmed, true)).toMatchObject({ tone: 'danger', statusKey: 'statuses.incident.confirmed', dimmed: false })
    expect(dotLook(outside, true)).toMatchObject({ dimmed: true })
    const suspect = f.items.find((i) => i.incident_status === 'suspect')!
    expect(dotLook(suspect, true).color).toBe(statusPalette.attention)
  })

  it('легенда инцидента — четыре цвета с подписями', () => {
    expect(INCIDENT_LEGEND.map((l) => [l.status, l.color])).toEqual([
      ['confirmed', statusPalette.danger],
      ['suspect', statusPalette.attention],
      ['excluded', statusPalette.success],
      ['unknown', statusPalette.neutral],
    ])
  })

  it('сокращение области 34 → 6', () => {
    const r = scopeReduction(frameScope6().incident!)
    expect(r).toMatchObject({ from: 34, to: 6 })
    expect(r.fraction).toBeCloseTo(28 / 34)
    expect(scopeReduction({ incident_id: 'x', label: '', scope_version: 1, size_at_creation: 0, size: 0 }).fraction).toBe(0)
  })

  it('изделия по цехам (FR-130)', async () => {
    const idx = await flangeIndex()
    const lanes = itemsPerLane(itemsByStep(frameMorning().items, V1), idx)
    const wc = idx.lanes.find((l) => l.workshop === 'WS-WC')!.bpmnId
    const ac = idx.lanes.find((l) => l.workshop === 'WS-AC')!.bpmnId
    expect(lanes.get(wc)).toBe(12 + 1 + 1)
    expect(lanes.get(ac)).toBe(14 + 8)
  })

  it('четыре состояния: норма / признак дефекта / оценка невозможна (NFR-UI-4)', () => {
    const calm = { ...frameMorning(), bottleneck: null }
    expect(mapDataState(calm)).toBe('normal')
    expect(mapDataState(frameScope34())).toBe('defect_indication')
    expect(mapDataState({ ...calm, data_gaps: ['welding.kt3_camera'] })).toBe('unable_to_assess')
    expect(mapDataState(null)).toBe('normal')
  })
})
