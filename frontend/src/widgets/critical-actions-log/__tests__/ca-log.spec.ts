// Журнал критических действий (FR-79, FR-106; AD-28): CA-‹n›, «было → стало»,
// кто, полномочие, клеймо, отмены новой записью; фильтр по группе; подробности.
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mockApi, mountWidget, settle } from '@/entities/run/__tests__/api'
import { criticalAction } from '@/entities/integrity/__tests__/fixtures'
import CriticalActionsLogWidget from '../ui/CriticalActionsLogWidget.vue'

afterEach(() => vi.unstubAllGlobals())
const props = { widgetId: 'critical-actions-log', titleKey: 'desks.criticalActionsLog', density: 'compact' }
const list = [criticalAction(), criticalAction({ ca_ref: 'CA-13', ca_no: 13, cancels: 'CA-9', ca_group: 'risk_scope', actor_id: undefined }), criticalAction({ ca_ref: 'CA-9', ca_no: 9, cancelled_by: 'CA-13', cancel_reason: 'ошибка области' })]

describe('журнал критических действий', () => {
  it('записи: номер, группа, действие, объект, было → стало, кто, полномочие и клеймо; отмены', async () => {
    mockApi({ 'GET /api/v1/critical-actions': { items: list } })
    const w = await mountWidget(CriticalActionsLogWidget, props)
    const ca12 = w.find('tr[data-ca="CA-12"]')
    expect(ca12.text()).toContain('Решение по несоответствию')
    expect(ca12.text()).toContain('nonconformity:NC-01')
    expect(ca12.text()).toContain('сигнал → подтверждено')
    expect(ca12.text()).toContain('qc_acceptance')
    expect(ca12.text()).toContain('ST-7')
    expect(w.find('tr[data-ca="CA-13"]').text()).toContain('Отменяет CA-9')
    expect(w.find('tr[data-ca="CA-9"]').attributes('data-cancelled')).toBe('true')
    expect(w.find('tr[data-ca="CA-9"]').text()).toContain('Отменено записью CA-13: ошибка области')
  })

  it('выбор записи — основания и связь с основным журналом', async () => {
    const calls = mockApi({ 'GET /api/v1/critical-actions': { items: list }, 'GET /api/v1/critical-actions/CA-12': criticalAction() })
    const w = await mountWidget(CriticalActionsLogWidget, props)
    await w.find('tr[data-ca="CA-12"]').trigger('click')
    await settle()
    expect(calls.some((c) => c.path === '/api/v1/critical-actions/CA-12')).toBe(true)
    const card = w.find('[data-testid="detail"]')
    expect(card.text()).toContain('ev-sig-1')
    expect(card.text()).toContain('ev-dec-1')
    expect(card.text()).toContain('sha256:c0ffee')
  })

  it('группа из среза стола уходит в запрос', async () => {
    const calls = mockApi({ 'GET /api/v1/critical-actions': { items: [] } })
    const w = await mountWidget(CriticalActionsLogWidget, { ...props, slice: { group: 'authority' } })
    expect(calls[0]?.query.get('group')).toBe('authority')
    expect(w.text()).toContain('Записей нет')
  })
})

describe('журнал критических действий — фильтр', () => {
  it('выбор группы уходит в запрос; «все группы» — без фильтра', async () => {
    const calls = mockApi({ 'GET /api/v1/critical-actions': { items: list } })
    const w = await mountWidget(CriticalActionsLogWidget, props)
    await w.find('[data-testid="group"]').setValue('risk_scope')
    await vi.waitFor(() => expect(calls.at(-1)?.query.get('group')).toBe('risk_scope'))
    await w.find('[data-testid="group"]').setValue('')
    await vi.waitFor(() => expect(calls.at(-1)?.query.has('group')).toBe(false))
  })
})
