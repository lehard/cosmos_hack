// Раздел «Аналитика» (кейс §2.4, §5.2; FR-86…FR-89): дефекты и изделия с
// дефектами раздельно; входной брак, оборудование, исполнители, гипотезы —
// раздельно; происхождение времени; сравнение сопоставимых работ; любое число
// раскрывается (FR-7).
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mockApi, mountWidget } from '@/entities/incident/__tests__/api-mock'
import { overview } from '@/entities/metric/__tests__/fixtures'
import { useMetricFocusStore } from '@/entities/metric'
import AnalyticsOverviewWidget from '../ui/AnalyticsOverviewWidget.vue'

const props = { widgetId: 'analytics-overview', titleKey: 'desks.analytics' }
const mountOverview = async () => {
  mockApi({ 'GET /api/v1/analytics': overview() })
  return mountWidget(AnalyticsOverviewWidget, props)
}

describe('раздел «Аналитика»', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('дефекты и изделия с дефектами — две колонки', async () => {
    const w = await mountOverview()
    const defects = w.find('[data-column="defects"]')
    const items = w.find('[data-column="items"]')
    expect(defects.find('[data-metric="confirmed_defects"] [data-testid="total"]').text()).toBe('5')
    expect(items.find('[data-metric="items_with_confirmed_nc"] [data-testid="total"]').text()).toBe('3')
    expect(defects.find('[data-metric="items_with_confirmed_nc"]').exists()).toBe(false)
    expect(w.find('[data-section="defects"]').text()).toContain('Три дефекта на одной детали — одно изделие с несоответствием')
  })

  it('раздельный учёт: входной брак отдельно от производственных дефектов; гипотез нет — «не передано», не ноль', async () => {
    const w = await mountOverview()
    expect(w.find('[data-account="incoming"] [data-metric="incoming_defects"]').exists()).toBe(true)
    expect(w.find('[data-column="defects"] [data-metric="incoming_defects"]').exists()).toBe(false)
    expect(w.find('[data-account="equipment"] [data-metric="equipment_downtime"]').exists()).toBe(true)
    expect(w.find('[data-account="people"] [data-metric="confirmed_performer_errors"]').exists()).toBe(true)
    expect(w.find('[data-account="hypotheses"] [data-testid="absent"]').text()).toBe('Не передано сервером')
    expect(w.find('[data-account="incoming"]').text()).toContain('Поставщик-3, партия П-117')
  })

  it('«оценка невозможна» — отдельная корзина в проверках', async () => {
    const w = await mountOverview()
    const s = w.find('[data-section="inspection"]')
    expect(s.find('[data-metric="unable_to_assess"]').exists()).toBe(true)
    expect(s.text()).toContain('Оценка невозможна — не годно и не брак')
  })

  it('время: плашек у чисел нет — происхождение и интервал в подсказке значка у заголовка; без пометки — предупреждение', async () => {
    const w = await mountOverview()
    const s = w.find('[data-section="time"]')
    expect(s.text()).not.toContain('Вычислено системой')
    expect(s.text()).not.toContain('Передано источником')
    expect(s.find('[data-meaning]').exists()).toBe(false)
    const notes = (id: string) => s.find(`[data-card="${id}"] [data-testid="metric-notes"]`)
    expect(notes('lead_time').attributes('aria-label')).toContain('Время вычислено системой')
    expect(notes('weld_cycle').attributes('aria-label')).toContain('Время передано источником')
    expect(notes('waiting_raw').attributes('data-warn')).toBeDefined()
    expect(w.find('[data-account="equipment"] [data-testid="metric-notes"]').attributes('aria-label')).toContain('Время вычислено системой')
  })

  it('один показатель в карточке — один заголовок: название показателя не повторяется, пояснение под числом', async () => {
    const w = await mountOverview()
    const incoming = w.find('[data-account="incoming"]')
    expect(incoming.find('[data-testid="card-title"]').text()).toBe('Входной брак')
    expect(incoming.find('[data-testid="metric-title"]').exists()).toBe(false)
    expect(incoming.text()).not.toContain('отдельно от производственных ошибок')
    expect(incoming.find('[data-testid="metric-notes"]').attributes('aria-label')).toContain('Входной брак (отдельно от производственных ошибок)')
    // Разбивка — таблица на всю ширину карточки: название среза и число в отдельных ячейках.
    expect(w.find('[data-account="equipment"] .slices tbody th').exists()).toBe(true)
  })

  it('сравнение сопоставимых работ — таблица по исполнителям, не рейтинг', async () => {
    const w = await mountOverview()
    const s = w.find('[data-section="comparison"]')
    expect(s.text()).toContain('Это не рейтинг людей')
    expect(s.text()).toContain('Участвовал — не значит стал причиной')
    expect(s.find('tr[data-slice="W21"]').text()).toContain('22')
    expect(s.find('tr[data-slice="W22"]').findAll('td')[1]!.text()).toBe('—')
  })

  it('любое число — итог или срез — выбирается для раскрытия', async () => {
    const w = await mountOverview()
    const focus = useMetricFocusStore()
    await w.find('[data-metric="confirmed_defects"] [data-testid="total"]').trigger('click')
    expect(focus.pick).toEqual({ metricId: 'confirmed_defects', title: 'Подтверждённые дефекты (производственные)' })
    await w.find('[data-metric="confirmed_defects"] tr[data-slice="porosity"] button').trigger('click')
    expect(focus.pick).toMatchObject({ metricId: 'confirmed_defects', sliceKey: 'porosity', sliceLabel: 'Пористость' })
    expect(w.find('[data-metric="confirmed_defects"] tr[data-slice="porosity"] button').attributes('data-picked')).toBeDefined()
    await w.find('[data-section="comparison"] tr[data-slice="W21"] td button').trigger('click')
    expect(focus.pick).toMatchObject({ metricId: 'comparable_welds', sliceKey: 'W21' })
  })

  it('период и запись журнала, на которой построены показатели', async () => {
    const w = await mountOverview()
    expect(w.find('[data-testid="period-range"]').text()).toContain('по записи журнала № 1402')
  })
})
