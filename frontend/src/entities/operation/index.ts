/**
 * Выполнение операций и перемещения (эпик 17; FR-16, FR-44, FR-47, FR-55, FR-137):
 * команды терминала исполнителя и мастера — начать и остановить операцию,
 * подтвердить приёмку, в том числе физическое перемещение в изолятор; изделия
 * на шаге процесса.
 *
 * Операции — `process.operation.start|pause|resume|finish`,
 * `process.movement.receive`, `item.item.list` (contracts/openapi.yaml),
 * сгенерированный клиент. Гарды (предусловия FR-17, лимит доработок FR-18,
 * точка предъявления FR-19) проверяет сервер; отказ приходит кодом ошибки.
 *
 * Команды меняют изделие, живую карту, несоответствие (FR-55: приёмка в
 * изоляторе снимает расхождение «изолировано в системе, физически не
 * перемещено»), сроки и задачи — после успеха перечитывается всё это; SSE
 * сделает то же на других столах.
 */
import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { keepPreviousData, useMutation, useQuery, useQueryClient, type QueryClient } from '@tanstack/vue-query'
import { itemItemList, itemPassportRead, processMovementReceive, processOperationFinish, processOperationPause, processOperationStart } from '@/shared/api/generated/client'
import type {
  FinishOperation,
  FinishOperationCompletion,
  ItemRow,
  PauseOperation,
  ReceiveMovement,
  ReceiveMovementInspectionOnReceipt,
  StartOperation,
} from '@/shared/api/generated/model'
import { entityKeys } from '@/shared/api/keys'
import type { ApiError } from '@/shared/api/problem'
import type { Envelope } from '@/shared/api/response'
import { useMomentStore } from '@/shared/model/moment'

export type { FinishOperationCompletion, ItemRow, ReceiveMovementInspectionOnReceipt }

const itemKeys = entityKeys('item')

/** Изделия на шаге процесса на момент — `item.item.list` (первые 100). */
export function useItemsAtStep(stepKey: MaybeRefOrGetter<string | null | undefined>) {
  const moment = useMomentStore()
  const params = computed(() => ({ step_key: toValue(stepKey) ?? '', limit: 100, ...moment.params }))
  return useQuery({
    queryKey: computed(() => itemKeys.list('at-step', params.value)),
    queryFn: async ({ signal }): Promise<Envelope<ItemRow[]>> => {
      const res = await itemItemList(params.value, { signal })
      return { data: res.data.items, headers: res.headers }
    },
    enabled: computed(() => !!toValue(stepKey)),
    placeholderData: keepPreviousData,
    retry: false,
  })
}

/** Команда исполнения. */
export type OperationCommand =
  | { kind: 'start'; item_id: string; body: StartOperation }
  | { kind: 'pause'; item_id: string; run_id: string; body: PauseOperation }
  | { kind: 'finish'; item_id: string; run_id: string; body: FinishOperation }
  | { kind: 'receive'; item_id: string; body: ReceiveMovement }

function send(c: OperationCommand) {
  switch (c.kind) {
    case 'start':
      return processOperationStart(c.item_id, c.body)
    case 'pause':
      return processOperationPause(c.run_id, c.body)
    case 'finish':
      return processOperationFinish(c.run_id, c.body)
    case 'receive':
      return processMovementReceive(c.item_id, c.body)
  }
}

/** Что перечитать после команды над изделием. */
export function invalidateAfterItemCommand(queryClient: QueryClient, itemId: string): void {
  for (const key of [['item', itemId], ['item', '@list'], ['live_map'], ['nonconformity'], ['notification'], ['task'], ['equipment'], ['workplace']]) {
    void queryClient.invalidateQueries({ queryKey: key })
  }
}

/** Команды исполнения операции и перемещения (квитанция с seq записи). */
export function useOperationCommand() {
  const queryClient = useQueryClient()
  return useMutation<Awaited<ReturnType<typeof send>>, ApiError, OperationCommand>({
    mutationFn: send,
    onSuccess: (_r, c) => invalidateAfterItemCommand(queryClient, c.item_id),
  })
}

/**
 * `basis_seq` команды над изделием (AD-39): seq паспорта изделия на момент —
 * из кэша паспорта (тот же ключ, что у usePassport) или свежим чтением.
 * Паспорт не прочитался — 0: сервер сверит по текущему состоянию и при
 * расхождении ответит 409.
 */
export function useItemBasis() {
  const queryClient = useQueryClient()
  const moment = useMomentStore()
  return async (itemId: string | null | undefined): Promise<number> => {
    if (!itemId) return 0
    const params = { ...moment.params }
    try {
      const res = await queryClient.fetchQuery({
        queryKey: itemKeys.one(itemId, 'passport', params),
        queryFn: ({ signal }) => itemPassportRead(itemId, params, { signal }),
        staleTime: 5_000,
      })
      return res.data.basis_seq
    } catch {
      return 0
    }
  }
}
