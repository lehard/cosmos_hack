// Страница «Предложения» (эпик 42, FR-63): список с основаниями, генераторы
// (подключён / вход не подключён), «Сформировать предложения», правое окно
// предложения с кнопками в нижней панели (Д-70); решение — с основанием.
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mockApi, mountWidget, settle } from '@/entities/incident/__tests__/api-mock'
import ProposalsWidget from '../ui/ProposalsWidget.vue'

const suggestion = (id: string, status = 'new') => ({
  suggestion_id: id,
  generator: 'rules.risk_scope',
  kind: 'risk_scope',
  title: 'Сузить область риска RS-02 (34 изд.)',
  statement: 'Область риска инцидента RS-02 (общий фактор IS-2): 34 изделий, сужений по основаниям не было.',
  estimate: 'Под подозрением 34 изд.',
  responsible_role: 'technologist',
  incident_id: 'RS-02',
  step_key: 'welding.weld',
  basis: ['0190a0b2-0000-7000-8000-000000000001'],
  status,
  recorded_at: '2026-09-23T08:45:00.000Z',
  event_id: '0190a0b2-0000-5000-8000-000000000009',
  basis_seq: 41,
  history: [{ type: 'recorded', actor: 'rules.risk_scope', at: '2026-09-23T08:45:00.000Z', event_id: '0190a0b2-0000-5000-8000-000000000009' }],
})
const list = (items: unknown[]) => ({
  items,
  generators: [
    { id: 'rules.bottleneck', kind: 'bottleneck', connected: false },
    { id: 'rules.risk_scope', kind: 'risk_scope', connected: true },
    { id: 'rules.data_deficit', kind: 'data_deficit', connected: true },
  ],
  basis_seq: 41,
})
const $ = (sel: string) => document.querySelector<HTMLElement>(sel)

describe('страница «Предложения»', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    document.body.innerHTML = ''
  })
  const props = { widgetId: 'proposals', titleKey: 'desks.proposals' }

  it('пусто — так и сказано; генераторы с подключением; принцип «ничего само»', async () => {
    mockApi({ 'GET /api/v1/suggestions': list([]) })
    const w = await mountWidget(ProposalsWidget, props)
    expect(w.find('[data-testid="principle"]').text()).toContain('Система ничего не меняет сама')
    expect(w.find('[data-testid="no-proposals"]').exists()).toBe(true)
    expect(w.findAll('[data-generator]')).toHaveLength(3)
    expect(w.find('[data-generator="rules.bottleneck"]').text()).toContain('Вход не подключён')
    expect(w.find('[data-generator="rules.risk_scope"]').text()).toContain('Подключён')
  })

  it('строка открывает правое окно; «Принять» — только с основанием; кнопки в нижней панели', async () => {
    const calls = mockApi({
      'GET /api/v1/suggestions': list([suggestion('SUG-1')]),
      'POST /api/v1/suggestions/SUG-1/resolve': { command_id: 'c', seq: 42, event_ids: ['e'], recorded_at: '2026-09-23T08:50:00.000Z' },
    })
    const w = await mountWidget(ProposalsWidget, props)
    const row = w.find('tr[data-suggestion="SUG-1"]')
    expect(row.text()).toContain('Сузить область риска RS-02')
    expect(row.text()).toContain('Новое')
    await row.trigger('click')
    await settle()
    expect($('[data-testid="suggestion-drawer"]')?.textContent).toContain('34 изделий')
    expect($('[data-testid="basis"]')?.textContent).toContain('0190a0b2')
    expect($('[data-testid="record-drawer-actions"] [data-testid="forward"]')).not.toBeNull()
    $('[data-testid="record-drawer-actions"] [data-testid="accept"]')!.click()
    await settle()
    expect($('[data-testid="submit"]')!.hasAttribute('disabled')).toBe(true)
    const reason = $('[data-testid="reason"] textarea') as HTMLTextAreaElement
    reason.value = 'Сужаем по журналам источников'
    reason.dispatchEvent(new Event('input'))
    await settle()
    $('[data-testid="submit"]')!.click()
    await settle()
    const post = calls.find((c) => c.method === 'POST')
    expect(post?.path).toBe('/api/v1/suggestions/SUG-1/resolve')
    expect(post?.body).toMatchObject({ resolution: 'accepted', basis_seq: 41, reason: { text: 'Сужаем по журналам источников' } })
    w.unmount()
  })

  it('«Сформировать предложения» — команда генерации и число новых', async () => {
    const calls = mockApi({
      'GET /api/v1/suggestions': list([]),
      'POST /api/v1/suggestions/generate': { command_id: 'c', seq: 50, event_ids: ['a', 'b'], recorded_at: '2026-09-23T08:50:00.000Z' },
    })
    const w = await mountWidget(ProposalsWidget, props)
    await w.find('[data-testid="generate"]').trigger('click')
    await settle()
    expect(calls.some((c) => c.method === 'POST' && c.path === '/api/v1/suggestions/generate')).toBe(true)
    expect(w.find('[data-testid="generated"]').text()).toContain('2')
  })
})
