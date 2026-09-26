/**
 * Уведомления и задачи (FR-57): сводка для шапки. Список и работа с задачами —
 * виджет tasks (эпик 13).
 */
import { computed } from 'vue'
import { useNotificationsSummaryRead } from '@/shared/api/generated/client'
import { entityKeys } from '@/shared/api/keys'
import { useMomentStore } from '@/shared/model/moment'

export type { NotificationSummary } from '@/shared/api/generated/model'

export const notificationKeys = entityKeys('notification')

/** Сводка непрочитанного. */
export function useNotificationSummary() {
  const moment = useMomentStore()
  return useNotificationsSummaryRead(
    computed(() => moment.params),
    { query: { queryKey: computed(() => notificationKeys.list('summary', moment.params)) } },
  )
}
