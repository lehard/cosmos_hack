/**
 * Целостность журнала «по данным сервера» (AD-46): последний отчёт верификатора.
 * Ключ ['integrity', 'global'] совпадает с сообщением SSE (entity integrity, id global).
 */
import { useSecurityIntegrityRead } from '@/shared/api/generated/client'
import type { IntegrityStatus } from '@/shared/api/generated/model'
import { entityKeys } from '@/shared/api/keys'

export type { IntegrityStatus }

export const integrityKeys = entityKeys('integrity')

/** Состояние целостности для шапки. */
export const useIntegrity = () =>
  useSecurityIntegrityRead({ query: { queryKey: integrityKeys.one('global'), refetchInterval: 60_000 } })

/**
 * Индикатор желтеет сам, если свежего отчёта нет дольше двух интервалов (AD-46) —
 * даже если сервер (возможно, взломанный) продолжает отвечать «цело».
 * @param s — ответ сервера
 * @param now — текущее время, мс
 */
export function effectiveIntegrity(s: IntegrityStatus | undefined, now: number): IntegrityStatus['status'] {
  if (!s) return 'unknown'
  if (s.status === 'ok' && (!s.checked_at || now - Date.parse(s.checked_at) > 2 * s.interval_seconds * 1000)) return 'stale'
  return s.status
}
