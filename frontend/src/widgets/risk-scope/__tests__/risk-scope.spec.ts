// Область риска — «тающая область» (FR-61, FR-62): версии 34 → 13 → 6 с
// основаниями, разбивка по местам, две оси статуса изделия.
import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { i18n } from '@/shared/i18n'
import { mockApi, mountWidget, permissions, settle } from '@/entities/incident/__tests__/api-mock'
import { weldCircumstances, weldScope } from '@/entities/incident/__tests__/fixtures'
import RiskScopeView from '../ui/RiskScopeView.vue'
import RiskScopeWidget from '../ui/RiskScopeWidget.vue'

const mountView = (model = weldScope(), props: Record<string, unknown> = {}) =>
  mount(RiskScopeView, { props: { model, ...props }, global: { plugins: [createPinia(), i18n] } })

describe('область риска', () => {
  it('герой — путь 34 → 13 → 6 кнопками; открыта текущая ступень; изделие выходит только с доказательством', () => {
    const w = mountView()
    expect(w.findAll('[data-testid="size"]').map((s) => s.text())).toEqual(['34', '13', '6'])
    expect(w.find('[data-testid="path-step"][aria-pressed="true"]').attributes('data-version')).toBe('3')
    expect(w.find('[data-testid="reduction"]').text()).toBe('Область сокращена: 34 → 6')
    expect(w.find('[data-testid="not-defective"]').text()).toContain('«нет данных» — не «годно»')
  })

  it('щелчок по числу — почему такой размер: что произошло, основание, автор, время, где изделия', async () => {
    const w = mountView()
    await w.find('[data-testid="path-step"][data-version="2"]').trigger('click')
    const step = w.find('[data-testid="version-line"]')
    expect(step.attributes('data-version')).toBe('2')
    expect(step.find('.delta').text()).toBe('−21')
    expect(step.text()).toContain('Сужено человеком по основанию')
    expect(step.text()).toContain('Журнал станка: до 08:05 режим в норме')
    expect(step.text()).toContain('Технолог Т-03')
    expect(step.text()).toContain('доказательств: 2')
    await w.find('[data-testid="path-step"][data-version="1"]').trigger('click')
    expect(w.find('[data-testid="version-line"]').text()).toContain('Система собрала область')
    expect(w.find('[data-testid="version-line"]').text()).toContain('Правило системы')
  })

  it('расширение новыми данными — отдельной ступенью, с приростом', () => {
    const m = weldScope()
    m.versions.push({ ...m.versions[2]!, scope_version: 4, change: 'expanded', size: 18, author: null, reason: { text: 'Поздний журнал: отклонение с 07:47' } })
    const step = mountView(m).find('[data-testid="version-line"]')
    expect(step.attributes('data-version')).toBe('4')
    expect(step.text()).toContain('Новые данные расширили область')
    expect(step.find('.delta').text()).toBe('+12')
  })

  it('изделия — по раскрытию, группами: серое (нет данных) — не зелёное; щелчок — открыть изделие', async () => {
    const w = mountView()
    const items = w.find('[data-testid="items"]')
    expect(items.element.tagName).toBe('DETAILS')
    expect(items.find('.group[data-known="unknown"]').text()).toContain('не исключено')
    expect(items.find('.group[data-known="confirmed"] [data-item="ANT:FL-0042"]').text()).toContain('Заблокировать')
    await items.find('[data-item="ANT:FL-0043"] button').trigger('click')
    expect(w.emitted('open-item')?.[0]).toEqual(['ANT:FL-0043'])
  })

  it('сужение без основания — видно и помечено ошибкой', () => {
    const m = weldScope()
    m.versions[2]!.reason = null
    m.versions[2]!.evidence_event_ids = []
    const w = mountView(m)
    expect(w.find('[data-testid="issues"]').text()).toContain('Версия 3: изделия вышли из области без основания')
    expect(w.find('[data-testid="path-step"][data-version="3"]').attributes('data-basis')).toBe('missing')
    expect(w.find('[data-testid="version-line"] .step-basis').text()).toContain('без основания')
  })

  it('без прав «сузить / расширить» выключены', () => {
    const w = mountView(weldScope(), { canNarrow: false, canExpand: false })
    expect(w.find('[data-testid="narrow"]').attributes('disabled')).toBeDefined()
    expect(w.find('[data-testid="expand"]').attributes('disabled')).toBeDefined()
  })

  it('сузить: изделия, запись-доказательство и основание; без доказательства не отправить — и сказано почему', async () => {
    const evidence = weldCircumstances().records
    const w = mountView(weldScope(), { evidenceOptions: evidence })
    await w.find('[data-testid="narrow"]').trigger('click')
    expect(w.find('[data-pick="ANT:FL-0042"]').exists()).toBe(true)
    await w.find('[data-pick="ANT:FL-0043"]').setValue(true)
    await w.find('[data-testid="scope-reason"]').setValue('Доп. ВИК: признаки не обнаружены')
    expect(w.find('[data-testid="scope-submit"]').attributes('disabled')).toBeDefined()
    expect(w.find('[data-testid="form-missing"]').text()).toContain('отметить запись-доказательство')
    const first = evidence[0]!.event_id
    await w.find(`[data-evidence="${first}"]`).setValue(true)
    expect(w.find('[data-testid="form-missing"]').exists()).toBe(false)
    await w.find('form').trigger('submit')
    expect(w.emitted('narrow')?.[0]).toEqual([{ item_ids: ['ANT:FL-0043'], reason: 'Доп. ВИК: признаки не обнаружены', evidence_event_ids: [first] }])
  })

  it('готовое сужение по данным — один щелчок: изделия, записи-доказательства и основание уходят командой', async () => {
    const m = weldScope()
    m.narrow_options = [{
      label: 'Исключить сваренные на ИС-1 — журнал в уставке',
      item_ids: ['ANT:FL-0043', 'ANT:FL-0044'],
      evidence: [{ event_id: 'e-is1', event_type: 'equipment.cycle.summarized', occurred_at: '2026-09-23T07:00:00Z', text: 'ИС-1: ток 158–163 А в уставке' }],
      reason_text: 'Журнал ИС-1 непрерывный и в уставке',
    }]
    const w = mountView(m)
    const opt = w.find('[data-testid="narrow-options"]')
    expect(opt.text()).toContain('Исключить сваренные на ИС-1')
    expect(opt.text()).toContain('ИС-1: ток 158–163 А в уставке')
    await opt.find('[data-testid="narrow-option"]').trigger('click')
    expect(w.emitted('narrow')?.[0]).toEqual([{ item_ids: ['ANT:FL-0043', 'ANT:FL-0044'], reason: 'Журнал ИС-1 непрерывный и в уставке', evidence_event_ids: ['e-is1'] }])
  })

  it('без права сужать готовых сужений не видно', () => {
    const m = weldScope()
    m.narrow_options = [{ label: 'x', item_ids: ['ANT:FL-0043'], evidence: [{ event_id: 'e', event_type: 't', occurred_at: '2026-09-23T07:00:00Z' }], reason_text: 'r' }]
    expect(mountView(m, { canNarrow: false }).find('[data-testid="narrow-options"]').exists()).toBe(false)
  })

  it('сослаться не на что — сузить нельзя, так и написано', async () => {
    const w = mountView()
    await w.find('[data-testid="narrow"]').trigger('click')
    expect(w.find('[data-testid="no-evidence"]').text()).toContain('Сузить область нельзя, пока нет данных')
  })

  it('автор ступени — по имени из ответа', async () => {
    const m = weldScope()
    m.versions[2]!.author = 'TEC-01'
    m.versions[2]!.author_name = 'Е. Орлова'
    expect(mountView(m).find('[data-testid="version-line"]').text()).toContain('Е. Орлова')
  })

  it('ступень: повод (опоздавшие данные), исключённые изделия и доказательства словами', () => {
    const m = weldScope()
    m.versions[2] = {
      ...m.versions[2]!,
      trigger: { kind: 'late_event', label: 'пришёл журнал ИС-2: ток 176 А при уставке 160 ± 10 А' },
      items_removed: ['ANT:FL-0040'],
      evidence: [{ event_id: 'e1', event_type: 'equipment.deviation.detected', occurred_at: '2026-09-22T07:20:00Z', text: 'Ток 176 А вне уставки' }],
    }
    const step = mountView(m).find('[data-testid="version-line"]')
    expect(step.find('[data-testid="step-trigger"]').text()).toBe('Пришли опоздавшие данные: пришёл журнал ИС-2: ток 176 А при уставке 160 ± 10 А')
    expect(step.find('[data-testid="step-items"]').text()).toContain('исключены')
    expect(step.find('[data-testid="step-items"]').text()).toContain('FL-0040')
    expect(step.find('[data-testid="step-evidence"]').text()).toContain('Ток 176 А вне уставки')
  })

  it('расширить: номера изделий списком', async () => {
    const w = mountView()
    await w.find('[data-testid="expand"]').trigger('click')
    await w.find('[data-testid="expand-items"]').setValue('ANT:FL-0050, ANT:FL-0051\nANT:FL-0052')
    await w.find('[data-testid="scope-reason"]').setValue('Тот же источник после 08:52')
    await w.find('form').trigger('submit')
    expect(w.emitted('expand')?.[0]).toEqual([{ item_ids: ['ANT:FL-0050', 'ANT:FL-0051', 'ANT:FL-0052'], reason: 'Тот же источник после 08:52', evidence_event_ids: [] }])
  })
})

describe('виджет области риска: чтение и команды через API', () => {
  afterEach(() => vi.unstubAllGlobals())

  const incident = (id: string, label: string) => ({ incident_id: id, label, size: 6, initial_size: 34, scope_version: 3, status: 'open', opened_at: '2026-09-23T08:53:00.000Z', primary_nc_id: 'NC-0142', nc_ids: ['NC-0142'] })

  it('область первого открытого инцидента; сужение — командой с основанием и правами', async () => {
    const calls = mockApi({
      'GET /api/v1/incidents': { items: [incident('INC-12', 'И-12'), incident('INC-13', 'И-13')] },
      'GET /api/v1/incidents/INC-12/risk-scope': { ...weldScope(), basis_seq: 1400 },
      'GET /api/v1/permissions': permissions([['analysis.scope.narrow', 'incident']], 9),
      'GET /api/v1/nonconformities/NC-0142/circumstances': { ...weldCircumstances(), basis_seq: 1250 },
      'POST /api/v1/incidents/INC-12/scope/narrow': { command_id: 'c', seq: 1401, event_ids: ['e'], replayed: false },
    })
    const w = await mountWidget(RiskScopeWidget, { widgetId: 'risk-scope', titleKey: 'riskScope.title' })
    expect(w.attributes('data-state')).toBe('defect_indication')
    expect(w.findAll('[data-testid="size"]').map((s) => s.text())).toEqual(['34', '13', '6'])
    expect(w.find('[data-testid="incident-pick"]').exists()).toBe(true)
    expect(w.find('[data-testid="expand"]').attributes('disabled')).toBeDefined()

    await w.find('[data-testid="narrow"]').trigger('click')
    await w.find('[data-pick="ANT:FL-0044"]').setValue(true)
    const ev = weldCircumstances().records[0]!.event_id
    await w.find(`[data-evidence="${ev}"]`).setValue(true)
    await w.find('[data-testid="scope-reason"]').setValue('Доп. ВИК: признаки не обнаружены')
    await w.find('form').trigger('submit')
    await settle()
    const post = calls.find((c) => c.method === 'POST')
    expect(post?.path).toBe('/api/v1/incidents/INC-12/scope/narrow')
    expect(post?.body).toMatchObject({ item_ids: ['ANT:FL-0044'], reason: { text: 'Доп. ВИК: признаки не обнаружены' }, evidence_event_ids: [ev], basis_seq: 1400, policy_seq: 9 })
  })

  it('инцидентов нет — «активных областей риска нет»', async () => {
    mockApi({ 'GET /api/v1/incidents': { items: [] } })
    const w = await mountWidget(RiskScopeWidget, { widgetId: 'risk-scope', titleKey: 'riskScope.title' })
    expect(w.attributes('data-state')).toBe('normal')
    expect(w.text()).toContain('Активных областей риска нет')
  })
})
