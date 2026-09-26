/**
 * «Почему вы можете / не можете» и «Запросить решение» (FR-85, FR-136, FR-146;
 * AD-15, AD-43). Интерфейс прав не вычисляет: какие действия над объектом
 * доступны, отдаёт `access.permission.list` (по объекту), объяснение —
 * `access.permission.explain` (роль, область, полномочие, клеймо, разделение
 * обязанностей и что можно сделать вместо). Обе операции уже в контракте.
 *
 * Ключи кэша — `[вид, id, PERMISSIONS, …]`: их сбрасывает и изменение самого
 * объекта, и изменение политики (shared/api/keys.ts).
 */
import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { useAllowedActions } from '@/entities/permission'
import { useAccessPermissionExplain } from '@/shared/api/generated/client'
import type { EntityKind, Explanation } from '@/shared/api/generated/model'
import { PERMISSIONS } from '@/shared/api/keys'
import { useMomentStore } from '@/shared/model/moment'

export type { Explanation }

/** id операции «Запросить решение» — оформить документ с маршрутом (FR-146, AD-43). */
export const REQUEST_DECISION_ACTION = 'documents.version.request'

/**
 * Доступные действия над объектом: множество x-ant-action id; null — ответа
 * ещё нет (тогда кнопки не включаются: «пока список прав не пришёл, ничего нельзя»).
 * В воспроизведении — только чтение (AD-21).
 */
export function useObjectActions(subject: MaybeRefOrGetter<EntityKind>, id: MaybeRefOrGetter<string>) {
  const moment = useMomentStore()
  const query = useAllowedActions(subject, id)
  const allowed = computed<Set<string> | null>(() => {
    const items = query.data.value?.data?.items
    if (!items) return null
    return new Set(items.filter((p) => !moment.isReplay || p.action_class === 'read').map((p) => p.action))
  })
  return { allowed, query }
}

/**
 * Объяснение по одному действию над объектом. Запрос идёт, только когда
 * пользователь спросил «Почему?» (`action` не пуст).
 */
export function useExplanation(action: MaybeRefOrGetter<string | null>, subject: MaybeRefOrGetter<EntityKind>, id: MaybeRefOrGetter<string>) {
  const moment = useMomentStore()
  const params = computed(() => ({ action: toValue(action) ?? '', subject: toValue(subject), id: toValue(id) }))
  return useAccessPermissionExplain(params, {
    query: {
      queryKey: computed(() => [toValue(subject), toValue(id), PERMISSIONS, 'explain', toValue(action), moment.params] as const),
      enabled: computed(() => !!toValue(action) && !!toValue(id)),
      retry: false,
    },
  })
}

/**
 * Можно ли вместо действия запросить решение: сервер назвал это в
 * `allowed_actions` объяснения или дал право на `documents.version.request`.
 */
export function canRequestDecision(explanation: Explanation | null | undefined, allowed: ReadonlySet<string> | null): boolean {
  return !!explanation?.allowed_actions.includes(REQUEST_DECISION_ACTION) || !!allowed?.has(REQUEST_DECISION_ACTION)
}
