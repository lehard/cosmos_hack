/**
 * Состояние компонентов (эпик 14; FR-127, AD-45, AD-46): сервисы и копии,
 * очереди и отставание курсоров, интеграции, изделия «обработка остановлена»,
 * последний отчёт верификатора.
 *
 * Операции — `ops.health.read`, `ops.stopped_item.list`, `ops.processing.retry`
 * (contracts/openapi.yaml), сгенерированный клиент. Здоровье не сущность SSE —
 * перечитывается по таймеру; ключ `[integrity, '@list', 'health']` дополнительно
 * сбрасывает SSE `integrity`. Остановленные изделия — `[item, '@list', 'stopped']`.
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { opsHealthRead, opsProcessingRetry, opsStoppedItemList } from '@/shared/api/generated/client'
import type { ComponentState, IntegrationState, OpsHealth, QueueState, RetryProcessing, StoppedItem } from '@/shared/api/generated/model'
import { entityKeys } from '@/shared/api/keys'
import type { ApiError } from '@/shared/api/problem'

export type { ComponentState, IntegrationState, OpsHealth, QueueState, RetryProcessing, StoppedItem }

/** Период перечитывания состояния компонентов, мс. */
export const HEALTH_POLL_MS = 10_000

const keys = entityKeys('integrity')
const itemKeys = entityKeys('item')

/** Состояние компонентов (`ops.health.read`). */
export function useOpsHealth() {
  return useQuery({
    queryKey: keys.list('health'),
    queryFn: ({ signal }) => opsHealthRead({ signal }),
    retry: false,
    refetchInterval: HEALTH_POLL_MS,
  })
}

/** Изделия с остановленной обработкой (`ops.stopped_item.list`, AD-45). */
export function useStoppedItems() {
  return useQuery({
    queryKey: itemKeys.list('stopped'),
    queryFn: ({ signal }) => opsStoppedItemList({ limit: 200 }, { signal }),
    retry: false,
    refetchInterval: HEALTH_POLL_MS,
  })
}

/** Повторить обработку изделия (`ops.processing.retry`); после — перечитать список и здоровье. */
export function useRetryProcessing() {
  const queryClient = useQueryClient()
  return useMutation<Awaited<ReturnType<typeof opsProcessingRetry>>, ApiError, { itemId: string; body: RetryProcessing }>({
    mutationFn: ({ itemId, body }) => opsProcessingRetry(itemId, body),
    onSuccess: (_r, { itemId }) => {
      void queryClient.invalidateQueries({ queryKey: itemKeys.list('stopped') })
      void queryClient.invalidateQueries({ queryKey: itemKeys.one(itemId) })
      void queryClient.invalidateQueries({ queryKey: keys.list('health') })
    },
  })
}
