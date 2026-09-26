// Экран «Разбор обстоятельств» (FR-153, FR-58): три синхронные дорожки, окно
// возможного возникновения, ход «до / во время / после», подсветка связанного
// кадра и записи журнала по клику, формулировки без «причины».
import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { describe, expect, it } from 'vitest'
import { i18n } from '@/shared/i18n'
import { incomingCircumstances, weldCircumstances } from '@/entities/incident/__tests__/fixtures'
import CircumstancesView from '../ui/CircumstancesView.vue'
import CircumstancesWidget from '../ui/CircumstancesWidget.vue'

const mountView = (model = weldCircumstances()) =>
  mount(CircumstancesView, { props: { model, selected: null, 'onUpdate:selected': () => {} }, global: { plugins: [createPinia(), i18n] } })

const leftOf = (w: ReturnType<typeof mountView>, id: string) => (w.find(`[data-event="${id}"]`).attributes('style') ?? '').match(/left: ([\d.]+)%/)?.[1]

describe('разбор обстоятельств', () => {
  it('три дорожки: изделие, человек, оборудование — с записями своих источников', () => {
    const w = mountView()
    const lanes = w.findAll('.lane').map((l) => l.attributes('data-lane'))
    expect(lanes).toEqual(['item', 'person', 'equipment'])
    expect(w.findAll('[data-lane="item"] .mark')).toHaveLength(2)
    expect(w.findAll('[data-lane="person"] .mark')).toHaveLength(3)
    expect(w.findAll('[data-lane="equipment"] .mark')).toHaveLength(4)
    expect(w.text()).toContain('Изделие')
    expect(w.text()).toContain('Человек')
    expect(w.text()).toContain('Оборудование')
  })

  it('дорожки синхронны: одно время — одна позиция на всех дорожках', () => {
    const w = mountView()
    // 08:27 — ручная подача у исполнителя и ручное изменение режима у станка
    expect(leftOf(w, 'e-manual')).toBeDefined()
    expect(leftOf(w, 'e-manual')).toBe(leftOf(w, 'e-override'))
    expect(Number(leftOf(w, 'e-kt2-ok'))).toBeLessThan(Number(leftOf(w, 'e-cycle-start')))
  })

  it('до сварки чисто → во время две нештатности → после признак дефекта', () => {
    const w = mountView()
    const phase = (p: string) => w.find(`.phase[data-phase="${p}"]`).text()
    expect(phase('before')).toContain('До операции')
    expect(phase('before')).toContain('ВИК: признаки не обнаружены')
    expect(phase('during')).toContain('Сварка')
    expect(phase('during')).toContain('Вне уставки: ток 212 А при уставке 180 А')
    expect(phase('during')).toContain('Ручное изменение режима: подача проволоки 130 %')
    expect(w.findAll('.phase[data-phase="during"] li[data-tone="deviation"]')).toHaveLength(2)
    expect(phase('after')).toContain('Визуальный контроль: обнаружен признак дефекта')
  })

  it('окно возможного возникновения — поверх дорожек, от нормы до первой находки', () => {
    const w = mountView()
    const band = w.find('[data-testid="window-band"]')
    expect(band.exists()).toBe(true)
    const style = band.attributes('style') ?? ''
    expect(style).toContain(`left: ${leftOf(w, 'e-kt2-ok')}%`)
    expect(w.find('[data-testid="window-legend"]').text()).toContain('Окно возможного возникновения')
    expect(w.find('[data-testid="operation-band"]').text()).toContain('Сварка')
  })

  it('клик по находке подсвечивает связанные записи, кадры и запись журнала', async () => {
    const w = mountView()
    await w.find('[data-event="e-kt3-defect"]').trigger('click')
    const emitted = w.emitted('update:selected')
    expect(emitted?.[0]).toEqual(['e-kt3-defect'])
    await w.setProps({ selected: 'e-kt3-defect' })
    expect(w.find('[data-event="e-kt3-defect"]').classes()).toContain('is-selected')
    expect(w.find('[data-event="e-current"]').classes()).toContain('is-linked')
    expect(w.find('[data-event="e-override"]').classes()).toContain('is-linked')
    expect(w.find('[data-event="e-took"]').classes()).toContain('is-dim')
    const details = w.find('[data-testid="details"]')
    expect(details.text()).toContain('Запись журнала № 1240')
    expect(details.findAll('[data-testid="evidence"]')).toHaveLength(2)
    await details.find('[data-testid="journal-record"]').trigger('click')
    expect(w.emitted('open-record')?.[0]).toEqual([1240, 'e-kt3-defect'])
    await details.find('[data-testid="evidence"]').trigger('click')
    expect(w.emitted('open-evidence')?.[0]).toEqual(['kt3-0042-07', 'e-kt3-defect'])
  })

  it('формулировки — «возможные обстоятельства», не «причина»; вывод не категоричен', () => {
    const w = mountView()
    expect(w.text()).toContain('Возможные обстоятельства')
    expect(w.text()).not.toMatch(/причин/i)
    expect(w.find('[data-testid="not-categorical"]').text()).toContain('категоричного вывода нет')
    expect(w.find('[data-testid="missing"]').text()).toContain('Неизвестен инструмент')
  })

  it('входной дефект не связывается с исполнителем операции', () => {
    const w = mountView(incomingCircumstances())
    expect(w.find('[data-testid="incoming-note"]').text()).toContain('с исполнителем операции не связывается')
    for (const m of w.findAll('[data-lane="person"] .mark')) expect(m.classes()).toContain('is-muted')
  })

  it('без окна — явное «определить нельзя»', () => {
    const w = mountView({ ...weldCircumstances(), window: null })
    expect(w.find('[data-testid="window-band"]').exists()).toBe(false)
    expect(w.find('[data-testid="window-unknown"]').exists()).toBe(true)
  })

  it('неизвестный тип записи — UNKNOWN(тип), а не похожая подпись', () => {
    const m = weldCircumstances()
    m.records.push({ event_id: 'e-x', lane: 'equipment', event_type: 'equipment.zz.new', occurred_at: '2026-09-23T08:30:00.000Z' })
    expect(mountView(m).find('[data-event="e-x"]').text()).toContain('UNKNOWN(equipment.zz.new)')
  })
})

describe('виджет разбора обстоятельств', () => {
  it('без операции API — рамка в норме и «записей нет», без выдуманных данных', () => {
    const w = mount(CircumstancesWidget, {
      props: { widgetId: 'circumstances', titleKey: 'desks.circumstances', slotId: 'tracks', slice: {}, density: 'compact' },
      global: { plugins: [createPinia(), i18n] },
    })
    expect(w.attributes('data-state')).toBe('normal')
    expect(w.attributes('data-widget')).toBe('circumstances')
    expect(w.text()).toContain('Записей нет')
  })
})
