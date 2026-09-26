/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface AccessAuditParameters {
  /**
     * Интервал контрольных точек, с.
     * @minimum 1
     */
  checkpoint_interval_s: number;
  /**
     * Предельный разрыв контрольных точек, с.
     * @minimum 1
     */
  checkpoint_max_gap_s: number;
  /** Перечень критических типов записей. */
  critical_types: string[];
  /** Отпечаток ключа хранителя (streebog256:…). */
  keeper_key_fingerprint: string;
  policy_seq: number;
  /** Подписчики шины безопасности (AD-24). */
  security_bus_subscribers: string[];
  set_at?: string;
  set_by?: string;
}
