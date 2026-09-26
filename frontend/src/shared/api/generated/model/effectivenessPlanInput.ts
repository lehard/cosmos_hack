/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface EffectivenessPlanInput {
  /**
     * Базовый уровень до меры.
     * @maxLength 500
     */
  baseline?: string;
  /**
     * Усиленный контроль на время окна.
     * @maxLength 500
     */
  enhanced_control?: string;
  /**
     * Что измеряем.
     * @maxLength 500
     */
  metric?: string;
  /**
     * Критерий успеха.
     * @maxLength 500
     */
  success_criterion?: string;
  /**
     * Окно наблюдения, дней.
     * @minimum 0
     * @maximum 3650
     */
  window_days?: number;
  [key: string]: unknown;
 }
