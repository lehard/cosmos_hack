// Карантин сообщений (FR-41; кейс §5.1 S12): записи с кодами, содержимое,
// переобработка с основанием (принимается один раз, AD-7), сводка на чужом столе.
import { afterEach, describe, expect, it, vi } from 'vitest'
import { adminSession, mockApi, mountWidget, settle } from '@/entities/run/__tests__/api'
import { entries, entry } from '@/entities/quarantine/__tests__/fixtures'
import QuarantineWidget from '../ui/QuarantineWidget.vue'

afterEach(() => vi.unstubAllGlobals())
const props = { widgetId: 'quarantine', titleKey: 'desks.quarantine' }
const routes = () => ({
  'GET /api/v1/quarantine': { items: entries() },
  'GET /api/v1/quarantine/Q-001': entry({ content: '{"event_type":"operation.operation.finished"}' }),
  'GET /api/v1/auth/session': adminSession,
  'POST /api/v1/quarantine/Q-001/reprocess': { command_id: 'x', seq: 1, event_ids: ['e'], replayed: false },
})

describe('карантин', () => {
  it('открытые записи: код причины по-русски, источник и номер, время; запрос с state=open', async () => {
    const calls = mockApi(routes())
    const w = await mountWidget(QuarantineWidget, props)
    expect(calls.find((c) => c.path === '/api/v1/quarantine')?.query.get('state')).toBe('open')
    const rows = w.findAll('li.row')
    expect(rows).toHaveLength(3)
    expect(rows[0]!.text()).toContain('Нет обязательного поля')
    expect(rows[0]!.text()).toContain('MES № 77')
    expect(rows[0]!.text()).toContain('нет поля item_id')
    expect(rows[2]!.text()).toContain('Принято после переобработки')
  })

  it('выбор: исходное содержимое, отпечаток, адрес; «номер изделия без человека не подставляется»', async () => {
    mockApi(routes())
    const w = await mountWidget(QuarantineWidget, props)
    await w.find('li.row[data-id="Q-001"]').trigger('click')
    await settle()
    const card = w.find('[data-testid="detail"]')
    expect(card.find('[data-testid="content"]').text()).toContain('operation.operation.finished')
    expect(card.text()).toContain('sha256:abcd')
    expect(card.text()).toContain('Номер изделия без человека не подставляется')
  })

  it('переобработка — только с основанием; принята один раз', async () => {
    const calls = mockApi(routes())
    const w = await mountWidget(QuarantineWidget, props)
    await w.find('li.row[data-id="Q-001"]').trigger('click')
    await settle()
    expect(w.find('[data-testid="reprocess"]').attributes('disabled')).toBeDefined()
    await w.find('[data-testid="reason"] input').setValue('Источник прислал исправленное')
    await w.find('form.reprocess').trigger('submit')
    await settle()
    const post = calls.find((c) => c.method === 'POST')
    expect(post?.path).toBe('/api/v1/quarantine/Q-001/reprocess')
    expect(post?.body).toMatchObject({ reason: { text: 'Источник прислал исправленное' }, policy_seq: 7 })
    expect(post?.body).not.toHaveProperty('discard')
    expect(w.find('[data-testid="done"]').text()).toBe('Обработано повторно и принято один раз')
  })

  it('«все» — запрос без фильтра состояния', async () => {
    const calls = mockApi(routes())
    const w = await mountWidget(QuarantineWidget, props)
    await w.find('[data-testid="filter-all"] input').setValue(true)
    await settle()
    expect(calls.filter((c) => c.path === '/api/v1/quarantine').at(-1)?.query.has('state')).toBe(false)
  })

  it('сводка (срез view: summary): список без содержимого и команд', async () => {
    const calls = mockApi(routes())
    const w = await mountWidget(QuarantineWidget, { ...props, slice: { view: 'summary' } })
    await w.find('li.row[data-id="Q-001"]').trigger('click')
    await settle()
    expect(w.find('[data-testid="detail"]').exists()).toBe(false)
    expect(calls.some((c) => c.path === '/api/v1/quarantine/Q-001')).toBe(false)
  })

  it('пусто — «карантин пуст»', async () => {
    mockApi({ 'GET /api/v1/quarantine': { items: [] } })
    const w = await mountWidget(QuarantineWidget, props)
    expect(w.text()).toContain('Карантин пуст')
  })
})
