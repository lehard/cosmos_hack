/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { IntegrationEntry } from './integrationEntry';
import type { IntegrationListProfile } from './integrationListProfile';

export interface IntegrationList {
  items: IntegrationEntry[];
  /** В prod стенд запрещён. */
  profile: IntegrationListProfile;
}
