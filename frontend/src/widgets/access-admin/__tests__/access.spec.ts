// Доступ (FR-78, FR-128, FR-145; AD-11): сотрудники с ролями в области,
// роли и полномочия, клейма; выдача и отзыв с основанием и квитанцией CA.
import { afterEach, describe, expect, it, vi } from 'vitest'
import { adminSession, mockApi, mountWidget, settle } from '@/entities/run/__tests__/api'
import { persons, roles, stamps } from '@/entities/policy/__tests__/fixtures'
import { activationReady, grantReady } from '../model/draft'
import AccessAdminWidget from '../ui/AccessAdminWidget.vue'

afterEach(() => vi.unstubAllGlobals())
const props = { widgetId: 'access-admin', titleKey: 'desks.access' }
const routes = () => ({
  'GET /api/v1/persons': { items: persons() },
  'GET /api/v1/roles': roles(),
  'GET /api/v1/stamps': { items: stamps() },
  'GET /api/v1/auth/session': adminSession,
  'POST /api/v1/grants': { command_id: 'x', seq: 1, event_ids: ['e'], replayed: false, ca_ref: 'CA-21' },
  'POST /api/v1/grants/revocations': { command_id: 'x', seq: 2, event_ids: ['e'], replayed: false, ca_ref: 'CA-22' },
  'POST /api/v1/persons/P-NEW/account': { command_id: 'x', seq: 23, event_ids: ['e'], replayed: false },
  'POST /api/v1/persons/P-NONE/account': { command_id: 'x', seq: 24, event_ids: ['e'], replayed: false },
})

describe('доступ', () => {
  it('сотрудники: роль по названию политики, область и срок, состояние учётной записи', async () => {
    mockApi(routes())
    const w = await mountWidget(AccessAdminWidget, props)
    const qc = w.find('tr[data-person="P-QC-1"]')
    expect(qc.text()).toContain('Контролёр качества')
    expect(qc.text()).toContain('ent01/b1/qa')
    expect(qc.text()).toContain('Действует')
    expect(w.find('tr[data-person="P-NEW"]').text()).toContain('Ждёт активации')
    expect(w.find('tr[data-person="P-NEW"]').text()).toContain('Ролей нет')
  })

  it('роли и полномочия; клейма', async () => {
    mockApi(routes())
    const w = await mountWidget(AccessAdminWidget, props)
    await w.find('[data-testid="section-roles"] input').setValue(true)
    expect(w.find('[data-role="head_of_qc"]').text()).toContain('Наследует: Контролёр качества')
    expect(w.find('[data-role="quality_inspector"]').text()).toContain('роль кейса')
    expect(w.find('[data-authority="qc_acceptance"]').exists()).toBe(true)
    await w.find('[data-testid="section-stamps"] input').setValue(true)
    expect(w.find('[data-stamp="ST-7"]').text()).toContain('Выдано по приказу Приказ 12, вид контроля «ВИК»')
  })

  it('отзыв роли — с основанием; access.policy.revoke; квитанция с номером CA', async () => {
    const calls = mockApi(routes())
    const w = await mountWidget(AccessAdminWidget, props)
    await w.find('tr[data-person="P-QC-1"] [data-testid="revoke"]').trigger('click')
    expect(w.find('[data-testid="confirm-revoke"]').attributes('disabled')).toBeDefined()
    await w.find('tr[data-person="P-QC-1"] form input').setValue('Перевод на другой участок')
    await w.find('tr[data-person="P-QC-1"] form').trigger('submit')
    await settle()
    const post = calls.find((c) => c.method === 'POST')
    expect(post?.path).toBe('/api/v1/grants/revocations')
    expect(post?.body).toMatchObject({ kind: 'role', person_id: 'P-QC-1', subject_id: 'quality_inspector', scope: 'ent01/b1/qa', reason: { text: 'Перевод на другой участок' }, policy_seq: 7 })
    expect(w.find('[data-testid="result"]').text()).toBe('Отзыв записан: CA-22')
  })

  it('выдача клейма: без приказа и вида контроля не отправляется', () => {
    const base = { kind: 'stamp' as const, person_id: 'P-QC-1', subject_id: 'ST-8', scope: 'ent01', valid_from: '2026-09-26', valid_until: '', order_ref: '', inspection_kind: '', document_id: '' }
    expect(grantReady(base)).toBe(false)
    expect(grantReady({ ...base, order_ref: 'Приказ 13', inspection_kind: 'ВИК' })).toBe(true)
    expect(grantReady({ ...base, kind: 'role', subject_id: '' })).toBe(false)
  })

  it('выдача роли: access.policy.grant, даты — начало суток UTC, предупреждение о второй подписи', async () => {
    const calls = mockApi({ ...routes(), 'GET /api/v1/persons': { items: [] }, 'GET /api/v1/roles': { ...roles(), items: [] } })
    const w = await mountWidget(AccessAdminWidget, { ...props, slice: { section: 'grant' } })
    const form = w.find('[data-testid="grant"]')
    expect(form.text()).toContain('нужна вторая подпись руководителя производства')
    await form.find('[data-testid="person"] input').setValue('P-QC-2')
    await form.find('[data-testid="subject"] input').setValue('head_of_qc')
    await form.find('[data-testid="valid-from"]').setValue('2026-10-01')
    await form.trigger('submit')
    await settle()
    const post = calls.find((c) => c.method === 'POST')
    expect(post?.path).toBe('/api/v1/grants')
    expect(post?.body).toMatchObject({ kind: 'role', person_id: 'P-QC-2', role_id: 'head_of_qc', scope: 'ent01', valid_from: '2026-10-01T00:00:00Z', policy_seq: 7 })
    expect(post?.body).not.toHaveProperty('valid_until')
    expect(w.find('[data-testid="result"]').text()).toBe('Выдача записана: CA-21')
  })

  it('ни одна операция доступа не отвечает — «ошибка входа», но разделы и форма на месте', async () => {
    mockApi({})
    const w = await mountWidget(AccessAdminWidget, props)
    expect(w.find('.widget-frame').attributes('data-state')).toBe('input_error')
    expect(w.find('[data-testid="persons"]').text()).toContain('Не удалось выполнить действие')
    expect(w.find('[data-testid="section-grant"]').exists()).toBe(true)
  })

  it('активация по заявке: логин, начальная роль и область; пароль необязателен — его задал сотрудник (FR-128)', async () => {
    const calls = mockApi(routes())
    const w = await mountWidget(AccessAdminWidget, props)
    const row = () => w.find('tr[data-person="P-NEW"]')
    expect(w.find('tr[data-person="P-QC-1"] [data-testid="activate"]').exists()).toBe(false)
    await row().find('[data-testid="activate"]').trigger('click')
    const form = row().find('[data-testid="activation"]')
    expect(form.text()).toContain('Необязательно: сотрудник задал пароль в заявке')
    await form.find('[data-testid="activation-login"] input').setValue('novik')
    await form.find('[data-testid="activation-scope"] input').setValue('ent01/b1/qa')
    await form.trigger('submit')
    await settle()
    const post = calls.find((c) => c.method === 'POST' && c.path === '/api/v1/persons/P-NEW/account')
    expect(post?.body).toMatchObject({ login: 'novik', scope: 'ent01/b1/qa', basis_seq: 7, policy_seq: 7 })
    expect(post?.body).not.toHaveProperty('password')
    expect(w.find('[data-testid="result"]').text()).toBe('Учётная запись активирована: запись № 23')
  })

  it('активация без заявки: начальный пароль обязателен (не короче 8 символов)', async () => {
    const calls = mockApi({ ...routes(), 'GET /api/v1/persons': { items: [...persons(), { person_id: 'P-NONE', display_name: 'Без заявки Б.', account_status: 'none', policy_seq: 7, roles: [] }] } })
    const w = await mountWidget(AccessAdminWidget, props)
    const row = () => w.find('tr[data-person="P-NONE"]')
    await row().find('[data-testid="activate"]').trigger('click')
    await row().find('[data-testid="activation-login"] input').setValue('bez')
    expect(row().find('[data-testid="confirm-activation"]').attributes('disabled')).toBeDefined()
    await row().find('[data-testid="activation-password"] input').setValue('korotko')
    expect(row().find('[data-testid="confirm-activation"]').attributes('disabled')).toBeDefined()
    await row().find('[data-testid="activation-password"] input').setValue('dostatochno-dlinnyi')
    await row().find('[data-testid="activation"]').trigger('submit')
    await settle()
    const post = calls.find((c) => c.method === 'POST' && c.path === '/api/v1/persons/P-NONE/account')
    expect(post?.body).toMatchObject({ login: 'bez', password: 'dostatochno-dlinnyi', scope: 'ent01' })
  })

  it('готовность активации', () => {
    const d = { login: 'a', password: '', initial_role_id: '', scope: '' }
    expect(activationReady(d, true)).toBe(true)
    expect(activationReady(d, false)).toBe(false)
    expect(activationReady({ ...d, password: '12345678' }, false)).toBe(true)
    expect(activationReady({ ...d, login: ' ' }, true)).toBe(false)
  })
})
