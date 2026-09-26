// Образцы данных цеха — только в тестах (PRD §11.10: заготовок в коде интерфейса
// нет). Сценарий UJ-7 «мастер участка разбирает очередь»: сварочный цех, у поста
// сварки 2 растёт очередь — назначенный сварщик не вставил ключ, источник ИС-2
// без данных; изделие Ф-017 изолировано в системе, физически не перемещено.
import type {
  AccessAssignmentList,
  AccessQualification,
  AnalyticsOverview,
  EquipmentState,
  ItemRow,
  LiveMap,
  PostRow,
  RefEquipment,
  RefLocation,
  RefShift,
  RunProfile,
  Session,
  TaskEntry,
} from '@/shared/api/generated/model'
import { flangePassport } from '@/entities/item/__tests__/fixtures'
import { ncCard } from '@/entities/nonconformity/__tests__/fixtures'

/** Время 23.09.2026, UTC. */
export const at = (hhmm: string) => `2026-09-23T${hhmm}:00.000Z`

/** Сеанс: роль, область, смена, рабочее место (AD-15). */
export const session = (over: Partial<Session> = {}): Session => ({
  user: { id: 'W21', name: 'Сварщик W21' },
  role: { id: 'performer', title: 'Исполнитель', inherits: ['staff'] },
  scope: 'ent01/b1/wc',
  shift: { id: 'SHIFT-1', title: 'Первая смена 08:00–16:30' },
  workplace: { id: 'WP-WELD-2', title: 'Пост сварки 2 (источник ИС-2)' },
  policy_seq: 3,
  demo: true,
  ...over,
})

/** Сеанс мастера сварочного цеха. */
export const foremanSession = (): Session =>
  session({ user: { id: 'FOR-WC', name: 'Мастер сварочного цеха' }, role: { id: 'site_foreman', title: 'Мастер участка', inherits: ['staff'] }, workplace: undefined })

/** Справочник мест: цех, участок, посты, изолятор. */
export const locations = (): RefLocation[] => [
  { location_id: 'B1', kind: 'building', name: 'Корпус 1', scope: 'ent01/b1', valid_from: at('00:00') },
  { location_id: 'WS-WC', kind: 'workshop', name: 'Сварочный цех', scope: 'ent01/b1/wc', parent_id: 'B1', valid_from: at('00:00') },
  { location_id: 'WS-MC', kind: 'workshop', name: 'Механический цех', scope: 'ent01/b1/mc', parent_id: 'B1', valid_from: at('00:00') },
  { location_id: 'ST-WELD', kind: 'station', name: 'Участок сварки', scope: 'ent01/b1/wc/weld', parent_id: 'WS-WC', valid_from: at('00:00') },
  { location_id: 'WP-WELD-1', kind: 'workplace', name: 'Пост сварки 1', scope: 'ent01/b1/wc/weld/wp1', parent_id: 'ST-WELD', valid_from: at('00:00') },
  { location_id: 'WP-WELD-2', kind: 'workplace', name: 'Пост сварки 2', scope: 'ent01/b1/wc/weld/wp2', parent_id: 'ST-WELD', valid_from: at('00:00') },
  { location_id: 'ISO-WC', kind: 'isolator', name: 'Изолятор сварочного цеха', scope: 'ent01/b1/wc/iso', parent_id: 'WS-WC', valid_from: at('00:00') },
  { location_id: 'ISO-MC', kind: 'isolator', name: 'Изолятор механического цеха', scope: 'ent01/b1/mc/iso', parent_id: 'WS-MC', valid_from: at('00:00') },
]

/** Схема процесса: дорожки цехов, операции с нормами и лимитом доработок (AD-17). */
export const bpmnXml = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:ant="urn:ant:bpmn-ext:1" id="D">
  <bpmn:process id="Process_Flange">
    <bpmn:laneSet id="LS">
      <bpmn:lane id="Lane_MC" name="Механический цех">
        <bpmn:extensionElements><ant:properties workshop="WS-MC" /></bpmn:extensionElements>
        <bpmn:flowNodeRef>M1</bpmn:flowNodeRef>
      </bpmn:lane>
      <bpmn:lane id="Lane_WC" name="Сварочный цех">
        <bpmn:extensionElements><ant:properties workshop="WS-WC" /></bpmn:extensionElements>
        <bpmn:flowNodeRef>W0</bpmn:flowNodeRef>
        <bpmn:flowNodeRef>W1</bpmn:flowNodeRef>
        <bpmn:flowNodeRef>W2</bpmn:flowNodeRef>
      </bpmn:lane>
    </bpmn:laneSet>
    <bpmn:task id="M1" name="Обработка на станке ЧПУ">
      <bpmn:extensionElements><ant:properties stepKey="machining.cnc" stepKind="operation" operationCode="010" reworkLimit="1" reworkLimitScope="item" /><ant:norm timeMinutes="60" timeMinMinutes="40" timeMaxMinutes="90" /></bpmn:extensionElements>
    </bpmn:task>
    <bpmn:task id="W0" name="Приёмка в сварочном цехе">
      <bpmn:extensionElements><ant:properties stepKey="welding.receive" stepKind="movement" /></bpmn:extensionElements>
    </bpmn:task>
    <bpmn:task id="W1" name="Подготовка кромок">
      <bpmn:extensionElements><ant:properties stepKey="welding.edge_prep" stepKind="operation" operationCode="020" reworkLimit="3" reworkLimitScope="loop" /><ant:norm timeMinMinutes="10" timeMaxMinutes="15" /></bpmn:extensionElements>
    </bpmn:task>
    <bpmn:task id="W2" name="Сварка фланца с патрубком">
      <bpmn:extensionElements><ant:properties stepKey="welding.weld" stepKind="operation" specialProcess="true" operationCode="030" reworkLimit="3" reworkLimitScope="zone" /><ant:norm timeMinMinutes="40" timeMaxMinutes="90" queueNormMinutes="240" /></bpmn:extensionElements>
    </bpmn:task>
  </bpmn:process>
</bpmn:definitions>`

/** Живая карта: у сварки очередь выше нормы, ИС-2 — ограничение линии. */
export const liveMap = (over: Partial<LiveMap> = {}): LiveMap => ({
  basis_seq: 9000,
  bpmn_xml: bpmnXml,
  process_version: { process_version_id: 'flange-1', label: 'v1', is_current: true, items: 47 },
  versions: [{ process_version_id: 'flange-1', label: 'v1', is_current: true, items: 47 }],
  counters: [
    { step_key: 'welding.edge_prep', queue: 1, in_progress: 0, passed: 3, defects: 0 },
    { step_key: 'welding.weld', queue: 7, in_progress: 1, passed: 2, defects: 1, nonconformities: 1 },
    { step_key: 'machining.cnc', queue: 0, in_progress: 1, passed: 9, defects: 0 },
  ],
  anomalies: [{ step_key: 'welding.weld', kind: 'queue_above_norm', threshold: '4 ч' }],
  bottleneck: { step_key: 'welding.weld', wait: '2 ч 10 мин' },
  data_gaps: [],
  items: [],
  ...over,
})

const minutes = (value: number) => ({ value, scale: 0, unit: 'min', origin: 'computed_by_system' as const, meaning: 'active_processing' as const })
const pcs = (value: number) => ({ value, scale: 0, unit: 'pcs' })

/** Обзор аналитики: показатели по шагу. */
export const overview = (): AnalyticsOverview => ({
  basis_seq: 9000,
  period: { from: at('05:00'), kind: 'shift', to: at('13:30') },
  items: [
    { metric_id: 'rework_runs', title: 'Повторные выполнения операций', group: 'defects', counts: 'operations', total: pcs(2), unknown: false, slices: [{ dimension: 'step', key: 'step:welding.weld', label: 'Сварка', value: pcs(2) }] },
    { metric_id: 'unfinished_operations', title: 'Незавершённые операции', group: 'defects', counts: 'operations', total: pcs(1), unknown: false, slices: [{ dimension: 'step', key: 'step:welding.weld', label: 'Сварка', value: pcs(1) }] },
    { metric_id: 'operation_duration', title: 'Длительность операций', group: 'time', counts: 'time', total: minutes(95), unknown: false, slices: [{ dimension: 'step', key: 'step:welding.weld', label: 'Сварка', value: minutes(95) }] },
  ],
})

/** Посты сварочного цеха (FR-6). */
export const posts = (): PostRow[] => [
  { workplace_id: 'WP-WELD-1', station: 'Пост сварки 1', workshop: 'WS-WC', assigned: { person_id: 'W22', display: 'Сварщик W22' }, presence: 'present', current_item: { item_id: 'ENT01:F-015', label: 'Ф-015' } },
  { workplace_id: 'WP-WELD-2', station: 'Пост сварки 2', workshop: 'WS-WC', assigned: { person_id: 'W21', display: 'Сварщик W21' }, presence: 'key_missing', current_item: { item_id: 'ENT01:F-017', label: 'Ф-017' } },
  { workplace_id: 'WP-QC-WC', station: 'Пост ОТК сварочного цеха', workshop: 'WS-WC', presence: 'not_assigned' },
]

/** Оборудование постов: ИС-1 работает, ресурс горелки на исходе; ИС-2 без данных. */
export const equipment = (over: Partial<EquipmentState>[] = []): EquipmentState[] => {
  const base: EquipmentState[] = [
    {
      equipment_id: 'IS-1',
      title: 'Сварочный источник ИС-1',
      station_id: 'WP-WELD-1',
      execution: 'running',
      controller_mode: 'automatic',
      condition: 'warning',
      special_process: true,
      verification: { status: 'valid', valid_till: '2027-01-31' },
      warnings: [{ kind: 'tool_life_warning', text: 'ресурс инструмента 73/75', since: at('08:00') }],
      tool_id: 'Горелка Г-2',
      tool_life_used: 73,
      tool_life_limit: 75,
      current_run_id: 'RUN-15',
    },
    {
      equipment_id: 'IS-2',
      title: 'Сварочный источник ИС-2',
      station_id: 'WP-WELD-2',
      execution: 'unknown',
      controller_mode: 'automatic',
      condition: 'unknown',
      special_process: true,
      verification: { status: 'unknown' },
      warnings: [{ kind: 'other', text: 'Нет данных от источника: связь со шлюзом потеряна', since: at('06:50') }],
    },
  ]
  return base.map((e, i) => ({ ...e, ...(over[i] ?? {}) }))
}

/** Справочник оборудования: поверка (FR-17). */
export const registry = (): RefEquipment[] => [
  { equipment_id: 'IS-1', kind: 'welding_source', location_id: 'ST-WELD', name: 'Сварочный источник ИС-1', is_measuring_instrument: false, usable: true },
  { equipment_id: 'IS-2', kind: 'welding_source', location_id: 'ST-WELD', name: 'Сварочный источник ИС-2', is_measuring_instrument: false, usable: true },
  { equipment_id: 'TW-9', kind: 'torque_wrench', location_id: 'ST-WELD', name: 'Ключ с регистрацией момента', is_measuring_instrument: true, usable: false, unusable_reason: 'verification_expired', verified_until: '2026-09-01' },
]

/** Профиль выполнения: сварка Ф-015 идёт с 08:00. */
export const runProfile = (over: Partial<RunProfile> = {}): RunProfile => ({
  operation_run_id: 'RUN-15',
  item_id: 'ENT01:F-015',
  step_key: 'welding.weld',
  started_at: at('08:00'),
  finished_at: null,
  interval_origin: 'source_reported',
  operator_id: 'W22',
  equipment_id: 'IS-1',
  events: [],
  parameters: [],
  special_process: true,
  violation: false,
  ...over,
})

const row = (id: string, label: string, position: ItemRow['status']['position'], summary: ItemRow['status']['summary'] = 'in_process'): ItemRow => ({
  item_id: id,
  item_type_id: 'ФЛ-100.02',
  label,
  step_key: 'welding.weld',
  status: { position, summary, quality: 'not_inspected', disposition: 'none', containment: 'none', erp_accounting: 'accepted_into_work' } as ItemRow['status'],
})

/** Изделия шага сварки: очередь, ждущее контроля, в изоляции. */
export const itemsAtWeld = (): ItemRow[] => [
  row('ENT01:F-020', 'Ф-020', 'in_queue'),
  row('ENT01:F-021', 'Ф-021', 'at_inspection'),
  row('ENT01:F-017', 'Ф-017', 'isolated', 'hold'),
]

/** Паспорт изолированного Ф-017 с несоответствием. */
export const isolatedPassport = () => flangePassport({ item_id: 'ENT01:F-017', label: 'Ф-017', basis_seq: 120, nonconformities: ['NC-17'] })

/** Паспорт изделия без несоответствий. */
export const cleanPassport = (id: string, label: string) => flangePassport({ item_id: id, label, basis_seq: 130, nonconformities: [], incidents: [] })

/**
 * Карточка несоответствия Ф-017: изолировано решением, приёмки в изоляторе
 * ещё нет (`moved = false`) или уже есть.
 */
export const isolationCard = (moved: boolean) =>
  ncCard({
    nc_id: 'NC-17',
    item_id: 'ENT01:F-017',
    item_label: 'Ф-017',
    basis_seq: 121,
    physically_not_moved: !moved,
    isolation: { event_id: 'e-iso', isolated_at: at('08:08'), isolator_location_id: 'ISO-WC', physically_moved: moved, overdue: false },
  })

/** Задачи мастера: перенести в изолятор, запрос решения, выполненная. */
export const tasks = (): TaskEntry[] => [
  {
    task_id: 'TASK-003',
    kind: 'isolate_move',
    title: 'Перенести Ф-017 в изолятор и подтвердить',
    state: 'open',
    assignee_role: 'site_foreman',
    assignee_id: 'FOR-WC',
    location_id: 'WS-WC',
    created_at: at('08:08'),
    due_at: at('08:38'),
    overdue: true,
    ref: { entity: 'item', id: 'ENT01:F-017' },
  },
  {
    task_id: 'TASK-004',
    kind: 'recheck',
    title: 'Изделие Ф-019 в области риска RS-1: доп. проверка',
    state: 'open',
    assignee_role: 'site_foreman',
    assignee_id: null,
    created_at: at('08:20'),
    due_at: null,
    overdue: false,
    ref: { entity: 'item', id: 'ENT01:F-019' },
  },
  {
    task_id: 'TASK-001',
    kind: 'decision_required',
    title: 'Подтвердить сигнал по Ф-012',
    state: 'done',
    assignee_role: 'quality_inspector',
    assignee_id: 'INS-01',
    created_at: at('07:06'),
    due_at: at('09:06'),
    overdue: false,
    ref: { entity: 'item', id: 'ENT01:F-012' },
  },
]

/** Смены сварочного цеха. */
export const shifts = (): RefShift[] => [
  { shift_id: 'SHIFT-1', location_id: 'WS-WC', name: 'Первая смена', starts_at: at('05:00'), ends_at: at('13:30') },
  { shift_id: 'SHIFT-2', location_id: 'WS-WC', name: 'Вторая смена', starts_at: at('13:30'), ends_at: at('22:00') },
]

/** Назначения первой смены. */
export const assignments = (): AccessAssignmentList => ({
  basis_seq: 700,
  items: [
    { workplace_id: 'WP-WELD-1', person_id: 'W22', assignee_role: 'performer', shift_id: 'SHIFT-1', admitted: true, qualification_ok: true },
    { workplace_id: 'WP-WELD-2', person_id: 'W21', assignee_role: 'performer', shift_id: 'SHIFT-1', admitted: false, qualification_ok: true },
  ],
})

/** Квалификации: у W23 аттестация сварщика истекла (UJ-7). */
export const qualifications = (): AccessQualification[] => [
  { qualification_id: 'Q-W21', person_id: 'W21', scope: 'Аттестация НАКС', status: 'valid', valid_from: '2026-01-01', valid_until: '2027-01-01' },
  { qualification_id: 'Q-W22', person_id: 'W22', scope: 'Аттестация НАКС', status: 'valid', valid_from: '2026-01-01', valid_until: '2027-01-01' },
  { qualification_id: 'Q-W23', person_id: 'W23', scope: 'Аттестация НАКС', status: 'expired', valid_from: '2024-01-01', valid_until: '2026-09-01' },
]
