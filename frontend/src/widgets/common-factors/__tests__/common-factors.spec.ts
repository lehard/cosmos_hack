// Общие факторы «сколько из N» (FR-135): общий для всей группы — первым,
// вход из строки в гипотезу и в сужение области риска.
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useAnalysisFocusStore } from '@/entities/incident'
import { mockApi, mountWidget } from '@/entities/incident/__tests__/api-mock'
import { i18n } from '@/shared/i18n'
import { useMomentStore } from '@/shared/model/moment'
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
  it('строки по убыванию «сколько из N», общий для всех — первым и отмечен', () => {
    const w = mountView()
    const rows = w.findAll('tbody tr')
    expect(rows.map((r) => r.attributes('data-factor'))).toEqual(['machine', 'program', 'fixture', 'material_batch', 'performer', 'tool'])
    expect(rows[0]!.classes()).toContain('common')
    expect(rows[0]!.text()).toContain('Сварочный источник ИС-3')
    expect(rows[0]!.text()).toContain('3 из 3')
    expect(rows[0]!.text()).toContain('Общий для всей группы')
    expect(w.text()).toContain('3\u00a0несоответствия')
  })

  it('«сварщики разные»: у исполнителя несколько значений', () => {
    const performer = mountView().find('tr[data-factor="performer"]')
    expect(performer.text()).toContain('разные (3)')
    expect(performer.text()).toContain('1 из 3')
  })

  it('из строки — вход в гипотезу и в сужение области', async () => {
    const w = mountView()
    await w.find('tr[data-factor="machine"] [data-testid="to-hypothesis"]').trigger('click')
    await w.find('tr[data-factor="machine"] [data-testid="to-narrow-scope"]').trigger('click')
    expect(w.emitted('to-hypothesis')?.[0]?.[0]).toMatchObject({ factor: 'machine', matches: 3 })
    expect(w.emitted('to-narrow-scope')?.[0]?.[0]).toMatchObject({ factor: 'machine' })
  })

  it('неизвестный фактор не сужает; в воспроизведении входы выключены', async () => {
    const w = mountView()
    expect(w.find('tr[data-factor="tool"] [data-testid="to-narrow-scope"]').attributes('disabled')).toBeDefined()
    expect(w.find('tr[data-factor="tool"]').text()).toContain('Неизвестно')
    useMomentStore().travel('2026-09-23T09:00:00.000Z')
    await w.vm.$nextTick()
    expect(w.find('tr[data-factor="machine"] [data-testid="to-hypothesis"]').attributes('disabled')).toBeDefined()
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
    expect(w.find('tbody tr').attributes('data-factor')).toBe('machine')
    await w.find('tr[data-factor="machine"] [data-testid="to-hypothesis"]').trigger('click')
    expect(useAnalysisFocusStore().factor).toMatchObject({ intent: 'hypothesis', row: { factor: 'machine' } })
  })
})
