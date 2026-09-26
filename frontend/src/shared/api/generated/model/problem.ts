/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: frontend/dev/openapi.draft.yaml (черновик до появления contracts/openapi.yaml)
 */
import type { ProblemParams } from './problemParams';

/**
 * RFC 9457 с расширениями ant (contracts/problem.schema.json).
 */
export interface Problem {
  /** urn:ant:problem:‹код› */
  type: string;
  title: string;
  status: number;
  detail?: string;
  instance?: string;
  /** Код из contracts/errors.yaml */
  code: string;
  params?: ProblemParams;
  allowed_actions?: string[];
}
