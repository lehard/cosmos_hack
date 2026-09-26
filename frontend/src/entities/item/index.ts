/**
 * Изделие (кейс «история изделия»): ключи кэша, поиск по номеру детали для шапки
 * и паспорт изделия (эпик 11: FR-42, FR-43, FR-45, FR-46, FR-140).
 *
 * Чтение — сгенерированным клиентом (`item.passport.read`, `item.history.list`,
 * `item.genealogy.read`) с ключами по соглашению shared/api/keys.ts:
 * `[item, id, …, момент]` — их инвалидирует SSE по изделию.
 */
import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { useQuery } from '@tanstack/vue-query'
import { itemGenealogyRead, itemHistoryList, itemItemLookup, itemPassportRead } from '@/shared/api/generated/client'
import { entityKeys } from '@/shared/api/keys'
import { useMomentStore, type MomentParams } from '@/shared/model/moment'

export type { ItemLookup } from '@/shared/api/generated/model'

export * from './model/types'
export * from './model/passport'
export { default as SourceMark } from './ui/SourceMark.vue'
export { default as SignatureMark } from './ui/SignatureMark.vue'
export { default as SummaryTag } from './ui/SummaryTag.vue'

export const itemKeys = entityKeys('item')

/** Опции поиска изделия по номеру детали или скану DataMatrix. */
export const itemLookupQueryOptions = (q: string, moment: MomentParams) => ({
  queryKey: itemKeys.list('lookup', q, moment),
  queryFn: ({ signal }: { signal: AbortSignal }) => itemItemLookup({ q, ...moment }, { signal }),
  retry: false,
})

/** Необязательные параметры чтения, кроме момента. */
export interface ItemReadOptions {
  /** Прогон сценария (AD-38). */
  run_id?: string
}

/** Параметры запроса: момент из useMomentStore и прогон. */
function useReadParams(opts: MaybeRefOrGetter<ItemReadOptions>) {
  const moment = useMomentStore()
  return computed(() => {
    const run = toValue(opts).run_id
    return run ? { ...moment.params, run_id: run } : { ...moment.params }
  })
}

/**
 * Паспорт изделия на момент (FR-42, AD-22) — `item.passport.read`. Пустой id —
 * запрос не выполняется.
 */
export function usePassport(itemId: MaybeRefOrGetter<string | null | undefined>, opts: MaybeRefOrGetter<ItemReadOptions> = {}) {
  const params = useReadParams(opts)
  return useQuery({
    queryKey: computed(() => itemKeys.one(toValue(itemId) ?? '', 'passport', params.value)),
    queryFn: ({ signal }) => itemPassportRead(toValue(itemId) ?? '', params.value, { signal }),
    enabled: computed(() => !!toValue(itemId)),
    retry: false,
    // Смена момента не мигает пустым паспортом; другое изделие — никогда не
    // показывается паспортом прежнего (NFR-UI-4).
    placeholderData: (prev, prevQuery) => (prevQuery?.queryKey[1] === toValue(itemId) ? prev : undefined),
  })
}

/**
 * Журнал изменений паспорта (FR-43) — `item.history.list`: что изменилось,
 * было → стало, кто, основание, ссылка на запись журнала.
 */
export function useItemHistory(itemId: MaybeRefOrGetter<string | null | undefined>, opts: MaybeRefOrGetter<ItemReadOptions & { enabled?: boolean }> = {}) {
  const params = useReadParams(opts)
  return useQuery({
    queryKey: computed(() => itemKeys.one(toValue(itemId) ?? '', 'history', params.value)),
    queryFn: ({ signal }) => itemHistoryList(toValue(itemId) ?? '', params.value, { signal }),
    enabled: computed(() => !!toValue(itemId) && toValue(opts).enabled !== false),
    retry: false,
  })
}

/** Генеалогия (FR-45) — `item.genealogy.read`: куда входит, из чего состоит, партии. */
export function useItemGenealogy(itemId: MaybeRefOrGetter<string | null | undefined>, opts: MaybeRefOrGetter<ItemReadOptions & { enabled?: boolean }> = {}) {
  const params = useReadParams(opts)
  return useQuery({
    queryKey: computed(() => itemKeys.one(toValue(itemId) ?? '', 'genealogy', params.value)),
    queryFn: ({ signal }) => itemGenealogyRead(toValue(itemId) ?? '', params.value, { signal }),
    enabled: computed(() => !!toValue(itemId) && toValue(opts).enabled !== false),
    retry: false,
  })
}
