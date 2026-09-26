// Каркас «Карта дефицита данных» (FR-143): узлы без данных источника — «оценка
// невозможна»; сводка по расследованиям — «не передано», не ноль.
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mockApi, mountWidget } from '@/entities/incident/__tests__/api-mock'
import DataDeficitMapWidget from '../ui/DataDeficitMapWidget.vue'

const nodes = (data_gaps: string[]) => ({ period: { kind: 'shift', from: '2026-09-23T05:00:00.000Z', to: '2026-09-23T09:30:00.000Z' }, process_version_id: 'PV-1', counters: [], anomalies: [], data_gaps, basis_seq: 1 })

describe('каркас «Карта дефицита данных»', () => {
  afterEach(() => vi.unstubAllGlobals())
  const props = { widgetId: 'data-deficit-map', titleKey: 'desks.dataDeficit' }

  it('узлы без данных источника → состояние «оценка невозможна»', async () => {
    mockApi({ 'GET /api/v1/metrics/node-counters': nodes(['welding.weld']) })
    const w = await mountWidget(DataDeficitMapWidget, props)
    expect(w.attributes('data-state')).toBe('unable_to_assess')
    expect(w.find('[data-step="welding.weld"]').text()).toContain('Оценка невозможна')
  })

  it('виды недостающих сведений — «не передано», без выдуманных чисел', async () => {
    mockApi({ 'GET /api/v1/metrics/node-counters': nodes([]) })
    const w = await mountWidget(DataDeficitMapWidget, props)
    expect(w.find('[data-testid="no-gaps"]').exists()).toBe(true)
    expect(w.find('tr[data-missing="toolUnknown"]').text()).toContain('Неизвестен инструмент')
    expect(w.find('tr[data-missing="toolUnknown"]').text()).toContain('Не передано сервером')
    expect(w.find('[data-testid="pending"]').exists()).toBe(true)
  })
})
