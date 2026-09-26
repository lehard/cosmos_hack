// Правила пульта (FR-129, AD-26, AD-38): текущий прогон, доступные команды по
// состоянию, шаг для человека, скорость ×1…×1000, конфигурация, значения табло.
import { describe, expect, it } from 'vitest'
import {
  actionKey,
  boardRows,
  boardSummary,
  boardValue,
  clampSpeed,
  configError,
  controlsOf,
  defaultConfig,
  displayStep,
  isIdle,
  pickCurrentRun,
  progressOf,
} from '@/entities/run'
import { board, idleRun, run, scenario } from './fixtures'

describe('текущий прогон', () => {
  it('выбранный на пульте, если он ещё есть в списке', () => {
    const runs = [run({ run_id: 'a' }), run({ run_id: 'b', state: 'completed' })]
    expect(pickCurrentRun(runs, 'b')?.run_id).toBe('b')
    expect(pickCurrentRun(runs, 'нет-такого')?.run_id).toBe('a')
  })

  it('иначе — последний незапущенный уступает последнему идущему; нет идущих — последний начатый', () => {
    const runs = [
      run({ run_id: 'old', state: 'completed', started_at: '2026-09-26T08:00:00Z' }),
      run({ run_id: 'new', state: 'stopped', started_at: '2026-09-26T10:00:00Z' }),
      run({ run_id: 'live', state: 'waiting_for_decision', started_at: '2026-09-26T09:00:00Z' }),
    ]
    expect(pickCurrentRun(runs, null)?.run_id).toBe('live')
    expect(pickCurrentRun(runs.slice(0, 2), null)?.run_id).toBe('new')
    expect(pickCurrentRun([], null)).toBeNull()
  })

  it('сценарий по умолчанию без прогона — «не запущен»', () => {
    expect(isIdle(idleRun())).toBe(true)
    expect(isIdle(run())).toBe(false)
  })
})

describe('команды по состоянию прогона', () => {
  it('идёт — пауза и остановка; пауза — продолжение и остановка; ждёт решения — пауза', () => {
    expect(controlsOf(run({ state: 'running' }))).toEqual({ pause: true, resume: false, stop: true })
    expect(controlsOf(run({ state: 'paused' }))).toEqual({ pause: false, resume: true, stop: true })
    expect(controlsOf(run({ state: 'waiting_for_decision' }))).toEqual({ pause: true, resume: false, stop: true })
  })

  it('завершённый, остановленный и незапущенный — никаких команд', () => {
    for (const r of [run({ state: 'completed' }), run({ state: 'stopped' }), run({ state: 'failed' }), idleRun()]) {
      expect(controlsOf(r)).toEqual({ pause: false, resume: false, stop: false })
    }
  })
})

describe('шаг, ход и скорость', () => {
  it('шаги сервера с нуля — на экране с единицы; последний шаг — 100 %', () => {
    expect(displayStep(0)).toBe(1)
    expect(progressOf({ step: 25, steps: 26 })).toBe(100)
    expect(progressOf({ step: 12, steps: 26 })).toBe(50)
    expect(progressOf({ step: 0, steps: 0 })).toBe(0)
  })

  it('скорость ×1…×1000', () => {
    expect(clampSpeed(0)).toBe(1)
    expect(clampSpeed(5000)).toBe(1000)
    expect(clampSpeed(59.6)).toBe(60)
    expect(clampSpeed(null)).toBe(1)
  })
})

describe('конфигурация прогона', () => {
  it('изделия и seed — из определения сценария; нули — «по определению»', () => {
    expect(defaultConfig(scenario())).toEqual({ items: 40, seed: 17, speed: 60, mode: 'interactive' })
    expect(defaultConfig(scenario({ default_items: 0, default_seed: 0 }))).toMatchObject({ items: null, seed: null })
  })

  it('проверка полей до отправки', () => {
    const ok = defaultConfig(scenario())
    expect(configError(ok)).toBeNull()
    expect(configError({ ...ok, items: 10001 })).toBe('widgets.scenarios.config.itemsInvalid')
    expect(configError({ ...ok, seed: -1 })).toBe('widgets.scenarios.config.seedInvalid')
    expect(configError({ ...ok, speed: 1001 })).toBe('widgets.scenarios.config.speedInvalid')
  })
})

describe('табло «ожидалось → получилось»', () => {
  it('значения JSON — строки без кавычек, числа как есть; null — не проверено', () => {
    expect(boardValue('"released"')).toBe('released')
    expect(boardValue('34')).toBe('34')
    expect(boardValue('{"a":1}')).toBe('{"a":1}')
    expect(boardValue(null)).toBeNull()
    expect(boardValue('не JSON')).toBe('не JSON')
  })

  it('строки по шагу; фильтры «не совпало» и «ещё не совпало»', () => {
    const b = board()
    expect(boardRows(b.rows).map((r) => r.assertion_id)).toEqual(['S08-01', 'S12-01', 'S01-01', 'S03-01', 'S05-01'])
    expect(boardRows(b.rows, 'failed').map((r) => r.assertion_id)).toEqual(['S08-01'])
    expect(boardRows(b.rows, 'open').map((r) => r.assertion_id)).toEqual(['S08-01', 'S03-01', 'S05-01'])
  })

  it('сводка: «все совпали» — только когда совпали все строки', () => {
    expect(boardSummary(board())).toMatchObject({ total: 5, passed: 2, failed: 1, pending: 1, notReached: 1, allMatched: false })
    const all = board({ passed: 1, failed: 0, pending: 0, rows: board().rows.filter((r) => r.assertion_id === 'S01-01') })
    expect(boardSummary(all).allMatched).toBe(true)
    expect(boardSummary(board({ rows: [], passed: 0, failed: 0, pending: 0 })).allMatched).toBe(false)
  })

  it('ключ текста действия ожидания', () => {
    expect(actionKey('nonconformity.disposition.set')).toBe('nonconformityDispositionSet')
    expect(actionKey('analysis.risk_scope.narrow')).toBe('analysisRiskScopeNarrow')
  })
})
