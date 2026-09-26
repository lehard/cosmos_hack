/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ComponentStateState } from './componentStateState';

export interface ComponentState {
  /** @nullable */
  checked_at: string | null;
  /** ant/api, ant/worker, crossitem, projector, scheduler, outbox, stands, keeper, verifier, edge-agent, demo-signer, postgres… */
  component: string;
  detail?: string;
  /** @minimum 0 */
  instances: number;
  /** Копия-лидер по аренде (AD-6). */
  leader?: string;
  state: ComponentStateState;
}
