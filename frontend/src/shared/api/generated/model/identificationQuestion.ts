/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { IdentificationQuestionCause } from './identificationQuestionCause';

export interface IdentificationQuestion {
  at: string;
  basis: string[];
  candidates?: string[];
  cause: IdentificationQuestionCause;
  closed_by?: string;
  open: boolean;
  /** Запись item.identification.questioned — её снимает подтверждение. */
  questioned_event_id: string;
  /** Событие с неоднозначной привязкой. */
  subject_event_id?: string;
}
