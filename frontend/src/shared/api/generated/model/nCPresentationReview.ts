/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { NCRecordRef } from './nCRecordRef';

export interface NCPresentationReview {
  /** Прежнее решение на точке (автор, время). */
  decision: NCRecordRef;
  /** Что было в основании при подписи (результаты методов решения). */
  known_at_decision: NCRecordRef[];
  /** Что пришло после решения (с числами режима reading, если есть). */
  new_facts: NCRecordRef[];
  /** Почему именно эти факты значимы: связь с операцией до приёмки, уставка, специальный процесс, чего не было при подписи. */
  why_significant?: string[];
}
