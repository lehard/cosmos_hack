// «Мои расследования» (UI-32): карточки инцидентов, выбранный — в фокусе разбора.
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { useAnalysisFocusStore } from '@/entities/incident'
import { mockApi, mountWidget, permissions, settle } from '@/entities/incident/__tests__/api-mock'
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
  nc_ids: [],
  close_blockers: [],
  counts: { confirmed: 0, suspect: 0, unknown: 0, excluded: 0 },
  stage: 'scope_defined',
  next_step: null,
  ...over,
})

describe('мои расследования', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('выбранное — шапкой (сколько сейчас, сколько было, фактор, состояние); остальные — переключателями, открытые и свежие первыми', async () => {
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
    const rs1 = w.find('section.hero')
    expect(rs1.attributes('data-incident')).toBe('RS-01')
    expect(rs1.attributes('aria-current')).toBe('true')
    expect(rs1.text()).toContain('6 изделий')
    expect(rs1.find('[data-testid="path"]').text()).toBe('было 34 → сейчас 6')
    expect(rs1.text()).toContain('ИС-2')
    expect(rs1.find('[data-testid="state"]').text()).toContain('Расследование идёт')
    expect(w.findAll('button.other').map((c) => c.attributes('data-incident'))).toEqual(['RS-02', 'RS-03'])
    expect(w.find('button.other[data-incident="RS-03"]').text()).toContain('Закрыто')
    await w.find('button.other[data-incident="RS-02"]').trigger('click')
    expect(w.emitted('select')?.[0]).toEqual(['RS-02'])
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
    expect(w.find('[data-testid="next-block"]').text()).toContain('Что делать дальше')
    expect(w.find('[data-testid="next-step"]').text()).toBe('Контрольный образец на ИС-2 при уставке 160 А')
    expect(w.text()).toContain('Сварочный источник ИС-2')
    expect(w.text()).not.toContain('IS-2')
  })

  it('открытое расследование: почему ещё нельзя закрыть — словами сервера', () => {
    const w = mount(InvestigationsView, {
      props: {
        incidents: [incident('RS-01', { close_blockers: [{ code: 'incident.cause_branch_open', text: 'Нет вывода о причине «почему не остановили раньше»' }] } as Partial<IncidentSummary>)],
        selected: null,
      },
      global: { plugins: [createPinia(), i18n] },
    })
    expect(w.find('[data-testid="close-blockers"]').text()).toContain('Почему ещё нельзя закрыть')
    expect(w.find('[data-testid="close-blockers"]').text()).toContain('«почему не остановили раньше»')
  })

  it('виджет: несоответствия расследования — ссылками с номером; щелчок — расследование и несоответствие в фокусе', async () => {
    mockApi({
      'GET /api/v1/incidents': { items: [incident('RS-01', { nc_ids: ['NC-03'] } as Partial<IncidentSummary>)] },
      'GET /api/v1/nonconformities/NC-03': { nc_id: 'NC-03', number: 'НС-03', item_label: 'Ф-003', status: 'isolated' },
    })
    const w = await mountWidget(InvestigationsWidget, { widgetId: 'investigations', titleKey: 'widgets.analysis.investigations.title' })
    const link = w.find('[data-testid="nc-link"]')
    expect(link.text()).toContain('НС-03')
    expect(link.text()).toContain('Ф-003')
    await link.trigger('click')
    setActivePinia(w.vm.$pinia)
    expect(useAnalysisFocusStore().incidentId).toBe('RS-01')
    expect(useAnalysisFocusStore().ncId).toBe('NC-03')
  })

  it('виджет: «Что делать дальше» — «Запросить проверку» одним щелчком по гипотезе с next_check', async () => {
    const calls = mockApi({
      'GET /api/v1/incidents': { items: [incident('RS-01', { primary_nc_id: 'NC-01', next_step: 'Контрольный образец на ИС-2' } as Partial<IncidentSummary>)] },
      'GET /api/v1/nonconformities/NC-01/hypotheses': {
        nc_id: 'NC-01', version: 3, missing_information: [], conclusion_is_categorical: false, similar_cases: [], basis_seq: 150010,
        hypotheses: [{ hypothesis_id: 'H1', category: 'equipment', status: 'proposed_by_system', confidence_bp: 8500, supporting: [], contradicting: [], history: [], next_check: { text: 'Контрольный образец на ИС-2 при 160 А', measurement_kind: 'control_sample', unlocks_text: '', could_exclude: 0, scope_size: 6 } }],
      },
      'GET /api/v1/permissions': permissions([['analysis.measurement.request', 'nonconformity']], 9),
      'POST /api/v1/nonconformities/NC-01/measurements': { command_id: 'c', seq: 1, event_ids: ['e'], replayed: false },
    })
    const w = await mountWidget(InvestigationsWidget, { widgetId: 'investigations', titleKey: 'widgets.analysis.investigations.title' })
    await w.find('[data-testid="cta-request-check"]').trigger('click')
    await settle()
    const post = calls.find((c) => c.method === 'POST')
    expect(post?.path).toBe('/api/v1/nonconformities/NC-01/measurements')
    expect(post?.body).toMatchObject({ hypothesis_id: 'H1', what: 'Контрольный образец на ИС-2 при 160 А', basis_seq: 150010, policy_seq: 9 })
    expect(w.find('[data-testid="cta-sent"]').exists()).toBe(true)
  })

  it('виджет: первый открытый инцидент выбран; щелчок — инцидент в фокус разбора', async () => {
    mockApi({ 'GET /api/v1/incidents': { items: [incident('RS-01'), incident('RS-02', { opened_at: '2026-09-23T12:00:00Z' })] } })
    const w = await mountWidget(InvestigationsWidget, { widgetId: 'investigations', titleKey: 'widgets.analysis.investigations.title' })
    expect(w.find('section.hero').attributes('data-incident')).toBe('RS-01')
    await w.find('button.other[data-incident="RS-02"]').trigger('click')
    setActivePinia(w.vm.$pinia)
    expect(useAnalysisFocusStore().incidentId).toBe('RS-02')
    expect(w.find('section.hero').attributes('data-incident')).toBe('RS-02')
  })
})
