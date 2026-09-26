/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { KeyViewAlgorithm } from './keyViewAlgorithm';
import type { KeyViewProfileId } from './keyViewProfileId';
import type { KeyViewProvenanceClass } from './keyViewProvenanceClass';
import type { KeyViewStatus } from './keyViewStatus';
import type { KeyViewSubjectKind } from './keyViewSubjectKind';

export interface KeyView {
  algorithm: KeyViewAlgorithm;
  /** «Скомпрометирован с» — подписи позже под сомнением (AD-11). */
  compromised_since?: string;
  /** Отпечаток открытого ключа; в сводке акта — «сверьте отпечаток» (AD-11). */
  fingerprint: string;
  /** key_id@версия. */
  key_ref: string;
  /** Допустимые классы пакетов (AD-10). */
  payload_classes: string[];
  profile_id: KeyViewProfileId;
  /** Класс доверия ключа (наследуется от регистрирующих подписей, AD-11). */
  provenance_class: KeyViewProvenanceClass;
  /** Документ акта с маршрутом подписей (AD-13). */
  registration_document_id?: string;
  /** Акт регистрации key.registration.recorded. */
  registration_event_id: string;
  revocation_event_id?: string;
  /** Ключ, который заменяет (ротация). */
  rotates?: string;
  /** unavailable — ключ недоступен: «не проверяемо», никогда не «валидно» (AD-32). */
  status: KeyViewStatus;
  subject_id: string;
  subject_kind: KeyViewSubjectKind;
  valid_from: string;
  valid_until?: string;
}
