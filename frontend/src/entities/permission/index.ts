/**
 * Разрешённые действия (AD-15 «Для фронтенда», FR-85, FR-146): плоский список для
 * @casl/vue и допустимые действия по объекту. Оба запроса помечены PERMISSIONS —
 * их сбрасывает любое изменение политики из SSE; запрос по объекту сбрасывается
 * и изменением самого объекта (ключ начинается с [вид, id]).
 */
import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { useAccessPermissionList } from '@/shared/api/generated/client'
import type { EntityKind } from '@/shared/api/generated/model'
import { LIST, PERMISSIONS } from '@/shared/api/keys'
import { useMomentStore } from '@/shared/model/moment'

export type { Permission, PermissionList } from '@/shared/api/generated/model'

/** Плоский список прав пользователя на текущий момент. */
export function usePermissions() {
  const moment = useMomentStore()
  const params = computed(() => moment.params)
  return useAccessPermissionList(params, {
    query: { queryKey: computed(() => ['policy', LIST, PERMISSIONS, moment.params] as const), staleTime: 30_000 },
  })
}

/**
 * Допустимые действия над объектом — для «Почему вы можете / не можете» и
 * «Запросить решение» (FR-146).
 */
export function useAllowedActions(subject: MaybeRefOrGetter<EntityKind>, id: MaybeRefOrGetter<string>) {
  const moment = useMomentStore()
  const params = computed(() => ({ subject: toValue(subject), id: toValue(id), ...moment.params }))
  return useAccessPermissionList(params, {
    query: { queryKey: computed(() => [toValue(subject), toValue(id), PERMISSIONS, moment.params] as const) },
  })
}
