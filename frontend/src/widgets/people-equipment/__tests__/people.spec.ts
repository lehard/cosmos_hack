// Люди и оборудование участка (PRD §3a, FR-6, FR-17, FR-80, FR-83, FR-84, FR-149):
// назначен / на месте / ключ, допуск и квалификация; состояние оборудования,
// предупреждения, ресурс инструмента, срок поверки; только свой цех.
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mockApi, mountWidget } from '@/entities/run/__tests__/api'
import { assignments, equipment, foremanSession, locations, posts, qualifications, registry } from '@/entities/workplace/__tests__/fixtures'
import { verificationOf } from '../model/people'
import PeopleEquipmentWidget from '../ui/PeopleEquipmentWidget.vue'

afterEach(() => vi.unstubAllGlobals())
const props = { widgetId: 'people-equipment', titleKey: 'desks.peopleAndEquipment', density: 'large' }

/** Станок механического цеха — не должен попасть на стол мастера сварки. */
const cnc = { ...equipment()[0]!, equipment_id: 'CNC-1', title: 'Станок ЧПУ', station_id: 'WP-CNC-1', warnings: [], tool_life_used: null, tool_life_limit: null, current_run_id: undefined }

const routes = () => ({
  'GET /api/v1/auth/session': foremanSession(),
  'GET /api/v1/reference/locations': { items: locations() },
  'GET /api/v1/workplaces': { items: posts() },
  'GET /api/v1/assignments': assignments(),
  'GET /api/v1/qualifications': { items: qualifications() },
  'GET /api/v1/equipment': { items: [...equipment(), cnc] },
  'GET /api/v1/reference/equipment': { items: registry() },
})

describe('люди и оборудование', () => {
  it('люди: назначен, на месте ли, допуск, квалификация', async () => {
    mockApi(routes())
    const w = await mountWidget(PeopleEquipmentWidget, props)
    const w2 = w.find('[data-testid="people"] [data-workplace="WP-WELD-2"]')
    expect(w2.text()).toContain('Сварщик W21')
    expect(w2.text()).toContain('По графику на месте — ключ не вставлен')
    expect(w2.find('[data-testid="admission"]').text()).toBe('Нет допуска к рабочему месту')
    expect(w2.find('[data-qualification="valid"]').text()).toBe('Аттестация НАКС: действует до 01.01.2027')
    expect(w.find('[data-testid="people"] [data-workplace="WP-QC-WC"]').text()).toContain('Никто не назначен')
  })

  it('оборудование: работает ли, ресурс инструмента, предупреждения, поверка; только свой цех', async () => {
    mockApi(routes())
    const w = await mountWidget(PeopleEquipmentWidget, props)
    const is1 = w.find('[data-equipment="IS-1"]')
    expect(is1.find('[data-testid="execution"]').text()).toBe('Работает')
    expect(is1.find('[data-testid="tool-life"]').text()).toBe('Ресурс инструмента: 73 из 75')
    expect(is1.find('[data-testid="equipment-warning"]').text()).toContain('ресурс инструмента 73/75')
    expect(is1.find('[data-testid="verification"]').text()).toBe('Поверка не требуется')
    const is2 = w.find('[data-equipment="IS-2"]')
    expect(is2.find('[data-testid="execution"]').text()).toBe('Нет данных — неизвестно')
    expect(w.find('[data-equipment="TW-9"] [data-testid="verification"]').text()).toBe('Срок поверки истёк 01.09.2026')
    expect(w.find('[data-equipment="CNC-1"]').exists()).toBe(false)
    // Источник без данных — «оценка невозможна», а не «норма» (NFR-UI-4).
    expect(w.find('.widget-frame').attributes('data-state')).toBe('unable_to_assess')
  })

  it('поверка: справочник — источник правды; нет записи — по журналам; ничего — неизвестно', () => {
    const state = equipment()[0]!
    expect(verificationOf({ id: 'X', title: 'X', state: null, registry: { ...registry()[2]!, usable: false, unusable_reason: 'not_verified' } })).toEqual({ kind: 'unusable', reason: 'not_verified' })
    expect(verificationOf({ id: 'X', title: 'X', state, registry: null })).toEqual({ kind: 'valid', until: '2027-01-31' })
    expect(verificationOf({ id: 'X', title: 'X', state: { ...state, verification: { status: 'unknown' } }, registry: null })).toEqual({ kind: 'unknown' })
  })

  it('ничего не отвечает — «ошибка входа»', async () => {
    mockApi({ 'GET /api/v1/auth/session': foremanSession() })
    const w = await mountWidget(PeopleEquipmentWidget, props)
    expect(w.find('.widget-frame').attributes('data-state')).toBe('input_error')
  })
})
