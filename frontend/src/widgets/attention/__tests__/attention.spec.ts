// «Требует вашего внимания» (FR-8): просроченное решение с ценой задержки —
// числом стоящих изделий и операций; меры; переход к объекту.
import { flushPromises, mount } from '@vue/test-utils'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import { notificationKeys, type AttentionEntry } from '@/entities/notification'
import type { Envelope } from '@/shared/api/response'
import { i18n } from '@/shared/i18n'
import AttentionWidget from '../ui/AttentionWidget.vue'

/** Образец — только для тестов. */
const entries = (): AttentionEntry[] => [
  { kind: 'overdue_decision', entry_id: 'a1', target: 'ЗТ-3, ФЛ-0001', overdue_minutes: 37, items: 18, operations: 2, ref: { entity: 'item', id: 'ENT:FL-0001' } },
  { kind: 'overdue_decision', entry_id: 'a2', target: 'НС-12', overdue_minutes: 75, items: 1, operations: 1, ref: { entity: 'nonconformity', id: 'NC-12' } },
  { kind: 'unverified_measures', entry_id: 'a3', n: 2 },
  { kind: 'temporary_measures', entry_id: 'a4', n: 1 },
]

async function mountWidget(seed: AttentionEntry[] | null) {
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
  if (seed) queryClient.setQueryData<Envelope<AttentionEntry[]>>(notificationKeys.list('attention', {}, { axis: 'occurred' }), { data: seed })
  const w = mount(AttentionWidget, {
    props: { widgetId: 'attention', titleKey: 'liveMap.attention.title', slotId: 'attention', slice: {}, density: 'comfortable' },
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

describe('виджет «Требует вашего внимания»', () => {
  it('просроченное решение — с числом стоящих изделий и операций (проверка FR-8)', async () => {
    const { w } = await mountWidget(entries())
    const items = w.findAll('li').map((li) => norm(li.text()))
    expect(items[0]).toBe('ЗТ-3, ФЛ-0001: просрочено на 37 мин — стоят 18 изделий, 2 операции')
    expect(items[1]).toBe('НС-12: просрочено на 1 ч 15 мин — стоят 1 изделие, 1 операция')
    expect(items[2]).toBe('Меры без подтверждённой результативности: 2')
    expect(items[3]).toBe('Временные меры, условие выхода не достигнуто: 1')
    expect(w.find('.widget-frame').attributes('data-state')).toBe('defect_indication')
  })

  it('переход к объекту; экрана ещё нет — строка без ссылки', async () => {
    const { w, router } = await mountWidget(entries())
    expect(w.find('[data-kind="overdue_decision"]:nth-child(2) .link').exists()).toBe(false)
    await w.find('.link').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.fullPath).toBe('/items/ENT:FL-0001')
  })

  it('ничего не требует внимания — пусто, норма', async () => {
    const { w } = await mountWidget([])
    expect(w.find('.widget-frame').attributes('data-state')).toBe('normal')
    expect(w.text()).toContain('Записей нет')
  })

  it('сервер ответил ошибкой — «ошибка входа»', async () => {
    serverFails()
    const { w } = await mountWidget(null)
    await vi.waitFor(() => expect(w.find('.widget-frame').attributes('data-state')).toBe('input_error'))
  })
})
