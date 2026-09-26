// Меню стола в оболочке (Д-73): разделы стола роли слева вместо вкладок сверху;
// раздел один — меню нет; Alt+1…9 — раздел по номеру; свёрнутое состояние
// помнит браузер; на странице вне стола ни один раздел не выделен.
import { flushPromises, mount } from '@vue/test-utils'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { afterEach, describe, expect, it } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import { deskKey, type Desk } from '@/entities/desk'
import { i18n } from '@/shared/i18n'
import DeskNav from '../DeskNav.vue'

const tab = (id: string, title_key: string): Desk['tabs'][number] => ({ id, title_key, layout: 'single', slots: [] })
const manager: Desk = {
  version: 1,
  role: 'production_manager',
  title_key: 'desks.dashboard',
  density: 'comfortable',
  tabs: [tab('overview', 'desks.dashboard'), tab('proposals', 'desks.proposals'), tab('analytics', 'desks.analytics')],
} as Desk
const inspector: Desk = { ...manager, role: 'quality_inspector', tabs: [tab('queue', 'desks.decisionQueue')] } as Desk

afterEach(() => localStorage.clear())

async function mountNav(desk: Desk, path = '/desk') {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/desk/:tab?', name: 'desk', component: { template: '<div />' } },
      { path: '/items/:id', name: 'item', component: { template: '<div />' } },
    ],
  })
  await router.push(path)
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity } } })
  queryClient.setQueryData(deskKey, { data: desk, status: 200, headers: new Headers() })
  const w = mount(DeskNav, { attachTo: document.body, global: { plugins: [i18n, router, [VueQueryPlugin, { queryClient }]] } })
  await flushPromises()
  return { w, router }
}

describe('меню стола', () => {
  it('«Работа / Справочно»: у руководителя процессы, документы и аналитика — внизу под тихой подписью; Alt — по показанному порядку', async () => {
    const desk = { ...manager, tabs: [tab('overview', 'desks.dashboard'), tab('process', 'desks.processes'), tab('proposals', 'desks.proposals'), tab('analytics', 'desks.analytics')] } as Desk
    const { w } = await mountNav(desk)
    expect(w.findAll('[data-item]').map((x) => x.attributes('data-item'))).toEqual(['overview', 'proposals', 'process', 'analytics'])
    expect(w.find('[data-testid="nav-group"]').text()).toBe('Справочно')
    expect(w.find('.group-start [data-item]').attributes('data-item')).toBe('process')
  })

  it('разделы стола по порядку yaml; первый выбран по умолчанию; щелчок — адрес раздела', async () => {
    const { w, router } = await mountNav(manager)
    const items = w.findAll('[data-item]')
    expect(items.map((x) => x.attributes('data-item'))).toEqual(['overview', 'proposals', 'analytics'])
    expect(items[0]!.text()).toBe(i18n.global.t('desks.dashboard'))
    expect(items[0]!.attributes('aria-current')).toBe('page')
    await items[2]!.trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.params.tab).toBe('analytics')
    expect(w.find('[data-item="analytics"]').attributes('aria-current')).toBe('page')
    w.unmount()
  })

  it('раздел один — меню нет', async () => {
    const { w } = await mountNav(inspector)
    expect(w.find('[data-testid="desk-nav"]').exists()).toBe(false)
    w.unmount()
  })

  it('Alt+2 — второй раздел (по коду клавиши, при любой раскладке); Alt+9 без раздела — ничего', async () => {
    const { w, router } = await mountNav(manager)
    window.dispatchEvent(new KeyboardEvent('keydown', { code: 'Digit2', key: '"', altKey: true }))
    await flushPromises()
    expect(router.currentRoute.value.params.tab).toBe('proposals')
    window.dispatchEvent(new KeyboardEvent('keydown', { code: 'Digit9', key: '9', altKey: true }))
    window.dispatchEvent(new KeyboardEvent('keydown', { code: 'Digit3', key: '3' }))
    await flushPromises()
    expect(router.currentRoute.value.params.tab).toBe('proposals')
    w.unmount()
  })

  it('свернуть — полоса значков, состояние помнит браузер', async () => {
    const first = await mountNav(manager)
    await first.w.find('[data-testid="side-nav-toggle"]').trigger('click')
    expect(first.w.find('nav').attributes('data-collapsed')).toBe('true')
    first.w.unmount()
    const again = await mountNav(manager)
    expect(again.w.find('nav').attributes('data-collapsed')).toBe('true')
    again.w.unmount()
  })

  it('на странице паспорта меню видно, раздел не выделен', async () => {
    const { w } = await mountNav(manager, '/items/ENT:FL-0042')
    expect(w.findAll('[data-item]')).toHaveLength(3)
    expect(w.find('[aria-current="page"]').exists()).toBe(false)
    w.unmount()
  })
})
