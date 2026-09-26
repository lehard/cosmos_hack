/**
 * Ключи кэша Vue Query (AD-21): серверное состояние живёт только в кэше, живые
 * обновления SSE инвалидируют ключ `[сущность, id]`.
 *
 * Соглашение для всех слоёв:
 * - запрос одной сущности — `[сущность, id, ...уточнения]`;
 * - запрос списка или сводки по виду сущности — `[сущность, LIST, ...уточнения]`;
 * - параметры момента (`axis`, `as_of`) — последним элементом ключа.
 *
 * Сообщение SSE `(сущность, id)` инвалидирует `[сущность, id]` и
 * `[сущность, LIST]` — по префиксу, со всеми уточнениями. Поэтому читать
 * сервер нужно через обёртки `entities/*`, которые ставят такие ключи
 * (у сгенерированных хуков ключ по URL).
 */
import type { EntityKind } from './generated/model'

/** Маркер списка или сводки во втором элементе ключа. */
export const LIST = '@list' as const

/** Маркер запросов прав — их инвалидирует и изменение политики. */
export const PERMISSIONS = '@permissions' as const

/** Фабрика ключей одного вида сущности. */
export interface EntityKeys<K extends EntityKind> {
  /** Все запросы этого вида. */
  all: readonly [K]
  /** Одна сущность. */
  one: (id: string, ...rest: unknown[]) => readonly [K, string, ...unknown[]]
  /** Списки и сводки. */
  list: (...rest: unknown[]) => readonly [K, typeof LIST, ...unknown[]]
}

/**
 * Ключи Vue Query для вида сущности.
 * @param kind — вид сущности из контракта SSE (EntityKind)
 */
export function entityKeys<K extends EntityKind>(kind: K): EntityKeys<K> {
  return {
    all: [kind] as const,
    one: (id, ...rest) => [kind, id, ...rest] as const,
    list: (...rest) => [kind, LIST, ...rest] as const,
  }
}
