// «Разбор причин»: группы несоответствий вид дефекта × операция × оборудование.
import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { describe, expect, it } from 'vitest'
import { i18n } from '@/shared/i18n'
import { ncGroups } from '@/entities/incident/__tests__/fixtures'
import CauseAnalysisView from '../ui/CauseAnalysisView.vue'
import CauseAnalysisWidget from '../ui/CauseAnalysisWidget.vue'

describe('разбор причин', () => {
  it('группы по убыванию числа несоответствий; выбор группы', async () => {
    const w = mount(CauseAnalysisView, { props: { groups: ncGroups(), selected: null }, global: { plugins: [createPinia(), i18n] } })
    const rows = w.findAll('tbody tr')
    expect(rows.map((r) => r.attributes('data-group'))).toEqual(['burn_through|welding|IS-3', 'burr|milling|CNC-7', 'porosity|welding|IS-2'])
    expect(rows[0]!.text()).toContain('Прожог')
    expect(rows[0]!.text()).toContain('Есть только гипотеза')
    await rows[0]!.trigger('click')
    expect(w.emitted('update:selected')?.[0]).toEqual(['burn_through|welding|IS-3'])
  })

  it('виджет без операции API — «несоответствий нет», рамка в норме', () => {
    const w = mount(CauseAnalysisWidget, {
      props: { widgetId: 'cause-analysis', titleKey: 'desks.causeAnalysis', slotId: 'groups', slice: {}, density: 'compact' },
      global: { plugins: [createPinia(), i18n] },
    })
    expect(w.attributes('data-state')).toBe('normal')
    expect(w.text()).toContain('За период несоответствий нет')
  })
})
