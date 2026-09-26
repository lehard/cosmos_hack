/**
 * Сотрудник (FR-78, FR-80): профиль для окна записи «Сотрудник» (Д-70) —
 * учётная запись, подразделение, роли в областях. Квалификации — в
 * entities/workplace (`useQualifications`), там же, где их проверяет назначение.
 *
 * Операция — `access.person.read` (contracts/openapi.yaml), сгенерированный клиент.
 * Ключи — `[person, ‹id›, …]`: событие SSE `person` перечитывает профиль.
 * Чтение закрыто правом `access.person.read` (у администратора и аудитора ИБ);
 * без права сервер отвечает ошибкой — окно показывает её, а не пустой профиль.
 */
import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { useQuery } from '@tanstack/vue-query'
import { accessPersonRead } from '@/shared/api/generated/client'
import type { AccessPerson, AccessPersonAccountStatus, AccessRoleGrant } from '@/shared/api/generated/model'
import { entityKeys } from '@/shared/api/keys'
import type { Envelope } from '@/shared/api/response'
import { useMomentStore } from '@/shared/model/moment'

export type { AccessPerson, AccessPersonAccountStatus, AccessRoleGrant }

export const personKeys = entityKeys('person')

/** Профиль сотрудника на момент из useMomentStore — `access.person.read`. */
export function usePerson(personId: MaybeRefOrGetter<string | null | undefined>, runId: MaybeRefOrGetter<string | undefined> = undefined) {
  const moment = useMomentStore()
  const params = computed(() => {
    const run = toValue(runId)
    return run ? { ...moment.params, run_id: run } : { ...moment.params }
  })
  return useQuery({
    queryKey: computed(() => personKeys.one(toValue(personId) ?? '', 'profile', params.value)),
    queryFn: async ({ signal }): Promise<Envelope<AccessPerson>> => {
      const res = await accessPersonRead(toValue(personId) ?? '', params.value, { signal })
      return { data: res.data, headers: res.headers }
    },
    enabled: computed(() => !!toValue(personId)),
    retry: false,
  })
}
