// Центр управления руководителя: четыре состояния, «Требует моего решения» одной лентой, главный инцидент.
import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { describe, expect, it } from 'vitest'
import { i18n } from '@/shared/i18n'
import { decisionRows, stateTiles } from '../model/center'
import ControlCenterView from '../ui/ControlCenterView.vue'

const map = () =>
  ({
    counters: [
      { step_key: 'a', queue: 3, in_progress: 5, passed: 2, defects: 0, nonconformities: 1 },
      { step_key: 'b', queue: 1, in_progress: 2, passed: 0, defects: 1, nonconformities: 0 },
    ],
    anomalies: [{ step_key: 'b', kind: 'queue_above_norm' }, { step_key: 'a', kind: 'output_spike' }],
    bottleneck: { step_key: 'b', step_name: 'Сварка фланца', wait: '38 мин' },
    items: [], data_gaps: [], bpmn_xml: '',
  }) as never

const incident = { incident_id: 'RS-01', label: 'Инцидент ИС-2', status: 'open', size: 6, initial_size: 34, scope_version: 3, opened_at: '', stage: 'hypothesis', counts: { confirmed: 1, suspect: 4, unknown: 1, excluded: 28 }, next_step: 'Контрольный образец', nc_ids: [], close_blockers: [], common_factor: { factor: 'machine', value: 'IS-2', label: 'Сварочный источник ИС-2' } } as never

const attention = [{ entry_id: 'e1', kind: 'unverified_measures', n: 2 }, { entry_id: 'e2', kind: 'overdue_decision', target: 'Ф-003', overdue_minutes: 37, items: 18, operations: 2 }] as never
const alerts = [
  { alert_id: 'a1', kind: 'anomaly', at: '2026-09-23T08:21:00Z', anomaly: 'queue_above_norm', node_name: 'Сварка' },
  { alert_id: 'a2', kind: 'integrity_violation', at: '2026-09-23T08:30:00Z' },
  { alert_id: 'a3', kind: 'not_moved_to_isolator', at: '2026-09-23T08:40:00Z', item: 'Ф-002' },
] as never

describe('центр управления', () => {
  it('лента решений: критичное и опасное первыми; отклонения узлов — не решение руководителя', () => {
    const rows = decisionRows(attention, alerts)
    expect(rows.map((r) => r.kind)).toEqual(['integrity_violation', 'not_moved_to_isolator', 'overdue_decision', 'unverified_measures'])
  })

  it('четыре состояния: в работе и очередь, область риска, ограничение линии, решения и просрочки', () => {
    const tiles = stateTiles(map(), incident, decisionRows(attention, alerts))
    expect(tiles.map((x) => x.id)).toEqual(['production', 'quality', 'flow', 'decisions'])
    expect(tiles[0]!.data).toMatchObject({ inProgress: 7, queue: 4, delayed: 1 })
    expect(tiles[1]).toMatchObject({ tone: 'danger', data: { scope: 6, confirmed: 1 } })
    expect(tiles[2]!.data).toMatchObject({ bottleneck: 'Сварка фланца', wait: '38 мин' })
    expect(tiles[3]).toMatchObject({ tone: 'danger', data: { n: 4, overdue: 1 } })
  })

  it('экран: состояния словами, лента решений, главный инцидент с путём и локализацией, «открыть на карте»', async () => {
    const decisions = decisionRows(attention, alerts)
    const w = mount(ControlCenterView, {
      props: { tiles: stateTiles(map(), incident, decisions), decisions, incident, scopePath: [34, 13, 6], contained: { inScope: 2, isolated: 1 } },
      global: { plugins: [createPinia(), i18n] },
    })
    expect(w.find('[data-tile="quality"]').text()).toContain('6 изделий в области риска')
    expect(w.find('[data-tile="flow"]').text()).toContain('Ограничение: Сварка фланца')
    expect(w.find('[data-testid="decisions"]').text()).toContain('Требует моего решения · 4')
    expect(w.find('[data-testid="decisions"]').text()).toContain('Ф-002')
    expect(w.find('[data-testid="incident-scope"]').text()).toBe('34 → 13 → 6')
    expect(w.find('[data-testid="incident-contained"]').text()).toBe('1 / 2')
    expect(w.find('[data-testid="main-incident"]').text()).toContain('Проверка гипотез')
    await w.find('[data-testid="open-map"]').trigger('click')
    expect(w.emitted('open-map')?.[0]).toEqual(['RS-01'])
  })

  it('решений нет — одна тихая фраза; инцидента нет — карточки нет', () => {
    const w = mount(ControlCenterView, { props: { tiles: stateTiles(null, null, []), decisions: [] }, global: { plugins: [createPinia(), i18n] } })
    expect(w.text()).toContain('Сейчас ничего не требует вашего решения')
    expect(w.find('[data-testid="main-incident"]').exists()).toBe(false)
  })
})
