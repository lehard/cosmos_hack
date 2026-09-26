/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ProcessElementKind } from './processElementKind';
import type { ProcessElementProperties } from './processElementProperties';
import type { ProcessElementThresholds } from './processElementThresholds';

export interface ProcessElement {
  /** id элемента BPMN. */
  id: string;
  kind: ProcessElementKind;
  /** Цех (дорожка). */
  lane?: string;
  name: string;
  /** id следующих элементов. */
  next?: string[];
  /** Ключи process.properties.*; значения — строка, число, логическое или null. */
  properties: ProcessElementProperties;
  /** ant:properties/@stepKey — сопоставляет элементы версий. */
  step_key?: string;
  /** Пороги уверенности карты реакций по видам дефектов, б. п. */
  thresholds?: ProcessElementThresholds;
}
