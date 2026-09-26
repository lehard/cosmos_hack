/**
 * Документы, маршруты подписей и запросы решения (эпик 11: FR-65, FR-66,
 * FR-136, FR-139; AD-12, AD-13, AD-43). Ключи кэша — по соглашению
 * shared/api/keys.ts.
 *
 * Все операции — сгенерированный клиент по contracts/openapi.yaml (эпик 28):
 * список запросов решения, подпись этапа, отказ с замечанием, печать с QR,
 * заверение бумажной подписи (скан — через хранилище материалов) и «Запросить
 * решение». Заголовок команды (AD-7, AD-39) собирает `commandHeader`.
 */
import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import {
  documentsPaperAttest,
  documentsPaperPrint,
  documentsPaperPrintView,
  documentsRequestList,
  documentsSignatureDecline,
  documentsSignatureRecord,
  documentsVersionRequest,
  materialsMaterialUpload,
} from '@/shared/api/generated/client'
import type { DsseEnvelope, PrintAccepted, Receipt, RequestAccepted } from '@/shared/api/generated/model'
import { entityKeys } from '@/shared/api/keys'
import type { Envelope } from '@/shared/api/response'
import { newCommandId } from '@/shared/lib/command-id'
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
    // Эпик 28: операция в контракте — список без обёртки items для виджета.
    queryFn: async (): Promise<Envelope<DecisionRequest[]>> => {
      const r = await documentsRequestList({ ...moment.params, ...toValue(params) })
      return { data: r.data.items as unknown as DecisionRequest[], headers: r.headers }
    },
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

/**
 * Заголовок команды (AD-7, AD-39): id намерения, seq, на котором клиент видел
 * документ, версия политики сеанса и рабочее место (барьер 2, AD-15).
 */
export interface DocumentCommandHeader {
  command_id?: string
  basis_seq?: number
  policy_seq?: number
  workplace_id?: string
}

/** Поля заголовка команды для тела запроса; нет id — новый UUIDv7. */
function commandHeader(h: DocumentCommandHeader) {
  return {
    command_id: h.command_id ?? newCommandId(),
    basis_seq: h.basis_seq ?? 0,
    policy_seq: h.policy_seq ?? 0,
    ...(h.workplace_id ? { workplace_id: h.workplace_id } : {}),
  }
}

/** Подпись этапа агентом токена. */
export interface SignDocumentVars extends DocumentCommandHeader {
  document_id: string
  version: number
  stage: number
  key_ref: string
  signature_b64: string
  /** Отпечаток, который увидел подписант. */
  doc_digest?: string
  /** Конверт пакета document-signature целиком: его проверяет сервер (signing, Д-59). */
  signature?: DsseEnvelope
}

/** Отказ в согласовании с замечанием (`documents.signature.decline`). */
export interface DeclineDocumentVars extends DocumentCommandHeader {
  document_id: string
  version: number
  stage: number
  comment: string
  /** Отпечаток, который увидел подписант; расхождение — `signing.document_changed`. */
  doc_digest?: string
}

/** Печать бумажного экземпляра (`documents.paper.print`). */
export interface PrintPaperVars extends DocumentCommandHeader {
  document_id: string
  /** Версия; 0 — текущая. */
  version: number
  /** Номер экземпляра. */
  copy_no?: string
}

/** Итог печати: квитанция сервера и готовая печатная форма (HTML с рамкой и QR). */
export interface PrintResult extends PrintAccepted {
  /** Полная страница для печати (`documents.paper.print_view`). */
  html: string
}

/** Скан бумажной подписи и заверение (`documents.paper.attest`). */
export interface AttestPaperVars extends DocumentCommandHeader {
  document_id: string
  version: number
  stage: number
  /** Скан распечатки с QR — загружается в хранилище материалов перед заверением. */
  file: File
  /** Учётный номер бумажного оригинала в архиве ОТК. */
  archive_no: string
  /** Кто подписал ручкой — ожидаемый подписант этапа из QR. */
  signer_person_id: string
  /** Отпечаток из QR распечатки. */
  doc_digest: string
  /** Изделие — метаданные материала. */
  item_id?: string
}

/** «Запросить решение»: действие и объект (`item:‹id›`, `nonconformity:‹id›`). */
export interface RequestDecisionVars extends DocumentCommandHeader {
  action: string
  subject_ref: string
  comment?: string
}

/** Подписать документ агентом токена (FR-136): подпись над отпечатком, этап маршрута. */
export function useSignDocument() {
  const invalidate = useDocumentInvalidation()
  return useMutation<Envelope<SignatureAccepted>, ApiError, SignDocumentVars>({
    // Эпик 28: documents.signature.record — подпись этапа с конвертом агента.
    mutationFn: async ({ document_id, command_id, basis_seq, policy_seq, workplace_id, ...rest }) => {
      const r = await documentsSignatureRecord(document_id, {
        ...rest,
        ...commandHeader({ command_id, basis_seq, policy_seq, workplace_id }),
      })
      return { data: r.data as unknown as SignatureAccepted, headers: r.headers }
    },
    onSuccess: (_r, v) => invalidate(v.document_id),
  })
}

/** Не согласовать — вернуть с замечанием (замечание обязательно, FR-136). */
export function useDeclineDocument() {
  const invalidate = useDocumentInvalidation()
  return useMutation<Envelope<Receipt>, ApiError, DeclineDocumentVars>({
    mutationFn: async ({ document_id, command_id, basis_seq, policy_seq, workplace_id, ...rest }) => {
      const r = await documentsSignatureDecline(document_id, {
        ...rest,
        ...commandHeader({ command_id, basis_seq, policy_seq, workplace_id }),
      })
      return { data: r.data, headers: r.headers }
    },
    onSuccess: (_r, v) => invalidate(v.document_id),
  })
}

/**
 * Напечатать бумажный экземпляр с QR (FR-139): запись «напечатан»
 * (`documents.paper.print`), затем печатная форма той версии, что вернул
 * сервер (`documents.paper.print_view`, у живой карты — новая версия): HTML
 * с рамкой и QR рисует сервер (AD-12), интерфейс только открывает его.
 */
export function usePrintPaper() {
  const invalidate = useDocumentInvalidation()
  return useMutation<Envelope<PrintResult>, ApiError, PrintPaperVars>({
    mutationFn: async ({ document_id, command_id, basis_seq, policy_seq, workplace_id, ...rest }) => {
      const r = await documentsPaperPrint(document_id, {
        ...rest,
        ...commandHeader({ command_id, basis_seq, policy_seq, workplace_id }),
      })
      const view = await documentsPaperPrintView(document_id, { version: r.data.version })
      return { data: { ...r.data, html: view.data.html }, headers: r.headers }
    },
    onSuccess: (_r, v) => invalidate(v.document_id),
  })
}

/**
 * Загрузить скан и заверить бумажную подпись (FR-139, AD-43): скан — в
 * хранилище материалов (`materials.material.upload`, вид scan), затем
 * заверение с адресом скана и учётным номером оригинала; заверитель ≠
 * подписант, подпись заверителя уровня 2 — гарды сервера.
 */
export function useAttestPaper() {
  const invalidate = useDocumentInvalidation()
  return useMutation<Envelope<Receipt>, ApiError, AttestPaperVars>({
    mutationFn: async ({ document_id, file, archive_no, item_id, command_id, basis_seq, policy_seq, workplace_id, ...rest }) => {
      const scan = await materialsMaterialUpload(
        file,
        { kind: 'scan', ...(item_id ? { item_id } : {}) },
        { headers: { 'Content-Type': file.type || 'application/octet-stream' } },
      )
      const r = await documentsPaperAttest(document_id, {
        ...rest,
        paper_original_no: archive_no,
        scan_address: scan.data.material_address,
        ...commandHeader({ command_id, basis_seq, policy_seq, workplace_id }),
      })
      return { data: r.data, headers: r.headers }
    },
    onSuccess: (_r, v) => invalidate(v.document_id),
  })
}

/**
 * «Запросить решение» (FR-146): документ с маршрутом подписей у тех, у кого
 * есть полномочия (`documents.version.request` → `document.version.requested`).
 * Объект — изделие или несоответствие; шаблон выбирает сервер по действию.
 */
export function useRequestDecision() {
  const queryClient = useQueryClient()
  return useMutation<Envelope<RequestAccepted>, ApiError, RequestDecisionVars>({
    mutationFn: async ({ command_id, basis_seq, policy_seq, workplace_id, ...rest }) => {
      const r = await documentsVersionRequest({ ...rest, ...commandHeader({ command_id, basis_seq, policy_seq, workplace_id }) })
      return { data: r.data, headers: r.headers }
    },
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: documentKeys.list() }),
  })
}
