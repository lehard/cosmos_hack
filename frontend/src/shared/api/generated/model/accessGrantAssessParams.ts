/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { AccessGrantAssessKind } from './accessGrantAssessKind';

export type AccessGrantAssessParams = {
/**
 * Кому выдаётся.
 * @maxLength 64
 */
person_id: string;
kind: AccessGrantAssessKind;
/**
 * Роль, полномочие или id клейма.
 * @maxLength 64
 */
subject_id: string;
/**
 * Вид контроля клейма.
 * @maxLength 64
 */
inspection_kind?: string;
/**
 * Область; пусто — всё предприятие.
 * @maxLength 256
 */
scope?: string;
};
