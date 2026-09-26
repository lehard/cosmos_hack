// Часы прогона в шапке (Д-85, AD-37): досчёт доменного времени по скорости
// между ответами сервера, остановка в ожидании человека и на паузе, формат.
import { describe, expect, it } from 'vitest'
import { RUN_CLOCK_MAX_AHEAD_MS, formatRunClock, runClockAt, runClockTickMs, showsRunClock } from '../model/clock'
import { idleRun, run } from './fixtures'

describe('часы прогона', () => {
  const base = Date.parse('2026-09-23T07:31:00Z')

  it('идущий прогон: реальная секунда при ×60 — доменная минута', () => {
    expect(runClockAt(run({ speed: 60 }), 1_000, 2_000)).toBe(base + 60_000)
    expect(runClockAt(run({ speed: 1 }), 1_000, 2_000)).toBe(base + 1_000)
  })

  it('ждёт решения человека или на паузе — часы стоят', () => {
    expect(runClockAt(run({ state: 'waiting_for_decision' }), 1_000, 50_000)).toBe(base)
    expect(runClockAt(run({ state: 'paused' }), 1_000, 50_000)).toBe(base)
  })

  it('без ответа сервера не убегают дальше предела', () => {
    expect(runClockAt(run({ speed: 60 }), 1_000, 1_000 + 10 * RUN_CLOCK_MAX_AHEAD_MS)).toBe(base + RUN_CLOCK_MAX_AHEAD_MS * 60)
    expect(runClockAt(run(), 5_000, 1_000)).toBe(base)
  })

  it('формат «23.09 10:31» — время завода, без года', () => {
    expect(formatRunClock(base)).toBe('23.09 10:31')
    expect(formatRunClock(NaN)).toBe('—')
  })

  it('видны только у настоящего незавершённого прогона', () => {
    expect(showsRunClock(run())).toBe(true)
    expect(showsRunClock(run({ state: 'waiting_for_decision' }))).toBe(true)
    expect(showsRunClock(run({ state: 'completed' }))).toBe(false)
    expect(showsRunClock(idleRun())).toBe(false)
    expect(showsRunClock(null)).toBe(false)
  })

  it('перерисовка: не реже раза в секунду и не чаще 4 раз в секунду', () => {
    expect(runClockTickMs(1)).toBe(1000)
    expect(runClockTickMs(60)).toBe(500)
    expect(runClockTickMs(1000)).toBe(250)
  })
})
