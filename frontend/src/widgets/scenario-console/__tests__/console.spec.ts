// Пульт демонстрации (FR-129, FR-152; AD-26, AD-37, AD-38): список сценариев и
// конфигурация, запуск, пауза, продолжение, скорость, шаг и доменное время,
// ожидание решения человека, кнопки цифрового стенда; команды — через
// сгенерированный клиент с command_id, basis_seq и policy_seq.
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defaultConfig, useRunFocusStore } from '@/entities/run'
import { adminSession, mockApi, mountWidget, settle } from '@/entities/run/__tests__/api'
import { board, idleRun, injections, receipt, run, scenario, scenarios } from '@/entities/run/__tests__/fixtures'
import { i18n } from '@/shared/i18n'
import { useMomentStore } from '@/shared/model/moment'
import ScenarioConsoleView from '../ui/ScenarioConsoleView.vue'
import ScenarioConsoleWidget from '../ui/ScenarioConsoleWidget.vue'

let pinia: ReturnType<typeof createPinia>
beforeEach(() => {
  pinia = createPinia()
  setActivePinia(pinia)
})
afterEach(() => vi.unstubAllGlobals())

const norm = (x: string) => x.replace(/\s+/g, ' ')
const mountView = (props = {}) =>
  mount(ScenarioConsoleView, {
    props: { scenarios: scenarios(), selectedScenario: 'flange-bad-day', config: defaultConfig(scenario()), run: run(), injections: injections(), ...props },
    global: { plugins: [pinia, i18n] },
  })

describe('пульт — представление', () => {
  it('список сценариев: название, ссылки на кейс, остановки на решениях и строки табло; выбор', async () => {
    const w = mountView()
    const items = w.findAll('li.scenario')
    expect(items).toHaveLength(2)
    expect(items[0]!.attributes('aria-selected')).toBe('true')
    expect(items[0]!.text()).toContain('Кейс: §1.5, §5.1')
    expect(items[0]!.text()).toContain('Остановок на решении человека: 6')
    expect(items[0]!.text()).toContain('Строк табло: 22')
    await items[1]!.trigger('click')
    expect(w.emitted('select-scenario')?.[0]).toEqual(['S06'])
  })

  it('конфигурация: изделия и seed из определения, скорость кнопками, режим; запуск', async () => {
    const w = mountView()
    const cfg = w.find('[data-testid="config"]')
    expect((cfg.find('[data-testid="cfg-items"] input').element as HTMLInputElement).value).toBe('40')
    expect((cfg.find('[data-testid="cfg-seed"] input').element as HTMLInputElement).value).toBe('17')
    expect(cfg.text()).toContain('Скорость времени: ×60')
    await cfg.find('[data-testid="cfg-speed-1000"]').trigger('click')
    expect(w.emitted('update:config')?.[0]?.[0]).toMatchObject({ speed: 1000, items: 40, seed: 17 })
    await cfg.find('[data-testid="cfg-mode-autocheck"] input').setValue(true)
    expect(w.emitted('update:config')?.[1]?.[0]).toMatchObject({ mode: 'autocheck' })
    await cfg.trigger('submit')
    expect(w.emitted('start')).toHaveLength(1)
  })

  it('неверная конфигурация — запуск выключен и сказано почему', () => {
    const w = mountView({ config: { ...defaultConfig(scenario()), seed: -5 } })
    expect(w.find('[data-testid="cfg-error"]').text()).toBe('Начальное число — целое не меньше 0')
    expect(w.find('[data-testid="start"]').attributes('disabled')).toBeDefined()
  })

  it('идущий прогон: номер, состояние, шаг с единицы, доменное время, скорость, счёт табло; пауза и остановка', async () => {
    const w = mountView()
    expect(w.find('[data-testid="run-id"]').text()).toBe('Прогон fx-1a2b3c4d')
    expect(w.find('[data-testid="run-state"]').text()).toBe('Идёт')
    expect(w.find('[data-testid="run-step"]').text()).toBe('Шаг 8 из 26')
    expect(norm(w.find('[data-testid="run-clock"]').text())).toBe('23.09.2026, 10:31')
    expect(w.find('[data-testid="run-speed"]').text()).toBe('×60')
    expect(w.find('[data-testid="run-board"]').text()).toBe('Табло: совпало 3 из 5')
    expect(w.find('[data-testid="resume"]').exists()).toBe(false)
    await w.find('[data-testid="pause"]').trigger('click')
    await w.find('[data-testid="stop"]').trigger('click')
    expect(w.emitted('control')).toEqual([['pause'], ['stop']])
    await w.find('[data-testid="speed-1000"]').trigger('click')
    expect(w.emitted('speed')?.[0]).toEqual([1000])
  })

  it('пауза: «все рабочие столы показывают состояние на …» и кнопка «Продолжить»', async () => {
    const w = mountView({ run: run({ state: 'paused' }) })
    expect(norm(w.find('[data-testid="paused"]').text())).toBe('Пауза: все рабочие столы показывают состояние на 23.09.2026, 10:31')
    expect(w.find('[data-testid="pause"]').exists()).toBe(false)
    await w.find('[data-testid="resume"]').trigger('click')
    expect(w.emitted('control')).toEqual([['resume']])
  })

  it('ждёт решения: чья роль, что решить и над каким объектом', () => {
    const w = mountView({
      run: run({ state: 'waiting_for_decision', waiting_for: { role: 'quality_inspector', action: 'nonconformity.nonconformity.confirm', object_id: 'ENT01:fx-1a2b3c4d/F-017' } }),
    })
    const text = norm(w.find('[data-testid="waiting"]').text())
    expect(text).toContain('Сценарий ждёт решения: Контролёр качества — подтвердить сигнал как несоответствие')
    expect(text).toContain('объект ENT01:fx-1a2b3c4d/F-017')
    expect(text).toContain('Войдите под этой ролью')
  })

  it('прогон не запускался: только «Запустить», кнопки стенда недоступны', () => {
    const w = mountView({ run: idleRun() })
    expect(w.find('[data-testid="idle"]').text()).toContain('прогон не запущен')
    expect(w.find('[data-testid="pause"]').exists()).toBe(false)
    expect(w.find('[data-testid="resume"]').exists()).toBe(false)
    expect(w.find('[data-testid="stop"]').exists()).toBe(false)
    expect(w.find('[data-testid="stand"]').text()).toContain('Кнопки стенда доступны, когда прогон запущен')
    expect(w.findAll('[data-injection]')).toHaveLength(0)
  })

  it('воспроизведение — команды выключены', () => {
    const w = mountView({ canAct: false })
    expect(w.find('[data-testid="start"]').attributes('disabled')).toBeDefined()
    expect(w.find('[data-testid="pause"]').attributes('disabled')).toBeDefined()
    expect(w.find('[data-testid="stop"]').attributes('disabled')).toBeDefined()
  })

  it('стенд: недоступная кнопка выключена; повтору нужен номер события; подделка — только в демо', async () => {
    const w = mountView()
    const dup = w.find('[data-injection="duplicate_event"]')
    expect(dup.find('button').attributes('disabled')).toBeDefined()
    await dup.find('input').setValue('EV-WS2-0412')
    expect(dup.find('button').attributes('disabled')).toBeUndefined()
    await dup.find('button').trigger('click')
    expect(w.emitted('inject')?.[0]).toEqual(['duplicate_event', 'EV-WS2-0412'])
    await w.find('[data-injection="corrupt_frame"] button').trigger('click')
    expect(w.emitted('inject')?.[1]).toEqual(['corrupt_frame', undefined])
    const fault = w.find('[data-injection="machine_fault"]')
    expect(fault.find('button').attributes('disabled')).toBeDefined()
    expect(fault.text()).toContain('Недоступно в текущем состоянии прогона')
    expect(w.find('[data-injection="tamper_outside"]').text()).toContain('только в демонстрационном режиме')
  })

  it('стенд: цель необязательна — пусто, сервер берёт последнее подходящее событие; указанная уходит с кнопкой', async () => {
    const w = mountView()
    const frame = w.find('[data-injection="corrupt_frame"]')
    expect(frame.find('input').attributes('placeholder')).toBe('Номер события (пусто — последнее подходящее)')
    expect(frame.find('button').attributes('disabled')).toBeUndefined()
    await frame.find('input').setValue('0b7d6c1e-8f4a-5d2b-9c3e-1a2b3c4d5e6f')
    await frame.find('button').trigger('click')
    expect(w.emitted('inject')?.[0]).toEqual(['corrupt_frame', '0b7d6c1e-8f4a-5d2b-9c3e-1a2b3c4d5e6f'])
  })
})

describe('пульт — контейнер', () => {
  const routes = (over: Record<string, unknown> = {}) => ({
    'GET /api/v1/scenarios': { items: scenarios() },
    'GET /api/v1/runs': { items: [run()] },
    'GET /api/v1/runs/fx-1a2b3c4d': run(),
    'GET /api/v1/runs/fx-1a2b3c4d/injections': { items: injections() },
    'GET /api/v1/runs/fx-1a2b3c4d/board': board(),
    'GET /api/v1/auth/session': adminSession,
    'POST /api/v1/runs/fx-1a2b3c4d/pause': (b: unknown) => receipt((b as { command_id: string }).command_id),
    'POST /api/v1/runs/fx-1a2b3c4d/speed': (b: unknown) => receipt((b as { command_id: string }).command_id),
    'POST /api/v1/runs/fx-1a2b3c4d/injections': (b: unknown) => receipt((b as { command_id: string }).command_id, 2),
    'POST /api/v1/scenarios/S06/runs': (b: unknown) => receipt((b as { command_id: string }).command_id),
    ...over,
  })
  const props = { widgetId: 'scenario-console', titleKey: 'desks.testScenarios' }

  it('читает сценарии и текущий прогон; метка «Демо на заготовках»', async () => {
    mockApi(routes())
    const w = await mountWidget(ScenarioConsoleWidget, props, pinia)
    expect(w.find('.widget-frame').attributes('data-mode')).toBe('fixtures')
    expect(w.findAll('li.scenario')).toHaveLength(2)
    expect(w.find('[data-testid="run-step"]').text()).toBe('Шаг 8 из 26')
    // Выбран сценарий текущего прогона.
    expect(w.find('li.scenario[data-id="flange-bad-day"]').attributes('aria-selected')).toBe('true')
  })

  it('пауза и скорость уходят командами с command_id (UUIDv7), basis_seq прогона и policy_seq сеанса', async () => {
    const calls = mockApi(routes())
    const w = await mountWidget(ScenarioConsoleWidget, props, pinia)
    await w.find('[data-testid="pause"]').trigger('click')
    await settle()
    await w.find('[data-testid="speed-300"]').trigger('click')
    await settle()
    const posts = calls.filter((c) => c.method === 'POST')
    expect(posts.map((c) => c.path)).toEqual(['/api/v1/runs/fx-1a2b3c4d/pause', '/api/v1/runs/fx-1a2b3c4d/speed'])
    expect(posts[0]!.body).toMatchObject({ basis_seq: 79999, policy_seq: 7, workplace_id: 'WP-ADM' })
    expect((posts[0]!.body as { command_id: string }).command_id).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/)
    expect(posts[1]!.body).toMatchObject({ speed: 300 })
    expect((posts[0]!.body as { command_id: string }).command_id).not.toBe((posts[1]!.body as { command_id: string }).command_id)
    // После команды прогон перечитан.
    expect(calls.filter((c) => c.path === '/api/v1/runs/fx-1a2b3c4d').length).toBeGreaterThan(1)
  })

  it('запуск другого сценария: конфигурация уходит в simulation.run.start, пустые поля — «по определению»', async () => {
    const calls = mockApi(routes())
    const w = await mountWidget(ScenarioConsoleWidget, props, pinia)
    await w.find('li.scenario[data-id="S06"]').trigger('click')
    await settle()
    expect(useRunFocusStore().scenarioId).toBe('S06')
    await w.find('[data-testid="cfg-speed-1000"]').trigger('click')
    await w.find('[data-testid="config"]').trigger('submit')
    await settle()
    const start = calls.find((c) => c.method === 'POST' && c.path === '/api/v1/scenarios/S06/runs')
    expect(start?.body).toMatchObject({ mode: 'interactive', speed: 1000, policy_seq: 7 })
    expect(start?.body).not.toHaveProperty('items')
    expect(start?.body).not.toHaveProperty('seed')
  })

  it('кнопка стенда: simulation.injection.apply с целевым событием; итог — сколько записей внесено', async () => {
    const calls = mockApi(routes())
    const w = await mountWidget(ScenarioConsoleWidget, props, pinia)
    const dup = w.find('[data-injection="duplicate_event"]')
    await dup.find('input').setValue('EV-1')
    await dup.find('button').trigger('click')
    await settle()
    expect(calls.find((c) => c.method === 'POST' && c.path.endsWith('/injections'))?.body).toMatchObject({ injection: 'duplicate_event', target_event_id: 'EV-1' })
    expect(w.find('[data-testid="injected"]').text()).toBe('Сбой внесён: записей 2')
  })

  it('отказ сервера на команду — текст ошибки по коду, пульт остаётся', async () => {
    mockApi(routes({ 'POST /api/v1/runs/fx-1a2b3c4d/pause': { __status: 404, __body: { type: 'urn:ant:problem:simulation.run_not_found', title: 'Нет прогона', status: 404, code: 'simulation.run_not_found' } } }))
    const w = await mountWidget(ScenarioConsoleWidget, props, pinia)
    await w.find('[data-testid="pause"]').trigger('click')
    await settle()
    expect(w.find('[data-testid="command-error"]').text()).toContain('Команда пульта не выполнена')
    expect(w.find('[data-testid="run-step"]').exists()).toBe(true)
  })

  it('воспроизведение (момент в прошлом): команды выключены, прогон читается на момент', async () => {
    const calls = mockApi(routes())
    useMomentStore().travel('2026-09-23T08:00:00Z')
    const w = await mountWidget(ScenarioConsoleWidget, props, pinia)
    expect(w.find('[data-testid="pause"]').attributes('disabled')).toBeDefined()
    expect(calls.find((c) => c.path === '/api/v1/runs/fx-1a2b3c4d')?.query.get('as_of')).toBe('2026-09-23T08:00:00Z')
  })

  it('сценарии не читаются — «ошибка входа», а не выдуманные данные', async () => {
    mockApi({})
    const w = await mountWidget(ScenarioConsoleWidget, props, pinia)
    expect(w.find('.widget-frame').attributes('data-state')).toBe('input_error')
    expect(w.find('[data-testid="scenario-console"]').exists()).toBe(false)
  })
})
