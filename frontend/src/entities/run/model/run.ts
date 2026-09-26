/**
 * Прогон тестового сценария — чистые правила пульта (FR-129, AD-26, AD-37, AD-38):
 * какой прогон считать текущим, какие команды доступны в каком состоянии,
 * номер шага для человека, скорость ×1…×1000, разбор значений табло.
 * Серверного состояния здесь нет — только функции над ответами API.
 */
import type { Board, BoardRow, BoardRowStatus, Injection, Run, RunState, Scenario } from '@/shared/api/generated/model'

export type { Board, BoardRow, BoardRowStatus, Injection, Run, RunState, Scenario }

/** Границы ускорения доменных часов (контракт SetSpeed: ×1…×1000, AD-37). */
export const SPEED_MIN = 1
export const SPEED_MAX = 1000

/** Готовые скорости пульта: реальное время, «минута за секунду», быстрый прогон. */
export const SPEED_PRESETS: readonly number[] = [1, 10, 60, 300, 1000]

/** Изделий в прогоне — не больше, чем допускает контракт StartRun. */
export const ITEMS_MAX = 10000

/** Прогон ещё идёт: его можно ставить на паузу, продолжать, останавливать. */
export const isActive = (state: RunState): boolean => state === 'running' || state === 'paused' || state === 'waiting_for_decision'

/**
 * Прогон не запускался: сервер показывает сценарий по умолчанию под id
 * сценария (у настоящего прогона свой run_id, AD-38). Пульт тогда предлагает
 * только «Запустить».
 */
export const isIdle = (run: Pick<Run, 'run_id' | 'scenario_id'>): boolean => run.run_id === run.scenario_id

/** Команда пульта над прогоном. */
export type RunControlKind = 'pause' | 'resume' | 'stop'

/**
 * Какие команды доступны прогону (FR-129): пауза — у идущего и ждущего
 * решения, продолжение — у стоящего на паузе, остановка — у любого
 * незавершённого. Завершённый, остановленный и незапущенный — никаких.
 */
export function controlsOf(run: Pick<Run, 'run_id' | 'scenario_id' | 'state'>): Record<RunControlKind, boolean> {
  const live = isActive(run.state) && !isIdle(run)
  return {
    pause: live && (run.state === 'running' || run.state === 'waiting_for_decision'),
    resume: live && run.state === 'paused',
    stop: live,
  }
}

/**
 * Текущий прогон пульта: выбранный (если он ещё есть в списке), иначе
 * последний незавершённый, иначе последний начатый.
 * @param runs — `simulation.run.list`
 * @param preferred — выбранный на пульте run_id
 */
export function pickCurrentRun(runs: readonly Run[], preferred: string | null | undefined): Run | null {
  if (preferred) {
    const chosen = runs.find((r) => r.run_id === preferred)
    if (chosen) return chosen
  }
  const byStart = [...runs].sort((a, b) => Date.parse(b.started_at) - Date.parse(a.started_at))
  return byStart.find((r) => isActive(r.state) && !isIdle(r)) ?? byStart.find((r) => isActive(r.state)) ?? byStart[0] ?? null
}

/**
 * Номер шага для человека: сервер считает шаги с нуля, на экране — с единицы
 * (и на пульте, и в табло одинаково).
 */
export const displayStep = (step: number): number => step + 1

/** Доля пройденных шагов 0…100 для полосы хода сценария. */
export function progressOf(run: Pick<Run, 'step' | 'steps'>): number {
  if (run.steps <= 0) return 0
  return Math.min(100, Math.max(0, Math.round((displayStep(run.step) / run.steps) * 100)))
}

/** Скорость в допустимых границах; не число — ×1. */
export function clampSpeed(v: number | null | undefined): number {
  if (typeof v !== 'number' || !Number.isFinite(v)) return SPEED_MIN
  return Math.min(SPEED_MAX, Math.max(SPEED_MIN, Math.round(v)))
}

/** Конфигурация запуска на пульте: пусто — «по определению сценария». */
export interface RunConfig {
  items: number | null
  seed: number | null
  speed: number
  mode: 'interactive' | 'autocheck'
}

/** Конфигурация по умолчанию для сценария: изделия и seed — из определения. */
export const defaultConfig = (scenario: Scenario | null | undefined): RunConfig => ({
  items: scenario && scenario.default_items > 0 ? scenario.default_items : null,
  seed: scenario && scenario.default_seed > 0 ? scenario.default_seed : null,
  speed: 60,
  mode: 'interactive',
})

/** Ошибка конфигурации — ключ текста; null — можно запускать. */
export function configError(c: RunConfig): string | null {
  if (c.items !== null && (!Number.isInteger(c.items) || c.items < 0 || c.items > ITEMS_MAX)) return 'widgets.scenarios.config.itemsInvalid'
  if (c.seed !== null && (!Number.isInteger(c.seed) || c.seed < 0)) return 'widgets.scenarios.config.seedInvalid'
  if (c.speed !== clampSpeed(c.speed)) return 'widgets.scenarios.config.speedInvalid'
  return null
}

/**
 * Значение табло (JSON-строка контракта) для показа: строка — без кавычек,
 * число и логическое — как есть, объект — компактным JSON. null — «ещё не проверено».
 */
export function boardValue(raw: string | null): string | null {
  if (raw === null) return null
  try {
    const v: unknown = JSON.parse(raw)
    if (typeof v === 'string') return v
    if (typeof v === 'number' || typeof v === 'boolean') return String(v)
    return JSON.stringify(v)
  } catch {
    return raw
  }
}

/** Фильтр табло: все строки, только несовпадения, только ещё не совпавшие. */
export type BoardFilter = 'all' | 'failed' | 'open'

/** Порядок строк табло: по шагу сценария, затем по id утверждения. */
export function boardRows(rows: readonly BoardRow[], filter: BoardFilter = 'all'): BoardRow[] {
  const keep = (r: BoardRow): boolean => filter === 'all' || (filter === 'failed' ? r.status === 'failed' : r.status !== 'passed')
  return rows.filter(keep).sort((a, b) => a.step - b.step || a.assertion_id.localeCompare(b.assertion_id))
}

/** Сводка табло: все ли совпали, сколько строк каждого состояния. */
export function boardSummary(board: Pick<Board, 'rows' | 'passed' | 'failed' | 'pending'>) {
  const total = board.rows.length
  const notReached = board.rows.filter((r) => r.status === 'not_reached').length
  return { total, passed: board.passed, failed: board.failed, pending: board.pending, notReached, allMatched: total > 0 && board.passed === total }
}

/** Тон состояния прогона (палитра словаря статусов, AD-30). */
export const RUN_STATE_TONE: Record<RunState, 'info' | 'attention' | 'success' | 'neutral' | 'danger'> = {
  running: 'info',
  paused: 'neutral',
  waiting_for_decision: 'attention',
  completed: 'success',
  stopped: 'neutral',
  failed: 'danger',
}

/** Тон строки табло: совпало — успех, не совпало — опасность, остальное — нейтрально. */
export const BOARD_STATUS_TONE: Record<BoardRowStatus, 'success' | 'danger' | 'neutral' | 'muted'> = {
  passed: 'success',
  failed: 'danger',
  pending: 'neutral',
  not_reached: 'muted',
}

/**
 * Ключ текста для кода x-ant-action: `nonconformity.disposition.set` →
 * `nonconformityDispositionSet` (сегменты и snake_case склеиваются в camelCase).
 */
export const actionKey = (action: string): string =>
  action
    .split(/[._]/)
    .filter(Boolean)
    .map((s, i) => (i === 0 ? s : s[0]!.toUpperCase() + s.slice(1)))
    .join('')
