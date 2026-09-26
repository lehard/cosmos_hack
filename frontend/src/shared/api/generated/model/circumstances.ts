/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { CausalWindow } from './causalWindow';
import type { CircumstanceRecord } from './circumstanceRecord';
import type { CircumstancesMissingInformationItem } from './circumstancesMissingInformationItem';
import type { OperationSpan } from './operationSpan';

export interface Circumstances {
  /** seq, на котором построен ответ (для basis_seq команд, AD-39). */
  basis_seq: number;
  conclusion_is_categorical: boolean;
  missing_information: CircumstancesMissingInformationItem[];
  nc_id: string;
  /** Нет — выполнение операции не установлено. */
  operation?: OperationSpan;
  records: CircumstanceRecord[];
  /** Нет — окно определить нельзя. */
  window?: CausalWindow;
}
