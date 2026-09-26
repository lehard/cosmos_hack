// Общие факторы «сколько из N» (FR-135): общий для всей группы — первым,
// вход из строки в гипотезу и в сужение области риска.
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it } from 'vitest'
import { i18n } from '@/shared/i18n'
import { useMomentStore } from '@/shared/model/moment'
import { weldFactors } from '@/entities/incident/__tests__/fixtures'
import CommonFactorsView from '../ui/CommonFactorsView.vue'

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
