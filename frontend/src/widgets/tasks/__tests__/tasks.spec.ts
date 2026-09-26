// Задачи и уведомления (FR-57, FR-8, FR-55, UJ-7): виды — информация, тревога,
// задача, запрос решения; отметка задачи; задача «перенести в изолятор»
// закрывается подтверждённой приёмкой; эскалация с ценой задержки.
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mockApi, mountWidget, settle } from '@/entities/run/__tests__/api'
import { at, foremanSession, isolatedPassport, isolationCard, locations, tasks } from '@/entities/workplace/__tests__/fixtures'
import { noticeText, OBLIGATION_BASES } from '@/entities/notification'
import { codeToKey, i18n } from '@/shared/i18n'
import { ownWorkplaceTasks, sectionsOf, splitTasks } from '../model/slice'
import TasksWidget from '../ui/TasksWidget.vue'

afterEach(() => vi.unstubAllGlobals())
/** Текст без неразрывных пробелов (единицы измерения в текстах — через NBSP). */
const plain = (s: string) => s.replace(/\u00a0/g, ' ')

const props = { widgetId: 'tasks', titleKey: 'desks.tasks', density: 'large' }
const receipt = (seq: number) => ({ command_id: 'c', seq, event_ids: ['e'], replayed: false })

function world() {
  const state = { moved: false }
  const routes = {
    'GET /api/v1/auth/session': foremanSession(),
    'GET /api/v1/journal/head': { seq: 9100, ca_seq: 12, clock_mode: 'system', recorded_at: at('09:00') },
    'GET /api/v1/tasks': { items: tasks() },
    'GET /api/v1/alerts': {
      items: [
        { alert_id: 'AL-ISO', at: at('08:38'), kind: 'not_moved_to_isolator', item: 'Ф-017', ref: { entity: 'item', id: 'ENT01:F-017' } },
        { alert_id: 'AL-AN', at: at('08:30'), kind: 'anomaly', node: 'welding.weld', anomaly: 'queue_above_norm', ref: { entity: 'live_map', id: 'welding.weld' } },
      ],
    },
    'GET /api/v1/attention': {
      items: [{ kind: 'overdue_decision', entry_id: 'ATT-1', target: 'Решение по изолированному изделию Ф-017', overdue_minutes: 37, items: 31, operations: 2, ref: { entity: 'item', id: 'ENT01:F-017' } }],
    },
    'GET /api/v1/notifications/summary': { unread: 6, by_kind: { info: 1, alarm: 2, task: 2, decision_request: 1 } },
    'GET /api/v1/reference/locations': { items: locations() },
    'GET /api/v1/items/ENT01:F-017/passport': () => isolatedPassport(),
    'GET /api/v1/nonconformities/NC-17': () => isolationCard(state.moved),
    'POST /api/v1/items/ENT01:F-017/movements/receive': () => {
      state.moved = true
      return receipt(141)
    },
    'POST /api/v1/tasks/TASK-004/acknowledge': receipt(142),
  }
  return { state, routes }
}

describe('задачи и уведомления', () => {
  it('виды уведомлений, задачи со сроком, тревоги, эскалация с ценой задержки', async () => {
    mockApi(world().routes)
    const w = await mountWidget(TasksWidget, props)
    const summary = w.find('[data-testid="summary"]')
    expect(summary.find('[data-kind="info"]').text()).toBe('Информация: 1')
    expect(summary.find('[data-kind="alarm"]').text()).toBe('Тревога: 2')
    expect(summary.find('[data-kind="task"]').text()).toBe('Задача: 2')
    expect(summary.find('[data-kind="decision_request"]').text()).toBe('Запрос решения: 1')

    const open = w.findAll('[data-testid="section-tasks"] > .n-list [data-task]')
    expect(open.map((t) => t.attributes('data-task'))).toEqual(['TASK-003', 'TASK-004'])
    expect(open[0]!.text()).toContain('Просрочена')
    expect(open[0]!.text()).toContain('Перенести Ф-017 в изолятор и подтвердить')
    expect(w.find('[data-testid="closed-tasks"]').text()).toContain('Выполненные и снятые: 1')

    expect(plain(w.find('[data-escalation="ATT-1"]').text())).toContain('Решение по изолированному изделию Ф-017: просрочено на 37 мин — стоят 31 изделие, 2 операции')
    expect(plain(w.find('[data-alert="AL-ISO"]').text())).toContain('Изолировано в системе, физически не перемещено: Ф-017')
    expect(plain(w.find('[data-alert="AL-AN"]').text())).toContain('Аномалия узла welding.weld: Очередь выше нормы узла')
    expect(w.find('.widget-frame').attributes('data-state')).toBe('normal')
  })

  it('отметить задачу «выполнено»: notifications.task.acknowledge, basis_seq — голова журнала', async () => {
    const calls = mockApi(world().routes)
    const w = await mountWidget(TasksWidget, props)
    await w.find('[data-task="TASK-004"] [data-testid="ack-done"]').trigger('click')
    await settle()
    const post = calls.find((c) => c.method === 'POST' && c.path === '/api/v1/tasks/TASK-004/acknowledge')
    expect(post?.body).toMatchObject({ outcome: 'done', basis_seq: 9100, policy_seq: 3 })
    expect(post?.body).not.toHaveProperty('note')
    expect(w.find('[data-task="TASK-004"] [data-testid="acked"]').text()).toBe('Записано в журнал: запись № 142')
  })

  it('отклонить — только с примечанием', async () => {
    const calls = mockApi(world().routes)
    const w = await mountWidget(TasksWidget, props)
    await w.find('[data-task="TASK-004"] [data-testid="ack-decline"]').trigger('click')
    expect(w.find('[data-task="TASK-004"] [data-testid="confirm-decline"]').attributes('disabled')).toBeDefined()
    await w.find('[data-task="TASK-004"] [data-testid="decline-note"] input').setValue('Изделие уже на рентгене')
    await w.find('[data-task="TASK-004"] form').trigger('submit')
    await settle()
    const post = calls.find((c) => c.method === 'POST' && c.path === '/api/v1/tasks/TASK-004/acknowledge')
    expect(post?.body).toMatchObject({ outcome: 'declined', note: 'Изделие уже на рентгене' })
  })

  it('задача «перенести в изолятор» закрывается приёмкой в изоляторе, а не отметкой (UJ-7)', async () => {
    const calls = mockApi(world().routes)
    const w = await mountWidget(TasksWidget, props)
    const task = () => w.find('[data-task="TASK-003"]')
    expect(task().find('[data-testid="ack-done"]').exists()).toBe(false)
    await task().find('[data-testid="open-isolator-move"]').trigger('click')
    await settle()
    expect(task().find('[data-testid="not-moved"]').exists()).toBe(true)
    await task().find('[data-testid="confirm-isolator-move"]').trigger('click')
    await settle()
    const post = calls.find((c) => c.method === 'POST' && c.path === '/api/v1/items/ENT01:F-017/movements/receive')
    expect(post?.body).toMatchObject({ destination_kind: 'isolator', to_location_id: 'ISO-WC' })
    expect(calls.some((c) => c.path.endsWith('/acknowledge'))).toBe(false)
    expect(task().find('[data-testid="moved"]').exists()).toBe(true)
  })

  it('срез стола: только задачи; своё рабочее место', () => {
    expect(sectionsOf({ kinds: ['task'] })).toEqual(['task'])
    expect(sectionsOf({ kinds: ['nonsense'] })).toEqual(['task', 'alarm', 'escalation'])
    const list = tasks()
    expect(ownWorkplaceTasks(list, 'WP-WELD-2').map((t) => t.task_id)).toEqual(['TASK-004', 'TASK-001'])
    expect(splitTasks(list).open.map((t) => t.task_id)).toEqual(['TASK-003', 'TASK-004'])
  })

  it('ни одна операция не отвечает — «ошибка входа»', async () => {
    mockApi({ 'GET /api/v1/auth/session': foremanSession() })
    const w = await mountWidget(TasksWidget, props)
    expect(w.find('.widget-frame').attributes('data-state')).toBe('input_error')
  })
})

describe('тексты уведомлений модуля notifications', () => {
  const { t, te, d } = i18n.global
  const x = { t: (k: string, p?: Record<string, unknown>) => t(k, p ?? {}), te: (k: string) => te(k), d: (v: Date, f: string) => d(v, f) }

  it('у каждого основания срока есть текст тревоги; ключ контракта → текст (codeToKey)', () => {
    for (const basis of OBLIGATION_BASES) expect(te(codeToKey(`notifications.overdue.${basis}`)), basis).toBe(true)
    expect(te(codeToKey('notifications.info.moved_to_isolator'))).toBe(true)
  })

  it('параметры: время — по часам завода, неизвестный ключ — как есть', () => {
    const text = noticeText(x, 'notifications.overdue.isolation_move', { item: 'ENT01:F-017', first_due_at: '2026-09-23T05:38:00.000Z', level: '2', title: 'Переместить в изолятор' })
    expect(text).toBe('Изолировано в системе, физически не перемещено: ENT01:F-017 — срок перемещения 23.09.2026, 08:38 истёк, уровень эскалации 2')
    expect(noticeText(x, 'notifications.overdue.unknown_basis', {})).toBe('notifications.overdue.unknown_basis')
  })
})
