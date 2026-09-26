/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: frontend/dev/openapi.draft.yaml (черновик до появления contracts/openapi.yaml)
 */

/**
 * Плотность — мастеру и исполнителю крупно (large), технологу — плотный инженерный вид (compact).
 */
export type Density = typeof Density[keyof typeof Density];


export const Density = {
  large: 'large',
  comfortable: 'comfortable',
  compact: 'compact',
} as const;
