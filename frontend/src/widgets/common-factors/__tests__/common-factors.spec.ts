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
  it('что объединяет случаи — совпадения фишками «значение · k/n», общий для всех первым; «совпадение — не причина»', () => {
    const w = mountView()
    const chips = w.findAll('li.chip')
    expect(chips.map((c) => c.attributes('data-factor'))).toEqual(['machine', 'program', 'fixture', 'material_batch'])
    expect(chips[0]!.attributes('data-kind')).toBe('all')
    expect(chips[0]!.text()).toContain('Станок: Сварочный источник ИС-3')
    expect(chips[0]!.find('[data-testid="how-many"]').text()).toBe('3/3')
    expect(w.find('li.chip[data-factor="fixture"] [data-testid="how-many"]').text()).toBe('2/3')
    expect(w.find('.lead').text()).toContain('Что объединяет случаи')
    expect(w.text()).toContain('обстоятельство, а не причина')
  })

  it('не совпадает и нет данных — одной строкой; кнопок-пустышек нет', () => {
    const w = mountView()
    expect(w.find('[data-testid="differs"]').text()).toBe('Не совпадает: исполнитель')
    expect(w.find('[data-testid="unknown"]').text()).toBe('Нет данных: инструмент')
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
    expect(w.find('li.chip').attributes('data-factor')).toBe('machine')
  })
})
