// «Разбор причин»: группы несоответствий вид дефекта × операция × оборудование.
import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { useAnalysisFocusStore } from '@/entities/incident'
import { mockApi, mountWidget } from '@/entities/incident/__tests__/api-mock'
import { i18n } from '@/shared/i18n'
import { ncGroups } from '@/entities/incident/__tests__/fixtures'
import CauseAnalysisView from '../ui/CauseAnalysisView.vue'
import CauseAnalysisWidget from '../ui/CauseAnalysisWidget.vue'

describe('разбор причин', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('группы по убыванию числа несоответствий; выбор группы', async () => {
    const w = mount(CauseAnalysisView, { props: { groups: ncGroups(), selected: null }, global: { plugins: [createPinia(), i18n] } })
    const rows = w.findAll('tbody tr')
    expect(rows.map((r) => r.attributes('data-group'))).toEqual(['burn_through|welding|IS-3', 'burr|milling|CNC-7', 'porosity|welding|IS-2'])
    expect(rows[0]!.text()).toContain('Прожог')
    expect(rows[0]!.text()).toContain('Есть только гипотеза')
    await rows[0]!.trigger('click')
    expect(w.emitted('update:selected')?.[0]).toEqual(['burn_through|welding|IS-3'])
  })

  it('виджет: группы с сервера, выбор группы — в фокус разбора вместе с первым несоответствием', async () => {
    mockApi({ 'GET /api/v1/analysis/groups': { items: ncGroups().map((g, i) => ({ ...g, nc_ids: [`NC-${i}`] })) } })
    const w = await mountWidget(CauseAnalysisWidget, { widgetId: 'cause-analysis', titleKey: 'desks.causeAnalysis' })
    expect(w.attributes('data-state')).toBe('normal')
    await w.find('tr[data-group="burn_through|welding|IS-3"]').trigger('click')
    const focus = useAnalysisFocusStore()
    expect([focus.groupKey, focus.ncId]).toEqual(['burn_through|welding|IS-3', 'NC-1'])
  })

  it('виджет: групп нет — «несоответствий нет», рамка в норме', async () => {
    mockApi({ 'GET /api/v1/analysis/groups': { items: [] } })
    const w = await mountWidget(CauseAnalysisWidget, { widgetId: 'cause-analysis', titleKey: 'desks.causeAnalysis' })
    expect(w.attributes('data-state')).toBe('normal')
    expect(w.text()).toContain('За период несоответствий нет')
  })
})
