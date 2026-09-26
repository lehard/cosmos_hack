// Таймлайн и воспроизведение (FR-4, FR-155; AD-22): движение момента ×1…×1000,
// пауза, переход к моменту и к метке, возврат к «сейчас», пуск прогона с начала.
import { flushPromises, mount } from '@vue/test-utils'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { effectScope } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import { liveMapKeys, type TimelineData } from '@/entities/live-map'
import type { Envelope } from '@/shared/api/response'
import { i18n } from '@/shared/i18n'
import { useMomentStore } from '@/shared/model/moment'
import { advance, toMoment, usePlayback } from '@/features/playback'
import MapTimelineWidget from '../ui/MapTimelineWidget.vue'

const FROM = Date.parse('2026-09-23T06:00:00.000Z')
const TO = Date.parse('2026-09-23T14:00:00.000Z')

/** Образец таймлайна главной истории — только для тестов. */
const timeline = (): TimelineData => ({
  from: toMoment(FROM),
  to: toMoment(TO),
  marks: [
    { mark_id: 'm1', at: '2026-09-23T10:41:00.000Z', kind: 'spike', title: 'КТ-3' },
    { mark_id: 'm2', at: '2026-09-23T11:05:00.000Z', kind: 'escalation', title: 'ЗТ-3', ref: { entity: 'item', id: 'ENT:FL-0001' } },
    { mark_id: 'm3', at: '2026-09-23T11:30:00.000Z', kind: 'process_stop' },
    { mark_id: 'm4', at: '2026-09-23T12:10:00.000Z', kind: 'revision' },
    { mark_id: 'out', at: '2026-09-22T12:10:00.000Z', kind: 'revision' },
  ],
})

beforeEach(() => {
  setActivePinia(createPinia())
  vi.useFakeTimers()
})
afterEach(() => {
  vi.unstubAllGlobals()
  vi.useRealTimers()
  document.body.innerHTML = ''
})


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

describe('модель воспроизведения', () => {
  it('шаг времени и конец истории', () => {
    expect(advance(0, 1000, 1000, 10_000_000)).toEqual({ at: 1_000_000, ended: false })
    expect(advance(9_900_000, 1000, 1000, 10_000_000)).toEqual({ at: 10_000_000, ended: true })
    expect(toMoment(FROM)).toBe('2026-09-23T06:00:00.000Z')
  })

  it('пуск из «сейчас» — с начала истории; ×1000 за секунду — 16 мин 40 с; пауза; дошли до конца — «сейчас»', () => {
    const scope = effectScope()
    const pb = scope.run(() => usePlayback(() => ({ from: FROM, to: TO })))!
    const moment = useMomentStore()
    pb.speed.value = 1000
    pb.play()
    expect(moment.asOf).toBe(toMoment(FROM))
    expect(moment.isReplay).toBe(true)
    vi.advanceTimersByTime(1000)
    expect(moment.asOf).toBe(toMoment(FROM + 1_000_000))
    pb.pause()
    vi.advanceTimersByTime(5000)
    expect(moment.asOf).toBe(toMoment(FROM + 1_000_000))
    pb.play()
    vi.advanceTimersByTime(60_000)
    expect(moment.asOf).toBeNull()
    expect(pb.playing.value).toBe(false)
    scope.stop()
  })

  it('переход к моменту ставит паузу; «сейчас» — живой режим', () => {
    const scope = effectScope()
    const pb = scope.run(() => usePlayback(() => ({ from: FROM, to: TO })))!
    const moment = useMomentStore()
    pb.play()
    pb.jump('2026-09-23T11:05:00.000Z')
    expect(pb.playing.value).toBe(false)
    expect(moment.asOf).toBe('2026-09-23T11:05:00.000Z')
    pb.goLive()
    expect(moment.asOf).toBeNull()
    scope.stop()
  })

  it('без диапазона пуск невозможен', () => {
    const scope = effectScope()
    const pb = scope.run(() => usePlayback(() => null))!
    pb.play()
    expect(pb.playing.value).toBe(false)
    expect(useMomentStore().asOf).toBeNull()
    scope.stop()
  })
})

describe('виджет «Таймлайн»', () => {
  async function mountWidget(seed: TimelineData | null, route = '/desk') {
    const pinia = createPinia()
    setActivePinia(pinia)
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity } } })
    const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/desk', component: { template: '<div />' } }] })
    await router.push(route)
    if (seed) {
      // Смена оси — новый ключ и настоящий запрос: сервер отдаёт тот же таймлайн.
      vi.stubGlobal('fetch', vi.fn(async () => new Response(JSON.stringify(seed), { status: 200, headers: { 'Content-Type': 'application/json' } })))
      const params = router.currentRoute.value.query.run ? { run_id: String(router.currentRoute.value.query.run) } : {}
      queryClient.setQueryData<Envelope<TimelineData>>(liveMapKeys.list('timeline', params, { axis: 'occurred' }), { data: seed })
    }
    const w = mount(MapTimelineWidget, {
      props: { widgetId: 'map-timeline', titleKey: 'widgets.mapTimeline', slotId: 'timeline', slice: {}, density: 'comfortable' },
      attachTo: document.body,
      global: { plugins: [pinia, i18n, router, [VueQueryPlugin, { queryClient }]] },
    })
    await flushPromises()
    return { w, moment: useMomentStore() }
  }

  it('метки в пределах истории; клик по метке — переход к её моменту (FR-4)', async () => {
    const { w, moment } = await mountWidget(timeline())
    expect(w.findAll('[data-mark]').map((m) => m.attributes('data-kind'))).toEqual(['spike', 'escalation', 'process_stop', 'revision'])
    expect(w.find('[data-mark="m2"]').attributes('title')).toContain('Эскалация: ЗТ-3')
    await w.find('[data-mark="m2"]').trigger('click')
    expect(moment.asOf).toBe('2026-09-23T11:05:00.000Z')
    await flushPromises()
    expect(w.find('[data-testid="replay-note"]').text()).toContain('ничего не меняет')
    expect(w.find('.widget-frame').text()).toContain('Как было')
  })

  it('воспроизведение, скорость, пауза и «Сейчас» из кнопок', async () => {
    const { w, moment } = await mountWidget(timeline(), '/desk?run=RUN-7')
    await w.find('[data-speed="1000"] input').setValue(true)
    await w.find('[data-action="play"]').trigger('click')
    expect(moment.asOf).toBe(toMoment(FROM))
    vi.advanceTimersByTime(2000)
    expect(moment.asOf).toBe(toMoment(FROM + 2_000_000))
    await flushPromises()
    await w.find('[data-action="pause"]').trigger('click')
    await flushPromises()
    expect(w.find('[data-action="play"]').text()).toContain('Продолжить')
    await w.find('[data-action="live"]').trigger('click')
    expect(moment.asOf).toBeNull()
  })

  it('ось «что мы знали» (AD-37)', async () => {
    const { w, moment } = await mountWidget(timeline())
    await w.find('[data-mark="m1"]').trigger('click')
    await w.find('[data-axis="recorded"] input').setValue(true)
    expect(moment.axis).toBe('recorded')
    expect(moment.asOf).toBe('2026-09-23T10:41:00.000Z')
  })

  it('сервер ответил ошибкой — «ошибка входа», но «Сейчас» и переход к моменту остаются', async () => {
    serverFails()
    const { w } = await mountWidget(null)
    await vi.waitFor(() => expect(w.find('.widget-frame').attributes('data-state')).toBe('input_error'))
    expect(w.find('[data-action="live"]').exists()).toBe(true)
    expect(w.find('[data-action="play"]').attributes('disabled')).toBeDefined()
  })
})
