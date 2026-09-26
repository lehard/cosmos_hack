// Шина событий безопасности (FR-106): тип по-русски, важность, объект,
// источник, связанное критическое действие; фильтр по типу; тревога — признак.
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mockApi, mountWidget } from '@/entities/run/__tests__/api'
import { securityEvents } from '@/entities/integrity/__tests__/fixtures'
import SecurityEventsWidget from '../ui/SecurityEventsWidget.vue'

afterEach(() => vi.unstubAllGlobals())
const props = { widgetId: 'security-events', titleKey: 'audit.securityEvents.title', density: 'compact' }

describe('события безопасности', () => {
  it('лента: тип и важность по-русски, объект, CA; тревога — не «норма»', async () => {
    mockApi({ 'GET /api/v1/security-events': { items: securityEvents() } })
    const w = await mountWidget(SecurityEventsWidget, props)
    const alarm = w.find('li.row[data-seq="220001"]')
    expect(alarm.text()).toContain('Нарушение целостности')
    expect(alarm.text()).toContain('Тревога')
    expect(alarm.text()).toContain('Подмена записи EV-WS2-0412')
    expect(alarm.text()).toContain('Запись CA-17')
    expect(w.find('li.row[data-seq="90001"]').text()).toContain('Ошибка аутентификации')
    expect(w.find('li.row[data-seq="90001"]').text()).toContain('Источник: web')
    expect(w.find('.widget-frame').attributes('data-state')).toBe('defect_indication')
  })

  it('тип из среза стола уходит в запрос', async () => {
    const calls = mockApi({ 'GET /api/v1/security-events': { items: [] } })
    await mountWidget(SecurityEventsWidget, { ...props, slice: { event_type: 'security.signature.invalid' } })
    expect(calls[0]?.query.get('event_type')).toBe('security.signature.invalid')
  })
})

describe('события безопасности — фильтр', () => {
  it('выбор типа уходит в запрос', async () => {
    const calls = mockApi({ 'GET /api/v1/security-events': { items: securityEvents() } })
    const w = await mountWidget(SecurityEventsWidget, props)
    await w.find('[data-testid="event-type"]').setValue('security.auth.failed')
    await new Promise((r) => setTimeout(r, 0))
    await vi.waitFor(() => expect(calls.at(-1)?.query.get('event_type')).toBe('security.auth.failed'))
    expect(w.find('[data-testid="event-type"] option[value="security.auth.failed"]').text()).toBe('Ошибка аутентификации')
  })
})
