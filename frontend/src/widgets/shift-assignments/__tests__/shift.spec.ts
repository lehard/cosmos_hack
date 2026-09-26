// Смена и назначения (FR-81, PRD §11.18, UJ-7): исполнителей назначает мастер —
// только допущенных по квалификации; контролёра — запрос мастера с
// согласованием начальника ОТК, назначение — по закрытому маршруту документа.
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mockApi, mountWidget, problem, settle } from '@/entities/run/__tests__/api'
import { assignments, at, foremanSession, locations, posts, qualifications, shifts } from '@/entities/workplace/__tests__/fixtures'
import type { AccessPerson, AccessRoleList } from '@/shared/api/generated/model'
import { candidates, rolesInheriting, scopeTouches } from '../model/shift'
import ShiftAssignmentsView from '../ui/ShiftAssignmentsView.vue'
import ShiftAssignmentsWidget from '../ui/ShiftAssignmentsWidget.vue'

afterEach(() => vi.unstubAllGlobals())
const props = { widgetId: 'shift-assignments', titleKey: 'desks.shift', density: 'large' }
const receipt = (seq: number) => ({ command_id: 'c', seq, event_ids: ['e'], replayed: false })

const grant = (role_id: string, scope: string) => ({ role_id, scope, valid_from: '2026-01-01T00:00:00Z' })
const persons = (): AccessPerson[] => [
  { person_id: 'W21', display_name: 'Сварщик W21', account_status: 'active', policy_seq: 3, roles: [grant('welder', 'ent01/b1/wc')] },
  { person_id: 'W22', display_name: 'Сварщик W22', account_status: 'active', policy_seq: 3, roles: [grant('performer', 'ent01/b1/wc')] },
  { person_id: 'W23', display_name: 'Сварщик W23', account_status: 'active', policy_seq: 3, roles: [grant('performer', 'ent01/b1/wc')] },
  { person_id: 'O17', display_name: 'Оператор ЧПУ O17', account_status: 'active', policy_seq: 3, roles: [grant('performer', 'ent01/b1/mc')] },
  { person_id: 'INS-01', display_name: 'Контролёр ОТК 1', account_status: 'active', policy_seq: 3, roles: [grant('quality_inspector', 'ent01/b1')] },
]
const roles = (): AccessRoleList => ({
  policy_seq: 3,
  items: [
    { id: 'performer', title: 'Исполнитель', case_role: false, inherits: ['staff'], actions: [] },
    { id: 'welder', title: 'Сварщик', case_role: false, inherits: ['performer'], actions: [] },
    { id: 'quality_inspector', title: 'Контролёр качества', case_role: true, inherits: ['staff'], actions: [] },
  ],
  authorities: [],
  stamp_kinds: [],
})

function routes(over: Record<string, unknown> = {}) {
  return {
    'GET /api/v1/auth/session': foremanSession(),
    'GET /api/v1/reference/locations': { items: locations() },
    'GET /api/v1/reference/shifts': { items: shifts() },
    'GET /api/v1/workplaces': { items: posts() },
    'GET /api/v1/assignments': assignments(),
    'GET /api/v1/persons': { items: persons() },
    'GET /api/v1/roles': roles(),
    'GET /api/v1/qualifications': { items: qualifications() },
    'GET /api/v1/documents': {
      items: [
        { document_id: 'DOC-CA-1', subject: { entity: 'workplace', id: 'WP-QC-WC' }, template: 'controller-assignment@1', title: 'Назначение контролёра', status: 'route_closed', version: 1, closed_at: at('07:40') },
      ],
    },
    'GET /api/v1/workplaces/WP-WELD-2/candidates': {
      workplace_id: 'WP-WELD-2',
      basis_seq: 9000,
      items: [
        { person_id: 'W22', display: 'Сварщик W22', role: 'performer', allowed: true, qualification_verdict: 'ok', why: 'Аттестация сварщика действует' },
        { person_id: 'W23', display: 'Сварщик W23', role: 'performer', allowed: false, qualification_verdict: 'expired', why: 'Аттестация истекла 21.09' },
      ],
    },
    'GET /api/v1/workplaces/WP-QC-WC/candidates': {
      workplace_id: 'WP-QC-WC',
      basis_seq: 9000,
      items: [{ person_id: 'INS-01', display: 'Контролёр ОТК 1', role: 'quality_inspector', allowed: false, needs_approval: true, qualification_verdict: 'ok', why: 'Назначение — по согласованию начальника ОТК' }],
    },
    'POST /api/v1/assignments': receipt(701),
    'POST /api/v1/assignments/clear': receipt(702),
    'POST /api/v1/documents/requests': receipt(703),
    ...over,
  }
}

describe('кандидаты на пост', () => {
  it('наследники роли, область цеха, квалификация «на глаз»', () => {
    const r = rolesInheriting(roles().items, 'performer')
    expect([...r].sort()).toEqual(['performer', 'welder'])
    expect(scopeTouches('ent01/b1', 'ent01/b1/wc')).toBe(true)
    expect(scopeTouches('ent01/b1/mc', 'ent01/b1/wc')).toBe(false)
    const list = candidates(persons(), r, 'ent01/b1/wc', qualifications())
    expect(list.map((c) => [c.person.person_id, c.verdict])).toEqual([
      ['W21', 'ok'],
      ['W22', 'ok'],
      ['W23', 'expired'],
    ])
  })
})

describe('виджет «Смена»', () => {
  it('план и факт смены по постам; сварщика с истёкшей аттестацией выбрать нельзя (UJ-7)', async () => {
    mockApi(routes())
    const w = await mountWidget(ShiftAssignmentsWidget, props)
    const w2 = w.find('[data-workplace="WP-WELD-2"]')
    expect(w2.text()).toContain('По графику на месте — ключ не вставлен')
    expect(w2.find('[data-testid="performers"]').text()).toContain('Сварщик W21')
    expect(w2.find('[data-testid="performers"]').text()).toContain('Нет допуска к рабочему месту')
    // Назначение — в правом окне: кандидаты с вердиктом квалификации от сервера (разбор стола мастера).
    await w2.find('[data-testid="configure"]').trigger('click')
    await settle()
    const w23 = document.querySelector('[data-testid="cand-W23"]') as HTMLElement
    expect(w23.textContent).toContain('Аттестация истекла 21.09')
    expect((w23.querySelector('input') as HTMLInputElement).disabled).toBe(true)
    expect((document.querySelector('[data-testid="cand-W22"] input') as HTMLInputElement).disabled).toBe(false)
    w.unmount()
  })

  it('назначить исполнителя: access.assignment.set со сменой, basis_seq назначений и policy_seq сеанса', async () => {
    const calls = mockApi(routes())
    const w = await mountWidget(ShiftAssignmentsWidget, props)
    await w.findComponent(ShiftAssignmentsView).vm.$emit('assign', 'WP-WELD-2', 'performer', 'W22', null)
    await settle()
    const post = calls.find((c) => c.method === 'POST' && c.path === '/api/v1/assignments')
    expect(post?.body).toMatchObject({ assignee_role: 'performer', person_id: 'W22', workplace_id: 'WP-WELD-2', shift_id: 'SHIFT-1', basis_seq: 700, policy_seq: 3 })
    expect(post?.body).not.toHaveProperty('approval_document_id')
    expect(w.find('[data-testid="result"]').text()).toBe('Назначение записано: Сварщик W22, запись № 701')
  })

  it('сервер отказал (квалификация истекла) — текст отказа, назначения нет', async () => {
    mockApi(routes({ 'POST /api/v1/assignments': problem(422, 'process.qualification_expired') }))
    const w = await mountWidget(ShiftAssignmentsWidget, props)
    await w.findComponent(ShiftAssignmentsView).vm.$emit('assign', 'WP-WELD-2', 'performer', 'W23', null)
    await settle()
    expect(w.find('[data-testid="command-error"]').exists()).toBe(true)
    expect(w.find('[data-testid="result"]').exists()).toBe(false)
  })

  it('снять с поста — с основанием', async () => {
    const calls = mockApi(routes())
    const w = await mountWidget(ShiftAssignmentsWidget, props)
    const row = w.find('[data-workplace="WP-WELD-2"] [data-assigned="W21"]')
    await row.find('[data-testid="clear"]').trigger('click')
    await row.find('[data-testid="clear-reason"] input').setValue('Не вошёл на рабочее место')
    await row.find('form').trigger('submit')
    await settle()
    const post = calls.find((c) => c.method === 'POST' && c.path === '/api/v1/assignments/clear')
    expect(post?.body).toMatchObject({ person_id: 'W21', workplace_id: 'WP-WELD-2', shift_id: 'SHIFT-1', reason: { text: 'Не вошёл на рабочее место' } })
  })

  it('контролёра — запросом с согласованием начальника ОТК (PRD §11.18)', async () => {
    const calls = mockApi(routes())
    const w = await mountWidget(ShiftAssignmentsWidget, props)
    expect(w.text()).toContain('Контролёра назначает мастер по согласованию начальника ОТК')
    await w.findComponent(ShiftAssignmentsView).vm.$emit('requestController', 'WP-QC-WC', 'INS-01', 'На смену вместо К-05')
    await settle()
    const post = calls.find((c) => c.method === 'POST' && c.path === '/api/v1/documents/requests')
    expect(post?.body).toMatchObject({
      template: 'controller-assignment@1',
      subject: { entity: 'workplace', id: 'WP-QC-WC' },
      decision: 'quality_inspector:INS-01@SHIFT-1',
      comment: 'На смену вместо К-05',
      policy_seq: 3,
    })
    expect(calls.some((c) => c.method === 'POST' && c.path === '/api/v1/assignments')).toBe(false)
  })

  it('маршрут согласования закрыт — назначить контролёра по документу', async () => {
    const calls = mockApi(routes())
    const w = await mountWidget(ShiftAssignmentsWidget, props)
    const qc = w.find('[data-workplace="WP-QC-WC"]')
    expect(qc.find('[data-document="DOC-CA-1"]').text()).toContain('Согласовано')
    expect(w.find('[data-workplace="WP-WELD-1"] [data-testid="approvals"]').exists()).toBe(false)
    await qc.find('[data-testid="use-approval"]').trigger('click')
    await settle()
    const ins = document.querySelector('[data-record="post-assignment"] input[value="INS-01"]') as HTMLInputElement
    ins.checked = true
    ins.dispatchEvent(new Event('change'))
    await settle()
    ;(document.querySelector('[data-testid="assign-inspector"]') as HTMLElement).click()
    await settle()
    const post = calls.find((c) => c.method === 'POST' && c.path === '/api/v1/assignments')
    expect(post?.body).toMatchObject({ assignee_role: 'quality_inspector', person_id: 'INS-01', approval_document_id: 'DOC-CA-1' })
  })
})
