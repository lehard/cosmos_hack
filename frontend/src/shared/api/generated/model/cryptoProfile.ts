/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { CryptoProfileProfileId } from './cryptoProfileProfileId';

export interface CryptoProfile {
  algorithm?: string;
  /** Составные профили гибрида. */
  components?: string[];
  /** С какой записи действует (key.profile.registered). */
  effective_from_seq?: number;
  /** Объекты, для которых профиль обязателен. */
  object_classes: string[];
  /** Промышленный — через сертифицированное СКЗИ (MVP — не СКЗИ). */
  production: boolean;
  profile_id: CryptoProfileProfileId;
  /** @minimum 1 */
  signatures_required: number;
  title: string;
}
