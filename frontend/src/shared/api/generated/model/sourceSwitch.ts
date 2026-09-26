/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface SourceSwitch {
  /** Кто решил (псевдоним). */
  actor?: string;
  at: string;
  enabled: boolean;
  reason: string;
  /** basis_seq для следующей команды над источником (AD-39). */
  seq: number;
  source_id: string;
}
