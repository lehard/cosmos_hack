/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ProblemParams } from './problemParams';
import type { Violation } from './violation';

export interface Problem {
  /** Что пользователь может сделать вместо: например, «Запросить решение» (FR-146). */
  allowed_actions?: string[];
  /** Для journal.stale_*: на каком seq проверено. */
  basis_seq?: number;
  /** Критическое действие, записанное из-за отказа. */
  ca_ref?: string;
  /** Код из contracts/errors.yaml. */
  code: string;
  /** Пояснение по-русски для этого случая. */
  detail?: string;
  /** URI конкретного случая (путь запроса или id записи карантина). */
  instance?: string;
  /** Параметры шаблона текста интерфейса. */
  params?: ProblemParams;
  /** Запись карантина (FR-30). */
  quarantine_id?: string;
  /**
     * HTTP-статус.
     * @minimum 200
     * @maximum 599
     */
  status: number;
  /** Краткий заголовок по-русски (из каталога). */
  title: string;
  /** URI типа проблемы: urn:ant:problem:‹код›. */
  type: string;
  /** Нарушения по полям или элементам. */
  violations?: Violation[];
}
