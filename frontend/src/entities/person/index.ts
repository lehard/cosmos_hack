/**
 * Сотрудник (FR-78, FR-80, FR-81): карточка для окна записи «Сотрудник» (Д-70,
 * UI-16) — имя, подразделение, роли в областях, квалификации со сроками и
 * текущие посты. Логина и состояния учётной записи в карточке нет — их видит
 * только администратор в своём разделе.
 *
 * Операция — `access.person.card` (contracts/openapi.yaml), сгенерированный клиент.
 * Ключи — `[person, ‹id›, …]`: событие SSE `person` перечитывает карточку.
 * Чтение выдано руководителю производства, мастеру и начальнику ОТК; без права
 * сервер отвечает ошибкой — окно показывает её, а не пустую карточку.
 */
import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { useQuery } from '@tanstack/vue-query'
import { accessPersonCard } from '@/shared/api/generated/client'
import type { AccessQualification, AccessRoleGrant, PersonCard, PersonPost } from '@/shared/api/generated/model'
import { entityKeys } from '@/shared/api/keys'
import type { Envelope } from '@/shared/api/response'
import { useMomentStore } from '@/shared/model/moment'

export type { AccessQualification, AccessRoleGrant, PersonCard, PersonPost }

export const personKeys = entityKeys('person')

/** Карточка сотрудника на момент из useMomentStore — `access.person.card`. */
export function usePersonCard(personId: MaybeRefOrGetter<string | null | undefined>, runId: MaybeRefOrGetter<string | undefined> = undefined) {
  const moment = useMomentStore()
  const params = computed(() => {
    const run = toValue(runId)
    return run ? { ...moment.params, run_id: run } : { ...moment.params }
  })
  return useQuery({
    queryKey: computed(() => personKeys.one(toValue(personId) ?? '', 'card', params.value)),
    queryFn: async ({ signal }): Promise<Envelope<PersonCard>> => {
      const res = await accessPersonCard(toValue(personId) ?? '', params.value, { signal })
      return { data: res.data, headers: res.headers }
    },
    enabled: computed(() => !!toValue(personId)),
    retry: false,
  })
}
