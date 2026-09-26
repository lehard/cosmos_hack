// «Надёжность данных» (разбор роли администратора): нет данных ≠ всё нормально —
// пропуски, карантин, недоступность внешней системы и приостановка VisionQC видны
// прямо, со ссылкой в свой раздел; всё в норме — одной строкой.
import { describe, expect, it } from 'vitest'
import type { AdaptationEscape, AnalyzerSummary, IntegrationEntry, SourceView } from '@/shared/api/generated/model'
import { areaSummary, integrationIssues, overallTone, sourceIssues, visionIssues } from '../model/reliability'

const src = (over: Partial<SourceView>): SourceView => ({ basis_seq: 1, gap_count: 0, last_received_at: null, last_seq: 1, quarantined: 0, source_id: 'S', source_kind: 'machine', state: 'active', ...over })
const integ = (over: Partial<IntegrationEntry>): IntegrationEntry =>
  ({ basis_seq: 1, default: true, installed: true, real_available: true, stand_available: true, state: 'enabled', system: 'onec', ...over }) as IntegrationEntry

describe('надёжность данных', () => {
  it('источник: пропуски последовательности — опасно; карантин — замечание, ведёт в «Карантин»; выключенный — не считается', () => {
    const issues = sourceIssues([src({ source_id: 'IS-2', gap_count: 7, state: 'loss_suspected' }), src({ source_id: 'CAM', quarantined: 2 }), src({ source_id: 'OFF', state: 'disabled', gap_count: 3 })])
    expect(issues.map((x) => [x.key, x.tone, x.tab])).toEqual([
      ['s:IS-2:gap', 'danger', 'sources'],
      ['s:CAM:q', 'warn', 'quarantine'],
    ])
    expect(areaSummary('sources', 2, issues)).toMatchObject({ ok: 0, total: 2, tone: 'danger' })
  })

  it('внешняя система недоступна — опасно, с очередью; окно записи интеграции', () => {
    const issues = integrationIssues([integ({ last_check: { at: 'x', result: 'unreachable', seq: 1 }, queued: 7 }), integ({ system: 'mes', state: 'disabled', queued: 5 })], (s) => (s === 'onec' ? '1С' : s))
    expect(issues).toHaveLength(1)
    expect(issues[0]).toMatchObject({ tone: 'danger', text: 'dataReliability.issue.unreachable', params: { system: '1С', n: 7 }, tab: 'integrations', open: 'integration:onec' })
  })

  it('VisionQC: приостановленный паспорт — опасно, открывает паспорт; изделия на перепроверку — по уникальным изделиям', () => {
    const analyzers = [{ analyzer_id: 'vqc', kind: 'visionqc', status: 'suspended', title: 'VisionQC сварки', passport_id: 'AP-1' }] as AnalyzerSummary[]
    const escapes = [
      { defect_id: 'D', event_id: 'E1', item_id: 'I', method_covers_defect: true, missed: [], recorded_at: 'x', recheck: [{ item_id: 'A', last_at: 'x', observation_event_ids: [] }, { item_id: 'B', last_at: 'x', observation_event_ids: [] }] },
      { defect_id: 'D2', event_id: 'E2', item_id: 'I', method_covers_defect: true, missed: [], recorded_at: 'x', recheck: [{ item_id: 'B', last_at: 'x', observation_event_ids: [] }] },
    ] as AdaptationEscape[]
    const issues = visionIssues(analyzers, escapes)
    expect(issues[0]).toMatchObject({ tone: 'danger', open: 'passport:AP-1', tab: 'vision-adaptation' })
    expect(issues[1]).toMatchObject({ tone: 'warn', params: { n: 2 } })
    expect(overallTone(issues)).toBe('danger')
    expect(overallTone([])).toBe('ok')
  })
})
