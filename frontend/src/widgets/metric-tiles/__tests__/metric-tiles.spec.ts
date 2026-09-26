// Плитки стола руководителя (FR-8, FR-86, FR-89; Д-11): название от сервера,
// единицы, происхождение времени, «оценка невозможна» — не ноль; нажатие —
// раскрытие числа (FR-7).
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mockApi, mountWidget } from '@/entities/incident/__tests__/api-mock'
import { tiles } from '@/entities/metric/__tests__/fixtures'
import { useMetricFocusStore } from '@/entities/metric'
import MetricTilesWidget from '../ui/MetricTilesWidget.vue'

const norm = (s: string) => s.replace(/[\u00a0\u202f]/g, ' ')
const props = { widgetId: 'metric-tiles', titleKey: 'widgets.metricTiles' }

describe('плитки показателей', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('название по Д-11, значения в своих единицах, прошлый период, происхождение времени', async () => {
    const calls = mockApi({ 'GET /api/v1/metrics/tiles': tiles() })
    const w = await mountWidget(MetricTilesWidget, props)
    expect(calls[0]!.path).toBe('/api/v1/metrics/tiles')
    expect(w.attributes('data-mode')).toBe('fixtures')
    expect(w.find('[data-metric="first_pass_yield"] [data-testid="tile-title"]').text()).toBe('Прохождение контроля с первого раза (ЗТ-3)')
    expect(norm(w.find('[data-metric="first_pass_yield"] [data-testid="value"]').text())).toBe('92,5 %')
    expect(norm(w.find('[data-metric="first_pass_yield"] [data-testid="previous"]').text())).toBe('Прошлый период: 95 % (−2,50 п. п.)')
    expect(w.find('[data-metric="inspected_items"] [data-testid="previous"]').text()).toBe('Прошлый период: 38 (+3)')
    expect(norm(w.find('[data-metric="lead_time"]').text())).toContain('1 ч 15 мин')
    expect(w.find('[data-metric="lead_time"] [data-origin="computed_by_system"]').text()).toBe('Вычислено системой')
  })

  it('«оценка невозможна» — словами и прочерком, не ноль', async () => {
    mockApi({ 'GET /api/v1/metrics/tiles': tiles() })
    const w = await mountWidget(MetricTilesWidget, props)
    const tile = w.find('[data-metric="cause_established"]')
    expect(tile.attributes('data-unknown')).toBeDefined()
    expect(tile.find('[data-testid="value"]').exists()).toBe(false)
    expect(tile.find('[data-testid="unknown"]').text()).toBe('Оценка невозможна — не ноль')
    expect(w.attributes('data-state')).toBe('normal')
  })

  it('все плитки неизвестны — состояние «оценка невозможна», а не «норма»', async () => {
    const t = tiles()
    t.items = t.items.map((x) => ({ ...x, unknown: true }))
    mockApi({ 'GET /api/v1/metrics/tiles': t })
    const w = await mountWidget(MetricTilesWidget, props)
    expect(w.attributes('data-state')).toBe('unable_to_assess')
  })

  it('нажатие на плитку выбирает число для раскрытия; смена периода — новый запрос', async () => {
    const calls = mockApi({ 'GET /api/v1/metrics/tiles': tiles() })
    const w = await mountWidget(MetricTilesWidget, props)
    await w.find('[data-metric="items_with_confirmed_nc"] button').trigger('click')
    const focus = useMetricFocusStore()
    expect(focus.pick).toEqual({ metricId: 'items_with_confirmed_nc', title: 'Изделия с подтверждёнными несоответствиями' })
    focus.period = 'week'
    await new Promise((r) => setTimeout(r, 0))
    await vi.waitFor(() => expect(calls.length).toBe(2))
  })

  it('ошибка сервера — «ошибка входа» с текстом по коду', async () => {
    mockApi({})
    const w = await mountWidget(MetricTilesWidget, props)
    expect(w.attributes('data-state')).toBe('input_error')
  })
})
