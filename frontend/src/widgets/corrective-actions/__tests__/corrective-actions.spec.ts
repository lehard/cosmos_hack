// «Меры и качество» (эпик 42; FR-64, FR-138): сводка руководителя по качеству,
// меры с флагами, организационная память; правое окно меры — план проверки
// эффективности; «Эффективна» выключена до конца окна наблюдения.
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mockApi, mountWidget, settle } from '@/entities/incident/__tests__/api-mock'
import CorrectiveActionsWidget from '../ui/CorrectiveActionsWidget.vue'

const action = (over: Record<string, unknown> = {}) => ({
  action_id: 'ACT-1',
  incident_id: 'RS-02',
  action_type: 'corrective_action',
  direction: 'prevent_occurrence',
  owner: 'TEC-01',
  title: 'Проверка источника каждые 75 циклов',
  plan: { metric: 'доля швов с трещиной', baseline: '6 из 34', window_days: 7, success_criterion: '0 трещин за 7 дней', enhanced_control: '100 % визуальный контроль' },
  status: 'implemented',
  cycle: 2,
  assigned_at: '2026-09-20T08:00:00.000Z',
  implemented_at: '2026-09-23T08:00:00.000Z',
  evaluation_due_at: '2026-09-30T08:00:00.000Z',
  flags: ['ineffective', 'awaiting_window'],
  history: [{ type: 'assigned', at: '2026-09-20T08:00:00.000Z', cycle: 1, event_id: 'e1' }],
  basis_seq: 5,
  ...over,
})
const list = {
  items: [action()],
  summary: { open: 1, overdue: 0, ineffective: 1, hanging_temporary: 0, evaluation_due: 0, recurring: 1 },
  recurring: [{ defect_type: 'crack', step_key: 'welding.weld', count: 3, nc_ids: ['NC-1', 'NC-2', 'NC-3'], with_action: true }],
  memory: [{ action_id: 'ACT-0', incident_id: 'RS-01', title: 'Обучение сварщиков', action_type: 'corrective_action', direction: 'prevent_occurrence', outcome: 'not_helped', cycles: 2 }],
  as_of: '2026-09-24T08:00:00.000Z',
  basis_seq: 5,
}
const $ = (sel: string) => document.querySelector<HTMLElement>(sel)

describe('меры и качество', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    document.body.innerHTML = ''
  })
  const props = { widgetId: 'corrective-actions', titleKey: 'widgets.quality.title' }

  it('сводка, меры с флагами, повторяющиеся проблемы и организационная память', async () => {
    mockApi({ 'GET /api/v1/corrective-actions': list })
    const w = await mountWidget(CorrectiveActionsWidget, props)
    expect(w.attributes('data-state')).toBe('defect_indication')
    expect(w.find('[data-tile="ineffective"]').text()).toContain('1')
    expect(w.find('[data-tile="ineffective"]').attributes('data-warn')).toBe('true')
    const row = w.find('tr[data-action="ACT-1"]')
    expect(row.text()).toContain('Внедрена, идёт окно наблюдения')
    expect(row.find('[data-flag="ineffective"]').text()).toBe('Не помогла')
    expect(w.find('[data-testid="recurring"]').text()).toContain('crack на welding.weld: 3')
    expect(w.find('[data-testid="memory"]').text()).toContain('Обучение сварщиков — не помогло')
  })

  it('окно меры: план эффективности; «Эффективна» выключена до конца окна; «Не помогла» — команда оценки', async () => {
    const calls = mockApi({
      'GET /api/v1/corrective-actions': list,
      'POST /api/v1/incidents/RS-02/actions/ACT-1/evaluation': { command_id: 'c', seq: 6, event_ids: ['e'], recorded_at: '2026-09-24T08:00:00.000Z' },
    })
    const w = await mountWidget(CorrectiveActionsWidget, props)
    await w.find('tr[data-action="ACT-1"]').trigger('click')
    await settle()
    expect($('[data-testid="plan"]')?.textContent).toContain('0 трещин за 7 дней')
    expect($('[data-testid="record-drawer-actions"] [data-testid="effective"]')!.hasAttribute('disabled')).toBe(true)
    $('[data-testid="record-drawer-actions"] [data-testid="failed"]')!.click()
    await settle()
    $('[data-testid="submit"]')!.click()
    await settle()
    const post = calls.find((c) => c.method === 'POST')
    expect(post?.path).toBe('/api/v1/incidents/RS-02/actions/ACT-1/evaluation')
    expect(post?.body).toMatchObject({ result: 'failed' })
    w.unmount()
  })
})
