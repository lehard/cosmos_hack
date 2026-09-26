// Терминал исполнителя (FR-137, FR-55, UJ-7): только своё рабочее место;
// подтверждение перемещения в изолятор снимает расхождение «изолировано в
// системе, физически не перемещено»; предупреждение «ресурс инструмента 73/75»;
// начать и остановить операцию; отклонение; воспроизведение — без действий.
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { mockApi, mountWidget, settle } from '@/entities/run/__tests__/api'
import {
  cleanPassport,
  equipment,
  isolatedPassport,
  isolationCard,
  itemsAtWeld,
  liveMap,
  locations,
  posts,
  registry,
  runProfile,
  session,
} from '@/entities/workplace/__tests__/fixtures'
import { useMomentStore } from '@/shared/model/moment'
import PerformerTerminalView from '../ui/PerformerTerminalView.vue'
import PerformerTerminalWidget from '../ui/PerformerTerminalWidget.vue'

afterEach(() => vi.unstubAllGlobals())
/** Текст без неразрывных пробелов (единицы измерения в текстах — через NBSP). */
const plain = (s: string) => s.replace(/\u00a0/g, ' ')

const props = { widgetId: 'performer-terminal', titleKey: 'desks.terminal', slice: { scope: 'own_workplace' }, density: 'large' }
const receipt = (seq: number) => ({ command_id: 'c', seq, event_ids: ['e'], replayed: false })

/** Оборудование поста сварки 2: ИС-2 работает, ресурс горелки 73 из 75. */
const postEquipment = (currentRun?: string) => [
  {
    ...equipment()[0]!,
    equipment_id: 'IS-2',
    title: 'Сварочный источник ИС-2',
    station_id: 'WP-WELD-2',
    current_run_id: currentRun,
  },
]

/** Мир терминала; `state.moved` меняет приёмка в изоляторе. */
function world(opts: { currentRun?: string; sessionOver?: Parameters<typeof session>[0] } = {}) {
  const state = { moved: false }
  const routes = {
    'GET /api/v1/auth/session': session(opts.sessionOver),
    'GET /api/v1/equipment': { items: postEquipment(opts.currentRun) },
    'GET /api/v1/reference/equipment': { items: registry() },
    'GET /api/v1/reference/locations': { items: locations() },
    'GET /api/v1/workplaces': { items: posts() },
    'GET /api/v1/live-map': liveMap(),
    'GET /api/v1/items': { items: itemsAtWeld() },
    'GET /api/v1/operation-runs/RUN-15/profile': runProfile({ item_id: 'ENT01:F-020' }),
    'GET /api/v1/items/ENT01:F-017/passport': () => isolatedPassport(),
    'GET /api/v1/items/ENT01:F-020/passport': cleanPassport('ENT01:F-020', 'Ф-020'),
    'GET /api/v1/nonconformities/NC-17': () => isolationCard(state.moved),
    'POST /api/v1/items/ENT01:F-017/movements/receive': () => {
      state.moved = true
      return receipt(140)
    },
    'POST /api/v1/items/ENT01:F-020/operations': receipt(150),
    'POST /api/v1/operation-runs/RUN-15/finish': receipt(151),
    'POST /api/v1/workplaces/WP-WELD-2/deviations': receipt(152),
    'POST /api/v1/workplaces/WP-WELD-2/inspection-requests': receipt(153),
  }
  return { state, routes }
}

describe('терминал исполнителя', () => {
  it('подтверждение перемещения с терминала снимает расхождение «изолировано в системе, физически нет» (FR-137, FR-55)', async () => {
    const { routes } = world()
    const calls = mockApi(routes)
    const w = await mountWidget(PerformerTerminalWidget, props)

    const item = () => w.find('[data-testid="items"] [data-item="ENT01:F-017"]')
    expect(item().find('[data-isolation="not_moved"]').exists()).toBe(true)
    expect(item().find('[data-testid="not-moved"]').text()).toContain('Изолировано в системе, физически не перемещено: Ф-017')

    await item().find('[data-testid="confirm-isolator-move"]').trigger('click')
    await settle()

    const post = calls.find((c) => c.method === 'POST' && c.path === '/api/v1/items/ENT01:F-017/movements/receive')
    expect(post?.body).toMatchObject({
      destination_kind: 'isolator',
      to_location_id: 'ISO-WC',
      inspection_on_receipt: 'no_damage',
      basis_seq: 120,
      policy_seq: 3,
      workplace_id: 'WP-WELD-2',
    })
    // Карточка перечитана: расхождения больше нет — по данным сервера.
    expect(item().find('[data-isolation="moved"]').exists()).toBe(true)
    expect(item().find('[data-testid="not-moved"]').exists()).toBe(false)
    expect(item().find('[data-testid="moved"]').text()).toContain('Изделие физически перемещено в изолятор')
    expect(item().find('[data-testid="moved"]').text()).toContain('запись № 140')
  })

  it('только своё рабочее место; предупреждение «ресурс инструмента 73 из 75»', async () => {
    mockApi(world().routes)
    const w = await mountWidget(PerformerTerminalWidget, props)
    expect(w.find('[data-testid="workplace"]').text()).toContain('Рабочее место: Пост сварки 2 (источник ИС-2)')
    expect(w.text()).toContain('Действия принимаются только с вашего рабочего места')
    expect(plain(w.find('[data-warning="tool_life"]').text())).toBe('Сварочный источник ИС-2: Ресурс инструмента 73 из 75 — заменить после этой детали')
    // Операции — только своего цеха (сварочного), по схеме процесса.
    const view = w.findComponent(PerformerTerminalView)
    expect(view.props('operations').map((o: { stepKey: string }) => o.stepKey)).toEqual(['welding.edge_prep', 'welding.weld'])
  })

  it('начать операцию: изделие из очереди шага; ждущее контроля — «сначала контроль»', async () => {
    const calls = mockApi(world().routes)
    const w = await mountWidget(PerformerTerminalWidget, props)
    const view = w.findComponent(PerformerTerminalView)
    await view.vm.$emit('update:stepKey', 'welding.weld')
    await settle()
    expect(view.props('candidates')).toEqual([
      expect.objectContaining({ row: expect.objectContaining({ item_id: 'ENT01:F-020' }), blocked: false }),
      expect.objectContaining({ row: expect.objectContaining({ item_id: 'ENT01:F-021' }), blocked: true }),
    ])
    await view.vm.$emit('start', 'ENT01:F-020')
    await settle()
    const post = calls.find((c) => c.method === 'POST' && c.path === '/api/v1/items/ENT01:F-020/operations')
    expect(post?.body).toMatchObject({
      step_key: 'welding.weld',
      operation_code: '030',
      station_id: 'WP-WELD-2',
      equipment_id: 'IS-2',
      workplace_id: 'WP-WELD-2',
      policy_seq: 3,
      basis_seq: 130,
    })
    expect((post?.body as { operation_run_id: string }).operation_run_id).toMatch(/^[0-9a-f-]{36}$/)
    expect(w.find('[data-testid="result"]').text()).toBe('Операция начата: запись № 150')
  })

  it('текущая операция: идёт против нормы; остановить — process.operation.finish', async () => {
    const calls = mockApi(world({ currentRun: 'RUN-15' }).routes)
    const w = await mountWidget(PerformerTerminalWidget, props)
    const run = w.find('[data-testid="current-run"]')
    expect(plain(run.text())).toContain('Сварка фланца с патрубком')
    expect(plain(run.text())).toContain('норма 40 мин–1 ч 30 мин')
    expect(plain(run.text())).toContain('выше нормы')
    // Начать новую, пока идёт текущая, нельзя.
    expect(w.find('[data-testid="start-operation"]').attributes('disabled')).toBeDefined()
    await w.find('[data-testid="stop"]').trigger('click')
    await settle()
    const post = calls.find((c) => c.method === 'POST' && c.path === '/api/v1/operation-runs/RUN-15/finish')
    expect(post?.body).toMatchObject({ completion: 'completed', workplace_id: 'WP-WELD-2', policy_seq: 3 })
  })

  it('сообщить об отклонении — с рабочего места сеанса', async () => {
    const calls = mockApi(world().routes)
    const w = await mountWidget(PerformerTerminalWidget, props)
    await w.find('[data-testid="deviation-text"] textarea').setValue('Кромка с заусенцем после подготовки')
    await w.find('[data-testid="deviation"] form').trigger('submit')
    await settle()
    const post = calls.find((c) => c.method === 'POST' && c.path === '/api/v1/workplaces/WP-WELD-2/deviations')
    expect(post?.body).toMatchObject({ description: 'Кромка с заусенцем после подготовки', workplace_id: 'WP-WELD-2' })
    expect(w.find('[data-testid="result"]').text()).toBe('Отклонение записано: запись № 152')
  })

  it('нет рабочего места в сеансе — действий нет', async () => {
    mockApi(world({ sessionOver: { workplace: undefined } }).routes)
    const w = await mountWidget(PerformerTerminalWidget, props)
    expect(w.find('[data-testid="workplace"]').text()).toContain('Рабочее место не выбрано')
    expect(w.find('[data-testid="report-deviation"]').attributes('disabled')).toBeDefined()
    expect(w.find('[data-testid="request-inspection"]').attributes('disabled')).toBeDefined()
  })

  it('воспроизведение — действия выключены, расхождение видно', async () => {
    mockApi(world().routes)
    const pinia = createPinia()
    setActivePinia(pinia)
    useMomentStore().travel('2026-09-23T09:00:00.000Z')
    const w = await mountWidget(PerformerTerminalWidget, props, pinia)
    expect(w.find('[data-testid="not-moved"]').exists()).toBe(true)
    expect(w.find('[data-testid="confirm-isolator-move"]').attributes('disabled')).toBeDefined()
    expect(w.find('[data-testid="request-inspection"]').attributes('disabled')).toBeDefined()
  })
})
