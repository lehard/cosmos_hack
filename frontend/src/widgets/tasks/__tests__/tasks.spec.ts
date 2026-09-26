// Задачи и уведомления (FR-57, FR-8, FR-55, UJ-7): виды — информация, тревога,
// задача, запрос решения; отметка задачи; задача «перенести в изолятор»
// закрывается подтверждённой приёмкой; эскалация с ценой задержки.
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mockApi, mountWidget, settle } from '@/entities/run/__tests__/api'
import { at, foremanSession, isolatedPassport, isolationCard, locations, tasks } from '@/entities/workplace/__tests__/fixtures'
import { alertText, noticeText, OBLIGATION_BASES } from '@/entities/notification'
import { codeToKey, i18n } from '@/shared/i18n'
import { ownWorkplaceTasks, sectionsOf, splitTasks } from '../model/slice'
import TasksWidget from '../ui/TasksWidget.vue'
import { NSelect } from 'naive-ui'

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
    // Без плашек-счётчиков и без второго заголовка «Задачи» внутри рамки (UI-38).
    expect(w.find('[data-testid="summary"]').exists()).toBe(false)
    expect(w.find('[data-testid="section-tasks"] h3').exists()).toBe(false)

    const open = w.findAll('[data-testid="section-tasks"] [data-task][data-state="open"]')
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
    // Итог виден сразу: что отмечено; кнопки у задачи убраны (UI-37).
    const acked = w.find('[data-task="TASK-004"] [data-testid="acked"]').text()
    expect(acked).toContain('Вы отметили задачу: Выполнена')
    expect(acked).toContain('Записано в журнал: запись № 142')
    expect(w.find('[data-task="TASK-004"] [data-testid="ack-done"]').exists()).toBe(false)
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

  it('задача процесса «Принять в цех»: форма прямо в задаче, id изделия кодируется в пути', async () => {
    const item = 'ENT01:show-is2-20260921-1/I-3CDF7159'
    const receive = {
      task_id: 'TASK-RCV-1',
      kind: 'other',
      title: 'Принять в цех Ф-001',
      state: 'open',
      assignee_role: 'site_foreman',
      assignee_id: null,
      created_at: at('08:00'),
      due_at: null,
      overdue: false,
      ref: { entity: 'item', id: item },
      operation_id: 'process.movement.receive',
      item_id: item,
      item_label: 'Ф-001',
    }
    const { routes } = world()
    const calls = mockApi({
      ...routes,
      'GET /api/v1/tasks': { items: [receive] },
      'GET /api/v1/permissions': { policy_seq: 3, items: [{ action: 'process.movement.receive', subject: 'item', action_class: 'record' }] },
      [`POST /api/v1/items/${item}/movements/receive`]: receipt(150),
    })
    const w = await mountWidget(TasksWidget, props)
    const task = () => w.find('[data-task="TASK-RCV-1"]')
    expect(task().text()).toContain('Принять в цех Ф-001')
    expect(task().text()).not.toContain('I-3CDF7159')
    // Действие — глаголом, а не «Открыть»; отметки «выполнено» у задачи с действием нет.
    expect(task().find('[data-testid="ack-done"]').exists()).toBe(false)
    await task().find('[data-action="receive-item"]').trigger('click')
    await settle()
    expect(task().find('[data-testid="receive-place"]').exists()).toBe(true)
    await task().find('[data-action="confirm-receive"]').trigger('click')
    await settle()
    const post = calls.find((c) => c.method === 'POST' && c.path === `/api/v1/items/${item}/movements/receive`)
    expect(post?.body).toMatchObject({ destination_kind: 'workshop', to_location_id: 'WS-WC', inspection_on_receipt: 'no_damage', basis_seq: 9100 })
    expect(task().find('[data-testid="receive-receipt"]').text()).toContain('запись № 150')
  })

  it('задача процесса «Отправить» после ЗТ-3: форма в задаче — откуда цех задачи, куда выбирает мастер', async () => {
    const item = 'ENT01:show-is2-20260921-1/I-3CDF7159'
    const send = {
      task_id: 'TASK-SND-1',
      kind: 'process_step',
      title: 'Отправить Ф-001: передача в сборочный цех',
      state: 'open',
      assignee_role: 'site_foreman',
      assignee_id: null,
      created_at: at('09:12'),
      due_at: null,
      overdue: false,
      ref: { entity: 'item', id: item },
      operation_id: 'process.movement.send',
      item_id: item,
      item_label: 'Ф-001',
      location_id: 'WS-WC',
      step_key: 'welding.send_to_assembly',
    }
    const { routes } = world()
    const calls = mockApi({
      ...routes,
      'GET /api/v1/tasks': { items: [send] },
      'GET /api/v1/permissions': { policy_seq: 3, items: [{ action: 'process.movement.send', subject: 'item', action_class: 'record' }] },
      [`POST /api/v1/items/${item}/movements`]: receipt(151),
    })
    const w = await mountWidget(TasksWidget, props)
    const task = () => w.find('[data-task="TASK-SND-1"]')
    expect(task().find('[data-testid="ack-done"]').exists()).toBe(false)
    expect(task().find('[data-testid="send-from"]').text()).toContain('Сварочный цех')
    // Куда — не выбрано: кнопка неактивна, пока мастер не выбрал цех.
    expect(task().find('[data-action="confirm-send"]').attributes('disabled')).toBeDefined()
    w.findComponent({ name: 'SendMoveForm' }).findComponent(NSelect).vm.$emit('update:value', 'WS-MC')
    await settle()
    await task().find('[data-action="confirm-send"]').trigger('click')
    await settle()
    const post = calls.find((c) => c.method === 'POST' && c.path === `/api/v1/items/${item}/movements`)
    expect(post?.body).toMatchObject({ from_location_id: 'WS-WC', to_location_id: 'WS-MC', step_key: 'welding.send_to_assembly', basis_seq: 9100 })
    expect(task().find('[data-testid="send-receipt"]').text()).toContain('запись № 151')
  })

  it('открытых задач нет — одной строкой, выполненные ниже; аномалия узла открывает окно операции (UI-44, UI-45)', async () => {
    const { routes } = world()
    mockApi({ ...routes, 'GET /api/v1/tasks': { items: tasks().filter((t) => t.state !== 'open'), basis_seq: 9100 } })
    const w = await mountWidget(TasksWidget, props)
    expect(w.find('[data-testid="no-open-tasks"]').text()).toBe('Открытых задач нет')
    expect(w.find('[data-testid="closed-tasks"]').exists()).toBe(true)
  })

  it('срез стола: только задачи; своё рабочее место', () => {
    expect(sectionsOf({ kinds: ['task'] })).toEqual(['task'])
    expect(sectionsOf({ kinds: ['nonsense'] })).toEqual(['task', 'alarm', 'escalation'])
    const list = tasks()
    // Своё место — задачи поста и задачи цеха (без привязки к другому посту): TASK-003 на WS-WC тоже своя.
    expect(ownWorkplaceTasks(list, 'WP-WELD-2').map((t) => t.task_id)).toEqual(['TASK-003', 'TASK-004', 'TASK-001'])
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

  it('аномалия узла — именем узла по схеме, код — только если имени нет (UI-38)', () => {
    const base = { alert_id: 'A', kind: 'anomaly', at: '2026-09-23T08:00:00Z', anomaly: 'queue_above_norm', node: 'welding.weld' } as const
    expect(alertText(x, { ...base, node_name: 'Сварка фланца' } as never)).toContain('Аномалия узла Сварка фланца')
    expect(alertText(x, base as never)).toContain('welding.weld')
  })

  it('параметры: время — по часам завода, неизвестный ключ — как есть', () => {
    const text = noticeText(x, 'notifications.overdue.isolation_move', { item: 'ENT01:F-017', first_due_at: '2026-09-23T05:38:00.000Z', level: '2', title: 'Переместить в изолятор' })
    expect(text).toBe('Изолировано в системе, физически не перемещено: ENT01:F-017 — срок перемещения 23.09.2026, 08:38 истёк, уровень эскалации 2')
    expect(noticeText(x, 'notifications.overdue.unknown_basis', {})).toBe('notifications.overdue.unknown_basis')
  })
})
