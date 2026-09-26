// Окно записи оболочки (Д-70): открывается по адресу `?open=‹тип›:‹id›` на
// любой странице (ссылка из уведомления, поиска), неизвестный тип — окна нет,
// закрытие убирает параметр из адреса.
import { defineComponent, h, provide } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { createPinia } from 'pinia'
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import { i18n } from '@/shared/i18n'
import { RECORD_DRAWER } from '@/shared/model/record'
import { flangePassport } from '@/entities/item/__tests__/fixtures'
import RecordDrawerHost from '../RecordDrawerHost.vue'
import { RECORD_KINDS, recordKinds } from '../registry'

const $ = (sel: string) => document.querySelector<HTMLElement>(sel)
const until = (check: () => void) => vi.waitFor(check, { timeout: 10_000 })
const json = (body: unknown) => new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json', 'Ant-Backend': 'fixtures' } })
const fetchMock = vi.fn(async (url: string) => {
  if (String(url).startsWith('/api/v1/items/ENT:FL-0042/passport')) return json(flangePassport())
  return new Response(JSON.stringify({ code: 'api.not_found', title: 'нет', status: 404 }), { status: 404 })
})
// Первый импорт содержимого окна (с виджетами) в тестах долгий — грузим заранее.
beforeAll(async () => {
  await Promise.all([...Object.values(recordKinds).map((k) => k.load()), import('@/widgets/item-passport')])
}, 60_000)
beforeEach(() => vi.stubGlobal('fetch', fetchMock))
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
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/desk', component: { template: '<div />' } },
      { path: '/items/:id', name: 'item', component: { template: '<div />' } },
    ],
  })
  await router.push(path)
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const w = mount(Shell, { attachTo: document.body, global: { plugins: [createPinia(), i18n, router, [VueQueryPlugin, { queryClient }]] } })
  await flushPromises()
  return { w, router }
}

describe('окно записи оболочки', () => {
  it('ссылка с ?open=item:… открывает окно изделия с паспортом; крестик убирает параметр', async () => {
    const { w, router } = await mountAt('/desk?open=item:ENT:FL-0042')
    await until(() => expect($('[data-record="item"] [data-testid="item-passport"]')).not.toBeNull())
    expect($('[data-testid="record-drawer-head"]')!.textContent).toContain('FL-0042')

    $('[data-record="item"] .n-base-close')!.click()
    await until(() => expect(router.currentRoute.value.fullPath).toBe('/desk'))
    w.unmount()
  })

  it('«Открыть на отдельной странице» — страница паспорта без окна', async () => {
    const { w, router } = await mountAt('/desk?open=item:ENT:FL-0042')
    await until(() => expect($('[data-testid="open-item-page"]')).not.toBeNull())
    $('[data-testid="open-item-page"]')!.click()
    await until(() => expect(router.currentRoute.value.name).toBe('item'))
    expect(router.currentRoute.value.query.open).toBeUndefined()
    w.unmount()
  })

  it('неизвестный тип и испорченный адрес — окна нет', async () => {
    for (const path of ['/desk?open=alarm:A-1', '/desk?open=item', '/desk']) {
      const { w } = await mountAt(path)
      expect($('[data-record]')).toBeNull()
      w.unmount()
    }
  })
})
