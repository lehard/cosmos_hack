// Гипотезы с доводами «за / против» и кнопками решения (FR-59, FR-135),
// похожие случаи (FR-60).
import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { i18n } from '@/shared/i18n'
import { mockApi, mountWidget, permissions, settle } from '@/entities/incident/__tests__/api-mock'
import { ncGroups, weldHypotheses } from '@/entities/incident/__tests__/fixtures'
import HypothesisView from '../ui/HypothesisView.vue'
import HypothesisWidget from '../ui/HypothesisWidget.vue'

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
    expect(mountView().find('[data-branch="why_made"]').find('article[data-category="equipment"]').exists()).toBe(true)
  })

  it('кнопки «подтвердить причину / отклонить / запросить измерение» открывают форму с обязательными полями', async () => {
    const w = mountView()
    const eq = () => w.find('article[data-category="equipment"]')
    expect(eq().find('[data-testid="next-check"]').text()).toContain('Что проверить следующим')
    expect(eq().find('[data-testid="next-check"]').text()).toContain('ток источника ИС-3 на эталонном образце')
    expect(eq().find('[data-testid="request-measurement"]').text()).toBe('Запросить проверку')

    await eq().find('[data-testid="confirm"]').trigger('click')
    expect(eq().find('[data-testid="form-submit"]').attributes('disabled')).toBeDefined()
    await eq().find('[data-testid="form-first"]').setValue('Эталонный образец: ток 212 А воспроизводит прожог')
    await eq().find('[data-testid="form-second"]').setValue('Отклонение режима подтверждено измерением')
    await eq().find('form').trigger('submit')
    expect(w.emitted('confirm')?.[0]?.[0]).toMatchObject({ hypothesis_id: 'h-equipment' })
    expect(w.emitted('confirm')?.[0]?.[1]).toEqual({ verification: 'Эталонный образец: ток 212 А воспроизводит прожог', reason: 'Отклонение режима подтверждено измерением' })

    await eq().find('[data-testid="reject"]').trigger('click')
    await eq().find('[data-testid="form-first"]').setValue('Журнал станка не подтверждает')
    await eq().find('form').trigger('submit')
    expect(w.emitted('reject')?.[0]?.[1]).toEqual({ reason: 'Журнал станка не подтверждает' })

    await eq().find('[data-testid="request-measurement"]').trigger('click')
    expect((eq().find('[data-testid="form-first"]').element as HTMLTextAreaElement).value).toBe('ток источника ИС-3 на эталонном образце')
    await eq().find('form').trigger('submit')
    expect(w.emitted('request-measurement')?.[0]?.[1]).toEqual({ what: 'ток источника ИС-3 на эталонном образце' })
  })

  it('две причины: почему возник и почему не остановили; пустая ветка пропуска — так и сказано', () => {
    const w = mountView()
    const branches = w.findAll('[data-testid="branch"]')
    expect(branches.map((b) => b.attributes('data-branch'))).toEqual(['why_made', 'why_missed'])
    expect(branches[0]!.text()).toContain('Почему возник дефект')
    expect(branches[0]!.findAll('article.card')).toHaveLength(3)
    expect(branches[1]!.text()).toContain('Почему контроль не остановил его раньше')
    expect(branches[1]!.find('[data-testid="branch-empty"]').text()).toContain('Причина пропуска ещё не разобрана')
  })

  it('гипотеза пропуска — во второй ветке; без подсказки проверки — кнопка измерения в действиях', () => {
    const m = weldHypotheses()
    m.hypotheses.push({ ...m.hypotheses[2]!, hypothesis_id: 'h-miss', category: 'documentation', branch: 'why_missed', status: 'recorded', statement: 'КТ-2: камера не видит зону У2', measurement_hint: null })
    const miss = mount(HypothesisView, { props: { model: m }, global: { plugins: [createPinia(), i18n] } }).find('[data-branch="why_missed"]')
    expect(miss.find('[data-testid="branch-empty"]').exists()).toBe(false)
    expect(miss.find('.statement').text()).toBe('КТ-2: камера не видит зону У2')
    expect(miss.find('[data-testid="next-check"]').exists()).toBe(false)
    expect(miss.find('[data-testid="request-measurement"]').exists()).toBe(true)
  })

  it('«что проверить следующим» от сервера: сколько изделий может исключить; «что меняло уверенность»', () => {
    const m = weldHypotheses()
    m.hypotheses[2] = {
      ...m.hypotheses[2]!,
      next_check: { text: 'Контрольный образец на ИС-2', measurement_kind: 'control_sample', unlocks_text: 'Подтвердит причину и позволит сузить область', could_exclude: 7, scope_size: 13 },
      history: [{ at: '2026-09-23T09:15:00Z', confidence_bp: 8500, event_id: 'e1', text: 'Пришёл журнал ИС-2: совпадение 3 из 3' }],
    }
    const eq = mount(HypothesisView, { props: { model: m }, global: { plugins: [createPinia(), i18n] } }).find('article[data-category="equipment"]')
    expect(eq.find('[data-testid="next-check"]').text()).toContain('Контрольный образец на ИС-2')
    expect(eq.find('[data-testid="next-gain"]').text()).toBe('может исключить 7 из 13 изделий области')
    expect(eq.find('[data-testid="history"]').text()).toContain('Пришёл журнал ИС-2: совпадение 3 из 3')
    expect(eq.find('[data-testid="history"]').text()).toContain('уверенность 0,85')
  })

  it('ошибка исполнителя — только после расследования и объяснения работника', () => {
    const perf = mountView().find('article[data-category="performer"]')
    expect(perf.find('[data-testid="performer-note"]').text()).toContain('письменного объяснения работника')
  })

  it('без права на действие его кнопка выключена, остальные — нет', () => {
    const w = mountView({ canConfirm: false })
    for (const b of w.findAll('[data-testid="confirm"]')) expect(b.attributes('disabled')).toBeDefined()
    for (const b of w.findAll('[data-testid="reject"]')) expect(b.attributes('disabled')).toBeUndefined()
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

describe('виджет гипотез: чтение и команды через API', () => {
  afterEach(() => vi.unstubAllGlobals())

  const routes = (actions: [string, string][]) => ({
    'GET /api/v1/analysis/groups': { items: [{ ...ncGroups()[1], nc_ids: ['NC-0142'] }] },
    'GET /api/v1/incidents': { items: [{ incident_id: 'INC-12', label: 'И-12', size: 6, initial_size: 34, scope_version: 3, status: 'open', opened_at: '2026-09-23T08:53:00.000Z' }] },
    'GET /api/v1/nonconformities/NC-0142/hypotheses': { ...weldHypotheses(), basis_seq: 1250 },
    'GET /api/v1/permissions': permissions(actions),
    'POST /api/v1/nonconformities/NC-0142/hypotheses/reject': { command_id: 'c', seq: 1300, event_ids: ['e'], replayed: false },
  })

  it('кнопки — по списку прав сервера', async () => {
    mockApi(routes([['analysis.hypothesis.reject', 'nonconformity']]))
    const w = await mountWidget(HypothesisWidget, { widgetId: 'hypothesis', titleKey: 'ncCard.hypotheses.title' })
    const eq = w.find('article[data-category="equipment"]')
    expect(eq.find('[data-testid="reject"]').attributes('disabled')).toBeUndefined()
    expect(eq.find('[data-testid="confirm"]').attributes('disabled')).toBeDefined()
    expect(eq.find('[data-testid="request-measurement"]').attributes('disabled')).toBeDefined()
  })

  it('«отклонить» уходит командой с basis_seq ответа и policy_seq политики', async () => {
    const calls = mockApi(routes([['analysis.hypothesis.reject', 'nonconformity']]))
    const w = await mountWidget(HypothesisWidget, { widgetId: 'hypothesis', titleKey: 'ncCard.hypotheses.title' })
    const eq = () => w.find('article[data-category="equipment"]')
    await eq().find('[data-testid="reject"]').trigger('click')
    await eq().find('[data-testid="form-first"]').setValue('Журнал станка не подтверждает')
    await eq().find('form').trigger('submit')
    await settle()
    const post = calls.find((c) => c.method === 'POST')
    expect(post?.path).toBe('/api/v1/nonconformities/NC-0142/hypotheses/reject')
    expect(post?.body).toMatchObject({ hypothesis_id: 'h-equipment', reason: { text: 'Журнал станка не подтверждает' }, basis_seq: 1250, policy_seq: 7 })
    expect((post?.body as { command_id: string }).command_id).toMatch(/^[0-9a-f-]{36}$/)
    expect(w.find('[data-testid="command-sent"]').exists()).toBe(true)
  })
})
