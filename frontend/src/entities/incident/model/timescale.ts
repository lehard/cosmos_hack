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
  /** Деления, у которых показать дату (начало скопления или смена суток); нет — только время. */
  dayTicks?: ReadonlySet<number>
  /** Свёрнутые промежутки без событий: где на шкале и сколько длились. */
  breaks?: readonly ScaleBreak[]
}

/** Свёрнутый промежуток без событий. */
export interface ScaleBreak {
  /** Левый край разрыва, %. */
  left: number
  /** Ширина разрыва, %. */
  width: number
  /** Начало и конец промежутка, мс эпохи. */
  from: number
  to: number
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

const HOUR = 60 * MINUTE
/** Промежуток без событий дольше этого — сворачивается в разрыв. */
const GAP_MS = 3 * HOUR
/** Ширина разрыва на шкале, %. */
const BREAK_PCT = 6

/**
 * Сжатая шкала (UI-33): события идут скоплениями (операция — минуты, между
 * операциями — дни), и на линейной шкале двадцать минут сварки сжимаются в
 * точку. Промежутки без событий дольше трёх часов сворачиваются в разрыв
 * постоянной ширины («2 дн. 19 ч без событий»); скопления делят остальное
 * место пропорционально длительности, но каждое — не меньше заметной доли.
 * Без длинных промежутков — обычная линейная шкала.
 * @param moments — все моменты на экране (записи, окно, операция)
 */
export function makeCompressedScale(moments: readonly (string | number)[], maxTicks = 8): TimeScale {
  const ms = [...new Set(moments.map(toMs).filter(Number.isFinite))].sort((a, b) => a - b)
  if (ms.length < 2) return makeTimeScale(moments, maxTicks)
  // Скопления: соседние моменты ближе GAP_MS.
  const clusters: { a: number; b: number }[] = [{ a: ms[0]!, b: ms[0]! }]
  for (const t of ms.slice(1)) {
    const last = clusters[clusters.length - 1]!
    if (t - last.b > GAP_MS) clusters.push({ a: t, b: t })
    else last.b = t
  }
  if (clusters.length === 1) return makeTimeScale(moments, maxTicks)

  // Поля скопления: 10 % длительности, не меньше 5 минут.
  const padded = clusters.map((c) => {
    const pad = Math.max((c.b - c.a) * 0.1, 5 * MINUTE)
    return { a: c.a - pad, b: c.b + pad }
  })
  const free = 100 - BREAK_PCT * (padded.length - 1)
  const durs = padded.map((c) => c.b - c.a)
  const total = durs.reduce((x, y) => x + y, 0)
  // Не меньше половины равной доли — короткое скопление остаётся читаемым.
  const floor = free / padded.length / 2
  const raw = durs.map((d) => Math.max((d / total) * free, floor))
  const k = free / raw.reduce((x, y) => x + y, 0)
  const widths = raw.map((w) => w * k)

  const segs: { a: number; b: number; left: number; width: number }[] = []
  const breaks: ScaleBreak[] = []
  let x = 0
  padded.forEach((c, i) => {
    segs.push({ ...c, left: x, width: widths[i]! })
    x += widths[i]!
    const next = padded[i + 1]
    if (next) {
      breaks.push({ left: x, width: BREAK_PCT, from: c.b, to: next.a })
      x += BREAK_PCT
    }
  })

  const pos = (t: string | number) => {
    const v = toMs(t)
    if (!Number.isFinite(v)) return 0
    const first = segs[0]!
    const last = segs[segs.length - 1]!
    if (v <= first.a) return 0
    if (v >= last.b) return 100
    for (let i = 0; i < segs.length; i++) {
      const s = segs[i]!
      if (v <= s.b) {
        if (v >= s.a) return s.left + ((v - s.a) / (s.b - s.a)) * s.width
        // Внутри разрыва перед скоплением — пропорционально в разрыве.
        const br = breaks[i - 1]!
        return br.left + ((v - br.from) / (br.to - br.from)) * br.width
      }
    }
    return 100
  }

  // Деления: по каждому скоплению — шаг по его длительности и ширине; первое деление скопления — с датой.
  const ticks: number[] = []
  const dayTicks = new Set<number>()
  segs.forEach((s) => {
    const n = Math.max(2, Math.round((s.width / 100) * maxTicks))
    const span = s.b - s.a
    const step = (STEPS.find((m) => span / (m * MINUTE) <= n) ?? STEPS[STEPS.length - 1]!) * MINUTE
    let firstOfSeg = true
    for (let t = Math.ceil(s.a / step) * step; t <= s.b; t += step) {
      ticks.push(t)
      if (firstOfSeg) dayTicks.add(t)
      firstOfSeg = false
    }
  })
  return { start: padded[0]!.a, end: padded[padded.length - 1]!.b, pos, ticks, dayTicks, breaks }
}
