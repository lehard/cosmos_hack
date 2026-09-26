/**
 * Часы прогона в шапке столов (Д-85, AD-37): доменное «сейчас» прогона и
 * скорость — «23.09 11:08 · ×60». Сервер отдаёт `clock_at` и `speed`
 * (`simulation.run.read`); между ответами часы идущего прогона досчитываются
 * по скорости, чтобы было видно, что время идёт. Пока прогон ждёт решения
 * человека или стоит на паузе, доменные часы стоят — и в шапке тоже.
 * Только чистые функции: серверного состояния здесь нет.
 */
import { PLANT_TIME_ZONE } from '@/shared/i18n'
import { isActive, isIdle, type Run } from './run'

/**
 * Насколько вперёд (реального времени, мс) досчитывать часы без ответа
 * сервера: дальше — ждать свежих данных, а не убегать.
 */
export const RUN_CLOCK_MAX_AHEAD_MS = 10_000

/** Показывать часы прогона в шапке: есть настоящий незавершённый прогон. */
export const showsRunClock = (run: Pick<Run, 'run_id' | 'scenario_id' | 'state'> | null | undefined): boolean =>
  !!run && isActive(run.state) && !isIdle(run)

/** Часы идут (а не стоят в ожидании человека или на паузе). */
export const runClockTicking = (run: Pick<Run, 'state'>): boolean => run.state === 'running'

/**
 * Доменное «сейчас» прогона, мс UTC: `clock_at` из ответа плюс прошедшее с
 * ответа реальное время, умноженное на скорость (только пока прогон идёт).
 * @param run — ответ `simulation.run.read`
 * @param observedAt — когда получен ответ, мс по часам браузера
 * @param now — сейчас по часам браузера, мс
 */
export function runClockAt(run: Pick<Run, 'clock_at' | 'speed' | 'state'>, observedAt: number, now: number): number {
  const base = Date.parse(run.clock_at)
  if (Number.isNaN(base)) return NaN
  if (!runClockTicking(run) || !observedAt) return base
  const elapsed = Math.min(RUN_CLOCK_MAX_AHEAD_MS, Math.max(0, now - observedAt))
  return base + elapsed * Math.max(1, run.speed)
}

const shortFormat = new Intl.DateTimeFormat('ru-RU', {
  day: '2-digit',
  month: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
  timeZone: PLANT_TIME_ZONE,
})

/** «23.09 11:08» — день, месяц и время завода (без года: в шапке места мало). */
export function formatRunClock(at: number): string {
  if (Number.isNaN(at)) return '—'
  const p = Object.fromEntries(shortFormat.formatToParts(new Date(at)).map((x) => [x.type, x.value]))
  return `${p.day}.${p.month} ${p.hour}:${p.minute}`
}

/**
 * Период перерисовки часов, мс: чтобы минута на экране менялась вовремя, но не
 * чаще раза в 250 мс и не реже раза в секунду.
 */
export const runClockTickMs = (speed: number): number => Math.min(1000, Math.max(250, Math.floor(60_000 / Math.max(1, speed) / 2)))
