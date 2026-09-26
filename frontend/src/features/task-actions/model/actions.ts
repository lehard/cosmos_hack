/**
 * Реестр действий задачи (процесс настоящий: роль входит и видит свою работу).
 * Задачу роли порождает бэкенд из шага процесса: объект — изделие (`item_id`,
 * `item_label`), действие — operationId (`operation`). По operationId решается,
 * что показать в задаче:
 * - `form` — форма прямо в задаче («Принять в цех» → куда принято / осмотр);
 * - `window` — кнопка с глаголом действия, открывающая окно записи
 *   (`?open=‹тип›:‹id›`, Д-70), где это действие в нижней панели;
 * - `terminal` — действие исполнителя на его терминале (допуск, начать,
 *   завершить): кнопка ведёт к терминалу на этом же столе, иначе — окно изделия.
 * Задача без operationId — по виду задачи (`kind`), как раньше.
 */
import type { DrillRef, TaskEntry } from '@/shared/api/generated/model'

/**
 * Поля задачи процесса (ветка fix/process-tasks). Пока их нет в
 * сгенерированном клиенте — типизированы здесь, все необязательные.
 */
export interface ProcessTaskFields {
  /** operationId действия шага процесса. */
  operation?: string | null
  /** Изделие — объект задачи. */
  item_id?: string | null
  /** Метка изделия для людей (Ф-001, DM-код), не внутренний id. */
  item_label?: string | null
  role?: string | null
  scope?: string | null
}

/** Задача с полями процесса. */
export type ProcessTask = TaskEntry & ProcessTaskFields

/** Форма прямо в задаче. */
export type TaskForm = 'receive' | 'isolator_move'

/** Как задача показывает своё действие. */
export type TaskActionDef =
  | { kind: 'form'; form: TaskForm; verbKey: string }
  | { kind: 'window'; window: DrillRef['entity'] | 'presentation' | 'operation'; verbKey: string }
  | { kind: 'terminal'; verbKey: string }

const V = 'taskActions.verb.'

/** operationId → действие в задаче. Новое действие процесса — строка здесь. */
export const TASK_ACTIONS: Readonly<Record<string, TaskActionDef>> = {
  'process.movement.receive': { kind: 'form', form: 'receive', verbKey: 'receiveAction.title' },
  'access.workplace.admit': { kind: 'terminal', verbKey: `${V}admit` },
  'process.operation.start': { kind: 'terminal', verbKey: `${V}start` },
  'process.operation.finish': { kind: 'terminal', verbKey: `${V}finish` },
  'nonconformity.presentation.resolve': { kind: 'window', window: 'presentation', verbKey: `${V}resolvePresentation` },
  'nonconformity.presentation.review': { kind: 'window', window: 'presentation', verbKey: `${V}reviewPresentation` },
  'nonconformity.recheck.request': { kind: 'window', window: 'nonconformity', verbKey: `${V}recheck` },
  'nonconformity.nonconformity.confirm': { kind: 'window', window: 'nonconformity', verbKey: `${V}confirmSignal` },
  'nonconformity.signal.reject': { kind: 'window', window: 'nonconformity', verbKey: `${V}confirmSignal` },
  'nonconformity.disposition.set': { kind: 'window', window: 'nonconformity', verbKey: `${V}disposition` },
  'nonconformity.disposition.verify': { kind: 'window', window: 'nonconformity', verbKey: `${V}verifyDisposition` },
  'nonconformity.nonconformity.close': { kind: 'window', window: 'nonconformity', verbKey: `${V}closeNc` },
  'analysis.cause.conclude': { kind: 'window', window: 'nonconformity', verbKey: `${V}cause` },
  'analysis.hypothesis.record': { kind: 'window', window: 'nonconformity', verbKey: `${V}cause` },
  'nonconformity.process_hold.set': { kind: 'window', window: 'workplace', verbKey: `${V}holdPost` },
  'nonconformity.process_hold.release': { kind: 'window', window: 'workplace', verbKey: `${V}releasePost` },
  'nonconformity.item.isolate': { kind: 'window', window: 'item', verbKey: `${V}isolate` },
  'nonconformity.containment.set': { kind: 'window', window: 'item', verbKey: `${V}containment` },
  'nonconformity.containment.release': { kind: 'window', window: 'item', verbKey: `${V}containmentRelease` },
  'process.movement.send': { kind: 'window', window: 'item', verbKey: `${V}send` },
}

/** Действие задачи, уже привязанное к объекту. */
export type TaskAction =
  | { kind: 'form'; form: TaskForm; verbKey: string; itemId: string }
  | { kind: 'window'; verbKey: string; ref: DrillRef }
  | { kind: 'terminal'; verbKey: string; ref: DrillRef | null }
  /** Нет своего действия — отметка «выполнено / принято / отклонено». */
  | { kind: 'ack'; ref: DrillRef | null }

/** Изделие задачи: поле процесса или ссылка на изделие. */
export function taskItemId(task: ProcessTask): string | null {
  return task.item_id || (task.ref?.entity === 'item' ? task.ref.id : null) || null
}

/** Окно для действия: тип окна и id по объекту задачи; нечем — ссылка задачи. */
function windowRef(task: ProcessTask, window: string): DrillRef | null {
  const item = taskItemId(task)
  const ref = task.ref ?? null
  const as = (id: string) => ({ entity: window, id }) as unknown as DrillRef
  if (ref && ref.entity === window) return ref
  if (window === 'presentation' && item) return as(item)
  if (window === 'item' && item) return as(item)
  if (window === 'workplace' && task.location_id) return as(task.location_id)
  return ref ?? (item ? ({ entity: 'item', id: item } as DrillRef) : null)
}

/** Запрос решения по виду задачи — окно объекта с глаголом «Принять решение». */
const DECISION_KINDS = new Set(['decision_required', 'review_after_new_data', 'protection_basis_changed'])

/**
 * Действие задачи по operationId (реестр), иначе по виду задачи.
 * @param task — задача с полями процесса
 */
export function taskActionOf(task: ProcessTask): TaskAction {
  const item = taskItemId(task)
  const ref = task.ref ?? (item ? ({ entity: 'item', id: item } as DrillRef) : null)
  const def = task.operation ? TASK_ACTIONS[task.operation] : undefined
  // Задачу «в изолятор» закрывает приёмка в изоляторе (FR-55) — своя форма.
  if (task.kind === 'isolate_move' && item) return { kind: 'form', form: 'isolator_move', verbKey: 'decisions.containment.confirmIsolatorMove', itemId: item }
  if (def?.kind === 'form' && item) return { kind: 'form', form: def.form, verbKey: def.verbKey, itemId: item }
  if (def?.kind === 'terminal') return { kind: 'terminal', verbKey: def.verbKey, ref }
  if (def?.kind === 'window') {
    const target = windowRef(task, def.window)
    if (target) return { kind: 'window', verbKey: def.verbKey, ref: target }
  }
  if (!def && ref && DECISION_KINDS.has(task.kind)) return { kind: 'window', verbKey: `${V}decide`, ref }
  return { kind: 'ack', ref }
}

/** Метка изделия задачи для людей: `item_label`; внутренний id не показываем. */
export const taskItemLabel = (task: ProcessTask): string | null => task.item_label || null
