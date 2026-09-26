/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { CriticalActionCaGroup } from './criticalActionCaGroup';
import type { CriticalActionSigner } from './criticalActionSigner';
import type { DrillRef } from './drillRef';

export interface CriticalAction {
  /** Название действия словами (перечень критических действий AD-28 или каталог типов). */
  action_name?: string;
  /** Тип основной записи (семейство.сущность.действие). */
  action_type: string;
  /** Кто — имя из справочника сотрудников (псевдоним кейса §4.6). */
  actor_display?: string;
  /** Кто (псевдоним). */
  actor_id?: string;
  /** Стало. */
  after: string;
  /** Заверитель бумажной подписи (AD-43). */
  attested_by?: string;
  /** Полномочие. */
  authority_id?: string;
  /** Основания. */
  basis_event_ids: string[];
  /** Было. */
  before: string;
  ca_group: CriticalActionCaGroup;
  /** @minimum 1 */
  ca_no: number;
  /** CA-‹n› — позиция в цепочке критических действий. */
  ca_ref: string;
  cancel_reason?: string;
  /** Отменено записью CA-…. */
  cancelled_by?: string;
  /** Отменяет CA-… (отмена — только новой записью с причиной). */
  cancels?: string;
  /** Обязательство основной записи (AD-44). */
  main_commit: string;
  /** Подписанная запись основного журнала. */
  main_event_id: string;
  object: DrillRef;
  /** Поток объекта (stream_ref). */
  object_ref: string;
  /** Ревизия политики. */
  policy_seq?: number;
  recorded_at: string;
  /** Подписанты основной записи с классом ключа (AD-10, AD-14): кто подписал решение, которое записано в CA. */
  signers?: CriticalActionSigner[];
  /** Цифровое клеймо (FR-145). */
  stamp_id?: string;
}
