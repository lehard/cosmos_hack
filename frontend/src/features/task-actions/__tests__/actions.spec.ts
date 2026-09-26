// Реестр действий задачи: operationId → форма прямо в задаче, окно с глаголом
// или терминал исполнителя; задача без operationId — по виду, как раньше.
import { describe, expect, it } from 'vitest'
import { i18n } from '@/shared/i18n'
import { TASK_ACTIONS, taskActionOf, taskItemLabel, type ProcessTask } from '../model/actions'

const base: ProcessTask = {
  task_id: 'T-1',
  kind: 'other',
  title: 'Принять в цех Ф-001',
  state: 'open',
  assignee_role: 'site_foreman',
  assignee_id: null,
  created_at: '2026-09-21T08:00:00Z',
  due_at: null,
  overdue: false,
}
const ITEM = 'ENT01:show-is2-20260921-1/I-3CDF7159'

describe('реестр действий задачи', () => {
  it('«Принять в цех» — форма прямо в задаче по изделию задачи', () => {
    const a = taskActionOf({ ...base, operation_id: 'process.movement.receive', item_id: ITEM, item_label: 'Ф-001' })
    expect(a).toEqual({ kind: 'form', form: 'receive', verbKey: 'receiveAction.title', itemId: ITEM })
  })

  it('изделие — из ссылки задачи, если item_id нет', () => {
    const a = taskActionOf({ ...base, operation_id: 'process.movement.receive', ref: { entity: 'item', id: ITEM } })
    expect(a).toMatchObject({ kind: 'form', form: 'receive', itemId: ITEM })
  })

  it('решение на ЗТ — окно предъявления по изделию; остановка поста — окно поста', () => {
    expect(taskActionOf({ ...base, operation_id: 'nonconformity.presentation.resolve', item_id: ITEM })).toEqual({
      kind: 'window',
      verbKey: 'taskActions.verb.resolvePresentation',
      ref: { entity: 'presentation', id: ITEM },
    })
    expect(taskActionOf({ ...base, operation_id: 'nonconformity.process_hold.set', ref: { entity: 'workplace', id: 'WP-IS2' } })).toMatchObject({
      kind: 'window',
      ref: { entity: 'workplace', id: 'WP-IS2' },
    })
    expect(taskActionOf({ ...base, operation_id: 'nonconformity.process_hold.set', location_id: 'WP-IS2', item_id: ITEM })).toMatchObject({
      ref: { entity: 'workplace', id: 'WP-IS2' },
    })
  })

  it('причина и решение по НС — окно несоответствия; нет НС — окно изделия', () => {
    expect(taskActionOf({ ...base, operation_id: 'analysis.cause.conclude', ref: { entity: 'nonconformity', id: 'NC-3' } })).toMatchObject({
      kind: 'window',
      verbKey: 'taskActions.verb.cause',
      ref: { entity: 'nonconformity', id: 'NC-3' },
    })
    expect(taskActionOf({ ...base, operation_id: 'nonconformity.disposition.set', item_id: ITEM })).toMatchObject({ kind: 'window', ref: { entity: 'item', id: ITEM } })
  })

  it('допуск и операции сварки — терминал исполнителя', () => {
    for (const op of ['access.workplace.admit', 'process.operation.start', 'process.operation.finish']) {
      expect(taskActionOf({ ...base, operation_id: op, item_id: ITEM }).kind, op).toBe('terminal')
    }
  })

  it('без operationId: «в изолятор» — форма, запрос решения — окно с глаголом, прочее — отметка', () => {
    expect(taskActionOf({ ...base, kind: 'isolate_move', ref: { entity: 'item', id: 'ENT01:F-017' } })).toMatchObject({ kind: 'form', form: 'isolator_move' })
    expect(taskActionOf({ ...base, kind: 'decision_required', ref: { entity: 'item', id: 'ENT01:F-012' } })).toMatchObject({ kind: 'window', verbKey: 'taskActions.verb.decide' })
    expect(taskActionOf({ ...base, kind: 'recheck', ref: { entity: 'item', id: 'ENT01:F-019' } }).kind).toBe('ack')
    expect(taskActionOf({ ...base, operation_id: 'unknown.op' }).kind).toBe('ack')
  })

  it('у каждого действия реестра есть глагол; метка изделия — item_label, не id', () => {
    const { te } = i18n.global
    for (const [op, def] of Object.entries(TASK_ACTIONS)) expect(te(def.verbKey), op).toBe(true)
    expect(te('taskActions.verb.decide')).toBe(true)
    expect(taskItemLabel({ ...base, item_id: ITEM, item_label: 'Ф-001' })).toBe('Ф-001')
    expect(taskItemLabel({ ...base, item_id: ITEM })).toBeNull()
  })
})
