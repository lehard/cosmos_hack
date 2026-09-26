// Заготовки пульта тестовых сценариев: сценарий главной истории, прогоны в
// разных состояниях, табло с четырьмя состояниями строк, кнопки стенда.
import type { Board, Injection, Run, Scenario } from '@/shared/api/generated/model'

export const scenario = (over: Partial<Scenario> = {}): Scenario => ({
  scenario_id: 'flange-bad-day',
  version: 'fixtures-1',
  title: 'Плохой день сварочного участка',
  description: 'Главная история демо',
  case_refs: ['§1.5', '§5.1'],
  decisions: 6,
  assertions: 22,
  default_items: 40,
  default_seed: 17,
  ...over,
})

export const scenarios = (): Scenario[] => [
  scenario(),
  scenario({ scenario_id: 'S06', title: 'Повторная доставка', decisions: 0, assertions: 3, default_items: 0, default_seed: 0, case_refs: [] }),
]

export const run = (over: Partial<Run> = {}): Run => ({
  run_id: 'fx-1a2b3c4d',
  scenario_id: 'flange-bad-day',
  scenario_version: 'fixtures-1',
  state: 'running',
  mode: 'interactive',
  seed: 17,
  speed: 60,
  step: 7,
  steps: 26,
  items: 40,
  clock_at: '2026-09-23T07:31:00Z',
  started_at: '2026-09-26T09:00:00Z',
  finished_at: null,
  board_passed: 3,
  board_total: 5,
  basis_seq: 79999,
  ...over,
})

/** Сервер без запущенного прогона: сценарий по умолчанию под своим id. */
export const idleRun = (): Run => run({ run_id: 'flange-bad-day', state: 'paused', step: 16, speed: 1, seed: 0 })

export const board = (over: Partial<Board> = {}): Board => ({
  run_id: 'fx-1a2b3c4d',
  basis_seq: 79999,
  passed: 2,
  failed: 1,
  pending: 1,
  rows: [
    { assertion_id: 'S05-01', title: 'Область RS-01, версия 1 — 34 изделия', operation_id: 'analysis.risk_scope.read', path: '/versions/0/size', step: 10, expected: '34', actual: null, status: 'not_reached' },
    { assertion_id: 'S01-01', title: 'Ф-001 — годно, выпущен', operation_id: 'item.passport.read', path: '/status/summary', step: 7, expected: '"released"', actual: '"released"', status: 'passed' },
    { assertion_id: 'S12-01', title: 'В карантине 3 сообщения с кодами', operation_id: 'ingest.quarantine.list', path: '/items/length', step: 5, expected: '3', actual: '3', status: 'passed' },
    { assertion_id: 'S08-01', title: 'Ложный сигнал КТ-4 отклонён', operation_id: 'quality.signal.read', path: '/state', step: 4, expected: '"rejected"', actual: '"under_review"', status: 'failed' },
    { assertion_id: 'S03-01', title: 'НС-01 подтверждено', operation_id: 'nonconformity.card.read', path: '/status', step: 7, expected: '"confirmed"', actual: null, status: 'pending' },
  ],
  ...over,
})

export const injections = (): Injection[] => [
  { injection: 'duplicate_event', title: 'Прислать повтор события', description: 'Повтор отсеян (S06)', available: true, needs_target: true },
  { injection: 'corrupt_frame', title: 'Испортить кадр', available: true, needs_target: false },
  { injection: 'machine_fault', title: 'Сбой станка: ток вне уставки', available: false, needs_target: false },
  { injection: 'tamper_outside', title: 'Подделать запись в обход системы', available: true, needs_target: false },
]

/** Квитанция команды (AD-7). */
export const receipt = (command_id = 'c-1', events = 1) => ({
  command_id,
  seq: 80001,
  event_ids: Array.from({ length: events }, (_, i) => `ev-${i + 1}`),
  replayed: false,
})
