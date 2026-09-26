/**
 * Документы, маршруты подписей и запросы решения (эпик 11; FR-65, FR-66,
 * FR-136, FR-139; AD-12, AD-13, AD-43).
 *
 * Поля — как в семействе `document` (contracts/events/document): набор
 * обязательных подписей `required_approvals` замораживается при
 * `document.version.drafted`, подписи — `document.signature.recorded`, закрытие
 * — только реакция `document.route.closed`. Какие подписи засчитаны, решает
 * модуль documents (AD-43); интерфейс только показывает (`counted`).
 */
import type { DocumentVersionDraftedV1, EvidenceRef } from '@/shared/contracts/events'
import type { DocumentStatus, RecordSignature } from '@/entities/item'

export type { EvidenceRef }

/** Описание этапа маршрута — элемент `required_approvals` (AD-43). */
export type RequiredApproval = DocumentVersionDraftedV1['required_approvals'][number]

/** Подпись на этапе маршрута. */
export interface StageSignature extends RecordSignature {
  /** Запись подписи (`document.signature.recorded`). */
  event_id: string
  /** Когда подписано. */
  signed_at: string
  /** Засчитана модулем documents: текущий отпечаток, полномочие и клеймо на `seq`, разделение обязанностей. */
  counted: boolean
  /** Подпись по прежней версии документа — видна, но не засчитывается. */
  previous_version?: boolean
}

/** Этап маршрута подписей с тем, что уже подписано (FR-66, AD-13). */
export interface RouteStage extends RequiredApproval {
  /** Кто подписывает на этапе — подпись полномочия или роли для людей. */
  authority_label: string
  /** Сколько засчитанных подписей нужно на этапе (вычисляет RequiredApprovals, AD-43). */
  required: number
  /** Подписи этапа. */
  signatures: StageSignature[]
}

/** Заголовок документа: что подписывается. */
export interface DocumentHead {
  document_id: string
  version: number
  /** Шаблон `‹id›@‹версия›`. */
  template_ref: string
  /** Вид документа — ключ `documents.types.*` в snake_case. */
  doc_type: string
  /** Отпечаток — входит в QR бумажного экземпляра (AD-43). */
  doc_digest: string
  status: DocumentStatus
  drafted_at: string
}

/** Документ с маршрутом подписей. */
export interface RoutedDocument extends DocumentHead {
  route: RouteStage[]
}

/** Похожее прошлое решение для редкого подписанта (FR-136): принятое или отклонённое. */
export interface SimilarDecision {
  /** Ссылка на прошлый документ или несоответствие. */
  ref_id: string
  /** Номер для людей. */
  number: string
  /** Краткое описание от сервера. */
  summary: string
  /** Чем кончилось. */
  outcome: 'accepted' | 'rejected'
  decided_at: string
}

/** Материал-довод: запись журнала и её материалы. */
export interface EvidenceItem {
  event_id: string
  event_type: string
  variant?: string | null
  occurred_at: string
  params?: Record<string, string | number>
  /** Материалы записи; иллюстрация помечена явно (FR-102). */
  evidence_refs?: EvidenceRef[]
}

/** Вид предлагаемого решения в карточке редкого подписанта. */
export type ProposalKind = 'disposition' | 'concession' | 'presentation' | 'other'

/**
 * Карточка «требуется ваше решение» (FR-136): что предлагается; почему пришло к
 * вам; доказательства; похожие случаи; чьи подписи нужны и чьи уже есть;
 * остаток срока. Подписывается документ с маршрутом (AD-13).
 */
export interface DecisionRequest {
  /** Документ, который ждёт подписи. */
  document: RoutedDocument
  proposal: {
    kind: ProposalKind
    /** Код предложения: вариант решения (`use_as_is`), `concession`, `accept`… */
    code: string
    /** Что предлагается — сводка от сервера. */
    summary: string
    item_id: string
    item_label: string
    nc_id?: string | null
    nc_number?: string | null
  }
  /** Почему это пришло к вам: режим автоматизации 4–5 (FR-50) и основание. */
  escalation: { reason: string; automation_mode: number; rule_id?: string | null; rule_rev?: string | null }
  evidence: EvidenceItem[]
  similar_accepted: SimilarDecision[]
  similar_rejected: SimilarDecision[]
  /** Этап маршрута, который ждёт подписи текущего пользователя; null — не ваш черёд. */
  my_stage: number | null
  /** Ожидаемый подписант этапа — он же в QR бумажного экземпляра (AD-43). */
  expected_signer: string | null
  /** Срок решения. */
  due_at: string | null
}

/** Поле сводки окна подписи уровня 2: подпись и значение (готовый текст или ключ). */
export interface SummaryField {
  labelKey: string
  /** Готовое значение (номер, текст причины). */
  value?: string
  /** Или ключ текста значения. */
  valueKey?: string
  valueParams?: Record<string, string | number>
}

/** Сводка уровня 2 — от 3 до 7 полей (FR-66). */
export const SUMMARY_MAX = 7
