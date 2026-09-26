/**
 * Провайдер панели наших свойств редактора процесса (FR-25, FR-12, AD-17).
 *
 * Панель показывает у выбранного элемента BPMN имя, описание
 * (`bpmn:documentation`, FR-154) и наши свойства — элементы `extensionElements`
 * пространства `urn:ant:bpmn-ext:1`. Состав групп и полей повторяет дескриптор
 * moddle `contracts/bpmn-ext/ant.json` (его копия — `ant-moddle.json` рядом,
 * её же получает модельер bpmn-js), допустимые значения — `rules.yaml → enums`.
 * Тест `__tests__/descriptor.spec.ts` сверяет копию с контрактом, а группы и
 * поля — с дескриптором: новое свойство в контракте без поля панели краснеет.
 *
 * Модуль без Vue и bpmn-js: чтение значений и приведение ввода — чистые
 * функции над «похожими на moddle» объектами, запись делает модельер.
 */
import descriptor from './ant-moddle.json'

/** Префикс нашего пространства в moddle (`ant:Properties`). */
export const ANT_PREFIX = descriptor.prefix

/** Вид поля панели: строка, число, флажок, многострочный текст. */
export type FieldKind = 'string' | 'integer' | 'boolean' | 'text'

/** Поле панели — одно свойство типа дескриптора. */
export interface PanelField {
  /** Имя свойства в дескрипторе (`stepKey`). */
  name: string
  kind: FieldKind
  /** Допустимые значения (rules.yaml → enums); нет — свободный ввод. */
  options?: readonly string[]
}

/** Группа панели — тип дескриптора (`Properties` → `ant:properties`). */
export interface PanelGroup {
  /** Имя типа в дескрипторе. */
  type: string
  /** Сколько элементов у узла: один или список (README расширения). */
  many: boolean
  fields: PanelField[]
}

const s = (name: string, options?: readonly string[]): PanelField => (options ? { name, kind: 'string', options } : { name, kind: 'string' })
const i = (name: string): PanelField => ({ name, kind: 'integer' })
const b = (name: string): PanelField => ({ name, kind: 'boolean' })
const t = (name: string): PanelField => ({ name, kind: 'text' })

/** Перечисления rules.yaml, которые панель предлагает списком. */
export const ENUMS = {
  stepKind: ['operation', 'automated_inspection', 'human_inspection', 'movement', 'storage'],
  reworkLimitScope: ['item', 'zone', 'loop'],
  erpAction: [
    'accept_into_work',
    'warehouse_transfer',
    'scrap_transfer_rework',
    'scrap_transfer_writeoff',
    'scrap_transfer_reprocess',
    'return_to_supplier',
    'return_from_defect',
    'release',
  ],
  timerScope: ['until_started', 'activity'],
  outcome: ['rework_or_repair', 'use_as_is', 'scrapped', 'returned'],
  closingPoint: ['ZT-1', 'ZT-2', 'ZT-3', 'ZT-4.1', 'ZT-4.2', 'ZT-5', 'ZT-6', 'ZT-R', 'ZT-V'],
  inspectionMethod: ['camera', 'cmm', 'radiography', 'ultrasonic', 'penetrant', 'leak_test', 'torque', 'visual_human', 'supplier_documents', 'laboratory', 'other'],
  inspectionPhase: ['incoming', 'before_operation', 'after_operation', 'before_zone_closure', 'assembly', 'test', 'final', 'other'],
  preconditionKind: [
    'qualification',
    'equipment_verification',
    'document_revision',
    'material_expiry',
    'time_window',
    'zone_check',
    'item_blocked',
    'open_intervention',
    'lot_accepted',
    'rework_limit',
    'status_expired',
  ],
  preconditionMode: ['block', 'record_violation'],
} as const

/**
 * Группы панели в порядке показа — по дескриптору и таблице «Элементы» README
 * расширения (сколько элементов у узла).
 */
export const PANEL_GROUPS: readonly PanelGroup[] = [
  {
    type: 'Properties',
    many: false,
    fields: [
      s('stepKey'),
      s('stepKind', ENUMS.stepKind),
      s('workshop'),
      s('warehouse'),
      b('specialProcess'),
      b('bufferPlace'),
      s('closesZoneAccess'),
      s('paperAttester'),
      s('closingPoint', ENUMS.closingPoint),
      s('inspectionPoint'),
      s('operationCode'),
      i('reworkLimit'),
      s('reworkLimitScope', ENUMS.reworkLimitScope),
      s('erpAction', ENUMS.erpAction),
      s('triggerEventType'),
      s('timerScope', ENUMS.timerScope),
      s('outcome', ENUMS.outcome),
    ],
  },
  {
    type: 'Inspection',
    many: false,
    fields: [s('method', ENUMS.inspectionMethod), s('phase', ENUMS.inspectionPhase), s('coverage'), s('recipeRef'), i('observationQualityMinBp')],
  },
  { type: 'PresentationPoint', many: false, fields: [s('authority'), s('role'), s('stampKind'), i('waitLimitMinutes'), i('waitLimitWorkDays'), s('repeatAuthority'), b('customerAcceptance')] },
  { type: 'Norm', many: false, fields: [i('timeMinutes'), i('timeMinMinutes'), i('timeMaxMinutes'), i('queueNormMinutes'), i('throughputPerShift'), i('idleThresholdMinutes')] },
  { type: 'NormRef', many: true, fields: [s('standard'), s('clause'), t('check'), t('systemAction')] },
  { type: 'Requirement', many: true, fields: [s('characteristic'), s('tolerance'), s('kdRef')] },
  { type: 'ZoneRef', many: true, fields: [s('zone')] },
  { type: 'Precondition', many: true, fields: [s('kind', ENUMS.preconditionKind), s('mode', ENUMS.preconditionMode), s('ref')] },
  { type: 'Document', many: true, fields: [s('template')] },
  { type: 'ReactionMap', many: false, fields: [s('ref')] },
]

/** Полное имя типа moddle: `ant:Properties`. */
export const moddleType = (type: string): string => `${ANT_PREFIX}:${type}`

/** Значение свойства в панели: строка, число, флажок или «нет». */
export type FieldValue = string | number | boolean | null

/** Минимум элемента moddle, который читает панель. */
export interface ModdleLike {
  $type: string
  get?(name: string): unknown
  [key: string]: unknown
}

/** Минимум бизнес-объекта BPMN, который читает панель. */
export interface BusinessObjectLike extends ModdleLike {
  id?: string
  name?: string
  documentation?: { text?: string }[]
  extensionElements?: { values?: ModdleLike[] } | null
}

const read = (el: ModdleLike, name: string): unknown => (typeof el.get === 'function' ? el.get(name) : el[name])

/** Наши элементы узла типа `type` (в порядке документа). */
export function entriesOf(bo: BusinessObjectLike, type: string): ModdleLike[] {
  const want = moddleType(type)
  return (bo.extensionElements?.values ?? []).filter((v) => v.$type === want)
}

/** Значение поля элемента для панели; отсутствующее — null. */
export function fieldValue(el: ModdleLike, f: PanelField): FieldValue {
  const v = read(el, f.name)
  if (v === undefined || v === null || v === '') return null
  if (f.kind === 'boolean') return v === true || v === 'true'
  if (f.kind === 'integer') {
    const n = typeof v === 'number' ? v : Number.parseInt(String(v), 10)
    return Number.isFinite(n) ? n : null
  }
  return String(v)
}

/**
 * Ввод пользователя → значение свойства moddle: пустая строка и «нет» —
 * свойство снимается (undefined), число — только целое (AD-4: без дробей).
 */
export function coerce(f: PanelField, raw: FieldValue): string | number | boolean | undefined {
  if (raw === null || raw === '') return undefined
  switch (f.kind) {
    case 'boolean':
      return raw === true || raw === 'true' ? true : undefined
    case 'integer': {
      const n = typeof raw === 'number' ? raw : Number.parseInt(String(raw).trim(), 10)
      return Number.isInteger(n) ? n : undefined
    }
    default: {
      const text = String(raw)
      return text.trim() === '' ? undefined : text
    }
  }
}

/** Описание элемента (`bpmn:documentation`, первое), пусто — "". */
export const documentationOf = (bo: BusinessObjectLike): string => bo.documentation?.[0]?.text ?? ''

/** step_key узла (`ant:properties/@stepKey`), нет — null. */
export function stepKeyOf(bo: BusinessObjectLike): string | null {
  const p = entriesOf(bo, 'Properties')[0]
  const v = p ? read(p, 'stepKey') : null
  return typeof v === 'string' && v !== '' ? v : null
}
