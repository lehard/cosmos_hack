/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { GenealogyHold } from './genealogyHold';
import type { IdentificationQuestion } from './identificationQuestion';
import type { ItemCarrier } from './itemCarrier';
import type { ItemDocumentRef } from './itemDocumentRef';
import type { ItemIncident } from './itemIncident';
import type { ItemIntervention } from './itemIntervention';
import type { ItemPassportIdentification } from './itemPassportIdentification';
import type { ItemRefChange } from './itemRefChange';
import type { ItemStatus } from './itemStatus';
import type { ItemZone } from './itemZone';
import type { PassportEntry } from './passportEntry';
import type { WitnessResult } from './witnessResult';

export interface ItemPassport {
  /** seq, на котором построен паспорт (для команд, AD-39). */
  basis_seq: number;
  carriers: ItemCarrier[];
  documents: ItemDocumentRef[];
  /** Записи паспорта по времени. */
  entries: PassportEntry[];
  /** Сдерживание по генеалогии: блок партии или компонента (AD-42). */
  genealogy_holds?: GenealogyHold[];
  /** Идентификация (AD-16): под сомнением — изоляция до повторной идентификации. */
  identification: ItemPassportIdentification;
  /** Идентификация под сомнением: причина, кандидаты; открытое — изоляция до повторной идентификации (AD-16). */
  identification_questions?: IdentificationQuestion[];
  /** Статус изделия в каждом инциденте: что известно и что делать (FR-62). */
  incident_statuses?: ItemIncident[];
  /** Инциденты, в области которых изделие. */
  incidents: string[];
  /** Вмешательства в собранное изделие (FR-21). */
  interventions?: ItemIntervention[];
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
  /** Изменения справочника с действием в прошлом, затронувшие изделие (AD-31). */
  reference_changes?: ItemRefChange[];
  /** Изделие, из которого выделено разделением 1→N (FR-15). */
  split_from?: string;
  status: ItemStatus;
  step_key?: string;
  /** Результаты образца-свидетеля садки или групповой операции (FR-15). */
  witness_results?: WitnessResult[];
  zones: ItemZone[];
}
