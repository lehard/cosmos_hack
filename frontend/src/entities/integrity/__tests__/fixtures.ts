// Заготовки состояния компонентов и стола Аудитора ИБ.
import type { CriticalAction, OpsHealth, SecurityEvent, StoppedItem, VerifierReport, VerifierReportSummary } from '@/shared/api/generated/model'

export const health = (over: Partial<OpsHealth> = {}): OpsHealth => ({
  version: '0.9.0',
  profile: 'fixtures',
  mode: 'fixtures',
  components: [
    { component: 'ant/api', state: 'ok', instances: 2, leader: 'api-1', checked_at: '2026-09-26T09:00:00Z' },
    { component: 'projector', state: 'degraded', instances: 1, checked_at: '2026-09-26T09:00:00Z', detail: 'отставание 120 записей' },
    { component: 'keeper', state: 'not_implemented', instances: 0, checked_at: null },
  ],
  queues: [{ name: 'p-3', scope: 'partition', pending: 4, lag_seq: 120 }],
  integrations: [{ system: 'onec', state: 'ok', since: '2026-09-26T08:00:00Z' }],
  quarantine_open: 3,
  stopped_items: 1,
  verifier: { verdict: 'intact', checked_at: '2026-09-26T08:55:00Z', report_ref: 'rep-1' },
  ...over,
})

export const stopped = (): StoppedItem[] => [
  { item_id: 'ENT01:F-031', consumer: 'worker/p-3', error: 'гард: нет версии процесса', failed_at: '2026-09-23T09:00:00Z', failed_seq: 120001, failure_event_id: 'ev-fail-1', retries: 2 },
]

export const reportSummary = (over: Partial<VerifierReportSummary> = {}): VerifierReportSummary => ({
  report_digest: 'sha256:0123456789abcdef0123',
  checked_at: '2026-09-23T13:40:00Z',
  checked_up_to_seq: 229999,
  server_side: true,
  verdict: 'violated',
  verifier_build: 'sha256:feedfacecafebeef00',
  ...over,
})

export const report = (): VerifierReport => ({
  summary: reportSummary(),
  signed_by: 'verifier-key@1',
  virtual_time: true,
  checks: [
    { check: 'цепочки', status: 'rejected', count: 1, ca_ref: 'CA-17', details: 'EV-WS2-0412: хеш не сходится' },
    { check: 'подписи', status: 'intact', count: 2300 },
    { check: 'момент подписи', status: 'unverifiable', count: 3 },
  ],
  signature_classes: { personal: 12, scenario: 2280, paper: 1 },
  paper_decisions: [{ entity: 'document', id: 'DOC-9' }],
})

export const criticalAction = (over: Partial<CriticalAction> = {}): CriticalAction => ({
  ca_ref: 'CA-12',
  ca_no: 12,
  ca_group: 'nc_decision',
  action_type: 'decision.nonconformity.confirmed',
  object: { entity: 'nonconformity', id: 'NC-01' },
  object_ref: 'nonconformity:NC-01',
  before: 'сигнал',
  after: 'подтверждено',
  actor_id: 'P-QC-1',
  authority_id: 'qc_acceptance',
  stamp_id: 'ST-7',
  basis_event_ids: ['ev-sig-1', 'ev-obs-2'],
  main_event_id: 'ev-dec-1',
  main_commit: 'sha256:c0ffee',
  policy_seq: 7,
  recorded_at: '2026-09-23T08:08:00Z',
  ...over,
})

export const securityEvents = (): SecurityEvent[] => [
  { event_id: 'se-1', seq: 220001, event_type: 'security.integrity.violated', severity: 'alarm', occurred_at: '2026-09-23T13:35:00Z', summary: 'Подмена записи EV-WS2-0412', object: { entity: 'integrity', id: 'global' }, ca_ref: 'CA-17' },
  { event_id: 'se-2', seq: 90001, event_type: 'security.auth.failed', severity: 'warning', occurred_at: '2026-09-23T08:00:00Z', summary: 'Неверный пароль', source_id: 'web' },
]
