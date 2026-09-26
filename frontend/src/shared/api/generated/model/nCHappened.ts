/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { NCOperationContext } from './nCOperationContext';
import type { NCRecordRef } from './nCRecordRef';

export interface NCHappened {
  after: NCRecordRef[];
  before: NCRecordRef[];
  /** Состояние оборудования и действия исполнителя на операции. */
  during: NCRecordRef[];
  operation?: NCOperationContext;
}
