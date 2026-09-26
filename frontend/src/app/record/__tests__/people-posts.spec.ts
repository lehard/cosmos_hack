// Окна поста и сотрудника (Д-70): `?open=workplace:…` — что на посту сейчас,
// назначения в смене, история поста из журнала; `?open=person:…` — профиль,
// пост сейчас, квалификации. Чтения без права — ошибка в своём блоке.
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
const posts = {
  items: [
    { workplace_id: 'WP-W2', station: 'Сварочный пост 2', workshop: 'Сварочный цех', assigned: { person_id: 'P-17', display: 'Сварщик С-17' }, presence: 'present', current_item: { item_id: 'ENT:FL-0041', label: 'ФЛ-0041' } },
  ],
}
const assignments = {
  basis_seq: 10,
  items: [
    { workplace_id: 'WP-W2', shift_id: 'S-1', person_id: 'P-17', assignee_role: 'performer', admitted: true, qualification_ok: true },
    { workplace_id: 'WP-W9', shift_id: 'S-1', person_id: 'P-99', assignee_role: 'performer', admitted: true, qualification_ok: true },
  ],
}
const entry = (seq: number, event_type: string, data: Record<string, unknown>) => ({
  seq,
  chain: 'main',
  entry_kind: 'decision',
  event_type,
  event_id: `E-${seq}`,
  causation_id: null,
  correlation_id: `C-${seq}`,
  committed_at: '2026-09-26T06:00:00Z',
  occurred_at: `2026-09-26T0${seq}:00:00Z`,
  received_at: '2026-09-26T06:00:00Z',
  recorded_at: '2026-09-26T06:00:00Z',
  provenance_class: 'personal',
  schema_version: 1,
  signature_status: 'not_checked',
  signers: [],
  source_id: 'web',
  stream: 'workplace:WP-W2',
  data,
})
const journal = {
  items: [
    entry(1, 'access.assignment.set', { person_id: 'P-17', workplace_id: 'WP-W2', shift_id: 'S-1', assignee_role: 'performer' }),
    entry(2, 'access.token.presence_changed', { person_id: 'P-17', workplace_id: 'WP-W2', present: true }),
  ],
}
const person = {
  person_id: 'P-17',
  display_name: 'Сварщик С-17',
  org_unit: 'Сварочный цех',
  login: 'welder17',
  account_status: 'active',
  policy_seq: 3,
  roles: [{ role_id: 'performer', scope: 'ent01/b1/wc', valid_from: '2026-01-01T00:00:00Z' }],
}
const quals = { items: [{ person_id: 'P-17', qualification_id: 'Q-1', scope: 'Сварка НАКС', status: 'valid', valid_from: '2026-01-01T00:00:00Z', valid_until: '2027-01-01T00:00:00Z' }] }

/** Ответы сервера по пути; `deny` — операции, закрытые правом. */
function serve(deny: string[] = [], pending: string[] = []) {
  vi.stubGlobal(
    'fetch',
    vi.fn(async (url: string) => {
      const path = String(url).split('?')[0]!
      if (deny.includes(path)) return problem(403, 'access.forbidden')
      if (pending.includes(path)) return problem(501, 'api.not_implemented')
      if (path === '/api/v1/workplaces') return json(posts)
      if (path === '/api/v1/assignments') return json(assignments)
      if (path === '/api/v1/journal') return json(journal)
      if (path === '/api/v1/persons/P-17') return json(person)
      if (path === '/api/v1/qualifications') return json(quals)
      return problem(404, 'api.not_found')
    }),
  )
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
  it('заголовок — пост и цех; сейчас, назначения смены, история (новые сверху)', async () => {
    serve()
    const { w } = await mountAt('/desk?open=workplace:WP-W2')
    await until(() => expect($$('[data-testid="workplace-history-row"]').length).toBe(2))
    const head = $('[data-testid="record-drawer-head"]')!.textContent!
    expect(head).toContain('Пост')
    expect(head).toContain('Сварочный пост 2')
    expect(head).toContain('Сварочный цех')
    expect($('[data-testid="workplace-now"]')!.textContent).toContain('ФЛ-0041')
    // Только назначения этого поста.
    const shift = $('[data-testid="workplace-assignments"]')!.textContent!
    expect(shift).toContain('Сварщик С-17')
    expect(shift).not.toContain('P-99')
    const rows = $$('[data-testid="workplace-history-row"]').map((r) => r.textContent)
    expect(rows[0]).toContain('ключ вставлен')
    expect(rows[1]).toContain('Назначение на пост')
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

  it('журнал закрыт правом — ошибка в блоке истории; операции нет — «появится»', async () => {
    serve(['/api/v1/journal'])
    let m = await mountAt('/desk?open=workplace:WP-W2')
    await until(() => expect($('[data-testid="workplace-history-error"]')).not.toBeNull())
    expect($('[data-testid="workplace-now"]')!.textContent).toContain('Сварщик С-17')
    m.w.unmount()
    document.body.innerHTML = ''

    serve([], ['/api/v1/journal'])
    m = await mountAt('/desk?open=workplace:WP-W2')
    await until(() => expect($('[data-testid="workplace-history-pending"]')).not.toBeNull())
    expect($('[data-testid="workplace-history-pending"]')!.textContent).toContain('История постов появится, когда бэкенд её отдаст')
    m.w.unmount()
  })
})

describe('окно сотрудника', () => {
  it('профиль, роли, пост сейчас, квалификации', async () => {
    serve()
    const { w, router } = await mountAt('/desk?open=person:P-17')
    await until(() => expect($('[data-qualification="valid"]')).not.toBeNull())
    const head = $('[data-testid="record-drawer-head"]')!.textContent!
    expect(head).toContain('Сотрудник')
    expect(head).toContain('Сварщик С-17')
    expect($('[data-testid="person-profile"]')!.textContent).toContain('welder17')
    expect($('[data-testid="person-roles"]')!.textContent).toContain('Исполнитель')
    expect($('[data-qualification="valid"]')!.textContent).toContain('Сварка НАКС')
    // Пост сейчас → окно поста.
    $('[data-testid="person-post"] [data-workplace="WP-W2"]')!.click()
    await until(() => expect(router.currentRoute.value.query.open).toBe('workplace:WP-W2'))
    w.unmount()
  })

  it('профиль закрыт правом — ошибка, имя и пост — из панели «Посты»', async () => {
    serve(['/api/v1/persons/P-17', '/api/v1/qualifications'])
    const { w } = await mountAt('/desk?open=person:P-17')
    await until(() => expect($('[data-testid="person-profile-error"]')).not.toBeNull())
    expect($('[data-testid="record-drawer-head"]')!.textContent).toContain('Сварщик С-17')
    await until(() => expect($('[data-testid="person-post"]')!.textContent).toContain('Сварочный пост 2'))
    expect($('[data-testid="person-qualifications"]')!.textContent).toContain('Квалификации недоступны')
    w.unmount()
  })
})
