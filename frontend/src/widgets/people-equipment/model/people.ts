/**
 * Люди и оборудование участка (PRD §3a «Мастер участка — Люди и оборудование»,
 * FR-6, FR-17, FR-80, FR-83, FR-84, FR-149): назначен / на месте / ключ,
 * допуск и квалификация; состояние оборудования, предупреждения, остановки,
 * срок поверки. Чистые функции над ответами сервера, без пересчёта.
 */
import type { EquipmentState, RefEquipment } from '@/entities/equipment'
import type { RefLocation } from '@/entities/reference'
import { subtreeIds } from '@/entities/reference'
import type { AccessAssignment, AccessQualification, PostRow } from '@/entities/workplace'

/** Человек на посту. */
export interface PersonRow {
  post: PostRow
  /** Назначение в смене (допуск, квалификация на дату смены), если известно. */
  assignment: AccessAssignment | null
  /** Квалификации назначенного. */
  qualifications: AccessQualification[]
}

/** Оборудование участка: состояние по журналам и запись справочника. */
export interface EquipmentRow {
  id: string
  title: string
  state: EquipmentState | null
  registry: RefEquipment | null
}

/** Люди на постах участка. */
export function buildPeople(
  posts: readonly PostRow[],
  assignments: readonly AccessAssignment[] | null | undefined,
  qualifications: readonly AccessQualification[] | null | undefined,
): PersonRow[] {
  return posts.map((post) => {
    const person = post.assigned?.person_id
    return {
      post,
      assignment: (person && assignments?.find((a) => a.workplace_id === post.workplace_id && a.person_id === person)) || null,
      qualifications: person ? (qualifications ?? []).filter((q) => q.person_id === person) : [],
    }
  })
}

/**
 * Оборудование участка: посты участка (`station_id`) и места цеха по
 * справочнику. Цех неизвестен — всё оборудование.
 */
export function buildEquipment(
  states: readonly EquipmentState[] | null | undefined,
  registry: readonly RefEquipment[] | null | undefined,
  scope: { postIds: ReadonlySet<string> | null; locations: readonly RefLocation[] | null; workshopId: string | null },
): EquipmentRow[] {
  const subtree = scope.workshopId && scope.locations?.length ? subtreeIds(scope.locations, scope.workshopId) : null
  const filtered = !!scope.workshopId
  const inScope = (stationId: string | undefined, locationId: string | undefined) =>
    !filtered || (!!stationId && !!scope.postIds?.has(stationId)) || (!!subtree && ((!!locationId && subtree.has(locationId)) || (!!stationId && subtree.has(stationId))))
  const rows = new Map<string, EquipmentRow>()
  for (const s of states ?? []) {
    const reg = registry?.find((r) => r.equipment_id === s.equipment_id) ?? null
    if (inScope(s.station_id, reg?.location_id)) rows.set(s.equipment_id, { id: s.equipment_id, title: s.title, state: s, registry: reg })
  }
  for (const r of registry ?? []) {
    if (!rows.has(r.equipment_id) && inScope(undefined, r.location_id)) rows.set(r.equipment_id, { id: r.equipment_id, title: r.name, state: null, registry: r })
  }
  return [...rows.values()]
}

/** Поверка для экрана (FR-17, FR-149). */
export type VerificationView =
  | { kind: 'valid'; until: string | null }
  | { kind: 'expiring'; until: string | null }
  | { kind: 'expired'; until: string | null }
  | { kind: 'unusable'; reason: string }
  | { kind: 'not_required' }
  | { kind: 'unknown' }

/**
 * Поверка: справочник — источник правды о пригодности (FR-17); без записи в
 * справочнике — состояние по журналам оборудования; ничего нет — неизвестно.
 */
export function verificationOf(row: EquipmentRow): VerificationView {
  const r = row.registry
  if (r) {
    if (!r.usable) return r.unusable_reason === 'verification_expired' ? { kind: 'expired', until: r.verified_until ?? null } : { kind: 'unusable', reason: r.unusable_reason ?? 'unknown' }
    if (r.verified_until) return { kind: 'valid', until: r.verified_until }
    if (!r.is_measuring_instrument) return { kind: 'not_required' }
  }
  const v = row.state?.verification
  if (v && v.status !== 'unknown') return { kind: v.status, until: v.valid_till ?? null }
  return { kind: 'unknown' }
}

/** Есть ли в данных «оценка невозможна»: присутствие неизвестно или оборудование без данных. */
export function peopleEquipmentState(people: readonly PersonRow[], equipment: readonly EquipmentRow[]): 'normal' | 'unable_to_assess' {
  return people.some((p) => p.post.presence === 'unknown') || equipment.some((e) => e.state?.condition === 'unknown') ? 'unable_to_assess' : 'normal'
}
