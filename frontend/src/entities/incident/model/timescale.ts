/**
 * Общая шкала времени трёх дорожек разбора обстоятельств (FR-153): одна функция
 * «время → позиция в процентах» на все дорожки, окно и операцию — поэтому
 * дорожки синхронны по построению.
 */

/** Шкала времени. */
export interface TimeScale {
  /** Начало шкалы, мс эпохи. */
  start: number
  /** Конец шкалы, мс эпохи. */
  end: number
  /** Позиция момента на шкале, 0…100 %. */
  pos: (t: string | number) => number
  /** Деления шкалы, мс эпохи. */
  ticks: number[]
}

const MINUTE = 60_000
/** Шаги делений, минуты. */
const STEPS = [1, 2, 5, 10, 15, 30, 60, 120, 240, 480, 720, 1440]

/** Момент RFC 3339 → мс эпохи; NaN — время не разобрано. */
export const toMs = (t: string | number): number => (typeof t === 'number' ? t : Date.parse(t))

/**
 * Шкала, вмещающая все моменты с полями по краям.
 * @param moments — все моменты на экране (записи, окно, операция)
 * @param maxTicks — не больше стольких делений
 */
export function makeTimeScale(moments: readonly (string | number)[], maxTicks = 8): TimeScale {
  const ms = moments.map(toMs).filter(Number.isFinite)
  let lo = ms.length ? Math.min(...ms) : 0
  let hi = ms.length ? Math.max(...ms) : 0
  if (hi - lo < MINUTE) {
    // одна точка или все в одну минуту — шкала в пять минут вокруг
    lo -= 2.5 * MINUTE
    hi += 2.5 * MINUTE
  }
  const pad = (hi - lo) * 0.04
  const start = lo - pad
  const end = hi + pad
  const span = end - start

  const step = (STEPS.find((s) => span / (s * MINUTE) <= maxTicks) ?? STEPS[STEPS.length - 1]!) * MINUTE
  const ticks: number[] = []
  for (let t = Math.ceil(start / step) * step; t <= end; t += step) ticks.push(t)

  const pos = (t: string | number) => {
    const v = toMs(t)
    return Number.isFinite(v) ? Math.min(100, Math.max(0, ((v - start) / span) * 100)) : 0
  }
  return { start, end, pos, ticks }
}

/**
 * Раскладка подписей по ярусам, чтобы подписи соседних отметок не наезжали:
 * жадно — в первый ярус, где предыдущая подпись кончилась левее.
 * @param positions — позиции отметок, % (в порядке времени)
 * @param widths — оценка ширины подписи, %
 * @returns ярус каждой отметки (0 — верхний)
 */
export function stackLabels(positions: readonly number[], widths: readonly number[]): number[] {
  const ends: number[] = []
  return positions.map((p, i) => {
    let level = ends.findIndex((e) => e <= p)
    if (level < 0) level = ends.length
    ends[level] = p + (widths[i] ?? 0)
    return level
  })
}
