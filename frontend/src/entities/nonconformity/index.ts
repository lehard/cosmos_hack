/**
 * Несоответствие (эпик 11; FR-51…FR-55, FR-146; кейс §2.3): карточка, очередь
 * «Ждут моего решения», разрешения на отклонение, команды решений. Ключи кэша —
 * по соглашению shared/api/keys.ts: `[nonconformity, id, …]` и
 * `[nonconformity, '@list', …]` — их инвалидирует SSE.
 *
 * Операций ещё нет в contracts/openapi.yaml (эпик 02): запросы и команды стоят
 * на заглушке `api.not_implemented` (shared/api/pending.ts), форма данных
 * (model/types.ts) — предложение для контракта. Когда операции появятся,
 * функции запроса меняются на вызовы сгенерированного клиента; ключи, команды
 * и виджеты остаются прежними.
 */
import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { entityKeys } from '@/shared/api/keys'
import { pendingOperation, type Envelope } from '@/shared/api/pending'
import type { ApiError } from '@/shared/api/problem'
import { useMomentStore } from '@/shared/model/moment'
import type { DecisionCommand } from './model/actions'
import type { ConcessionOption, NcCard, QueueEntry } from './model/types'

export * from './model/types'
export * from './model/texts'
export * from './model/card'
export * from './model/actions'
export * from './model/focus'

export const nonconformityKeys = entityKeys('nonconformity')

/** Ожидаемые операции чтения модуля nonconformity (AD-40). */
export const NC_READ_OPERATIONS = {
  /** Карточка несоответствия (GET /api/v1/nonconformities/{nc_id}). */
  card: 'nonconformity.card.read',
  /** Очередь «Ждут моего решения» (GET /api/v1/decision-queue). */
  queue: 'nonconformity.queue.list',
  /** Разрешения на отклонение, применимые к изделию несоответствия. */
  concessions: 'nonconformity.concession.list',
} as const

/** Карточка несоответствия на момент (FR-51, AD-22). Пустой id — запрос не выполняется. */
export function useNcCard(ncId: MaybeRefOrGetter<string | null | undefined>) {
  const moment = useMomentStore()
  return useQuery({
    queryKey: computed(() => nonconformityKeys.one(toValue(ncId) ?? '', 'card', moment.params)),
    queryFn: () => pendingOperation<Envelope<NcCard>>(NC_READ_OPERATIONS.card)(),
    enabled: computed(() => !!toValue(ncId)),
    retry: false,
    placeholderData: (prev: Envelope<NcCard> | undefined) => prev,
  })
}

/** Очередь «Ждут моего решения» текущего пользователя (PRD §3a, FR-55, FR-57). */
export function useDecisionQueue(params: MaybeRefOrGetter<{ run_id?: string }> = {}) {
  const moment = useMomentStore()
  return useQuery({
    queryKey: computed(() => nonconformityKeys.list('decision-queue', toValue(params), moment.params)),
    queryFn: () => pendingOperation<Envelope<QueueEntry[]>>(NC_READ_OPERATIONS.queue)(),
    retry: false,
  })
}

/** Разрешения на отклонение для решения по несоответствию (FR-54). */
export function useConcessions(ncId: MaybeRefOrGetter<string | null | undefined>) {
  const moment = useMomentStore()
  return useQuery({
    queryKey: computed(() => nonconformityKeys.one(toValue(ncId) ?? '', 'concessions', moment.params)),
    queryFn: () => pendingOperation<Envelope<ConcessionOption[]>>(NC_READ_OPERATIONS.concessions)(),
    enabled: computed(() => !!toValue(ncId)),
    retry: false,
  })
}

/**
 * Подписанная команда решения: поля записи и подпись уровня 2 из порта
 * подписи; для бумажного пути — `paper: true`, документ печатается и
 * заверяется позже (FR-139, AD-43).
 */
export interface SignedDecisionCommand {
  command: DecisionCommand
  /** Подпись агента токена (base64 пакета) или null при подписи на бумаге. */
  signature: { key_ref: string; signature_b64: string } | null
  paper: boolean
}

/** Ответ на команду решения: запись журнала и документ для бумажного пути. */
export interface DecisionAccepted {
  event_id: string
  /** Номер критического действия `CA-‹n›` (AD-28). */
  ca_id?: string | null
  /** Документ решения — для печати с QR при подписи на бумаге. */
  document?: { document_id: string; version: number; doc_digest: string } | null
}

/**
 * Команда решения (FR-52, FR-53). Операция — из `command.operation`; пока
 * операций нет, ответ — `501 api.not_implemented`. После успеха
 * перечитываются карточка, очередь и паспорт изделия.
 */
export function useDecisionCommand() {
  const queryClient = useQueryClient()
  return useMutation<Envelope<DecisionAccepted>, ApiError, SignedDecisionCommand>({
    mutationFn: (signed) => pendingOperation<Envelope<DecisionAccepted>>(signed.command.operation)(),
    onSuccess: (_res, signed) => {
      void queryClient.invalidateQueries({ queryKey: nonconformityKeys.one(signed.command.nc_id) })
      void queryClient.invalidateQueries({ queryKey: nonconformityKeys.list() })
      void queryClient.invalidateQueries({ queryKey: ['item', signed.command.item_id] })
    },
  })
}
