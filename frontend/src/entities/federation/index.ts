/**
 * Федерация предприятий (эпик 41; FR-131, FR-132, AD-19): предприятия-партнёры
 * с корнями доверия и выписки паспорта — входящие (получатель сам проверил
 * подписи цепочкой к корням партнёра) и исходящие.
 *
 * Операции (contracts/openapi.yaml, сгенерированный клиент):
 * `federation.partner.list`, `federation.partner.register`,
 * `federation.extract.list`, `federation.extract.read`,
 * `federation.extract.receive`, `federation.extract.send`.
 * Ключи — `[partner, …]` (SSE `partner` перечитывает).
 */
import type { Ref } from 'vue'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import {
  federationExtractList, federationExtractRead, federationExtractReceive, federationExtractSend, federationPartnerList, federationPartnerRegister,
} from '@/shared/api/generated/client'
import type {
  Partner, PartnerList, PartnerSignature, PassportExtract, PassportExtractList, PassportExtractView, ReceiveExtract, RegisterPartner, SendExtract,
} from '@/shared/api/generated/model'
import { entityKeys } from '@/shared/api/keys'
import type { ApiError } from '@/shared/api/problem'
import type { StatusTone } from '@/shared/api/generated/statuses'

export type { Partner, PartnerList, PartnerSignature, PassportExtract, PassportExtractList, PassportExtractView }

/** Период перечитывания, мс. */
export const FEDERATION_POLL_MS = 15_000

const keys = entityKeys('partner')

/** Статус происхождения выписки. */
export type OriginStatus = PassportExtract['origin_status']

/** Тон статуса происхождения: подтверждено — успех, только сервером — внимание, не подтверждено — предупреждение. */
export const ORIGIN_TONE: Record<string, StatusTone> = {
  verified: 'success',
  server_confirmed_only: 'attention',
  unverified: 'danger',
  not_applicable: 'neutral',
}

/** Партнёры (`federation.partner.list`). */
export function usePartners() {
  return useQuery({
    queryKey: keys.list('partners'),
    queryFn: ({ signal }) => federationPartnerList(undefined, { signal }),
    retry: false,
    refetchInterval: FEDERATION_POLL_MS,
  })
}

/** Выписки (`federation.extract.list`); партнёр — отбор по коду. */
export function useExtracts(partnerCode?: Ref<string>) {
  return useQuery({
    queryKey: keys.list('extracts', partnerCode ?? ''),
    queryFn: ({ signal }) => federationExtractList(partnerCode?.value ? { partner_code: partnerCode.value } : undefined, { signal }),
    retry: false,
    refetchInterval: FEDERATION_POLL_MS,
  })
}

/** Выписка с подписями (`federation.extract.read`). */
export function useExtract(digest: Ref<string>, enabled: Ref<boolean>) {
  return useQuery({
    queryKey: keys.list('extract', digest),
    queryFn: ({ signal }) => federationExtractRead(digest.value, { signal }),
    enabled,
    retry: false,
  })
}

/** Принять выписку партнёра (`federation.extract.receive`): изменённая — 422 federation.extract_tampered. */
export function useReceiveExtract() {
  const qc = useQueryClient()
  return useMutation<Awaited<ReturnType<typeof federationExtractReceive>>, ApiError, ReceiveExtract>({
    mutationFn: (body) => federationExtractReceive(body),
    onSettled: () => void qc.invalidateQueries({ queryKey: keys.all }),
  })
}

/** Зарегистрировать партнёра (`federation.partner.register`). */
export function useRegisterPartner() {
  const qc = useQueryClient()
  return useMutation<Awaited<ReturnType<typeof federationPartnerRegister>>, ApiError, RegisterPartner>({
    mutationFn: (body) => federationPartnerRegister(body),
    onSettled: () => void qc.invalidateQueries({ queryKey: keys.all }),
  })
}

/** Отправить выписку партнёру (`federation.extract.send`). */
export function useSendExtract() {
  const qc = useQueryClient()
  return useMutation<Awaited<ReturnType<typeof federationExtractSend>>, ApiError, SendExtract>({
    mutationFn: (body) => federationExtractSend(body),
    onSettled: () => void qc.invalidateQueries({ queryKey: keys.all }),
  })
}

/** Отпечаток короче для строки списка: `streebog256:ab12…ef90`. */
export const shortDigest = (d: string): string => (d.length > 28 ? `${d.slice(0, 16)}…${d.slice(-6)}` : d)

/** Скачать подписанный пакет выписки файлом (проверка у получателя без доступа к нашему журналу). */
export function downloadEnvelope(view: PassportExtractView): void {
  if (!view.envelope) return
  const blob = new Blob([view.envelope], { type: 'application/json' })
  const a = document.createElement('a')
  a.href = URL.createObjectURL(blob)
  a.download = `extract-${view.extract.partner_code}-${view.extract.extract_digest.replace(/^streebog256:/, '').slice(0, 12)}.json`
  a.click()
  URL.revokeObjectURL(a.href)
}
