// Правила показателей на экране (кейс §2.4, §5.2; FR-86…FR-89): единицы,
// происхождение времени, раскладка раздельного учёта, сверка раскрытия,
// геометрия контрольной карты.
import { describe, expect, it } from 'vitest'
import { i18n } from '@/shared/i18n'
import { accountOf, chartGeometry, deltaOf, formatValue, originOf, sumCheck, toOverviewModel } from '../index'
import { chart, drilldown, overview } from './fixtures'

const x = {
  t: i18n.global.t as (k: string, p?: Record<string, unknown>) => string,
  n: (v: number, f: string) => i18n.global.n(v, f),
}
const norm = (s: string) => s.replace(/[\u00a0\u202f]/g, ' ')

describe('значение показателя', () => {
  it('штуки, доли в базисных пунктах, минуты, амперы, неизвестная единица', () => {
    expect(norm(formatValue(x, { value: 41, scale: 0, unit: 'pcs' }))).toBe('41')
    expect(norm(formatValue(x, { value: 9250, scale: 0, unit: 'bp' }))).toBe('92,5 %')
    expect(norm(formatValue(x, { value: 75, scale: 0, unit: 'min' }))).toBe('1 ч 15 мин')
    expect(norm(formatValue(x, { value: 1605, scale: 1, unit: 'A' }))).toBe('160,50 А')
    expect(norm(formatValue(x, { value: 7, scale: 0, unit: 'kg' }))).toBe('7 kg')
  })

  it('происхождение времени: передано / вычислено / смешанное; без пометки — предупреждение', () => {
    expect(originOf({ unit: 'min', origin: 'reported_by_source' })).toMatchObject({ code: 'reported_by_source', warn: false })
    expect(x.t(originOf({ unit: 'min', origin: 'computed_by_system' })!.key)).toBe('Вычислено системой')
    expect(originOf({ unit: 's', origin: 'mixed' })?.code).toBe('mixed')
    expect(originOf({ unit: 'min' })).toMatchObject({ code: 'missing', warn: true })
    expect(originOf({ unit: 'pcs' })).toBeNull()
  })

  it('сравнение с прошлым периодом — только при одинаковых единицах', () => {
    expect(deltaOf({ value: 41, scale: 0, unit: 'pcs' }, { value: 38, scale: 0, unit: 'pcs' })).toBe(3)
    expect(deltaOf({ value: 41, scale: 0, unit: 'pcs' }, { value: 38, scale: 0, unit: 'bp' })).toBeNull()
    expect(deltaOf({ value: 41, scale: 0, unit: 'pcs' }, undefined)).toBeNull()
  })
})

describe('раскладка раздела «Аналитика»', () => {
  const m = toOverviewModel(overview().items)
  const ids = (rows: { metric_id: string }[]) => rows.map((r) => r.metric_id)

  it('дефекты и изделия с дефектами — раздельно; повторные операции — отдельно', () => {
    expect(ids(m.defects)).toEqual(['confirmed_defects'])
    expect(ids(m.items)).toEqual(['items_with_confirmed_nc'])
    expect(ids(m.operations)).toEqual(['rework_runs'])
  })

  it('входной брак не попадает в производственные дефекты и причины; оборудование и исполнители — свои графы', () => {
    expect(ids(m.accounts.incoming)).toEqual(['incoming_defects'])
    expect(ids(m.accounts.equipment)).toEqual(['equipment_downtime'])
    expect(ids(m.accounts.people)).toEqual(['confirmed_performer_errors'])
    expect(m.accounts.hypotheses).toEqual([])
    expect(ids(m.causes)).toEqual(['cause_established'])
    expect(accountOf({ metric_id: 'hypotheses_open', group: 'causes', slices: [] })).toBe('hypotheses')
  })

  it('«оценка невозможна» — в проверках, отдельной строкой; ни одна строка не теряется', () => {
    expect(ids(m.inspection)).toEqual(['inspected_items', 'unable_to_assess', 'first_pass_yield'])
    const placed = [m.inspection, m.defects, m.items, m.operations, m.causes, m.time, m.other, ...Object.values(m.accounts)].flat()
    const inComparison = m.comparison.flatMap((c) => c.columns).filter((c) => c.group === 'comparison')
    expect(new Set([...ids(placed), ...ids(inComparison)])).toEqual(new Set(ids(overview().items)))
  })

  it('сравнение сопоставимых работ: срез × показатель, нет значения — null, а не ноль', () => {
    expect(m.comparison).toHaveLength(1)
    const tbl = m.comparison[0]!
    expect(tbl.dimension).toBe('performer')
    expect(ids(tbl.columns)).toEqual(['comparable_welds', 'confirmed_performer_errors'])
    expect(tbl.rows.map((r) => r.key)).toEqual(['W21', 'W22'])
    expect(tbl.rows[1]!.cells[0]!.value.value).toBe(18)
    expect(tbl.rows[1]!.cells[1]).toBeNull()
  })
})

describe('раскрытие до записей', () => {
  it('штучный показатель: сумма вкладов сверяется с итогом', () => {
    const dd = drilldown()
    expect(sumCheck(dd.total, dd.items, true)).toBe('match')
    expect(sumCheck({ ...dd.total, value: 3 }, dd.items, true)).toBe('mismatch')
    expect(sumCheck(dd.total, dd.items, false)).toBeNull()
    expect(sumCheck({ value: 9250, scale: 0, unit: 'bp' }, dd.items, true)).toBeNull()
  })
})

describe('контрольная карта', () => {
  it('границы и точки в координатах; точка вне границ выше верхней линии', () => {
    const g = chartGeometry(chart())
    expect(g.upper).toBeLessThan(g.center)
    expect(g.center).toBeLessThan(g.lower)
    expect(g.points).toHaveLength(3)
    expect(g.points[2]!.y).toBeLessThan(g.upper)
    expect(g.points[0]!.x).toBe(g.box.left)
    expect(g.points[2]!.x).toBe(g.box.width - g.box.right)
    expect(g.path.split(' ')).toHaveLength(3)
  })

  it('одна точка — по центру, без деления на ноль', () => {
    const c = chart()
    c.points = c.points.slice(0, 1)
    const g = chartGeometry(c)
    expect(Number.isFinite(g.points[0]!.x)).toBe(true)
  })
})
