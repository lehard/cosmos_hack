// Уведомления в шапке (FR-57): число непрочитанных; по нажатию — виды
// уведомлений и открытые задачи с отметкой.
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mockApi, mountWidget, settle } from '@/entities/run/__tests__/api'
import { at, foremanSession, tasks } from '@/entities/workplace/__tests__/fixtures'
import NotificationsBell from '../header/NotificationsBell.vue'

afterEach(() => {
  vi.unstubAllGlobals()
  document.body.innerHTML = ''
})

describe('колокольчик уведомлений', () => {
  it('непрочитанное по видам и открытые задачи', async () => {
    const calls = mockApi({
      'GET /api/v1/auth/session': foremanSession(),
      'GET /api/v1/notifications/summary': { unread: 6, by_kind: { info: 1, alarm: 2, task: 2, decision_request: 1 } },
      'GET /api/v1/journal/head': { seq: 9100, ca_seq: 0, clock_mode: 'system', recorded_at: at('09:00') },
      'GET /api/v1/tasks': { items: tasks().filter((t) => t.state === 'open') },
    })
    const w = await mountWidget(NotificationsBell, {}, undefined)
    expect(w.text()).toContain('6')
    expect(calls.some((c) => c.path === '/api/v1/tasks')).toBe(false)
    await w.find('[data-testid="notifications-bell"]').trigger('click')
    await settle()
    const panel = document.body.querySelector('[data-testid="notifications-panel"]')
    expect(panel?.querySelector('[data-kind="alarm"]')?.textContent?.trim()).toBe('Тревога: 2')
    expect(panel?.querySelector('[data-kind="decision_request"]')?.textContent?.trim()).toBe('Запрос решения: 1')
    expect(panel?.textContent).toContain('Перенести Ф-017 в изолятор и подтвердить')
    const req = calls.find((c) => c.path === '/api/v1/tasks')
    expect(req?.query.get('state')).toBe('open')
  })
})
