/**
 * Формы виджета «Доступ»: разделы и черновик выдачи прав (FR-78, FR-145).
 */
import type { GrantPolicyKind } from '@/entities/policy'

/** Раздел виджета. */
export type AccessSection = 'persons' | 'roles' | 'stamps' | 'grant'

/** Разделы в порядке кнопок. */
export const ACCESS_SECTIONS: readonly AccessSection[] = ['persons', 'roles', 'stamps', 'grant']

/** Черновик выдачи из формы: даты — ГГГГ-ММ-ДД. */
export interface GrantDraft {
  kind: GrantPolicyKind
  person_id: string
  subject_id: string
  scope: string
  valid_from: string
  valid_until: string
  order_ref: string
  inspection_kind: string
  document_id: string
}

/** Черновик готов к отправке: сотрудник, что выдать, область, дата; клейму — приказ и вид контроля. */
export const grantReady = (d: GrantDraft): boolean =>
  !!d.person_id.trim() && !!d.subject_id.trim() && !!d.scope.trim() && !!d.valid_from && (d.kind !== 'stamp' || (!!d.order_ref.trim() && !!d.inspection_kind.trim()))
