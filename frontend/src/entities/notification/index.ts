/**
 * Уведомления и задачи (FR-57): сводка для шапки. Список и работа с задачами —
 * виджет tasks (эпик 13).
 */
import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { useQuery } from '@tanstack/vue-query'
import { useNotificationsSummaryRead } from '@/shared/api/generated/client'
import { entityKeys } from '@/shared/api/keys'
import { pendingOperation, type Envelope } from '@/shared/api/pending'
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
// Операций ещё нет в contracts/openapi.yaml (эпик 02): запросы на заглушке
// `api.not_implemented`, форма данных — предложение для контракта. Сроки и
// эскалации порождает модуль notifications (AD-40), здесь только показ.

/** Строка «требует вашего внимания» (FR-8). */
export type AttentionEntry =
  | {
      kind: 'overdue_decision'
      entry_id: string
      /** Что ждёт решения: изделие, несоответствие, точка предъявления — подпись. */
      target: string
      /** На сколько просрочено, минуты. */
      overdue_minutes: number
      /** Цена задержки: сколько изделий стоит и сколько операций (FR-8). */
      items: number
      operations: number
      ref?: DrillRef
    }
  | { kind: 'unverified_measures' | 'temporary_measures'; entry_id: string; n: number; ref?: DrillRef }

/** Вид тревоги ленты (FR-8). */
export type AlertKind = 'overdue_isolation' | 'gate_overdue' | 'not_moved_to_isolator' | 'anomaly' | 'escalation' | 'integrity_violation'

/** Тревога ленты: параметры текста — по виду. */
export interface AlertEntry {
  alert_id: string
  at: string
  kind: AlertKind
  /** Изделие (просроченная изоляция, не перемещено в изолятор). */
  item?: string
  /** Точка предъявления. */
  gate?: string
  /** Узел и вид аномалии (коды entities/live-map AnomalyKind). */
  node?: string
  anomaly?: string
  /** Эскалация: цель и цена задержки. */
  target?: string
  overdue_minutes?: number
  items?: number
  operations?: number
  /** Объект тревоги (FR-7: по тревоге → объект тревоги). */
  ref?: DrillRef
}

/** Блок «требует вашего внимания». Ожидаемая операция — `notifications.attention.list`. */
export function useAttention(params: MaybeRefOrGetter<{ run_id?: string }>) {
  const moment = useMomentStore()
  return useQuery({
    queryKey: computed(() => notificationKeys.list('attention', toValue(params), moment.params)),
    queryFn: () => pendingOperation<Envelope<AttentionEntry[]>>('notifications.attention.list')(),
    placeholderData: (prev: Envelope<AttentionEntry[]> | undefined) => prev,
  })
}

/** Лента тревог. Ожидаемая операция — `notifications.alert.list`. */
export function useAlerts(params: MaybeRefOrGetter<{ run_id?: string }>) {
  const moment = useMomentStore()
  return useQuery({
    queryKey: computed(() => notificationKeys.list('alerts', toValue(params), moment.params)),
    queryFn: () => pendingOperation<Envelope<AlertEntry[]>>('notifications.alert.list')(),
    placeholderData: (prev: Envelope<AlertEntry[]> | undefined) => prev,
  })
}
