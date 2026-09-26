/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { NCDecisionActionDisposition } from './nCDecisionActionDisposition';
import type { NCDecisionActionOperation } from './nCDecisionActionOperation';
import type { NCDecisionActionResolution } from './nCDecisionActionResolution';

export interface NCDecisionAction {
  /** Пройдёт гарды для вошедшего: доменный гард операции, полномочие, разрешение на отклонение. */
  allowed: boolean;
  /** Действующее разрешение на отклонение, по которому пройдёт решение (ремонт, «как есть», приёмка по разрешению). */
  concession_id?: string;
  /** Что произойдёт по делу: с изделием и маршрутом, кому уйдёт действие, чьё ещё решение нужно. */
  consequences: string[];
  /** disposition команды nonconformity.disposition.set. */
  disposition?: NCDecisionActionDisposition;
  /** Надпись кнопки: действие и направление. */
  label: string;
  /** Операция API. */
  operation: NCDecisionActionOperation;
  /** Основание в политике: полномочие или правило подписи. */
  policy_ref?: string;
  /** resolution команды nonconformity.presentation.resolve. */
  resolution?: NCDecisionActionResolution;
  /** Что изменится в системе: статусы, 1С, история. */
  technical_consequences: string[];
  /** Почему доступно или почему нет — словами. */
  why_available: string;
}
