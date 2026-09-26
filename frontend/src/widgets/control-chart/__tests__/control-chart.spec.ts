// Контрольная карта узла (FR-5, FR-89): границы от сервера, выход за границы —
// формой, цветом и строкой; точка ведёт к изделию; узел — из среза стола.
import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { i18n } from '@/shared/i18n'
import { mockApi, mountWidget } from '@/entities/incident/__tests__/api-mock'
import { chart } from '@/entities/metric/__tests__/fixtures'
import ControlChartView from '../ui/ControlChartView.vue'
import ControlChartWidget from '../ui/ControlChartWidget.vue'

const norm = (s: string) => s.replace(/[\u00a0\u202f]/g, ' ')
const nodes = (extra: Record<string, unknown> = {}) => ({
  period: { kind: 'shift', from: '2026-09-23T05:00:00.000Z', to: '2026-09-23T09:30:00.000Z' },
  process_version_id: 'PV-1',
  counters: [
    { step_key: 'incoming.zt1_lot_acceptance', queue: 0, in_progress: 0, passed: 10, defects: 0 },
    { step_key: 'welding.weld', queue: 2, in_progress: 1, passed: 30, defects: 3 },
  ],
  anomalies: [],
  data_gaps: [],
  basis_seq: 10,
  ...extra,
})

describe('контрольная карта: представление', () => {
  const mountView = () => mount(ControlChartView, { props: { chart: chart() }, global: { plugins: [createPinia(), i18n] } })

  it('границы, точки, выход за границы — не только цветом', () => {
    const w = mountView()
    expect(norm(w.find('[data-limit="upper"]').text())).toContain('170 А')
    expect(w.findAll('[data-testid="point"]')).toHaveLength(3)
    const out = w.findAll('[data-testid="point"][data-out]')
    expect(out).toHaveLength(1)
    expect(out[0]!.find('rect').exists()).toBe(true)
    expect(out[0]!.find('title').text()).toContain('Вне контрольных границ')
    expect(w.find('[data-testid="outside"]').text()).toContain('Вне контрольных границ: 1')
    expect(w.text()).toContain('Ток сварки')
  })

  it('точка ведёт к изделию', async () => {
    const w = mountView()
    await w.findAll('[data-testid="point"]')[2]!.trigger('click')
    expect(w.emitted('open')?.[0]).toEqual([{ entity: 'item', id: 'ANT:FL-0042' }])
  })
})

describe('контрольная карта: виджет', () => {
  afterEach(() => vi.unstubAllGlobals())
  const props = { widgetId: 'control-chart', titleKey: 'widgets.controlChart' }

  it('узел из среза стола; точка вне границ — «признак дефекта», не брак', async () => {
    const calls = mockApi({ 'GET /api/v1/metrics/node-counters': nodes(), 'GET /api/v1/analytics/control-charts/welding.weld': chart() })
    const w = await mountWidget(ControlChartWidget, { ...props, slice: { step_key: 'welding.weld', metric_id: 'current_a' } })
    const req = calls.find((c) => c.path.startsWith('/api/v1/analytics/control-charts/'))
    expect(req?.path).toBe('/api/v1/analytics/control-charts/welding.weld')
    expect(w.attributes('data-state')).toBe('defect_indication')
  })

  it('без среза — узел с аномалией «доля дефектов вне границ»', async () => {
    const calls = mockApi({
      'GET /api/v1/metrics/node-counters': nodes({ anomalies: [{ step_key: 'welding.weld', kind: 'defect_rate_out_of_control' }] }),
      'GET /api/v1/analytics/control-charts/welding.weld': { ...chart(), points: chart().points.slice(0, 2) },
    })
    const w = await mountWidget(ControlChartWidget, props)
    expect(calls.some((c) => c.path === '/api/v1/analytics/control-charts/welding.weld')).toBe(true)
    expect(w.attributes('data-state')).toBe('normal')
    expect(w.find('[data-testid="inside"]').exists()).toBe(true)
  })
})
