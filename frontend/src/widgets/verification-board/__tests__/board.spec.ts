// Табло «ожидалось → получилось» (AD-26, кейс §5.1): строки утверждений со
// значениями, четыре состояния строки, сводка, фильтр несовпадений; контейнер
// читает табло прогона, выбранного на пульте.
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useRunFocusStore } from '@/entities/run'
import { mockApi, mountWidget, settle } from '@/entities/run/__tests__/api'
import { board, run } from '@/entities/run/__tests__/fixtures'
import { i18n } from '@/shared/i18n'
import VerificationBoardView from '../ui/VerificationBoardView.vue'
import VerificationBoardWidget from '../ui/VerificationBoardWidget.vue'

let pinia: ReturnType<typeof createPinia>
beforeEach(() => {
  pinia = createPinia()
  setActivePinia(pinia)
})
afterEach(() => vi.unstubAllGlobals())

const mountView = (props = {}) => mount(VerificationBoardView, { props: { board: board(), filter: 'all', ...props }, global: { plugins: [pinia, i18n] } })

describe('табло — представление', () => {
  it('строки по шагу: состояние, ожидалось → получилось, чем проверяется', () => {
    const rows = mountView().findAll('li.row')
    expect(rows.map((r) => r.attributes('data-id'))).toEqual(['S08-01', 'S12-01', 'S01-01', 'S03-01', 'S05-01'])
    const failed = rows[0]!
    expect(failed.attributes('data-status')).toBe('failed')
    expect(failed.text()).toContain('Не совпало')
    expect(failed.text()).toContain('Шаг 5')
    expect(failed.find('[data-testid="expected"]').text()).toBe('rejected')
    expect(failed.find('[data-testid="actual"]').text()).toBe('under_review')
    expect(failed.text()).toContain('Проверка: quality.signal.read /state')
    expect(rows[3]!.text()).toContain('Ждёт проверки')
    expect(rows[3]!.find('[data-testid="actual"]').text()).toBe('ещё не проверено')
    expect(rows[4]!.text()).toContain('Сценарий не дошёл до шага')
  })

  it('сводка: совпало N из M, несовпадения, ждут, не дошли', () => {
    const s = mountView().find('[data-testid="board-summary"]').text()
    expect(s).toContain('Совпало 2 из 5')
    expect(s).toContain('Не совпало: 1')
    expect(s).toContain('Ждут проверки: 1')
    expect(s).toContain('Не дошли: 1')
  })

  it('все совпали — так и сказано', () => {
    const b = board({ passed: 1, failed: 0, pending: 0, rows: board().rows.filter((r) => r.status === 'passed').slice(0, 1) })
    expect(mountView({ board: b }).find('[data-testid="all-matched"]').text()).toBe('Все проверки сценария совпали')
  })

  it('фильтр: только несовпадения', async () => {
    const w = mountView({ filter: 'failed' })
    expect(w.findAll('li.row').map((r) => r.attributes('data-id'))).toEqual(['S08-01'])
    await w.find('[data-testid="filter-open"] input').setValue(true)
    expect(w.emitted('update:filter')?.[0]).toEqual(['open'])
  })
})

describe('табло — контейнер', () => {
  const props = { widgetId: 'verification-board', titleKey: 'desks.verificationBoard' }

  it('табло текущего прогона; метка режима', async () => {
    const calls = mockApi({
      'GET /api/v1/runs': { items: [run()] },
      'GET /api/v1/runs/fx-1a2b3c4d': run({ state: 'paused' }),
      'GET /api/v1/runs/fx-1a2b3c4d/board': board(),
    })
    const w = await mountWidget(VerificationBoardWidget, props, pinia)
    expect(w.find('.widget-frame').attributes('data-mode')).toBe('fixtures')
    expect(w.findAll('li.row')).toHaveLength(5)
    expect(calls.some((c) => c.path === '/api/v1/runs/fx-1a2b3c4d/board')).toBe(true)
  })

  it('прогон, выбранный на пульте, — его табло', async () => {
    const calls = mockApi({
      'GET /api/v1/runs': { items: [run(), run({ run_id: 'fx-2', state: 'completed' })] },
      'GET /api/v1/runs/fx-2': run({ run_id: 'fx-2', state: 'completed' }),
      'GET /api/v1/runs/fx-2/board': board({ run_id: 'fx-2' }),
      'GET /api/v1/runs/fx-1a2b3c4d': run(),
      'GET /api/v1/runs/fx-1a2b3c4d/board': board(),
    })
    useRunFocusStore().selectRun('fx-2')
    await mountWidget(VerificationBoardWidget, props, pinia)
    await settle()
    expect(calls.some((c) => c.path === '/api/v1/runs/fx-2/board')).toBe(true)
    expect(calls.some((c) => c.path === '/api/v1/runs/fx-1a2b3c4d/board')).toBe(false)
  })

  it('прогонов нет — «табло появится, когда будет прогон»', async () => {
    mockApi({ 'GET /api/v1/runs': { items: [] } })
    const w = await mountWidget(VerificationBoardWidget, props, pinia)
    expect(w.text()).toContain('Табло появится, когда будет прогон')
  })
})
