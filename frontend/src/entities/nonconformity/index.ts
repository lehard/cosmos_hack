/**
 * Несоответствие (эпик 11; FR-51…FR-55, FR-146; кейс §2.3): карточка, очередь
 * «Ждут моего решения», разрешения на отклонение, команды решений.
 *
 * Чтение и команды — сгенерированным клиентом (`nonconformity.card.read`,
 * `nonconformity.queue.list`, `nonconformity.concession.list`,
 * `nonconformity.nonconformity.confirm`, `nonconformity.signal.reject`,
 * `nonconformity.recheck.request`, `nonconformity.item.isolate`,
 * `nonconformity.disposition.set`) с ключами по соглашению shared/api/keys.ts:
 * `[nonconformity, id, …]` и `[nonconformity, '@list', …]` — их инвалидирует SSE.
 */
import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import {
  nonconformityCardRead,
  nonconformityConcessionList,
  nonconformityDispositionSet,
  nonconformityItemIsolate,
  nonconformityNonconformityConfirm,
  nonconformityQueueList,
  nonconformityRecheckRequest,
  nonconformitySignalReject,
} from '@/shared/api/generated/client'
import { entityKeys } from '@/shared/api/keys'
import type { ApiError } from '@/shared/api/problem'
import { useMomentStore } from '@/shared/model/moment'
import type { DecisionRequest } from './model/actions'
import type { QueueSort } from './model/card'

export * from './model/types'
export * from './model/texts'
export * from './model/card'
export * from './model/actions'
export * from './model/command-id'

export const nonconformityKeys = entityKeys('nonconformity')

/** Параметры момента и прогона для чтения. */
function useReadParams(runId: MaybeRefOrGetter<string | undefined>) {
  const moment = useMomentStore()
  return computed(() => {
    const run = toValue(runId)
    return run ? { ...moment.params, run_id: run } : { ...moment.params }
  })
}

/** Карточка несоответствия на момент (FR-51, AD-22). Пустой id — запрос не выполняется. */
export function useNcCard(ncId: MaybeRefOrGetter<string | null | undefined>, runId: MaybeRefOrGetter<string | undefined> = undefined) {
  const params = useReadParams(runId)
  return useQuery({
    queryKey: computed(() => nonconformityKeys.one(toValue(ncId) ?? '', 'card', params.value)),
    queryFn: ({ signal }) => nonconformityCardRead(toValue(ncId) ?? '', params.value, { signal }),
    enabled: computed(() => !!toValue(ncId)),
    retry: false,
    // Другое несоответствие — никогда не показывается карточкой прежнего.
    placeholderData: (prev, prevQuery) => (prevQuery?.queryKey[1] === toValue(ncId) ? prev : undefined),
  })
}

/**
 * Очередь «Ждут моего решения» (PRD §3a, FR-55): порядок по риску или сроку
 * считает сервер (`sort`).
 */
export function useDecisionQueue(sort: MaybeRefOrGetter<QueueSort>, runId: MaybeRefOrGetter<string | undefined> = undefined) {
  const params = useReadParams(runId)
  const full = computed(() => ({ ...params.value, sort: toValue(sort) }))
  return useQuery({
    queryKey: computed(() => nonconformityKeys.list('decision-queue', full.value)),
    queryFn: ({ signal }) => nonconformityQueueList(full.value, { signal }),
    retry: false,
    placeholderData: (prev) => prev,
  })
}

/** Разрешения на отклонение по изделию (FR-54) — для выбора при ремонте и «как есть». */
export function useConcessions(itemId: MaybeRefOrGetter<string | null | undefined>, enabled: MaybeRefOrGetter<boolean> = true) {
  const params = useReadParams(undefined)
  const full = computed(() => ({ ...params.value, item_id: toValue(itemId) ?? '' }))
  return useQuery({
    queryKey: computed(() => nonconformityKeys.list('concessions', full.value)),
    queryFn: ({ signal }) => nonconformityConcessionList(full.value, { signal }),
    enabled: computed(() => !!toValue(itemId) && toValue(enabled)),
    retry: false,
  })
}

/** Выполнить команду решения сгенерированным клиентом. */
function send(req: DecisionRequest) {
  switch (req.action) {
    case 'confirm_nc':
      return nonconformityNonconformityConfirm(req.nc_id, req.body)
    case 'reject_signal':
      return nonconformitySignalReject(req.item_id, req.body)
    case 'request_recheck':
      return nonconformityRecheckRequest(req.item_id, req.body)
    case 'isolate':
      return nonconformityItemIsolate(req.item_id, req.body)
    case 'disposition':
      return nonconformityDispositionSet(req.nc_id, req.body)
  }
}

/**
 * Команда решения (FR-52, FR-53). Ответ — квитанция с номером критического
 * действия (`ca_ref`, AD-28). После успеха перечитываются карточка, очередь,
 * разрешения и паспорт изделия; SSE сделает то же для других столов.
 */
export function useDecisionCommand() {
  const queryClient = useQueryClient()
  return useMutation<Awaited<ReturnType<typeof send>>, ApiError, DecisionRequest>({
    mutationFn: send,
    onSuccess: (_res, req) => {
      void queryClient.invalidateQueries({ queryKey: nonconformityKeys.one(req.nc_id) })
      void queryClient.invalidateQueries({ queryKey: nonconformityKeys.list() })
      void queryClient.invalidateQueries({ queryKey: ['item', req.item_id] })
    },
  })
}
