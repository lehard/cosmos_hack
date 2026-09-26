// «Камеры + ИИ» в карточке НС (разбор ресерчера): схема шва по участкам из данных,
// окно возникновения признака по записям контроля, покрытие методами, три исхода.
import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { describe, expect, it, vi } from 'vitest'
import { i18n } from '@/shared/i18n'
import { at } from '@/entities/item/__tests__/fixtures'
import { emergenceWindow, observationEventId, type NCCard } from '@/entities/nonconformity'
import { ncCard } from '@/entities/nonconformity/__tests__/fixtures'
import type { InspectionCoverage } from '@/shared/api/generated/model'
import CoverageMap from '../ui/CoverageMap.vue'
import NcCardView from '../ui/NcCardView.vue'

// Дорожка решения (Ф1) ходит за наблюдением и паспортом — здесь заглушка; сама дорожка проверена в features/decision-trace.
vi.mock('@/features/decision-trace', async () => {
  const { defineComponent, h } = await import('vue')
  return { NcDecisionTrace: defineComponent({ props: { card: { type: Object, required: true } }, setup: () => () => h('div', { 'data-testid': 'nc-decision-trace' }) }) }
})

const withAfter = (): NCCard => {
  const card = ncCard()
  card.happened.after = [
    { event_id: 'e-kt3a', event_type: 'inspection.result.recorded', kind: 'fact', occurred_at: at('08:52'), summary: 'КТ-3 камера: признак — пора', source_kind: 'camera', params: { outcome: 'defect_indicated' } },
    { event_id: 'e-kt3b', event_type: 'inspection.result.recorded', kind: 'fact', occurred_at: at('08:53'), summary: 'КТ-3 ракурс 2: пора', source_kind: 'camera', params: { outcome: 'defect_indicated' } },
  ]
  return card
}
const g = { plugins: [createPinia(), i18n], stubs: { EvidenceMaterial: true, CameraObservation: true } }
const zones = [
  { zone_id: 'Z-weld-0', name: 'Шов 1, участок 0–40 мм', status: 'inspected' },
  { zone_id: 'Z-weld-1', name: 'Шов 1, участок 40–80 мм', status: 'inspected' },
  { zone_id: 'Z-weld-2', name: 'Шов 1, участок 80–120 мм' },
]

describe('камера в карточке НС', () => {
  it('окно возникновения: последняя чистая проверка → первая с признаком; наблюдение сигнала', () => {
    const card = withAfter()
    const w = emergenceWindow(card)
    expect(w?.clean?.event_id).toBe('e-kt2')
    expect(w?.first.event_id).toBe('e-kt3a')
    expect(observationEventId(card)).toBe('e-kt3a')
    expect(emergenceWindow(ncCard())).toBeNull()
  })

  it('в карточке: схема шва с подсвеченным участком из данных, окно возникновения словами, легенда трёх исходов', () => {
    const w = mount(NcCardView, { props: { card: withAfter(), now: Date.parse(at('11:23')), seamZones: zones }, global: g })
    const seam = w.find('[data-testid="seam-scheme"]')
    expect(seam.findAll('.cell')).toHaveLength(3)
    expect(seam.find('[data-zone="Z-weld-1"]').attributes('data-tone')).toBe('hit')
    expect(seam.find('[data-zone="Z-weld-0"]').attributes('data-tone')).toBe('ok')
    expect(seam.find('figcaption').text()).toContain('Шов 1, участок 40–80 мм')
    expect(w.find('[data-testid="emergence"]').text()).toContain('Признак появился между «КТ-2: признаки не обнаружены»')
    expect(w.find('[data-testid="emergence"]').text()).toContain('не причина и не вина')
    expect(w.find('[data-testid="outcome-legend"]').text()).toContain('«Оценка невозможна» — никогда не «годно»')
  })

  it('схемы нет, если участок сигнала не среди участков шва', () => {
    const w = mount(NcCardView, { props: { card: withAfter(), now: 0, seamZones: [zones[0]!] }, global: g })
    expect(w.find('[data-testid="seam-scheme"]').exists()).toBe(false)
  })

  it('покрытие методами: камера получена, рентген ждём → «визуальный контроль пройден, полный ещё нет»; не проверенное методом — отдельно', () => {
    const coverage: InspectionCoverage = {
      item_id: 'ENT:FL-0042',
      basis_seq: 1,
      complete: false,
      points: [
        { step_key: 'welding.kt3_camera', inspection_point: 'KT-3', method: 'camera', required: true, status: 'received', outcome: 'defect_indicated' },
        { step_key: 'welding.kt3_radiography', inspection_point: 'KT-3', method: 'radiography', required: true, status: 'pending' },
      ],
      types: [{ code: 'W-PORE-I', name: 'Внутренние поры', severity: 'major', status: 'not_checked' }],
    }
    const w = mount(CoverageMap, { props: { coverage }, global: { plugins: [i18n] } })
    expect(w.find('[data-method="camera"]').attributes('data-tone')).toBe('danger')
    expect(w.find('[data-method="radiography"]').attributes('data-tone')).toBe('pending')
    expect(w.find('[data-testid="coverage-summary"]').text()).toBe('Визуальный контроль пройден, полный контроль ещё не завершён')
    expect(w.find('[data-testid="coverage-not-checked"]').text()).toContain('Внутренние поры')
  })
})
