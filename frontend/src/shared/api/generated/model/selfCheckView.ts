/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { SelfCheckFinding } from './selfCheckFinding';

export interface SelfCheckView {
  checked_at: string;
  findings: SelfCheckFinding[];
  /** Критических находок нет. */
  ok: boolean;
  /** «инициализация без критических ошибок» или перечень критических находок. */
  summary: string;
}
