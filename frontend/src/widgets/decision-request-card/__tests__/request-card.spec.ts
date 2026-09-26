// Карточка «Требуется ваше решение» для редких подписантов (FR-136): что
// предлагается, почему пришло, доказательства, похожие случаи, чьи подписи
// нужны и чьи есть, остаток срока, кнопка подписи; «не согласовать» — с
// замечанием. Операций documents в контракте нет — рамка честно в «ошибке входа».
import { flushPromises, mount } from '@vue/test-utils'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { i18n } from '@/shared/i18n'
import { at } from '@/entities/item/__tests__/fixtures'
import { useAsIsRequest } from '@/entities/document/__tests__/fixtures'
import DecisionRequestCardWidget from '../ui/DecisionRequestCardWidget.vue'
import DecisionRequestView from '../ui/DecisionRequestView.vue'

let pinia: ReturnType<typeof createPinia>
beforeEach(() => {
  pinia = createPinia()
  setActivePinia(pinia)
})
const norm = (x: string) => x.replace(/\s/g, ' ')
const mountView = (over = {}, props = {}) =>
  mount(DecisionRequestView, { props: { request: useAsIsRequest(over), now: Date.parse(at('13:30')), ...props }, global: { plugins: [pinia, i18n] } })

describe('карточка «Требуется ваше решение»', () => {
  it('что предлагается и почему пришло к вам (режим 5 — внешние полномочия)', () => {
    const w = mountView()
    expect(w.find('[data-testid="proposal"]').text()).toBe('Принять FL-0042 «как есть» по разрешению РО-12/26')
    expect(w.text()).toContain('Почему это пришло к вам')
    expect(w.text()).toContain('Внешние полномочия')
    expect(w.find('[data-testid="concession-note"]').text()).toBe('Итог: «годно по разрешению на отклонение» — это не «годно»')
  })

  it('доказательства: снимок и иллюстрация помечена явно', () => {
    const e = mountView().find('[data-testid="evidence"]')
    expect(e.text()).toContain('Результат контроля записан')
    expect(e.find('[data-illustration="true"]').text()).toBe('Иллюстрация из открытого набора. Не снимок этого изделия')
  })

  it('похожие случаи — принятые и отклонённые раздельно', () => {
    const w = mountView()
    expect(w.find('[data-testid="similar-accepted"]').text()).toContain('НС-0117')
    expect(w.find('[data-testid="similar-rejected"]').text()).toContain('НС-0098')
  })

  it('чьи подписи нужны и чьи уже есть; подпись по прежней версии не засчитана', () => {
    const r = mountView().find('[data-testid="route"]')
    expect(r.text()).toContain('подписей 1 из 2')
    expect(r.find('[data-stage="1"]').attributes('data-done')).toBe('true')
    expect(r.find('[data-stage="1"]').text()).toContain('Подписано')
    expect(r.find('[data-stage="1"]').text()).toContain('Подписи по прежней версии документа не засчитываются')
    expect(r.find('[data-stage="2"]').attributes('data-mine')).toBe('true')
    expect(r.find('[data-stage="2"]').text()).toContain('Ожидает подписи')
    expect(r.find('[data-stage="2"]').text()).toContain('представитель заказчика')
  })

  it('остаток срока и кнопка «Подписать — Как есть»', async () => {
    const w = mountView()
    expect(norm(w.find('[data-testid="remaining"]').text())).toBe('Осталось: 2 ч 30 мин')
    expect(w.find('[data-testid="sign"]').text()).toBe('Подписать — Как есть')
    await w.find('[data-testid="sign"]').trigger('click')
    expect(w.emitted('sign')).toHaveLength(1)
  })

  it('не ваш этап — подписать нельзя', () => {
    const w = mountView({ my_stage: null })
    expect(w.find('[data-testid="not-my-turn"]').exists()).toBe(true)
    expect(w.find('[data-testid="sign"]').attributes('disabled')).toBeDefined()
  })

  it('«Не согласовать» — только с замечанием', async () => {
    const w = mountView()
    await w.find('[data-testid="decline-open"]').trigger('click')
    expect(w.find('[data-testid="decline-send"]').attributes('disabled')).toBeDefined()
    await w.find('[data-testid="decline-comment"] textarea').setValue('Нужен протокол рентгена')
    await w.find('[data-testid="decline-send"]').trigger('click')
    expect(w.emitted('decline')?.[0]).toEqual(['Нужен протокол рентгена'])
  })

  it('контейнер: операций documents ещё нет — «ошибка входа», а не выдуманная карточка (FR-150)', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const w = mount(DecisionRequestCardWidget, {
      props: { widgetId: 'decision-request-card', titleKey: 'desks.decisionCard', slotId: 'card', slice: {}, density: 'comfortable' },
      global: { plugins: [pinia, i18n, [VueQueryPlugin, { queryClient }]] },
    })
    await vi.waitFor(() => expect(w.find('.widget-frame').attributes('data-state')).toBe('input_error'))
    await flushPromises()
    expect(w.find('[data-testid="decision-request"]').exists()).toBe(false)
  })
})
