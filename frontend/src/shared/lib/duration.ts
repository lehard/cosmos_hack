/**
 * Длительность в минутах → текст интерфейса: «37 мин», «2 ч», «1 ч 15 мин»
 * (тексты common.units, NFR-UI-3).
 */
type Translate = (key: string, params: Record<string, unknown>) => string

/**
 * @param t — функция текстов vue-i18n
 * @param minutes — целое число минут
 */
export function formatMinutes(t: Translate, minutes: number): string {
  const m = Math.max(0, Math.round(minutes))
  if (m < 60) return t('common.units.minutes', { value: m })
  const h = Math.floor(m / 60)
  const rest = m % 60
  return rest ? t('common.units.hoursMinutes', { h, m: rest }) : t('common.units.hours', { value: h })
}
