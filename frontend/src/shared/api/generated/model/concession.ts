/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ConcessionKind } from './concessionKind';
import type { ConcessionStatus } from './concessionStatus';

export interface Concession {
  /** seq последней записи потока разрешения (basis_seq отзыва, AD-39). */
  basis_seq?: number;
  concession_id: string;
  /** Документ разрешения (закрытый маршрут). */
  document_id: string;
  /** Для какого решения. */
  kind: ConcessionKind;
  /**
     * Лимит изделий.
     * @minimum 0
     */
  limit?: number;
  /** Номер по стандарту предприятия. */
  number?: string;
  /** Пункт КД/ТУ. */
  requirement_ref?: string;
  /** Область действия — изделия. */
  scope_item_ids?: string[];
  scope_range_from?: string;
  scope_range_to?: string;
  status: ConcessionStatus;
  title: string;
  /** @minimum 0 */
  used: number;
  valid_until?: string;
}
