/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface CadLink {
  components: string[];
  kind: string;
  link_id: string;
  /** Вид связи в файле, если шире kind. */
  link_type?: string;
  note?: string;
  zone_id?: string;
}
