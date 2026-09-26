/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ItemCarrier } from './itemCarrier';
import type { ItemDocumentRef } from './itemDocumentRef';
import type { ItemPassportIdentification } from './itemPassportIdentification';
import type { ItemStatus } from './itemStatus';
import type { ItemZone } from './itemZone';
import type { PassportEntry } from './passportEntry';

export interface ItemPassport {
  /** seq, на котором построен паспорт (для команд, AD-39). */
  basis_seq: number;
  carriers: ItemCarrier[];
  documents: ItemDocumentRef[];
  /** Записи паспорта по времени. */
  entries: PassportEntry[];
  /** Идентификация (AD-16): под сомнением — изоляция до повторной идентификации. */
  identification: ItemPassportIdentification;
  /** Инциденты, в области которых изделие. */
  incidents: string[];
  item_id: string;
  item_revision: string;
  item_type_id: string;
  label: string;
  lot_ids: string[];
  nonconformities: string[];
  /** Задание 1С. */
  order_id?: string;
  /** Закреплённая версия процесса (хеш, AD-17). */
  process_version: string;
  status: ItemStatus;
  step_key?: string;
  zones: ItemZone[];
}
