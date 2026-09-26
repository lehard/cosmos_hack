// Область риска — «тающая область» (FR-61, FR-62): версии 34 → 13 → 6 с
// основаниями, разбивка по местам, две оси статуса изделия.
import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { i18n } from '@/shared/i18n'
import { mockApi, mountWidget, permissions, settle } from '@/entities/incident/__tests__/api-mock'
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

  it('без прав «сузить / расширить» выключены', () => {
    const w = mountView(weldScope(), { canNarrow: false, canExpand: false })
    expect(w.find('[data-testid="narrow"]').attributes('disabled')).toBeDefined()
    expect(w.find('[data-testid="expand"]').attributes('disabled')).toBeDefined()
  })

  it('сузить: выбрать изделия и указать основание; без основания не отправить', async () => {
    const w = mountView()
    await w.find('[data-testid="narrow"]').trigger('click')
    expect(w.find('[data-pick="ANT:FL-0042"]').exists()).toBe(true)
    await w.find('[data-pick="ANT:FL-0043"]').setValue(true)
    expect(w.find('[data-testid="scope-submit"]').attributes('disabled')).toBeDefined()
    await w.find('[data-testid="scope-reason"]').setValue('Доп. ВИК: признаки не обнаружены')
    await w.find('form').trigger('submit')
    expect(w.emitted('narrow')?.[0]).toEqual([{ item_ids: ['ANT:FL-0043'], reason: 'Доп. ВИК: признаки не обнаружены' }])
  })

  it('расширить: номера изделий списком', async () => {
    const w = mountView()
    await w.find('[data-testid="expand"]').trigger('click')
    await w.find('[data-testid="expand-items"]').setValue('ANT:FL-0050, ANT:FL-0051\nANT:FL-0052')
    await w.find('[data-testid="scope-reason"]').setValue('Тот же источник после 08:52')
    await w.find('form').trigger('submit')
    expect(w.emitted('expand')?.[0]).toEqual([{ item_ids: ['ANT:FL-0050', 'ANT:FL-0051', 'ANT:FL-0052'], reason: 'Тот же источник после 08:52' }])
  })
})

describe('виджет области риска: чтение и команды через API', () => {
  afterEach(() => vi.unstubAllGlobals())

  const incident = (id: string, label: string) => ({ incident_id: id, label, size: 6, initial_size: 34, scope_version: 3, status: 'open', opened_at: '2026-09-23T08:53:00.000Z' })

  it('область первого открытого инцидента; сужение — командой с основанием и правами', async () => {
    const calls = mockApi({
      'GET /api/v1/incidents': { items: [incident('INC-12', 'И-12'), incident('INC-13', 'И-13')] },
      'GET /api/v1/incidents/INC-12/risk-scope': { ...weldScope(), basis_seq: 1400 },
      'GET /api/v1/permissions': permissions([['analysis.scope.narrow', 'incident']], 9),
      'POST /api/v1/incidents/INC-12/scope/narrow': { command_id: 'c', seq: 1401, event_ids: ['e'], replayed: false },
    })
    const w = await mountWidget(RiskScopeWidget, { widgetId: 'risk-scope', titleKey: 'riskScope.title' })
    expect(w.attributes('data-state')).toBe('defect_indication')
    expect(w.findAll('[data-testid="size"]').map((s) => s.text())).toEqual(['34', '13', '6'])
    expect(w.find('[data-testid="incident-pick"]').exists()).toBe(true)
    expect(w.find('[data-testid="expand"]').attributes('disabled')).toBeDefined()

    await w.find('[data-testid="narrow"]').trigger('click')
    await w.find('[data-pick="ANT:FL-0044"]').setValue(true)
    await w.find('[data-testid="scope-reason"]').setValue('Доп. ВИК: признаки не обнаружены')
    await w.find('form').trigger('submit')
    await settle()
    const post = calls.find((c) => c.method === 'POST')
    expect(post?.path).toBe('/api/v1/incidents/INC-12/scope/narrow')
    expect(post?.body).toMatchObject({ item_ids: ['ANT:FL-0044'], reason: { text: 'Доп. ВИК: признаки не обнаружены' }, basis_seq: 1400, policy_seq: 9 })
  })

  it('инцидентов нет — «активных областей риска нет»', async () => {
    mockApi({ 'GET /api/v1/incidents': { items: [] } })
    const w = await mountWidget(RiskScopeWidget, { widgetId: 'risk-scope', titleKey: 'riskScope.title' })
    expect(w.attributes('data-state')).toBe('normal')
    expect(w.text()).toContain('Активных областей риска нет')
  })
})
