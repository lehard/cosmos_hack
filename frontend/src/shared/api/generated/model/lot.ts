/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { LotStatus } from './lotStatus';

export interface Lot {
  /**
     * Фактическое количество при регистрации; null — ещё не зарегистрирована.
     * @minimum 0
     * @nullable
     */
  actual_quantity: number | null;
  basis_seq: number;
  /** @nullable */
  certificate_present: boolean | null;
  /** Сдерживание партии (ось containment, владелец nonconformity). */
  containment?: string;
  /**
     * Количество по документам поставщика.
     * @minimum 0
     */
  declared_quantity: number;
  /** Номер партии во внешней системе (соответствие — reference.external_id.mapped). */
  external_ref?: string;
  /** @minimum 0 */
  issued_quantity: number;
  item_type_id?: string;
  lot_id: string;
  received_at?: string;
  registered_at?: string;
  /** Состояние партии; блок партии — ось «сдерживание» nonconformity. */
  status: LotStatus;
  supplier?: string;
}
