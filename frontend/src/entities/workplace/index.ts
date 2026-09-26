/**
 * Рабочее место и пост (FR-6, FR-81, FR-83, FR-137): ключи кэша и панель «Посты»
 * под живой картой — участок, назначенный сотрудник, присутствие (СКУД, ключ
 * вставлен), текущее изделие. Люди на схеме не показываются — только здесь (PRD §4.1).
 * Эпик 13: назначения на посты в смене, квалификации, действия исполнителя
 * с рабочего места и запрос назначения контролёра с согласованием начальника ОТК.
 *
 * Операции — `access.workplace.list`, `journal.entry.list` (история поста),
 * `access.assignment.list|set|clear`,
 * `access.qualification.list`, `access.operator.report_deviation|request_inspection`,
 * `documents.document.request|list` (contracts/openapi.yaml), сгенерированный клиент.
 */
import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { entityKeys } from '@/shared/api/keys'
import {
  accessAssignmentClear,
  accessAssignmentList,
  accessAssignmentSet,
  accessOperatorReportDeviation,
  accessOperatorRequestInspection,
  accessQualificationList,
  accessWorkplaceList,
  documentsDocumentList,
  documentsDocumentRequest,
  journalEntryList,
} from '@/shared/api/generated/client'
import type {
  AccessAssignment,
  AccessAssignmentList,
  AccessQualification,
  AccessQualificationStatus,
  ClearAssignment,
  DocumentSummary,
  JournalEntryList,
  JournalEntryView,
  PostRow,
  PostRowPresence,
  ReportDeviation,
  RequestDecision,
  RequestInspection,
  SetAssignment,
} from '@/shared/api/generated/model'
import type { StatusTone } from '@/shared/api/generated/statuses'
import type { ApiError } from '@/shared/api/problem'
import type { Envelope } from '@/shared/api/response'
import { useMomentStore } from '@/shared/model/moment'

export type { AccessAssignment, AccessAssignmentList, AccessQualification, AccessQualificationStatus, DocumentSummary, JournalEntryView, SetAssignment, ClearAssignment }

export const workplaceKeys = entityKeys('workplace')

/** Присутствие на посту (FR-6): «неизвестно» — не «на месте». */
export type PostPresence = PostRowPresence
export type { PostRow }

/** Посты под картой на момент из useMomentStore — `access.workplace.list`. */
export function usePosts(params: MaybeRefOrGetter<{ workshop?: string; run_id?: string }>) {
  const moment = useMomentStore()
  return useQuery({
    queryKey: computed(() => workplaceKeys.list('posts', toValue(params), moment.params)),
    queryFn: async ({ signal }): Promise<Envelope<PostRow[]>> => {
      const res = await accessWorkplaceList({ ...toValue(params), ...moment.params }, { signal })
      return { data: res.data.items, headers: res.headers }
    },
    placeholderData: keepPreviousData,
  })
}

/**
 * История поста — записи журнала в потоке `workplace:‹id›` (`journal.entry.list`):
 * назначения и снятия с поста, ключ вставлен/вынут, допуск открыт/снят, сеанс
 * завершён, отклонения присутствия (каталог событий, поток `workplace`).
 * Журнал отдаёт записи по возрастанию seq страницами; `limit` — сколько взять.
 * Чтение закрыто правом `journal.entry.list` — без него сервер отвечает ошибкой.
 */
export function useWorkplaceHistory(
  workplaceId: MaybeRefOrGetter<string | null | undefined>,
  runId: MaybeRefOrGetter<string | undefined> = undefined,
  limit = 200,
) {
  const moment = useMomentStore()
  const params = computed(() => {
    const run = toValue(runId)
    return { stream: `workplace:${toValue(workplaceId) ?? ''}`, limit, ...(run ? { run_id: run } : {}), ...moment.params }
  })
  return useQuery({
    queryKey: computed(() => workplaceKeys.one(toValue(workplaceId) ?? '', 'history', params.value)),
    queryFn: async ({ signal }): Promise<Envelope<JournalEntryList>> => {
      const res = await journalEntryList(params.value, { signal })
      return { data: res.data, headers: res.headers }
    },
    enabled: computed(() => !!toValue(workplaceId)),
    retry: false,
  })
}

// ─────────────────── смены и назначения на посты (FR-81, PRD §11.18) ───────────────────

/** Назначения на посты в смене — `access.assignment.list` (ответ целиком: нужен basis_seq). */
export function useAssignments(params: MaybeRefOrGetter<{ shift_id?: string; workshop?: string; run_id?: string }>) {
  const moment = useMomentStore()
  const full = computed(() => ({ ...toValue(params), ...moment.params }))
  return useQuery({
    queryKey: computed(() => workplaceKeys.list('assignments', full.value)),
    queryFn: async ({ signal }): Promise<Envelope<AccessAssignmentList>> => {
      const res = await accessAssignmentList(full.value, { signal })
      return { data: res.data, headers: res.headers }
    },
    placeholderData: keepPreviousData,
    retry: false,
  })
}

/**
 * Квалификации и аттестации (FR-80) — `access.qualification.list`; сотрудник пуст — все.
 * `runId` — прогон сценария (окно записи берёт его из адреса).
 */
export function useQualifications(personId: MaybeRefOrGetter<string | null | undefined> = null, runId: MaybeRefOrGetter<string | undefined> = undefined) {
  const moment = useMomentStore()
  const full = computed(() => {
    const p = toValue(personId)
    const run = toValue(runId)
    return { ...(p ? { person_id: p } : {}), ...(run ? { run_id: run } : {}), ...moment.params }
  })
  return useQuery({
    queryKey: computed(() => workplaceKeys.list('qualifications', full.value)),
    queryFn: async ({ signal }): Promise<Envelope<AccessQualification[]>> => {
      const res = await accessQualificationList(full.value, { signal })
      return { data: res.data.items, headers: res.headers }
    },
    placeholderData: keepPreviousData,
    retry: false,
  })
}

/**
 * Допуск по квалификации «на глаз» для списка выбора: есть квалификации и все
 * истекли или отозваны — назначать нельзя (FR-81). Окончательно решает сервер
 * гардом назначения; нет сведений — неизвестно (null), а не «допущен».
 */
export function qualificationVerdict(quals: readonly AccessQualification[]): 'ok' | 'expiring' | 'expired' | null {
  if (!quals.length) return null
  if (quals.some((q) => q.status === 'valid')) return 'ok'
  if (quals.some((q) => q.status === 'expiring')) return 'expiring'
  return 'expired'
}

/** Команда назначения: назначить или снять с поста. */
export type AssignmentCommand = { kind: 'set'; body: SetAssignment } | { kind: 'clear'; body: ClearAssignment }

/**
 * Назначить исполнителя или контролёра на пост и снять с поста
 * (`access.assignment.set|clear`). Контролёра — только с документом
 * согласования начальника ОТК (`approval_document_id`, PRD §11.18).
 */
export function useAssignmentCommand() {
  const queryClient = useQueryClient()
  return useMutation<Awaited<ReturnType<typeof accessAssignmentSet>>, ApiError, AssignmentCommand>({
    mutationFn: (c) => (c.kind === 'set' ? accessAssignmentSet(c.body) : accessAssignmentClear(c.body)),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: workplaceKeys.all }),
  })
}

/**
 * Шаблон документа «назначение контролёра на пост»: маршрут «запрос мастера →
 * согласование начальника ОТК» (полномочие `controller_assignment_approval`,
 * PRD §11.18). Шаблон и маршрут строит модуль documents (эпик 28).
 */
export const CONTROLLER_ASSIGNMENT_TEMPLATE = 'controller-assignment@1'

/**
 * Запросить назначение контролёра (`documents.document.request`, FR-146):
 * документ с маршрутом подписей; объект — пост, решение — кто и в какую смену.
 */
export function useControllerAssignmentRequest() {
  const queryClient = useQueryClient()
  return useMutation<Awaited<ReturnType<typeof documentsDocumentRequest>>, ApiError, RequestDecision>({
    mutationFn: (body) => documentsDocumentRequest(body),
    onSuccess: (_r, body) => {
      void queryClient.invalidateQueries({ queryKey: ['document'] })
      void queryClient.invalidateQueries({ queryKey: workplaceKeys.one(body.subject.id) })
    },
  })
}

/** Документы поста (согласования назначений) — `documents.document.list`. */
export function useWorkplaceDocuments(workplaceId: MaybeRefOrGetter<string | null | undefined>) {
  const moment = useMomentStore()
  return useQuery({
    queryKey: computed(() => workplaceKeys.one(toValue(workplaceId) ?? '', 'documents', moment.params)),
    queryFn: async ({ signal }): Promise<Envelope<DocumentSummary[]>> => {
      const res = await documentsDocumentList({ subject: 'workplace', id: toValue(workplaceId) ?? '', ...moment.params }, { signal })
      return { data: res.data.items, headers: res.headers }
    },
    enabled: computed(() => !!toValue(workplaceId)),
    retry: false,
  })
}

// ─────────────────────── действия исполнителя (FR-137) ───────────────────────

/** Действие с рабочего места: сообщить об отклонении или запросить контроль. */
export type OperatorCommand =
  | { kind: 'report_deviation'; workplace_id: string; body: ReportDeviation }
  | { kind: 'request_inspection'; workplace_id: string; body: RequestInspection }

/**
 * Действия исполнителя (`access.operator.report_deviation|request_inspection`):
 * принимаются только с рабочего места, на которое он назначен (сервер сверяет
 * место сеанса, барьер 2 AD-15).
 */
export function useOperatorCommand() {
  const queryClient = useQueryClient()
  return useMutation<Awaited<ReturnType<typeof accessOperatorReportDeviation>>, ApiError, OperatorCommand>({
    mutationFn: (c) =>
      c.kind === 'report_deviation' ? accessOperatorReportDeviation(c.workplace_id, c.body) : accessOperatorRequestInspection(c.workplace_id, c.body),
    onSuccess: (_r, c) => {
      void queryClient.invalidateQueries({ queryKey: workplaceKeys.one(c.workplace_id) })
      void queryClient.invalidateQueries({ queryKey: ['task'] })
    },
  })
}

/**
 * Присутствие на посту → ключ текста (FR-6, FR-84): «неизвестно» — не «на месте»
 * (NFR-UI-4). Те же тексты, что у панели «Посты» под живой картой.
 */
export const PRESENCE_TEXT: Record<PostPresence, string> = {
  present: 'liveMap.posts.present',
  key_missing: 'liveMap.posts.keyMissing',
  owner_absent: 'liveMap.posts.ownerAbsent',
  absent: 'mapWidgets.posts.absent',
  not_assigned: 'liveMap.posts.notAssigned',
  unknown: 'empty.noDataUnknown',
}

/**
 * Присутствие → тон словаря статусов (AD-30): на месте — зелёный; расхождение
 * СКУД и ключа — жёлтый или красный; данных нет — серый «неизвестно».
 */
export const PRESENCE_TONE: Record<PostPresence, StatusTone> = {
  present: 'success',
  key_missing: 'attention',
  owner_absent: 'danger',
  absent: 'danger',
  not_assigned: 'neutral',
  unknown: 'neutral',
}

/** Тип метки Naive UI для присутствия: на месте — успех, расхождение — предупреждение. */
export const presenceTagType = (p: PostPresence): 'success' | 'warning' | 'default' =>
  p === 'present' ? 'success' : p === 'not_assigned' || p === 'unknown' ? 'default' : 'warning'
