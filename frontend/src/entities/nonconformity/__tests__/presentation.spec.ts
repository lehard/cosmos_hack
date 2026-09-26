// Решение на точке предъявления и пересмотр (FR-19, FR-32, Д-81, UI-28): тело
// команды — из выбранного действия сервера и ответа чтения; basis_seq — из
// ответа; пересмотр — с пересматриваемой записью и рассмотренными фактами.
import { describe, expect, it } from 'vitest'
import { buildPresentationCommand, presentationEventType, presentationSummary, reasonRequired, type NCPresentationAction, type NCPresentationView } from '@/entities/nonconformity'

const view = (over: Partial<NCPresentationView> = {}): NCPresentationView => ({
  item_id: 'ENT01:F-015',
  item_label: 'Ф-015',
  basis_seq: 170010,
  method_results: [],
  presentation: { event_id: 'p-1', step_key: 'welding.zt3_acceptance', closing_point: 'ZT-3', closing_point_label: 'ЗТ-3 · приёмка сварки', presentation_no: 1, method_event_ids: ['m-1', 'm-2'], allowed_resolutions: ['accept', 'reject'] },
  ...over,
})
const accept: NCPresentationAction = { operation: 'nonconformity.presentation.resolve', resolution: 'accept', label: 'Принять — передать на сборку', allowed: true, why_available: 'Все обязательные методы пройдены', consequences: ['Изделие уйдёт на сборку'] }
const revoke: NCPresentationAction = { operation: 'nonconformity.presentation.review', outcome: 'revoked', label: 'Отозвать приёмку', allowed: true, why_available: 'Решение принято до новых данных', consequences: ['Блок изделия', 'Исправление в 1С'] }
const meta = { command_id: '0190-c', policy_seq: 262, workplace_id: 'WP-QC' }

describe('команды на точке предъявления', () => {
  it('решение на точке: точка, номер, методы и basis_seq — из ответа чтения', () => {
    const cmd = buildPresentationCommand(accept, view(), meta, '')
    expect(cmd.kind).toBe('resolve')
    expect(cmd.body).toMatchObject({ step_key: 'welding.zt3_acceptance', closing_point: 'ZT-3', presentation_no: 1, method_event_ids: ['m-1', 'm-2'], resolution: 'accept', basis_seq: 170010, policy_seq: 262, workplace_id: 'WP-QC' })
    expect(cmd.body).not.toHaveProperty('reason')
    expect(presentationEventType(accept)).toBe('decision.presentation.resolved')
    expect(reasonRequired(accept)).toBe(false)
  })

  it('пересмотр: исход, пересматриваемая запись, рассмотренные новые факты, основание обязательно', () => {
    const v = view({
      review: {
        decision: { event_id: 'd-zt3', event_type: 'decision.presentation.resolved', kind: 'decision', occurred_at: '2026-09-23T12:30:00Z', summary: 'Принято при неполных данных' },
        known_at_decision: [],
        new_facts: [{ event_id: 'ev-ws2', event_type: 'equipment.deviation.detected', kind: 'fact', occurred_at: '2026-09-23T13:05:00Z', summary: 'ИС-2: ток 176 А' }],
      },
    })
    const cmd = buildPresentationCommand(revoke, v, meta, ' Ток вне уставки ')
    expect(cmd.kind).toBe('review')
    expect(cmd.body).toMatchObject({ outcome: 'revoked', reviewed_event_id: 'd-zt3', new_fact_ids: ['ev-ws2'], reason: { text: 'Ток вне уставки' }, basis_seq: 170010 })
    expect(presentationEventType(revoke)).toBe('decision.presentation.reviewed')
    expect(reasonRequired(revoke)).toBe(true)
  })

  it('сводка подписи: действие, изделие, точка, последствия', () => {
    const s = presentationSummary(revoke, view(), 'Ток вне уставки')
    expect(s.map((f) => f.value ?? f.valueKey)).toEqual(['Отозвать приёмку', 'Ф-015', 'ЗТ-3 · приёмка сварки', 'Ток вне уставки', 'Блок изделия; Исправление в 1С', 'decisions.signature.level2'])
  })
})
