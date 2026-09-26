/**
 * Кэш запросов Vue Query — единственное место серверного состояния (AD-21).
 * Один экземпляр на приложение: им пользуются охранник маршрутов и канал SSE.
 */
import { MutationCache, QueryCache, QueryClient } from '@tanstack/vue-query'
import { statusOf } from '@/shared/api'
import { noteServerNow } from '@/shared/model/server-clock'

/** Ответ API в конверте с заголовками — берём из него «сейчас» сервера (`Ant-Now`). */
const noteHeaders = (data: unknown): void => {
  const headers = (data as { headers?: unknown } | null | undefined)?.headers
  if (headers instanceof Headers) noteServerNow(headers)
}

export const queryClient = new QueryClient({
  queryCache: new QueryCache({ onSuccess: noteHeaders }),
  mutationCache: new MutationCache({ onSuccess: noteHeaders }),
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      refetchOnWindowFocus: false,
      // Ответ 4xx/501 — это ответ, а не сбой связи: повтор ничего не изменит.
      retry: (count, err) => {
        const status = statusOf(err)
        return count < 1 && (status === undefined || status >= 502)
      },
    },
  },
})
