/**
 * Длительность показателя в минутах (FR-12, PRD §3a «длительность операции
 * против нормы»): норма — из схемы процесса, длительность — от сервера.
 */
import { numeric, type MetricValue } from '@/entities/metric'

/** Значение длительности в минутах; единица не времени — null. */
export function minutesOf(v: MetricValue | null | undefined): number | null {
  if (!v) return null
  const n = numeric(v)
  switch (v.unit) {
    case 'min':
      return n
    case 's':
      return n / 60
    case 'ms':
      return n / 60000
    case 'h':
      return n * 60
    case 'd':
      return n * 1440
    default:
      return null
  }
}
