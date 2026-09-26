// План прогона на пульте (Д-85): «Сейчас ждём: ‹роль› — ‹действие› (‹изделие›)»,
// запланированный сбой виден заранее, ближайшие события с доменным временем,
// остановки выделены; без операции плана пульт работает как раньше.
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defaultConfig, planWait, upcoming, type RunPlan } from '@/entities/run'
import { adminSession, mockApi, mountWidget } from '@/entities/run/__tests__/api'
import { injections, run, scenario, scenarios } from '@/entities/run/__tests__/fixtures'
import { i18n } from '@/shared/i18n'
import ScenarioConsoleView from '../ui/ScenarioConsoleView.vue'
import ScenarioConsoleWidget from '../ui/ScenarioConsoleWidget.vue'

let pinia: ReturnType<typeof createPinia>
beforeEach(() => {
  pinia = createPinia()
  setActivePinia(pinia)
})
afterEach(() => vi.unstubAllGlobals())

const norm = (x: string) => x.replace(/\s+/g, ' ').trim()

const plan = (over: Partial<RunPlan> = {}): RunPlan => ({
  run_id: 'fx-1a2b3c4d',
  state: 'waiting_for_decision',
  clock_at: '2026-09-21T05:00:00Z',
  speed: 60,
  waiting_for: { role: 'site_foreman', action: 'process.movement.receive', object_id: 'Ф-001', title: 'принять в цех Ф-001' },
  items: [
    { at: '2026-09-21T05:00:00Z', kind: 'event', title: 'Из мехцеха пришла партия Ф-001…Ф-003', stop: false, done: true, waiting: false },
    { at: '2026-09-21T05:00:00Z', kind: 'decision', title: 'Принять в цех', stop: true, role: 'site_foreman', persona: 'FOR-WC', item_id: 'Ф-001', done: false, waiting: true },
    { at: '2026-09-21T05:05:00Z', kind: 'decision', title: 'Начать сварку', stop: true, role: 'performer', persona: 'W21', item_id: 'Ф-001', done: false, waiting: false },
    { at: '2026-09-21T05:05:00Z', until: '2026-09-21T05:45:00Z', kind: 'event', title: 'Журнал тока ИС-2: 158–163 А', stop: false, done: false, waiting: false },
    { at: '2026-09-21T07:10:00Z', kind: 'alert', title: 'На 2-й сварке ИС-2 уйдёт из уставки: 176–182 А', stop: false, item_id: 'Ф-002', done: false, waiting: false },
  ],
  ...over,
})

const mountView = (props = {}) =>
  mount(ScenarioConsoleView, {
    props: { scenarios: scenarios(), selectedScenario: 'flange-bad-day', config: defaultConfig(scenario()), run: run({ state: 'waiting_for_decision' }), injections: injections(), plan: plan(), ...props },
    global: { plugins: [pinia, i18n] },
  })

describe('план прогона — правила', () => {
  it('«сейчас ждём» — из строки плана с изделием и персоной', () => {
    expect(planWait(plan())).toEqual({ role: 'site_foreman', what: 'Принять в цех', object: 'Ф-001', persona: 'FOR-WC', at: '2026-09-21T05:00:00Z' })
  })
  it('нет строки ожидания — ожидание прогона; ничего не ждём — null', () => {
    expect(planWait(plan({ items: [] }))).toMatchObject({ role: 'site_foreman', what: 'принять в цех Ф-001', object: 'Ф-001' })
    expect(planWait(plan({ items: [], waiting_for: undefined }))).toBeNull()
  })
  it('ближайшие — только не случившиеся, в порядке сервера', () => {
    expect(upcoming(plan(), 2).map((e) => e.title)).toEqual(['Принять в цех', 'Начать сварку'])
  })
})

describe('план прогона на пульте', () => {
  it('сейчас ждём: роль — действие (изделие), персона', () => {
    const w = mountView()
    expect(norm(w.find('[data-testid="plan-wait"]').text())).toBe('Сейчас ждём: Мастер участка — Принять в цех (Ф-001) · FOR-WC')
    // Общий «ждёт решения» не дублируется — его заменяет план.
    expect(w.find('[data-testid="waiting"]').exists()).toBe(false)
  })

  it('запланированный сбой виден заранее; остановки и сбой выделены; время доменное', () => {
    const w = mountView()
    const alert = w.find('[data-testid="plan-alert"]')
    expect(alert.findAll('strong, span').map((x) => x.text())).toEqual(['Запланированный сбой:', 'На 2-й сварке ИС-2 уйдёт из уставки: 176–182 А', 'по плану 21.09 10:10'])
    const rows = w.findAll('[data-testid="plan-entries"] li')
    expect(rows).toHaveLength(4)
    expect(rows[0]!.attributes('data-waiting')).toBe('true')
    expect(rows[0]!.findAll('.at, .title, .meta, .tag').map((x) => x.text())).toEqual(['21.09 08:00', 'Принять в цех', 'Мастер участка · FOR-WC · Ф-001', 'ждём сейчас'])
    expect(rows[1]!.attributes('data-stop')).toBe('true')
    expect(rows[1]!.text()).toContain('остановка')
    expect(rows[2]!.find('.at').text()).toBe('21.09 08:05–08:45')
    expect(rows[2]!.attributes('data-stop')).toBeUndefined()
    expect(rows[3]!.attributes('data-kind')).toBe('alert')
    expect(rows[3]!.text()).toContain('сбой')
  })

  it('длинный план — первые строки и «Показать ещё»', async () => {
    const w = mountView({ plan: plan() })
    await w.setProps({ plan: plan({ items: Array.from({ length: 12 }, (_, i) => ({ at: '2026-09-21T05:00:00Z', kind: 'event' as const, title: `Событие ${i}`, stop: false, done: false, waiting: false })) }) })
    expect(w.findAll('[data-testid="plan-entries"] li')).toHaveLength(8)
    await w.find('[data-testid="plan-more"]').trigger('click')
    expect(w.findAll('[data-testid="plan-entries"] li')).toHaveLength(12)
  })

  it('контейнер: читает simulation.run.plan текущего прогона', async () => {
    const calls = mockApi({
      'GET /api/v1/scenarios': { items: scenarios() },
      'GET /api/v1/runs': { items: [run({ state: 'waiting_for_decision' })] },
      'GET /api/v1/runs/fx-1a2b3c4d': run({ state: 'waiting_for_decision' }),
      'GET /api/v1/runs/fx-1a2b3c4d/plan': plan(),
      'GET /api/v1/runs/fx-1a2b3c4d/injections': { items: injections() },
      'GET /api/v1/auth/session': adminSession,
    })
    const w = await mountWidget(ScenarioConsoleWidget, { widgetId: 'scenario-console', titleKey: 'desks.testScenarios' }, pinia)
    expect(calls.some((c) => c.path === '/api/v1/runs/fx-1a2b3c4d/plan' && c.query.get('limit') === '50')).toBe(true)
    expect(w.find('[data-testid="plan-wait"]').exists()).toBe(true)
    w.unmount()
  })

  it('контейнер: операции плана нет (501) — пульт показывает «ждёт решения» как раньше', async () => {
    mockApi({
      'GET /api/v1/scenarios': { items: scenarios() },
      'GET /api/v1/runs': { items: [run({ state: 'waiting_for_decision', waiting_for: { role: 'site_foreman', action: 'process.movement.receive', object_id: 'Ф-001' } })] },
      'GET /api/v1/runs/fx-1a2b3c4d': run({ state: 'waiting_for_decision', waiting_for: { role: 'site_foreman', action: 'process.movement.receive', object_id: 'Ф-001' } }),
      'GET /api/v1/auth/session': adminSession,
    })
    const w = await mountWidget(ScenarioConsoleWidget, { widgetId: 'scenario-console', titleKey: 'desks.testScenarios' }, pinia)
    expect(w.find('[data-testid="run-plan"]').exists()).toBe(false)
    expect(w.find('[data-testid="waiting"]').exists()).toBe(true)
    w.unmount()
  })
})
