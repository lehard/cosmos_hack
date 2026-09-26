/**
 * Интеграции — экран «Интеграции» стола администратора (эпик 48; FR-157,
 * AD-47, Д-71): внешние системы, их состояние «включена / выключена / стенд»,
 * живой канал, последний обмен, ошибки, очередь и карантин исходящих.
 *
 * Операции (contracts/openapi.yaml, сгенерированный клиент):
 * `ops.integration.list`, `ops.integration.set`, `ops.integration.check`;
 * карантин исходящих учётной системы — `erp.message.list` и ручная
 * переотправка `erp.posting.resend` (FR-96). Ключи — `[integrity, '@list',
 * 'integrations']` (SSE `integrity` перечитывает) и `[erp_message, …]`.
 */
import type { Ref } from 'vue'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { erpMessageList, erpPostingResend, opsIntegrationCheck, opsIntegrationList, opsIntegrationSet } from '@/shared/api/generated/client'
import type { ErpMessage, IntegrationEntry, IntegrationList, ResendPosting, SetIntegrationState } from '@/shared/api/generated/model'
import { entityKeys } from '@/shared/api/keys'
import type { ApiError } from '@/shared/api/problem'

export type { ErpMessage, IntegrationEntry, IntegrationList }

/** Состояние интеграции. */
export type IntegrationStateCode = IntegrationEntry['state']

/** Период перечитывания экрана, мс. */
export const INTEGRATIONS_POLL_MS = 10_000

const keys = entityKeys('integrity')
const erpKeys = entityKeys('erp_message')

/** Системы, у которых очередь исходящих — учётные сообщения erp (карантин и переотправка, FR-96). */
export const LEDGER_SYSTEMS: ReadonlySet<string> = new Set(['onec', 'galaktika'])

/** Экран «Интеграции» (`ops.integration.list`). */
export function useIntegrations() {
  return useQuery({
    queryKey: keys.list('integrations'),
    queryFn: ({ signal }) => opsIntegrationList({ signal }),
    retry: false,
    refetchInterval: INTEGRATIONS_POLL_MS,
  })
}

/** Включить, выключить, «стенд ↔ реальная» (`ops.integration.set`). */
export function useSetIntegration() {
  const qc = useQueryClient()
  return useMutation<Awaited<ReturnType<typeof opsIntegrationSet>>, ApiError, { system: string; body: SetIntegrationState }>({
    mutationFn: ({ system, body }) => opsIntegrationSet(system, body),
    onSettled: () => void qc.invalidateQueries({ queryKey: keys.all }),
  })
}

/** Проверить соединение (`ops.integration.check`). */
export function useCheckIntegration() {
  const qc = useQueryClient()
  return useMutation<Awaited<ReturnType<typeof opsIntegrationCheck>>, ApiError, { system: string; commandId: string; policySeq: number }>({
    mutationFn: ({ system, commandId, policySeq }) => opsIntegrationCheck(system, { command_id: commandId, basis_seq: 0, policy_seq: policySeq }),
    onSettled: () => void qc.invalidateQueries({ queryKey: keys.all }),
  })
}

/** Сообщение ждёт решения человека: карантин или ошибка данных (FR-96). */
export const isQuarantined = (m: ErpMessage): boolean => m.status === 'quarantined' || m.status === 'rejected'

/**
 * Учётные сообщения системы (`erp.message.list`); карантин — отбором
 * `isQuarantined` на клиенте (заготовки отдают один список на все отборы).
 */
export function useLedgerMessages(system: Ref<string>, enabled: Ref<boolean>) {
  return useQuery({
    queryKey: erpKeys.list('integration', system),
    queryFn: ({ signal }) => erpMessageList({ system: system.value as 'onec' | 'galaktika' }, { signal }),
    enabled,
    retry: false,
    refetchInterval: INTEGRATIONS_POLL_MS,
  })
}

/** Переотправить сообщение из карантина (`erp.posting.resend`, FR-96). */
export function useResendPosting() {
  const qc = useQueryClient()
  return useMutation<Awaited<ReturnType<typeof erpPostingResend>>, ApiError, { businessKey: string; body: ResendPosting }>({
    mutationFn: ({ businessKey, body }) => erpPostingResend(businessKey, body),
    onSettled: () => {
      void qc.invalidateQueries({ queryKey: erpKeys.all })
      void qc.invalidateQueries({ queryKey: keys.all })
    },
  })
}

/**
 * Какие переходы доступны (FR-157): «Включить» — в прежний режим (стенд, если
 * он разрешён, иначе реальная), «Выключить», «Стенд ↔ реальная система».
 */
export function integrationActions(e: IntegrationEntry): { enable: IntegrationStateCode | null; disable: boolean; toggle: IntegrationStateCode | null } {
  if (!e.installed) return { enable: null, disable: false, toggle: null }
  const prev = e.last_decision?.previous
  let enable: IntegrationStateCode | null = null
  if (e.state === 'disabled') {
    if (prev === 'stand' && e.stand_available) enable = 'stand'
    else if (prev === 'enabled' && e.real_available) enable = 'enabled'
    else if (e.stand_available) enable = 'stand'
    else if (e.real_available) enable = 'enabled'
  }
  let toggle: IntegrationStateCode | null = null
  if (e.state === 'stand' && e.real_available) toggle = 'enabled'
  if (e.state === 'enabled' && e.stand_available) toggle = 'stand'
  return { enable, disable: e.state !== 'disabled', toggle }
}
