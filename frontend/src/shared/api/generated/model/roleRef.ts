/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: frontend/dev/openapi.draft.yaml (черновик до появления contracts/openapi.yaml)
 */

export interface RoleRef {
  /**
     * id роли из политики (normative/policy)
     * @pattern ^[a-z][a-z0-9_]*$
     */
  id: string;
  /** Название роли из политики */
  title: string;
  /** Базовые роли (руководящие подписанты — наследники) */
  inherits?: string[];
}
