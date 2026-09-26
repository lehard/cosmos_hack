// Участок мастера (PRD §3a, FR-81, UJ-7): операции цеха — очередь, в работе,
// длительность против нормы, повторные выполнения против лимита, незавершённые;
// «почему растёт очередь» — факты участка; посты — присутствие, текущая деталь,
// оборудование и текущее выполнение.
import { afterEach, describe, expect, it, vi } from 'vitest'
import { parseProcessSteps, operationsOf } from '@/entities/live-map'
import { mockApi, mountWidget } from '@/entities/run/__tests__/api'
import { bpmnXml, equipment, foremanSession, liveMap, locations, overview, posts, registry, runProfile } from '@/entities/workplace/__tests__/fixtures'
import { buildPosts, buildSteps, queueFacts, stepMetric } from '../model/station'
import StationPostsWidget from '../ui/StationPostsWidget.vue'

afterEach(() => vi.unstubAllGlobals())
/** Текст без неразрывных пробелов (единицы измерения в текстах — через NBSP). */
const plain = (s: string) => s.replace(/\u00a0/g, ' ')

const props = { widgetId: 'station-posts', titleKey: 'desks.station', slice: { scope: 'own_station' }, density: 'large' }

const routes = (over: Record<string, unknown> = {}, omit: string[] = []) => {
  const all: Record<string, unknown> = {
  'GET /api/v1/auth/session': foremanSession(),
  'GET /api/v1/reference/locations': { items: locations() },
  'GET /api/v1/live-map': liveMap(),
  'GET /api/v1/analytics': overview(),
  'GET /api/v1/workplaces': { items: posts() },
  'GET /api/v1/equipment': { items: equipment() },
  'GET /api/v1/reference/equipment': { items: registry() },
  'GET /api/v1/operation-runs/RUN-15/profile': runProfile(),
  ...over,
  }
  for (const k of omit) delete all[k]
  return all
}

describe('схема процесса: операции цеха', () => {
  it('операции дорожки цеха с нормами и лимитом доработок', () => {
    const parsed = parseProcessSteps(bpmnXml)
    const ops = operationsOf(parsed, 'WS-WC')
    expect(ops.map((s) => s.stepKey)).toEqual(['welding.edge_prep', 'welding.weld'])
    const weld = parsed.byKey.get('welding.weld')!
    expect(weld).toMatchObject({ operationCode: '030', reworkLimit: 3, reworkLimitScope: 'zone', specialProcess: true, workshop: 'WS-WC' })
    expect(weld.norm).toEqual({ timeMinMinutes: 40, timeMaxMinutes: 90, timeMinutes: null, queueNormMinutes: 240 })
    expect(operationsOf(parsed, null)).toHaveLength(3)
    expect(parseProcessSteps('<не xml').steps).toEqual([])
  })

  it('показатели по шагу — из среза `step:‹step_key›`; нет среза — null, а не ноль', () => {
    const rows = overview().items
    expect(stepMetric(rows, 'rework_runs', 'welding.weld')?.value).toBe(2)
    expect(stepMetric(rows, 'rework_runs', 'welding.edge_prep')).toBeNull()
  })

  it('факты растущей очереди: ключ не вставлен, пост без людей, источник без данных, непригодный ключ, незавершённые', () => {
    const steps = buildSteps(operationsOf(parseProcessSteps(bpmnXml), 'WS-WC'), liveMap(), overview().items)
    const weld = steps.find((s) => s.step.stepKey === 'welding.weld')!
    expect(weld.growing).toBe(true)
    const withWrench = equipment().concat([{ ...equipment()[0]!, equipment_id: 'TW-9', title: 'Ключ с регистрацией момента', station_id: 'WP-WELD-1', condition: 'normal', execution: 'idle', warnings: [] }])
    const facts = queueFacts(weld, buildPosts(posts(), withWrench), registry())
    expect(facts.map((f) => f.kind)).toEqual(['equipment_unusable', 'presence', 'equipment_no_data', 'not_assigned', 'unfinished'])
  })
})

describe('виджет «Участок»', () => {
  it('мастер видит, почему растёт очередь у поста сварки (UJ-7)', async () => {
    mockApi(routes())
    const w = await mountWidget(StationPostsWidget, props)
    expect(w.find('[data-testid="workshop"]').text()).toBe('Участок: Сварочный цех')
    const weld = w.find('[data-step="welding.weld"]')
    expect(weld.attributes('data-growing')).toBe('true')
    expect(weld.find('[data-testid="queue"]').text()).toBe('В очереди: 7')
    expect(plain(weld.find('[data-anomaly="queue_above_norm"]').text())).toContain('Очередь выше нормы узла (4 ч)')
    expect(plain(weld.find('[data-testid="bottleneck"]').text())).toContain('Ограничение линии · 2 ч 10 мин')
    const why = weld.find('[data-testid="why"]')
    expect(why.text()).toContain('Почему растёт очередь у «Сварка фланца с патрубком»')
    expect(why.text()).toContain('Факты участка рядом с очередью — не вывод о причине')
    const facts = why.findAll('[data-testid="why-fact"]').map((f) => f.text())
    expect(facts).toContain('Пост сварки 2: Сварщик W21 — По графику на месте — ключ не вставлен')
    expect(facts).toContain('Пост ОТК сварочного цеха: никто не назначен')
    expect(facts).toContain('Сварочный источник ИС-2: нет данных от источника')
    expect(facts).toContain('Незавершённых операций: 1')
    // Операция без растущей очереди — без блока «почему».
    expect(w.find('[data-step="welding.edge_prep"] [data-testid="why"]').exists()).toBe(false)
  })

  it('длительность против нормы, повторные выполнения против лимита доработок, незавершённые', async () => {
    mockApi(routes())
    const w = await mountWidget(StationPostsWidget, props)
    const weld = w.find('[data-step="welding.weld"]')
    expect(plain(weld.find('[data-testid="duration"]').text())).toContain('1 ч 35 мин')
    expect(plain(weld.find('[data-testid="duration"]').text())).toContain('норма 40 мин–1 ч 30 мин')
    expect(plain(weld.find('[data-testid="duration"]').text())).toContain('выше нормы')
    expect(plain(weld.find('[data-testid="reworks"]').text())).toContain('2')
    expect(plain(weld.find('[data-testid="reworks"]').text())).toContain('лимит доработок на зону: 3')
    expect(plain(weld.find('[data-testid="unfinished"]').text())).toContain('1')
    // Нет показателя — «неизвестно», а не ноль.
    expect(plain(w.find('[data-step="welding.edge_prep"] [data-testid="reworks"]').text())).toContain('Нет данных — неизвестно')
  })

  it('посты: присутствие, текущая деталь, оборудование, ресурс инструмента, текущее выполнение', async () => {
    mockApi(routes())
    const w = await mountWidget(StationPostsWidget, props)
    const p1 = w.find('[data-testid="posts"] [data-workplace="WP-WELD-1"]')
    expect(p1.text()).toContain('Сварщик W22')
    expect(p1.text()).toContain('На месте')
    expect(p1.find('[data-testid="current-item"]').text()).toBe('Ф-015')
    expect(p1.find('[data-equipment="IS-1"]').text()).toContain('Ресурс инструмента: 73 из 75')
    expect(p1.find('[data-testid="equipment-warning"]').text()).toContain('ресурс инструмента 73/75')
    expect(plain(p1.find('[data-run="RUN-15"]').text())).toContain('Сварка фланца с патрубком')
    expect(plain(p1.find('[data-run="RUN-15"]').text())).toContain('норма 40 мин–1 ч 30 мин')
    expect(w.find('[data-testid="posts"] [data-workplace="WP-WELD-2"]').text()).toContain('По графику на месте — ключ не вставлен')
    // Присутствие известно на всех постах, у узлов есть данные — рамка в «норме».
    expect(w.find('.widget-frame').attributes('data-state')).toBe('normal')
  })

  it('узел без данных источника — «оценка невозможна»', async () => {
    mockApi(routes({ 'GET /api/v1/live-map': liveMap({ data_gaps: ['welding.weld'] }) }))
    const w = await mountWidget(StationPostsWidget, props)
    expect(w.find('.widget-frame').attributes('data-state')).toBe('unable_to_assess')
    expect(w.find('[data-step="welding.weld"] [data-testid="data-gap"]').text()).toBe('Нет данных источника — оценка невозможна')
  })

  it('справочник мест недоступен — участок не определён, показано всё предприятие; ошибки разделов — отдельно', async () => {
    mockApi(routes({}, ['GET /api/v1/reference/locations', 'GET /api/v1/workplaces']))
    const w = await mountWidget(StationPostsWidget, props)
    expect(w.find('[data-testid="workshop"]').text()).toContain('Участок не определён')
    expect(w.findAll('[data-step]').map((s) => s.attributes('data-step'))).toEqual(['machining.cnc', 'welding.edge_prep', 'welding.weld'])
    expect(w.find('[data-testid="posts"]').text()).toContain('Не удалось')
  })

  it('ни карта, ни посты не отвечают — «ошибка входа»', async () => {
    mockApi({ 'GET /api/v1/auth/session': foremanSession() })
    const w = await mountWidget(StationPostsWidget, props)
    expect(w.find('.widget-frame').attributes('data-state')).toBe('input_error')
  })
})
