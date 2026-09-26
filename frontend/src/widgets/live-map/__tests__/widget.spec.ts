// Контейнер живой карты: читает сервер только через entities/live-map, четыре
// состояния рамки, метка режима, инцидент из адреса, переход в паспорт (AD-21, FR-7, FR-150).
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter, type Router } from 'vue-router'
import { liveMapKeys, type LiveMapData, type LiveMapParams } from '@/entities/live-map'
import type { Envelope } from '@/shared/api/pending'
import { i18n } from '@/shared/i18n'
import LiveMapWidget from '../ui/LiveMapWidget.vue'
import { frameMorning, frameScope34 } from './fixtures'
import { installSvgStubs } from './svg-env'

installSvgStubs()

let pinia: ReturnType<typeof createPinia>
let queryClient: QueryClient
let router: Router
let wrapper: VueWrapper | null = null

beforeEach(async () => {
  pinia = createPinia()
  setActivePinia(pinia)
  queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity } } })
  router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/desk', name: 'desk', component: { template: '<div />' } },
      { path: '/items/:id', name: 'item', component: { template: '<div />' } },
    ],
  })
  await router.push('/desk')
})
afterEach(() => {
  wrapper?.unmount()
  wrapper = null
  document.body.innerHTML = ''
})

const seed = (params: LiveMapParams, data: LiveMapData, mode = 'fixtures') =>
  queryClient.setQueryData<Envelope<LiveMapData>>(liveMapKeys.list('view', params, { axis: 'occurred' }), {
    data,
    headers: new Headers({ 'Ant-Backend': mode }),
  })

function mountWidget(slice: Record<string, unknown> = { period: 'shift' }) {
  wrapper = mount(LiveMapWidget, {
    props: { widgetId: 'live-map', titleKey: 'liveMap.title', slotId: 'map', slice, density: 'comfortable' },
    attachTo: document.body,
    global: { plugins: [pinia, i18n, router, [VueQueryPlugin, { queryClient }]] },
  })
  return wrapper
}

describe('виджет «Живая карта»', () => {
  it('операции ещё нет — «ошибка входа», а не выдуманные данные (FR-150)', async () => {
    const w = mountWidget()
    await vi.waitFor(() => expect(w.find('.widget-frame').attributes('data-state')).toBe('input_error'))
    expect(w.find('.live-map').exists()).toBe(false)
  })

  it('данные из кэша запроса: карта, метка режима, признак дефекта', async () => {
    seed({ period: 'shift' }, frameScope34())
    const w = mountWidget()
    await flushPromises()
    const frame = w.find('.widget-frame')
    expect(frame.attributes('data-state')).toBe('defect_indication')
    expect(frame.attributes('data-mode')).toBe('fixtures')
    expect(frame.text()).toContain('Демо на заготовках')
    await vi.waitFor(() => expect(document.querySelector('.dot[data-item="ENT:FL-0041"]')).not.toBeNull())
  })

  it('инцидент из адреса страницы попадает в запрос (FR-9)', async () => {
    await router.push('/desk?incident=INC-1')
    seed({ period: 'shift', incident_id: 'INC-1' }, frameScope34())
    const w = mountWidget()
    await flushPromises()
    expect(w.find('[data-testid="incident"]').exists()).toBe(true)
  })

  it('клик по точке — страница паспорта изделия (FR-2)', async () => {
    seed({ period: 'day' }, frameMorning())
    mountWidget({ period: 'day' })
    await vi.waitFor(() => expect(document.querySelector('.dot[data-item="ENT:FL-0041"]')).not.toBeNull())
    document.querySelector<HTMLElement>('.dot[data-item="ENT:FL-0041"]')!.click()
    await flushPromises()
    expect(router.currentRoute.value.fullPath).toBe('/items/ENT:FL-0041')
  })
})
