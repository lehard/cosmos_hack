// Область риска — «тающая область» (FR-61, FR-62): версии 34 → 13 → 6 с
// основаниями, разбивка по местам, две оси статуса изделия.
import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { describe, expect, it } from 'vitest'
import { i18n } from '@/shared/i18n'
import { weldScope } from '@/entities/incident/__tests__/fixtures'
import RiskScopeView from '../ui/RiskScopeView.vue'
import RiskScopeWidget from '../ui/RiskScopeWidget.vue'

const mountView = (model = weldScope(), props: Record<string, unknown> = {}) =>
  mount(RiskScopeView, { props: { model, ...props }, global: { plugins: [createPinia(), i18n] } })

describe('область риска', () => {
  it('сужение 34 → 13 → 6 и итог сокращения', () => {
    const w = mountView()
    expect(w.findAll('[data-testid="size"]').map((s) => s.text())).toEqual(['34', '13', '6'])
    expect(w.find('[data-testid="reduction"]').text()).toBe('Область сокращена: 34 → 6')
    expect(w.text()).toContain('Сокращение на 82,4')
  })

  it('у каждой версии — изменение, основание, автор и время', () => {
    const lines = mountView().findAll('[data-testid="version-line"]').map((l) => l.text())
    expect(lines[0]).toContain('Версия 1: Размер при создании: 34')
    expect(lines[0]).toContain('Правило системы')
    expect(lines[1]).toContain('Версия 2: Сужено: 34 → 13. Основание: Журнал станка: до 08:05 режим в норме, доказательств: 2. Технолог Т-03')
    expect(lines[2]).toContain('Сужено: 13 → 6')
    expect(lines[2]).toContain('доказательств: 7. Контролёр К-07')
  })

  it('разбивка текущей версии: в производстве / ушли дальше / собраны / отгружены', () => {
    const tiles = mountView().findAll('.tile')
    expect(tiles.map((t) => t.attributes('data-location'))).toEqual(['in_production', 'moved_on', 'assembled', 'shipped'])
    expect(tiles.map((t) => t.find('.tile-n').text())).toEqual(['5', '1', '0', '0'])
    expect(tiles[1]!.text()).toContain('Ушли дальше')
  })

  it('две оси статуса: что известно и что делать; изделия в области — не брак', () => {
    const w = mountView()
    const row = w.find('tr[data-item="ANT:FL-0042"]')
    expect(row.text()).toContain('Подтверждено')
    expect(row.text()).toContain('Заблокировать')
    expect(w.find('tr[data-item="ANT:FL-0046"]').text()).toContain('Неизвестно')
    expect(w.find('[data-testid="not-defective"]').text()).toContain('Это не брак')
    expect(w.text()).toContain('6\u00a0изделий')
  })

  it('сужение без основания — видно и помечено ошибкой', () => {
    const m = weldScope()
    m.versions[2]!.reason = null
    m.versions[2]!.evidence_event_ids = []
    const w = mountView(m)
    expect(w.find('[data-testid="issues"]').text()).toContain('Версия 3: изделия вышли из области без основания')
    expect(w.find('li[data-version="3"]').attributes('data-basis')).toBe('missing')
    expect(w.findAll('[data-testid="version-line"]')[2]!.text()).toContain('Основание: без основания')
  })

  it('без доступных команд «сузить / расширить» выключены', () => {
    const w = mountView(weldScope(), { canAct: false })
    expect(w.find('[data-testid="narrow"]').attributes('disabled')).toBeDefined()
    expect(w.find('[data-testid="expand"]').attributes('disabled')).toBeDefined()
  })

  it('виджет без операции API — «активных областей риска нет»', () => {
    const w = mount(RiskScopeWidget, {
      props: { widgetId: 'risk-scope', titleKey: 'riskScope.title', slotId: 'scope', slice: {}, density: 'compact' },
      global: { plugins: [createPinia(), i18n] },
    })
    expect(w.attributes('data-state')).toBe('normal')
    expect(w.text()).toContain('Активных областей риска нет')
  })
})
