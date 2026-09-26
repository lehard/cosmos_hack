// Контейнер живой карты: читает сервер только через entities/live-map, четыре
// состояния рамки, метка режима, инцидент из адреса, переход в паспорт (AD-21, FR-7, FR-150).
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter, type Router } from 'vue-router'
import { liveMapKeys, type LiveMapData, type LiveMapParams } from '@/entities/live-map'
import type { Envelope } from '@/shared/api/response'
import { i18n } from '@/shared/i18n'
import { useMomentStore } from '@/shared/model/moment'
import LiveMapWidget from '../ui/LiveMapWidget.vue'
import { frameMorning, frameScope34 } from './fixtures'
import { installSvgStubs } from './svg-env'

installSvgStubs()
// bpmn-js открывает схему на 300 узлов — под общей нагрузкой прогона это секунды.
vi.setConfig({ testTimeout: 30_000 })
const WAIT = { timeout: 15_000 }

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
  vi.unstubAllGlobals()
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


/** Сервер отвечает problem+json с кодом (сгенерированный клиент бросает ошибку с info). */
const serverFails = () =>
  vi.stubGlobal(
    'fetch',
    vi.fn(async () =>
      new Response(JSON.stringify({ type: 'urn:ant:problem:api.not_implemented', title: 'Операция ещё не реализована', status: 501, code: 'api.not_implemented' }), {
        status: 501,
        headers: { 'Content-Type': 'application/problem+json' },
      }),
    ),
  )

describe('виджет «Живая карта»', () => {
  it('сервер ответил ошибкой — «ошибка входа», а не выдуманные данные (FR-150)', async () => {
    serverFails()
    const w = mountWidget()
    await vi.waitFor(() => expect(w.find('.widget-frame').attributes('data-state')).toBe('input_error'), WAIT)
    expect(w.find('.live-map').exists()).toBe(false)
  })

  it('данные из кэша запроса: карта, метка режима, признак дефекта', async () => {
    seed({ period: 'shift' }, frameScope34())
    const w = mountWidget()
    await flushPromises()
    const frame = w.find('.widget-frame')
    expect(frame.attributes('data-state')).toBe('defect_indication')
    expect(frame.attributes('data-mode')).toBe('fixtures')
    // Д-70: режим — атрибутом рамки, меткой в заголовке панели не пишется.
    expect(frame.text()).not.toContain('Демо на заготовках')
    await vi.waitFor(() => expect(document.querySelector('.dot[data-item="ENT:FL-0041"]')).not.toBeNull(), WAIT)
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
    await vi.waitFor(() => expect(document.querySelector('.dot[data-item="ENT:FL-0041"]')).not.toBeNull(), WAIT)
    document.querySelector<HTMLElement>('.dot[data-item="ENT:FL-0041"]')!.click()
    await flushPromises()
    expect(router.currentRoute.value.fullPath).toBe('/items/ENT:FL-0041')
  })
  it('через сгенерированный клиент: параметры и момент в запросе, метка режима из заголовка (AD-21, AD-22)', async () => {
    const urls: string[] = []
    vi.stubGlobal(
      'fetch',
      vi.fn(async (url: string) => {
        urls.push(url)
        return new Response(JSON.stringify(frameScope34()), { status: 200, headers: { 'Content-Type': 'application/json', 'Ant-Backend': 'live' } })
      }),
    )
    await router.push('/desk?incident=INC-1&run=RUN-7')
    const w = mountWidget()
    await vi.waitFor(() => expect(w.find('.widget-frame').attributes('data-mode')).toBe('live'), WAIT)
    expect(urls[0]).toBe('/api/v1/live-map?period=shift&incident_id=INC-1&run_id=RUN-7&axis=occurred')
    await vi.waitFor(() => expect(document.querySelector('.dot[data-item="ENT:FL-0001"]')).not.toBeNull(), WAIT)
    // Таймлайн сдвинул момент — та же операция на момент (воспроизведение).
    useMomentStore().travel('2026-09-23T11:05:00.000Z')
    await vi.waitFor(() => expect(urls.at(-1)).toContain('as_of=2026-09-23T11%3A05%3A00.000Z'), WAIT)
  })

  it('без ограничения и инцидента в ответе — карта без них', async () => {
    const f = frameMorning()
    delete f.bottleneck
    seed({ period: 'shift' }, f)
    mountWidget()
    await vi.waitFor(() => expect(document.querySelector('.node-badge[data-step="welding.weld"]')).not.toBeNull(), WAIT)
    expect(document.querySelector('[data-flag="bottleneck"]')).toBeNull()
    expect(document.querySelector('[data-testid="incident"]')).toBeNull()
  })
})
