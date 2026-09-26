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
  /** Что произойдёт по делу: изделие, маршрут, блокировка, область риска, кому уйдёт действие (строки 1С и истории — в technical_consequences). */
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
  /** Что изменится в системе: статусы, 1С, история. */
  technical_consequences?: string[];
  /** Почему доступно или почему нет — словами. */
  why_available: string;
}
