// Заготовки приёма для стола администратора: источники, каналы обмена,
// метрики приёма, записи карантина (кейс §5.1 S12 — три сообщения с кодами).
import type { ErpChannel, IngestMetrics, QuarantineEntry, SourceView } from '@/shared/api/generated/model'

export const sources = (): SourceView[] => [
  { source_id: 'WS-2', source_kind: 'станок', state: 'active', key_ref: 'dev-ws2@1', last_received_at: '2026-09-23T09:06:00Z', last_seq: 1240, gap_count: 0, quarantined: 0, clock_skew_ms: 120 },
  { source_id: 'GW-2', source_kind: 'шлюз', state: 'loss_suspected', key_ref: 'dev-gw2@1', last_received_at: '2026-09-22T06:50:00Z', last_seq: 312, gap_count: 35, quarantined: 1 },
  { source_id: 'CAM-9', source_kind: 'камера', state: 'disabled', last_received_at: null, last_seq: null, gap_count: 0, quarantined: 0 },
]

export const channels = (): ErpChannel[] => [
  { system: 'onec', state: 'ok', endpoint: 'stand://1c', stand: true, contract_version: '1', last_exchange_at: '2026-09-23T07:31:00Z', queued: 0, quarantined: 0 },
  { system: 'mes', state: 'degraded', endpoint: 'stand://mes', stand: true, contract_version: '1', last_exchange_at: null, queued: 3, quarantined: 1, detail: 'ответ 503' },
]

export const metrics = (): IngestMetrics => ({
  accepted: 1200,
  received: 1250,
  duplicates: 312,
  rejected: 2,
  quarantined: 3,
  quarantine_open: 3,
  completeness_bp: 9720,
  latency_p50_ms: 40,
  latency_p95_ms: 180,
  event_to_screen_p95_ms: 900,
  window_from: '2026-09-23T08:00:00Z',
  window_to: '2026-09-23T09:00:00Z',
})

export const entry = (over: Partial<QuarantineEntry> = {}): QuarantineEntry => ({
  quarantine_id: 'Q-001',
  source_id: 'MES',
  source_seq: 77,
  event_type: 'operation.operation.finished',
  problem_code: 'ingest.missing_required_field',
  detail: 'нет поля item_id',
  fingerprint: 'sha256:abcd',
  material_address: 'cas://sha256/abcd',
  quarantined_at: '2026-09-22T11:00:00Z',
  state: 'open',
  ...over,
})

export const entries = (): QuarantineEntry[] => [
  entry(),
  entry({ quarantine_id: 'Q-002', problem_code: 'ingest.unknown_schema_version', detail: undefined, event_type: undefined }),
  entry({ quarantine_id: 'Q-003', problem_code: 'ingest.unknown_enum_value_critical', state: 'accepted' }),
]
