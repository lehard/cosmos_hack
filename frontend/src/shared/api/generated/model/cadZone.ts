/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface CadZone {
  /** Тип изделия, на котором лежит зона. */
  item_type_id?: string;
  /** weld_section, joint, hole, surface, cavity, groove, other. */
  kind: string;
  link_id: string;
  name: string;
  zone_id: string;
}
