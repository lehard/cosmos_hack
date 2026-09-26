/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { AccessGrantEntryAction } from './accessGrantEntryAction';
import type { AccessGrantEntryKind } from './accessGrantEntryKind';

export interface AccessGrantEntry {
  /** Выдано, отозвано, установлено. */
  action: AccessGrantEntryAction;
  at: string;
  /** Кто выдал (псевдоним). */
  by: string;
  /** Критическое действие CA-‹n› (AD-28). */
  ca_ref?: string;
  /** Документ выдачи с маршрутом подписей. */
  document_id?: string;
  event_id: string;
  /** Что выдано или отозвано. */
  kind: AccessGrantEntryKind;
  person_id?: string;
  scope?: string;
  /** Вторая подпись независимой стороны (AD-11). */
  second_signature_by?: string;
  seq: number;
  /** Роль, полномочие, клеймо, квалификация. */
  subject_id: string;
}
