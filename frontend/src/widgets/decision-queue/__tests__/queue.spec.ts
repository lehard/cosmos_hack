// Очередь «Ждут моего решения» (UI-25): задачи по группам — сигналы, изолированные
// изделия со сроком, точки предъявления; порядок по риску или сроку считает сервер; выбор строки
// открывает запись в правом окне (Д-70); работа с клавиатуры (PRD §3a, FR-55).
import { flushPromises, mount } from '@vue/test-utils'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import { nonconformityKeys } from '@/entities/nonconformity'
import { RECORD_DRAWER } from '@/shared/model/record'
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
  it('задачи, а не записи: группы по тому, что нужно сделать, со счётчиком; строки в порядке сервера', async () => {
    const w = mountView()
    const groups = w.findAll('section.group')
    expect(groups.map((g) => g.find('.group-title').text())).toEqual([
      expect.stringContaining('Подтвердить или отклонить сигнал'),
      expect.stringContaining('Решить, что делать с изделием'),
      expect.stringContaining('Плановая приёмка на точках предъявления'),
    ])
    expect(groups.map((g) => g.find('[data-testid="group-count"]').text())).toEqual(['1', '1', '1'])
    expect(groups[1]!.find('[data-testid="group-overdue"]').text()).toBe('просрочено: 1')
    // Плановая приёмка свёрнута, пока её не раскрыли.
    expect(w.findAll('li.row')).toHaveLength(2)
    await w.find('[data-testid="toggle-routine"]').trigger('click')
    const rows = w.findAll('li.row')
    expect(rows.map((r) => r.attributes('data-key'))).toEqual(['signal:SIG-77', 'isolated:NC-0139', 'presentation:PR-5'])
    expect(rows[0]!.text()).toContain('FL-0042')
    expect(rows[0]!.text()).toContain('Пора в шве · КТ-3')
    expect(rows[0]!.text()).toContain('Тяжесть: Значительный')
    expect(norm(rows[0]!.find('[data-testid="due"]').text())).toBe('Осталось 37 мин')
    expect(rows[1]!.attributes('data-overdue')).toBe('true')
    expect(norm(rows[1]!.find('[data-testid="due"]').text())).toContain('Просрочено на 2 ч 23 мин')
    expect(rows[2]!.text()).toContain('Предъявление № 2')
    // Тяжесть «неизвестна» — не «малозначительный».
    expect(rows[2]!.text()).toContain('Тяжесть: Неизвестно')
  })

  it('группы идут в порядке первой своей строки: порядок сервера не теряется', async () => {
    const [a, b, c] = queueRows()
    const w = mountView({ rows: [c!, a!, { ...b!, kind: 'presentation', object_id: 'PR-6' }] })
    expect(w.findAll('section.group').map((g) => g.attributes('data-group'))).toEqual(['presentation', 'signal'])
    await w.find('[data-testid="toggle-routine"]').trigger('click')
    expect(w.findAll('li.row').map((r) => r.attributes('data-key'))).toEqual(['presentation:PR-5', 'presentation:PR-6', 'signal:SIG-77'])
  })

  it('пересмотр после новых данных — первой группой; в строке суть (reason), а не общий заголовок', () => {
    const [a, b, c] = queueRows()
    const review = { ...c!, kind: 'review' as const, object_id: 'REV-1', title: 'Решение ЗТ-3 принято до новых данных — пересмотрите: пришёл журнал', review_since: at('10:20'), reason: 'Ток выше уставки' }
    const w = mountView({ rows: [a!, { ...b!, reason: 'Трещина · кромка' }, review] })
    expect(w.findAll('section.group').map((g) => g.attributes('data-group'))).toEqual(['review', 'signal', 'isolated'])
    expect(w.find('section.group .group-title').text()).toContain('Пересмотреть решение — пришли новые данные')
    const rows = w.findAll('li.row')
    expect(rows[0]!.find('[data-testid="row-title"]').text()).toBe('Ток выше уставки')
    expect(rows[0]!.find('[data-testid="review-since"]').text()).toContain('новые данные с')
    expect(rows[0]!.text()).toContain('пришёл журнал')
    expect(rows[2]!.find('[data-testid="row-title"]').text()).toBe('Трещина · кромка')
  })

  it('сначала исключения: сводка сверху, пересмотр — крупным блоком, открытая строка приёмки раскрывает группу', () => {
    const [a, b, c] = queueRows()
    const review = { ...c!, kind: 'review' as const, object_id: 'REV-1', title: 'Решение ЗТ-3 принято до новых данных — пересмотрите', review_since: at('10:20') }
    const w = mountView({ rows: [a!, b!, c!, review] })
    const sum = norm(w.find('[data-testid="queue-summary"]').text())
    expect(sum).toContain('1 решение на пересмотр')
    expect(sum).toContain('2 по отклонениям')
    expect(sum).toContain('1 плановая приёмка')
    const hero = w.find('li.row.hero')
    expect(hero.attributes('data-key')).toBe('review:REV-1')
    expect(hero.text()).toContain('Изменились данные после принятого решения')
    expect(hero.text()).toContain('Пересмотреть')
    expect(mountView({ selected: 'presentation:PR-5' }).find('[data-key="presentation:PR-5"]').exists()).toBe(true)
  })

  it('выбор мышью и с клавиатуры', async () => {
    const w = mountView({ selected: 'signal:SIG-77' })
    await w.find('[data-testid="toggle-routine"]').trigger('click')
    await w.findAll('li.row')[2]!.trigger('click')
    expect(w.emitted('select')?.[0]?.[0]).toMatchObject({ object_id: 'PR-5' })
    await w.find('[data-testid="queue-groups"]').trigger('keydown', { key: 'ArrowDown' })
    expect(w.emitted('select')?.[1]?.[0]).toMatchObject({ object_id: 'NC-0139' })
  })

  it('переключатель порядка: по риску / по сроку', async () => {
    const w = mountView()
    await w.find('[data-testid="sort-deadline"] input').setValue(true)
    expect(w.emitted('update:sort')?.[0]).toEqual(['deadline'])
  })

  it('контейнер: порядок уходит на сервер; щелчок открывает окно записи в адресе, строка выделена', async () => {
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
      global: {
        plugins: [pinia, i18n, router, [VueQueryPlugin, { queryClient }]],
        provide: { [RECORD_DRAWER as symbol]: { kinds: new Set(['nonconformity', 'item', 'presentation']) } },
      },
    })
    await flushPromises()
    const rows = () => w.findAll('li.row')
    expect(rows()).toHaveLength(2)
    await w.find('[data-testid="toggle-routine"]').trigger('click')
    expect(rows()).toHaveLength(3)
    // Ничего не открывается само: окно — только по щелчку.
    expect(router.currentRoute.value.query.open).toBeUndefined()
    expect(rows().some((r) => r.attributes('aria-selected') === 'true')).toBe(false)
    await rows()[0]!.trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.query.open).toBe('nonconformity:NC-0142')
    expect(rows()[0]!.attributes('aria-selected')).toBe('true')
    // Точка предъявления без несоответствия — окно решения на точке, а не паспорт (UI-28).
    await rows()[2]!.trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.query.open).toBe('presentation:ENT:FL-0031')
    expect(rows()[2]!.attributes('aria-selected')).toBe('true')
  })
})
