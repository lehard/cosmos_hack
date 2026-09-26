// Очередь «Ждут моего решения»: точки предъявления, сигналы, изолированные
// изделия со сроком; порядок по риску или сроку считает сервер; выбор строки
// связывает виджеты стола; работа с клавиатуры (PRD §3a, FR-55).
import { flushPromises, mount } from '@vue/test-utils'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import { nonconformityKeys, useDecisionFocusStore } from '@/entities/nonconformity'
import { i18n } from '@/shared/i18n'
import { at } from '@/entities/item/__tests__/fixtures'
import { queueRows } from '@/entities/nonconformity/__tests__/fixtures'
import DecisionQueueView from '../ui/DecisionQueueView.vue'
import DecisionQueueWidget from '../ui/DecisionQueueWidget.vue'

let pinia: ReturnType<typeof createPinia>
beforeEach(() => {
  pinia = createPinia()
  setActivePinia(pinia)
})
const norm = (x: string) => x.replace(/\s/g, ' ')
const mountView = (props = {}) =>
  mount(DecisionQueueView, { props: { rows: queueRows(), sort: 'risk', now: Date.parse(at('11:23')), ...props }, global: { plugins: [pinia, i18n] } })

describe('очередь «Ждут моего решения»', () => {
  it('строки в порядке сервера: вид, изделие, тяжесть, срок', () => {
    const rows = mountView().findAll('li.row')
    expect(rows.map((r) => r.attributes('data-key'))).toEqual(['signal:SIG-77', 'isolated:NC-0139', 'presentation:PR-5'])
    expect(rows[0]!.text()).toContain('Сигнал на рассмотрение')
    expect(rows[0]!.text()).toContain('Тяжесть: Значительный')
    expect(norm(rows[0]!.find('[data-testid="due"]').text())).toBe('Осталось 37 мин')
    expect(rows[1]!.attributes('data-overdue')).toBe('true')
    expect(norm(rows[1]!.find('[data-testid="due"]').text())).toContain('Просрочено на 2 ч 23 мин')
    expect(rows[2]!.text()).toContain('Точка предъявления')
    expect(rows[2]!.text()).toContain('Предъявление № 2')
    // Тяжесть «неизвестна» — не «малозначительный».
    expect(rows[2]!.text()).toContain('Тяжесть: Неизвестно')
  })

  it('выбор мышью и с клавиатуры', async () => {
    const w = mountView({ selected: 'signal:SIG-77' })
    await w.findAll('li.row')[2]!.trigger('click')
    expect(w.emitted('select')?.[0]?.[0]).toMatchObject({ object_id: 'PR-5' })
    await w.find('ol.rows').trigger('keydown', { key: 'ArrowDown' })
    expect(w.emitted('select')?.[1]?.[0]).toMatchObject({ object_id: 'NC-0139' })
  })

  it('переключатель порядка: по риску / по сроку', async () => {
    const w = mountView()
    await w.find('[data-testid="sort-deadline"] input').setValue(true)
    expect(w.emitted('update:sort')?.[0]).toEqual(['deadline'])
  })

  it('контейнер: порядок уходит на сервер, первая строка выбрана для карточки и паспорта', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity } } })
    const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/', component: { template: '<div />' } }] })
    await router.push('/')
    queryClient.setQueryData(nonconformityKeys.list('decision-queue', { axis: 'occurred', sort: 'deadline' }), {
      data: { items: queueRows() },
      status: 200,
      headers: new Headers({ 'Ant-Backend': 'fixtures' }),
    })
    const w = mount(DecisionQueueWidget, {
      props: { widgetId: 'decision-queue', titleKey: 'desks.decisionQueue', slotId: 'queue', slice: { sort: ['deadline', 'risk'] }, density: 'comfortable' },
      global: { plugins: [pinia, i18n, router, [VueQueryPlugin, { queryClient }]] },
    })
    await flushPromises()
    expect(w.findAll('li.row')).toHaveLength(3)
    const focus = useDecisionFocusStore()
    expect(focus.rowId).toBe('signal:SIG-77')
    expect(focus.ncId).toBe('NC-0142')
    expect(focus.itemId).toBe('ENT:FL-0042')
  })
})
