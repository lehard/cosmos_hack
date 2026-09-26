/**
 * Чтение и команды разбора через сгенерированный клиент (AD-20, AD-21):
 * операции `analysis.*` из contracts/openapi.yaml. Ключи кэша — по соглашению
 * shared/api/keys.ts: данные несоответствия — `[nonconformity, nc_id, …]`,
 * инцидента — `[incident, id, …]`; параметры момента — последним элементом,
 * поэтому SSE-инвалидация `(сущность, id)` обновляет экраны сама.
 */
import { computed, inject, toValue, type MaybeRefOrGetter, type Ref } from 'vue'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { routeLocationKey } from 'vue-router'
import {
  analysisCauseConclude,
  analysisCircumstancesRead,
  analysisCommonFactorsRead,
  analysisGroupList,
  analysisHypothesisList,
  analysisHypothesisReject,
  analysisIncidentList,
  analysisMeasurementRequest,
  analysisRiskScopeRead,
  analysisScopeExpand,
  analysisScopeNarrow,
} from '@/shared/api/generated/client'
import type { ChangeScope, ConcludeCause, Permission, RejectHypothesis, RequestMeasurement } from '@/shared/api/generated/model'
import { entityKeys } from '@/shared/api/keys'
import { backendModeOf } from '@/shared/api/response'
import { usePermissions } from '@/entities/permission'
import { useMomentStore } from '@/shared/model/moment'
import { useAnalysisFocusStore } from './model/focus'
import { sortGroups } from './model/groups'
import { toCircumstancesModel, toCommonFactorsModel, toHypothesesModel, toNcGroup, toRiskScopeModel } from './model/map'
import type { WidgetSource } from './model/source'
import type { CircumstancesModel, CommonFactorsModel, HypothesesModel, NcGroup, RiskScopeModel } from './model/types'

export const incidentKeys = entityKeys('incident')
const ncKeys = entityKeys('nonconformity')

type Enveloped<A> = { data: A; headers?: Headers }

/**
 * Запрос → источник виджета: данные уже в форме экрана, режим fixtures | live
 * из заголовка ответа. Выключенный запрос (нет выбора) — не «загрузка».
 */
export function toSource<A, T>(
  q: { data: Readonly<Ref<Enveloped<A> | undefined>>; isLoading: Readonly<Ref<boolean>>; error: Readonly<Ref<unknown>> },
  map: (a: A) => T,
): WidgetSource<T> {
  return {
    data: computed(() => (q.data.value ? map(q.data.value.data) : null)),
    isPending: computed(() => q.isLoading.value),
    error: computed(() => q.error.value ?? undefined),
    mode: computed(() => backendModeOf(q.data.value)),
    basisSeq: computed(() => (q.data.value?.data as { basis_seq?: number } | undefined)?.basis_seq ?? null),
  }
}

/** Разбор обстоятельств несоответствия (FR-153). */
export function useCircumstances(ncId: MaybeRefOrGetter<string | null>) {
  const moment = useMomentStore()
  const q = useQuery({
    queryKey: computed(() => ncKeys.one(toValue(ncId) ?? '', 'circumstances', moment.params)),
    queryFn: ({ signal }) => analysisCircumstancesRead(toValue(ncId)!, moment.params, { signal }),
    enabled: computed(() => Boolean(toValue(ncId))),
  })
  return toSource<Parameters<typeof toCircumstancesModel>[0], CircumstancesModel>(q, toCircumstancesModel)
}

/** Гипотезы и похожие случаи несоответствия (FR-59, FR-60). */
export function useHypotheses(ncId: MaybeRefOrGetter<string | null>) {
  const moment = useMomentStore()
  const q = useQuery({
    queryKey: computed(() => ncKeys.one(toValue(ncId) ?? '', 'hypotheses', moment.params)),
    queryFn: ({ signal }) => analysisHypothesisList(toValue(ncId)!, moment.params, { signal }),
    enabled: computed(() => Boolean(toValue(ncId))),
  })
  return toSource<Parameters<typeof toHypothesesModel>[0], HypothesesModel>(q, toHypothesesModel)
}

/** Группы несоответствий «вид дефекта × операция × оборудование». */
export function useNcGroups() {
  const moment = useMomentStore()
  const q = useQuery({
    queryKey: computed(() => ncKeys.list('analysis-groups', moment.params)),
    queryFn: ({ signal }) => analysisGroupList(moment.params, { signal }),
  })
  return toSource<{ items: Parameters<typeof toNcGroup>[0][] }, NcGroup[]>(q, (a) => a.items.map(toNcGroup))
}

/** Общие факторы группы (FR-135). */
export function useCommonFactors(groupKey: MaybeRefOrGetter<string | null>) {
  const moment = useMomentStore()
  const q = useQuery({
    queryKey: computed(() => ncKeys.list('common-factors', toValue(groupKey) ?? '', moment.params)),
    queryFn: ({ signal }) => analysisCommonFactorsRead(toValue(groupKey)!, moment.params, { signal }),
    enabled: computed(() => Boolean(toValue(groupKey))),
  })
  return toSource<Parameters<typeof toCommonFactorsModel>[0], CommonFactorsModel>(q, toCommonFactorsModel)
}

/** Инциденты — для выбора области риска. */
export function useIncidents() {
  const moment = useMomentStore()
  return useQuery({
    queryKey: computed(() => incidentKeys.list(moment.params)),
    queryFn: ({ signal }) => analysisIncidentList(moment.params, { signal }),
  })
}

/** Область риска инцидента (FR-61, FR-62). */
export function useRiskScope(incidentId: MaybeRefOrGetter<string | null>) {
  const moment = useMomentStore()
  const q = useQuery({
    queryKey: computed(() => incidentKeys.one(toValue(incidentId) ?? '', 'risk-scope', moment.params)),
    queryFn: ({ signal }) => analysisRiskScopeRead(toValue(incidentId)!, moment.params, { signal }),
    enabled: computed(() => Boolean(toValue(incidentId))),
  })
  return toSource<Parameters<typeof toRiskScopeModel>[0], RiskScopeModel>(q, toRiskScopeModel)
}

// ─────────────────────────────── права и команды ───────────────────────────────

/**
 * Есть ли у пользователя действие над объектом — по списку прав сервера
 * (AD-15: интерфейс права не вычисляет). Право без `object_id` — на все объекты вида.
 */
export const canPerform = (items: readonly Permission[] | undefined, action: string, subject: string, id?: string | null): boolean =>
  (items ?? []).some((p) => p.action === action && p.subject === subject && (!p.object_id || p.object_id === id))

/** Проверка действия по текущему списку прав; в воспроизведении — нельзя (FR-4). */
export function useCan() {
  const permissions = usePermissions()
  const moment = useMomentStore()
  return (action: string, subject: string, id?: string | null) => !moment.isReplay && canPerform(permissions.data.value?.data.items, action, subject, id)
}

/** Идентификатор команды — UUIDv7 ставит сервер у фактов; у команды клиент даёт свой UUID (AD-7). */
const commandId = (): string => globalThis.crypto.randomUUID()

type Cmd<B> = Omit<B, 'command_id' | 'policy_seq'>

/**
 * Команды технолога по разбору (AD-39: каждая несёт `basis_seq` — на чём
 * построен экран — и `policy_seq` — по какой политике прав). После записи
 * сбрасывается кэш инцидента и несоответствия — экраны перечитываются.
 */
export function useAnalysisCommands() {
  const qc = useQueryClient()
  const permissions = usePermissions()
  const policySeq = () => permissions.data.value?.data.policy_seq ?? 0
  const refresh = () => Promise.all([qc.invalidateQueries({ queryKey: incidentKeys.all }), qc.invalidateQueries({ queryKey: ncKeys.all })])

  return {
    /** Подтвердить причину (`analysis.cause.conclude`, критическое действие). */
    concludeCause: useMutation({
      mutationFn: (v: { incidentId: string; body: Cmd<ConcludeCause> }) =>
        analysisCauseConclude(v.incidentId, { ...v.body, command_id: commandId(), policy_seq: policySeq() }),
      onSuccess: refresh,
    }),
    /** Отклонить гипотезу (`analysis.hypothesis.reject`). */
    rejectHypothesis: useMutation({
      mutationFn: (v: { ncId: string; body: Cmd<RejectHypothesis> }) =>
        analysisHypothesisReject(v.ncId, { ...v.body, command_id: commandId(), policy_seq: policySeq() }),
      onSuccess: refresh,
    }),
    /** Запросить измерение (`analysis.measurement.request`). */
    requestMeasurement: useMutation({
      mutationFn: (v: { ncId: string; body: Cmd<RequestMeasurement> }) =>
        analysisMeasurementRequest(v.ncId, { ...v.body, command_id: commandId(), policy_seq: policySeq() }),
      onSuccess: refresh,
    }),
    /** Сузить область (`analysis.scope.narrow`) — только с основанием. */
    narrowScope: useMutation({
      mutationFn: (v: { incidentId: string; body: Cmd<ChangeScope> }) =>
        analysisScopeNarrow(v.incidentId, { ...v.body, command_id: commandId(), policy_seq: policySeq() }),
      onSuccess: refresh,
    }),
    /** Расширить область (`analysis.scope.expand`). */
    expandScope: useMutation({
      mutationFn: (v: { incidentId: string; body: Cmd<ChangeScope> }) =>
        analysisScopeExpand(v.incidentId, { ...v.body, command_id: commandId(), policy_seq: policySeq() }),
      onSuccess: refresh,
    }),
  }
}

// ─────────────────────────────── что сейчас разбираем ───────────────────────────────

/** Параметр запроса адреса страницы (`?nc=…`, `?incident=…`) — если роутер есть. */
function useQueryParam(name: string) {
  const route = inject(routeLocationKey, null)
  return computed(() => {
    const v = route?.query[name]
    return typeof v === 'string' && v ? v : null
  })
}

/**
 * Группа в разборе: выбранная в «Разборе причин», иначе самая крупная.
 * Общие факторы и разбор обстоятельств открываются сразу, без лишнего клика.
 */
export function useFocusedGroup() {
  const focus = useAnalysisFocusStore()
  const groups = useNcGroups()
  const group = computed(() => {
    const all = groups.data.value ?? []
    return all.find((g) => g.group_key === focus.groupKey) ?? sortGroups(all)[0] ?? null
  })
  return { groups, group }
}

/**
 * Несоответствие в разборе: выбранное (фокус), из адреса (`?nc=` — переход из
 * карточки несоответствия или паспорта), иначе первое несоответствие группы.
 */
export function useFocusedNc() {
  const focus = useAnalysisFocusStore()
  const fromRoute = useQueryParam('nc')
  const { group } = useFocusedGroup()
  // Несоответствие расследования (primary_nc_id) — вход гипотез и дорожек; первое в группе — запасной путь.
  const { list, id } = useFocusedIncident()
  const primary = computed(() => list.value.find((i) => i.incident_id === id.value)?.primary_nc_id ?? null)
  return computed(() => focus.ncId ?? fromRoute.value ?? primary.value ?? group.value?.nc_ids?.[0] ?? null)
}

/** Инцидент в разборе: выбранный, из адреса (`?incident=`), иначе первый открытый. */
export function useFocusedIncident() {
  const focus = useAnalysisFocusStore()
  const fromRoute = useQueryParam('incident')
  const incidents = useIncidents()
  const list = computed(() => incidents.data.value?.data.items ?? [])
  const id = computed(() => focus.incidentId ?? fromRoute.value ?? (list.value.find((i) => i.status === 'open') ?? list.value[0])?.incident_id ?? null)
  return { incidents, list, id }
}
