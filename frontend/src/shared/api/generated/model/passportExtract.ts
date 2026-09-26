/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DrillRef } from './drillRef';
import type { PassportExtractDirection } from './passportExtractDirection';
import type { PassportExtractOriginStatus } from './passportExtractOriginStatus';

export interface PassportExtract {
  /**
     * Квитанция партнёра; null — ещё нет.
     * @nullable
     */
  acknowledged: boolean | null;
  at: string;
  direction: PassportExtractDirection;
  /** Исходящая выписка — документ с маршрутом (AD-19). */
  document_id?: string;
  /** Отпечаток выписки. */
  extract_digest: string;
  /** Глобальный ID предмета «код_предприятия:локальный_id» (соглашения спайна). */
  global_id?: string;
  /** Плавка. */
  heat_no?: string;
  /** Что в выписке для человека: материал, партия или изделие (эпик 41). */
  label?: string;
  /** Исходные байты выписки в хранилище материалов. */
  material_address?: string;
  /** Межзаводское сообщение. */
  message_id?: string;
  /** Происхождение подтверждено / подтверждено только сервером отправителя / не подтверждено (AD-19); у исходящей — not_applicable. */
  origin_status: PassportExtractOriginStatus;
  partner_code: string;
  /** Партия или изделие. */
  subject?: DrillRef;
}
