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

/**
 * Черновик активации учётной записи (FR-128, эпик 08): логин; начальный пароль —
 * обязателен, если сотрудник не подавал заявку (учётной записи нет), по заявке —
 * необязателен (сотрудник задал свой); начальная роль и её область (пусто — всё
 * предприятие).
 */
export interface ActivationDraft {
  login: string
  password: string
  initial_role_id: string
  scope: string
}

/** Минимальная длина пароля (как у заявки на регистрацию, contracts/openapi.yaml). */
export const PASSWORD_MIN = 8

/** Активация готова к отправке. */
export function activationReady(d: ActivationDraft, hasRequest: boolean): boolean {
  if (!d.login.trim()) return false
  if (!hasRequest && d.password.length < PASSWORD_MIN) return false
  if (hasRequest && d.password && d.password.length < PASSWORD_MIN) return false
  return true
}
