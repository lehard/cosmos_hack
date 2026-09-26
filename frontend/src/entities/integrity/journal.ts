/**
 * Общий журнал (PRD §3a; AD-2, AD-37, AD-44; FR-68, FR-140; кейс §7.2):
 * записи основной цепочки с видом записи (факт, вывод системы, решение
 * человека, служебная), пометкой источника и статусом подписи; голова журнала
 * (последний номер записи и критического действия).
 *
 * Операции — `journal.entry.list`, `journal.entry.read`, `journal.head.read`
 * (contracts/openapi.yaml), сгенерированный клиент. Журнал не вид сущности SSE:
 * ключи — `[integrity, '@list', 'journal', …]`, их сбрасывает сообщение SSE
 * `integrity` (новый отчёт верификатора), а «сейчас» — перечитывание по таймеру.
 * Читаются на момент (AD-37): в воспроизведении — «что было на момент».
 */
import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { keepPreviousData, useInfiniteQuery, useQuery } from '@tanstack/vue-query'
import { journalEntryList, journalEntryRead, journalHeadRead } from '@/shared/api/generated/client'
import type {
  JournalEntryListEntryKind,
  JournalEntryListParams,
  JournalEntryView,
  JournalEntryViewSignatureStatus,
  JournalHead,
} from '@/shared/api/generated/model'
import { entityKeys } from '@/shared/api/keys'
import { backendModeOf } from '@/shared/api/response'
import { useMomentStore } from '@/shared/model/moment'
import { AUDIT_POLL_MS } from './audit'

export type { JournalEntryView, JournalHead }

/** Вид записи журнала (AD-2). */
export type JournalEntryKind = JournalEntryListEntryKind
/** Статус проверки подписи записи (FR-68). */
export type JournalSignatureStatus = JournalEntryViewSignatureStatus

/** Виды записей в порядке фильтра: исходные события, анализ системы, решения людей, служебные. */
export const JOURNAL_ENTRY_KINDS: readonly JournalEntryKind[] = ['fact', 'reaction', 'decision', 'service']

/** Вид записи из среза стола или адреса; чужое значение — null (все записи). */
export const asEntryKind = (v: unknown): JournalEntryKind | null =>
  typeof v === 'string' && (JOURNAL_ENTRY_KINDS as readonly string[]).includes(v) ? (v as JournalEntryKind) : null

/** Фильтр журнала: изделие, тип (или префикс семейства «item.»), вид записи. */
export type JournalFilter = Pick<JournalEntryListParams, 'item_id' | 'event_type' | 'entry_kind' | 'stream'>

/** Размер страницы журнала. */
export const JOURNAL_PAGE = 50

const keys = entityKeys('integrity')

/** Голова журнала на момент (`journal.head.read`): номер последней записи и CA-‹n›. */
export function useJournalHead() {
  const moment = useMomentStore()
  const params = computed(() => ({ ...moment.params }))
  return useQuery({
    queryKey: computed(() => keys.list('journal-head', params.value)),
    queryFn: ({ signal }) => journalHeadRead(params.value, { signal }),
    retry: false,
    placeholderData: keepPreviousData,
    refetchInterval: () => (moment.isReplay ? false : AUDIT_POLL_MS),
  })
}

/**
 * Записи журнала на момент (`journal.entry.list`) страницами по курсору:
 * «Показать ещё» дочитывает следующую страницу; страницы склеены в один список.
 */
export function useJournalEntries(filter: MaybeRefOrGetter<JournalFilter> = {}) {
  const moment = useMomentStore()
  const params = computed<JournalEntryListParams>(() => {
    const f = toValue(filter)
    const clean = Object.fromEntries(Object.entries(f).filter(([, v]) => v !== undefined && v !== null && v !== '')) as JournalFilter
    return { limit: JOURNAL_PAGE, ...clean, ...moment.params }
  })
  const q = useInfiniteQuery({
    queryKey: computed(() => keys.list('journal', params.value)),
    queryFn: ({ signal, pageParam }) => journalEntryList({ ...params.value, ...(pageParam ? { cursor: pageParam } : {}) }, { signal }),
    initialPageParam: '' as string,
    getNextPageParam: (last) => last.data.next_cursor || undefined,
    retry: false,
    placeholderData: keepPreviousData,
    refetchInterval: () => (moment.isReplay ? false : AUDIT_POLL_MS),
  })
  const pages = computed(() => q.data.value?.pages ?? [])
  return {
    /** Все загруженные записи; null — первая страница ещё не пришла. */
    items: computed<JournalEntryView[] | null>(() => (pages.value.length ? pages.value.flatMap((p) => p.data.items) : null)),
    isPending: computed(() => q.isPending.value),
    error: computed(() => q.error.value ?? undefined),
    mode: computed(() => backendModeOf(pages.value[0])),
    hasMore: computed(() => q.hasNextPage.value),
    loadingMore: computed(() => q.isFetchingNextPage.value),
    loadMore: () => void q.fetchNextPage(),
  }
}

/** Запись журнала по номеру (`journal.entry.read`) — когда её нет среди загруженных. */
export function useJournalEntry(seq: MaybeRefOrGetter<number | null>) {
  return useQuery({
    queryKey: computed(() => keys.list('journal-entry', toValue(seq) ?? 0)),
    queryFn: ({ signal }) => journalEntryRead(toValue(seq) ?? 0, { signal }),
    enabled: computed(() => toValue(seq) != null),
    retry: false,
  })
}
