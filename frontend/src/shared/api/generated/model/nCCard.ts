/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { NCCardStatus } from './nCCardStatus';
import type { NCEvidence } from './nCEvidence';
import type { NCHappened } from './nCHappened';
import type { NCItemAxes } from './nCItemAxes';
import type { NCRecordRef } from './nCRecordRef';
import type { NCSystemAnalysis } from './nCSystemAnalysis';
import type { NCToDecide } from './nCToDecide';

export interface NCCard {
  axes: NCItemAxes;
  /** seq, на котором построена карточка (для basis_seq команд, AD-39). */
  basis_seq: number;
  evidence: NCEvidence;
  happened: NCHappened;
  /** Решения людей с подписью (отдельно от вывода системы). */
  human_decisions: NCRecordRef[];
  item_id: string;
  item_label: string;
  nc_id: string;
  /** Номер для людей. */
  number: string;
  status: NCCardStatus;
  system_analysis: NCSystemAnalysis;
  to_decide: NCToDecide;
}
