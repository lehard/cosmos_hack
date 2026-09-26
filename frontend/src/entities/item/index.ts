/**
 * Изделие (кейс «история изделия»): ключи кэша, поиск по номеру детали для шапки
 * и паспорт изделия (эпик 11: FR-42, FR-43, FR-45, FR-46, FR-140).
 *
 * Операции чтения паспорта ещё нет в contracts/openapi.yaml (эпик 02): запрос
 * стоит на заглушке `api.not_implemented` (shared/api/pending.ts) с настоящим
 * ключом кэша `[item, id, 'passport', момент]` — его инвалидирует SSE по
 * изделию. Форма данных (model/types.ts) — предложение для контракта; когда
 * операция появится, функция запроса меняется на вызов клиента, ключи и
 * виджеты остаются прежними.
 */
import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { useQuery } from '@tanstack/vue-query'
import { itemItemLookup } from '@/shared/api/generated/client'
import { entityKeys } from '@/shared/api/keys'
import { pendingOperation, type Envelope } from '@/shared/api/pending'
import { useMomentStore, type MomentParams } from '@/shared/model/moment'
import type { ItemPassport } from './model/types'

export type { ItemLookup } from '@/shared/api/generated/model'
export type { Envelope }

export * from './model/types'
export * from './model/passport'

export const itemKeys = entityKeys('item')

/** Опции поиска изделия по номеру детали или скану DataMatrix. */
export const itemLookupQueryOptions = (q: string, moment: MomentParams) => ({
  queryKey: itemKeys.list('lookup', q, moment),
  queryFn: ({ signal }: { signal: AbortSignal }) => itemItemLookup({ q, ...moment }, { signal }),
  retry: false,
})

/** id ожидаемой операции чтения паспорта (AD-40). */
export const PASSPORT_OPERATION = 'item.passport.read'

/**
 * Паспорт изделия на момент из useMomentStore (FR-42, AD-22). Ожидаемая
 * операция — `item.passport.read` (GET /api/v1/items/{item_id}/passport).
 * Пустой id — запрос не выполняется.
 */
export function usePassport(itemId: MaybeRefOrGetter<string | null | undefined>) {
  const moment = useMomentStore()
  return useQuery({
    queryKey: computed(() => itemKeys.one(toValue(itemId) ?? '', 'passport', moment.params)),
    queryFn: () => pendingOperation<Envelope<ItemPassport>>(PASSPORT_OPERATION)(),
    enabled: computed(() => !!toValue(itemId)),
    retry: false,
    // Смена момента не мигает пустым паспортом — прежний виден до нового ответа.
    placeholderData: (prev: Envelope<ItemPassport> | undefined) => prev,
  })
}

export { default as SourceMark } from './ui/SourceMark.vue'
export { default as SignatureMark } from './ui/SignatureMark.vue'
export { default as SummaryTag } from './ui/SummaryTag.vue'
