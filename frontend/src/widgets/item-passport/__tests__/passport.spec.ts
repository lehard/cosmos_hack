// Паспорт изделия: кто · что · когда с пометкой источника и проверкой подписи,
// журнал изменений, генеалогия, зоны, носители, компактный вид и контейнер на
// кэше запроса (FR-42, FR-43, FR-45, FR-46, FR-140, AD-21).
import { flushPromises, mount } from '@vue/test-utils'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import { itemKeys } from '@/entities/item'
import { i18n } from '@/shared/i18n'
import { flangeGenealogy, flangeHistory, flangePassport } from '@/entities/item/__tests__/fixtures'
import ItemPassportView from '../ui/ItemPassportView.vue'
import ItemPassportWidget from '../ui/ItemPassportWidget.vue'
import PassportChanges from '../ui/PassportChanges.vue'
import PassportEntries from '../ui/PassportEntries.vue'
import PassportGenealogy from '../ui/PassportGenealogy.vue'
import PassportZones from '../ui/PassportZones.vue'
import PassportCarriers from '../ui/PassportCarriers.vue'

let pinia: ReturnType<typeof createPinia>
beforeEach(() => {
  pinia = createPinia()
  setActivePinia(pinia)
})
const global = () => ({ plugins: [pinia, i18n] })

describe('записи паспорта «кто · что · когда» (FR-42, FR-140)', () => {
  const mountEntries = () => mount(PassportEntries, { props: { entries: flangePassport().entries }, global: global() })

  it('у каждой записи — пометка источника, автор, подпись и итог проверки', () => {
    const rows = mountEntries().findAll('li.row')
    expect(rows).toHaveLength(7)
    for (const r of rows) {
      expect(r.find('.source').exists()).toBe(true)
      expect(r.find('.signature').exists()).toBe(true)
    }
  })

  it('ручная отметка конца операции — «Ручной ввод», не «Станок»', () => {
    const row = mountEntries().find('li[data-id="e-weld-end"]')
    const source = row.find('.source')
    expect(source.attributes('data-source')).toBe('manual_entry')
    expect(source.attributes('data-manual')).toBe('true')
    expect(source.text()).toContain('Ручной ввод')
    expect(source.text()).not.toContain('Станок')
    expect(row.text()).toContain('Автор: op-12')
  })

  it('факт, вывод системы и решение человека различимы', () => {
    const w = mountEntries()
    expect(w.find('li[data-id="e-kt3"] .layer').text()).toBe('Факт')
    expect(w.find('li[data-id="e-signal"] .layer').text()).toBe('Вывод системы')
    expect(w.find('li[data-id="e-signal"] .source').text()).toContain('Вывод системы')
    expect(w.find('li[data-id="e-reg"] .layer').text()).toBe('Решение человека')
  })

  it('подпись: «не проверялась» ≠ «действительна»; бумага — с заверителем (FR-68, FR-139)', () => {
    const w = mountEntries()
    expect(w.find('li[data-id="e-kt3"] .signature').text()).toContain('Подпись действительна')
    const unchecked = w.find('li[data-id="e-weld-end"] .signature')
    expect(unchecked.attributes('data-check')).toBe('unchecked')
    expect(unchecked.text()).toContain('не проверялась')
    expect(unchecked.text()).not.toContain('Подпись действительна')
    const paper = w.find('li[data-id="e-fix"]')
    expect(paper.find('[data-testid="paper-line"]').text()).toContain('заверил qc-03')
    expect(paper.find('[data-testid="corrects"]').text()).toContain('Исправляет запись e-weld-end')
  })

  it('критическое действие показано номером CA', () => {
    expect(mountEntries().find('li[data-id="e-hold"] .ca').text()).toBe('CA-311')
  })
})

describe('журнал изменений, генеалогия, зоны, носители', () => {
  it('журнал изменений: ось статуса — словарём, было → стало, запись журнала (FR-43)', () => {
    const w = mount(PassportChanges, { props: { changes: flangeHistory().items }, global: global() })
    const hold = w.find('tr[data-field="containment"]')
    expect(hold.text()).toContain('Сдерживание')
    expect(hold.findAll('.status-tag').map((x) => x.text())).toEqual(['Без сдерживания', 'Блок изделия'])
    expect(hold.text()).toContain('Запись журнала № 1242')
    expect(hold.text()).toContain('Правило RM-weld-7')
    expect(w.find('tr[data-field="carrier"]').text()).toContain('TAG-7788 → DM-FL-0042')
  })

  it('генеалогия: куда входит, из чего состоит, партии, выписка без подтверждения (FR-45)', async () => {
    const w = mount(PassportGenealogy, { props: { genealogy: flangeGenealogy() }, global: global() })
    expect(w.find('[data-section="up"]').text()).toContain('Сборка ASM-9')
    expect(w.find('[data-section="down"]').text()).toContain('Кольцо RING-3')
    expect(w.find('[data-section="lots"]').text()).toContain('Плавка ST-19')
    expect(w.find('[data-section="extracts"] [data-testid="provenance"]').text()).toContain('Происхождение не подтверждено')
    await w.find('[data-section="down"] button').trigger('click')
    expect(w.emitted('open-item')?.[0]).toEqual(['ENT:RING-3'])
  })

  it('зоны: закрыта / устарела после вмешательства; без статуса — не «проверена» (FR-46)', () => {
    const w = mount(PassportZones, { props: { zones: flangePassport().zones }, global: global() })
    expect(w.find('[data-zone="Z-weld-1"]').attributes('data-status')).toBe('none')
    expect(w.find('[data-zone="Z-weld-1"]').text()).not.toContain('Проверена')
    expect(w.find('[data-zone="Z-bore"]').text()).toContain('Закрыта')
    expect(w.find('[data-zone="Z-seal"]').text()).toContain('Устарела после вмешательства')
    expect(w.find('[data-zone="Z-seal"] [data-testid="intervention"]').exists()).toBe(true)
  })

  it('носители: временная бирка снята, DataMatrix считан (AD-16)', () => {
    const w = mount(PassportCarriers, { props: { carriers: flangePassport().carriers }, global: global() })
    const items = w.findAll('li')
    expect(items[0]!.text()).toContain('По метке')
    expect(items[0]!.text()).toContain('Снят')
    expect(items[0]!.text()).toContain('временный')
    expect(items[1]!.text()).toContain('По коду DataMatrix')
    expect(items[1]!.text()).toContain('Считан')
  })
})

describe('вид паспорта', () => {
  it('компактный: шапка с осями статуса, последние записи и «Открыть паспорт»', async () => {
    const w = mount(ItemPassportView, { props: { passport: flangePassport(), view: 'compact' }, global: global() })
    expect(w.find('[data-testid="axes"]').text()).toContain('В изоляции')
    expect(w.findAll('li.row')).toHaveLength(5)
    await w.find('[data-testid="open-full"]').trigger('click')
    expect(w.emitted('open-full')).toHaveLength(1)
  })

  it('полный: несоответствия изделия открываются карточкой, документы посчитаны (FR-7, FR-65)', async () => {
    const w = mount(ItemPassportView, { props: { passport: flangePassport(), view: 'full', canOpenNc: true }, global: global() })
    expect(w.find('[data-testid="documents-from-history"]').text()).toBe('Документов собрано из истории: 1 — вручную не понадобилось')
    await w.find('[data-testid="nonconformities"] button').trigger('click')
    expect(w.emitted('open-nc')?.[0]).toEqual(['NC-0142'])
  })
})

describe('контейнер паспорта на кэше запроса (AD-21)', () => {
  it('изделие из среза (окно записи), состояние рамки — по оси качества, режим данных', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity } } })
    const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/', component: { template: '<div />' } }] })
    await router.push('/')
    queryClient.setQueryData(itemKeys.one('ENT:FL-0042', 'passport', { axis: 'occurred' }), {
      data: flangePassport(),
      status: 200,
      headers: new Headers({ 'Ant-Backend': 'fixtures' }),
    })
    const w = mount(ItemPassportWidget, {
      props: { widgetId: 'item-passport', titleKey: 'desks.passport', slotId: 'passport', slice: { item_id: 'ENT:FL-0042', view: 'compact' }, density: 'comfortable' },
      global: { plugins: [pinia, i18n, router, [VueQueryPlugin, { queryClient }]] },
    })
    await flushPromises()
    const frame = w.find('.widget-frame')
    expect(frame.attributes('data-state')).toBe('defect_indication')
    expect(frame.attributes('data-mode')).toBe('fixtures')
    expect(w.find('[data-testid="item-passport"]').attributes('data-view')).toBe('compact')
  })

  it('изделие не выбрано — «выберите изделие», а не пустой паспорт', async () => {
    const queryClient = new QueryClient()
    const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/', component: { template: '<div />' } }] })
    await router.push('/')
    const w = mount(ItemPassportWidget, {
      props: { widgetId: 'item-passport', titleKey: 'desks.passport', slotId: 'passport', slice: {}, density: 'comfortable' },
      global: { plugins: [pinia, i18n, router, [VueQueryPlugin, { queryClient }]] },
    })
    await flushPromises()
    expect(w.text()).toContain('Выберите изделие в очереди')
  })
})
