/**
 * Уведомления и задачи (FR-57): сводка для шапки. Список и работа с задачами —
 * виджет tasks (эпик 13).
 */
import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { keepPreviousData, useQuery } from '@tanstack/vue-query'
import { notificationsAlertList, notificationsAttentionList, useNotificationsSummaryRead } from '@/shared/api/generated/client'
import type { AlertEntry, AlertEntryKind, AttentionEntry } from '@/shared/api/generated/model'
import { entityKeys } from '@/shared/api/keys'
import type { Envelope } from '@/shared/api/response'
import type { DrillRef } from '@/shared/model/drill'
import { useMomentStore } from '@/shared/model/moment'

export type { DrillRef }

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

// ──────────── «Требует вашего внимания» и лента тревог (FR-8, эпик 10) ────────────
// Операции — `notifications.attention.list` и `notifications.alert.list`.
// Сроки и эскалации порождает модуль notifications (AD-40), здесь только показ.

export type { AlertEntry, AttentionEntry }
export type AlertKind = AlertEntryKind

/** Блок «требует вашего внимания» — `notifications.attention.list`. */
export function useAttention(params: MaybeRefOrGetter<{ run_id?: string }>) {
  const moment = useMomentStore()
  return useQuery({
    queryKey: computed(() => notificationKeys.list('attention', toValue(params), moment.params)),
    queryFn: async ({ signal }): Promise<Envelope<AttentionEntry[]>> => {
      const res = await notificationsAttentionList({ ...toValue(params), ...moment.params }, { signal })
      return { data: res.data.items, headers: res.headers }
    },
    placeholderData: keepPreviousData,
  })
}

/** Лента тревог — `notifications.alert.list`, первая страница (новые сверху). */
export function useAlerts(params: MaybeRefOrGetter<{ run_id?: string }>) {
  const moment = useMomentStore()
  return useQuery({
    queryKey: computed(() => notificationKeys.list('alerts', toValue(params), moment.params)),
    queryFn: async ({ signal }): Promise<Envelope<AlertEntry[]>> => {
      const res = await notificationsAlertList({ ...toValue(params), ...moment.params }, { signal })
      return { data: res.data.items, headers: res.headers }
    },
    placeholderData: keepPreviousData,
  })
}
