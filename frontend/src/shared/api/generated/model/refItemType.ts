/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { RefComponent } from './refComponent';
import type { RefZone } from './refZone';

export interface RefItemType {
  components: RefComponent[];
  designation: string;
  item_type_id: string;
  /** Ожидаемый носитель идентификатора (AD-16). */
  marking?: string;
  name: string;
  revision: string;
  valid_from: string;
  valid_until?: string;
  zones: RefZone[];
}
