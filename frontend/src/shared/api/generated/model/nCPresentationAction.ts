/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { NCPresentationActionOperation } from './nCPresentationActionOperation';
import type { NCPresentationActionOutcome } from './nCPresentationActionOutcome';
import type { NCPresentationActionResolution } from './nCPresentationActionResolution';

export interface NCPresentationAction {
  /** Пройдёт гарды для вошедшего (полномочие точки, разделение обязанностей, блок, результаты методов). */
  allowed: boolean;
  /** Что произойдёт: изделие, маршрут, блокировка, область риска, 1С, история. */
  consequences: string[];
  /** Надпись кнопки: действие и направление. */
  label: string;
  /** Операция API. */
  operation: NCPresentationActionOperation;
  /** outcome команды nonconformity.presentation.review. */
  outcome?: NCPresentationActionOutcome;
  /** Основание в политике: полномочие точки или правило подписи. */
  policy_ref?: string;
  /** resolution команды nonconformity.presentation.resolve. */
  resolution?: NCPresentationActionResolution;
  /** Почему доступно или почему нет — словами. */
  why_available: string;
}
