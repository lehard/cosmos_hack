/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface ProcessBpmn {
  bpmn_xml: string;
  /** H(байты XML как загружены), без канонизации XML. */
  hash: string;
  version_id: string;
}
