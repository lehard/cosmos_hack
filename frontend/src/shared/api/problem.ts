/**
 * Ошибки API (RFC 9457 problem+json, коды — contracts/errors.yaml, FR-28).
 * Сгенерированный клиент бросает Error с полями `status` и `info` (тело ответа);
 * здесь — разбор такой ошибки и выбор текста интерфейса по коду.
 */
import { errorCatalog } from './generated/errors'
import type { Problem } from './generated/model'

/** Ошибка сгенерированного клиента. */
export type ApiError = Error & { status?: number; info?: unknown }

/** Тело problem+json из ошибки клиента, если оно есть. */
export function problemOf(err: unknown): Problem | null {
  const info = (err as ApiError | null)?.info
  if (info && typeof info === 'object' && typeof (info as Problem).code === 'string') return info as Problem
  return null
}

/** HTTP-статус ошибки клиента (нет ответа — undefined). */
export const statusOf = (err: unknown): number | undefined => (err as ApiError | null)?.status

/** Сети нет или сервер не ответил: fetch бросает TypeError без статуса. */
export const isNetworkError = (err: unknown): boolean => err instanceof TypeError && statusOf(err) === undefined

/** Что показать пользователю: ключ текста интерфейса и параметры. */
export interface ProblemMessage {
  key: string
  params: Record<string, string>
  /** Заголовок из ответа — если текста по ключу нет. */
  fallback?: string
  /** Пояснение сервера (problem.detail) — человекочитаемое, важнее общего заголовка. */
  detail?: string
}

/**
 * Ключ текста интерфейса для ошибки: `ui_key` из каталога кодов, иначе
 * `errors.generic`; нет связи — `errors.network`.
 */
export function problemMessage(err: unknown): ProblemMessage {
  const problem = problemOf(err)
  if (!problem) {
    return { key: isNetworkError(err) ? 'errors.network' : 'errors.loadFailed', params: {} }
  }
  const entry = (errorCatalog as Record<string, { uiKey: string | null } | undefined>)[problem.code]
  return { key: entry?.uiKey ?? 'errors.generic', params: problem.params ?? {}, fallback: problem.title, detail: problem.detail }
}
