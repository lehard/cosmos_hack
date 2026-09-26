/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DrillRef } from './drillRef';
import type { SecurityEventSeverity } from './securityEventSeverity';

export interface SecurityEvent {
  /** Критическое действие, если событие его породило. */
  ca_ref?: string;
  event_id: string;
  /** Тип записи шины безопасности: семейство security и выдача прав (policy.role.*, policy.authority.*). */
  event_type: string;
  object?: DrillRef;
  occurred_at: string;
  /** Сотрудник словами. */
  person_display?: string;
  /** Сотрудник, к которому относится событие (вход, отказ, допуск, присутствие, выдача прав). */
  person_id?: string;
  /** @minimum 1 */
  seq: number;
  severity: SecurityEventSeverity;
  source_id?: string;
  /** Краткое описание по-русски. */
  summary: string;
  /** Пост (допуск, присутствие по СКУД). */
  workplace_id?: string;
}
