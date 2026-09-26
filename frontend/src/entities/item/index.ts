/**
 * Изделие (кейс «история изделия»): ключи кэша и поиск по номеру детали для шапки.
 * Паспорт — виджет item-passport (эпик 11).
 */
import { itemItemLookup } from '@/shared/api/generated/client'
import { entityKeys } from '@/shared/api/keys'
import type { MomentParams } from '@/shared/model/moment'

export type { ItemLookup } from '@/shared/api/generated/model'

export const itemKeys = entityKeys('item')

/** Опции поиска изделия по номеру детали или скану DataMatrix. */
export const itemLookupQueryOptions = (q: string, moment: MomentParams) => ({
  queryKey: itemKeys.list('lookup', q, moment),
  queryFn: ({ signal }: { signal: AbortSignal }) => itemItemLookup({ q, ...moment }, { signal }),
  retry: false,
})
