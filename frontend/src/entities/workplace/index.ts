/**
 * Рабочее место и пост (FR-6, FR-81, FR-83, FR-137): ключи кэша и панель «Посты»
 * под живой картой — участок, назначенный сотрудник, присутствие (СКУД, ключ
 * вставлен), текущее изделие. Люди на схеме не показываются — только здесь (PRD §4.1).
 * Эпик 13: назначения на посты в смене, квалификации, действия исполнителя
 * с рабочего места и запрос назначения контролёра с согласованием начальника ОТК.
 *
 * Операции — `access.workplace.list`, `access.workplace.read|history` (окно поста),
 * `access.assignment.list|set|clear`,
 * `access.qualification.list`, `access.operator.report_deviation|request_inspection`,
 * `documents.document.request|list` (contracts/openapi.yaml), сгенерированный клиент.
 */
import { computed, toValue, watch, type MaybeRefOrGetter } from 'vue'
import { keepPreviousData, useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { entityKeys } from '@/shared/api/keys'
import {
  accessAssignmentClear,
  accessAssignmentList,
  accessCandidateList,
  accessAssignmentSet,
  accessOperatorReportDeviation,
  accessOperatorRequestInspection,
  accessQualificationList,
  accessWorkplaceAdmit,
  accessWorkplaceHistory,
  accessWorkplaceList,
  accessWorkplaceRead,
  accessWorkplaceRelease,
  documentsDocumentList,
  documentsDocumentRequest,
} from '@/shared/api/generated/client'
import type {
  AdmitWorkplace,
  ReleaseWorkplace,
  AccessAssignment,
  AccessAssignmentList,
  AccessQualification,
  AccessQualificationStatus,
  ClearAssignment,
  DocumentSummary,
  PostRow,
  PostRowPresence,
  ReportDeviation,
  RequestDecision,
  RequestInspection,
  SetAssignment,
  WorkplaceAssignee,
  WorkplaceCard,
  WorkplaceEvent,
  WorkplaceEventKind,
} from '@/shared/api/generated/model'
import type { StatusTone } from '@/shared/api/generated/statuses'
import type { ApiError } from '@/shared/api/problem'
import type { Envelope } from '@/shared/api/response'
import { useMomentStore } from '@/shared/model/moment'

export type { AccessAssignment, AccessAssignmentList, AccessQualification, AccessQualificationStatus, DocumentSummary, SetAssignment, ClearAssignment }
export type { WorkplaceAssignee, WorkplaceCard, WorkplaceEvent, WorkplaceEventKind }

export const workplaceKeys = entityKeys('workplace')

/** Присутствие на посту (FR-6): «неизвестно» — не «на месте». */
export type PostPresence = PostRowPresence
export type { PostRow }

/** Ключи сотрудника: проход СКУД пишется в поток сотрудника (эпик 37). */
const personKeys = entityKeys('person')

/**
 * Посты под картой на момент из useMomentStore — `access.workplace.list`.
 * Присутствие меняют и проходы СКУД (FR-6, эпик 37): они в потоке сотрудника,
 * SSE инвалидирует `[person, LIST]` — метка `presence-tick` перечитывается, и
 * список постов перечитывается вслед за ней (без опроса по таймеру).
 */
export function usePosts(params: MaybeRefOrGetter<{ workshop?: string; run_id?: string }>) {
  const moment = useMomentStore()
  const tick = useQuery({ queryKey: personKeys.list('presence-tick'), queryFn: () => Date.now(), staleTime: Infinity })
  const posts = useQuery({
    queryKey: computed(() => workplaceKeys.list('posts', toValue(params), moment.params)),
    queryFn: async ({ signal }): Promise<Envelope<PostRow[]>> => {
      const res = await accessWorkplaceList({ ...toValue(params), ...moment.params }, { signal })
      return { data: res.data.items, headers: res.headers }
    },
    placeholderData: keepPreviousData,
  })
  watch(
    () => tick.dataUpdatedAt.value,
    (now, before) => {
      if (before && now !== before) void posts.refetch()
    },
  )
  return posts
}

/** Параметры чтения окна поста: прогон сценария (из адреса) и момент. */
function readParams(runId: MaybeRefOrGetter<string | undefined>) {
  const moment = useMomentStore()
  return computed(() => {
    const run = toValue(runId)
    return run ? { ...moment.params, run_id: run } : { ...moment.params }
  })
}

/**
 * Карточка поста (UI-16) — `access.workplace.read`: строка панели «Посты»
 * (назначен, присутствие, текущее изделие), область поста и назначения
 * текущей смены с именами и допуском по квалификации.
 */
export function useWorkplaceCard(workplaceId: MaybeRefOrGetter<string | null | undefined>, runId: MaybeRefOrGetter<string | undefined> = undefined) {
  const params = readParams(runId)
  return useQuery({
    queryKey: computed(() => workplaceKeys.one(toValue(workplaceId) ?? '', 'card', params.value)),
    queryFn: async ({ signal }): Promise<Envelope<WorkplaceCard>> => {
      const res = await accessWorkplaceRead(toValue(workplaceId) ?? '', params.value, { signal })
      return { data: res.data, headers: res.headers }
    },
    enabled: computed(() => !!toValue(workplaceId)),
    retry: false,
  })
}

/**
 * История поста (UI-16) — `access.workplace.history`: назначения и снятия,
 * ключ вставлен/вынут, допуск открыт/снят/отозван, отклонения присутствия.
 * Сервер отдаёт новые сверху страницами по `limit`; следующая — по `next_cursor`
 * («показать ещё»). Страницы склеиваются в один список.
 */
export function useWorkplaceHistory(
  workplaceId: MaybeRefOrGetter<string | null | undefined>,
  runId: MaybeRefOrGetter<string | undefined> = undefined,
  limit = 50,
) {
  const params = readParams(runId)
  const q = useInfiniteQuery({
    queryKey: computed(() => workplaceKeys.one(toValue(workplaceId) ?? '', 'history', limit, params.value)),
    queryFn: async ({ signal, pageParam }) => {
      const res = await accessWorkplaceHistory(toValue(workplaceId) ?? '', { ...params.value, limit, ...(pageParam ? { cursor: pageParam } : {}) }, { signal })
      return res.data
    },
    initialPageParam: '' as string,
    getNextPageParam: (last) => last.next_cursor || undefined,
    enabled: computed(() => !!toValue(workplaceId)),
    retry: false,
  })
  return {
    /** Записи всех загруженных страниц; null — первая страница ещё не пришла. */
    items: computed<WorkplaceEvent[] | null>(() => (q.data.value ? q.data.value.pages.flatMap((p) => p.items) : null)),
    isPending: computed(() => q.isLoading.value),
    error: computed(() => q.error.value ?? null),
    hasMore: computed(() => q.hasNextPage.value),
    loadingMore: computed(() => q.isFetchingNextPage.value),
    loadMore: () => void q.fetchNextPage(),
  }
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

/** Квалификации и аттестации (FR-80) — `access.qualification.list`, все сотрудники. */
export function useQualifications() {
  const moment = useMomentStore()
  return useQuery({
    queryKey: computed(() => workplaceKeys.list('qualifications', moment.params)),
    queryFn: async ({ signal }): Promise<Envelope<AccessQualification[]>> => {
      const res = await accessQualificationList(moment.params, { signal })
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

/** Решение в документе назначения контролёра: кто и в какую смену. */
export const controllerDecision = (personId: string, shiftId: string): string => `quality_inspector:${personId}@${shiftId}`

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

/** Команда допуска к рабочему месту (барьер 2, FR-83; эпик 37). */
export type AdmissionCommand =
  | { kind: 'admit'; workplace_id: string; body: AdmitWorkplace }
  | { kind: 'release'; workplace_id: string; body: ReleaseWorkplace }

/**
 * Допуск к рабочему месту и его снятие — `access.workplace.admit|release`:
 * после ответа перечитываются пост, посты и сеанс (рабочее место сеанса).
 */
export function useAdmission() {
  const queryClient = useQueryClient()
  return useMutation<Awaited<ReturnType<typeof accessWorkplaceAdmit>>, ApiError, AdmissionCommand>({
    mutationFn: (c) => (c.kind === 'admit' ? accessWorkplaceAdmit(c.workplace_id, c.body) : accessWorkplaceRelease(c.workplace_id, c.body)),
    onSuccess: (_r, c) => {
      void queryClient.invalidateQueries({ queryKey: workplaceKeys.one(c.workplace_id) })
      void queryClient.invalidateQueries({ queryKey: workplaceKeys.all })
      void queryClient.invalidateQueries({ queryKey: ['session'] })
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

/**
 * Кандидаты на пост в смену (access.candidate.list): кто подходит по роли и
 * области, вердикт квалификации на дату смены, можно ли назначить и почему.
 */
export function useCandidates(workplaceId: MaybeRefOrGetter<string | null | undefined>, shiftId: MaybeRefOrGetter<string | null | undefined>) {
  const moment = useMomentStore()
  const params = computed(() => ({ ...(toValue(shiftId) ? { shift_id: toValue(shiftId)! } : {}), ...moment.params }))
  return useQuery({
    queryKey: computed(() => workplaceKeys.one(toValue(workplaceId) ?? '', 'candidates', params.value)),
    queryFn: ({ signal }) => accessCandidateList(toValue(workplaceId) ?? '', params.value, { signal }),
    enabled: computed(() => !!toValue(workplaceId)),
    retry: false,
  })
}
