/**
 * Кэш запросов Vue Query — единственное место серверного состояния (AD-21).
 * Один экземпляр на приложение: им пользуются охранник маршрутов и канал SSE.
 */
import { QueryClient } from '@tanstack/vue-query'
import { statusOf } from '@/shared/api'

export const queryClient = new QueryClient({
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
