// Экран входа (FR-128, UI-1, UI-2): корпоративная карточка «логин — пароль»,
// под ней — роли из ТЗ одной кнопкой и свёрнутый блок остальных демо-персон (только если сервер их отдал), который
// разворачивается в список по ролям с поиском.
import { flushPromises, mount } from '@vue/test-utils'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { createPinia } from 'pinia'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import { personasKey, type DemoPersona } from '@/entities/session'
import { i18n } from '@/shared/i18n'
import LoginPage from '../LoginPage.vue'

/** Образец персон — только для тестов. */
const personas = (): DemoPersona[] => [
  { id: 'P-QI-1', name: 'Контролёр ОТК 1', role: { id: 'quality_inspector', title: 'Контролёр качества' } },
  { id: 'P-QI-2', name: 'Контролёр ОТК 2', role: { id: 'quality_inspector', title: 'Контролёр качества' } },
  {
    id: 'P-SF-4',
    name: 'Мастер сборочно-испытательного цеха',
    role: { id: 'site_foreman', title: 'Мастер участка' },
    scope: 'Корпус 1 → Сборочно-испытательный цех',
  },
  { id: 'P-SF-5', name: 'Мастер склада и ВК', role: { id: 'site_foreman', title: 'Мастер участка' }, scope: 'Корпус 1 → Склад' },
  { id: 'P-AUD', name: 'Аудитор ИБ', role: { id: 'security_auditor', title: 'Аудитор ИБ' } },
]

/** Сервер: демо-персон нет (профиль live) — access.persona.list отвечает 404. */
const noDemoServer = () =>
  vi.stubGlobal(
    'fetch',
    vi.fn(async () =>
      new Response(JSON.stringify({ type: 'urn:ant:problem:api.not_found', title: 'Не найдено', status: 404, code: 'api.not_found' }), {
        status: 404,
        headers: { 'Content-Type': 'application/problem+json' },
      }),
    ),
  )

async function mountLogin(seed: DemoPersona[] | null) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  if (seed) queryClient.setQueryData(personasKey, { data: { items: seed }, status: 200, headers: new Headers() })
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/login', name: 'login', component: LoginPage },
      { path: '/desk', name: 'desk', component: { template: '<div />' } },
    ],
  })
  await router.push('/login')
  const w = mount(LoginPage, { global: { plugins: [createPinia(), i18n, router, [VueQueryPlugin, { queryClient }]] } })
  await flushPromises()
  return w
}

afterEach(() => vi.unstubAllGlobals())

describe('экран входа', () => {
  it('корпоративный вход: имя системы, логин, пароль, «Войти», заявка на доступ', async () => {
    const w = await mountLogin(personas())
    const card = w.get('[data-testid="login-card"]')
    expect(w.text()).toContain('Главный')
    expect(w.text()).toContain('Платформа управления качеством производства')
    expect(card.find('input#login-name').exists()).toBe(true)
    expect(card.find('input#login-password').attributes('type')).toBe('password')
    expect(card.get('[data-testid="login-submit"]').text()).toBe('Войти')
    expect(card.get('[data-testid="request-open"]').text()).toBe('Подать заявку на доступ')
  })

  it('роли из ТЗ — сразу: первый сотрудник каждой роли одной кнопкой; вход по нажатию', async () => {
    const w = await mountLogin(personas())
    const quick = w.get('[data-testid="demo-quick"]')
    expect(quick.text()).toContain('Демо-вход по ролям')
    expect(quick.findAll('[data-persona]').map((b) => b.attributes('data-persona'))).toEqual(['P-QI-1', 'P-SF-4'])
    const qi = quick.get('[data-persona="P-QI-1"]')
    expect(qi.text()).toContain('Контролёр ОТК 1')
    expect(qi.get('.quick-name').classes()).toContain('ant-ellipsis')
  })

  it('остальные — под «Другие сотрудники»: свёрнуто, по ролям, без тех, кто уже на виду', async () => {
    const w = await mountLogin(personas())
    const demo = w.get('[data-testid="demo-personas"]')
    const toggle = demo.get('[data-testid="demo-toggle"]')
    expect(toggle.text()).toContain('Другие сотрудники')
    expect(toggle.text()).toContain('3 сотрудника')
    expect(toggle.attributes('aria-expanded')).toBe('false')
    expect(w.find('[data-testid="demo-list"]').exists()).toBe(false)

    await toggle.trigger('click')
    expect(toggle.attributes('aria-expanded')).toBe('true')
    const list = w.get('[data-testid="demo-list"]')
    expect(list.findAll('[data-role]').map((g) => g.attributes('data-role'))).toEqual(['quality_inspector', 'site_foreman', 'security_auditor'])
    expect(list.findAll('[data-persona]').map((b) => b.attributes('data-persona'))).toEqual(['P-QI-2', 'P-SF-5', 'P-AUD'])
    const long = list.get('[data-persona="P-SF-5"]')
    expect(long.get('.persona-name').classes()).toContain('ant-ellipsis')
    expect(long.attributes('title')).toContain('Мастер склада и ВК')

    await toggle.trigger('click')
    expect(w.find('[data-testid="demo-list"]').exists()).toBe(false)
  })

  it('поиск по остальным: по имени, роли, области; «никого» — пустое состояние', async () => {
    const w = await mountLogin(personas())
    await w.get('[data-testid="demo-toggle"]').trigger('click')
    const list = () => w.get('[data-testid="demo-list"]')
    const search = w.get('[data-testid="demo-search"] input')
    await search.setValue('склад')
    expect(list().findAll('[data-persona]').map((b) => b.attributes('data-persona'))).toEqual(['P-SF-5'])
    await search.setValue('контролёр качества')
    expect(list().findAll('[data-persona]').map((b) => b.attributes('data-persona'))).toEqual(['P-QI-2'])
    await search.setValue('нет такого')
    expect(list().findAll('[data-persona]')).toHaveLength(0)
    expect(w.text()).toContain('Никого не нашли по запросу «нет такого»')
  })

  it('без демо-персон (вне профилей demo и fixtures) блока нет', async () => {
    noDemoServer()
    const w = await mountLogin(null)
    expect(w.find('[data-testid="demo-personas"]').exists()).toBe(false)
    expect(w.find('[data-testid="demo-quick"]').exists()).toBe(false)
    expect(w.find('[data-testid="login-form"]').exists()).toBe(true)
  })

  it('пустые логин и пароль — ошибки под полями, запроса нет', async () => {
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    const w = await mountLogin(personas())
    await w.get('[data-testid="login-form"]').trigger('submit')
    expect(w.text()).toContain('Введите логин')
    expect(w.text()).toContain('Введите пароль')
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('«Подать заявку на доступ» открывает форму заявки и возвращает ко входу', async () => {
    const w = await mountLogin(personas())
    await w.get('[data-testid="request-open"]').trigger('click')
    expect(w.find('[data-testid="request-form"]').exists()).toBe(true)
    expect(w.find('[data-testid="login-form"]').exists()).toBe(false)
    await w.get('[data-testid="request-form"]').trigger('submit')
    expect(w.text()).toContain('Укажите имя')
    await w.get('button.back').trigger('click')
    expect(w.find('[data-testid="login-form"]').exists()).toBe(true)
  })
})
