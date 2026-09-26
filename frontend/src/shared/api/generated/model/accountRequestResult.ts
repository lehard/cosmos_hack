/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { AccountRequestResultStatus } from './accountRequestResultStatus';

export interface AccountRequestResult {
  login: string;
  /** Псевдоним, под которым администратор активирует учётную запись. */
  person_id: string;
  /** Ждёт активации администратором. */
  status: AccountRequestResultStatus;
}
