/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DrillRef } from './drillRef';
import type { LotCardStatus } from './lotCardStatus';
import type { LotIssue } from './lotIssue';
import type { LotItem } from './lotItem';

export interface LotCard {
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
  /** Акт входного контроля, ярлык соответствия, выписка поставщика. */
  documents: DrillRef[];
  /** Номер партии во внешней системе (соответствие — reference.external_id.mapped). */
  external_ref?: string;
  /** Номер плавки партии. */
  heat_no?: string;
  /** @minimum 0 */
  issued_quantity: number;
  issues: LotIssue[];
  item_type_id?: string;
  items: LotItem[];
  /** Вид: lot — партия, heat — плавка (FR-45). */
  kind?: string;
  lot_id: string;
  received_at?: string;
  registered_at?: string;
  /** Состояние партии; блок партии — ось «сдерживание» nonconformity. */
  status: LotCardStatus;
  supplier?: string;
}
