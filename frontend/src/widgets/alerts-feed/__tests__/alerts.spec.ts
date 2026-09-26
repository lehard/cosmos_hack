// Лента тревог (FR-8): тексты по виду тревоги, цена задержки у эскалации,
// переход к объекту тревоги (FR-7), неизвестный вид аномалии — UNKNOWN(код).
import { flushPromises, mount } from '@vue/test-utils'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import { notificationKeys, type AlertEntry } from '@/entities/notification'
import type { Envelope } from '@/shared/api/response'
import { i18n } from '@/shared/i18n'
import AlertsFeedWidget from '../ui/AlertsFeedWidget.vue'

/** Образец — только для тестов. */
const alerts = (): AlertEntry[] => [
  { alert_id: 'x1', at: '2026-09-23T08:05:00.000Z', kind: 'overdue_isolation', item: 'ФЛ-0001', ref: { entity: 'item', id: 'ENT:FL-0001' } },
  { alert_id: 'x2', at: '2026-09-23T08:10:00.000Z', kind: 'gate_overdue', gate: 'ЗТ-3' },
  { alert_id: 'x3', at: '2026-09-23T08:11:00.000Z', kind: 'not_moved_to_isolator', item: 'ФЛ-0002' },
  { alert_id: 'x4', at: '2026-09-23T08:12:00.000Z', kind: 'anomaly', node: 'ЗТ-3 Приёмка ОТК', anomaly: 'queue_above_norm', ref: { entity: 'live_map', id: 'welding.zt3_acceptance' } },
  { alert_id: 'x5', at: '2026-09-23T08:13:00.000Z', kind: 'escalation', target: 'ЗТ-3', overdue_minutes: 37, items: 18, operations: 2 },
  { alert_id: 'x6', at: '2026-09-23T08:14:00.000Z', kind: 'integrity_violation', ref: { entity: 'integrity', id: 'main' } },
  { alert_id: 'x7', at: '2026-09-23T08:15:00.000Z', kind: 'anomaly', node: 'КТ-3', anomaly: 'mystery' },
]

async function mountWidget(seed: AlertEntry[] | null) {
  const pinia = createPinia()
  setActivePinia(pinia)
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity } } })
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/desk', component: { template: '<div />' } },
      { path: '/items/:id', name: 'item', component: { template: '<div />' } },
    ],
  })
  await router.push('/desk')
  if (seed) queryClient.setQueryData<Envelope<AlertEntry[]>>(notificationKeys.list('alerts', {}, { axis: 'occurred' }), { data: seed })
  const w = mount(AlertsFeedWidget, {
    props: { widgetId: 'alerts-feed', titleKey: 'liveMap.alerts.title', slotId: 'alerts', slice: {}, density: 'comfortable' },
    global: { plugins: [pinia, i18n, router, [VueQueryPlugin, { queryClient }]] },
  })
  await flushPromises()
  return { w, router }
}

const norm = (s: string) => s.replace(/\u00a0/g, ' ')


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

afterEach(() => vi.unstubAllGlobals())

describe('виджет «Тревоги»', () => {
  it('тексты по виду тревоги', async () => {
    const { w } = await mountWidget(alerts())
    const t = (id: string) => norm(w.findAll('li')[Number(id.slice(1)) - 1]!.text())
    expect(t('x1')).toContain('Просрочена изоляция: ФЛ-0001')
    expect(t('x2')).toContain('Истёк срок ожидания на точке предъявления ЗТ-3')
    expect(t('x3')).toContain('Изолировано в системе, физически не перемещено: ФЛ-0002')
    expect(t('x4')).toContain('Аномалия узла ЗТ-3 Приёмка ОТК: Очередь выше нормы узла')
    expect(t('x5')).toContain('Эскалация: ЗТ-3: просрочено на 37 мин — стоят 18 изделий, 2 операции')
    expect(t('x6')).toContain('Нарушена целостность журнала')
    expect(t('x7')).toContain('UNKNOWN(mystery)')
    expect(w.find('.widget-frame').attributes('data-state')).toBe('defect_indication')
  })

  it('по тревоге — к объекту тревоги, если его экран есть (FR-7)', async () => {
    const { w, router } = await mountWidget(alerts())
    expect(w.findAll('.link')).toHaveLength(1)
    await w.find('.link').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.fullPath).toBe('/items/ENT:FL-0001')
  })

  it('сервер ответил ошибкой — «ошибка входа»', async () => {
    serverFails()
    const { w } = await mountWidget(null)
    await vi.waitFor(() => expect(w.find('.widget-frame').attributes('data-state')).toBe('input_error'))
  })
})
