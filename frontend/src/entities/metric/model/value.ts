/**
 * Значение показателя (`MetricValue` контракта) → текст экрана.
 *
 * Числа в контракте — целое + масштаб (соглашение «Числа в домене»: без float):
 * значение = value × 10^(−scale). Доли приходят в базисных пунктах (`bp`,
 * 10 000 = 100 %). Длительности несут происхождение: передано источником или
 * вычислено системой (кейс §5.2, FR-88, соглашение «Длительности»).
 *
 * Здесь ничего не пересчитывается по смыслу (AD-21, NFR-UI-4): только перевод
 * единиц в текст. «Оценка невозможна» (`unknown`) — не ноль и не «норма».
 */
import type { MetricValue, MetricValueMeaning, MetricValueOrigin } from '@/shared/api/generated/model'
import { formatMinutes } from '@/shared/lib/duration'

export type { MetricValue, MetricValueMeaning, MetricValueOrigin }

/** Функции текстов vue-i18n, которые нужны форматированию. */
export interface MetricTexts {
  t: (key: string, params?: Record<string, unknown>) => string
  n: (value: number, format: string) => string
}

/** Число значения с учётом масштаба. */
export const numeric = (v: Pick<MetricValue, 'value' | 'scale'>): number => v.value / 10 ** (v.scale ?? 0)

/** Единицы длительности: у них обязательно видно происхождение (кейс §5.2). */
const DURATION_UNITS = new Set(['ms', 's', 'min', 'h', 'd'])

/** Длительность ли это. */
export const isDuration = (v: Pick<MetricValue, 'unit'>): boolean => DURATION_UNITS.has(v.unit)

/** Единица → ключ текста common.units (параметр `value`). */
const UNIT_TEXT: Record<string, string> = {
  s: 'common.units.seconds',
  h: 'common.units.hours',
  d: 'common.units.days',
  A: 'common.units.ampere',
  V: 'common.units.volt',
  mm: 'common.units.mm',
  um: 'common.units.um',
  C: 'common.units.celsius',
  rub: 'common.units.rubles',
  norm_h: 'common.units.normHours',
}

/**
 * Текст значения: «12», «93,3 %», «1 ч 15 мин», «160 А». Неизвестная единица —
 * число и код единицы как есть, а не угаданная единица (NFR-UI-4).
 * @param x — функции текстов
 * @param v — значение контракта
 */
export function formatValue(x: MetricTexts, v: MetricValue): string {
  const num = numeric(v)
  switch (v.unit) {
    case 'pcs':
      return x.n(num, 'integer')
    case 'bp':
      return x.n(num / 10000, 'percent')
    case 'min':
      return formatMinutes(x.t, num)
    case 'ms':
      return x.t('common.units.seconds', { value: x.n(num / 1000, 'decimal2') })
    default: {
      const key = UNIT_TEXT[v.unit]
      const shown = Number.isInteger(num) ? x.n(num, 'integer') : x.n(num, 'decimal2')
      return key ? x.t(key, { value: shown }) : `${shown} ${v.unit}`
    }
  }
}

/** Как показать происхождение значения времени. */
export interface OriginInfo {
  /** Код: происхождение контракта или `missing` — длительность без происхождения. */
  code: MetricValueOrigin | 'missing'
  /** Ключ текста. */
  key: string
  /** Требует внимания: происхождение не передано. */
  warn: boolean
}

const ORIGIN_TEXT: Record<MetricValueOrigin, string> = {
  reported_by_source: 'analytics.durationLabels.reportedBySource',
  computed_by_system: 'analytics.durationLabels.computedBySystem',
  mixed: 'widgets.analytics.origin.mixed',
}

/**
 * Происхождение значения (кейс §5.2: «передано источником либо определено
 * системой»). Для длительности без происхождения — явная пометка «не передано»,
 * а не молчание; у прочих величин без происхождения — null (пометка не нужна).
 * @param v — значение контракта
 */
export function originOf(v: Pick<MetricValue, 'origin' | 'unit'>): OriginInfo | null {
  if (v.origin) {
    const key = ORIGIN_TEXT[v.origin]
    return key ? { code: v.origin, key, warn: false } : { code: 'missing', key: 'widgets.analytics.origin.unknownCode', warn: true }
  }
  return isDuration(v) ? { code: 'missing', key: 'widgets.analytics.origin.missing', warn: true } : null
}

/**
 * Изменение к прошлому такому же периоду — в единицах значения; null — сравнивать
 * нельзя (нет прошлого значения или единицы не совпадают).
 */
export function deltaOf(now: MetricValue, previous: MetricValue | undefined): number | null {
  if (!previous || previous.unit !== now.unit) return null
  return numeric(now) - numeric(previous)
}

const MEANING_TEXT: Record<MetricValueMeaning, string> = {
  active_processing: 'analytics.durationLabels.activeWork',
  time_at_station: 'analytics.durationLabels.timeAtStation',
  other: 'widgets.analytics.meaning.other',
}

/** Смысл интервала длительности: ключ текста и пояснение сервера. */
export interface MeaningInfo {
  code: MetricValueMeaning
  key: string
  /** Пояснение «от чего до чего» (`meaning_note`), если сервер его дал. */
  note: string | null
}

/**
 * Смысл интервала длительности (соглашение «Длительности», FR-88): активная
 * обработка / полное время на участке / иной интервал. null — не длительность
 * или сервер смысл не передал.
 * @param v — значение контракта
 */
export function meaningOf(v: Pick<MetricValue, 'meaning' | 'meaning_note' | 'unit'>): MeaningInfo | null {
  if (!isDuration(v) || !v.meaning) return null
  const key = MEANING_TEXT[v.meaning]
  return key ? { code: v.meaning, key, note: v.meaning_note ?? null } : null
}

/**
 * Оговорки значения словами — для подсказки, а не плашками рядом с числом:
 * смысл интервала длительности (пояснение сервера важнее общего названия) и
 * происхождение времени (для длительности без происхождения — явная оговорка).
 * Пусто — оговаривать нечего (штуки, доли).
 * @param t — функция текстов
 * @param v — значение контракта
 */
export function valueNotes(t: MetricTexts['t'], v: Pick<MetricValue, 'origin' | 'unit' | 'meaning' | 'meaning_note'>): string[] {
  const out: string[] = []
  const m = meaningOf(v)
  if (m?.note) out.push(t('widgets.analytics.tiles.interval', { note: m.note }))
  else if (m?.code === 'active_processing') out.push(t('widgets.analytics.tiles.intervalActive'))
  else if (m?.code === 'time_at_station') out.push(t('widgets.analytics.tiles.intervalStation'))
  const o = originOf(v)
  if (o) out.push(o.code === 'reported_by_source' || o.code === 'computed_by_system' ? t(`widgets.analytics.tiles.origin.${o.code}`) : t(o.key))
  return out
}
