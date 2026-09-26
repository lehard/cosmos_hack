/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DocumentApprovalStage } from './documentApprovalStage';
import type { DocumentDecline } from './documentDecline';
import type { DocumentPaperMark } from './documentPaperMark';
import type { DocumentRouteStage } from './documentRouteStage';
import type { DocumentSignatureView } from './documentSignatureView';
import type { DocumentSummaryField } from './documentSummaryField';
import type { DocumentVersionRef } from './documentVersionRef';
import type { DocumentViewClass } from './documentViewClass';
import type { DocumentViewContent } from './documentViewContent';
import type { DocumentViewStatus } from './documentViewStatus';
import type { DocumentViewVerification } from './documentViewVerification';
import type { DrillRef } from './drillRef';

export interface DocumentView {
  basis_seq: number;
  class?: DocumentViewClass;
  /** Канонические данные документа (JCS без rendering_hash). */
  content: DocumentViewContent;
  /** Отказы в согласовании с замечаниями (FR-136). */
  declines?: DocumentDecline[];
  /** Отпечаток документа streebog256:…; в QR печатной рамки. */
  doc_digest: string;
  /** @minimum 1 */
  doc_format_version: number;
  /** Вид документа. */
  doc_type?: string;
  document_id: string;
  /** Версия ещё не зафиксирована: показан текущий сбор из истории (номер — следующей версии). */
  live?: boolean;
  /** Бумажные экземпляры: напечатан, подписан, уничтожен (AD-12). */
  paper?: DocumentPaperMark[];
  /** ant:doc:‹id›:‹отпечаток› — для печатной рамки. */
  qr: string;
  /** H(render(шаблон@версия, content)). */
  rendering_hash: string;
  /** Маршрут с подписями по этапам: засчитана или нет и почему (AD-43). */
  route?: DocumentRouteStage[];
  /** Реакция document.route.closed версии. */
  route_closed_event_id?: string;
  signatures: DocumentSignatureView[];
  /** Содержимое для подписи (AD-12, AD-14): base64 канонических байт {content, rendering_hash, template_ref, doc_format_version}, отпечаток которых — doc_digest. */
  signing_payload_b64?: string;
  /** События-источники документа. */
  source_event_ids: string[];
  /** Обязательные подписи — замороженный набор (AD-43). */
  stages: DocumentApprovalStage[];
  status: DocumentViewStatus;
  subject: DrillRef;
  /** Объект документа для людей. */
  subject_label?: string;
  /** Поля сводки уровня 2 — входят в отпечаток (AD-12). */
  summary_fields: DocumentSummaryField[];
  /** Предыдущая версия документа. */
  supersedes_version?: number;
  template: string;
  title: string;
  /** Как проверены подписи при закрытии маршрута; demo — без агента токена (Д-30). */
  verification?: DocumentViewVerification;
  /** @minimum 1 */
  version: number;
  /** Версии документа: у каждой свой отпечаток, подписи прежней остаются при ней (AD-12). */
  versions?: DocumentVersionRef[];
}
