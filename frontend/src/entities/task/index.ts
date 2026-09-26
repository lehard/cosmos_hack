/**
 * Задачи и запросы решения (FR-57, эпик 13): адресные задачи пользователя —
 * владелец, срок, подтверждение; порождает их только модуль notifications
 * (AD-40), интерфейс показывает и отмечает.
 *
 * Операции — `notifications.task.list` и `notifications.task.acknowledge`
 * (contracts/openapi.yaml), сгенерированный клиент. Ключи — `[task, …]` по
 * соглашению shared/api/keys.ts: SSE `task` перечитывает список сам.
 */
import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { journalHeadRead, notificationsTaskAcknowledge, notificationsTaskList } from '@/shared/api/generated/client'
import type { AcknowledgeTask, AcknowledgeTaskOutcome, TaskEntry, TaskEntryKind, TaskEntryState } from '@/shared/api/generated/model'
import { entityKeys } from '@/shared/api/keys'
import type { ApiError } from '@/shared/api/problem'
import type { Envelope } from '@/shared/api/response'
import { useMomentStore } from '@/shared/model/moment'

export type { AcknowledgeTask, AcknowledgeTaskOutcome, TaskEntry, TaskEntryKind, TaskEntryState }

export const taskKeys = entityKeys('task')

/**
 * Вид уведомления (FR-57): информация / тревога / задача / запрос решения.
 * Эскалация — запрос решения, срок которого прошёл (цена задержки, FR-8).
 */
export type NotificationKind = 'info' | 'alarm' | 'task' | 'decision_request'

/** Виды задач, которые — запрос решения (как `decisionKind` модуля notifications). */
const DECISION_KINDS: ReadonlySet<TaskEntryKind> = new Set<TaskEntryKind>(['decision_required', 'review_after_new_data', 'protection_basis_changed'])

/** Задача или запрос решения — по виду задачи. */
export const taskNotificationKind = (kind: TaskEntryKind): 'task' | 'decision_request' => (DECISION_KINDS.has(kind) ? 'decision_request' : 'task')

/** Задача ещё ждёт действия. */
export const isOpenTask = (t: Pick<TaskEntry, 'state'>): boolean => t.state === 'open'

/**
 * Физическое перемещение в изолятор (FR-55): такую задачу закрывает не отметка,
 * а подтверждённая приёмка в изоляторе — основание задачи снимается само.
 */
export const isIsolatorMoveTask = (t: Pick<TaskEntry, 'kind' | 'ref'>): boolean => t.kind === 'isolate_move' && t.ref?.entity === 'item'

/** Фильтр задач (без момента). */
export interface TaskFilter {
  state?: TaskEntryState
  location_id?: string
  run_id?: string
}

/** Задачи и seq, на котором их видел клиент. */
export interface TaskInboxData {
  items: TaskEntry[]
  /**
   * `basis_seq` отметки (AD-39). Ответ списка задач своего `basis_seq` не несёт,
   * поэтому берётся голова журнала, прочитанная до списка: всё, что было в
   * журнале к ней, список уже учёл — отметка по устаревшему виду получит 409.
   */
  basis_seq: number
}

/**
 * Задачи и запросы решения пользователя на момент — `notifications.task.list`
 * (первые 200, новые сверху; адресность решает сервер) и голова журнала
 * `journal.head.read` для `basis_seq` отметки.
 */
export function useTasks(filter: MaybeRefOrGetter<TaskFilter> = {}) {
  const moment = useMomentStore()
  return useQuery({
    queryKey: computed(() => taskKeys.list('inbox', toValue(filter), moment.params)),
    queryFn: async ({ signal }): Promise<Envelope<TaskInboxData>> => {
      const run = toValue(filter).run_id
      const head = await journalHeadRead(run ? { run_id: run } : {}, { signal }).catch(() => null)
      const res = await notificationsTaskList({ limit: 200, ...toValue(filter), ...moment.params }, { signal })
      return { data: { items: res.data.items, basis_seq: head?.data.seq ?? 0 }, headers: res.headers }
    },
    placeholderData: keepPreviousData,
    retry: false,
  })
}

/** Отметка задачи: что и с каким итогом. */
export interface AcknowledgeVars {
  task_id: string
  body: AcknowledgeTask
}

/**
 * Отметить задачу (`notifications.task.acknowledge`): выполнена, принята или
 * отклонена с примечанием. После успеха перечитываются задачи и сводка шапки.
 */
export function useAcknowledgeTask() {
  const queryClient = useQueryClient()
  return useMutation<Awaited<ReturnType<typeof notificationsTaskAcknowledge>>, ApiError, AcknowledgeVars>({
    mutationFn: (v) => notificationsTaskAcknowledge(v.task_id, v.body),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: taskKeys.all })
      void queryClient.invalidateQueries({ queryKey: ['notification'] })
    },
  })
}
