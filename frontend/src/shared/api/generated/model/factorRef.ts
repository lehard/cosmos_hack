/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { FactorRefFactor } from './factorRefFactor';

export interface FactorRef {
  factor: FactorRefFactor;
  /** Значение словами из справочника по виду фактора: «Сварочный источник ИС-2», «Партия П-117», имя исполнителя. Нет — показать value. */
  label?: string;
  value: string;
}
