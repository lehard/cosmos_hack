/**
 * Документы, маршруты подписей и запросы решения (эпик 11: FR-65, FR-66,
 * FR-136, FR-139; AD-12, AD-13, AD-43). Ключи кэша — по соглашению
 * shared/api/keys.ts.
 *
 * Операций ещё нет в contracts/openapi.yaml (эпик 02): запросы стоят на
 * заглушке `api.not_implemented` (shared/api/pending.ts) с настоящими ключами
 * кэша; форма данных (model/types.ts) — предложение для контракта.
 */
import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { entityKeys } from '@/shared/api/keys'
import { pendingOperation, type Envelope } from '@/shared/api/pending'
import type { ApiError } from '@/shared/api/problem'
import { useMomentStore } from '@/shared/model/moment'
import type { DecisionRequest } from './model/types'

export * from './model/types'
export * from './model/route'

export const documentKeys = entityKeys('document')

/** Ожидаемые операции модуля documents (AD-40). */
export const DOCUMENT_OPERATIONS = {
  /** Запросы решения, ждущие подписи текущего пользователя (FR-136). */
  requests: 'documents.request.list',
  /** Запросить решение — оформить документ с маршрутом (FR-146, `document.version.requested`). */
  request: 'documents.version.request',
  /** Записать подпись агентом токена (`document.signature.recorded`, method = token_agent). */
  sign: 'documents.signature.record',
  /** Не согласовать — вернуть с замечанием. */
  decline: 'documents.signature.decline',
  /** Бумажный экземпляр с QR для печати (`document.paper.status_changed` printed). */
  print: 'documents.paper.print',
  /** Загрузить скан и заверить бумажную подпись (FR-139, AD-43). */
  attestPaper: 'documents.paper.attest',
} as const

/**
 * Карточки «требуется ваше решение» текущего пользователя (FR-136). Ожидаемая
 * операция — `documents.request.list` (GET /api/v1/decision-requests).
 */
export function useDecisionRequests(params: MaybeRefOrGetter<{ run_id?: string }> = {}) {
  const moment = useMomentStore()
  return useQuery({
    queryKey: computed(() => documentKeys.list('decision-requests', toValue(params), moment.params)),
    queryFn: () => pendingOperation<Envelope<DecisionRequest[]>>(DOCUMENT_OPERATIONS.requests)(),
    retry: false,
  })
}

/** Ответ на подпись или заверение: запись журнала и номер критического действия. */
export interface SignatureAccepted {
  event_id: string
  ca_id?: string | null
}

/** Сбросить всё, что связано с документом: запросы решения, документ, паспорт изделия. */
function useDocumentInvalidation() {
  const queryClient = useQueryClient()
  return (documentId: string) => {
    void queryClient.invalidateQueries({ queryKey: documentKeys.one(documentId) })
    void queryClient.invalidateQueries({ queryKey: documentKeys.list() })
    void queryClient.invalidateQueries({ queryKey: ['item'] })
  }
}

/** Подпись этапа агентом токена. */
export interface SignDocumentVars {
  document_id: string
  version: number
  stage: number
  key_ref: string
  signature_b64: string
}

/** Отказ в согласовании с замечанием. */
export interface DeclineDocumentVars {
  document_id: string
  version: number
  stage: number
  comment: string
}

/** Печать бумажного экземпляра. */
export interface PrintPaperVars {
  document_id: string
  version: number
}

/** Скан бумажной подписи и заверение. */
export interface AttestPaperVars {
  document_id: string
  version: number
  stage: number
  file: File
  archive_no: string
}

/** «Запросить решение»: действие и объект (`item:‹id›`, `nonconformity:‹id›`). */
export interface RequestDecisionVars {
  action: string
  subject_ref: string
}

/** Подписать документ агентом токена (FR-136): подпись над отпечатком, этап маршрута. */
export function useSignDocument() {
  const invalidate = useDocumentInvalidation()
  return useMutation<Envelope<SignatureAccepted>, ApiError, SignDocumentVars>({
    mutationFn: pendingOperation(DOCUMENT_OPERATIONS.sign),
    onSuccess: (_r, v) => invalidate(v.document_id),
  })
}

/** Не согласовать — вернуть с замечанием (замечание обязательно, FR-136). */
export function useDeclineDocument() {
  const invalidate = useDocumentInvalidation()
  return useMutation<Envelope<SignatureAccepted>, ApiError, DeclineDocumentVars>({
    mutationFn: pendingOperation(DOCUMENT_OPERATIONS.decline),
    onSuccess: (_r, v) => invalidate(v.document_id),
  })
}

/** Напечатать бумажный экземпляр с QR (FR-139): сервер отдаёт лист для печати. */
export function usePrintPaper() {
  return useMutation<Envelope<{ print_url: string }>, ApiError, PrintPaperVars>({
    mutationFn: pendingOperation(DOCUMENT_OPERATIONS.print),
  })
}

/**
 * Загрузить скан и заверить бумажную подпись (FR-139, AD-43): заверитель ≠
 * подписант, учётный номер оригинала, подпись заверителя уровня 2.
 */
export function useAttestPaper() {
  const invalidate = useDocumentInvalidation()
  return useMutation<Envelope<SignatureAccepted>, ApiError, AttestPaperVars>({
    mutationFn: pendingOperation(DOCUMENT_OPERATIONS.attestPaper),
    onSuccess: (_r, v) => invalidate(v.document_id),
  })
}

/**
 * «Запросить решение» (FR-146): документ с маршрутом подписей у тех, у кого
 * есть полномочия (`document.version.requested`). Объект — изделие или
 * несоответствие; шаблон выбирает сервер по действию.
 */
export function useRequestDecision() {
  const queryClient = useQueryClient()
  return useMutation<Envelope<{ document_id: string }>, ApiError, RequestDecisionVars>({
    mutationFn: pendingOperation(DOCUMENT_OPERATIONS.request),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: documentKeys.list() }),
  })
}
