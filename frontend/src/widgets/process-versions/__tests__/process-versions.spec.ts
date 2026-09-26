// Раздел «Процесс» (FR-24): версии, читаемое представление, читаемая разница
// с действующей.
import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mockApi, mountWidget } from '@/entities/incident/__tests__/api-mock'
import { i18n } from '@/shared/i18n'
import { processVersions } from '@/entities/process-version/__tests__/fixtures'
import ProcessVersionsView from '../ui/ProcessVersionsView.vue'
import ProcessVersionsWidget from '../ui/ProcessVersionsWidget.vue'

const mountView = (props: Record<string, unknown> = {}) =>
  mount(ProcessVersionsView, {
    props: { versions: processVersions(), initialMode: 'diff', selected: null, 'onUpdate:selected': () => {}, ...props },
    global: { plugins: [createPinia(), i18n] },
  })

describe('раздел «Процесс»', () => {
  it('по умолчанию — версия на утверждении и её отличия от действующей', () => {
    const w = mountView()
    expect(w.emitted('update:selected')?.[0]).toEqual(['pv-0.2'])
    const lines = w.findAll('[data-testid="diff"] li').map((l) => l.text())
    expect(lines).toEqual([
      '«Сварка»: Норма времени 40 мин → 35 мин',
      'Добавлена точка предъявления после «Сварка»',
      'Порог уверенности для вида «burn_through»: 0,85 → 0,80',
    ])
    expect(w.text()).toContain('Кворум утверждения: 1 из 3')
    expect(w.text()).toContain('только для изделий, запущенных после вступления в силу')
  })

  it('читаемое представление: элементы, виды, свойства и пороги', async () => {
    const w = mountView({ selected: 'pv-0.1' })
    await w.find('[data-testid="mode-view"]').trigger('click')
    const r = w.find('[data-testid="readable"]')
    expect(r.findAll('li.element')).toHaveLength(5)
    const weld = r.find('li[data-kind="operation"]:nth-child(3)').text()
    expect(weld).toContain('Операция')
    expect(weld).toContain('Сварка')
    expect(weld).toContain('Специальный процесс')
    expect(weld).toContain('Да')
    expect(r.text()).toContain('burn_through: 0,85')
    expect(r.text()).toContain('Автоматизированный контроль')
  })

  it('свойства расширения BPMN — подписями, значения — словами; код — только без перевода', async () => {
    const [, active] = processVersions()
    const props = { triggerEventType: 'erp.order.received', stepKind: 'human_inspection', paperAttester: 'site_foreman', closingPoint: 'ZT-1', erpAction: 'unknown_action', futureAttr: 'x' }
    const v = { ...active!, elements: [{ id: 'n1', step_key: 'order.received', kind: 'humanInspection' as const, name: 'Приёмка', properties: props as never }] }
    const w = mountView({ versions: [v], selected: v.version_id })
    await w.find('[data-testid="mode-view"]').trigger('click')
    const text = (k: string) => w.find(`[data-prop="${k}"]`).text()
    expect(text('triggerEventType')).toBe('Событие-триггерПолучено производственное задание')
    expect(text('stepKind')).toBe('Вид шагаКонтроль человеком')
    expect(text('paperAttester')).toBe('Кто заверяет бумажную подписьМастер участка')
    expect(text('closingPoint')).toBe('Закрывающая точкаZT-1')
    expect(text('erpAction')).toBe('Учётное действие в 1Сunknown_action')
    expect(text('futureAttr')).toBe('futureAttrx')
    expect(w.text()).not.toContain('process.properties')
    // «Шаг:» — обычным шрифтом, код шага — отдельно моноширинным.
    const step = w.find('.step')
    expect(step.text()).toBe('Шаг: order.received')
    expect(step.find('code').text()).toBe('order.received')
  })

  it('отличия: свойство и значения — словами', () => {
    const [draft, active] = processVersions()
    const w = mountView({
      selected: draft!.version_id,
      versions: [draft, active],
      diff: [{ kind: 'propertyChanged', element: 'Приёмка', property: 'stepKind', from: 'movement', to: 'human_inspection' }],
    })
    expect(w.find('[data-testid="diff"] li').text()).toBe('«Приёмка»: Вид шага Перемещение → Контроль человеком')
  })

  it('у действующей версии отличий от себя нет — так и написано', () => {
    const w = mountView({ selected: 'pv-0.1' })
    expect(w.find('[data-testid="diff"]').text()).toContain('Это действующая версия')
  })

  it('без действующей версии сравнивать не с чем', () => {
    const vs = processVersions().filter((v) => v.status !== 'active')
    const w = mountView({ versions: vs })
    expect(w.find('[data-testid="diff"]').text()).toContain('Действующей версии нет')
  })

  it('готовый перечень отличий от сервера показывается как есть', () => {
    const w = mountView({ selected: 'pv-0.2', diff: [{ kind: 'elementAdded', element: 'Рентген шва' }] })
    expect(w.findAll('[data-testid="diff"] li').map((l) => l.text())).toEqual(['Добавлен элемент «Рентген шва»'])
  })

  it('выбор версии в списке', async () => {
    const w = mountView()
    await w.find('button[data-version="pv-0.1"]').trigger('click')
    expect(w.emitted('update:selected')?.at(-1)).toEqual(['pv-0.1'])
  })
})

describe('виджет «Процесс» через API', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('список, выбранная версия и отличия от действующей — с сервера', async () => {
    const [draft, active] = processVersions()
    const summary = (v: typeof draft) => ({ version_id: v!.version_id, label: v!.label, status: v!.status, created_at: v!.created_at, items_in_work: 0 })
    const calls = mockApi({
      'GET /api/v1/process/versions': { items: [summary(draft), summary(active)] },
      'GET /api/v1/process/versions/pv-0.2': { ...draft, basis_seq: 10 },
      'GET /api/v1/process/versions/pv-0.1': { ...active, basis_seq: 10 },
      'GET /api/v1/process/versions/pv-0.2/diff': { version_id: 'pv-0.2', against_id: 'pv-0.1', entries: [{ kind: 'presentationPointAdded', step: 'Сварка' }] },
    })
    const w = await mountWidget(ProcessVersionsWidget, { widgetId: 'process-versions', titleKey: 'desks.process', slice: { mode: 'diff' } })
    expect(w.attributes('data-mode')).toBe('fixtures')
    expect(w.findAll('[data-testid="diff"] li').map((l) => l.text())).toEqual(['Добавлена точка предъявления после «Сварка»'])
    expect(calls.map((c) => c.path)).toContain('/api/v1/process/versions/pv-0.2/diff')
  })
})
