/**
 * Уведомления и задачи (FR-57): сводка для шапки по видам, «требует вашего
 * внимания» и лента тревог (FR-8), тексты уведомлений модуля notifications.
 * Список и отметка задач — entities/task (эпик 13).
 */
import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { keepPreviousData, useQuery } from '@tanstack/vue-query'
import { notificationsAlertList, notificationsAttentionList, useNotificationsSummaryRead } from '@/shared/api/generated/client'
import type { AlertEntry, AlertEntryKind, AttentionEntry } from '@/shared/api/generated/model'
import { entityKeys } from '@/shared/api/keys'
import type { Envelope } from '@/shared/api/response'
import type { DrillRef } from '@/shared/model/drill'
import { useMomentStore } from '@/shared/model/moment'
import { codeToKey } from '@/shared/i18n'
import { formatMinutes } from '@/shared/lib/duration'

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

// ─────────────── тексты уведомлений модуля notifications (эпик 24) ───────────────
// Уведомление несёт ключ текста `text_key` и параметры (`task.notification.sent`):
// `notifications.overdue.‹основание срока›` — тревога о просрочке,
// `notifications.info.moved_to_isolator` — информация. Сегменты ключа в
// snake_case, как коды контракта; тексты интерфейса — в camelCase (codeToKey).

/** Основания сроков (`basis` в obligation.due.set) — у каждого свой текст тревоги. */
export const OBLIGATION_BASES = ['isolation', 'isolation_move', 'nonconformity', 'presentation', 'incident_scope', 'recheck', 'bpmn_timer'] as const
export type ObligationBasis = (typeof OBLIGATION_BASES)[number]

/** Параметры текста, которые — время (RFC 3339): показываются по часам завода. */
const TIME_PARAMS = new Set(['first_due_at', 'due_at'])

/** Функции текстов vue-i18n, нужные уведомлению. */
export interface NoticeTexts {
  t: (key: string, params?: Record<string, unknown>) => string
  te: (key: string) => boolean
  d: (value: Date, format: string) => string
}

/**
 * Текст уведомления по ключу контракта и параметрам. Нет текста — ключ как
 * есть, а не угаданная фраза (NFR-UI-4).
 * @param x — функции текстов
 * @param textKey — `text_key` уведомления (`notifications.overdue.isolation_move`)
 * @param params — параметры уведомления
 */
export function noticeText(x: NoticeTexts, textKey: string, params: Record<string, string> = {}): string {
  const key = codeToKey(textKey)
  if (!x.te(key)) return textKey
  const shown: Record<string, string> = {}
  for (const [k, v] of Object.entries(params)) {
    const at = TIME_PARAMS.has(k) ? new Date(v) : null
    shown[k] = at && !Number.isNaN(at.getTime()) ? x.d(at, 'dateTime') : v
  }
  return x.t(key, shown)
}

// ─────────────── тексты тревог и «требует внимания» (FR-8, FR-57) ───────────────

/** Функции текстов vue-i18n для тревог. */
export interface AlertTexts {
  t: (key: string, params?: Record<string, unknown>, plural?: number) => string
  te: (key: string) => boolean
}

const DASH = '—'

/** «Просрочено на 37 мин — стоят 18 изделий, 2 операции» (цена задержки, FR-57). */
export function costOfDelayText(x: AlertTexts, e: { target?: string; overdue_minutes?: number; items?: number; operations?: number }): string {
  const items = e.items ?? 0
  const operations = e.operations ?? 0
  return x.t('liveMap.attention.overdueDecision', {
    target: e.target ?? DASH,
    overdue: formatMinutes(x.t, e.overdue_minutes ?? 0),
    items: x.t('plural.items', { n: items }, items),
    operations: x.t('plural.operations', { n: operations }, operations),
  })
}

/** Текст тревоги по виду; неизвестный вид аномалии — UNKNOWN(код), а не похожий текст (NFR-UI-4). */
export function alertText(x: AlertTexts, a: AlertEntry): string {
  switch (a.kind) {
    case 'overdue_isolation':
      return x.t('liveMap.alerts.overdueIsolation', { item: a.item ?? DASH })
    case 'gate_overdue':
      return x.t('liveMap.alerts.gateOverdue', { gate: a.gate ?? DASH })
    case 'not_moved_to_isolator':
      return x.t('liveMap.alerts.notMovedToIsolator', { item: a.item ?? DASH })
    case 'anomaly': {
      const key = a.anomaly ? `liveMap.anomalies.${codeToKey(a.anomaly)}` : ''
      const what = key && x.te(key) ? x.t(key, { threshold: DASH }) : `UNKNOWN(${a.anomaly ?? ''})`
      return x.t('liveMap.alerts.anomaly', { node: a.node ?? DASH, what })
    }
    case 'escalation':
      return `${x.t('common.notifications.escalation')}: ${costOfDelayText(x, a)}`
    case 'integrity_violation':
      return x.t('liveMap.alerts.integrityViolation')
  }
  return `UNKNOWN(${String((a as { kind: unknown }).kind)})`
}

/** Текст строки «требует вашего внимания». */
export function attentionText(x: AlertTexts, e: AttentionEntry): string {
  if (e.kind === 'overdue_decision') return costOfDelayText(x, e)
  return x.t(e.kind === 'unverified_measures' ? 'liveMap.attention.unverifiedMeasures' : 'liveMap.attention.temporaryMeasures', { n: e.n ?? 0 })
}
