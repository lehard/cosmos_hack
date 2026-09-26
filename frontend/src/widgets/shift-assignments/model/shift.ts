/**
 * Смена и назначения на посты (FR-81, PRD §11.18, UJ-7): план — назначения
 * смены, факт — присутствие на посту. Исполнителей назначает мастер, только
 * допущенных по квалификации; контролёра — запрос мастера с согласованием
 * начальника ОТК (независимость ОТК). Чистые функции над ответами сервера.
 */
import type { AccessPerson, AccessRole } from '@/entities/policy'
import type { RefShift } from '@/entities/reference'
import { qualificationVerdict, type AccessAssignment, type AccessQualification, type DocumentSummary, type PostRow } from '@/entities/workplace'

/** Роль назначения на пост. */
export type AssigneeRole = 'performer' | 'quality_inspector'

/** Роли, наследующие базовую (включая её саму): «сварщик» — исполнитель. */
export function rolesInheriting(roles: readonly AccessRole[] | null | undefined, base: string): Set<string> {
  const out = new Set([base])
  let grew = true
  while (grew) {
    grew = false
    for (const r of roles ?? []) {
      if (!out.has(r.id) && r.inherits.some((i) => out.has(i))) {
        out.add(r.id)
        grew = true
      }
    }
  }
  return out
}

/** Область роли покрывает цех или лежит внутри него (AD-15: пути областей). */
export const scopeTouches = (grant: string, workshopScope: string | null): boolean =>
  !workshopScope || grant === workshopScope || workshopScope.startsWith(`${grant}/`) || grant.startsWith(`${workshopScope}/`)

/** Кандидат на пост. */
export interface Candidate {
  person: AccessPerson
  /** Квалификация «на глаз»: истекла — назначить нельзя (решает сервер). */
  verdict: 'ok' | 'expiring' | 'expired' | null
}

/**
 * Кандидаты на пост: сотрудники с ролью нужного вида в области цеха.
 * @param persons — сотрудники с ролями
 * @param roleIds — роли, подходящие для назначения (с наследниками)
 * @param workshopScope — область цеха; null — весь завод
 * @param qualifications — квалификации всех сотрудников
 */
export function candidates(
  persons: readonly AccessPerson[] | null | undefined,
  roleIds: ReadonlySet<string>,
  workshopScope: string | null,
  qualifications: readonly AccessQualification[] | null | undefined,
): Candidate[] {
  return (persons ?? [])
    .filter((p) => p.roles.some((r) => roleIds.has(r.role_id) && scopeTouches(r.scope, workshopScope)))
    .map((person) => ({ person, verdict: qualificationVerdict((qualifications ?? []).filter((q) => q.person_id === person.person_id)) }))
}

/** Строка поста в смене: план (назначения) и факт (присутствие). */
export interface ShiftPostRow {
  post: PostRow
  performers: AccessAssignment[]
  inspectors: AccessAssignment[]
}

/** Посты смены с назначениями. */
export function shiftRows(posts: readonly PostRow[], assignments: readonly AccessAssignment[] | null | undefined): ShiftPostRow[] {
  return posts.map((post) => {
    const own = (assignments ?? []).filter((a) => a.workplace_id === post.workplace_id)
    return { post, performers: own.filter((a) => a.assignee_role === 'performer'), inspectors: own.filter((a) => a.assignee_role === 'quality_inspector') }
  })
}

/** Документ — согласование назначения контролёра (по шаблону). */
export const isControllerApproval = (doc: Pick<DocumentSummary, 'template'>, template: string): boolean =>
  doc.template === template || doc.template.split('@')[0] === template.split('@')[0]

/** Подпись смены: название или время начала — конца по часам завода. */
export function shiftLabel(s: RefShift, time: (iso: string) => string): string {
  return s.name ? `${s.name} · ${time(s.starts_at)}–${time(s.ends_at)}` : `${s.shift_id} · ${time(s.starts_at)}–${time(s.ends_at)}`
}

/** Решение, которое оформляет документ запроса: кто и в какую смену (поле `decision`). */
export const controllerDecision = (personId: string, shiftId: string): string => `quality_inspector:${personId}@${shiftId}`
