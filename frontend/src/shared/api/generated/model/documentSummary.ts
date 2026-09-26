/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DocumentAwaiting } from './documentAwaiting';
import type { DocumentSummaryClass } from './documentSummaryClass';
import type { DocumentSummaryPaperStatus } from './documentSummaryPaperStatus';
import type { DocumentSummaryState } from './documentSummaryState';
import type { DocumentSummaryStatus } from './documentSummaryStatus';
import type { DrillRef } from './drillRef';

export interface DocumentSummary {
  /** Кто должен подписать сейчас: первый незакрытый этап маршрута; нет — маршрут закрыт или не начат. */
  awaiting?: DocumentAwaiting;
  /** Класс документа: запись, решение, требование, вход. */
  class?: DocumentSummaryClass;
  closed_at?: string;
  doc_digest?: string;
  /** Вид документа: traveler, nc_statement, nc_disposition, generic. */
  doc_type?: string;
  document_id: string;
  drafted_at?: string;
  /** Изделия, к которым относится документ (фильтр «по изделию»). */
  item_ids?: string[];
  /** Статус бумажного экземпляра. */
  paper_status?: DocumentSummaryPaperStatus;
  /** Процесс (главный bpmn:process), к которому относится документ. */
  process_id?: string;
  /** Версия процесса, по которой сформирован документ. */
  process_version_id?: string;
  /** Сколько этапов маршрута закрыто. */
  stages_done?: number;
  /** Сколько этапов в маршруте текущей версии. */
  stages_total?: number;
  /** Состояние для реестра: черновик, на подписи, подписан, аннулирован, возвращён с замечанием, на бумаге (напечатан, ждёт заверения). */
  state?: DocumentSummaryState;
  /** live — сопроводительная карта собирается из истории, версия ещё не зафиксирована; returned — подписант вернул версию с замечанием. */
  status: DocumentSummaryStatus;
  /** Объект документа: изделие, несоответствие, партия… */
  subject: DrillRef;
  /** Объект документа для людей: «Ф-017», «НС-01», «Фланец люка, версия v1». */
  subject_label?: string;
  /** Шаблон@версия из нормативного слоя. */
  template: string;
  title: string;
  /** Последнее событие документа: версия, подпись, отказ, бумага, закрытие. */
  updated_at?: string;
  /** @minimum 1 */
  version: number;
  /** Сколько версий сформировано. */
  versions?: number;
}
