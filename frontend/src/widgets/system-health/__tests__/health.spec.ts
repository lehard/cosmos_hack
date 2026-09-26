// Состояние компонентов (FR-127, AD-45, AD-46): сервисы, очереди, интеграции,
// верификатор, метрики приёма, остановленные изделия и «повторить обработку».
import { afterEach, describe, expect, it, vi } from 'vitest'
import { adminSession, mockApi, mountWidget, problem, settle } from '@/entities/run/__tests__/api'
import { health, stopped } from '@/entities/integrity/__tests__/fixtures'
import { metrics } from '@/entities/quarantine/__tests__/fixtures'
import SystemHealthWidget from '../ui/SystemHealthWidget.vue'

afterEach(() => vi.unstubAllGlobals())
const props = { widgetId: 'system-health', titleKey: 'desks.components' }
const routes = (over: Record<string, unknown> = {}) => ({
  'GET /api/v1/ops/health': health(),
  'GET /api/v1/ingest/metrics': metrics(),
  'GET /api/v1/ops/stopped-items': { items: stopped() },
  'GET /api/v1/auth/session': adminSession,
  'POST /api/v1/ops/stopped-items/ENT01:F-031/retry': { command_id: 'x', seq: 1, event_ids: ['e'], replayed: false },
  ...over,
})

describe('состояние компонентов', () => {
  it('сервисы: «работает», «с перебоями», «ещё не подключён» — не выдаются одно за другое', async () => {
    mockApi(routes())
    const w = await mountWidget(SystemHealthWidget, props)
    expect(w.find('tr[data-component="ant/api"]').text()).toContain('Работает')
    expect(w.find('tr[data-component="ant/api"]').text()).toContain('api-1')
    expect(w.find('tr[data-component="projector"]').text()).toContain('С перебоями')
    expect(w.find('tr[data-component="projector"]').text()).toContain('отставание 120 записей')
    expect(w.find('tr[data-component="keeper"]').text()).toContain('Ещё не подключён')
    expect(w.find('tr[data-queue="p-3"]').text()).toContain('120')
    expect(w.find('[data-integration="onec"]').text()).toContain('1С')
    expect(w.find('[data-testid="verifier"]').text()).toContain('Цело')
    expect(w.find('.widget-frame').attributes('data-state')).toBe('normal')
  })

  it('метрики приёма: полнота в процентах, повторы, карантин', async () => {
    mockApi(routes())
    const w = await mountWidget(SystemHealthWidget, props)
    const m = w.find('[data-testid="metrics"]')
    expect(m.find('[data-testid="completeness"]').text().replace(/\s/g, ' ')).toBe('97,2 %')
    expect(m.text()).toContain('40 / 180 мс')
    expect(m.find('[data-testid="quarantine-open"]').text()).toBe('3')
  })

  it('нарушение целостности у верификатора — не «норма»', async () => {
    mockApi(routes({ 'GET /api/v1/ops/health': health({ verifier: { verdict: 'violated', checked_at: '2026-09-23T13:40:00Z', report_ref: 'r' } }) }))
    const w = await mountWidget(SystemHealthWidget, props)
    expect(w.find('.widget-frame').attributes('data-state')).toBe('defect_indication')
    expect(w.find('[data-testid="verifier"]').text()).toContain('Нарушено')
  })

  it('метрики не читаются — сказано об этом, остальное на месте', async () => {
    mockApi(routes({ 'GET /api/v1/ingest/metrics': problem(501, 'api.not_implemented') }))
    const w = await mountWidget(SystemHealthWidget, props)
    expect(w.find('[data-testid="metrics"]').text()).toContain('Не удалось загрузить: Метрики приёма')
    expect(w.find('tr[data-component="ant/api"]').exists()).toBe(true)
  })

  it('«повторить обработку» — ops.processing.retry с событием сбоя и seq, на котором видели', async () => {
    const calls = mockApi(routes())
    const w = await mountWidget(SystemHealthWidget, props)
    await w.find('tr[data-item="ENT01:F-031"] button').trigger('click')
    await settle()
    const post = calls.find((c) => c.method === 'POST')
    expect(post?.path).toBe('/api/v1/ops/stopped-items/ENT01:F-031/retry')
    expect(post?.body).toMatchObject({ failure_event_id: 'ev-fail-1', basis_seq: 120001, policy_seq: 7, workplace_id: 'WP-ADM' })
    expect(w.find('tr[data-item="ENT01:F-031"]').text()).toContain('Обработка перезапущена')
  })

  it('здоровье не читается — «ошибка входа»', async () => {
    mockApi({})
    const w = await mountWidget(SystemHealthWidget, props)
    expect(w.find('.widget-frame').attributes('data-state')).toBe('input_error')
  })
})
