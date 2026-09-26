/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ConcessionKind } from './concessionKind';
import type { ConcessionStatus } from './concessionStatus';

export interface Concession {
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
  status: ConcessionStatus;
  title: string;
  /** @minimum 0 */
  used: number;
  valid_until?: string;
}
