// Окна поста и сотрудника (Д-70, UI-16): `?open=workplace:…` — карточка поста
// (`access.workplace.read`: сейчас, назначения смены) и история поста
// (`access.workplace.history`, «показать ещё» по курсору); `?open=person:…` —
// карточка сотрудника (`access.person.card`). Чтение без права — ошибка в блоке.
import { defineComponent, h, provide } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { createPinia } from 'pinia'
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import { i18n } from '@/shared/i18n'
import { RECORD_DRAWER } from '@/shared/model/record'
import RecordDrawerHost from '../RecordDrawerHost.vue'
import { RECORD_KINDS, recordKinds } from '../registry'

const $ = (sel: string) => document.querySelector<HTMLElement>(sel)
const $$ = (sel: string) => [...document.querySelectorAll<HTMLElement>(sel)]
const until = (check: () => void) => vi.waitFor(check, { timeout: 10_000 })
const json = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), { status, headers: { 'Content-Type': status === 200 ? 'application/json' : 'application/problem+json', 'Ant-Backend': 'fixtures' } })
const problem = (status: number, code: string) => json({ type: `urn:ant:problem:${code}`, title: 'нет', status, code }, status)

/** Образцы — только для тестов. */
const card = {
  workplace_id: 'WP-W2',
  station: 'Сварочный пост 2',
  workshop: 'Сварочный цех',
  scope: 'ent01/b1/wc/w2',
  assigned: { person_id: 'P-17', display: 'Сварщик С-17' },
  presence: 'present',
  current_item: { item_id: 'ENT:FL-0041', label: 'ФЛ-0041' },
  shift_id: 'S-1',
  assignments: [
    { person_id: 'P-17', person_display: 'Сварщик С-17', shift_id: 'S-1', assignee_role: 'performer', qualification_ok: true },
    { person_id: 'INS-01', person_display: 'Контролёр ОТК 1', shift_id: 'S-1', assignee_role: 'quality_inspector', qualification_ok: true },
  ],
}
const ev = (seq: number, kind: string, extra: Record<string, unknown> = {}) => ({
  seq,
  at: `2026-09-26T0${seq}:00:00Z`,
  event_type: 'access.x',
  kind,
  person_id: 'P-17',
  person_display: 'Сварщик С-17',
  shift_id: 'S-1',
  ...extra,
})
/** Первая страница истории (новые сверху) и вторая — по курсору. */
const history1 = { workplace_id: 'WP-W2', items: [ev(4, 'revoked', { reason: 'zone_exit' }), ev(3, 'token_in')], next_cursor: '2' }
const history2 = { workplace_id: 'WP-W2', items: [ev(2, 'cleared', { reason: 'Перевод на другой пост' }), ev(1, 'assigned')] }
const person = {
  person_id: 'P-17',
  display_name: 'Сварщик С-17',
  org_unit: 'Сварочный цех',
  policy_seq: 3,
  roles: [{ role_id: 'performer', scope: 'ent01/b1/wc', valid_from: '2026-01-01T00:00:00Z' }],
  qualifications: [
    { person_id: 'P-17', qualification_id: 'welder_argon_arc_amg6', scope: 'ent01/b1/wc', status: 'valid', valid_from: '2026-01-01T00:00:00Z', valid_until: '2027-01-01T00:00:00Z' },
  ],
  posts: [{ workplace_id: 'WP-W2', station: 'Сварочный пост 2', shift_id: 'S-1', assignee_role: 'performer' }],
}

/** Ответы сервера по пути; `deny` — операции, закрытые правом; `calls` — запрошенные пути. */
function serve(deny: string[] = []) {
  const calls: string[] = []
  vi.stubGlobal(
    'fetch',
    vi.fn(async (url: string) => {
      const [path, query = ''] = String(url).split('?') as [string, string?]
      calls.push(path)
      if (deny.includes(path)) return problem(403, 'access.forbidden')
      if (path === '/api/v1/workplaces/WP-W2') return json(card)
      if (path === '/api/v1/workplaces/WP-W2/history') return json(new URLSearchParams(query).get('cursor') === '2' ? history2 : history1)
      if (path === '/api/v1/persons/P-17/card') return json(person)
      return problem(404, 'api.not_found')
    }),
  )
  return calls
}

beforeAll(async () => {
  await Promise.all([recordKinds.workplace!.load(), recordKinds.person!.load()])
}, 60_000)
afterEach(() => {
  vi.unstubAllGlobals()
  document.body.innerHTML = ''
})

const Shell = defineComponent({
  setup() {
    provide(RECORD_DRAWER, { kinds: RECORD_KINDS })
    return () => h(RecordDrawerHost)
  },
})

async function mountAt(path: string) {
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/desk', component: { template: '<div />' } }] })
  await router.push(path)
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const w = mount(Shell, { attachTo: document.body, global: { plugins: [createPinia(), i18n, router, [VueQueryPlugin, { queryClient }]] } })
  await flushPromises()
  return { w, router }
}

describe('окно поста', () => {
  it('заголовок — пост и цех; сейчас и назначения смены из карточки, история (новые сверху)', async () => {
    const calls = serve()
    const { w } = await mountAt('/desk?open=workplace:WP-W2')
    await until(() => expect($$('[data-testid="workplace-history-row"]').length).toBe(2))
    const head = $('[data-testid="record-drawer-head"]')!.textContent!
    expect(head).toContain('Пост')
    expect(head).toContain('Сварочный пост 2')
    expect(head).toContain('Сварочный цех')
    expect($('[data-testid="workplace-now"]')!.textContent).toContain('ФЛ-0041')
    const shift = $('[data-testid="workplace-assignments"]')!.textContent!
    expect(shift).toContain('Сварщик С-17')
    expect(shift).toContain('Контролёр ОТК 1')
    expect(shift).toContain('Контролёр')
    const rows = $$('[data-testid="workplace-history-row"]').map((r) => r.textContent)
    // Вид события и основание — по-русски, без кодов.
    expect(rows[0]).toContain('Допуск к рабочему месту снят автоматически: выход из зоны')
    expect(rows[0]).not.toContain('zone_exit')
    expect(rows[1]).toContain('Ключ вставлен')
    // Только новые операции — без журнала, назначений и панели «Посты».
    expect(calls.every((p) => p.startsWith('/api/v1/workplaces/WP-W2'))).toBe(true)
    w.unmount()
  })

  it('«показать ещё» — следующая страница истории по курсору', async () => {
    serve()
    const { w } = await mountAt('/desk?open=workplace:WP-W2')
    await until(() => expect($('[data-testid="workplace-history-more"]')).not.toBeNull())
    $('[data-testid="workplace-history-more"]')!.click()
    await until(() => expect($$('[data-testid="workplace-history-row"]').length).toBe(4))
    const rows = $$('[data-testid="workplace-history-row"]').map((r) => r.textContent)
    expect(rows[2]).toContain('Снят с поста: Перевод на другой пост')
    expect(rows[3]).toContain('Назначен на пост')
    expect($('[data-testid="workplace-history-more"]')).toBeNull()
    w.unmount()
  })

  it('назначенный → окно сотрудника, текущее изделие → окно изделия', async () => {
    serve()
    const { w, router } = await mountAt('/desk?open=workplace:WP-W2')
    await until(() => expect($('[data-testid="workplace-person"]')).not.toBeNull())
    $('[data-testid="workplace-person"]')!.click()
    await until(() => expect(router.currentRoute.value.query.open).toBe('person:P-17'))
    await router.push('/desk?open=workplace:WP-W2')
    await until(() => expect($('[data-testid="workplace-item"]')).not.toBeNull())
    $('[data-testid="workplace-item"]')!.click()
    await until(() => expect(router.currentRoute.value.query.open).toBe('item:ENT:FL-0041'))
    w.unmount()
  })

  it('чтение закрыто правом — ошибка в своём блоке', async () => {
    serve(['/api/v1/workplaces/WP-W2', '/api/v1/workplaces/WP-W2/history'])
    const { w } = await mountAt('/desk?open=workplace:WP-W2')
    await until(() => expect($('[data-testid="workplace-card-error"]')).not.toBeNull())
    await until(() => expect($('[data-testid="workplace-history-error"]')).not.toBeNull())
    expect($('[data-testid="workplace-card-error"]')!.textContent).toContain('Карточка поста недоступна')
    expect($('[data-testid="workplace-assignments"]')).toBeNull()
    w.unmount()
  })
})

describe('окно сотрудника', () => {
  it('карточка: подразделение, роли, посты, квалификации — по-русски', async () => {
    const calls = serve()
    const { w, router } = await mountAt('/desk?open=person:P-17')
    await until(() => expect($('[data-qualification="valid"]')).not.toBeNull())
    const head = $('[data-testid="record-drawer-head"]')!.textContent!
    expect(head).toContain('Сотрудник')
    expect(head).toContain('Сварщик С-17')
    expect(head).toContain('Сварочный цех')
    expect($('[data-testid="person-roles"]')!.textContent).toContain('Исполнитель')
    expect($('[data-qualification="valid"]')!.textContent).toContain('Аргонодуговая сварка АМг6')
    expect(calls).toEqual(['/api/v1/persons/P-17/card'])
    // Пост → окно поста.
    $('[data-testid="person-post"] [data-workplace="WP-W2"]')!.click()
    await until(() => expect(router.currentRoute.value.query.open).toBe('workplace:WP-W2'))
    w.unmount()
  })

  it('карточка закрыта правом — ошибка с текстом по коду', async () => {
    serve(['/api/v1/persons/P-17/card'])
    const { w } = await mountAt('/desk?open=person:P-17')
    await until(() => expect($('[data-testid="person-card-error"]')).not.toBeNull())
    expect($('[data-testid="person-card-error"]')!.textContent).toContain('Карточка сотрудника недоступна')
    expect($('[data-testid="person-profile"]')).toBeNull()
    w.unmount()
  })
})
