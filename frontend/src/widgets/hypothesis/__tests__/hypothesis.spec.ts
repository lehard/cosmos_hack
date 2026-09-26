// Гипотезы с доводами «за / против» и кнопками решения (FR-59, FR-135),
// похожие случаи (FR-60).
import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { describe, expect, it } from 'vitest'
import { i18n } from '@/shared/i18n'
import { weldHypotheses } from '@/entities/incident/__tests__/fixtures'
import HypothesisView from '../ui/HypothesisView.vue'

const mountView = (props: Record<string, unknown> = {}) =>
  mount(HypothesisView, { props: { model: weldHypotheses(), ...props }, global: { plugins: [createPinia(), i18n] } })

describe('гипотезы причины', () => {
  it('порядок: открытые по уверенности, отклонённая — в конце и без кнопок', () => {
    const w = mountView()
    const cards = w.findAll('article.card')
    expect(cards.map((c) => c.attributes('data-category'))).toEqual(['equipment', 'performer', 'incoming'])
    expect(cards[2]!.attributes('data-status')).toBe('rejected')
    expect(cards[2]!.find('[data-testid="confirm"]').exists()).toBe(false)
  })

  it('доводы «за» и «против» — записи журнала в человекочитаемом виде', () => {
    const eq = mountView().find('article[data-category="equipment"]')
    const pro = eq.find('.arg[data-side="for"]').text()
    expect(pro).toContain('Доводы «за»')
    expect(pro).toContain('Вне уставки: ток 212 А при уставке 180 А')
    expect(pro).toContain('Ручное изменение режима: подача проволоки 130 %')
    expect(eq.find('.arg[data-side="against"]').text()).toContain('Доводов нет')
    expect(eq.text()).toContain('Уверенность вывода 0,72 — не вероятность вины')
    expect(eq.text()).toContain('Почему возник')
  })

  it('кнопки «подтвердить причину / отклонить / запросить измерение»', async () => {
    const w = mountView()
    const eq = w.find('article[data-category="equipment"]')
    expect(eq.find('[data-testid="request-measurement"]').text()).toBe('Запросить измерение — ток источника ИС-3 на эталонном образце')
    await eq.find('[data-testid="confirm"]').trigger('click')
    await eq.find('[data-testid="reject"]').trigger('click')
    await eq.find('[data-testid="request-measurement"]').trigger('click')
    expect(w.emitted('confirm')?.[0]?.[0]).toMatchObject({ hypothesis_id: 'h-equipment' })
    expect(w.emitted('reject')?.[0]?.[0]).toMatchObject({ hypothesis_id: 'h-equipment' })
    expect(w.emitted('request-measurement')?.[0]?.[0]).toMatchObject({ hypothesis_id: 'h-equipment' })
  })

  it('ошибка исполнителя — только после расследования и объяснения работника', () => {
    const perf = mountView().find('article[data-category="performer"]')
    expect(perf.find('[data-testid="performer-note"]').text()).toContain('письменного объяснения работника')
  })

  it('без доступных команд кнопки выключены', () => {
    const w = mountView({ canAct: false })
    for (const b of w.findAll('[data-testid="confirm"]')) expect(b.attributes('disabled')).toBeDefined()
  })

  it('довод по клику уходит на дорожки; вход из общего фактора подписан', async () => {
    const w = mountView({ fromFactor: 'Станок: Сварочный источник ИС-3' })
    expect(w.find('[data-testid="from-factor"]').text()).toContain('Станок: Сварочный источник ИС-3')
    await w.find('article[data-category="equipment"] .arg[data-side="for"] button').trigger('click')
    expect(w.emitted('select-record')?.[0]).toEqual(['e-current'])
  })

  it('при недостатке сведений категоричного вывода нет', () => {
    const w = mountView()
    expect(w.find('[data-testid="not-categorical"]').exists()).toBe(true)
    expect(w.text()).toContain('Неизвестен инструмент')
  })

  it('похожие случаи: причина, мера, результат', () => {
    const s = mountView().find('[data-testid="similar-cases"]').text()
    expect(s).toContain('НС-0117: причина (подтверждена) — Оборудование, мера — Замена кабеля массы ИС-3, результат — Результативно')
    expect(s).toContain('НС-0098: причина (гипотеза) — Исполнитель (отклонение от процедуры), мера — не назначена, результат — Неизвестно')
  })
})
