// Раскрытие числа до исходных записей (FR-7, AD-45): строки вклада изделий,
// сверка суммы с итогом, исходные записи журнала с видом записи и источником.
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mockApi, mountWidget, settle } from '@/entities/incident/__tests__/api-mock'
import { drilldown } from '@/entities/metric/__tests__/fixtures'
import { useMetricFocusStore } from '@/entities/metric'
import MetricDrilldownWidget from '../ui/MetricDrilldownWidget.vue'

const props = { widgetId: 'metric-drilldown', titleKey: 'widgets.metricDrilldown' }
const entry = (event_id: string, seq: number, kind: string, extra: Record<string, unknown> = {}) => ({
  event_id,
  seq,
  kind,
  event_type: kind === 'decision' ? 'decision.nonconformity.confirmed' : 'quality.signal.received',
  occurred_at: '2026-09-23T08:53:00.000Z',
  recorded_at: '2026-09-23T08:53:00.000Z',
  summary: kind === 'decision' ? 'Контролёр К-07 подтвердил несоответствие' : 'Сигнал визуального контроля',
  signatures: [],
  ...extra,
})

describe('раскрытие показателя', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('ничего не выбрано — подсказка, запросов нет', async () => {
    const calls = mockApi({})
    const w = await mountWidget(MetricDrilldownWidget, props)
    expect(calls).toHaveLength(0)
    expect(w.text()).toContain('Выберите число в плитках или в разделе «Аналитика»')
  })

  it('выбранное число → строки вклада, сверка с итогом, записи изделия', async () => {
    const calls = mockApi({
      'GET /api/v1/analytics/metrics/items_with_confirmed_nc/contributions': drilldown(),
      'GET /api/v1/items/ANT:FL-0042/passport': {
        item_id: 'ANT:FL-0042',
        entries: [entry('ev-1', 101, 'fact', { source_kind: 'camera' }), entry('ev-9', 102, 'fact'), entry('ev-2', 140, 'decision')],
      },
    })
    const w = await mountWidget(MetricDrilldownWidget, props)
    useMetricFocusStore().select({ metricId: 'items_with_confirmed_nc', title: 'Изделия с подтверждёнными несоответствиями', sliceKey: 'LINE-FL-1', sliceLabel: 'Линия ФЛ-100 № 1' })
    await settle()
    expect(calls[0]!.path).toBe('/api/v1/analytics/metrics/items_with_confirmed_nc/contributions')
    expect(w.find('[data-testid="drill-slice"]').text()).toBe('Линия ФЛ-100 № 1')
    expect(w.find('[data-testid="sum-check"]').attributes('data-check')).toBe('match')
    expect(w.findAll('li[data-item]').map((r) => r.attributes('data-item'))).toEqual(['ANT:FL-0042', 'ANT:FL-0043'])
    // Вид источника и неизвестный код — не подменяются похожим.
    const kinds = w.findAll('li[data-item="ANT:FL-0043"] [data-testid="source-kind"]').map((k) => k.text())
    expect(kinds[1]).toBe('UNKNOWN(robot)')
    expect(w.find('li[data-item="ANT:FL-0043"] [data-testid="toggle-records"]').attributes('disabled')).toBeDefined()

    await w.find('li[data-item="ANT:FL-0042"] [data-testid="toggle-records"]').trigger('click')
    await settle()
    const recs = w.findAll('li[data-item="ANT:FL-0042"] li[data-event]')
    expect(recs.map((r) => r.attributes('data-event'))).toEqual(['ev-1', 'ev-2'])
    expect(recs[0]!.text()).toContain('Сигнал визуального контроля')
    expect(recs[0]!.find('[data-source="camera"]').exists()).toBe(true)
    expect(recs[1]!.attributes('data-kind')).toBe('decision')
    expect(recs[1]!.text()).toContain('запись № 140')
  })

  it('строка вне изделия (оборудование): вывод системы — видом источника, записи — списком id', async () => {
    const dd = drilldown()
    dd.items = [{ item_id: 'IS-2', label: 'Сварочный источник IS-2', slice_key: 'equipment:IS-2', value: { value: 20, scale: 0, unit: 'min' },
      source_event_ids: ['ev-7', 'ev-8'], source_kinds: ['machine', 'system'], ref: { entity: 'equipment', id: 'IS-2' } }]
    const calls = mockApi({ 'GET /api/v1/analytics/metrics/equipment_downtime/contributions': dd })
    const w = await mountWidget(MetricDrilldownWidget, props)
    useMetricFocusStore().select({ metricId: 'equipment_downtime', title: 'Простой' })
    await settle()
    const kinds = w.findAll('li[data-item="IS-2"] [data-testid="source-kind"]').map((k) => k.text())
    expect(kinds).toEqual(['Станок', 'Вывод системы'])
    await w.find('li[data-item="IS-2"] [data-testid="toggle-records"]').trigger('click')
    await settle()
    expect(w.findAll('[data-testid="record-ids"] li').map((l) => l.text())).toEqual(['ev-7', 'ev-8'])
    expect(calls.some((c) => c.path.includes('/passport'))).toBe(false)
  })

  it('сумма вкладов не равна итогу — видно сразу', async () => {
    const dd = drilldown()
    dd.total = { ...dd.total, value: 3 }
    mockApi({ 'GET /api/v1/analytics/metrics/items_with_confirmed_nc/contributions': dd })
    const w = await mountWidget(MetricDrilldownWidget, props)
    useMetricFocusStore().select({ metricId: 'items_with_confirmed_nc', title: 'x' })
    await settle()
    expect(w.find('[data-testid="sum-check"]').attributes('data-check')).toBe('mismatch')
  })

  it('запись вклада, которой нет в паспорте, показана своим id', async () => {
    mockApi({
      'GET /api/v1/analytics/metrics/items_with_confirmed_nc/contributions': drilldown(),
      'GET /api/v1/items/ANT:FL-0042/passport': { item_id: 'ANT:FL-0042', entries: [entry('ev-1', 101, 'fact')] },
    })
    const w = await mountWidget(MetricDrilldownWidget, props)
    useMetricFocusStore().select({ metricId: 'items_with_confirmed_nc', title: 'x' })
    await settle()
    await w.find('li[data-item="ANT:FL-0042"] [data-testid="toggle-records"]').trigger('click')
    await settle()
    expect(w.find('[data-testid="missing-record"]').text()).toBe('Запись ev-2 не найдена в паспорте изделия')
  })

  it('раскрытие не отдано сервером — ошибка входа с текстом, а не пустота', async () => {
    mockApi({})
    const w = await mountWidget(MetricDrilldownWidget, props)
    useMetricFocusStore().select({ metricId: 'confirmed_defects', title: 'x' })
    await settle()
    expect(w.attributes('data-state')).toBe('input_error')
  })
})
