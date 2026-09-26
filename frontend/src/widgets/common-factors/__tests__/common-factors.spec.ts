// Общие факторы «сколько из N» (FR-135): общий для всей группы — первым,
// фразами, без таблицы и кнопок-пустышек (UI-33).
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mockApi, mountWidget } from '@/entities/incident/__tests__/api-mock'
import { i18n } from '@/shared/i18n'
import { ncGroups, weldFactors } from '@/entities/incident/__tests__/fixtures'
import CommonFactorsView from '../ui/CommonFactorsView.vue'
import CommonFactorsWidget from '../ui/CommonFactorsWidget.vue'

let pinia: ReturnType<typeof createPinia>
beforeEach(() => {
  pinia = createPinia()
  setActivePinia(pinia)
})

const mountView = () => mount(CommonFactorsView, { props: { model: weldFactors() }, global: { plugins: [pinia, i18n] } })

describe('общие факторы', () => {
  it('фразами: общий для всех — первым и выделен, дальше по убыванию', () => {
    const w = mountView()
    const rows = w.findAll('li.row')
    expect(rows.map((r) => r.attributes('data-factor'))).toEqual(['machine', 'program', 'fixture', 'material_batch', 'performer', 'tool'])
    expect(rows[0]!.attributes('data-kind')).toBe('all')
    expect(rows[0]!.text()).toContain('Станок: Сварочный источник ИС-3')
    expect(rows[0]!.find('[data-testid="how-many"]').text()).toBe('у всех 3')
    expect(w.find('.lead').text()).toBe('Что общего у несоответствий — всего 3')
    expect(w.text()).toContain('обстоятельство, а не причина')
  })

  it('«сварщики разные»: значения не совпадают — без значения, со словами', () => {
    const performer = mountView().find('li[data-factor="performer"]')
    expect(performer.attributes('data-kind')).toBe('varies')
    expect(performer.find('[data-testid="how-many"]').text()).toBe('не совпадает: вариантов — 3')
    const fixture = mountView().find('li[data-factor="fixture"]')
    expect(fixture.text()).toContain('Оснастка: Приспособление П-7')
    expect(fixture.find('[data-testid="how-many"]').text()).toBe('у 2 из 3')
  })

  it('неизвестный фактор — «данных нет»; кнопок-пустышек нет', () => {
    const w = mountView()
    expect(w.find('li[data-factor="tool"] [data-testid="how-many"]').text()).toBe('неизвестно — данных нет')
    expect(w.find('button').exists()).toBe(false)
  })
})

describe('виджет общих факторов через API', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('факторы самой крупной группы, пока группа не выбрана', async () => {
    const calls = mockApi({
      'GET /api/v1/analysis/groups': { items: ncGroups() },
      'GET /api/v1/analysis/groups/burn_through|welding|IS-3/common-factors': weldFactors(),
    })
    const w = await mountWidget(CommonFactorsWidget, { widgetId: 'common-factors', titleKey: 'desks.commonFactors' })
    expect(calls.map((c) => c.path)).toContain('/api/v1/analysis/groups/burn_through|welding|IS-3/common-factors')
    expect(w.find('li.row').attributes('data-factor')).toBe('machine')
  })
})
