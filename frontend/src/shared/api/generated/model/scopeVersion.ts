/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { Reason } from './reason';
import type { ScopeBreakdown } from './scopeBreakdown';
import type { ScopeVersionChange } from './scopeVersionChange';

export interface ScopeVersion {
  /**
     * Псевдоним; null — правило системы.
     * @nullable
     */
  author: string | null;
  breakdown: ScopeBreakdown;
  /** incident.scope.computed / expanded / narrowed. */
  change: ScopeVersionChange;
  evidence_event_ids: string[];
  reason?: Reason;
  recorded_at: string;
  /** @minimum 1 */
  scope_version: number;
  /** @minimum 0 */
  size: number;
}
