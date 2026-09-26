// Образцы экрана «Интеграции» (эпик 48) — только для тестов.
import type { IntegrationList } from '@/shared/api/generated/model'

export const integrations = (over: Partial<IntegrationList> = {}): IntegrationList => ({
  profile: 'demo',
  items: [
    {
      system: 'onec', installed: true, state: 'stand', default: true, stand_available: true, real_available: false, channel: 'ok',
      endpoint: 'http://stands:8090/onec', last_exchange_at: '2026-09-26T08:00:00Z', queued: 2, quarantined: 1, basis_seq: 0,
      last_error: { at: '2026-09-26T07:00:00Z', detail: 'HTTP 422: не найден договор' },
    },
    { system: 'galaktika', installed: false, state: 'disabled', default: true, stand_available: false, real_available: false, basis_seq: 0 },
    {
      system: 'visionqc', installed: true, state: 'disabled', default: false, stand_available: true, real_available: true, channel: 'disabled', basis_seq: 41,
      last_decision: { state: 'disabled', previous: 'enabled', reason: 'замена камеры', actor: 'adm-01', at: '2026-09-26T06:00:00Z', seq: 41 },
    },
  ],
  ...over,
})
