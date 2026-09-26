// Образцы данных карточки несоответствия, очереди и разрешений — только в тестах
// (PRD §11.10). Сценарий: фланец FL-0042, сигнал о поре в шве с КТ-3 (FR-38),
// вывод системы пересмотрен после позднего события журнала оборудования (FR-32).
import type { Concession, DecisionQueueRow, NCCard } from '@/entities/nonconformity'
import { at } from '@/entities/item/__tests__/fixtures'

export const ncCard = (over: Partial<NCCard> = {}): NCCard => ({
  nc_id: 'NC-0142',
  number: 'НС-0142',
  item_id: 'ENT:FL-0042',
  item_label: 'FL-0042',
  basis_seq: 1260,
  status: 'draft',
  axes: { position: 'isolated', quality: 'signal', disposition: 'none', containment: 'item_hold', erp_accounting: 'accepted_into_work' },
  happened: {
    before: [
      { event_id: 'e-kt2', event_type: 'inspection.result.recorded', kind: 'fact', occurred_at: at('07:55'), summary: 'КТ-2: признаки не обнаружены', source_kind: 'camera', seq: 1201 },
    ],
    operation: { operation_run_id: 'run-weld-0042', label: 'Сварка', step_key: 'weld', equipment_id: 'ИС-3', tool_id: 'Горелка Г-2', program_ref: 'P-17 rev.4', started_at: at('08:10'), finished_at: at('08:40') },
    during: [
      { event_id: 'e-current', event_type: 'equipment.deviation.detected', kind: 'fact', occurred_at: at('08:21'), summary: 'Вне уставки: ток 212 А при уставке 180 А', source_kind: 'machine', seq: 1220 },
      { event_id: 'e-override', event_type: 'operator.override.performed', kind: 'fact', occurred_at: at('08:24'), summary: 'Ручное изменение режима: подача 130 %', source_kind: 'manual_entry', author: 'op-12', seq: 1222 },
    ],
    after: [],
  },
  evidence: {
    signals: [
      {
        signal_id: 'SIG-77',
        basis_kind: 'inspection_result',
        defect_type_code: 'W-POR',
        defect_type_known: true,
        severity: 'major',
        zone_id: 'Z-weld-1',
        analyzer_confidence_bp: 8700,
        observation_quality_bp: 9100,
        stages: [
          { stage: 'Где дефект', version: 'loc-2.1', confidence_bp: 9300, output_note: 'шов 1, 40 мм' },
          { stage: 'Какой вид', version: 'cls-1.4', confidence_bp: 8700, output_note: 'пора' },
        ],
        versions: { recipe_ref: 'KT3-weld@5', analyzer_version: 'vqc-3.2', contract_version: 'v1' },
        evidence_refs: ['streebog256:beef01'],
        record: { event_id: 'e-kt3', event_type: 'inspection.result.recorded', kind: 'fact', occurred_at: at('08:52'), summary: 'КТ-3, камера: обнаружен признак дефекта', source_kind: 'camera', seq: 1240 },
      },
    ],
    requirement: { characteristic: 'Поры в шве', tolerance: 'не допускаются', kd_ref: 'ФЛ-100.02 п. 4.3 rev.C' },
    zone_history: [
      { event_id: 'e-kt2', event_type: 'inspection.result.recorded', kind: 'fact', occurred_at: at('07:55'), summary: 'КТ-2: признаки не обнаружены', source_kind: 'camera' },
    ],
    similar_count: 3,
  },
  system_analysis: {
    versions: [
      { version: 1, event_id: 'r-1', outcome: 'manual_review', rule_id: 'RM-weld-7', automation_mode: 1, recorded_at: at('08:53'), revised_due_to: null, causes: [] },
      {
        version: 2,
        event_id: 'r-2',
        outcome: 'isolate',
        rule_id: 'RM-weld-7',
        automation_mode: 2,
        recorded_at: at('09:10'),
        revised_due_to: 'e-late-log',
        causes: [{ event_id: 'e-current', event_type: 'equipment.deviation.detected', kind: 'fact', occurred_at: at('08:21'), summary: 'Вне уставки: ток 212 А при уставке 180 А', source_kind: 'machine' }],
      },
    ],
    why: ['Тяжесть «значительный» при уверенности 0,87 — строка 12 карты реакций'],
    alternatives: ['Блик на кромке шва'],
    missing_information: ['tool_unknown', 'no_observation_after_operation'],
  },
  human_decisions: [
    { event_id: 'd-1', event_type: 'decision.recheck.requested', kind: 'decision', occurred_at: at('09:00'), author: 'qc-03', summary: 'Назначена доп. проверка: рентген' },
  ],
  to_decide: {
    concession_required: false,
    decision_due_at: at('12:00'),
    decisions: ['nonconformity.nonconformity.confirm', 'nonconformity.signal.reject', 'nonconformity.recheck.request', 'nonconformity.item.isolate'],
  },
  ...over,
})

/** Та же карточка после подтверждения: открыто решение по изделию. */
export const confirmedCard = (): NCCard =>
  ncCard({
    status: 'confirmed',
    axes: { position: 'isolated', quality: 'nonconforming', disposition: 'none', containment: 'item_hold', erp_accounting: 'accepted_into_work' },
    to_decide: { concession_required: true, decision_due_at: at('12:00'), decisions: ['nonconformity.disposition.set'] },
  })

export const queueRows = (): DecisionQueueRow[] => [
  {
    kind: 'signal',
    object_id: 'SIG-77',
    nc_id: 'NC-0142',
    item_id: 'ENT:FL-0042',
    item_label: 'FL-0042',
    title: 'Пора в шве · КТ-3',
    severity: 'major',
    risk_rank: 0,
    step_key: 'weld',
    due_at: at('12:00'),
    overdue: false,
    basis_seq: 1260,
  },
  {
    kind: 'isolated',
    object_id: 'NC-0139',
    nc_id: 'NC-0139',
    item_id: 'ENT:FL-0039',
    item_label: 'FL-0039',
    title: 'Изолировано: трещина кромки',
    severity: 'critical',
    risk_rank: 1,
    step_key: 'edge',
    due_at: at('09:00'),
    overdue: true,
    basis_seq: 1190,
  },
  {
    kind: 'presentation',
    object_id: 'PR-5',
    item_id: 'ENT:FL-0031',
    item_label: 'FL-0031',
    title: 'ЗТ-3 · окончательный контроль',
    severity: 'unknown',
    risk_rank: 2,
    step_key: 'final',
    presentation_no: 2,
    overdue: false,
    basis_seq: 1100,
  },
]

export const concessions = (): Concession[] => [
  { concession_id: 'CON-1', document_id: 'DOC-CON-1', kind: 'use_as_is', status: 'active', title: 'РО-12/26 «Поры до 0,5 мм»', limit: 10, used: 3, valid_until: '2026-12-31T00:00:00.000Z' },
  { concession_id: 'CON-2', document_id: 'DOC-CON-2', kind: 'use_as_is', status: 'expired', title: 'РО-03/26', limit: 5, used: 1 },
  { concession_id: 'CON-3', document_id: 'DOC-CON-3', kind: 'repair', status: 'active', title: 'РО-14/26 ремонт', limit: 2, used: 2 },
]
