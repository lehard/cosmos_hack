/**
 * Допуск к рабочему месту (барьер 2, FR-83; эпик 37) — чистые функции фичи:
 * мои посты в смене из панели «Посты» и тело команды `access.workplace.admit`
 * по состоянию ключа из порта подписи (ключ вставлен и открыт PIN-ом).
 */
import type { AdmitWorkplace, PostRow } from '@/shared/api/generated/model'
import type { TokenInfo, TokenStatus } from '@/shared/lib/token-agent'

/** Посты, на которые назначен сотрудник (панель «Посты»). */
export function myPosts(posts: readonly PostRow[] | null | undefined, personId: string | null | undefined): PostRow[] {
  if (!posts || !personId) return []
  return posts.filter((p) => p.assigned?.person_id === personId)
}

/** Ключ сотрудника среди ключей расширения: `‹псевдоним›@‹версия›` без учёта регистра. */
export function keyRefOf(info: TokenInfo | null | undefined, personId: string): string | null {
  const refs = info?.key_refs ?? []
  const mine = refs.find((r) => r.split('@')[0]?.toLowerCase() === personId.toLowerCase())
  return mine ?? null
}

/**
 * Поля ключа команды допуска: ключ вставлен (`inserted`) — он открыт PIN-ом;
 * `locked` — вставлен, PIN не введён. Ключа сотрудника нет — null (кнопка
 * допуска подскажет вставить ключ). Проверку выполняет сервер.
 */
export function admissionKey(
  status: TokenStatus,
  info: TokenInfo | null | undefined,
  personId: string,
): Pick<AdmitWorkplace, 'key_ref' | 'pin_verified'> | null {
  if (status !== 'inserted' && status !== 'locked') return null
  const ref = keyRefOf(info, personId) ?? `${personId.toLowerCase()}@1`
  return { key_ref: ref, pin_verified: status === 'inserted' }
}
