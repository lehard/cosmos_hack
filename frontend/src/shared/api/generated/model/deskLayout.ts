/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Раскладка вкладки: single — main; main-side — main + right; queue-main-side — left + main + right; overview — top + main + right + bottom.
 */
export type DeskLayout = typeof DeskLayout[keyof typeof DeskLayout];


export const DeskLayout = {
  single: 'single',
  'main-side': 'main-side',
  'queue-main-side': 'queue-main-side',
  overview: 'overview',
} as const;
