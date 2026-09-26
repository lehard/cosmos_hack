/**
 * Справочники (эпик 19; AD-31, FR-81, FR-130): места (здание → цех → участок →
 * рабочее место, склады, изоляторы) и смены. Слой entities (FSD): ключи кэша
 * `[reference, …]` — изменение справочника в SSE перечитывает их само.
 *
 * Операции — `reference.location.list`, `reference.shift.list`
 * (contracts/openapi.yaml), сгенерированный клиент.
 */
import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { keepPreviousData, useQuery } from '@tanstack/vue-query'
import { referenceLocationList, referenceShiftList } from '@/shared/api/generated/client'
import type { RefLocation, RefLocationKind, RefShift } from '@/shared/api/generated/model'
import { entityKeys } from '@/shared/api/keys'
import type { Envelope } from '@/shared/api/response'
import { useMomentStore } from '@/shared/model/moment'

export type { RefLocation, RefLocationKind, RefShift }

export const referenceKeys = entityKeys('reference')

/** Места предприятия на момент — `reference.location.list`. */
export function useLocations() {
  const moment = useMomentStore()
  return useQuery({
    queryKey: computed(() => referenceKeys.list('locations', moment.params)),
    queryFn: async ({ signal }): Promise<Envelope<RefLocation[]>> => {
      const res = await referenceLocationList({ ...moment.params }, { signal })
      return { data: res.data.items, headers: res.headers }
    },
    placeholderData: keepPreviousData,
    retry: false,
    staleTime: 60_000,
  })
}

/** Смены места (цеха) на момент — `reference.shift.list`; место пусто — все. */
export function useShifts(locationId: MaybeRefOrGetter<string | null | undefined>) {
  const moment = useMomentStore()
  const params = computed(() => {
    const id = toValue(locationId)
    return id ? { location_id: id, ...moment.params } : { ...moment.params }
  })
  return useQuery({
    queryKey: computed(() => referenceKeys.list('shifts', params.value)),
    queryFn: async ({ signal }): Promise<Envelope<RefShift[]>> => {
      const res = await referenceShiftList(params.value, { signal })
      return { data: res.data.items, headers: res.headers }
    },
    placeholderData: keepPreviousData,
    retry: false,
  })
}

// ───────────────────────── дерево мест (чистые функции) ─────────────────────────

/** Цех места: само место, если это цех, иначе ближайший предок-цех. */
export function workshopOf(locations: readonly RefLocation[], locationId: string): RefLocation | null {
  const byId = new Map(locations.map((l) => [l.location_id, l]))
  let cur = byId.get(locationId)
  for (let guard = 0; cur && guard < 16; guard++) {
    if (cur.kind === 'workshop') return cur
    cur = cur.parent_id ? byId.get(cur.parent_id) : undefined
  }
  return null
}

/**
 * Цех по области роли (AD-15: области — пути `ent01/b1/wc/…`): цех, чья область
 * совпадает с областью роли или объемлет её. Область шире цеха — null (весь завод).
 */
export function workshopByScope(locations: readonly RefLocation[], scope: string | null | undefined): RefLocation | null {
  if (!scope) return null
  const s = scope.replace(/\/+$/, '')
  return locations.find((l) => l.kind === 'workshop' && (s === l.scope || s.startsWith(`${l.scope}/`))) ?? null
}

/** id места и всех его потомков. */
export function subtreeIds(locations: readonly RefLocation[], rootId: string): Set<string> {
  const out = new Set([rootId])
  let grew = true
  while (grew) {
    grew = false
    for (const l of locations) {
      if (l.parent_id && out.has(l.parent_id) && !out.has(l.location_id)) {
        out.add(l.location_id)
        grew = true
      }
    }
  }
  return out
}

/** Изоляторы: сначала изоляторы цеха места, затем остальные. */
export function isolatorsFor(locations: readonly RefLocation[], workshopId: string | null): RefLocation[] {
  const all = locations.filter((l) => l.kind === 'isolator')
  if (!workshopId) return all
  return [...all.filter((l) => l.parent_id === workshopId), ...all.filter((l) => l.parent_id !== workshopId)]
}

/** Цех, для которого строится стол: id и название. */
export interface WorkshopRef {
  id: string
  name: string
}

/**
 * Цех по срезу стола (AD-21): `workshop: WS-…` — явно; `scope: all` — весь
 * завод (null); иначе (`scope: own_station` или среза нет) — по области роли
 * сеанса: область цеха или уже — этот цех, область шире цеха — весь завод.
 * @param slice — срез слота стола
 * @param locations — справочник мест (может быть пуст, если не прочитался)
 * @param sessionScope — область роли сеанса
 */
export function resolveWorkshop(slice: Record<string, unknown>, locations: readonly RefLocation[], sessionScope: string | null | undefined): WorkshopRef | null {
  const explicit = typeof slice.workshop === 'string' && slice.workshop ? slice.workshop : null
  if (explicit) return { id: explicit, name: locations.find((l) => l.location_id === explicit)?.name ?? explicit }
  if (slice.scope === 'all') return null
  const ws = workshopByScope(locations, sessionScope)
  return ws ? { id: ws.location_id, name: ws.name } : null
}
