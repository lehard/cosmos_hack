/**
 * Доступ (эпик 14; FR-78, FR-79, FR-128, FR-145; AD-11, AD-15, AD-28): сотрудники
 * и их роли в области, роли и полномочия политики, цифровые клейма, история
 * выдачи прав, выдача и отзыв со второй подписью независимой стороны.
 *
 * Операции — `access.person.list`, `access.role.list`, `access.stamp.list`,
 * `access.grant.list`, `access.policy.grant|revoke` (contracts/openapi.yaml),
 * сгенерированный клиент. Ключи — `[policy, …]`: изменение политики в SSE
 * перечитывает и списки, и права на экране (shared/api/keys.ts, PERMISSIONS).
 */
import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { accessGrantList, accessPersonList, accessPolicyGrant, accessPolicyRevoke, accessRoleList, accessStampList } from '@/shared/api/generated/client'
import type {
  AccessGrantEntry,
  AccessGrantEntryAction,
  AccessGrantEntryKind,
  AccessPerson,
  AccessPersonAccountStatus,
  AccessRole,
  AccessRoleList,
  AccessStamp,
  GrantPolicy,
  GrantPolicyKind,
  RevokePolicy,
} from '@/shared/api/generated/model'
import { entityKeys } from '@/shared/api/keys'
import type { ApiError } from '@/shared/api/problem'
import { useMomentStore } from '@/shared/model/moment'

export type {
  AccessGrantEntry,
  AccessGrantEntryAction,
  AccessGrantEntryKind,
  AccessPerson,
  AccessPersonAccountStatus,
  AccessRole,
  AccessRoleList,
  AccessStamp,
  GrantPolicy,
  GrantPolicyKind,
  RevokePolicy,
}

export const policyKeys = entityKeys('policy')

/** Параметры момента для чтения политики. */
function useMomentParams() {
  const moment = useMomentStore()
  return computed(() => ({ ...moment.params }))
}

/** Сотрудники с ролями в области (`access.person.list`). */
export function usePersons() {
  const params = useMomentParams()
  return useQuery({
    queryKey: computed(() => policyKeys.list('persons', params.value)),
    queryFn: ({ signal }) => accessPersonList({ limit: 500, ...params.value }, { signal }),
    retry: false,
    placeholderData: keepPreviousData,
  })
}

/** Роли, полномочия и виды клейм политики (`access.role.list`). */
export function useRoles() {
  const params = useMomentParams()
  return useQuery({
    queryKey: computed(() => policyKeys.list('roles', params.value)),
    queryFn: ({ signal }) => accessRoleList(params.value, { signal }),
    retry: false,
    placeholderData: keepPreviousData,
  })
}

/** Реестр цифровых клейм (`access.stamp.list`, FR-145). */
export function useStamps() {
  const params = useMomentParams()
  return useQuery({
    queryKey: computed(() => policyKeys.list('stamps', params.value)),
    queryFn: ({ signal }) => accessStampList(params.value, { signal }),
    retry: false,
    placeholderData: keepPreviousData,
  })
}

/** История выдачи прав (`access.grant.list`, FR-79); сотрудник пуст — все. */
export function useGrantHistory(personId: MaybeRefOrGetter<string | null> = null) {
  const params = useMomentParams()
  const full = computed(() => {
    const p = toValue(personId)
    return { limit: 500, ...(p ? { person_id: p } : {}), ...params.value }
  })
  return useQuery({
    queryKey: computed(() => policyKeys.list('grants', full.value)),
    queryFn: ({ signal }) => accessGrantList(full.value, { signal }),
    retry: false,
    placeholderData: keepPreviousData,
  })
}

/** Команда политики: выдать или отозвать роль, полномочие, клеймо. */
export type PolicyCommand = { kind: 'grant'; body: GrantPolicy } | { kind: 'revoke'; body: RevokePolicy }

function send(cmd: PolicyCommand) {
  return cmd.kind === 'grant' ? accessPolicyGrant(cmd.body) : accessPolicyRevoke(cmd.body)
}

/**
 * Выдача и отзыв прав (FR-78, AD-11): привилегированная выдача исполняется
 * только после второй подписи независимой стороны — сервер вернёт квитанцию с
 * номером критического действия (AD-28). После успеха перечитывается вся политика.
 */
export function usePolicyCommand() {
  const queryClient = useQueryClient()
  return useMutation<Awaited<ReturnType<typeof send>>, ApiError, PolicyCommand>({
    mutationFn: send,
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: policyKeys.all }),
  })
}
