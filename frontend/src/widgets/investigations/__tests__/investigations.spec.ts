// «Мои расследования» (UI-32): карточки инцидентов, выбранный — в фокусе разбора.
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { useAnalysisFocusStore } from '@/entities/incident'
import { mockApi, mountWidget } from '@/entities/incident/__tests__/api-mock'
import { i18n } from '@/shared/i18n'
import type { IncidentSummary } from '@/shared/api/generated/model'
import InvestigationsView from '../ui/InvestigationsView.vue'
import InvestigationsWidget from '../ui/InvestigationsWidget.vue'

const incident = (id: string, over: Partial<IncidentSummary> = {}): IncidentSummary => ({
  incident_id: id,
  label: `Инцидент ${id}`,
  common_factor: { factor: 'machine', value: 'ИС-2' },
  size: 6,
  initial_size: 34,
  scope_version: 3,
  status: 'open',
  opened_at: '2026-09-23T08:10:00Z',
  ...over,
})

describe('мои расследования', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('открытые сверху, свежие первыми; в карточке — сколько сейчас, сколько было, общий фактор, состояние', () => {
    const w = mount(InvestigationsView, {
      props: {
        incidents: [
          incident('RS-03', { status: 'closed', opened_at: '2026-09-24T08:00:00Z', size: 1, initial_size: 1 }),
          incident('RS-01'),
          incident('RS-02', { opened_at: '2026-09-23T12:00:00Z', size: 3, initial_size: 3 }),
        ],
        selected: 'RS-01',
      },
      global: { plugins: [createPinia(), i18n] },
    })
    const cards = w.findAll('button.card')
    expect(cards.map((c) => c.attributes('data-incident'))).toEqual(['RS-02', 'RS-01', 'RS-03'])
    const rs1 = w.find('[data-incident="RS-01"]')
    expect(rs1.attributes('aria-current')).toBe('true')
    expect(rs1.text()).toContain('6 изделий')
    expect(rs1.find('[data-testid="path"]').text()).toBe('было 34 → сейчас 6')
    expect(rs1.text()).toContain('ИС-2')
    expect(rs1.find('[data-testid="state"]').text()).toContain('Расследование идёт')
    expect(w.find('[data-incident="RS-02"] [data-testid="path"]').exists()).toBe(false)
    expect(w.find('[data-incident="RS-03"] [data-testid="state"]').text()).toContain('Закрыто')
  })

  it('стадия из словаря, счётчики «что известно», «Дальше:» — текст сервера, общий фактор словами', () => {
    const w = mount(InvestigationsView, {
      props: {
        incidents: [
          incident('RS-01', {
            stage: 'hypothesis',
            counts: { confirmed: 1, suspect: 4, unknown: 1, excluded: 28 },
            next_step: 'Контрольный образец на ИС-2 при уставке 160 А',
            common_factor: { factor: 'machine', value: 'IS-2', label: 'Сварочный источник ИС-2' },
          } as Partial<IncidentSummary>),
        ],
        selected: null,
      },
      global: { plugins: [createPinia(), i18n] },
    })
    expect(w.find('[data-testid="stage"]').text()).toBe('Проверка гипотез')
    expect(w.find('[data-testid="counts"]').text()).toContain('1 подтверждено')
    expect(w.find('[data-testid="counts"]').text()).toContain('28 исключено с основанием')
    expect(w.find('[data-testid="next-step"]').text()).toBe('Дальше: Контрольный образец на ИС-2 при уставке 160 А')
    expect(w.text()).toContain('Сварочный источник ИС-2')
    expect(w.text()).not.toContain('IS-2')
  })

  it('виджет: первый открытый инцидент выбран; щелчок — инцидент в фокус разбора', async () => {
    mockApi({ 'GET /api/v1/incidents': { items: [incident('RS-01'), incident('RS-02', { opened_at: '2026-09-23T12:00:00Z' })] } })
    const w = await mountWidget(InvestigationsWidget, { widgetId: 'investigations', titleKey: 'widgets.analysis.investigations.title' })
    expect(w.find('[data-incident="RS-01"]').attributes('aria-current')).toBe('true')
    await w.find('[data-incident="RS-02"]').trigger('click')
    setActivePinia(w.vm.$pinia)
    expect(useAnalysisFocusStore().incidentId).toBe('RS-02')
    expect(w.find('[data-incident="RS-02"]').attributes('aria-current')).toBe('true')
  })
})
