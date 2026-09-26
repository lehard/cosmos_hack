// Образцы данных для тестов экранов разбора — только в тестах (PRD §11.10:
// заготовок в коде интерфейса нет). Сценарий «сварка вне режима» на фланце
// (FR-149, FR-153): до сварки чисто → во время две нештатности оборудования →
// после — признак дефекта; область риска 34 → 13 → 6 (FR-61).
import type { CircumstancesModel, CommonFactorsModel, HypothesesModel, NcGroup, RiskScopeModel } from '@/entities/incident'

/** Время 23.09.2026, UTC. */
const at = (hhmm: string) => `2026-09-23T${hhmm}:00.000Z`

export const weldCircumstances = (): CircumstancesModel => ({
  nc_id: 'NC-0142',
  operation: { operation_run_id: 'run-weld-0042', label: 'Сварка', started_at: at('08:10'), finished_at: at('08:40') },
  window: { start: at('07:55'), end: at('08:52'), lower_bound_event_id: 'e-kt2-ok', upper_bound_event_id: 'e-kt3-defect' },
  records: [
    // изделие
    {
      event_id: 'e-kt2-ok',
      lane: 'item',
      event_type: 'inspection.result.recorded',
      variant: 'no_defect_indicated',
      occurred_at: at('07:55'),
      params: { method: 'ВИК' },
      journal_seq: 1201,
      evidence_refs: ['kt2-0042-01'],
      source_kind: 'camera',
    },
    {
      event_id: 'e-kt3-defect',
      lane: 'item',
      event_type: 'inspection.result.recorded',
      variant: 'defect_indicated',
      occurred_at: at('08:52'),
      params: { method: 'Визуальный контроль' },
      journal_seq: 1240,
      evidence_refs: ['kt3-0042-07', 'kt3-0042-08'],
      related_event_ids: ['e-current', 'e-override'],
      source_kind: 'camera',
    },
    // человек
    { event_id: 'e-took', lane: 'person', event_type: 'operator.step.confirmed', occurred_at: at('08:06'), params: { step: 'Установка в приспособление' }, journal_seq: 1210 },
    { event_id: 'e-repos', lane: 'person', event_type: 'operator.action.observed', occurred_at: at('08:21'), params: { action: 'перепозиционирование детали' }, journal_seq: 1224 },
    {
      event_id: 'e-manual',
      lane: 'person',
      event_type: 'operator.mode.changed',
      occurred_at: at('08:27'),
      params: { what: 'ручная подача проволоки' },
      journal_seq: 1226,
      related_event_ids: ['e-override'],
    },
    // оборудование
    { event_id: 'e-cycle-start', lane: 'equipment', event_type: 'equipment.state.changed', variant: 'cycle_started', occurred_at: at('08:10'), journal_seq: 1212, source_kind: 'machine' },
    {
      event_id: 'e-current',
      lane: 'equipment',
      event_type: 'equipment.deviation.detected',
      variant: 'out_of_setpoint',
      occurred_at: at('08:18'),
      ended_at: at('08:23'),
      params: { parameter: 'ток', value: '212 А', setpoint: '180 А' },
      journal_seq: 1222,
      source_kind: 'machine',
    },
    {
      event_id: 'e-override',
      lane: 'equipment',
      event_type: 'equipment.deviation.detected',
      variant: 'manual_override',
      occurred_at: at('08:27'),
      params: { what: 'подача проволоки 130 %' },
      journal_seq: 1227,
      source_kind: 'machine',
    },
    { event_id: 'e-cycle-end', lane: 'equipment', event_type: 'equipment.state.changed', variant: 'cycle_finished', occurred_at: at('08:40'), journal_seq: 1233, source_kind: 'machine' },
  ],
  missing_information: ['tool_unknown'],
  conclusion_is_categorical: false,
})

/** Входной дефект: находка на входном контроле — раньше начала операции. */
export const incomingCircumstances = (): CircumstancesModel => {
  const m = weldCircumstances()
  m.window = { start: at('07:30'), end: at('07:55') }
  m.records = m.records.map((r) => (r.event_id === 'e-kt2-ok' ? { ...r, variant: 'defect_indicated' } : r)).filter((r) => r.event_id !== 'e-kt3-defect')
  return m
}

export const weldFactors = (): CommonFactorsModel => ({
  group_key: 'burn_through|welding|IS-3',
  group_label: 'Прожог × Сварка × ИС-3',
  nc_count: 3,
  rows: [
    { factor: 'performer', value: 'Сварщик С-14', matches: 1, distinct_values: 3 },
    { factor: 'tool', value: null, matches: 0, distinct_values: 0 },
    { factor: 'fixture', value: 'Приспособление П-7', matches: 2, distinct_values: 2 },
    { factor: 'material_batch', value: 'Плавка 5521', matches: 2, distinct_values: 2 },
    { factor: 'program', value: 'WPS-12 ред. 4', matches: 3, distinct_values: 1 },
    { factor: 'machine', value: 'Сварочный источник ИС-3', matches: 3, distinct_values: 1 },
  ],
})

export const weldHypotheses = (): HypothesesModel => ({
  nc_id: 'NC-0142',
  version: 2,
  hypotheses: [
    {
      hypothesis_id: 'h-incoming',
      category: 'incoming',
      status: 'rejected',
      confidence_bp: 300,
      supporting: [],
      contradicting: [{ event_id: 'e-kt2-ok', event_type: 'inspection.result.recorded', variant: 'no_defect_indicated', occurred_at: at('07:55'), params: { method: 'ВИК' } }],
    },
    {
      hypothesis_id: 'h-performer',
      category: 'performer',
      status: 'proposed_by_system',
      confidence_bp: 2100,
      supporting: [{ event_id: 'e-repos', event_type: 'operator.action.observed', occurred_at: at('08:21'), params: { action: 'перепозиционирование детали' } }],
      contradicting: [{ event_id: 'e-took', event_type: 'operator.step.confirmed', occurred_at: at('08:06'), params: { step: 'Установка в приспособление' } }],
    },
    {
      hypothesis_id: 'h-equipment',
      category: 'equipment',
      branch: 'why_made',
      status: 'proposed_by_system',
      confidence_bp: 7200,
      measurement_hint: 'ток источника ИС-3 на эталонном образце',
      supporting: [
        { event_id: 'e-override', event_type: 'equipment.deviation.detected', variant: 'manual_override', occurred_at: at('08:27'), params: { what: 'подача проволоки 130 %' } },
        { event_id: 'e-current', event_type: 'equipment.deviation.detected', variant: 'out_of_setpoint', occurred_at: at('08:18'), params: { parameter: 'ток', value: '212 А', setpoint: '180 А' } },
      ],
      contradicting: [],
    },
  ],
  missing_information: ['tool_unknown'],
  conclusion_is_categorical: false,
  similar_cases: [
    { nc_id: 'NC-0117', number: 'НС-0117', cause_category: 'equipment', cause_confirmed: true, measure: 'Замена кабеля массы ИС-3', result: 'effective' },
    { nc_id: 'NC-0098', number: 'НС-0098', cause_category: 'performer', cause_confirmed: false, measure: null, result: null },
  ],
})

const item = (n: number, known: 'confirmed' | 'suspect' | 'excluded' | 'unknown', action: 'observe' | 'check' | 'block' | 'release', location: 'in_production' | 'moved_on' = 'in_production') => ({
  item_id: `ANT:FL-00${n}`,
  label: `FL-00${n}`,
  known,
  action,
  location,
})

export const weldScope = (): RiskScopeModel => ({
  incident_id: 'INC-12',
  incident_label: 'Инцидент И-12 · прожог после сварки',
  common_factor: { factor: 'machine', value: 'Сварочный источник ИС-3' },
  window: { start: at('06:40'), end: at('08:52') },
  last_known_good: { label: 'FL-0031, ВИК после сварки', at: at('06:40') },
  versions: [
    {
      scope_version: 1,
      change: 'computed',
      size: 34,
      recorded_at: at('08:53'),
      author: null,
      reason: { text: 'Все изделия через ИС-3 от последней чистой проверки до обнаружения' },
      evidence_event_ids: ['e-kt3-defect'],
      breakdown: { in_production: 20, moved_on: 8, assembled: 4, shipped: 2 },
    },
    {
      scope_version: 2,
      change: 'narrowed',
      size: 13,
      recorded_at: at('09:30'),
      author: 'Технолог Т-03',
      reason: { code: 'machine_log', text: 'Журнал станка: до 08:05 режим в норме' },
      evidence_event_ids: ['e-cycle-start', 'e-log-0805'],
      breakdown: { in_production: 9, moved_on: 3, assembled: 1, shipped: 0 },
    },
    {
      scope_version: 3,
      change: 'narrowed',
      size: 6,
      recorded_at: at('11:15'),
      author: 'Контролёр К-07',
      reason: { text: 'Доп. ВИК семи изделий: признаки не обнаружены' },
      evidence_event_ids: ['e-v1', 'e-v2', 'e-v3', 'e-v4', 'e-v5', 'e-v6', 'e-v7'],
      breakdown: { in_production: 5, moved_on: 1, assembled: 0, shipped: 0 },
    },
  ],
  items: [
    item(42, 'confirmed', 'block'),
    item(43, 'suspect', 'check'),
    item(44, 'suspect', 'check'),
    item(45, 'suspect', 'block'),
    item(46, 'unknown', 'observe'),
    item(40, 'suspect', 'check', 'moved_on'),
  ],
})

export const ncGroups = (): NcGroup[] => [
  { group_key: 'porosity|welding|IS-2', defect_type: 'Пористость', operation: 'Сварка', equipment: 'ИС-2', nc_count: 1, investigation: 'not_started', last_found_at: at('06:00') },
  { group_key: 'burn_through|welding|IS-3', defect_type: 'Прожог', operation: 'Сварка', equipment: 'ИС-3', nc_count: 3, investigation: 'hypothesis_only', last_found_at: at('08:52') },
  { group_key: 'burr|milling|CNC-7', defect_type: 'Заусенец', operation: 'Фрезерование', equipment: 'ЧПУ-7', nc_count: 3, investigation: 'in_progress', last_found_at: at('05:10') },
]
