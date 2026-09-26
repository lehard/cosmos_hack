// Карта дефицита данных (эпик 42, FR-143): для каждого вида сведений — число
// расследований и оценка сужения; нет оценки — «оценка невозможна», не ноль;
// узлы без данных источника — состояние «оценка невозможна».
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mockApi, mountWidget, settle } from '@/entities/incident/__tests__/api-mock'
import DataDeficitMapWidget from '../ui/DataDeficitMapWidget.vue'

const nodes = (data_gaps: string[]) => ({ period: { kind: 'shift', from: '2026-09-23T05:00:00.000Z', to: '2026-09-23T09:30:00.000Z' }, process_version_id: 'PV-1', counters: [], anomalies: [], data_gaps, basis_seq: 1 })
const deficit = {
  investigations: 47,
  basis_seq: 9,
  rows: [
    {
      kind: 'tool_unknown',
      investigations: 31,
      places: [{ place: 'M-17', count: 20 }, { place: 'M-18', count: 11 }],
      nc_ids: ['NC-1', 'NC-2'],
      incident_ids: ['RS-01'],
      digitization: 'Учёт инструмента на станках (ввод при смене инструмента, метка на оправке)',
      estimate: { from_tenths: 130, to_tenths: 40, incidents: 2, text: 'в среднем с 13 до 4 деталей' },
    },
    { kind: 'no_observation_after_operation', investigations: 19, places: [{ place: 'welding.weld', count: 19 }], nc_ids: ['NC-3'], incident_ids: [], digitization: 'Контроль сразу после операции (камера или замер на посту)' },
  ],
}
const $ = (sel: string) => document.querySelector<HTMLElement>(sel)

describe('карта дефицита данных', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    document.body.innerHTML = ''
  })
  const props = { widgetId: 'data-deficit-map', titleKey: 'desks.dataDeficit' }

  it('виды сведений: «31 из 47», где, что внедрить и оценка сужения; без оценки — так и сказано', async () => {
    mockApi({ 'GET /api/v1/analysis/data-deficit': deficit, 'GET /api/v1/metrics/node-counters': nodes([]) })
    const w = await mountWidget(DataDeficitMapWidget, props)
    const tool = w.find('tr[data-missing="tool_unknown"]')
    expect(tool.text()).toContain('Неизвестен инструмент')
    expect(tool.text()).toContain('31 из 47')
    expect(tool.text()).toContain('M-17 (20)')
    expect(tool.text()).toContain('в среднем с 13 до 4 деталей')
    const after = w.find('tr[data-missing="no_observation_after_operation"]')
    expect(after.text()).toContain('Оценка невозможна')
    expect(w.find('[data-testid="no-gaps"]').exists()).toBe(true)
  })

  it('узлы без данных источника → «оценка невозможна»; строка открывает правое окно', async () => {
    mockApi({ 'GET /api/v1/analysis/data-deficit': deficit, 'GET /api/v1/metrics/node-counters': nodes(['welding.weld']) })
    const w = await mountWidget(DataDeficitMapWidget, props)
    expect(w.attributes('data-state')).toBe('unable_to_assess')
    await w.find('tr[data-missing="tool_unknown"]').trigger('click')
    await settle()
    expect($('[data-testid="deficit-drawer"]')?.textContent).toContain('NC-1, NC-2')
    expect($('[data-testid="record-drawer-actions"] [data-testid="generate"]')).not.toBeNull()
    w.unmount()
  })
})
