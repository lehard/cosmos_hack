/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import {
  useMutation,
  useQuery
} from '@tanstack/vue-query';
import type {
  DataTag,
  MutationFunction,
  QueryClient,
  QueryFunction,
  QueryKey,
  UseMutationOptions,
  UseMutationReturnType,
  UseQueryOptions,
  UseQueryReturnType
} from '@tanstack/vue-query';

import {
  computed,
  toValue,
  unref
} from 'vue';
import type {
  MaybeRefOrGetter
} from 'vue';

import type {
  AccessPermissionExplainParams,
  AccessPermissionListParams,
  AccessWorkplaceListParams,
  AcknowledgeTask,
  ActivateVersion,
  AdmitPassport,
  AlertList,
  AnalysisCircumstancesReadParams,
  AnalysisCommonFactorsReadParams,
  AnalysisGroupListParams,
  AnalysisHypothesisListParams,
  AnalysisIncidentListParams,
  AnalysisRiskScopeReadParams,
  AnalysisSimilarListParams,
  AnalyticsControlChartReadParams,
  AnalyticsMetricDrilldownParams,
  AnalyticsNodeCountersReadParams,
  AnalyticsOverview,
  AnalyticsOverviewReadParams,
  AnalyticsTileListParams,
  AnalyzerCheckList,
  AnalyzerList,
  AnalyzerPassport,
  ApplyCarrier,
  ApplyInjection,
  AssessItem,
  AssignAction,
  AttentionList,
  Board,
  ChangeScope,
  Circumstances,
  CloseIncident,
  CloseIntervention,
  CloseNonconformity,
  CommonFactors,
  CompensatePosting,
  ConcessionList,
  ConcludeCause,
  ConfirmIdentification,
  ConfirmNonconformity,
  ControlChart,
  DecisionQueue,
  DemoPersonaList,
  Desk,
  DraftVersion,
  EquipmentList,
  EquipmentState,
  EquipmentTimeline,
  ErpChannelList,
  ErpMessage,
  ErpMessageList,
  ErpMessageListParams,
  ErpMessageReadParams,
  ErpOrderList,
  ErpOrderListParams,
  EvaluateAction,
  Explanation,
  FinishOperation,
  Hypotheses,
  ImplementAction,
  ImportFile,
  ImportResult,
  IncidentList,
  IngestBatch,
  IngestMetrics,
  IngestOutcome,
  IngestQuarantineListParams,
  IngestResult,
  IngestSourceListParams,
  InjectionList,
  InspectionCoverage,
  InspectionResultList,
  IntegrityStatus,
  IsolateItem,
  ItemGenealogy,
  ItemGenealogyReadParams,
  ItemHistory,
  ItemHistoryListParams,
  ItemItemListParams,
  ItemItemLookupParams,
  ItemList,
  ItemLookup,
  ItemPassport,
  ItemPassportReadParams,
  JournalEntryList,
  JournalEntryListParams,
  JournalEntryView,
  JournalHead,
  JournalHeadReadParams,
  JournalStreamSubscribeParams,
  JournalTimelineReadParams,
  LiveMap,
  MachinelogsEquipmentListParams,
  MachinelogsEquipmentReadParams,
  MachinelogsRunProfileReadParams,
  MachinelogsTimelineReadParams,
  MachinelogsViolationListParams,
  ManualEvent,
  MetricDrilldown,
  MetricTileList,
  NCCard,
  NCList,
  NcGroupList,
  NodeCounterSet,
  NonconformityCardReadParams,
  NonconformityConcessionListParams,
  NonconformityNonconformityListParams,
  NonconformityQueueListParams,
  NotificationSummary,
  NotificationsAlertListParams,
  NotificationsAttentionListParams,
  NotificationsSummaryReadParams,
  NotificationsTaskListParams,
  OpenIntervention,
  OpsHealth,
  OpsStoppedItemListParams,
  PauseOperation,
  PermissionList,
  PostList,
  Problem,
  ProcessBpmn,
  ProcessLiveMapReadParams,
  ProcessNodeCard,
  ProcessNodeReadParams,
  ProcessVersion,
  ProcessVersionDiff,
  ProcessVersionDiffParams,
  ProcessVersionList,
  ProcessVersionListParams,
  ProcessVersionReadParams,
  QualityCoverageReadParams,
  QualityDefectList,
  QualityDefectListParams,
  QualityEscapeList,
  QualityEscapeListParams,
  QualityInspectionListParams,
  QualityReactionMapReadParams,
  QualitySignal,
  QualitySignalList,
  QualitySignalListParams,
  QualitySignalReadParams,
  QuarantineEntry,
  QuarantineList,
  ReactionMap,
  Receipt,
  ReceiveMovement,
  RecordAssembly,
  RecordHypothesis,
  RecordPresentation,
  RecordRelease,
  RegisterItem,
  ReinstatePassport,
  RejectHypothesis,
  RejectSignal,
  ReleaseContainment,
  ReleaseProcessHold,
  RemoveCarrier,
  ReprocessMessage,
  RequestMeasurement,
  RequestRecheck,
  ResendPosting,
  ResolveLot,
  ResolvePresentation,
  ResumeOperation,
  RetirePassport,
  RetireVersion,
  RetryProcessing,
  RevokeConcession,
  RiskScope,
  Run,
  RunControl,
  RunList,
  RunProfile,
  ScenarioList,
  ScopeAnalysis,
  SendMovement,
  Session,
  SessionCreate,
  SetContainment,
  SetDisposition,
  SetProcessHold,
  SetSpeed,
  SettingList,
  SimilarCaseList,
  SimulationBoardReadParams,
  SimulationRunListParams,
  SimulationRunReadParams,
  SourceList,
  StartOperation,
  StartRun,
  StoppedItemList,
  SubmitVersion,
  SwitchSource,
  TaskList,
  TimelineData,
  VerifyDisposition,
  ViolationList,
  VisionAnalyzerListParams,
  VisionCheckListParams,
  VisionPassportReadParams,
  WaiveReworkLimit
} from './model';


export type HTTPStatusCode1xx = 100 | 101 | 102 | 103;
export type HTTPStatusCode2xx = 200 | 201 | 202 | 203 | 204 | 205 | 206 | 207;
export type HTTPStatusCode3xx = 300 | 301 | 302 | 303 | 304 | 305 | 307 | 308;
export type HTTPStatusCode4xx = 400 | 401 | 402 | 403 | 404 | 405 | 406 | 407 | 408 | 409 | 410 | 411 | 412 | 413 | 414 | 415 | 416 | 417 | 418 | 419 | 420 | 421 | 422 | 423 | 424 | 426 | 428 | 429 | 431 | 451;
export type HTTPStatusCode5xx = 500 | 501 | 502 | 503 | 504 | 505 | 507 | 511;
export type HTTPStatusCodes = HTTPStatusCode1xx | HTTPStatusCode2xx | HTTPStatusCode3xx | HTTPStatusCode4xx | HTTPStatusCode5xx;




export type notificationsAlertListResponse200 = {
  data: AlertList
  status: 200
}

export type notificationsAlertListResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type notificationsAlertListResponseSuccess = (notificationsAlertListResponse200) & {
  headers: Headers;
};
export type notificationsAlertListResponseError = (notificationsAlertListResponseDefault) & {
  headers: Headers;
};

export const getNotificationsAlertListUrl = (params?: NotificationsAlertListParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/alerts?${stringifiedParams}` : `/api/v1/alerts`
}

/**
 * FR-8: просроченные изоляции, точки предъявления сверх срока, «не перемещено в изолятор», аномалии узлов, эскалации, нарушения целостности.
 * @summary Лента тревог
 */
export const notificationsAlertList = async (params?: NotificationsAlertListParams, options?: RequestInit): Promise<notificationsAlertListResponseSuccess> => {

  const res = await fetch(getNotificationsAlertListUrl(params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: notificationsAlertListResponseError['data'], status?: number} = new globalThis.Error();
    const data : notificationsAlertListResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: notificationsAlertListResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as notificationsAlertListResponseSuccess
}





export const getNotificationsAlertListQueryKey = (params?: MaybeRefOrGetter<NotificationsAlertListParams>,) => {
    return [
    'api','v1','alerts', ...(params ? [params] : [])
    ] as const;
    }


export const getNotificationsAlertListQueryOptions = <TData = Awaited<ReturnType<typeof notificationsAlertList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(params?: MaybeRefOrGetter<NotificationsAlertListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof notificationsAlertList>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getNotificationsAlertListQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof notificationsAlertList>>> = ({ signal }) => notificationsAlertList(toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof notificationsAlertList>>, TError, TData>
}

export type NotificationsAlertListQueryResult = NonNullable<Awaited<ReturnType<typeof notificationsAlertList>>>
export type NotificationsAlertListQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Лента тревог
 */

export function useNotificationsAlertList<TData = Awaited<ReturnType<typeof notificationsAlertList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 params?: MaybeRefOrGetter<NotificationsAlertListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof notificationsAlertList>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getNotificationsAlertListQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type analysisGroupListResponse200 = {
  data: NcGroupList
  status: 200
}

export type analysisGroupListResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type analysisGroupListResponseSuccess = (analysisGroupListResponse200) & {
  headers: Headers;
};
export type analysisGroupListResponseError = (analysisGroupListResponseDefault) & {
  headers: Headers;
};

export const getAnalysisGroupListUrl = (params?: AnalysisGroupListParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/analysis/groups?${stringifiedParams}` : `/api/v1/analysis/groups`
}

/**
 * Стол технолога, «Разбор причин»: несоответствия, сгруппированные по виду дефекта × операции × оборудованию, со статусом расследования.
 * @summary Группы несоответствий
 */
export const analysisGroupList = async (params?: AnalysisGroupListParams, options?: RequestInit): Promise<analysisGroupListResponseSuccess> => {

  const res = await fetch(getAnalysisGroupListUrl(params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: analysisGroupListResponseError['data'], status?: number} = new globalThis.Error();
    const data : analysisGroupListResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: analysisGroupListResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as analysisGroupListResponseSuccess
}





export const getAnalysisGroupListQueryKey = (params?: MaybeRefOrGetter<AnalysisGroupListParams>,) => {
    return [
    'api','v1','analysis','groups', ...(params ? [params] : [])
    ] as const;
    }


export const getAnalysisGroupListQueryOptions = <TData = Awaited<ReturnType<typeof analysisGroupList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(params?: MaybeRefOrGetter<AnalysisGroupListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof analysisGroupList>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getAnalysisGroupListQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof analysisGroupList>>> = ({ signal }) => analysisGroupList(toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof analysisGroupList>>, TError, TData>
}

export type AnalysisGroupListQueryResult = NonNullable<Awaited<ReturnType<typeof analysisGroupList>>>
export type AnalysisGroupListQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Группы несоответствий
 */

export function useAnalysisGroupList<TData = Awaited<ReturnType<typeof analysisGroupList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 params?: MaybeRefOrGetter<AnalysisGroupListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof analysisGroupList>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getAnalysisGroupListQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type analysisCommonFactorsReadResponse200 = {
  data: CommonFactors
  status: 200
}

export type analysisCommonFactorsReadResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type analysisCommonFactorsReadResponseSuccess = (analysisCommonFactorsReadResponse200) & {
  headers: Headers;
};
export type analysisCommonFactorsReadResponseError = (analysisCommonFactorsReadResponseDefault) & {
  headers: Headers;
};

export const getAnalysisCommonFactorsReadUrl = (groupKey: string,
    params?: AnalysisCommonFactorsReadParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/analysis/groups/${groupKey}/common-factors?${stringifiedParams}` : `/api/v1/analysis/groups/${groupKey}/common-factors`
}

/**
 * FR-135: таблица «сколько из N» — станок, инструмент, оснастка, программа, исполнитель, партия материала — со входом в гипотезу и сужение области.
 * @summary Общие факторы группы
 */
export const analysisCommonFactorsRead = async (groupKey: string,
    params?: AnalysisCommonFactorsReadParams, options?: RequestInit): Promise<analysisCommonFactorsReadResponseSuccess> => {

  const res = await fetch(getAnalysisCommonFactorsReadUrl(groupKey,params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: analysisCommonFactorsReadResponseError['data'], status?: number} = new globalThis.Error();
    const data : analysisCommonFactorsReadResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: analysisCommonFactorsReadResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as analysisCommonFactorsReadResponseSuccess
}





export const getAnalysisCommonFactorsReadQueryKey = (groupKey: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<AnalysisCommonFactorsReadParams>,) => {
    return [
    'api','v1','analysis','groups',groupKey,'common-factors', ...(params ? [params] : [])
    ] as const;
    }


export const getAnalysisCommonFactorsReadQueryOptions = <TData = Awaited<ReturnType<typeof analysisCommonFactorsRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(groupKey: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<AnalysisCommonFactorsReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof analysisCommonFactorsRead>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getAnalysisCommonFactorsReadQueryKey(groupKey,params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof analysisCommonFactorsRead>>> = ({ signal }) => analysisCommonFactorsRead(toValue(groupKey),toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, enabled: computed(() => toValue(groupKey) !== null && toValue(groupKey) !== undefined), ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof analysisCommonFactorsRead>>, TError, TData>
}

export type AnalysisCommonFactorsReadQueryResult = NonNullable<Awaited<ReturnType<typeof analysisCommonFactorsRead>>>
export type AnalysisCommonFactorsReadQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Общие факторы группы
 */

export function useAnalysisCommonFactorsRead<TData = Awaited<ReturnType<typeof analysisCommonFactorsRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 groupKey: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<AnalysisCommonFactorsReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof analysisCommonFactorsRead>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getAnalysisCommonFactorsReadQueryOptions(groupKey,params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type analyticsOverviewReadResponse200 = {
  data: AnalyticsOverview
  status: 200
}

export type analyticsOverviewReadResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type analyticsOverviewReadResponseSuccess = (analyticsOverviewReadResponse200) & {
  headers: Headers;
};
export type analyticsOverviewReadResponseError = (analyticsOverviewReadResponseDefault) & {
  headers: Headers;
};

export const getAnalyticsOverviewReadUrl = (params?: AnalyticsOverviewReadParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/analytics?${stringifiedParams}` : `/api/v1/analytics`
}

/**
 * FR-86…FR-89, кейс §2.4, §5.2: раздельный учёт (входной брак отличим от производственных ошибок), дефекты и изделия с дефектами раздельно, сравнение сопоставимых работ, происхождение времени (передано источником / вычислено системой).
 * @summary Аналитика: полный набор показателей
 */
export const analyticsOverviewRead = async (params?: AnalyticsOverviewReadParams, options?: RequestInit): Promise<analyticsOverviewReadResponseSuccess> => {

  const res = await fetch(getAnalyticsOverviewReadUrl(params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: analyticsOverviewReadResponseError['data'], status?: number} = new globalThis.Error();
    const data : analyticsOverviewReadResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: analyticsOverviewReadResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as analyticsOverviewReadResponseSuccess
}





export const getAnalyticsOverviewReadQueryKey = (params?: MaybeRefOrGetter<AnalyticsOverviewReadParams>,) => {
    return [
    'api','v1','analytics', ...(params ? [params] : [])
    ] as const;
    }


export const getAnalyticsOverviewReadQueryOptions = <TData = Awaited<ReturnType<typeof analyticsOverviewRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(params?: MaybeRefOrGetter<AnalyticsOverviewReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof analyticsOverviewRead>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getAnalyticsOverviewReadQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof analyticsOverviewRead>>> = ({ signal }) => analyticsOverviewRead(toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof analyticsOverviewRead>>, TError, TData>
}

export type AnalyticsOverviewReadQueryResult = NonNullable<Awaited<ReturnType<typeof analyticsOverviewRead>>>
export type AnalyticsOverviewReadQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Аналитика: полный набор показателей
 */

export function useAnalyticsOverviewRead<TData = Awaited<ReturnType<typeof analyticsOverviewRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 params?: MaybeRefOrGetter<AnalyticsOverviewReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof analyticsOverviewRead>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getAnalyticsOverviewReadQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type analyticsControlChartReadResponse200 = {
  data: ControlChart
  status: 200
}

export type analyticsControlChartReadResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type analyticsControlChartReadResponseSuccess = (analyticsControlChartReadResponse200) & {
  headers: Headers;
};
export type analyticsControlChartReadResponseError = (analyticsControlChartReadResponseDefault) & {
  headers: Headers;
};

export const getAnalyticsControlChartReadUrl = (stepKey: string,
    params?: AnalyticsControlChartReadParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/analytics/control-charts/${stepKey}?${stringifiedParams}` : `/api/v1/analytics/control-charts/${stepKey}`
}

/**
 * FR-5: доля дефектов или характеристика узла во времени с центральной линией и границами; выход за границы — аномалия узла.
 * @summary Контрольная карта узла
 */
export const analyticsControlChartRead = async (stepKey: string,
    params?: AnalyticsControlChartReadParams, options?: RequestInit): Promise<analyticsControlChartReadResponseSuccess> => {

  const res = await fetch(getAnalyticsControlChartReadUrl(stepKey,params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: analyticsControlChartReadResponseError['data'], status?: number} = new globalThis.Error();
    const data : analyticsControlChartReadResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: analyticsControlChartReadResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as analyticsControlChartReadResponseSuccess
}





export const getAnalyticsControlChartReadQueryKey = (stepKey: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<AnalyticsControlChartReadParams>,) => {
    return [
    'api','v1','analytics','control-charts',stepKey, ...(params ? [params] : [])
    ] as const;
    }


export const getAnalyticsControlChartReadQueryOptions = <TData = Awaited<ReturnType<typeof analyticsControlChartRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(stepKey: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<AnalyticsControlChartReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof analyticsControlChartRead>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getAnalyticsControlChartReadQueryKey(stepKey,params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof analyticsControlChartRead>>> = ({ signal }) => analyticsControlChartRead(toValue(stepKey),toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, enabled: computed(() => toValue(stepKey) !== null && toValue(stepKey) !== undefined), ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof analyticsControlChartRead>>, TError, TData>
}

export type AnalyticsControlChartReadQueryResult = NonNullable<Awaited<ReturnType<typeof analyticsControlChartRead>>>
export type AnalyticsControlChartReadQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Контрольная карта узла
 */

export function useAnalyticsControlChartRead<TData = Awaited<ReturnType<typeof analyticsControlChartRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 stepKey: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<AnalyticsControlChartReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof analyticsControlChartRead>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getAnalyticsControlChartReadQueryOptions(stepKey,params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type analyticsMetricDrilldownResponse200 = {
  data: MetricDrilldown
  status: 200
}

export type analyticsMetricDrilldownResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type analyticsMetricDrilldownResponseSuccess = (analyticsMetricDrilldownResponse200) & {
  headers: Headers;
};
export type analyticsMetricDrilldownResponseError = (analyticsMetricDrilldownResponseDefault) & {
  headers: Headers;
};

export const getAnalyticsMetricDrilldownUrl = (metricId: string,
    params?: AnalyticsMetricDrilldownParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/analytics/metrics/${metricId}/contributions?${stringifiedParams}` : `/api/v1/analytics/metrics/${metricId}/contributions`
}

/**
 * AD-45, соглашение «Показатели»: число раскрывается до строк вклада изделий и id исходных записей журнала.
 * @summary Раскрытие показателя
 */
export const analyticsMetricDrilldown = async (metricId: string,
    params?: AnalyticsMetricDrilldownParams, options?: RequestInit): Promise<analyticsMetricDrilldownResponseSuccess> => {

  const res = await fetch(getAnalyticsMetricDrilldownUrl(metricId,params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: analyticsMetricDrilldownResponseError['data'], status?: number} = new globalThis.Error();
    const data : analyticsMetricDrilldownResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: analyticsMetricDrilldownResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as analyticsMetricDrilldownResponseSuccess
}





export const getAnalyticsMetricDrilldownQueryKey = (metricId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<AnalyticsMetricDrilldownParams>,) => {
    return [
    'api','v1','analytics','metrics',metricId,'contributions', ...(params ? [params] : [])
    ] as const;
    }


export const getAnalyticsMetricDrilldownQueryOptions = <TData = Awaited<ReturnType<typeof analyticsMetricDrilldown>>, TError = globalThis.Error & { info?: Problem; status?: number }>(metricId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<AnalyticsMetricDrilldownParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof analyticsMetricDrilldown>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getAnalyticsMetricDrilldownQueryKey(metricId,params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof analyticsMetricDrilldown>>> = ({ signal }) => analyticsMetricDrilldown(toValue(metricId),toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, enabled: computed(() => toValue(metricId) !== null && toValue(metricId) !== undefined), ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof analyticsMetricDrilldown>>, TError, TData>
}

export type AnalyticsMetricDrilldownQueryResult = NonNullable<Awaited<ReturnType<typeof analyticsMetricDrilldown>>>
export type AnalyticsMetricDrilldownQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Раскрытие показателя
 */

export function useAnalyticsMetricDrilldown<TData = Awaited<ReturnType<typeof analyticsMetricDrilldown>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 metricId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<AnalyticsMetricDrilldownParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof analyticsMetricDrilldown>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getAnalyticsMetricDrilldownQueryOptions(metricId,params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type visionPassportAdmitResponse200 = {
  data: Receipt
  status: 200
}

export type visionPassportAdmitResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type visionPassportAdmitResponseSuccess = (visionPassportAdmitResponse200) & {
  headers: Headers;
};
export type visionPassportAdmitResponseError = (visionPassportAdmitResponseDefault) & {
  headers: Headers;
};

export const getVisionPassportAdmitUrl = () => {




  return `/api/v1/analyzer-passports`
}

/**
 * FR-98: допуск по закрытому маршруту протокола допуска (начальник ОТК + технолог + метролог) — разрешающее действие, изменение контроля (AD-28).
 * @summary Допустить версию анализатора
 */
export const visionPassportAdmit = async (admitPassport: AdmitPassport, options?: RequestInit): Promise<visionPassportAdmitResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getVisionPassportAdmitUrl(),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(admitPassport)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: visionPassportAdmitResponseError['data'], status?: number} = new globalThis.Error();
    const data : visionPassportAdmitResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: visionPassportAdmitResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as visionPassportAdmitResponseSuccess
}





export const getVisionPassportAdmitMutationKey = () => ['visionPassportAdmit'] as const;

export const getVisionPassportAdmitMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof visionPassportAdmit>>, TError,VisionPassportAdmitMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof visionPassportAdmit>>, TError,VisionPassportAdmitMutationVariables, TContext> => {

const mutationKey = getVisionPassportAdmitMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof visionPassportAdmit>>, VisionPassportAdmitMutationVariables> = (props) => {
          const {data} = props ?? {};

          return  visionPassportAdmit(data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type VisionPassportAdmitMutationResult = NonNullable<Awaited<ReturnType<typeof visionPassportAdmit>>>
    export type VisionPassportAdmitMutationBody = AdmitPassport
    export type VisionPassportAdmitMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type VisionPassportAdmitMutationVariables = {data: AdmitPassport}

    /**
 * @summary Допустить версию анализатора
 */
export const useVisionPassportAdmit = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof visionPassportAdmit>>, TError,VisionPassportAdmitMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof visionPassportAdmit>>,
        TError,
        VisionPassportAdmitMutationVariables,
        TContext
      > => {
      return useMutation(getVisionPassportAdmitMutationOptions(options), queryClient);
    }

export type visionPassportReadResponse200 = {
  data: AnalyzerPassport
  status: 200
}

export type visionPassportReadResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type visionPassportReadResponseSuccess = (visionPassportReadResponse200) & {
  headers: Headers;
};
export type visionPassportReadResponseError = (visionPassportReadResponseDefault) & {
  headers: Headers;
};

export const getVisionPassportReadUrl = (passportId: string,
    params?: VisionPassportReadParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/analyzer-passports/${passportId}?${stringifiedParams}` : `/api/v1/analyzer-passports/${passportId}`
}

/**
 * FR-98: паспорт допуска карты контроля — стадия, уровень доверия и допустимые автоматические действия, приостановка (откат) и её триггер.
 * @summary Паспорт допуска
 */
export const visionPassportRead = async (passportId: string,
    params?: VisionPassportReadParams, options?: RequestInit): Promise<visionPassportReadResponseSuccess> => {

  const res = await fetch(getVisionPassportReadUrl(passportId,params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: visionPassportReadResponseError['data'], status?: number} = new globalThis.Error();
    const data : visionPassportReadResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: visionPassportReadResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as visionPassportReadResponseSuccess
}





export const getVisionPassportReadQueryKey = (passportId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<VisionPassportReadParams>,) => {
    return [
    'api','v1','analyzer-passports',passportId, ...(params ? [params] : [])
    ] as const;
    }


export const getVisionPassportReadQueryOptions = <TData = Awaited<ReturnType<typeof visionPassportRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(passportId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<VisionPassportReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof visionPassportRead>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getVisionPassportReadQueryKey(passportId,params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof visionPassportRead>>> = ({ signal }) => visionPassportRead(toValue(passportId),toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, enabled: computed(() => toValue(passportId) !== null && toValue(passportId) !== undefined), ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof visionPassportRead>>, TError, TData>
}

export type VisionPassportReadQueryResult = NonNullable<Awaited<ReturnType<typeof visionPassportRead>>>
export type VisionPassportReadQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Паспорт допуска
 */

export function useVisionPassportRead<TData = Awaited<ReturnType<typeof visionPassportRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 passportId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<VisionPassportReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof visionPassportRead>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getVisionPassportReadQueryOptions(passportId,params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type visionCheckListResponse200 = {
  data: AnalyzerCheckList
  status: 200
}

export type visionCheckListResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type visionCheckListResponseSuccess = (visionCheckListResponse200) & {
  headers: Headers;
};
export type visionCheckListResponseError = (visionCheckListResponseDefault) & {
  headers: Headers;
};

export const getVisionCheckListUrl = (passportId: string,
    params?: VisionCheckListParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/analyzer-passports/${passportId}/checks?${stringifiedParams}` : `/api/v1/analyzer-passports/${passportId}/checks`
}

/**
 * FR-99, FR-100: экзамен, эталонный набор, теневое сравнение, контроль дрейфа — доли пропусков, ложных тревог и расхождений с людьми в б. п.
 * @summary Отчёты проверки анализатора
 */
export const visionCheckList = async (passportId: string,
    params?: VisionCheckListParams, options?: RequestInit): Promise<visionCheckListResponseSuccess> => {

  const res = await fetch(getVisionCheckListUrl(passportId,params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: visionCheckListResponseError['data'], status?: number} = new globalThis.Error();
    const data : visionCheckListResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: visionCheckListResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as visionCheckListResponseSuccess
}





export const getVisionCheckListQueryKey = (passportId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<VisionCheckListParams>,) => {
    return [
    'api','v1','analyzer-passports',passportId,'checks', ...(params ? [params] : [])
    ] as const;
    }


export const getVisionCheckListQueryOptions = <TData = Awaited<ReturnType<typeof visionCheckList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(passportId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<VisionCheckListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof visionCheckList>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getVisionCheckListQueryKey(passportId,params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof visionCheckList>>> = ({ signal }) => visionCheckList(toValue(passportId),toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, enabled: computed(() => toValue(passportId) !== null && toValue(passportId) !== undefined), ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof visionCheckList>>, TError, TData>
}

export type VisionCheckListQueryResult = NonNullable<Awaited<ReturnType<typeof visionCheckList>>>
export type VisionCheckListQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Отчёты проверки анализатора
 */

export function useVisionCheckList<TData = Awaited<ReturnType<typeof visionCheckList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 passportId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<VisionCheckListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof visionCheckList>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getVisionCheckListQueryOptions(passportId,params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type visionPassportReinstateResponse200 = {
  data: Receipt
  status: 200
}

export type visionPassportReinstateResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type visionPassportReinstateResponseSuccess = (visionPassportReinstateResponse200) & {
  headers: Headers;
};
export type visionPassportReinstateResponseError = (visionPassportReinstateResponseDefault) & {
  headers: Headers;
};

export const getVisionPassportReinstateUrl = (passportId: string,) => {




  return `/api/v1/analyzer-passports/${passportId}/reinstate`
}

/**
 * FR-101, AD-29: возврат после приостановки — разрешающее действие начальника ОТК (analyzer.reinstate_requires_head_of_qc).
 * @summary Вернуть анализатор после отката
 */
export const visionPassportReinstate = async (passportId: string,
    reinstatePassport: ReinstatePassport, options?: RequestInit): Promise<visionPassportReinstateResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getVisionPassportReinstateUrl(passportId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(reinstatePassport)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: visionPassportReinstateResponseError['data'], status?: number} = new globalThis.Error();
    const data : visionPassportReinstateResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: visionPassportReinstateResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as visionPassportReinstateResponseSuccess
}





export const getVisionPassportReinstateMutationKey = () => ['visionPassportReinstate'] as const;

export const getVisionPassportReinstateMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof visionPassportReinstate>>, TError,VisionPassportReinstateMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof visionPassportReinstate>>, TError,VisionPassportReinstateMutationVariables, TContext> => {

const mutationKey = getVisionPassportReinstateMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof visionPassportReinstate>>, VisionPassportReinstateMutationVariables> = (props) => {
          const {passportId,data} = props ?? {};

          return  visionPassportReinstate(passportId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type VisionPassportReinstateMutationResult = NonNullable<Awaited<ReturnType<typeof visionPassportReinstate>>>
    export type VisionPassportReinstateMutationBody = ReinstatePassport
    export type VisionPassportReinstateMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type VisionPassportReinstateMutationVariables = {passportId: string;data: ReinstatePassport}

    /**
 * @summary Вернуть анализатор после отката
 */
export const useVisionPassportReinstate = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof visionPassportReinstate>>, TError,VisionPassportReinstateMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof visionPassportReinstate>>,
        TError,
        VisionPassportReinstateMutationVariables,
        TContext
      > => {
      return useMutation(getVisionPassportReinstateMutationOptions(options), queryClient);
    }

export type visionPassportRetireResponse200 = {
  data: Receipt
  status: 200
}

export type visionPassportRetireResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type visionPassportRetireResponseSuccess = (visionPassportRetireResponse200) & {
  headers: Headers;
};
export type visionPassportRetireResponseError = (visionPassportRetireResponseDefault) & {
  headers: Headers;
};

export const getVisionPassportRetireUrl = (passportId: string,) => {




  return `/api/v1/analyzer-passports/${passportId}/retire`
}

/**
 * Паспорт выводится; старые наблюдения сохраняют «проанализировано версией …».
 * @summary Вывести паспорт из действия
 */
export const visionPassportRetire = async (passportId: string,
    retirePassport: RetirePassport, options?: RequestInit): Promise<visionPassportRetireResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getVisionPassportRetireUrl(passportId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(retirePassport)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: visionPassportRetireResponseError['data'], status?: number} = new globalThis.Error();
    const data : visionPassportRetireResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: visionPassportRetireResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as visionPassportRetireResponseSuccess
}





export const getVisionPassportRetireMutationKey = () => ['visionPassportRetire'] as const;

export const getVisionPassportRetireMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof visionPassportRetire>>, TError,VisionPassportRetireMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof visionPassportRetire>>, TError,VisionPassportRetireMutationVariables, TContext> => {

const mutationKey = getVisionPassportRetireMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof visionPassportRetire>>, VisionPassportRetireMutationVariables> = (props) => {
          const {passportId,data} = props ?? {};

          return  visionPassportRetire(passportId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type VisionPassportRetireMutationResult = NonNullable<Awaited<ReturnType<typeof visionPassportRetire>>>
    export type VisionPassportRetireMutationBody = RetirePassport
    export type VisionPassportRetireMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type VisionPassportRetireMutationVariables = {passportId: string;data: RetirePassport}

    /**
 * @summary Вывести паспорт из действия
 */
export const useVisionPassportRetire = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof visionPassportRetire>>, TError,VisionPassportRetireMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof visionPassportRetire>>,
        TError,
        VisionPassportRetireMutationVariables,
        TContext
      > => {
      return useMutation(getVisionPassportRetireMutationOptions(options), queryClient);
    }

export type visionAnalyzerListResponse200 = {
  data: AnalyzerList
  status: 200
}

export type visionAnalyzerListResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type visionAnalyzerListResponseSuccess = (visionAnalyzerListResponse200) & {
  headers: Headers;
};
export type visionAnalyzerListResponseError = (visionAnalyzerListResponseDefault) & {
  headers: Headers;
};

export const getVisionAnalyzerListUrl = (params?: VisionAnalyzerListParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/analyzers?${stringifiedParams}` : `/api/v1/analyzers`
}

/**
 * FR-97, FR-126: анализаторы визуального контроля и контроля действий оператора — версии, стадия допуска, уровень доверия паспорта (AD-29).
 * @summary Анализаторы
 */
export const visionAnalyzerList = async (params?: VisionAnalyzerListParams, options?: RequestInit): Promise<visionAnalyzerListResponseSuccess> => {

  const res = await fetch(getVisionAnalyzerListUrl(params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: visionAnalyzerListResponseError['data'], status?: number} = new globalThis.Error();
    const data : visionAnalyzerListResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: visionAnalyzerListResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as visionAnalyzerListResponseSuccess
}





export const getVisionAnalyzerListQueryKey = (params?: MaybeRefOrGetter<VisionAnalyzerListParams>,) => {
    return [
    'api','v1','analyzers', ...(params ? [params] : [])
    ] as const;
    }


export const getVisionAnalyzerListQueryOptions = <TData = Awaited<ReturnType<typeof visionAnalyzerList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(params?: MaybeRefOrGetter<VisionAnalyzerListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof visionAnalyzerList>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getVisionAnalyzerListQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof visionAnalyzerList>>> = ({ signal }) => visionAnalyzerList(toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof visionAnalyzerList>>, TError, TData>
}

export type VisionAnalyzerListQueryResult = NonNullable<Awaited<ReturnType<typeof visionAnalyzerList>>>
export type VisionAnalyzerListQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Анализаторы
 */

export function useVisionAnalyzerList<TData = Awaited<ReturnType<typeof visionAnalyzerList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 params?: MaybeRefOrGetter<VisionAnalyzerListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof visionAnalyzerList>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getVisionAnalyzerListQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type notificationsAttentionListResponse200 = {
  data: AttentionList
  status: 200
}

export type notificationsAttentionListResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type notificationsAttentionListResponseSuccess = (notificationsAttentionListResponse200) & {
  headers: Headers;
};
export type notificationsAttentionListResponseError = (notificationsAttentionListResponseDefault) & {
  headers: Headers;
};

export const getNotificationsAttentionListUrl = (params?: NotificationsAttentionListParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/attention?${stringifiedParams}` : `/api/v1/attention`
}

/**
 * FR-8: просроченные решения с ценой задержки («просрочено на 37 мин — стоят 18 изделий, 2 операции»), меры без подтверждённой эффективности, временные меры без достигнутого условия выхода. Блок поверх живой карты стола руководителя.
 * @summary Требует вашего внимания
 */
export const notificationsAttentionList = async (params?: NotificationsAttentionListParams, options?: RequestInit): Promise<notificationsAttentionListResponseSuccess> => {

  const res = await fetch(getNotificationsAttentionListUrl(params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: notificationsAttentionListResponseError['data'], status?: number} = new globalThis.Error();
    const data : notificationsAttentionListResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: notificationsAttentionListResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as notificationsAttentionListResponseSuccess
}





export const getNotificationsAttentionListQueryKey = (params?: MaybeRefOrGetter<NotificationsAttentionListParams>,) => {
    return [
    'api','v1','attention', ...(params ? [params] : [])
    ] as const;
    }


export const getNotificationsAttentionListQueryOptions = <TData = Awaited<ReturnType<typeof notificationsAttentionList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(params?: MaybeRefOrGetter<NotificationsAttentionListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof notificationsAttentionList>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getNotificationsAttentionListQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof notificationsAttentionList>>> = ({ signal }) => notificationsAttentionList(toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof notificationsAttentionList>>, TError, TData>
}

export type NotificationsAttentionListQueryResult = NonNullable<Awaited<ReturnType<typeof notificationsAttentionList>>>
export type NotificationsAttentionListQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Требует вашего внимания
 */

export function useNotificationsAttentionList<TData = Awaited<ReturnType<typeof notificationsAttentionList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 params?: MaybeRefOrGetter<NotificationsAttentionListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof notificationsAttentionList>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getNotificationsAttentionListQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type accessPersonaListResponse200 = {
  data: DemoPersonaList
  status: 200
}

export type accessPersonaListResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type accessPersonaListResponseSuccess = (accessPersonaListResponse200) & {
  headers: Headers;
};
export type accessPersonaListResponseError = (accessPersonaListResponseDefault) & {
  headers: Headers;
};

export const getAccessPersonaListUrl = () => {




  return `/api/v1/auth/personas`
}

/**
 * Демо-трек (эпик 08): экран входа предлагает выбрать демо-персону — псевдоним из стартовой политики (normative/policy) с ролью и областью. Вне профилей fixtures и demo операция отвечает 404 api.not_found, и экран показывает только вход по логину.
 * @summary Демо-персоны для входа без пароля
 */
export const accessPersonaList = async ( options?: RequestInit): Promise<accessPersonaListResponseSuccess> => {

  const res = await fetch(getAccessPersonaListUrl(),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: accessPersonaListResponseError['data'], status?: number} = new globalThis.Error();
    const data : accessPersonaListResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: accessPersonaListResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as accessPersonaListResponseSuccess
}





export const getAccessPersonaListQueryKey = () => {
    return [
    'api','v1','auth','personas'
    ] as const;
    }


export const getAccessPersonaListQueryOptions = <TData = Awaited<ReturnType<typeof accessPersonaList>>, TError = globalThis.Error & { info?: Problem; status?: number }>( options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof accessPersonaList>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getAccessPersonaListQueryKey();



    const queryFn: QueryFunction<Awaited<ReturnType<typeof accessPersonaList>>> = ({ signal }) => accessPersonaList({ signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof accessPersonaList>>, TError, TData>
}

export type AccessPersonaListQueryResult = NonNullable<Awaited<ReturnType<typeof accessPersonaList>>>
export type AccessPersonaListQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Демо-персоны для входа без пароля
 */

export function useAccessPersonaList<TData = Awaited<ReturnType<typeof accessPersonaList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof accessPersonaList>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getAccessPersonaListQueryOptions(options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type accessSessionDeleteResponse204 = {
  data: void
  status: 204
}

export type accessSessionDeleteResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 204>
}

export type accessSessionDeleteResponseSuccess = (accessSessionDeleteResponse204) & {
  headers: Headers;
};
export type accessSessionDeleteResponseError = (accessSessionDeleteResponseDefault) & {
  headers: Headers;
};

export const getAccessSessionDeleteUrl = () => {




  return `/api/v1/auth/session`
}

/**
 * Закрыть сеанс веба (FR-128).
 * @summary Выйти
 */
export const accessSessionDelete = async ( options?: RequestInit): Promise<accessSessionDeleteResponseSuccess> => {

  const res = await fetch(getAccessSessionDeleteUrl(),
  {
    ...options,
    method: 'DELETE'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: accessSessionDeleteResponseError['data'], status?: number} = new globalThis.Error();
    const data : accessSessionDeleteResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: accessSessionDeleteResponseSuccess['data'] = body ? JSON.parse(body) : undefined
  return { data, status: res.status, headers: res.headers } as accessSessionDeleteResponseSuccess
}





export const getAccessSessionDeleteMutationKey = () => ['accessSessionDelete'] as const;

export const getAccessSessionDeleteMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof accessSessionDelete>>, TError,void, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof accessSessionDelete>>, TError,void, TContext> => {

const mutationKey = getAccessSessionDeleteMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof accessSessionDelete>>, void> = () => {


          return  accessSessionDelete(fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type AccessSessionDeleteMutationResult = NonNullable<Awaited<ReturnType<typeof accessSessionDelete>>>

    export type AccessSessionDeleteMutationError = globalThis.Error & { info?: Problem; status?: number }


    /**
 * @summary Выйти
 */
export const useAccessSessionDelete = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof accessSessionDelete>>, TError,void, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof accessSessionDelete>>,
        TError,
        void,
        TContext
      > => {
      return useMutation(getAccessSessionDeleteMutationOptions(options), queryClient);
    }

export type accessSessionReadResponse200 = {
  data: Session
  status: 200
}

export type accessSessionReadResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type accessSessionReadResponseSuccess = (accessSessionReadResponse200) & {
  headers: Headers;
};
export type accessSessionReadResponseError = (accessSessionReadResponseDefault) & {
  headers: Headers;
};

export const getAccessSessionReadUrl = () => {




  return `/api/v1/auth/session`
}

/**
 * Пользователь, активная роль, область, смена, рабочее место, версия политики (FR-128); 401 access.unauthenticated — сеанса нет.
 * @summary Текущий сеанс
 */
export const accessSessionRead = async ( options?: RequestInit): Promise<accessSessionReadResponseSuccess> => {

  const res = await fetch(getAccessSessionReadUrl(),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: accessSessionReadResponseError['data'], status?: number} = new globalThis.Error();
    const data : accessSessionReadResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: accessSessionReadResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as accessSessionReadResponseSuccess
}





export const getAccessSessionReadQueryKey = () => {
    return [
    'api','v1','auth','session'
    ] as const;
    }


export const getAccessSessionReadQueryOptions = <TData = Awaited<ReturnType<typeof accessSessionRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>( options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof accessSessionRead>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getAccessSessionReadQueryKey();



    const queryFn: QueryFunction<Awaited<ReturnType<typeof accessSessionRead>>> = ({ signal }) => accessSessionRead({ signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof accessSessionRead>>, TError, TData>
}

export type AccessSessionReadQueryResult = NonNullable<Awaited<ReturnType<typeof accessSessionRead>>>
export type AccessSessionReadQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Текущий сеанс
 */

export function useAccessSessionRead<TData = Awaited<ReturnType<typeof accessSessionRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof accessSessionRead>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getAccessSessionReadQueryOptions(options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type accessSessionCreateResponse201 = {
  data: Session
  status: 201
}

export type accessSessionCreateResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 201>
}

export type accessSessionCreateResponseSuccess = (accessSessionCreateResponse201) & {
  headers: Headers;
};
export type accessSessionCreateResponseError = (accessSessionCreateResponseDefault) & {
  headers: Headers;
};

export const getAccessSessionCreateUrl = () => {




  return `/api/v1/auth/session`
}

/**
 * FR-128. Вход демо-персоной (persona_id, только профили fixtures и demo) или по логину. Пароль пока необязателен (демо-трек); после эпика 08 — обязателен для входа по логину (сеанс scs, argon2id). Ответ ставит cookie сеанса.
 * @summary Войти
 */
export const accessSessionCreate = async (sessionCreate: SessionCreate, options?: RequestInit): Promise<accessSessionCreateResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getAccessSessionCreateUrl(),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(sessionCreate)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: accessSessionCreateResponseError['data'], status?: number} = new globalThis.Error();
    const data : accessSessionCreateResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: accessSessionCreateResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as accessSessionCreateResponseSuccess
}





export const getAccessSessionCreateMutationKey = () => ['accessSessionCreate'] as const;

export const getAccessSessionCreateMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof accessSessionCreate>>, TError,AccessSessionCreateMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof accessSessionCreate>>, TError,AccessSessionCreateMutationVariables, TContext> => {

const mutationKey = getAccessSessionCreateMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof accessSessionCreate>>, AccessSessionCreateMutationVariables> = (props) => {
          const {data} = props ?? {};

          return  accessSessionCreate(data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type AccessSessionCreateMutationResult = NonNullable<Awaited<ReturnType<typeof accessSessionCreate>>>
    export type AccessSessionCreateMutationBody = SessionCreate
    export type AccessSessionCreateMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type AccessSessionCreateMutationVariables = {data: SessionCreate}

    /**
 * @summary Войти
 */
export const useAccessSessionCreate = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof accessSessionCreate>>, TError,AccessSessionCreateMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof accessSessionCreate>>,
        TError,
        AccessSessionCreateMutationVariables,
        TContext
      > => {
      return useMutation(getAccessSessionCreateMutationOptions(options), queryClient);
    }

export type nonconformityConcessionListResponse200 = {
  data: ConcessionList
  status: 200
}

export type nonconformityConcessionListResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type nonconformityConcessionListResponseSuccess = (nonconformityConcessionListResponse200) & {
  headers: Headers;
};
export type nonconformityConcessionListResponseError = (nonconformityConcessionListResponseDefault) & {
  headers: Headers;
};

export const getNonconformityConcessionListUrl = (params?: NonconformityConcessionListParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/concessions?${stringifiedParams}` : `/api/v1/concessions`
}

/**
 * FR-54: действующие разрешения на отклонение, применимые к изделию, с лимитом и остатком — для выбора при решении «ремонт» или «как есть».
 * @summary Разрешения на отклонение
 */
export const nonconformityConcessionList = async (params?: NonconformityConcessionListParams, options?: RequestInit): Promise<nonconformityConcessionListResponseSuccess> => {

  const res = await fetch(getNonconformityConcessionListUrl(params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: nonconformityConcessionListResponseError['data'], status?: number} = new globalThis.Error();
    const data : nonconformityConcessionListResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: nonconformityConcessionListResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as nonconformityConcessionListResponseSuccess
}





export const getNonconformityConcessionListQueryKey = (params?: MaybeRefOrGetter<NonconformityConcessionListParams>,) => {
    return [
    'api','v1','concessions', ...(params ? [params] : [])
    ] as const;
    }


export const getNonconformityConcessionListQueryOptions = <TData = Awaited<ReturnType<typeof nonconformityConcessionList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(params?: MaybeRefOrGetter<NonconformityConcessionListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof nonconformityConcessionList>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getNonconformityConcessionListQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof nonconformityConcessionList>>> = ({ signal }) => nonconformityConcessionList(toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof nonconformityConcessionList>>, TError, TData>
}

export type NonconformityConcessionListQueryResult = NonNullable<Awaited<ReturnType<typeof nonconformityConcessionList>>>
export type NonconformityConcessionListQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Разрешения на отклонение
 */

export function useNonconformityConcessionList<TData = Awaited<ReturnType<typeof nonconformityConcessionList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 params?: MaybeRefOrGetter<NonconformityConcessionListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof nonconformityConcessionList>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getNonconformityConcessionListQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type nonconformityConcessionRevokeResponse200 = {
  data: Receipt
  status: 200
}

export type nonconformityConcessionRevokeResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type nonconformityConcessionRevokeResponseSuccess = (nonconformityConcessionRevokeResponse200) & {
  headers: Headers;
};
export type nonconformityConcessionRevokeResponseError = (nonconformityConcessionRevokeResponseDefault) & {
  headers: Headers;
};

export const getNonconformityConcessionRevokeUrl = (concessionId: string,) => {




  return `/api/v1/concessions/${concessionId}/revoke`
}

/**
 * FR-54: отзыв разрешения — защитное действие; расход лимита прекращается.
 * @summary Отозвать разрешение на отклонение
 */
export const nonconformityConcessionRevoke = async (concessionId: string,
    revokeConcession: RevokeConcession, options?: RequestInit): Promise<nonconformityConcessionRevokeResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getNonconformityConcessionRevokeUrl(concessionId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(revokeConcession)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: nonconformityConcessionRevokeResponseError['data'], status?: number} = new globalThis.Error();
    const data : nonconformityConcessionRevokeResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: nonconformityConcessionRevokeResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as nonconformityConcessionRevokeResponseSuccess
}





export const getNonconformityConcessionRevokeMutationKey = () => ['nonconformityConcessionRevoke'] as const;

export const getNonconformityConcessionRevokeMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof nonconformityConcessionRevoke>>, TError,NonconformityConcessionRevokeMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof nonconformityConcessionRevoke>>, TError,NonconformityConcessionRevokeMutationVariables, TContext> => {

const mutationKey = getNonconformityConcessionRevokeMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof nonconformityConcessionRevoke>>, NonconformityConcessionRevokeMutationVariables> = (props) => {
          const {concessionId,data} = props ?? {};

          return  nonconformityConcessionRevoke(concessionId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type NonconformityConcessionRevokeMutationResult = NonNullable<Awaited<ReturnType<typeof nonconformityConcessionRevoke>>>
    export type NonconformityConcessionRevokeMutationBody = RevokeConcession
    export type NonconformityConcessionRevokeMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type NonconformityConcessionRevokeMutationVariables = {concessionId: string;data: RevokeConcession}

    /**
 * @summary Отозвать разрешение на отклонение
 */
export const useNonconformityConcessionRevoke = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof nonconformityConcessionRevoke>>, TError,NonconformityConcessionRevokeMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof nonconformityConcessionRevoke>>,
        TError,
        NonconformityConcessionRevokeMutationVariables,
        TContext
      > => {
      return useMutation(getNonconformityConcessionRevokeMutationOptions(options), queryClient);
    }

export type nonconformityQueueListResponse200 = {
  data: DecisionQueue
  status: 200
}

export type nonconformityQueueListResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type nonconformityQueueListResponseSuccess = (nonconformityQueueListResponse200) & {
  headers: Headers;
};
export type nonconformityQueueListResponseError = (nonconformityQueueListResponseDefault) & {
  headers: Headers;
};

export const getNonconformityQueueListUrl = (params?: NonconformityQueueListParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/decision-queue?${stringifiedParams}` : `/api/v1/decision-queue`
}

/**
 * PRD §3a «Контролёр качества»: точки предъявления, сигналы на рассмотрение, изолированные изделия со сроком решения (обратный отсчёт); сортировка по риску и сроку.
 * @summary Очередь «Ждут моего решения»
 */
export const nonconformityQueueList = async (params?: NonconformityQueueListParams, options?: RequestInit): Promise<nonconformityQueueListResponseSuccess> => {

  const res = await fetch(getNonconformityQueueListUrl(params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: nonconformityQueueListResponseError['data'], status?: number} = new globalThis.Error();
    const data : nonconformityQueueListResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: nonconformityQueueListResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as nonconformityQueueListResponseSuccess
}





export const getNonconformityQueueListQueryKey = (params?: MaybeRefOrGetter<NonconformityQueueListParams>,) => {
    return [
    'api','v1','decision-queue', ...(params ? [params] : [])
    ] as const;
    }


export const getNonconformityQueueListQueryOptions = <TData = Awaited<ReturnType<typeof nonconformityQueueList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(params?: MaybeRefOrGetter<NonconformityQueueListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof nonconformityQueueList>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getNonconformityQueueListQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof nonconformityQueueList>>> = ({ signal }) => nonconformityQueueList(toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof nonconformityQueueList>>, TError, TData>
}

export type NonconformityQueueListQueryResult = NonNullable<Awaited<ReturnType<typeof nonconformityQueueList>>>
export type NonconformityQueueListQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Очередь «Ждут моего решения»
 */

export function useNonconformityQueueList<TData = Awaited<ReturnType<typeof nonconformityQueueList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 params?: MaybeRefOrGetter<NonconformityQueueListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof nonconformityQueueList>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getNonconformityQueueListQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type qualityDefectListResponse200 = {
  data: QualityDefectList
  status: 200
}

export type qualityDefectListResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type qualityDefectListResponseSuccess = (qualityDefectListResponse200) & {
  headers: Headers;
};
export type qualityDefectListResponseError = (qualityDefectListResponseDefault) & {
  headers: Headers;
};

export const getQualityDefectListUrl = (params?: QualityDefectListParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/defects?${stringifiedParams}` : `/api/v1/defects`
}

/**
 * FR-37: физические дефекты (ключ — изделие, зона и место, без вида дефекта); дефекты и изделия с дефектами — раздельно.
 * @summary Дефекты
 */
export const qualityDefectList = async (params?: QualityDefectListParams, options?: RequestInit): Promise<qualityDefectListResponseSuccess> => {

  const res = await fetch(getQualityDefectListUrl(params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: qualityDefectListResponseError['data'], status?: number} = new globalThis.Error();
    const data : qualityDefectListResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: qualityDefectListResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as qualityDefectListResponseSuccess
}





export const getQualityDefectListQueryKey = (params?: MaybeRefOrGetter<QualityDefectListParams>,) => {
    return [
    'api','v1','defects', ...(params ? [params] : [])
    ] as const;
    }


export const getQualityDefectListQueryOptions = <TData = Awaited<ReturnType<typeof qualityDefectList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(params?: MaybeRefOrGetter<QualityDefectListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof qualityDefectList>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getQualityDefectListQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof qualityDefectList>>> = ({ signal }) => qualityDefectList(toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof qualityDefectList>>, TError, TData>
}

export type QualityDefectListQueryResult = NonNullable<Awaited<ReturnType<typeof qualityDefectList>>>
export type QualityDefectListQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Дефекты
 */

export function useQualityDefectList<TData = Awaited<ReturnType<typeof qualityDefectList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 params?: MaybeRefOrGetter<QualityDefectListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof qualityDefectList>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getQualityDefectListQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type accessDeskReadResponse200 = {
  data: Desk
  status: 200
}

export type accessDeskReadResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type accessDeskReadResponseSuccess = (accessDeskReadResponse200) & {
  headers: Headers;
};
export type accessDeskReadResponseError = (accessDeskReadResponseDefault) & {
  headers: Headers;
};

export const getAccessDeskReadUrl = () => {




  return `/api/v1/desk`
}

/**
 * AD-21: стол — данные normative/desks/‹роль›.yaml (раскладка → вкладки → слоты → виджеты → срез и плотность), отдаётся через access.Queries.Desks. Для роли-наследника без своего файла — стол ближайшей базовой роли (inherits).
 * @summary Стол активной роли
 */
export const accessDeskRead = async ( options?: RequestInit): Promise<accessDeskReadResponseSuccess> => {

  const res = await fetch(getAccessDeskReadUrl(),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: accessDeskReadResponseError['data'], status?: number} = new globalThis.Error();
    const data : accessDeskReadResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: accessDeskReadResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as accessDeskReadResponseSuccess
}





export const getAccessDeskReadQueryKey = () => {
    return [
    'api','v1','desk'
    ] as const;
    }


export const getAccessDeskReadQueryOptions = <TData = Awaited<ReturnType<typeof accessDeskRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>( options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof accessDeskRead>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getAccessDeskReadQueryKey();



    const queryFn: QueryFunction<Awaited<ReturnType<typeof accessDeskRead>>> = ({ signal }) => accessDeskRead({ signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof accessDeskRead>>, TError, TData>
}

export type AccessDeskReadQueryResult = NonNullable<Awaited<ReturnType<typeof accessDeskRead>>>
export type AccessDeskReadQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Стол активной роли
 */

export function useAccessDeskRead<TData = Awaited<ReturnType<typeof accessDeskRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof accessDeskRead>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getAccessDeskReadQueryOptions(options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type machinelogsEquipmentListResponse200 = {
  data: EquipmentList
  status: 200
}

export type machinelogsEquipmentListResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type machinelogsEquipmentListResponseSuccess = (machinelogsEquipmentListResponse200) & {
  headers: Headers;
};
export type machinelogsEquipmentListResponseError = (machinelogsEquipmentListResponseDefault) & {
  headers: Headers;
};

export const getMachinelogsEquipmentListUrl = (params?: MachinelogsEquipmentListParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/equipment?${stringifiedParams}` : `/api/v1/equipment`
}

/**
 * Стол мастера «Люди и оборудование»: режим, исправность, программа, инструмент и его ресурс, поверка на дату, предупреждения.
 * @summary Оборудование
 */
export const machinelogsEquipmentList = async (params?: MachinelogsEquipmentListParams, options?: RequestInit): Promise<machinelogsEquipmentListResponseSuccess> => {

  const res = await fetch(getMachinelogsEquipmentListUrl(params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: machinelogsEquipmentListResponseError['data'], status?: number} = new globalThis.Error();
    const data : machinelogsEquipmentListResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: machinelogsEquipmentListResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as machinelogsEquipmentListResponseSuccess
}





export const getMachinelogsEquipmentListQueryKey = (params?: MaybeRefOrGetter<MachinelogsEquipmentListParams>,) => {
    return [
    'api','v1','equipment', ...(params ? [params] : [])
    ] as const;
    }


export const getMachinelogsEquipmentListQueryOptions = <TData = Awaited<ReturnType<typeof machinelogsEquipmentList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(params?: MaybeRefOrGetter<MachinelogsEquipmentListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof machinelogsEquipmentList>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getMachinelogsEquipmentListQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof machinelogsEquipmentList>>> = ({ signal }) => machinelogsEquipmentList(toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof machinelogsEquipmentList>>, TError, TData>
}

export type MachinelogsEquipmentListQueryResult = NonNullable<Awaited<ReturnType<typeof machinelogsEquipmentList>>>
export type MachinelogsEquipmentListQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Оборудование
 */

export function useMachinelogsEquipmentList<TData = Awaited<ReturnType<typeof machinelogsEquipmentList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 params?: MaybeRefOrGetter<MachinelogsEquipmentListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof machinelogsEquipmentList>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getMachinelogsEquipmentListQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type machinelogsViolationListResponse200 = {
  data: ViolationList
  status: 200
}

export type machinelogsViolationListResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type machinelogsViolationListResponseSuccess = (machinelogsViolationListResponse200) & {
  headers: Headers;
};
export type machinelogsViolationListResponseError = (machinelogsViolationListResponseDefault) & {
  headers: Headers;
};

export const getMachinelogsViolationListUrl = (params?: MachinelogsViolationListParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/equipment/violations?${stringifiedParams}` : `/api/v1/equipment/violations`
}

/**
 * FR-151: нарушение режима на операции со специальным процессом — несоответствие для всех изделий окна, даже без найденного дефекта; решение — комиссией.
 * @summary Окна нарушений специального процесса
 */
export const machinelogsViolationList = async (params?: MachinelogsViolationListParams, options?: RequestInit): Promise<machinelogsViolationListResponseSuccess> => {

  const res = await fetch(getMachinelogsViolationListUrl(params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: machinelogsViolationListResponseError['data'], status?: number} = new globalThis.Error();
    const data : machinelogsViolationListResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: machinelogsViolationListResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as machinelogsViolationListResponseSuccess
}





export const getMachinelogsViolationListQueryKey = (params?: MaybeRefOrGetter<MachinelogsViolationListParams>,) => {
    return [
    'api','v1','equipment','violations', ...(params ? [params] : [])
    ] as const;
    }


export const getMachinelogsViolationListQueryOptions = <TData = Awaited<ReturnType<typeof machinelogsViolationList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(params?: MaybeRefOrGetter<MachinelogsViolationListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof machinelogsViolationList>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getMachinelogsViolationListQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof machinelogsViolationList>>> = ({ signal }) => machinelogsViolationList(toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof machinelogsViolationList>>, TError, TData>
}

export type MachinelogsViolationListQueryResult = NonNullable<Awaited<ReturnType<typeof machinelogsViolationList>>>
export type MachinelogsViolationListQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Окна нарушений специального процесса
 */

export function useMachinelogsViolationList<TData = Awaited<ReturnType<typeof machinelogsViolationList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 params?: MaybeRefOrGetter<MachinelogsViolationListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof machinelogsViolationList>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getMachinelogsViolationListQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type machinelogsEquipmentReadResponse200 = {
  data: EquipmentState
  status: 200
}

export type machinelogsEquipmentReadResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type machinelogsEquipmentReadResponseSuccess = (machinelogsEquipmentReadResponse200) & {
  headers: Headers;
};
export type machinelogsEquipmentReadResponseError = (machinelogsEquipmentReadResponseDefault) & {
  headers: Headers;
};

export const getMachinelogsEquipmentReadUrl = (equipmentId: string,
    params?: MachinelogsEquipmentReadParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/equipment/${equipmentId}?${stringifiedParams}` : `/api/v1/equipment/${equipmentId}`
}

/**
 * Состояние по классификации MTConnect (AD-29), поверка, предупреждения.
 * @summary Карточка оборудования
 */
export const machinelogsEquipmentRead = async (equipmentId: string,
    params?: MachinelogsEquipmentReadParams, options?: RequestInit): Promise<machinelogsEquipmentReadResponseSuccess> => {

  const res = await fetch(getMachinelogsEquipmentReadUrl(equipmentId,params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: machinelogsEquipmentReadResponseError['data'], status?: number} = new globalThis.Error();
    const data : machinelogsEquipmentReadResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: machinelogsEquipmentReadResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as machinelogsEquipmentReadResponseSuccess
}





export const getMachinelogsEquipmentReadQueryKey = (equipmentId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<MachinelogsEquipmentReadParams>,) => {
    return [
    'api','v1','equipment',equipmentId, ...(params ? [params] : [])
    ] as const;
    }


export const getMachinelogsEquipmentReadQueryOptions = <TData = Awaited<ReturnType<typeof machinelogsEquipmentRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(equipmentId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<MachinelogsEquipmentReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof machinelogsEquipmentRead>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getMachinelogsEquipmentReadQueryKey(equipmentId,params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof machinelogsEquipmentRead>>> = ({ signal }) => machinelogsEquipmentRead(toValue(equipmentId),toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, enabled: computed(() => toValue(equipmentId) !== null && toValue(equipmentId) !== undefined), ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof machinelogsEquipmentRead>>, TError, TData>
}

export type MachinelogsEquipmentReadQueryResult = NonNullable<Awaited<ReturnType<typeof machinelogsEquipmentRead>>>
export type MachinelogsEquipmentReadQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Карточка оборудования
 */

export function useMachinelogsEquipmentRead<TData = Awaited<ReturnType<typeof machinelogsEquipmentRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 equipmentId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<MachinelogsEquipmentReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof machinelogsEquipmentRead>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getMachinelogsEquipmentReadQueryOptions(equipmentId,params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type machinelogsTimelineReadResponse200 = {
  data: EquipmentTimeline
  status: 200
}

export type machinelogsTimelineReadResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type machinelogsTimelineReadResponseSuccess = (machinelogsTimelineReadResponse200) & {
  headers: Headers;
};
export type machinelogsTimelineReadResponseError = (machinelogsTimelineReadResponseDefault) & {
  headers: Headers;
};

export const getMachinelogsTimelineReadUrl = (equipmentId: string,
    params: MachinelogsTimelineReadParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/equipment/${equipmentId}/timeline?${stringifiedParams}` : `/api/v1/equipment/${equipmentId}/timeline`
}

/**
 * FR-121, AD-29: четыре слоя — что делал станок, чем, как шёл процесс (сводки на окно цикла), отклонения; сырые данные остаются на краю (FR-147).
 * @summary Журнал оборудования на окне
 */
export const machinelogsTimelineRead = async (equipmentId: string,
    params: MachinelogsTimelineReadParams, options?: RequestInit): Promise<machinelogsTimelineReadResponseSuccess> => {

  const res = await fetch(getMachinelogsTimelineReadUrl(equipmentId,params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: machinelogsTimelineReadResponseError['data'], status?: number} = new globalThis.Error();
    const data : machinelogsTimelineReadResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: machinelogsTimelineReadResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as machinelogsTimelineReadResponseSuccess
}





export const getMachinelogsTimelineReadQueryKey = (equipmentId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<MachinelogsTimelineReadParams>,) => {
    return [
    'api','v1','equipment',equipmentId,'timeline', ...(params ? [params] : [])
    ] as const;
    }


export const getMachinelogsTimelineReadQueryOptions = <TData = Awaited<ReturnType<typeof machinelogsTimelineRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(equipmentId: MaybeRefOrGetter<string>,
    params: MaybeRefOrGetter<MachinelogsTimelineReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof machinelogsTimelineRead>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getMachinelogsTimelineReadQueryKey(equipmentId,params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof machinelogsTimelineRead>>> = ({ signal }) => machinelogsTimelineRead(toValue(equipmentId),toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, enabled: computed(() => toValue(equipmentId) !== null && toValue(equipmentId) !== undefined), ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof machinelogsTimelineRead>>, TError, TData>
}

export type MachinelogsTimelineReadQueryResult = NonNullable<Awaited<ReturnType<typeof machinelogsTimelineRead>>>
export type MachinelogsTimelineReadQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Журнал оборудования на окне
 */

export function useMachinelogsTimelineRead<TData = Awaited<ReturnType<typeof machinelogsTimelineRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 equipmentId: MaybeRefOrGetter<string>,
    params: MaybeRefOrGetter<MachinelogsTimelineReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof machinelogsTimelineRead>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getMachinelogsTimelineReadQueryOptions(equipmentId,params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type erpChannelListResponse200 = {
  data: ErpChannelList
  status: 200
}

export type erpChannelListResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type erpChannelListResponseSuccess = (erpChannelListResponse200) & {
  headers: Headers;
};
export type erpChannelListResponseError = (erpChannelListResponseDefault) & {
  headers: Headers;
};

export const getErpChannelListUrl = () => {




  return `/api/v1/erp/channels`
}

/**
 * AD-18: при старте адаптер сверяет метаданные внешней системы ($metadata 1С) и версию контракта; расхождение — degraded, отправки нет.
 * @summary Каналы обмена
 */
export const erpChannelList = async ( options?: RequestInit): Promise<erpChannelListResponseSuccess> => {

  const res = await fetch(getErpChannelListUrl(),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: erpChannelListResponseError['data'], status?: number} = new globalThis.Error();
    const data : erpChannelListResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: erpChannelListResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as erpChannelListResponseSuccess
}





export const getErpChannelListQueryKey = () => {
    return [
    'api','v1','erp','channels'
    ] as const;
    }


export const getErpChannelListQueryOptions = <TData = Awaited<ReturnType<typeof erpChannelList>>, TError = globalThis.Error & { info?: Problem; status?: number }>( options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof erpChannelList>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getErpChannelListQueryKey();



    const queryFn: QueryFunction<Awaited<ReturnType<typeof erpChannelList>>> = ({ signal }) => erpChannelList({ signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof erpChannelList>>, TError, TData>
}

export type ErpChannelListQueryResult = NonNullable<Awaited<ReturnType<typeof erpChannelList>>>
export type ErpChannelListQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Каналы обмена
 */

export function useErpChannelList<TData = Awaited<ReturnType<typeof erpChannelList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof erpChannelList>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getErpChannelListQueryOptions(options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type erpMessageListResponse200 = {
  data: ErpMessageList
  status: 200
}

export type erpMessageListResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type erpMessageListResponseSuccess = (erpMessageListResponse200) & {
  headers: Headers;
};
export type erpMessageListResponseError = (erpMessageListResponseDefault) & {
  headers: Headers;
};

export const getErpMessageListUrl = (params?: ErpMessageListParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/erp/messages?${stringifiedParams}` : `/api/v1/erp/messages`
}

/**
 * AD-7, AD-18: «принято в работу», «смена склада», «перевод в брак», «возврат поставщику», «выпуск», «результат контроля» — со статусом отправки и квитанцией; ось «учёт в 1С» меняется только квитанцией.
 * @summary Исходящие учётные сообщения
 */
export const erpMessageList = async (params?: ErpMessageListParams, options?: RequestInit): Promise<erpMessageListResponseSuccess> => {

  const res = await fetch(getErpMessageListUrl(params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: erpMessageListResponseError['data'], status?: number} = new globalThis.Error();
    const data : erpMessageListResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: erpMessageListResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as erpMessageListResponseSuccess
}





export const getErpMessageListQueryKey = (params?: MaybeRefOrGetter<ErpMessageListParams>,) => {
    return [
    'api','v1','erp','messages', ...(params ? [params] : [])
    ] as const;
    }


export const getErpMessageListQueryOptions = <TData = Awaited<ReturnType<typeof erpMessageList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(params?: MaybeRefOrGetter<ErpMessageListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof erpMessageList>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getErpMessageListQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof erpMessageList>>> = ({ signal }) => erpMessageList(toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof erpMessageList>>, TError, TData>
}

export type ErpMessageListQueryResult = NonNullable<Awaited<ReturnType<typeof erpMessageList>>>
export type ErpMessageListQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Исходящие учётные сообщения
 */

export function useErpMessageList<TData = Awaited<ReturnType<typeof erpMessageList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 params?: MaybeRefOrGetter<ErpMessageListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof erpMessageList>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getErpMessageListQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type erpMessageReadResponse200 = {
  data: ErpMessage
  status: 200
}

export type erpMessageReadResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type erpMessageReadResponseSuccess = (erpMessageReadResponse200) & {
  headers: Headers;
};
export type erpMessageReadResponseError = (erpMessageReadResponseDefault) & {
  headers: Headers;
};

export const getErpMessageReadUrl = (businessKey: string,
    params?: ErpMessageReadParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/erp/messages/${businessKey}?${stringifiedParams}` : `/api/v1/erp/messages/${businessKey}`
}

/**
 * Версии сообщения с одним бизнес-ключом, попытки, квитанции и ошибки.
 * @summary Учётное сообщение
 */
export const erpMessageRead = async (businessKey: string,
    params?: ErpMessageReadParams, options?: RequestInit): Promise<erpMessageReadResponseSuccess> => {

  const res = await fetch(getErpMessageReadUrl(businessKey,params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: erpMessageReadResponseError['data'], status?: number} = new globalThis.Error();
    const data : erpMessageReadResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: erpMessageReadResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as erpMessageReadResponseSuccess
}





export const getErpMessageReadQueryKey = (businessKey: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<ErpMessageReadParams>,) => {
    return [
    'api','v1','erp','messages',businessKey, ...(params ? [params] : [])
    ] as const;
    }


export const getErpMessageReadQueryOptions = <TData = Awaited<ReturnType<typeof erpMessageRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(businessKey: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<ErpMessageReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof erpMessageRead>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getErpMessageReadQueryKey(businessKey,params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof erpMessageRead>>> = ({ signal }) => erpMessageRead(toValue(businessKey),toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, enabled: computed(() => toValue(businessKey) !== null && toValue(businessKey) !== undefined), ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof erpMessageRead>>, TError, TData>
}

export type ErpMessageReadQueryResult = NonNullable<Awaited<ReturnType<typeof erpMessageRead>>>
export type ErpMessageReadQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Учётное сообщение
 */

export function useErpMessageRead<TData = Awaited<ReturnType<typeof erpMessageRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 businessKey: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<ErpMessageReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof erpMessageRead>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getErpMessageReadQueryOptions(businessKey,params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type erpPostingCompensateResponse200 = {
  data: Receipt
  status: 200
}

export type erpPostingCompensateResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type erpPostingCompensateResponseSuccess = (erpPostingCompensateResponse200) & {
  headers: Headers;
};
export type erpPostingCompensateResponseError = (erpPostingCompensateResponseDefault) & {
  headers: Headers;
};

export const getErpPostingCompensateUrl = (businessKey: string,) => {




  return `/api/v1/erp/messages/${businessKey}/compensation`
}

/**
 * AD-7: новая версия реакции с тем же бизнес-ключом и другим содержимым даёт исправление (сторно + новое) только по решению человека.
 * @summary Решение о компенсации
 */
export const erpPostingCompensate = async (businessKey: string,
    compensatePosting: CompensatePosting, options?: RequestInit): Promise<erpPostingCompensateResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getErpPostingCompensateUrl(businessKey),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(compensatePosting)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: erpPostingCompensateResponseError['data'], status?: number} = new globalThis.Error();
    const data : erpPostingCompensateResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: erpPostingCompensateResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as erpPostingCompensateResponseSuccess
}





export const getErpPostingCompensateMutationKey = () => ['erpPostingCompensate'] as const;

export const getErpPostingCompensateMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof erpPostingCompensate>>, TError,ErpPostingCompensateMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof erpPostingCompensate>>, TError,ErpPostingCompensateMutationVariables, TContext> => {

const mutationKey = getErpPostingCompensateMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof erpPostingCompensate>>, ErpPostingCompensateMutationVariables> = (props) => {
          const {businessKey,data} = props ?? {};

          return  erpPostingCompensate(businessKey,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type ErpPostingCompensateMutationResult = NonNullable<Awaited<ReturnType<typeof erpPostingCompensate>>>
    export type ErpPostingCompensateMutationBody = CompensatePosting
    export type ErpPostingCompensateMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type ErpPostingCompensateMutationVariables = {businessKey: string;data: CompensatePosting}

    /**
 * @summary Решение о компенсации
 */
export const useErpPostingCompensate = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof erpPostingCompensate>>, TError,ErpPostingCompensateMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof erpPostingCompensate>>,
        TError,
        ErpPostingCompensateMutationVariables,
        TContext
      > => {
      return useMutation(getErpPostingCompensateMutationOptions(options), queryClient);
    }

export type erpPostingResendResponse200 = {
  data: Receipt
  status: 200
}

export type erpPostingResendResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type erpPostingResendResponseSuccess = (erpPostingResendResponse200) & {
  headers: Headers;
};
export type erpPostingResendResponseError = (erpPostingResendResponseDefault) & {
  headers: Headers;
};

export const getErpPostingResendUrl = (businessKey: string,) => {




  return `/api/v1/erp/messages/${businessKey}/resend`
}

/**
 * AD-18: повтор — только при транспортных ошибках, затем карантин и ручная переотправка администратором.
 * @summary Переотправить сообщение
 */
export const erpPostingResend = async (businessKey: string,
    resendPosting: ResendPosting, options?: RequestInit): Promise<erpPostingResendResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getErpPostingResendUrl(businessKey),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(resendPosting)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: erpPostingResendResponseError['data'], status?: number} = new globalThis.Error();
    const data : erpPostingResendResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: erpPostingResendResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as erpPostingResendResponseSuccess
}





export const getErpPostingResendMutationKey = () => ['erpPostingResend'] as const;

export const getErpPostingResendMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof erpPostingResend>>, TError,ErpPostingResendMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof erpPostingResend>>, TError,ErpPostingResendMutationVariables, TContext> => {

const mutationKey = getErpPostingResendMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof erpPostingResend>>, ErpPostingResendMutationVariables> = (props) => {
          const {businessKey,data} = props ?? {};

          return  erpPostingResend(businessKey,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type ErpPostingResendMutationResult = NonNullable<Awaited<ReturnType<typeof erpPostingResend>>>
    export type ErpPostingResendMutationBody = ResendPosting
    export type ErpPostingResendMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type ErpPostingResendMutationVariables = {businessKey: string;data: ResendPosting}

    /**
 * @summary Переотправить сообщение
 */
export const useErpPostingResend = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof erpPostingResend>>, TError,ErpPostingResendMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof erpPostingResend>>,
        TError,
        ErpPostingResendMutationVariables,
        TContext
      > => {
      return useMutation(getErpPostingResendMutationOptions(options), queryClient);
    }

export type erpOrderListResponse200 = {
  data: ErpOrderList
  status: 200
}

export type erpOrderListResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type erpOrderListResponseSuccess = (erpOrderListResponse200) & {
  headers: Headers;
};
export type erpOrderListResponseError = (erpOrderListResponseDefault) & {
  headers: Headers;
};

export const getErpOrderListUrl = (params?: ErpOrderListParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/erp/orders?${stringifiedParams}` : `/api/v1/erp/orders`
}

/**
 * Кейс §1.5: производственные задания из 1С и Галактики (erp.order.received) и сколько изделий по ним запущено.
 * @summary Задания учётных систем
 */
export const erpOrderList = async (params?: ErpOrderListParams, options?: RequestInit): Promise<erpOrderListResponseSuccess> => {

  const res = await fetch(getErpOrderListUrl(params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: erpOrderListResponseError['data'], status?: number} = new globalThis.Error();
    const data : erpOrderListResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: erpOrderListResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as erpOrderListResponseSuccess
}





export const getErpOrderListQueryKey = (params?: MaybeRefOrGetter<ErpOrderListParams>,) => {
    return [
    'api','v1','erp','orders', ...(params ? [params] : [])
    ] as const;
    }


export const getErpOrderListQueryOptions = <TData = Awaited<ReturnType<typeof erpOrderList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(params?: MaybeRefOrGetter<ErpOrderListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof erpOrderList>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getErpOrderListQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof erpOrderList>>> = ({ signal }) => erpOrderList(toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof erpOrderList>>, TError, TData>
}

export type ErpOrderListQueryResult = NonNullable<Awaited<ReturnType<typeof erpOrderList>>>
export type ErpOrderListQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Задания учётных систем
 */

export function useErpOrderList<TData = Awaited<ReturnType<typeof erpOrderList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 params?: MaybeRefOrGetter<ErpOrderListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof erpOrderList>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getErpOrderListQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type qualityEscapeListResponse200 = {
  data: QualityEscapeList
  status: 200
}

export type qualityEscapeListResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type qualityEscapeListResponseSuccess = (qualityEscapeListResponse200) & {
  headers: Headers;
};
export type qualityEscapeListResponseError = (qualityEscapeListResponseDefault) & {
  headers: Headers;
};

export const getQualityEscapeListUrl = (params?: QualityEscapeListParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/escapes?${stringifiedParams}` : `/api/v1/escapes`
}

/**
 * Пропущенный брак (quality.escape.recorded) — возврат в контур адаптации анализатора (FR-99).
 * @summary Пропуски брака
 */
export const qualityEscapeList = async (params?: QualityEscapeListParams, options?: RequestInit): Promise<qualityEscapeListResponseSuccess> => {

  const res = await fetch(getQualityEscapeListUrl(params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: qualityEscapeListResponseError['data'], status?: number} = new globalThis.Error();
    const data : qualityEscapeListResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: qualityEscapeListResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as qualityEscapeListResponseSuccess
}





export const getQualityEscapeListQueryKey = (params?: MaybeRefOrGetter<QualityEscapeListParams>,) => {
    return [
    'api','v1','escapes', ...(params ? [params] : [])
    ] as const;
    }


export const getQualityEscapeListQueryOptions = <TData = Awaited<ReturnType<typeof qualityEscapeList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(params?: MaybeRefOrGetter<QualityEscapeListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof qualityEscapeList>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getQualityEscapeListQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof qualityEscapeList>>> = ({ signal }) => qualityEscapeList(toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof qualityEscapeList>>, TError, TData>
}

export type QualityEscapeListQueryResult = NonNullable<Awaited<ReturnType<typeof qualityEscapeList>>>
export type QualityEscapeListQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Пропуски брака
 */

export function useQualityEscapeList<TData = Awaited<ReturnType<typeof qualityEscapeList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 params?: MaybeRefOrGetter<QualityEscapeListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof qualityEscapeList>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getQualityEscapeListQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type analysisIncidentListResponse200 = {
  data: IncidentList
  status: 200
}

export type analysisIncidentListResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type analysisIncidentListResponseSuccess = (analysisIncidentListResponse200) & {
  headers: Headers;
};
export type analysisIncidentListResponseError = (analysisIncidentListResponseDefault) & {
  headers: Headers;
};

export const getAnalysisIncidentListUrl = (params?: AnalysisIncidentListParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/incidents?${stringifiedParams}` : `/api/v1/incidents`
}

/**
 * Инциденты с общим фактором, размером и версией области риска (FR-61).
 * @summary Инциденты
 */
export const analysisIncidentList = async (params?: AnalysisIncidentListParams, options?: RequestInit): Promise<analysisIncidentListResponseSuccess> => {

  const res = await fetch(getAnalysisIncidentListUrl(params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: analysisIncidentListResponseError['data'], status?: number} = new globalThis.Error();
    const data : analysisIncidentListResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: analysisIncidentListResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as analysisIncidentListResponseSuccess
}





export const getAnalysisIncidentListQueryKey = (params?: MaybeRefOrGetter<AnalysisIncidentListParams>,) => {
    return [
    'api','v1','incidents', ...(params ? [params] : [])
    ] as const;
    }


export const getAnalysisIncidentListQueryOptions = <TData = Awaited<ReturnType<typeof analysisIncidentList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(params?: MaybeRefOrGetter<AnalysisIncidentListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof analysisIncidentList>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getAnalysisIncidentListQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof analysisIncidentList>>> = ({ signal }) => analysisIncidentList(toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof analysisIncidentList>>, TError, TData>
}

export type AnalysisIncidentListQueryResult = NonNullable<Awaited<ReturnType<typeof analysisIncidentList>>>
export type AnalysisIncidentListQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Инциденты
 */

export function useAnalysisIncidentList<TData = Awaited<ReturnType<typeof analysisIncidentList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 params?: MaybeRefOrGetter<AnalysisIncidentListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof analysisIncidentList>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getAnalysisIncidentListQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type analysisActionAssignResponse200 = {
  data: Receipt
  status: 200
}

export type analysisActionAssignResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type analysisActionAssignResponseSuccess = (analysisActionAssignResponse200) & {
  headers: Headers;
};
export type analysisActionAssignResponseError = (analysisActionAssignResponseDefault) & {
  headers: Headers;
};

export const getAnalysisActionAssignUrl = (incidentId: string,) => {




  return `/api/v1/incidents/${incidentId}/actions`
}

/**
 * FR-64: коррекция, корректирующее или предупреждающее действие с планом проверки эффективности.
 * @summary Назначить корректирующее действие
 */
export const analysisActionAssign = async (incidentId: string,
    assignAction: AssignAction, options?: RequestInit): Promise<analysisActionAssignResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getAnalysisActionAssignUrl(incidentId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(assignAction)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: analysisActionAssignResponseError['data'], status?: number} = new globalThis.Error();
    const data : analysisActionAssignResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: analysisActionAssignResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as analysisActionAssignResponseSuccess
}





export const getAnalysisActionAssignMutationKey = () => ['analysisActionAssign'] as const;

export const getAnalysisActionAssignMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof analysisActionAssign>>, TError,AnalysisActionAssignMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof analysisActionAssign>>, TError,AnalysisActionAssignMutationVariables, TContext> => {

const mutationKey = getAnalysisActionAssignMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof analysisActionAssign>>, AnalysisActionAssignMutationVariables> = (props) => {
          const {incidentId,data} = props ?? {};

          return  analysisActionAssign(incidentId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type AnalysisActionAssignMutationResult = NonNullable<Awaited<ReturnType<typeof analysisActionAssign>>>
    export type AnalysisActionAssignMutationBody = AssignAction
    export type AnalysisActionAssignMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type AnalysisActionAssignMutationVariables = {incidentId: string;data: AssignAction}

    /**
 * @summary Назначить корректирующее действие
 */
export const useAnalysisActionAssign = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof analysisActionAssign>>, TError,AnalysisActionAssignMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof analysisActionAssign>>,
        TError,
        AnalysisActionAssignMutationVariables,
        TContext
      > => {
      return useMutation(getAnalysisActionAssignMutationOptions(options), queryClient);
    }

export type analysisActionEvaluateResponse200 = {
  data: Receipt
  status: 200
}

export type analysisActionEvaluateResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type analysisActionEvaluateResponseSuccess = (analysisActionEvaluateResponse200) & {
  headers: Headers;
};
export type analysisActionEvaluateResponseError = (analysisActionEvaluateResponseDefault) & {
  headers: Headers;
};

export const getAnalysisActionEvaluateUrl = (incidentId: string,
    actionId: string,) => {




  return `/api/v1/incidents/${incidentId}/actions/${actionId}/evaluation`
}

/**
 * FR-64: эффективно / неэффективно по плану проверки.
 * @summary Оценить эффективность действия
 */
export const analysisActionEvaluate = async (incidentId: string,
    actionId: string,
    evaluateAction: EvaluateAction, options?: RequestInit): Promise<analysisActionEvaluateResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getAnalysisActionEvaluateUrl(incidentId,actionId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(evaluateAction)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: analysisActionEvaluateResponseError['data'], status?: number} = new globalThis.Error();
    const data : analysisActionEvaluateResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: analysisActionEvaluateResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as analysisActionEvaluateResponseSuccess
}





export const getAnalysisActionEvaluateMutationKey = () => ['analysisActionEvaluate'] as const;

export const getAnalysisActionEvaluateMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof analysisActionEvaluate>>, TError,AnalysisActionEvaluateMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof analysisActionEvaluate>>, TError,AnalysisActionEvaluateMutationVariables, TContext> => {

const mutationKey = getAnalysisActionEvaluateMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof analysisActionEvaluate>>, AnalysisActionEvaluateMutationVariables> = (props) => {
          const {incidentId,actionId,data} = props ?? {};

          return  analysisActionEvaluate(incidentId,actionId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type AnalysisActionEvaluateMutationResult = NonNullable<Awaited<ReturnType<typeof analysisActionEvaluate>>>
    export type AnalysisActionEvaluateMutationBody = EvaluateAction
    export type AnalysisActionEvaluateMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type AnalysisActionEvaluateMutationVariables = {incidentId: string;actionId: string;data: EvaluateAction}

    /**
 * @summary Оценить эффективность действия
 */
export const useAnalysisActionEvaluate = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof analysisActionEvaluate>>, TError,AnalysisActionEvaluateMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof analysisActionEvaluate>>,
        TError,
        AnalysisActionEvaluateMutationVariables,
        TContext
      > => {
      return useMutation(getAnalysisActionEvaluateMutationOptions(options), queryClient);
    }

export type analysisActionImplementResponse200 = {
  data: Receipt
  status: 200
}

export type analysisActionImplementResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type analysisActionImplementResponseSuccess = (analysisActionImplementResponse200) & {
  headers: Headers;
};
export type analysisActionImplementResponseError = (analysisActionImplementResponseDefault) & {
  headers: Headers;
};

export const getAnalysisActionImplementUrl = (incidentId: string,
    actionId: string,) => {




  return `/api/v1/incidents/${incidentId}/actions/${actionId}/implemented`
}

/**
 * FR-64.
 * @summary Отметить действие выполненным
 */
export const analysisActionImplement = async (incidentId: string,
    actionId: string,
    implementAction: ImplementAction, options?: RequestInit): Promise<analysisActionImplementResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getAnalysisActionImplementUrl(incidentId,actionId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(implementAction)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: analysisActionImplementResponseError['data'], status?: number} = new globalThis.Error();
    const data : analysisActionImplementResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: analysisActionImplementResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as analysisActionImplementResponseSuccess
}





export const getAnalysisActionImplementMutationKey = () => ['analysisActionImplement'] as const;

export const getAnalysisActionImplementMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof analysisActionImplement>>, TError,AnalysisActionImplementMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof analysisActionImplement>>, TError,AnalysisActionImplementMutationVariables, TContext> => {

const mutationKey = getAnalysisActionImplementMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof analysisActionImplement>>, AnalysisActionImplementMutationVariables> = (props) => {
          const {incidentId,actionId,data} = props ?? {};

          return  analysisActionImplement(incidentId,actionId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type AnalysisActionImplementMutationResult = NonNullable<Awaited<ReturnType<typeof analysisActionImplement>>>
    export type AnalysisActionImplementMutationBody = ImplementAction
    export type AnalysisActionImplementMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type AnalysisActionImplementMutationVariables = {incidentId: string;actionId: string;data: ImplementAction}

    /**
 * @summary Отметить действие выполненным
 */
export const useAnalysisActionImplement = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof analysisActionImplement>>, TError,AnalysisActionImplementMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof analysisActionImplement>>,
        TError,
        AnalysisActionImplementMutationVariables,
        TContext
      > => {
      return useMutation(getAnalysisActionImplementMutationOptions(options), queryClient);
    }

export type analysisAnalysisScopeResponse200 = {
  data: Receipt
  status: 200
}

export type analysisAnalysisScopeResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type analysisAnalysisScopeResponseSuccess = (analysisAnalysisScopeResponse200) & {
  headers: Headers;
};
export type analysisAnalysisScopeResponseError = (analysisAnalysisScopeResponseDefault) & {
  headers: Headers;
};

export const getAnalysisAnalysisScopeUrl = (incidentId: string,) => {




  return `/api/v1/incidents/${incidentId}/analysis-scope`
}

/**
 * П-02: полный разбор или упрощённый, с основанием.
 * @summary Определить глубину разбора
 */
export const analysisAnalysisScope = async (incidentId: string,
    scopeAnalysis: ScopeAnalysis, options?: RequestInit): Promise<analysisAnalysisScopeResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getAnalysisAnalysisScopeUrl(incidentId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(scopeAnalysis)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: analysisAnalysisScopeResponseError['data'], status?: number} = new globalThis.Error();
    const data : analysisAnalysisScopeResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: analysisAnalysisScopeResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as analysisAnalysisScopeResponseSuccess
}





export const getAnalysisAnalysisScopeMutationKey = () => ['analysisAnalysisScope'] as const;

export const getAnalysisAnalysisScopeMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof analysisAnalysisScope>>, TError,AnalysisAnalysisScopeMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof analysisAnalysisScope>>, TError,AnalysisAnalysisScopeMutationVariables, TContext> => {

const mutationKey = getAnalysisAnalysisScopeMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof analysisAnalysisScope>>, AnalysisAnalysisScopeMutationVariables> = (props) => {
          const {incidentId,data} = props ?? {};

          return  analysisAnalysisScope(incidentId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type AnalysisAnalysisScopeMutationResult = NonNullable<Awaited<ReturnType<typeof analysisAnalysisScope>>>
    export type AnalysisAnalysisScopeMutationBody = ScopeAnalysis
    export type AnalysisAnalysisScopeMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type AnalysisAnalysisScopeMutationVariables = {incidentId: string;data: ScopeAnalysis}

    /**
 * @summary Определить глубину разбора
 */
export const useAnalysisAnalysisScope = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof analysisAnalysisScope>>, TError,AnalysisAnalysisScopeMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof analysisAnalysisScope>>,
        TError,
        AnalysisAnalysisScopeMutationVariables,
        TContext
      > => {
      return useMutation(getAnalysisAnalysisScopeMutationOptions(options), queryClient);
    }

export type analysisCauseConcludeResponse200 = {
  data: Receipt
  status: 200
}

export type analysisCauseConcludeResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type analysisCauseConcludeResponseSuccess = (analysisCauseConcludeResponse200) & {
  headers: Headers;
};
export type analysisCauseConcludeResponseError = (analysisCauseConcludeResponseDefault) & {
  headers: Headers;
};

export const getAnalysisCauseConcludeUrl = (incidentId: string,) => {




  return `/api/v1/incidents/${incidentId}/cause`
}

/**
 * FR-59: подтвердить гипотезу как причину или «причина не установлена» — необратимое инженерное решение уполномоченного (AD-27), критическое действие (AD-28).
 * @summary Подтвердить причину
 */
export const analysisCauseConclude = async (incidentId: string,
    concludeCause: ConcludeCause, options?: RequestInit): Promise<analysisCauseConcludeResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getAnalysisCauseConcludeUrl(incidentId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(concludeCause)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: analysisCauseConcludeResponseError['data'], status?: number} = new globalThis.Error();
    const data : analysisCauseConcludeResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: analysisCauseConcludeResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as analysisCauseConcludeResponseSuccess
}





export const getAnalysisCauseConcludeMutationKey = () => ['analysisCauseConclude'] as const;

export const getAnalysisCauseConcludeMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof analysisCauseConclude>>, TError,AnalysisCauseConcludeMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof analysisCauseConclude>>, TError,AnalysisCauseConcludeMutationVariables, TContext> => {

const mutationKey = getAnalysisCauseConcludeMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof analysisCauseConclude>>, AnalysisCauseConcludeMutationVariables> = (props) => {
          const {incidentId,data} = props ?? {};

          return  analysisCauseConclude(incidentId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type AnalysisCauseConcludeMutationResult = NonNullable<Awaited<ReturnType<typeof analysisCauseConclude>>>
    export type AnalysisCauseConcludeMutationBody = ConcludeCause
    export type AnalysisCauseConcludeMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type AnalysisCauseConcludeMutationVariables = {incidentId: string;data: ConcludeCause}

    /**
 * @summary Подтвердить причину
 */
export const useAnalysisCauseConclude = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof analysisCauseConclude>>, TError,AnalysisCauseConcludeMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof analysisCauseConclude>>,
        TError,
        AnalysisCauseConcludeMutationVariables,
        TContext
      > => {
      return useMutation(getAnalysisCauseConcludeMutationOptions(options), queryClient);
    }

export type analysisIncidentCloseResponse200 = {
  data: Receipt
  status: 200
}

export type analysisIncidentCloseResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type analysisIncidentCloseResponseSuccess = (analysisIncidentCloseResponse200) & {
  headers: Headers;
};
export type analysisIncidentCloseResponseError = (analysisIncidentCloseResponseDefault) & {
  headers: Headers;
};

export const getAnalysisIncidentCloseUrl = (incidentId: string,) => {




  return `/api/v1/incidents/${incidentId}/close`
}

/**
 * Итог: исходный размер области, подтверждено, исключено.
 * @summary Закрыть инцидент
 */
export const analysisIncidentClose = async (incidentId: string,
    closeIncident: CloseIncident, options?: RequestInit): Promise<analysisIncidentCloseResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getAnalysisIncidentCloseUrl(incidentId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(closeIncident)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: analysisIncidentCloseResponseError['data'], status?: number} = new globalThis.Error();
    const data : analysisIncidentCloseResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: analysisIncidentCloseResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as analysisIncidentCloseResponseSuccess
}





export const getAnalysisIncidentCloseMutationKey = () => ['analysisIncidentClose'] as const;

export const getAnalysisIncidentCloseMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof analysisIncidentClose>>, TError,AnalysisIncidentCloseMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof analysisIncidentClose>>, TError,AnalysisIncidentCloseMutationVariables, TContext> => {

const mutationKey = getAnalysisIncidentCloseMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof analysisIncidentClose>>, AnalysisIncidentCloseMutationVariables> = (props) => {
          const {incidentId,data} = props ?? {};

          return  analysisIncidentClose(incidentId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type AnalysisIncidentCloseMutationResult = NonNullable<Awaited<ReturnType<typeof analysisIncidentClose>>>
    export type AnalysisIncidentCloseMutationBody = CloseIncident
    export type AnalysisIncidentCloseMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type AnalysisIncidentCloseMutationVariables = {incidentId: string;data: CloseIncident}

    /**
 * @summary Закрыть инцидент
 */
export const useAnalysisIncidentClose = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof analysisIncidentClose>>, TError,AnalysisIncidentCloseMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof analysisIncidentClose>>,
        TError,
        AnalysisIncidentCloseMutationVariables,
        TContext
      > => {
      return useMutation(getAnalysisIncidentCloseMutationOptions(options), queryClient);
    }

export type analysisItemAssessResponse200 = {
  data: Receipt
  status: 200
}

export type analysisItemAssessResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type analysisItemAssessResponseSuccess = (analysisItemAssessResponse200) & {
  headers: Headers;
};
export type analysisItemAssessResponseError = (analysisItemAssessResponseDefault) & {
  headers: Headers;
};

export const getAnalysisItemAssessUrl = (incidentId: string,) => {




  return `/api/v1/incidents/${incidentId}/items/assess`
}

/**
 * FR-62: подтверждено / исключено по доказательствам; исключение — разрешающее действие (AD-27).
 * @summary Оценить изделие в инциденте
 */
export const analysisItemAssess = async (incidentId: string,
    assessItem: AssessItem, options?: RequestInit): Promise<analysisItemAssessResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getAnalysisItemAssessUrl(incidentId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(assessItem)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: analysisItemAssessResponseError['data'], status?: number} = new globalThis.Error();
    const data : analysisItemAssessResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: analysisItemAssessResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as analysisItemAssessResponseSuccess
}





export const getAnalysisItemAssessMutationKey = () => ['analysisItemAssess'] as const;

export const getAnalysisItemAssessMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof analysisItemAssess>>, TError,AnalysisItemAssessMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof analysisItemAssess>>, TError,AnalysisItemAssessMutationVariables, TContext> => {

const mutationKey = getAnalysisItemAssessMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof analysisItemAssess>>, AnalysisItemAssessMutationVariables> = (props) => {
          const {incidentId,data} = props ?? {};

          return  analysisItemAssess(incidentId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type AnalysisItemAssessMutationResult = NonNullable<Awaited<ReturnType<typeof analysisItemAssess>>>
    export type AnalysisItemAssessMutationBody = AssessItem
    export type AnalysisItemAssessMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type AnalysisItemAssessMutationVariables = {incidentId: string;data: AssessItem}

    /**
 * @summary Оценить изделие в инциденте
 */
export const useAnalysisItemAssess = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof analysisItemAssess>>, TError,AnalysisItemAssessMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof analysisItemAssess>>,
        TError,
        AnalysisItemAssessMutationVariables,
        TContext
      > => {
      return useMutation(getAnalysisItemAssessMutationOptions(options), queryClient);
    }

export type analysisRiskScopeReadResponse200 = {
  data: RiskScope
  status: 200
}

export type analysisRiskScopeReadResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type analysisRiskScopeReadResponseSuccess = (analysisRiskScopeReadResponse200) & {
  headers: Headers;
};
export type analysisRiskScopeReadResponseError = (analysisRiskScopeReadResponseDefault) & {
  headers: Headers;
};

export const getAnalysisRiskScopeReadUrl = (incidentId: string,
    params?: AnalysisRiskScopeReadParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/incidents/${incidentId}/risk-scope?${stringifiedParams}` : `/api/v1/incidents/${incidentId}/risk-scope`
}

/**
 * FR-61, FR-62: версии области (вычислена, расширена, сужена) с основаниями и доказательствами, разбивка «в производстве / ушли дальше / собраны / отгружены», изделия текущей версии с двумя осями: что известно и что делать.
 * @summary Область риска с версиями
 */
export const analysisRiskScopeRead = async (incidentId: string,
    params?: AnalysisRiskScopeReadParams, options?: RequestInit): Promise<analysisRiskScopeReadResponseSuccess> => {

  const res = await fetch(getAnalysisRiskScopeReadUrl(incidentId,params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: analysisRiskScopeReadResponseError['data'], status?: number} = new globalThis.Error();
    const data : analysisRiskScopeReadResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: analysisRiskScopeReadResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as analysisRiskScopeReadResponseSuccess
}





export const getAnalysisRiskScopeReadQueryKey = (incidentId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<AnalysisRiskScopeReadParams>,) => {
    return [
    'api','v1','incidents',incidentId,'risk-scope', ...(params ? [params] : [])
    ] as const;
    }


export const getAnalysisRiskScopeReadQueryOptions = <TData = Awaited<ReturnType<typeof analysisRiskScopeRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(incidentId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<AnalysisRiskScopeReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof analysisRiskScopeRead>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getAnalysisRiskScopeReadQueryKey(incidentId,params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof analysisRiskScopeRead>>> = ({ signal }) => analysisRiskScopeRead(toValue(incidentId),toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, enabled: computed(() => toValue(incidentId) !== null && toValue(incidentId) !== undefined), ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof analysisRiskScopeRead>>, TError, TData>
}

export type AnalysisRiskScopeReadQueryResult = NonNullable<Awaited<ReturnType<typeof analysisRiskScopeRead>>>
export type AnalysisRiskScopeReadQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Область риска с версиями
 */

export function useAnalysisRiskScopeRead<TData = Awaited<ReturnType<typeof analysisRiskScopeRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 incidentId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<AnalysisRiskScopeReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof analysisRiskScopeRead>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getAnalysisRiskScopeReadQueryOptions(incidentId,params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type analysisScopeExpandResponse200 = {
  data: Receipt
  status: 200
}

export type analysisScopeExpandResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type analysisScopeExpandResponseSuccess = (analysisScopeExpandResponse200) & {
  headers: Headers;
};
export type analysisScopeExpandResponseError = (analysisScopeExpandResponseDefault) & {
  headers: Headers;
};

export const getAnalysisScopeExpandUrl = (incidentId: string,) => {




  return `/api/v1/incidents/${incidentId}/scope/expand`
}

/**
 * FR-61: добавить изделия в область — защитное действие (AD-27); новая версия области.
 * @summary Расширить область риска
 */
export const analysisScopeExpand = async (incidentId: string,
    changeScope: ChangeScope, options?: RequestInit): Promise<analysisScopeExpandResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getAnalysisScopeExpandUrl(incidentId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(changeScope)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: analysisScopeExpandResponseError['data'], status?: number} = new globalThis.Error();
    const data : analysisScopeExpandResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: analysisScopeExpandResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as analysisScopeExpandResponseSuccess
}





export const getAnalysisScopeExpandMutationKey = () => ['analysisScopeExpand'] as const;

export const getAnalysisScopeExpandMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof analysisScopeExpand>>, TError,AnalysisScopeExpandMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof analysisScopeExpand>>, TError,AnalysisScopeExpandMutationVariables, TContext> => {

const mutationKey = getAnalysisScopeExpandMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof analysisScopeExpand>>, AnalysisScopeExpandMutationVariables> = (props) => {
          const {incidentId,data} = props ?? {};

          return  analysisScopeExpand(incidentId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type AnalysisScopeExpandMutationResult = NonNullable<Awaited<ReturnType<typeof analysisScopeExpand>>>
    export type AnalysisScopeExpandMutationBody = ChangeScope
    export type AnalysisScopeExpandMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type AnalysisScopeExpandMutationVariables = {incidentId: string;data: ChangeScope}

    /**
 * @summary Расширить область риска
 */
export const useAnalysisScopeExpand = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof analysisScopeExpand>>, TError,AnalysisScopeExpandMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof analysisScopeExpand>>,
        TError,
        AnalysisScopeExpandMutationVariables,
        TContext
      > => {
      return useMutation(getAnalysisScopeExpandMutationOptions(options), queryClient);
    }

export type analysisScopeNarrowResponse200 = {
  data: Receipt
  status: 200
}

export type analysisScopeNarrowResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type analysisScopeNarrowResponseSuccess = (analysisScopeNarrowResponse200) & {
  headers: Headers;
};
export type analysisScopeNarrowResponseError = (analysisScopeNarrowResponseDefault) & {
  headers: Headers;
};

export const getAnalysisScopeNarrowUrl = (incidentId: string,) => {




  return `/api/v1/incidents/${incidentId}/scope/narrow`
}

/**
 * FR-61: исключить изделия из области по основаниям — разрешающее действие (AD-27), только человек; новая версия области.
 * @summary Сузить область риска
 */
export const analysisScopeNarrow = async (incidentId: string,
    changeScope: ChangeScope, options?: RequestInit): Promise<analysisScopeNarrowResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getAnalysisScopeNarrowUrl(incidentId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(changeScope)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: analysisScopeNarrowResponseError['data'], status?: number} = new globalThis.Error();
    const data : analysisScopeNarrowResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: analysisScopeNarrowResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as analysisScopeNarrowResponseSuccess
}





export const getAnalysisScopeNarrowMutationKey = () => ['analysisScopeNarrow'] as const;

export const getAnalysisScopeNarrowMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof analysisScopeNarrow>>, TError,AnalysisScopeNarrowMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof analysisScopeNarrow>>, TError,AnalysisScopeNarrowMutationVariables, TContext> => {

const mutationKey = getAnalysisScopeNarrowMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof analysisScopeNarrow>>, AnalysisScopeNarrowMutationVariables> = (props) => {
          const {incidentId,data} = props ?? {};

          return  analysisScopeNarrow(incidentId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type AnalysisScopeNarrowMutationResult = NonNullable<Awaited<ReturnType<typeof analysisScopeNarrow>>>
    export type AnalysisScopeNarrowMutationBody = ChangeScope
    export type AnalysisScopeNarrowMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type AnalysisScopeNarrowMutationVariables = {incidentId: string;data: ChangeScope}

    /**
 * @summary Сузить область риска
 */
export const useAnalysisScopeNarrow = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof analysisScopeNarrow>>, TError,AnalysisScopeNarrowMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof analysisScopeNarrow>>,
        TError,
        AnalysisScopeNarrowMutationVariables,
        TContext
      > => {
      return useMutation(getAnalysisScopeNarrowMutationOptions(options), queryClient);
    }

export type ingestBatchSubmitResponse200 = {
  data: IngestResult
  status: 200
}

export type ingestBatchSubmitResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type ingestBatchSubmitResponseSuccess = (ingestBatchSubmitResponse200) & {
  headers: Headers;
};
export type ingestBatchSubmitResponseError = (ingestBatchSubmitResponseDefault) & {
  headers: Headers;
};

export const getIngestBatchSubmitUrl = () => {




  return `/api/v1/ingest/batches`
}

/**
 * FR-26…FR-31, AD-46: подписанные конверты DSSE от edge-агента, шлюзов, терминалов. Сырое тело проверяется теми же JSON Schema (AD-20): неизвестная версия и нет обязательного поля — карантин с кодом; неизвестное значение перечисления — UNKNOWN с флагом (критичное — карантин); повтор — дубль без изменения показателей; другой payload с тем же ключом — конфликт целостности. Ответ — итог по каждому конверту.
 * @summary Принять пачку событий
 */
export const ingestBatchSubmit = async (ingestBatch: IngestBatch, options?: RequestInit): Promise<ingestBatchSubmitResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getIngestBatchSubmitUrl(),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(ingestBatch)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: ingestBatchSubmitResponseError['data'], status?: number} = new globalThis.Error();
    const data : ingestBatchSubmitResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: ingestBatchSubmitResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as ingestBatchSubmitResponseSuccess
}





export const getIngestBatchSubmitMutationKey = () => ['ingestBatchSubmit'] as const;

export const getIngestBatchSubmitMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof ingestBatchSubmit>>, TError,IngestBatchSubmitMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof ingestBatchSubmit>>, TError,IngestBatchSubmitMutationVariables, TContext> => {

const mutationKey = getIngestBatchSubmitMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof ingestBatchSubmit>>, IngestBatchSubmitMutationVariables> = (props) => {
          const {data} = props ?? {};

          return  ingestBatchSubmit(data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type IngestBatchSubmitMutationResult = NonNullable<Awaited<ReturnType<typeof ingestBatchSubmit>>>
    export type IngestBatchSubmitMutationBody = IngestBatch
    export type IngestBatchSubmitMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type IngestBatchSubmitMutationVariables = {data: IngestBatch}

    /**
 * @summary Принять пачку событий
 */
export const useIngestBatchSubmit = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof ingestBatchSubmit>>, TError,IngestBatchSubmitMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof ingestBatchSubmit>>,
        TError,
        IngestBatchSubmitMutationVariables,
        TContext
      > => {
      return useMutation(getIngestBatchSubmitMutationOptions(options), queryClient);
    }

export type ingestEventSubmitResponse200 = {
  data: IngestOutcome
  status: 200
}

export type ingestEventSubmitResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type ingestEventSubmitResponseSuccess = (ingestEventSubmitResponse200) & {
  headers: Headers;
};
export type ingestEventSubmitResponseError = (ingestEventSubmitResponseDefault) & {
  headers: Headers;
};

export const getIngestEventSubmitUrl = () => {




  return `/api/v1/ingest/events`
}

/**
 * FR-137, FR-140, FR-141: одно событие терминала участка или ручного ввода — такой же источник с проверкой входов, дублей и привязки; пометка «ручной ввод» видна в интерфейсе.
 * @summary Принять событие ручного ввода
 */
export const ingestEventSubmit = async (manualEvent: ManualEvent, options?: RequestInit): Promise<ingestEventSubmitResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getIngestEventSubmitUrl(),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(manualEvent)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: ingestEventSubmitResponseError['data'], status?: number} = new globalThis.Error();
    const data : ingestEventSubmitResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: ingestEventSubmitResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as ingestEventSubmitResponseSuccess
}





export const getIngestEventSubmitMutationKey = () => ['ingestEventSubmit'] as const;

export const getIngestEventSubmitMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof ingestEventSubmit>>, TError,IngestEventSubmitMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof ingestEventSubmit>>, TError,IngestEventSubmitMutationVariables, TContext> => {

const mutationKey = getIngestEventSubmitMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof ingestEventSubmit>>, IngestEventSubmitMutationVariables> = (props) => {
          const {data} = props ?? {};

          return  ingestEventSubmit(data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type IngestEventSubmitMutationResult = NonNullable<Awaited<ReturnType<typeof ingestEventSubmit>>>
    export type IngestEventSubmitMutationBody = ManualEvent
    export type IngestEventSubmitMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type IngestEventSubmitMutationVariables = {data: ManualEvent}

    /**
 * @summary Принять событие ручного ввода
 */
export const useIngestEventSubmit = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof ingestEventSubmit>>, TError,IngestEventSubmitMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof ingestEventSubmit>>,
        TError,
        IngestEventSubmitMutationVariables,
        TContext
      > => {
      return useMutation(getIngestEventSubmitMutationOptions(options), queryClient);
    }

export type ingestImportSubmitResponse200 = {
  data: ImportResult
  status: 200
}

export type ingestImportSubmitResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type ingestImportSubmitResponseSuccess = (ingestImportSubmitResponse200) & {
  headers: Headers;
};
export type ingestImportSubmitResponseError = (ingestImportSubmitResponseDefault) & {
  headers: Headers;
};

export const getIngestImportSubmitUrl = () => {




  return `/api/v1/ingest/imports`
}

/**
 * FR-141: файл сохраняется в хранилище материалов по адресу H(байты); строки проходят те же проверки, что события устройств; итог — ingest.import.completed. Идемпотентность — по отпечатку файла и строк.
 * @summary Импорт CSV / Excel
 */
export const ingestImportSubmit = async (importFile: ImportFile, options?: RequestInit): Promise<ingestImportSubmitResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getIngestImportSubmitUrl(),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(importFile)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: ingestImportSubmitResponseError['data'], status?: number} = new globalThis.Error();
    const data : ingestImportSubmitResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: ingestImportSubmitResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as ingestImportSubmitResponseSuccess
}





export const getIngestImportSubmitMutationKey = () => ['ingestImportSubmit'] as const;

export const getIngestImportSubmitMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof ingestImportSubmit>>, TError,IngestImportSubmitMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof ingestImportSubmit>>, TError,IngestImportSubmitMutationVariables, TContext> => {

const mutationKey = getIngestImportSubmitMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof ingestImportSubmit>>, IngestImportSubmitMutationVariables> = (props) => {
          const {data} = props ?? {};

          return  ingestImportSubmit(data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type IngestImportSubmitMutationResult = NonNullable<Awaited<ReturnType<typeof ingestImportSubmit>>>
    export type IngestImportSubmitMutationBody = ImportFile
    export type IngestImportSubmitMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type IngestImportSubmitMutationVariables = {data: ImportFile}

    /**
 * @summary Импорт CSV / Excel
 */
export const useIngestImportSubmit = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof ingestImportSubmit>>, TError,IngestImportSubmitMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof ingestImportSubmit>>,
        TError,
        IngestImportSubmitMutationVariables,
        TContext
      > => {
      return useMutation(getIngestImportSubmitMutationOptions(options), queryClient);
    }

export type ingestMetricsReadResponse200 = {
  data: IngestMetrics
  status: 200
}

export type ingestMetricsReadResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type ingestMetricsReadResponseSuccess = (ingestMetricsReadResponse200) & {
  headers: Headers;
};
export type ingestMetricsReadResponseError = (ingestMetricsReadResponseDefault) & {
  headers: Headers;
};

export const getIngestMetricsReadUrl = () => {




  return `/api/v1/ingest/metrics`
}

/**
 * FR-41: задержка, дубли, отказы, объём карантина, полнота; «событие → экран» (FR-2). Операционные метрики, не проекции (AD-7).
 * @summary Метрики приёма
 */
export const ingestMetricsRead = async ( options?: RequestInit): Promise<ingestMetricsReadResponseSuccess> => {

  const res = await fetch(getIngestMetricsReadUrl(),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: ingestMetricsReadResponseError['data'], status?: number} = new globalThis.Error();
    const data : ingestMetricsReadResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: ingestMetricsReadResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as ingestMetricsReadResponseSuccess
}





export const getIngestMetricsReadQueryKey = () => {
    return [
    'api','v1','ingest','metrics'
    ] as const;
    }


export const getIngestMetricsReadQueryOptions = <TData = Awaited<ReturnType<typeof ingestMetricsRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>( options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof ingestMetricsRead>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getIngestMetricsReadQueryKey();



    const queryFn: QueryFunction<Awaited<ReturnType<typeof ingestMetricsRead>>> = ({ signal }) => ingestMetricsRead({ signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof ingestMetricsRead>>, TError, TData>
}

export type IngestMetricsReadQueryResult = NonNullable<Awaited<ReturnType<typeof ingestMetricsRead>>>
export type IngestMetricsReadQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Метрики приёма
 */

export function useIngestMetricsRead<TData = Awaited<ReturnType<typeof ingestMetricsRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof ingestMetricsRead>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getIngestMetricsReadQueryOptions(options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type securityIntegrityReadResponse200 = {
  data: IntegrityStatus
  status: 200
}

export type securityIntegrityReadResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type securityIntegrityReadResponseSuccess = (securityIntegrityReadResponse200) & {
  headers: Headers;
};
export type securityIntegrityReadResponseError = (securityIntegrityReadResponseDefault) & {
  headers: Headers;
};

export const getSecurityIntegrityReadUrl = () => {




  return `/api/v1/integrity`
}

/**
 * AD-46: ant забирает последний подписанный отчёт верификатора у хранителя и журналирует security.integrity.checked. Индикатор на столах помечен «по данным сервера» и желтеет сам, если свежего отчёта нет дольше двух интервалов.
 * @summary Состояние целостности журнала «по данным сервера»
 */
export const securityIntegrityRead = async ( options?: RequestInit): Promise<securityIntegrityReadResponseSuccess> => {

  const res = await fetch(getSecurityIntegrityReadUrl(),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: securityIntegrityReadResponseError['data'], status?: number} = new globalThis.Error();
    const data : securityIntegrityReadResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: securityIntegrityReadResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as securityIntegrityReadResponseSuccess
}





export const getSecurityIntegrityReadQueryKey = () => {
    return [
    'api','v1','integrity'
    ] as const;
    }


export const getSecurityIntegrityReadQueryOptions = <TData = Awaited<ReturnType<typeof securityIntegrityRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>( options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof securityIntegrityRead>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getSecurityIntegrityReadQueryKey();



    const queryFn: QueryFunction<Awaited<ReturnType<typeof securityIntegrityRead>>> = ({ signal }) => securityIntegrityRead({ signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof securityIntegrityRead>>, TError, TData>
}

export type SecurityIntegrityReadQueryResult = NonNullable<Awaited<ReturnType<typeof securityIntegrityRead>>>
export type SecurityIntegrityReadQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Состояние целостности журнала «по данным сервера»
 */

export function useSecurityIntegrityRead<TData = Awaited<ReturnType<typeof securityIntegrityRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof securityIntegrityRead>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getSecurityIntegrityReadQueryOptions(options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type itemItemListResponse200 = {
  data: ItemList
  status: 200
}

export type itemItemListResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type itemItemListResponseSuccess = (itemItemListResponse200) & {
  headers: Headers;
};
export type itemItemListResponseError = (itemItemListResponseDefault) & {
  headers: Headers;
};

export const getItemItemListUrl = (params?: ItemItemListParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/items?${stringifiedParams}` : `/api/v1/items`
}

/**
 * Проваливание в детали (FR-7): изделия узла карты по step_key, по сводному статусу, партии или заданию 1С; изделия прежних версий — с пометкой версии.
 * @summary Изделия
 */
export const itemItemList = async (params?: ItemItemListParams, options?: RequestInit): Promise<itemItemListResponseSuccess> => {

  const res = await fetch(getItemItemListUrl(params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: itemItemListResponseError['data'], status?: number} = new globalThis.Error();
    const data : itemItemListResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: itemItemListResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as itemItemListResponseSuccess
}





export const getItemItemListQueryKey = (params?: MaybeRefOrGetter<ItemItemListParams>,) => {
    return [
    'api','v1','items', ...(params ? [params] : [])
    ] as const;
    }


export const getItemItemListQueryOptions = <TData = Awaited<ReturnType<typeof itemItemList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(params?: MaybeRefOrGetter<ItemItemListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof itemItemList>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getItemItemListQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof itemItemList>>> = ({ signal }) => itemItemList(toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof itemItemList>>, TError, TData>
}

export type ItemItemListQueryResult = NonNullable<Awaited<ReturnType<typeof itemItemList>>>
export type ItemItemListQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Изделия
 */

export function useItemItemList<TData = Awaited<ReturnType<typeof itemItemList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 params?: MaybeRefOrGetter<ItemItemListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof itemItemList>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getItemItemListQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type itemItemRegisterResponse200 = {
  data: Receipt
  status: 200
}

export type itemItemRegisterResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type itemItemRegisterResponseSuccess = (itemItemRegisterResponse200) & {
  headers: Headers;
};
export type itemItemRegisterResponseError = (itemItemRegisterResponseDefault) & {
  headers: Headers;
};

export const getItemItemRegisterUrl = () => {




  return `/api/v1/items`
}

/**
 * FR-42, AD-16: ID рождается в системе (код_предприятия:локальный_id) и из метки не выводится; закрепляется версия нормативного слоя (AD-17).
 * @summary Зарегистрировать изделие и запустить в работу
 */
export const itemItemRegister = async (registerItem: RegisterItem, options?: RequestInit): Promise<itemItemRegisterResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getItemItemRegisterUrl(),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(registerItem)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: itemItemRegisterResponseError['data'], status?: number} = new globalThis.Error();
    const data : itemItemRegisterResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: itemItemRegisterResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as itemItemRegisterResponseSuccess
}





export const getItemItemRegisterMutationKey = () => ['itemItemRegister'] as const;

export const getItemItemRegisterMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof itemItemRegister>>, TError,ItemItemRegisterMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof itemItemRegister>>, TError,ItemItemRegisterMutationVariables, TContext> => {

const mutationKey = getItemItemRegisterMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof itemItemRegister>>, ItemItemRegisterMutationVariables> = (props) => {
          const {data} = props ?? {};

          return  itemItemRegister(data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type ItemItemRegisterMutationResult = NonNullable<Awaited<ReturnType<typeof itemItemRegister>>>
    export type ItemItemRegisterMutationBody = RegisterItem
    export type ItemItemRegisterMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type ItemItemRegisterMutationVariables = {data: RegisterItem}

    /**
 * @summary Зарегистрировать изделие и запустить в работу
 */
export const useItemItemRegister = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof itemItemRegister>>, TError,ItemItemRegisterMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof itemItemRegister>>,
        TError,
        ItemItemRegisterMutationVariables,
        TContext
      > => {
      return useMutation(getItemItemRegisterMutationOptions(options), queryClient);
    }

export type itemItemLookupResponse200 = {
  data: ItemLookup
  status: 200
}

export type itemItemLookupResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type itemItemLookupResponseSuccess = (itemItemLookupResponse200) & {
  headers: Headers;
};
export type itemItemLookupResponseError = (itemItemLookupResponseDefault) & {
  headers: Headers;
};

export const getItemItemLookupUrl = (params: ItemItemLookupParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/items/lookup?${stringifiedParams}` : `/api/v1/items/lookup`
}

/**
 * Разрешение носителя (AD-41) → изделие на момент. Не найдено — 404 api.not_found.
 * @summary Найти изделие по номеру детали или скану DataMatrix
 */
export const itemItemLookup = async (params: ItemItemLookupParams, options?: RequestInit): Promise<itemItemLookupResponseSuccess> => {

  const res = await fetch(getItemItemLookupUrl(params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: itemItemLookupResponseError['data'], status?: number} = new globalThis.Error();
    const data : itemItemLookupResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: itemItemLookupResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as itemItemLookupResponseSuccess
}





export const getItemItemLookupQueryKey = (params?: MaybeRefOrGetter<ItemItemLookupParams>,) => {
    return [
    'api','v1','items','lookup', ...(params ? [params] : [])
    ] as const;
    }


export const getItemItemLookupQueryOptions = <TData = Awaited<ReturnType<typeof itemItemLookup>>, TError = globalThis.Error & { info?: Problem; status?: number }>(params: MaybeRefOrGetter<ItemItemLookupParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof itemItemLookup>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getItemItemLookupQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof itemItemLookup>>> = ({ signal }) => itemItemLookup(toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof itemItemLookup>>, TError, TData>
}

export type ItemItemLookupQueryResult = NonNullable<Awaited<ReturnType<typeof itemItemLookup>>>
export type ItemItemLookupQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Найти изделие по номеру детали или скану DataMatrix
 */

export function useItemItemLookup<TData = Awaited<ReturnType<typeof itemItemLookup>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 params: MaybeRefOrGetter<ItemItemLookupParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof itemItemLookup>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getItemItemLookupQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type itemAssemblyRecordResponse200 = {
  data: Receipt
  status: 200
}

export type itemAssemblyRecordResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type itemAssemblyRecordResponseSuccess = (itemAssemblyRecordResponse200) & {
  headers: Headers;
};
export type itemAssemblyRecordResponseError = (itemAssemblyRecordResponseDefault) & {
  headers: Headers;
};

export const getItemAssemblyRecordUrl = (itemId: string,) => {




  return `/api/v1/items/${itemId}/assembly`
}

/**
 * FR-45: факт сборки; связь генеалогии пишет межизделийная стадия (genealogy.link.added обоим изделиям, AD-42).
 * @summary Установить компонент в сборку
 */
export const itemAssemblyRecord = async (itemId: string,
    recordAssembly: RecordAssembly, options?: RequestInit): Promise<itemAssemblyRecordResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getItemAssemblyRecordUrl(itemId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(recordAssembly)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: itemAssemblyRecordResponseError['data'], status?: number} = new globalThis.Error();
    const data : itemAssemblyRecordResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: itemAssemblyRecordResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as itemAssemblyRecordResponseSuccess
}





export const getItemAssemblyRecordMutationKey = () => ['itemAssemblyRecord'] as const;

export const getItemAssemblyRecordMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof itemAssemblyRecord>>, TError,ItemAssemblyRecordMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof itemAssemblyRecord>>, TError,ItemAssemblyRecordMutationVariables, TContext> => {

const mutationKey = getItemAssemblyRecordMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof itemAssemblyRecord>>, ItemAssemblyRecordMutationVariables> = (props) => {
          const {itemId,data} = props ?? {};

          return  itemAssemblyRecord(itemId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type ItemAssemblyRecordMutationResult = NonNullable<Awaited<ReturnType<typeof itemAssemblyRecord>>>
    export type ItemAssemblyRecordMutationBody = RecordAssembly
    export type ItemAssemblyRecordMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type ItemAssemblyRecordMutationVariables = {itemId: string;data: RecordAssembly}

    /**
 * @summary Установить компонент в сборку
 */
export const useItemAssemblyRecord = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof itemAssemblyRecord>>, TError,ItemAssemblyRecordMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof itemAssemblyRecord>>,
        TError,
        ItemAssemblyRecordMutationVariables,
        TContext
      > => {
      return useMutation(getItemAssemblyRecordMutationOptions(options), queryClient);
    }

export type itemCarrierApplyResponse200 = {
  data: Receipt
  status: 200
}

export type itemCarrierApplyResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type itemCarrierApplyResponseSuccess = (itemCarrierApplyResponse200) & {
  headers: Headers;
};
export type itemCarrierApplyResponseError = (itemCarrierApplyResponseDefault) & {
  headers: Headers;
};

export const getItemCarrierApplyUrl = (itemId: string,) => {




  return `/api/v1/items/${itemId}/carriers`
}

/**
 * AD-16: бирка с QR, DPM, тара с ячейкой; перемаркировка — replaces_value.
 * @summary Нанести носитель
 */
export const itemCarrierApply = async (itemId: string,
    applyCarrier: ApplyCarrier, options?: RequestInit): Promise<itemCarrierApplyResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getItemCarrierApplyUrl(itemId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(applyCarrier)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: itemCarrierApplyResponseError['data'], status?: number} = new globalThis.Error();
    const data : itemCarrierApplyResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: itemCarrierApplyResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as itemCarrierApplyResponseSuccess
}





export const getItemCarrierApplyMutationKey = () => ['itemCarrierApply'] as const;

export const getItemCarrierApplyMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof itemCarrierApply>>, TError,ItemCarrierApplyMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof itemCarrierApply>>, TError,ItemCarrierApplyMutationVariables, TContext> => {

const mutationKey = getItemCarrierApplyMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof itemCarrierApply>>, ItemCarrierApplyMutationVariables> = (props) => {
          const {itemId,data} = props ?? {};

          return  itemCarrierApply(itemId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type ItemCarrierApplyMutationResult = NonNullable<Awaited<ReturnType<typeof itemCarrierApply>>>
    export type ItemCarrierApplyMutationBody = ApplyCarrier
    export type ItemCarrierApplyMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type ItemCarrierApplyMutationVariables = {itemId: string;data: ApplyCarrier}

    /**
 * @summary Нанести носитель
 */
export const useItemCarrierApply = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof itemCarrierApply>>, TError,ItemCarrierApplyMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof itemCarrierApply>>,
        TError,
        ItemCarrierApplyMutationVariables,
        TContext
      > => {
      return useMutation(getItemCarrierApplyMutationOptions(options), queryClient);
    }

export type itemCarrierRemoveResponse200 = {
  data: Receipt
  status: 200
}

export type itemCarrierRemoveResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type itemCarrierRemoveResponseSuccess = (itemCarrierRemoveResponse200) & {
  headers: Headers;
};
export type itemCarrierRemoveResponseError = (itemCarrierRemoveResponseDefault) & {
  headers: Headers;
};

export const getItemCarrierRemoveUrl = (itemId: string,) => {




  return `/api/v1/items/${itemId}/carriers/remove`
}

/**
 * AD-16: не снятый временный носитель — задача контроля полноты.
 * @summary Снять носитель
 */
export const itemCarrierRemove = async (itemId: string,
    removeCarrier: RemoveCarrier, options?: RequestInit): Promise<itemCarrierRemoveResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getItemCarrierRemoveUrl(itemId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(removeCarrier)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: itemCarrierRemoveResponseError['data'], status?: number} = new globalThis.Error();
    const data : itemCarrierRemoveResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: itemCarrierRemoveResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as itemCarrierRemoveResponseSuccess
}





export const getItemCarrierRemoveMutationKey = () => ['itemCarrierRemove'] as const;

export const getItemCarrierRemoveMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof itemCarrierRemove>>, TError,ItemCarrierRemoveMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof itemCarrierRemove>>, TError,ItemCarrierRemoveMutationVariables, TContext> => {

const mutationKey = getItemCarrierRemoveMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof itemCarrierRemove>>, ItemCarrierRemoveMutationVariables> = (props) => {
          const {itemId,data} = props ?? {};

          return  itemCarrierRemove(itemId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type ItemCarrierRemoveMutationResult = NonNullable<Awaited<ReturnType<typeof itemCarrierRemove>>>
    export type ItemCarrierRemoveMutationBody = RemoveCarrier
    export type ItemCarrierRemoveMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type ItemCarrierRemoveMutationVariables = {itemId: string;data: RemoveCarrier}

    /**
 * @summary Снять носитель
 */
export const useItemCarrierRemove = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof itemCarrierRemove>>, TError,ItemCarrierRemoveMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof itemCarrierRemove>>,
        TError,
        ItemCarrierRemoveMutationVariables,
        TContext
      > => {
      return useMutation(getItemCarrierRemoveMutationOptions(options), queryClient);
    }

export type nonconformityContainmentSetResponse200 = {
  data: Receipt
  status: 200
}

export type nonconformityContainmentSetResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type nonconformityContainmentSetResponseSuccess = (nonconformityContainmentSetResponse200) & {
  headers: Headers;
};
export type nonconformityContainmentSetResponseError = (nonconformityContainmentSetResponseDefault) & {
  headers: Headers;
};

export const getNonconformityContainmentSetUrl = (itemId: string,) => {




  return `/api/v1/items/${itemId}/containment`
}

/**
 * FR-49: наблюдать / доп. проверка / блок изделия или партии — защитное действие, критическое.
 * @summary Установить сдерживание
 */
export const nonconformityContainmentSet = async (itemId: string,
    setContainment: SetContainment, options?: RequestInit): Promise<nonconformityContainmentSetResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getNonconformityContainmentSetUrl(itemId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(setContainment)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: nonconformityContainmentSetResponseError['data'], status?: number} = new globalThis.Error();
    const data : nonconformityContainmentSetResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: nonconformityContainmentSetResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as nonconformityContainmentSetResponseSuccess
}





export const getNonconformityContainmentSetMutationKey = () => ['nonconformityContainmentSet'] as const;

export const getNonconformityContainmentSetMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof nonconformityContainmentSet>>, TError,NonconformityContainmentSetMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof nonconformityContainmentSet>>, TError,NonconformityContainmentSetMutationVariables, TContext> => {

const mutationKey = getNonconformityContainmentSetMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof nonconformityContainmentSet>>, NonconformityContainmentSetMutationVariables> = (props) => {
          const {itemId,data} = props ?? {};

          return  nonconformityContainmentSet(itemId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type NonconformityContainmentSetMutationResult = NonNullable<Awaited<ReturnType<typeof nonconformityContainmentSet>>>
    export type NonconformityContainmentSetMutationBody = SetContainment
    export type NonconformityContainmentSetMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type NonconformityContainmentSetMutationVariables = {itemId: string;data: SetContainment}

    /**
 * @summary Установить сдерживание
 */
export const useNonconformityContainmentSet = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof nonconformityContainmentSet>>, TError,NonconformityContainmentSetMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof nonconformityContainmentSet>>,
        TError,
        NonconformityContainmentSetMutationVariables,
        TContext
      > => {
      return useMutation(getNonconformityContainmentSetMutationOptions(options), queryClient);
    }

export type nonconformityContainmentReleaseResponse200 = {
  data: Receipt
  status: 200
}

export type nonconformityContainmentReleaseResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type nonconformityContainmentReleaseResponseSuccess = (nonconformityContainmentReleaseResponse200) & {
  headers: Headers;
};
export type nonconformityContainmentReleaseResponseError = (nonconformityContainmentReleaseResponseDefault) & {
  headers: Headers;
};

export const getNonconformityContainmentReleaseUrl = (itemId: string,) => {




  return `/api/v1/items/${itemId}/containment/release`
}

/**
 * FR-49, AD-27: снятие блока — разрешающее действие только уполномоченного; снятие блока ≠ годность; при активном инциденте — отказ гарда.
 * @summary Снять сдерживание
 */
export const nonconformityContainmentRelease = async (itemId: string,
    releaseContainment: ReleaseContainment, options?: RequestInit): Promise<nonconformityContainmentReleaseResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getNonconformityContainmentReleaseUrl(itemId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(releaseContainment)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: nonconformityContainmentReleaseResponseError['data'], status?: number} = new globalThis.Error();
    const data : nonconformityContainmentReleaseResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: nonconformityContainmentReleaseResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as nonconformityContainmentReleaseResponseSuccess
}





export const getNonconformityContainmentReleaseMutationKey = () => ['nonconformityContainmentRelease'] as const;

export const getNonconformityContainmentReleaseMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof nonconformityContainmentRelease>>, TError,NonconformityContainmentReleaseMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof nonconformityContainmentRelease>>, TError,NonconformityContainmentReleaseMutationVariables, TContext> => {

const mutationKey = getNonconformityContainmentReleaseMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof nonconformityContainmentRelease>>, NonconformityContainmentReleaseMutationVariables> = (props) => {
          const {itemId,data} = props ?? {};

          return  nonconformityContainmentRelease(itemId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type NonconformityContainmentReleaseMutationResult = NonNullable<Awaited<ReturnType<typeof nonconformityContainmentRelease>>>
    export type NonconformityContainmentReleaseMutationBody = ReleaseContainment
    export type NonconformityContainmentReleaseMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type NonconformityContainmentReleaseMutationVariables = {itemId: string;data: ReleaseContainment}

    /**
 * @summary Снять сдерживание
 */
export const useNonconformityContainmentRelease = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof nonconformityContainmentRelease>>, TError,NonconformityContainmentReleaseMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof nonconformityContainmentRelease>>,
        TError,
        NonconformityContainmentReleaseMutationVariables,
        TContext
      > => {
      return useMutation(getNonconformityContainmentReleaseMutationOptions(options), queryClient);
    }

export type qualityCoverageReadResponse200 = {
  data: InspectionCoverage
  status: 200
}

export type qualityCoverageReadResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type qualityCoverageReadResponseSuccess = (qualityCoverageReadResponse200) & {
  headers: Headers;
};
export type qualityCoverageReadResponseError = (qualityCoverageReadResponseDefault) & {
  headers: Headers;
};

export const getQualityCoverageReadUrl = (itemId: string,
    params?: QualityCoverageReadParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/items/${itemId}/coverage?${stringifiedParams}` : `/api/v1/items/${itemId}/coverage`
}

/**
 * FR-35, FR-14: точки контроля плана по изделию — результат получен, ждём или нет (с причиной пропуска).
 * @summary Полнота контроля
 */
export const qualityCoverageRead = async (itemId: string,
    params?: QualityCoverageReadParams, options?: RequestInit): Promise<qualityCoverageReadResponseSuccess> => {

  const res = await fetch(getQualityCoverageReadUrl(itemId,params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: qualityCoverageReadResponseError['data'], status?: number} = new globalThis.Error();
    const data : qualityCoverageReadResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: qualityCoverageReadResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as qualityCoverageReadResponseSuccess
}





export const getQualityCoverageReadQueryKey = (itemId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<QualityCoverageReadParams>,) => {
    return [
    'api','v1','items',itemId,'coverage', ...(params ? [params] : [])
    ] as const;
    }


export const getQualityCoverageReadQueryOptions = <TData = Awaited<ReturnType<typeof qualityCoverageRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(itemId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<QualityCoverageReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof qualityCoverageRead>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getQualityCoverageReadQueryKey(itemId,params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof qualityCoverageRead>>> = ({ signal }) => qualityCoverageRead(toValue(itemId),toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, enabled: computed(() => toValue(itemId) !== null && toValue(itemId) !== undefined), ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof qualityCoverageRead>>, TError, TData>
}

export type QualityCoverageReadQueryResult = NonNullable<Awaited<ReturnType<typeof qualityCoverageRead>>>
export type QualityCoverageReadQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Полнота контроля
 */

export function useQualityCoverageRead<TData = Awaited<ReturnType<typeof qualityCoverageRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 itemId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<QualityCoverageReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof qualityCoverageRead>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getQualityCoverageReadQueryOptions(itemId,params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type itemGenealogyReadResponse200 = {
  data: ItemGenealogy
  status: 200
}

export type itemGenealogyReadResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type itemGenealogyReadResponseSuccess = (itemGenealogyReadResponse200) & {
  headers: Headers;
};
export type itemGenealogyReadResponseError = (itemGenealogyReadResponseDefault) & {
  headers: Headers;
};

export const getItemGenealogyReadUrl = (itemId: string,
    params?: ItemGenealogyReadParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/items/${itemId}/genealogy?${stringifiedParams}` : `/api/v1/items/${itemId}/genealogy`
}

/**
 * FR-45: из чего собрано и куда вошло, партии и выписки партнёров; владелец генеалогии — межизделийная стадия (AD-42), паспорт её показывает.
 * @summary Генеалогия изделия
 */
export const itemGenealogyRead = async (itemId: string,
    params?: ItemGenealogyReadParams, options?: RequestInit): Promise<itemGenealogyReadResponseSuccess> => {

  const res = await fetch(getItemGenealogyReadUrl(itemId,params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: itemGenealogyReadResponseError['data'], status?: number} = new globalThis.Error();
    const data : itemGenealogyReadResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: itemGenealogyReadResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as itemGenealogyReadResponseSuccess
}





export const getItemGenealogyReadQueryKey = (itemId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<ItemGenealogyReadParams>,) => {
    return [
    'api','v1','items',itemId,'genealogy', ...(params ? [params] : [])
    ] as const;
    }


export const getItemGenealogyReadQueryOptions = <TData = Awaited<ReturnType<typeof itemGenealogyRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(itemId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<ItemGenealogyReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof itemGenealogyRead>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getItemGenealogyReadQueryKey(itemId,params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof itemGenealogyRead>>> = ({ signal }) => itemGenealogyRead(toValue(itemId),toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, enabled: computed(() => toValue(itemId) !== null && toValue(itemId) !== undefined), ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof itemGenealogyRead>>, TError, TData>
}

export type ItemGenealogyReadQueryResult = NonNullable<Awaited<ReturnType<typeof itemGenealogyRead>>>
export type ItemGenealogyReadQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Генеалогия изделия
 */

export function useItemGenealogyRead<TData = Awaited<ReturnType<typeof itemGenealogyRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 itemId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<ItemGenealogyReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof itemGenealogyRead>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getItemGenealogyReadQueryOptions(itemId,params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type itemHistoryListResponse200 = {
  data: ItemHistory
  status: 200
}

export type itemHistoryListResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type itemHistoryListResponseSuccess = (itemHistoryListResponse200) & {
  headers: Headers;
};
export type itemHistoryListResponseError = (itemHistoryListResponseDefault) & {
  headers: Headers;
};

export const getItemHistoryListUrl = (itemId: string,
    params?: ItemHistoryListParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/items/${itemId}/history?${stringifiedParams}` : `/api/v1/items/${itemId}/history`
}

/**
 * FR-43: проекция — было / стало / кто / причина; исправления — новыми записями (FR-122).
 * @summary Журнал изменений паспорта
 */
export const itemHistoryList = async (itemId: string,
    params?: ItemHistoryListParams, options?: RequestInit): Promise<itemHistoryListResponseSuccess> => {

  const res = await fetch(getItemHistoryListUrl(itemId,params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: itemHistoryListResponseError['data'], status?: number} = new globalThis.Error();
    const data : itemHistoryListResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: itemHistoryListResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as itemHistoryListResponseSuccess
}





export const getItemHistoryListQueryKey = (itemId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<ItemHistoryListParams>,) => {
    return [
    'api','v1','items',itemId,'history', ...(params ? [params] : [])
    ] as const;
    }


export const getItemHistoryListQueryOptions = <TData = Awaited<ReturnType<typeof itemHistoryList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(itemId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<ItemHistoryListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof itemHistoryList>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getItemHistoryListQueryKey(itemId,params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof itemHistoryList>>> = ({ signal }) => itemHistoryList(toValue(itemId),toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, enabled: computed(() => toValue(itemId) !== null && toValue(itemId) !== undefined), ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof itemHistoryList>>, TError, TData>
}

export type ItemHistoryListQueryResult = NonNullable<Awaited<ReturnType<typeof itemHistoryList>>>
export type ItemHistoryListQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Журнал изменений паспорта
 */

export function useItemHistoryList<TData = Awaited<ReturnType<typeof itemHistoryList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 itemId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<ItemHistoryListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof itemHistoryList>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getItemHistoryListQueryOptions(itemId,params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type itemIdentificationConfirmResponse200 = {
  data: Receipt
  status: 200
}

export type itemIdentificationConfirmResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type itemIdentificationConfirmResponseSuccess = (itemIdentificationConfirmResponse200) & {
  headers: Headers;
};
export type itemIdentificationConfirmResponseError = (itemIdentificationConfirmResponseDefault) & {
  headers: Headers;
};

export const getItemIdentificationConfirmUrl = (itemId: string,) => {




  return `/api/v1/items/${itemId}/identification/confirm`
}

/**
 * AD-16: после «идентификация под сомнением» — повторная идентификация человеком с подписью; снимает изоляцию по этой причине (разрешающее, критическое).
 * @summary Подтвердить идентификацию
 */
export const itemIdentificationConfirm = async (itemId: string,
    confirmIdentification: ConfirmIdentification, options?: RequestInit): Promise<itemIdentificationConfirmResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getItemIdentificationConfirmUrl(itemId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(confirmIdentification)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: itemIdentificationConfirmResponseError['data'], status?: number} = new globalThis.Error();
    const data : itemIdentificationConfirmResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: itemIdentificationConfirmResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as itemIdentificationConfirmResponseSuccess
}





export const getItemIdentificationConfirmMutationKey = () => ['itemIdentificationConfirm'] as const;

export const getItemIdentificationConfirmMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof itemIdentificationConfirm>>, TError,ItemIdentificationConfirmMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof itemIdentificationConfirm>>, TError,ItemIdentificationConfirmMutationVariables, TContext> => {

const mutationKey = getItemIdentificationConfirmMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof itemIdentificationConfirm>>, ItemIdentificationConfirmMutationVariables> = (props) => {
          const {itemId,data} = props ?? {};

          return  itemIdentificationConfirm(itemId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type ItemIdentificationConfirmMutationResult = NonNullable<Awaited<ReturnType<typeof itemIdentificationConfirm>>>
    export type ItemIdentificationConfirmMutationBody = ConfirmIdentification
    export type ItemIdentificationConfirmMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type ItemIdentificationConfirmMutationVariables = {itemId: string;data: ConfirmIdentification}

    /**
 * @summary Подтвердить идентификацию
 */
export const useItemIdentificationConfirm = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof itemIdentificationConfirm>>, TError,ItemIdentificationConfirmMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof itemIdentificationConfirm>>,
        TError,
        ItemIdentificationConfirmMutationVariables,
        TContext
      > => {
      return useMutation(getItemIdentificationConfirmMutationOptions(options), queryClient);
    }

export type qualityInspectionListResponse200 = {
  data: InspectionResultList
  status: 200
}

export type qualityInspectionListResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type qualityInspectionListResponseSuccess = (qualityInspectionListResponse200) & {
  headers: Headers;
};
export type qualityInspectionListResponseError = (qualityInspectionListResponseDefault) & {
  headers: Headers;
};

export const getQualityInspectionListUrl = (itemId: string,
    params?: QualityInspectionListParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/items/${itemId}/inspections?${stringifiedParams}` : `/api/v1/items/${itemId}/inspections`
}

/**
 * FR-36: результаты всех методов одним типом, три исхода — признак дефекта / признака нет / оценка невозможна; пометка источника (FR-140).
 * @summary Результаты контроля изделия
 */
export const qualityInspectionList = async (itemId: string,
    params?: QualityInspectionListParams, options?: RequestInit): Promise<qualityInspectionListResponseSuccess> => {

  const res = await fetch(getQualityInspectionListUrl(itemId,params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: qualityInspectionListResponseError['data'], status?: number} = new globalThis.Error();
    const data : qualityInspectionListResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: qualityInspectionListResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as qualityInspectionListResponseSuccess
}





export const getQualityInspectionListQueryKey = (itemId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<QualityInspectionListParams>,) => {
    return [
    'api','v1','items',itemId,'inspections', ...(params ? [params] : [])
    ] as const;
    }


export const getQualityInspectionListQueryOptions = <TData = Awaited<ReturnType<typeof qualityInspectionList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(itemId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<QualityInspectionListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof qualityInspectionList>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getQualityInspectionListQueryKey(itemId,params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof qualityInspectionList>>> = ({ signal }) => qualityInspectionList(toValue(itemId),toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, enabled: computed(() => toValue(itemId) !== null && toValue(itemId) !== undefined), ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof qualityInspectionList>>, TError, TData>
}

export type QualityInspectionListQueryResult = NonNullable<Awaited<ReturnType<typeof qualityInspectionList>>>
export type QualityInspectionListQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Результаты контроля изделия
 */

export function useQualityInspectionList<TData = Awaited<ReturnType<typeof qualityInspectionList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 itemId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<QualityInspectionListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof qualityInspectionList>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getQualityInspectionListQueryOptions(itemId,params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type itemInterventionOpenResponse200 = {
  data: Receipt
  status: 200
}

export type itemInterventionOpenResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type itemInterventionOpenResponseSuccess = (itemInterventionOpenResponse200) & {
  headers: Headers;
};
export type itemInterventionOpenResponseError = (itemInterventionOpenResponseDefault) & {
  headers: Headers;
};

export const getItemInterventionOpenUrl = (itemId: string,) => {




  return `/api/v1/items/${itemId}/interventions`
}

/**
 * FR-21: разборка собранного изделия — мастер открывает, зоны теряют статус проверенных до повторного контроля.
 * @summary Открыть вмешательство
 */
export const itemInterventionOpen = async (itemId: string,
    openIntervention: OpenIntervention, options?: RequestInit): Promise<itemInterventionOpenResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getItemInterventionOpenUrl(itemId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(openIntervention)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: itemInterventionOpenResponseError['data'], status?: number} = new globalThis.Error();
    const data : itemInterventionOpenResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: itemInterventionOpenResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as itemInterventionOpenResponseSuccess
}





export const getItemInterventionOpenMutationKey = () => ['itemInterventionOpen'] as const;

export const getItemInterventionOpenMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof itemInterventionOpen>>, TError,ItemInterventionOpenMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof itemInterventionOpen>>, TError,ItemInterventionOpenMutationVariables, TContext> => {

const mutationKey = getItemInterventionOpenMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof itemInterventionOpen>>, ItemInterventionOpenMutationVariables> = (props) => {
          const {itemId,data} = props ?? {};

          return  itemInterventionOpen(itemId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type ItemInterventionOpenMutationResult = NonNullable<Awaited<ReturnType<typeof itemInterventionOpen>>>
    export type ItemInterventionOpenMutationBody = OpenIntervention
    export type ItemInterventionOpenMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type ItemInterventionOpenMutationVariables = {itemId: string;data: OpenIntervention}

    /**
 * @summary Открыть вмешательство
 */
export const useItemInterventionOpen = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof itemInterventionOpen>>, TError,ItemInterventionOpenMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof itemInterventionOpen>>,
        TError,
        ItemInterventionOpenMutationVariables,
        TContext
      > => {
      return useMutation(getItemInterventionOpenMutationOptions(options), queryClient);
    }

export type itemInterventionCloseResponse200 = {
  data: Receipt
  status: 200
}

export type itemInterventionCloseResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type itemInterventionCloseResponseSuccess = (itemInterventionCloseResponse200) & {
  headers: Headers;
};
export type itemInterventionCloseResponseError = (itemInterventionCloseResponseDefault) & {
  headers: Headers;
};

export const getItemInterventionCloseUrl = (itemId: string,
    interventionId: string,) => {




  return `/api/v1/items/${itemId}/interventions/${interventionId}/close`
}

/**
 * FR-21: контролёр закрывает после повторной проверки зоны — разрешающее действие (AD-27), критическое (AD-28).
 * @summary Закрыть вмешательство
 */
export const itemInterventionClose = async (itemId: string,
    interventionId: string,
    closeIntervention: CloseIntervention, options?: RequestInit): Promise<itemInterventionCloseResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getItemInterventionCloseUrl(itemId,interventionId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(closeIntervention)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: itemInterventionCloseResponseError['data'], status?: number} = new globalThis.Error();
    const data : itemInterventionCloseResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: itemInterventionCloseResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as itemInterventionCloseResponseSuccess
}





export const getItemInterventionCloseMutationKey = () => ['itemInterventionClose'] as const;

export const getItemInterventionCloseMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof itemInterventionClose>>, TError,ItemInterventionCloseMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof itemInterventionClose>>, TError,ItemInterventionCloseMutationVariables, TContext> => {

const mutationKey = getItemInterventionCloseMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof itemInterventionClose>>, ItemInterventionCloseMutationVariables> = (props) => {
          const {itemId,interventionId,data} = props ?? {};

          return  itemInterventionClose(itemId,interventionId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type ItemInterventionCloseMutationResult = NonNullable<Awaited<ReturnType<typeof itemInterventionClose>>>
    export type ItemInterventionCloseMutationBody = CloseIntervention
    export type ItemInterventionCloseMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type ItemInterventionCloseMutationVariables = {itemId: string;interventionId: string;data: CloseIntervention}

    /**
 * @summary Закрыть вмешательство
 */
export const useItemInterventionClose = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof itemInterventionClose>>, TError,ItemInterventionCloseMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof itemInterventionClose>>,
        TError,
        ItemInterventionCloseMutationVariables,
        TContext
      > => {
      return useMutation(getItemInterventionCloseMutationOptions(options), queryClient);
    }

export type nonconformityItemIsolateResponse200 = {
  data: Receipt
  status: 200
}

export type nonconformityItemIsolateResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type nonconformityItemIsolateResponseSuccess = (nonconformityItemIsolateResponse200) & {
  headers: Headers;
};
export type nonconformityItemIsolateResponseError = (nonconformityItemIsolateResponseDefault) & {
  headers: Headers;
};

export const getNonconformityItemIsolateUrl = (itemId: string,) => {




  return `/api/v1/items/${itemId}/isolate`
}

/**
 * FR-55: изделие — в изоляцию со сроком решения; «изоляция» — положение, ожидающее решения (AD-30); защитное действие, критическое.
 * @summary Изолировать
 */
export const nonconformityItemIsolate = async (itemId: string,
    isolateItem: IsolateItem, options?: RequestInit): Promise<nonconformityItemIsolateResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getNonconformityItemIsolateUrl(itemId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(isolateItem)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: nonconformityItemIsolateResponseError['data'], status?: number} = new globalThis.Error();
    const data : nonconformityItemIsolateResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: nonconformityItemIsolateResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as nonconformityItemIsolateResponseSuccess
}





export const getNonconformityItemIsolateMutationKey = () => ['nonconformityItemIsolate'] as const;

export const getNonconformityItemIsolateMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof nonconformityItemIsolate>>, TError,NonconformityItemIsolateMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof nonconformityItemIsolate>>, TError,NonconformityItemIsolateMutationVariables, TContext> => {

const mutationKey = getNonconformityItemIsolateMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof nonconformityItemIsolate>>, NonconformityItemIsolateMutationVariables> = (props) => {
          const {itemId,data} = props ?? {};

          return  nonconformityItemIsolate(itemId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type NonconformityItemIsolateMutationResult = NonNullable<Awaited<ReturnType<typeof nonconformityItemIsolate>>>
    export type NonconformityItemIsolateMutationBody = IsolateItem
    export type NonconformityItemIsolateMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type NonconformityItemIsolateMutationVariables = {itemId: string;data: IsolateItem}

    /**
 * @summary Изолировать
 */
export const useNonconformityItemIsolate = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof nonconformityItemIsolate>>, TError,NonconformityItemIsolateMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof nonconformityItemIsolate>>,
        TError,
        NonconformityItemIsolateMutationVariables,
        TContext
      > => {
      return useMutation(getNonconformityItemIsolateMutationOptions(options), queryClient);
    }

export type processMovementSendResponse200 = {
  data: Receipt
  status: 200
}

export type processMovementSendResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type processMovementSendResponseSuccess = (processMovementSendResponse200) & {
  headers: Headers;
};
export type processMovementSendResponseError = (processMovementSendResponseDefault) & {
  headers: Headers;
};

export const getProcessMovementSendUrl = (itemId: string,) => {




  return `/api/v1/items/${itemId}/movements`
}

/**
 * FR-16: перемещение между участками и цехами; «в перемещении» до приёма.
 * @summary Отправить изделие
 */
export const processMovementSend = async (itemId: string,
    sendMovement: SendMovement, options?: RequestInit): Promise<processMovementSendResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getProcessMovementSendUrl(itemId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(sendMovement)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: processMovementSendResponseError['data'], status?: number} = new globalThis.Error();
    const data : processMovementSendResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: processMovementSendResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as processMovementSendResponseSuccess
}





export const getProcessMovementSendMutationKey = () => ['processMovementSend'] as const;

export const getProcessMovementSendMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof processMovementSend>>, TError,ProcessMovementSendMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof processMovementSend>>, TError,ProcessMovementSendMutationVariables, TContext> => {

const mutationKey = getProcessMovementSendMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof processMovementSend>>, ProcessMovementSendMutationVariables> = (props) => {
          const {itemId,data} = props ?? {};

          return  processMovementSend(itemId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type ProcessMovementSendMutationResult = NonNullable<Awaited<ReturnType<typeof processMovementSend>>>
    export type ProcessMovementSendMutationBody = SendMovement
    export type ProcessMovementSendMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type ProcessMovementSendMutationVariables = {itemId: string;data: SendMovement}

    /**
 * @summary Отправить изделие
 */
export const useProcessMovementSend = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof processMovementSend>>, TError,ProcessMovementSendMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof processMovementSend>>,
        TError,
        ProcessMovementSendMutationVariables,
        TContext
      > => {
      return useMutation(getProcessMovementSendMutationOptions(options), queryClient);
    }

export type processMovementReceiveResponse200 = {
  data: Receipt
  status: 200
}

export type processMovementReceiveResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type processMovementReceiveResponseSuccess = (processMovementReceiveResponse200) & {
  headers: Headers;
};
export type processMovementReceiveResponseError = (processMovementReceiveResponseDefault) & {
  headers: Headers;
};

export const getProcessMovementReceiveUrl = (itemId: string,) => {




  return `/api/v1/items/${itemId}/movements/receive`
}

/**
 * FR-16, FR-137: подтверждение приёма, в том числе перемещения в изолятор — снимает расхождение «изолировано в системе, физически нет».
 * @summary Принять изделие
 */
export const processMovementReceive = async (itemId: string,
    receiveMovement: ReceiveMovement, options?: RequestInit): Promise<processMovementReceiveResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getProcessMovementReceiveUrl(itemId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(receiveMovement)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: processMovementReceiveResponseError['data'], status?: number} = new globalThis.Error();
    const data : processMovementReceiveResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: processMovementReceiveResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as processMovementReceiveResponseSuccess
}





export const getProcessMovementReceiveMutationKey = () => ['processMovementReceive'] as const;

export const getProcessMovementReceiveMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof processMovementReceive>>, TError,ProcessMovementReceiveMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof processMovementReceive>>, TError,ProcessMovementReceiveMutationVariables, TContext> => {

const mutationKey = getProcessMovementReceiveMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof processMovementReceive>>, ProcessMovementReceiveMutationVariables> = (props) => {
          const {itemId,data} = props ?? {};

          return  processMovementReceive(itemId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type ProcessMovementReceiveMutationResult = NonNullable<Awaited<ReturnType<typeof processMovementReceive>>>
    export type ProcessMovementReceiveMutationBody = ReceiveMovement
    export type ProcessMovementReceiveMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type ProcessMovementReceiveMutationVariables = {itemId: string;data: ReceiveMovement}

    /**
 * @summary Принять изделие
 */
export const useProcessMovementReceive = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof processMovementReceive>>, TError,ProcessMovementReceiveMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof processMovementReceive>>,
        TError,
        ProcessMovementReceiveMutationVariables,
        TContext
      > => {
      return useMutation(getProcessMovementReceiveMutationOptions(options), queryClient);
    }

export type processOperationStartResponse200 = {
  data: Receipt
  status: 200
}

export type processOperationStartResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type processOperationStartResponseSuccess = (processOperationStartResponse200) & {
  headers: Headers;
};
export type processOperationStartResponseError = (processOperationStartResponseDefault) & {
  headers: Headers;
};

export const getProcessOperationStartUrl = (itemId: string,) => {




  return `/api/v1/items/${itemId}/operations`
}

/**
 * FR-137, FR-44: исполнитель на своём рабочем месте; предусловия (FR-17) и лимит доработок (FR-18) — гард; повтор — новый operation_run_id + rework_of (FR-47).
 * @summary Начать операцию
 */
export const processOperationStart = async (itemId: string,
    startOperation: StartOperation, options?: RequestInit): Promise<processOperationStartResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getProcessOperationStartUrl(itemId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(startOperation)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: processOperationStartResponseError['data'], status?: number} = new globalThis.Error();
    const data : processOperationStartResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: processOperationStartResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as processOperationStartResponseSuccess
}





export const getProcessOperationStartMutationKey = () => ['processOperationStart'] as const;

export const getProcessOperationStartMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof processOperationStart>>, TError,ProcessOperationStartMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof processOperationStart>>, TError,ProcessOperationStartMutationVariables, TContext> => {

const mutationKey = getProcessOperationStartMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof processOperationStart>>, ProcessOperationStartMutationVariables> = (props) => {
          const {itemId,data} = props ?? {};

          return  processOperationStart(itemId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type ProcessOperationStartMutationResult = NonNullable<Awaited<ReturnType<typeof processOperationStart>>>
    export type ProcessOperationStartMutationBody = StartOperation
    export type ProcessOperationStartMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type ProcessOperationStartMutationVariables = {itemId: string;data: StartOperation}

    /**
 * @summary Начать операцию
 */
export const useProcessOperationStart = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof processOperationStart>>, TError,ProcessOperationStartMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof processOperationStart>>,
        TError,
        ProcessOperationStartMutationVariables,
        TContext
      > => {
      return useMutation(getProcessOperationStartMutationOptions(options), queryClient);
    }

export type itemPassportReadResponse200 = {
  data: ItemPassport
  status: 200
}

export type itemPassportReadResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type itemPassportReadResponseSuccess = (itemPassportReadResponse200) & {
  headers: Headers;
};
export type itemPassportReadResponseError = (itemPassportReadResponseDefault) & {
  headers: Headers;
};

export const getItemPassportReadUrl = (itemId: string,
    params?: ItemPassportReadParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/items/${itemId}/passport?${stringifiedParams}` : `/api/v1/items/${itemId}/passport`
}

/**
 * FR-42, кейс «история изделия»: факты, решения и документы с автором, временем, подписью и статусом её проверки; пометка источника факта (FR-140); исходный сигнал, анализ системы и решение человека — раздельно; оси статуса §3b; зоны, носители, несоответствия, инциденты.
 * @summary Паспорт изделия
 */
export const itemPassportRead = async (itemId: string,
    params?: ItemPassportReadParams, options?: RequestInit): Promise<itemPassportReadResponseSuccess> => {

  const res = await fetch(getItemPassportReadUrl(itemId,params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: itemPassportReadResponseError['data'], status?: number} = new globalThis.Error();
    const data : itemPassportReadResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: itemPassportReadResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as itemPassportReadResponseSuccess
}





export const getItemPassportReadQueryKey = (itemId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<ItemPassportReadParams>,) => {
    return [
    'api','v1','items',itemId,'passport', ...(params ? [params] : [])
    ] as const;
    }


export const getItemPassportReadQueryOptions = <TData = Awaited<ReturnType<typeof itemPassportRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(itemId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<ItemPassportReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof itemPassportRead>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getItemPassportReadQueryKey(itemId,params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof itemPassportRead>>> = ({ signal }) => itemPassportRead(toValue(itemId),toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, enabled: computed(() => toValue(itemId) !== null && toValue(itemId) !== undefined), ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof itemPassportRead>>, TError, TData>
}

export type ItemPassportReadQueryResult = NonNullable<Awaited<ReturnType<typeof itemPassportRead>>>
export type ItemPassportReadQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Паспорт изделия
 */

export function useItemPassportRead<TData = Awaited<ReturnType<typeof itemPassportRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 itemId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<ItemPassportReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof itemPassportRead>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getItemPassportReadQueryOptions(itemId,params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type itemPresentationRecordResponse200 = {
  data: Receipt
  status: 200
}

export type itemPresentationRecordResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type itemPresentationRecordResponseSuccess = (itemPresentationRecordResponse200) & {
  headers: Headers;
};
export type itemPresentationRecordResponseError = (itemPresentationRecordResponseDefault) & {
  headers: Headers;
};

export const getItemPresentationRecordUrl = (itemId: string,) => {




  return `/api/v1/items/${itemId}/presentations`
}

/**
 * FR-19: мастер предъявляет изделие на точке предъявления; повторное предъявление — подписант уровнем выше.
 * @summary Предъявить ОТК
 */
export const itemPresentationRecord = async (itemId: string,
    recordPresentation: RecordPresentation, options?: RequestInit): Promise<itemPresentationRecordResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getItemPresentationRecordUrl(itemId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(recordPresentation)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: itemPresentationRecordResponseError['data'], status?: number} = new globalThis.Error();
    const data : itemPresentationRecordResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: itemPresentationRecordResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as itemPresentationRecordResponseSuccess
}





export const getItemPresentationRecordMutationKey = () => ['itemPresentationRecord'] as const;

export const getItemPresentationRecordMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof itemPresentationRecord>>, TError,ItemPresentationRecordMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof itemPresentationRecord>>, TError,ItemPresentationRecordMutationVariables, TContext> => {

const mutationKey = getItemPresentationRecordMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof itemPresentationRecord>>, ItemPresentationRecordMutationVariables> = (props) => {
          const {itemId,data} = props ?? {};

          return  itemPresentationRecord(itemId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type ItemPresentationRecordMutationResult = NonNullable<Awaited<ReturnType<typeof itemPresentationRecord>>>
    export type ItemPresentationRecordMutationBody = RecordPresentation
    export type ItemPresentationRecordMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type ItemPresentationRecordMutationVariables = {itemId: string;data: RecordPresentation}

    /**
 * @summary Предъявить ОТК
 */
export const useItemPresentationRecord = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof itemPresentationRecord>>, TError,ItemPresentationRecordMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof itemPresentationRecord>>,
        TError,
        ItemPresentationRecordMutationVariables,
        TContext
      > => {
      return useMutation(getItemPresentationRecordMutationOptions(options), queryClient);
    }

export type nonconformityPresentationResolveResponse200 = {
  data: Receipt
  status: 200
}

export type nonconformityPresentationResolveResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type nonconformityPresentationResolveResponseSuccess = (nonconformityPresentationResolveResponse200) & {
  headers: Headers;
};
export type nonconformityPresentationResolveResponseError = (nonconformityPresentationResolveResponseDefault) & {
  headers: Headers;
};

export const getNonconformityPresentationResolveUrl = (itemId: string,) => {




  return `/api/v1/items/${itemId}/presentations/resolve`
}

/**
 * FR-19, FR-56: «Принять — передать на ‹следующий шаг›», принять по разрешению, вернуть, недостаточно данных; участник изготовления не принимает (разделение обязанностей); без результатов обязательных методов — отказ (nonconformity.method_result_missing).
 * @summary Решение на точке предъявления
 */
export const nonconformityPresentationResolve = async (itemId: string,
    resolvePresentation: ResolvePresentation, options?: RequestInit): Promise<nonconformityPresentationResolveResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getNonconformityPresentationResolveUrl(itemId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(resolvePresentation)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: nonconformityPresentationResolveResponseError['data'], status?: number} = new globalThis.Error();
    const data : nonconformityPresentationResolveResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: nonconformityPresentationResolveResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as nonconformityPresentationResolveResponseSuccess
}





export const getNonconformityPresentationResolveMutationKey = () => ['nonconformityPresentationResolve'] as const;

export const getNonconformityPresentationResolveMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof nonconformityPresentationResolve>>, TError,NonconformityPresentationResolveMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof nonconformityPresentationResolve>>, TError,NonconformityPresentationResolveMutationVariables, TContext> => {

const mutationKey = getNonconformityPresentationResolveMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof nonconformityPresentationResolve>>, NonconformityPresentationResolveMutationVariables> = (props) => {
          const {itemId,data} = props ?? {};

          return  nonconformityPresentationResolve(itemId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type NonconformityPresentationResolveMutationResult = NonNullable<Awaited<ReturnType<typeof nonconformityPresentationResolve>>>
    export type NonconformityPresentationResolveMutationBody = ResolvePresentation
    export type NonconformityPresentationResolveMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type NonconformityPresentationResolveMutationVariables = {itemId: string;data: ResolvePresentation}

    /**
 * @summary Решение на точке предъявления
 */
export const useNonconformityPresentationResolve = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof nonconformityPresentationResolve>>, TError,NonconformityPresentationResolveMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof nonconformityPresentationResolve>>,
        TError,
        NonconformityPresentationResolveMutationVariables,
        TContext
      > => {
      return useMutation(getNonconformityPresentationResolveMutationOptions(options), queryClient);
    }

export type nonconformityRecheckRequestResponse200 = {
  data: Receipt
  status: 200
}

export type nonconformityRecheckRequestResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type nonconformityRecheckRequestResponseSuccess = (nonconformityRecheckRequestResponse200) & {
  headers: Headers;
};
export type nonconformityRecheckRequestResponseError = (nonconformityRecheckRequestResponseDefault) & {
  headers: Headers;
};

export const getNonconformityRecheckRequestUrl = (itemId: string,) => {




  return `/api/v1/items/${itemId}/recheck`
}

/**
 * FR-52: дополнительная проверка методом контроля — защитное действие.
 * @summary Назначить доп. проверку
 */
export const nonconformityRecheckRequest = async (itemId: string,
    requestRecheck: RequestRecheck, options?: RequestInit): Promise<nonconformityRecheckRequestResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getNonconformityRecheckRequestUrl(itemId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(requestRecheck)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: nonconformityRecheckRequestResponseError['data'], status?: number} = new globalThis.Error();
    const data : nonconformityRecheckRequestResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: nonconformityRecheckRequestResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as nonconformityRecheckRequestResponseSuccess
}





export const getNonconformityRecheckRequestMutationKey = () => ['nonconformityRecheckRequest'] as const;

export const getNonconformityRecheckRequestMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof nonconformityRecheckRequest>>, TError,NonconformityRecheckRequestMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof nonconformityRecheckRequest>>, TError,NonconformityRecheckRequestMutationVariables, TContext> => {

const mutationKey = getNonconformityRecheckRequestMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof nonconformityRecheckRequest>>, NonconformityRecheckRequestMutationVariables> = (props) => {
          const {itemId,data} = props ?? {};

          return  nonconformityRecheckRequest(itemId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type NonconformityRecheckRequestMutationResult = NonNullable<Awaited<ReturnType<typeof nonconformityRecheckRequest>>>
    export type NonconformityRecheckRequestMutationBody = RequestRecheck
    export type NonconformityRecheckRequestMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type NonconformityRecheckRequestMutationVariables = {itemId: string;data: RequestRecheck}

    /**
 * @summary Назначить доп. проверку
 */
export const useNonconformityRecheckRequest = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof nonconformityRecheckRequest>>, TError,NonconformityRecheckRequestMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof nonconformityRecheckRequest>>,
        TError,
        NonconformityRecheckRequestMutationVariables,
        TContext
      > => {
      return useMutation(getNonconformityRecheckRequestMutationOptions(options), queryClient);
    }

export type itemReleaseRecordResponse200 = {
  data: Receipt
  status: 200
}

export type itemReleaseRecordResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type itemReleaseRecordResponseSuccess = (itemReleaseRecordResponse200) & {
  headers: Headers;
};
export type itemReleaseRecordResponseError = (itemReleaseRecordResponseDefault) & {
  headers: Headers;
};

export const getItemReleaseRecordUrl = (itemId: string,) => {




  return `/api/v1/items/${itemId}/release`
}

/**
 * Закрывающая точка выпуска: реакция «выпуск» в 1С (с признаком after_rework для принятого после переделки, AD-18).
 * @summary Принять на склад выпуска
 */
export const itemReleaseRecord = async (itemId: string,
    recordRelease: RecordRelease, options?: RequestInit): Promise<itemReleaseRecordResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getItemReleaseRecordUrl(itemId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(recordRelease)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: itemReleaseRecordResponseError['data'], status?: number} = new globalThis.Error();
    const data : itemReleaseRecordResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: itemReleaseRecordResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as itemReleaseRecordResponseSuccess
}





export const getItemReleaseRecordMutationKey = () => ['itemReleaseRecord'] as const;

export const getItemReleaseRecordMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof itemReleaseRecord>>, TError,ItemReleaseRecordMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof itemReleaseRecord>>, TError,ItemReleaseRecordMutationVariables, TContext> => {

const mutationKey = getItemReleaseRecordMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof itemReleaseRecord>>, ItemReleaseRecordMutationVariables> = (props) => {
          const {itemId,data} = props ?? {};

          return  itemReleaseRecord(itemId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type ItemReleaseRecordMutationResult = NonNullable<Awaited<ReturnType<typeof itemReleaseRecord>>>
    export type ItemReleaseRecordMutationBody = RecordRelease
    export type ItemReleaseRecordMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type ItemReleaseRecordMutationVariables = {itemId: string;data: RecordRelease}

    /**
 * @summary Принять на склад выпуска
 */
export const useItemReleaseRecord = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof itemReleaseRecord>>, TError,ItemReleaseRecordMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof itemReleaseRecord>>,
        TError,
        ItemReleaseRecordMutationVariables,
        TContext
      > => {
      return useMutation(getItemReleaseRecordMutationOptions(options), queryClient);
    }

export type nonconformityReworkLimitWaiveResponse200 = {
  data: Receipt
  status: 200
}

export type nonconformityReworkLimitWaiveResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type nonconformityReworkLimitWaiveResponseSuccess = (nonconformityReworkLimitWaiveResponse200) & {
  headers: Headers;
};
export type nonconformityReworkLimitWaiveResponseError = (nonconformityReworkLimitWaiveResponseDefault) & {
  headers: Headers;
};

export const getNonconformityReworkLimitWaiveUrl = (itemId: string,) => {




  return `/api/v1/items/${itemId}/rework-limit/waive`
}

/**
 * FR-18: разрешение на доработку сверх лимита по зоне — полномочие «разрешение сверх лимита доработок».
 * @summary Разрешить сверх лимита доработок
 */
export const nonconformityReworkLimitWaive = async (itemId: string,
    waiveReworkLimit: WaiveReworkLimit, options?: RequestInit): Promise<nonconformityReworkLimitWaiveResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getNonconformityReworkLimitWaiveUrl(itemId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(waiveReworkLimit)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: nonconformityReworkLimitWaiveResponseError['data'], status?: number} = new globalThis.Error();
    const data : nonconformityReworkLimitWaiveResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: nonconformityReworkLimitWaiveResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as nonconformityReworkLimitWaiveResponseSuccess
}





export const getNonconformityReworkLimitWaiveMutationKey = () => ['nonconformityReworkLimitWaive'] as const;

export const getNonconformityReworkLimitWaiveMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof nonconformityReworkLimitWaive>>, TError,NonconformityReworkLimitWaiveMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof nonconformityReworkLimitWaive>>, TError,NonconformityReworkLimitWaiveMutationVariables, TContext> => {

const mutationKey = getNonconformityReworkLimitWaiveMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof nonconformityReworkLimitWaive>>, NonconformityReworkLimitWaiveMutationVariables> = (props) => {
          const {itemId,data} = props ?? {};

          return  nonconformityReworkLimitWaive(itemId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type NonconformityReworkLimitWaiveMutationResult = NonNullable<Awaited<ReturnType<typeof nonconformityReworkLimitWaive>>>
    export type NonconformityReworkLimitWaiveMutationBody = WaiveReworkLimit
    export type NonconformityReworkLimitWaiveMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type NonconformityReworkLimitWaiveMutationVariables = {itemId: string;data: WaiveReworkLimit}

    /**
 * @summary Разрешить сверх лимита доработок
 */
export const useNonconformityReworkLimitWaive = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof nonconformityReworkLimitWaive>>, TError,NonconformityReworkLimitWaiveMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof nonconformityReworkLimitWaive>>,
        TError,
        NonconformityReworkLimitWaiveMutationVariables,
        TContext
      > => {
      return useMutation(getNonconformityReworkLimitWaiveMutationOptions(options), queryClient);
    }

export type nonconformitySignalRejectResponse200 = {
  data: Receipt
  status: 200
}

export type nonconformitySignalRejectResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type nonconformitySignalRejectResponseSuccess = (nonconformitySignalRejectResponse200) & {
  headers: Headers;
};
export type nonconformitySignalRejectResponseError = (nonconformitySignalRejectResponseDefault) & {
  headers: Headers;
};

export const getNonconformitySignalRejectUrl = (itemId: string,) => {




  return `/api/v1/items/${itemId}/signals/reject`
}

/**
 * FR-52: отклонение сигнала с обязательной причиной — разрешающее действие, критическое (AD-27, AD-28); можно пометить для контура адаптации.
 * @summary Отклонить сигнал
 */
export const nonconformitySignalReject = async (itemId: string,
    rejectSignal: RejectSignal, options?: RequestInit): Promise<nonconformitySignalRejectResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getNonconformitySignalRejectUrl(itemId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(rejectSignal)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: nonconformitySignalRejectResponseError['data'], status?: number} = new globalThis.Error();
    const data : nonconformitySignalRejectResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: nonconformitySignalRejectResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as nonconformitySignalRejectResponseSuccess
}





export const getNonconformitySignalRejectMutationKey = () => ['nonconformitySignalReject'] as const;

export const getNonconformitySignalRejectMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof nonconformitySignalReject>>, TError,NonconformitySignalRejectMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof nonconformitySignalReject>>, TError,NonconformitySignalRejectMutationVariables, TContext> => {

const mutationKey = getNonconformitySignalRejectMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof nonconformitySignalReject>>, NonconformitySignalRejectMutationVariables> = (props) => {
          const {itemId,data} = props ?? {};

          return  nonconformitySignalReject(itemId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type NonconformitySignalRejectMutationResult = NonNullable<Awaited<ReturnType<typeof nonconformitySignalReject>>>
    export type NonconformitySignalRejectMutationBody = RejectSignal
    export type NonconformitySignalRejectMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type NonconformitySignalRejectMutationVariables = {itemId: string;data: RejectSignal}

    /**
 * @summary Отклонить сигнал
 */
export const useNonconformitySignalReject = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof nonconformitySignalReject>>, TError,NonconformitySignalRejectMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof nonconformitySignalReject>>,
        TError,
        NonconformitySignalRejectMutationVariables,
        TContext
      > => {
      return useMutation(getNonconformitySignalRejectMutationOptions(options), queryClient);
    }

export type journalEntryListResponse200 = {
  data: JournalEntryList
  status: 200
}

export type journalEntryListResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type journalEntryListResponseSuccess = (journalEntryListResponse200) & {
  headers: Headers;
};
export type journalEntryListResponseError = (journalEntryListResponseDefault) & {
  headers: Headers;
};

export const getJournalEntryListUrl = (params?: JournalEntryListParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/journal?${stringifiedParams}` : `/api/v1/journal`
}

/**
 * PRD §3a, общий экран: записи журнала по изделию, потоку, типу, виду записи; исходные события, анализ системы, решения людей и служебные записи различимы (AD-2, кейс §7.2); пометка источника и статус подписи у каждой записи (FR-140, FR-68).
 * @summary Журнал событий
 */
export const journalEntryList = async (params?: JournalEntryListParams, options?: RequestInit): Promise<journalEntryListResponseSuccess> => {

  const res = await fetch(getJournalEntryListUrl(params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: journalEntryListResponseError['data'], status?: number} = new globalThis.Error();
    const data : journalEntryListResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: journalEntryListResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as journalEntryListResponseSuccess
}





export const getJournalEntryListQueryKey = (params?: MaybeRefOrGetter<JournalEntryListParams>,) => {
    return [
    'api','v1','journal', ...(params ? [params] : [])
    ] as const;
    }


export const getJournalEntryListQueryOptions = <TData = Awaited<ReturnType<typeof journalEntryList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(params?: MaybeRefOrGetter<JournalEntryListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof journalEntryList>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getJournalEntryListQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof journalEntryList>>> = ({ signal }) => journalEntryList(toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof journalEntryList>>, TError, TData>
}

export type JournalEntryListQueryResult = NonNullable<Awaited<ReturnType<typeof journalEntryList>>>
export type JournalEntryListQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Журнал событий
 */

export function useJournalEntryList<TData = Awaited<ReturnType<typeof journalEntryList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 params?: MaybeRefOrGetter<JournalEntryListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof journalEntryList>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getJournalEntryListQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type journalHeadReadResponse200 = {
  data: JournalHead
  status: 200
}

export type journalHeadReadResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type journalHeadReadResponseSuccess = (journalHeadReadResponse200) & {
  headers: Headers;
};
export type journalHeadReadResponseError = (journalHeadReadResponseDefault) & {
  headers: Headers;
};

export const getJournalHeadReadUrl = (params?: JournalHeadReadParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/journal/head?${stringifiedParams}` : `/api/v1/journal/head`
}

/**
 * Последний seq и доменное время записи (AD-37), последний номер CA, режим часов.
 * @summary Голова журнала
 */
export const journalHeadRead = async (params?: JournalHeadReadParams, options?: RequestInit): Promise<journalHeadReadResponseSuccess> => {

  const res = await fetch(getJournalHeadReadUrl(params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: journalHeadReadResponseError['data'], status?: number} = new globalThis.Error();
    const data : journalHeadReadResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: journalHeadReadResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as journalHeadReadResponseSuccess
}





export const getJournalHeadReadQueryKey = (params?: MaybeRefOrGetter<JournalHeadReadParams>,) => {
    return [
    'api','v1','journal','head', ...(params ? [params] : [])
    ] as const;
    }


export const getJournalHeadReadQueryOptions = <TData = Awaited<ReturnType<typeof journalHeadRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(params?: MaybeRefOrGetter<JournalHeadReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof journalHeadRead>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getJournalHeadReadQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof journalHeadRead>>> = ({ signal }) => journalHeadRead(toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof journalHeadRead>>, TError, TData>
}

export type JournalHeadReadQueryResult = NonNullable<Awaited<ReturnType<typeof journalHeadRead>>>
export type JournalHeadReadQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Голова журнала
 */

export function useJournalHeadRead<TData = Awaited<ReturnType<typeof journalHeadRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 params?: MaybeRefOrGetter<JournalHeadReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof journalHeadRead>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getJournalHeadReadQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type journalEntryReadResponse200 = {
  data: JournalEntryView
  status: 200
}

export type journalEntryReadResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type journalEntryReadResponseSuccess = (journalEntryReadResponse200) & {
  headers: Headers;
};
export type journalEntryReadResponseError = (journalEntryReadResponseDefault) & {
  headers: Headers;
};

export const getJournalEntryReadUrl = (seq: number,) => {




  return `/api/v1/journal/${seq}`
}

/**
 * Одна запись по seq: открытые поля (AD-44), подписанты, статус подписи, содержимое.
 * @summary Запись журнала
 */
export const journalEntryRead = async (seq: number, options?: RequestInit): Promise<journalEntryReadResponseSuccess> => {

  const res = await fetch(getJournalEntryReadUrl(seq),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: journalEntryReadResponseError['data'], status?: number} = new globalThis.Error();
    const data : journalEntryReadResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: journalEntryReadResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as journalEntryReadResponseSuccess
}





export const getJournalEntryReadQueryKey = (seq: MaybeRefOrGetter<number>,) => {
    return [
    'api','v1','journal',seq
    ] as const;
    }


export const getJournalEntryReadQueryOptions = <TData = Awaited<ReturnType<typeof journalEntryRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(seq: MaybeRefOrGetter<number>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof journalEntryRead>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getJournalEntryReadQueryKey(seq);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof journalEntryRead>>> = ({ signal }) => journalEntryRead(toValue(seq), { signal, ...fetchOptions });





   return  { queryKey, queryFn, enabled: computed(() => toValue(seq) !== null && toValue(seq) !== undefined), ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof journalEntryRead>>, TError, TData>
}

export type JournalEntryReadQueryResult = NonNullable<Awaited<ReturnType<typeof journalEntryRead>>>
export type JournalEntryReadQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Запись журнала
 */

export function useJournalEntryRead<TData = Awaited<ReturnType<typeof journalEntryRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 seq: MaybeRefOrGetter<number>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof journalEntryRead>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getJournalEntryReadQueryOptions(seq,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type processLiveMapReadResponse200 = {
  data: LiveMap
  status: 200
}

export type processLiveMapReadResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type processLiveMapReadResponseSuccess = (processLiveMapReadResponse200) & {
  headers: Headers;
};
export type processLiveMapReadResponseError = (processLiveMapReadResponseDefault) & {
  headers: Headers;
};

export const getProcessLiveMapReadUrl = (params?: ProcessLiveMapReadParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/live-map?${stringifiedParams}` : `/api/v1/live-map`
}

/**
 * FR-1…FR-5, FR-9: схема действующей (или выбранной) версии, изделия-точки по step_key, счётчики узлов за период (считает analytics), ограничение линии и аномалии, узлы «оценка невозможна», режим инцидента. Запрос на момент — та же свёртка без записи (AD-22).
 * @summary Живая карта процесса
 */
export const processLiveMapRead = async (params?: ProcessLiveMapReadParams, options?: RequestInit): Promise<processLiveMapReadResponseSuccess> => {

  const res = await fetch(getProcessLiveMapReadUrl(params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: processLiveMapReadResponseError['data'], status?: number} = new globalThis.Error();
    const data : processLiveMapReadResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: processLiveMapReadResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as processLiveMapReadResponseSuccess
}





export const getProcessLiveMapReadQueryKey = (params?: MaybeRefOrGetter<ProcessLiveMapReadParams>,) => {
    return [
    'api','v1','live-map', ...(params ? [params] : [])
    ] as const;
    }


export const getProcessLiveMapReadQueryOptions = <TData = Awaited<ReturnType<typeof processLiveMapRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(params?: MaybeRefOrGetter<ProcessLiveMapReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof processLiveMapRead>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getProcessLiveMapReadQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof processLiveMapRead>>> = ({ signal }) => processLiveMapRead(toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof processLiveMapRead>>, TError, TData>
}

export type ProcessLiveMapReadQueryResult = NonNullable<Awaited<ReturnType<typeof processLiveMapRead>>>
export type ProcessLiveMapReadQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Живая карта процесса
 */

export function useProcessLiveMapRead<TData = Awaited<ReturnType<typeof processLiveMapRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 params?: MaybeRefOrGetter<ProcessLiveMapReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof processLiveMapRead>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getProcessLiveMapReadQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type nonconformityLotResolveResponse200 = {
  data: Receipt
  status: 200
}

export type nonconformityLotResolveResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type nonconformityLotResolveResponseSuccess = (nonconformityLotResolveResponse200) & {
  headers: Headers;
};
export type nonconformityLotResolveResponseError = (nonconformityLotResolveResponseDefault) & {
  headers: Headers;
};

export const getNonconformityLotResolveUrl = (lotId: string,) => {




  return `/api/v1/lots/${lotId}/resolve`
}

/**
 * Входной контроль (ЗТ-1): принять, принять частично, отклонить, недостаточно данных — по результатам методов.
 * @summary Решение по партии
 */
export const nonconformityLotResolve = async (lotId: string,
    resolveLot: ResolveLot, options?: RequestInit): Promise<nonconformityLotResolveResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getNonconformityLotResolveUrl(lotId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(resolveLot)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: nonconformityLotResolveResponseError['data'], status?: number} = new globalThis.Error();
    const data : nonconformityLotResolveResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: nonconformityLotResolveResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as nonconformityLotResolveResponseSuccess
}





export const getNonconformityLotResolveMutationKey = () => ['nonconformityLotResolve'] as const;

export const getNonconformityLotResolveMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof nonconformityLotResolve>>, TError,NonconformityLotResolveMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof nonconformityLotResolve>>, TError,NonconformityLotResolveMutationVariables, TContext> => {

const mutationKey = getNonconformityLotResolveMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof nonconformityLotResolve>>, NonconformityLotResolveMutationVariables> = (props) => {
          const {lotId,data} = props ?? {};

          return  nonconformityLotResolve(lotId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type NonconformityLotResolveMutationResult = NonNullable<Awaited<ReturnType<typeof nonconformityLotResolve>>>
    export type NonconformityLotResolveMutationBody = ResolveLot
    export type NonconformityLotResolveMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type NonconformityLotResolveMutationVariables = {lotId: string;data: ResolveLot}

    /**
 * @summary Решение по партии
 */
export const useNonconformityLotResolve = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof nonconformityLotResolve>>, TError,NonconformityLotResolveMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof nonconformityLotResolve>>,
        TError,
        NonconformityLotResolveMutationVariables,
        TContext
      > => {
      return useMutation(getNonconformityLotResolveMutationOptions(options), queryClient);
    }

export type analyticsNodeCountersReadResponse200 = {
  data: NodeCounterSet
  status: 200
}

export type analyticsNodeCountersReadResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type analyticsNodeCountersReadResponseSuccess = (analyticsNodeCountersReadResponse200) & {
  headers: Headers;
};
export type analyticsNodeCountersReadResponseError = (analyticsNodeCountersReadResponseDefault) & {
  headers: Headers;
};

export const getAnalyticsNodeCountersReadUrl = (params?: AnalyticsNodeCountersReadParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/metrics/node-counters?${stringifiedParams}` : `/api/v1/metrics/node-counters`
}

/**
 * FR-2, FR-3, FR-5, AD-21: по step_key — очередь, в работе, прошло, дефекты за период; узел-ограничение линии и аномалии считает сервер; узлы без данных источника — «оценка невозможна», не «норма».
 * @summary Счётчики узлов живой карты
 */
export const analyticsNodeCountersRead = async (params?: AnalyticsNodeCountersReadParams, options?: RequestInit): Promise<analyticsNodeCountersReadResponseSuccess> => {

  const res = await fetch(getAnalyticsNodeCountersReadUrl(params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: analyticsNodeCountersReadResponseError['data'], status?: number} = new globalThis.Error();
    const data : analyticsNodeCountersReadResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: analyticsNodeCountersReadResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as analyticsNodeCountersReadResponseSuccess
}





export const getAnalyticsNodeCountersReadQueryKey = (params?: MaybeRefOrGetter<AnalyticsNodeCountersReadParams>,) => {
    return [
    'api','v1','metrics','node-counters', ...(params ? [params] : [])
    ] as const;
    }


export const getAnalyticsNodeCountersReadQueryOptions = <TData = Awaited<ReturnType<typeof analyticsNodeCountersRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(params?: MaybeRefOrGetter<AnalyticsNodeCountersReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof analyticsNodeCountersRead>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getAnalyticsNodeCountersReadQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof analyticsNodeCountersRead>>> = ({ signal }) => analyticsNodeCountersRead(toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof analyticsNodeCountersRead>>, TError, TData>
}

export type AnalyticsNodeCountersReadQueryResult = NonNullable<Awaited<ReturnType<typeof analyticsNodeCountersRead>>>
export type AnalyticsNodeCountersReadQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Счётчики узлов живой карты
 */

export function useAnalyticsNodeCountersRead<TData = Awaited<ReturnType<typeof analyticsNodeCountersRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 params?: MaybeRefOrGetter<AnalyticsNodeCountersReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof analyticsNodeCountersRead>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getAnalyticsNodeCountersReadQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type analyticsTileListResponse200 = {
  data: MetricTileList
  status: 200
}

export type analyticsTileListResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type analyticsTileListResponseSuccess = (analyticsTileListResponse200) & {
  headers: Headers;
};
export type analyticsTileListResponseError = (analyticsTileListResponseDefault) & {
  headers: Headers;
};

export const getAnalyticsTileListUrl = (params?: AnalyticsTileListParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/metrics/tiles?${stringifiedParams}` : `/api/v1/metrics/tiles`
}

/**
 * Стол руководителя: проверено изделий, с подтверждёнными несоответствиями, прохождение контроля с первого раза, дефекты по видам, причина установлена / не установлена, время детали в системе, ожидание. «Оценка невозможна» ≠ ноль (NFR-UI-4).
 * @summary Плитки показателей
 */
export const analyticsTileList = async (params?: AnalyticsTileListParams, options?: RequestInit): Promise<analyticsTileListResponseSuccess> => {

  const res = await fetch(getAnalyticsTileListUrl(params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: analyticsTileListResponseError['data'], status?: number} = new globalThis.Error();
    const data : analyticsTileListResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: analyticsTileListResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as analyticsTileListResponseSuccess
}





export const getAnalyticsTileListQueryKey = (params?: MaybeRefOrGetter<AnalyticsTileListParams>,) => {
    return [
    'api','v1','metrics','tiles', ...(params ? [params] : [])
    ] as const;
    }


export const getAnalyticsTileListQueryOptions = <TData = Awaited<ReturnType<typeof analyticsTileList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(params?: MaybeRefOrGetter<AnalyticsTileListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof analyticsTileList>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getAnalyticsTileListQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof analyticsTileList>>> = ({ signal }) => analyticsTileList(toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof analyticsTileList>>, TError, TData>
}

export type AnalyticsTileListQueryResult = NonNullable<Awaited<ReturnType<typeof analyticsTileList>>>
export type AnalyticsTileListQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Плитки показателей
 */

export function useAnalyticsTileList<TData = Awaited<ReturnType<typeof analyticsTileList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 params?: MaybeRefOrGetter<AnalyticsTileListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof analyticsTileList>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getAnalyticsTileListQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type nonconformityNonconformityListResponse200 = {
  data: NCList
  status: 200
}

export type nonconformityNonconformityListResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type nonconformityNonconformityListResponseSuccess = (nonconformityNonconformityListResponse200) & {
  headers: Headers;
};
export type nonconformityNonconformityListResponseError = (nonconformityNonconformityListResponseDefault) & {
  headers: Headers;
};

export const getNonconformityNonconformityListUrl = (params?: NonconformityNonconformityListParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/nonconformities?${stringifiedParams}` : `/api/v1/nonconformities`
}

/**
 * Список несоответствий с фильтром по изделию и статусу (журнал регистрации несоответствий — проекция).
 * @summary Несоответствия
 */
export const nonconformityNonconformityList = async (params?: NonconformityNonconformityListParams, options?: RequestInit): Promise<nonconformityNonconformityListResponseSuccess> => {

  const res = await fetch(getNonconformityNonconformityListUrl(params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: nonconformityNonconformityListResponseError['data'], status?: number} = new globalThis.Error();
    const data : nonconformityNonconformityListResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: nonconformityNonconformityListResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as nonconformityNonconformityListResponseSuccess
}





export const getNonconformityNonconformityListQueryKey = (params?: MaybeRefOrGetter<NonconformityNonconformityListParams>,) => {
    return [
    'api','v1','nonconformities', ...(params ? [params] : [])
    ] as const;
    }


export const getNonconformityNonconformityListQueryOptions = <TData = Awaited<ReturnType<typeof nonconformityNonconformityList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(params?: MaybeRefOrGetter<NonconformityNonconformityListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof nonconformityNonconformityList>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getNonconformityNonconformityListQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof nonconformityNonconformityList>>> = ({ signal }) => nonconformityNonconformityList(toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof nonconformityNonconformityList>>, TError, TData>
}

export type NonconformityNonconformityListQueryResult = NonNullable<Awaited<ReturnType<typeof nonconformityNonconformityList>>>
export type NonconformityNonconformityListQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Несоответствия
 */

export function useNonconformityNonconformityList<TData = Awaited<ReturnType<typeof nonconformityNonconformityList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 params?: MaybeRefOrGetter<NonconformityNonconformityListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof nonconformityNonconformityList>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getNonconformityNonconformityListQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type nonconformityCardReadResponse200 = {
  data: NCCard
  status: 200
}

export type nonconformityCardReadResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type nonconformityCardReadResponseSuccess = (nonconformityCardReadResponse200) & {
  headers: Headers;
};
export type nonconformityCardReadResponseError = (nonconformityCardReadResponseDefault) & {
  headers: Headers;
};

export const getNonconformityCardReadUrl = (ncId: string,
    params?: NonconformityCardReadParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/nonconformities/${ncId}?${stringifiedParams}` : `/api/v1/nonconformities/${ncId}`
}

/**
 * FR-51: три зоны — «что произошло» (до операции / операция: станок, инструмент, программа, исполнитель / после), «доказательства», «что решить»; блок «почему система это предлагает»; раздельно исходный сигнал, анализ системы (все версии вывода с причиной пересмотра) и решения людей; оси статуса изделия.
 * @summary Карточка несоответствия
 */
export const nonconformityCardRead = async (ncId: string,
    params?: NonconformityCardReadParams, options?: RequestInit): Promise<nonconformityCardReadResponseSuccess> => {

  const res = await fetch(getNonconformityCardReadUrl(ncId,params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: nonconformityCardReadResponseError['data'], status?: number} = new globalThis.Error();
    const data : nonconformityCardReadResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: nonconformityCardReadResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as nonconformityCardReadResponseSuccess
}





export const getNonconformityCardReadQueryKey = (ncId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<NonconformityCardReadParams>,) => {
    return [
    'api','v1','nonconformities',ncId, ...(params ? [params] : [])
    ] as const;
    }


export const getNonconformityCardReadQueryOptions = <TData = Awaited<ReturnType<typeof nonconformityCardRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(ncId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<NonconformityCardReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof nonconformityCardRead>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getNonconformityCardReadQueryKey(ncId,params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof nonconformityCardRead>>> = ({ signal }) => nonconformityCardRead(toValue(ncId),toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, enabled: computed(() => toValue(ncId) !== null && toValue(ncId) !== undefined), ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof nonconformityCardRead>>, TError, TData>
}

export type NonconformityCardReadQueryResult = NonNullable<Awaited<ReturnType<typeof nonconformityCardRead>>>
export type NonconformityCardReadQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Карточка несоответствия
 */

export function useNonconformityCardRead<TData = Awaited<ReturnType<typeof nonconformityCardRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 ncId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<NonconformityCardReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof nonconformityCardRead>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getNonconformityCardReadQueryOptions(ncId,params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type analysisCircumstancesReadResponse200 = {
  data: Circumstances
  status: 200
}

export type analysisCircumstancesReadResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type analysisCircumstancesReadResponseSuccess = (analysisCircumstancesReadResponse200) & {
  headers: Headers;
};
export type analysisCircumstancesReadResponseError = (analysisCircumstancesReadResponseDefault) & {
  headers: Headers;
};

export const getAnalysisCircumstancesReadUrl = (ncId: string,
    params?: AnalysisCircumstancesReadParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/nonconformities/${ncId}/circumstances?${stringifiedParams}` : `/api/v1/nonconformities/${ncId}/circumstances`
}

/**
 * FR-58, FR-153: проекция analysis.circumstances — события изделия, исполнителя и оборудования на общей шкале, окно возможного возникновения (от последнего подтверждённо нормального состояния до первой находки), ссылки на кадры и записи журнала. Формулировка — «возможные обстоятельства», не «причина».
 * @summary Разбор обстоятельств
 */
export const analysisCircumstancesRead = async (ncId: string,
    params?: AnalysisCircumstancesReadParams, options?: RequestInit): Promise<analysisCircumstancesReadResponseSuccess> => {

  const res = await fetch(getAnalysisCircumstancesReadUrl(ncId,params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: analysisCircumstancesReadResponseError['data'], status?: number} = new globalThis.Error();
    const data : analysisCircumstancesReadResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: analysisCircumstancesReadResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as analysisCircumstancesReadResponseSuccess
}





export const getAnalysisCircumstancesReadQueryKey = (ncId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<AnalysisCircumstancesReadParams>,) => {
    return [
    'api','v1','nonconformities',ncId,'circumstances', ...(params ? [params] : [])
    ] as const;
    }


export const getAnalysisCircumstancesReadQueryOptions = <TData = Awaited<ReturnType<typeof analysisCircumstancesRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(ncId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<AnalysisCircumstancesReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof analysisCircumstancesRead>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getAnalysisCircumstancesReadQueryKey(ncId,params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof analysisCircumstancesRead>>> = ({ signal }) => analysisCircumstancesRead(toValue(ncId),toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, enabled: computed(() => toValue(ncId) !== null && toValue(ncId) !== undefined), ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof analysisCircumstancesRead>>, TError, TData>
}

export type AnalysisCircumstancesReadQueryResult = NonNullable<Awaited<ReturnType<typeof analysisCircumstancesRead>>>
export type AnalysisCircumstancesReadQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Разбор обстоятельств
 */

export function useAnalysisCircumstancesRead<TData = Awaited<ReturnType<typeof analysisCircumstancesRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 ncId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<AnalysisCircumstancesReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof analysisCircumstancesRead>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getAnalysisCircumstancesReadQueryOptions(ncId,params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type nonconformityNonconformityCloseResponse200 = {
  data: Receipt
  status: 200
}

export type nonconformityNonconformityCloseResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type nonconformityNonconformityCloseResponseSuccess = (nonconformityNonconformityCloseResponse200) & {
  headers: Headers;
};
export type nonconformityNonconformityCloseResponseError = (nonconformityNonconformityCloseResponseDefault) & {
  headers: Headers;
};

export const getNonconformityNonconformityCloseUrl = (ncId: string,) => {




  return `/api/v1/nonconformities/${ncId}/close`
}

/**
 * Итог несоответствия после выполнения решения.
 * @summary Закрыть несоответствие
 */
export const nonconformityNonconformityClose = async (ncId: string,
    closeNonconformity: CloseNonconformity, options?: RequestInit): Promise<nonconformityNonconformityCloseResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getNonconformityNonconformityCloseUrl(ncId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(closeNonconformity)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: nonconformityNonconformityCloseResponseError['data'], status?: number} = new globalThis.Error();
    const data : nonconformityNonconformityCloseResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: nonconformityNonconformityCloseResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as nonconformityNonconformityCloseResponseSuccess
}





export const getNonconformityNonconformityCloseMutationKey = () => ['nonconformityNonconformityClose'] as const;

export const getNonconformityNonconformityCloseMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof nonconformityNonconformityClose>>, TError,NonconformityNonconformityCloseMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof nonconformityNonconformityClose>>, TError,NonconformityNonconformityCloseMutationVariables, TContext> => {

const mutationKey = getNonconformityNonconformityCloseMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof nonconformityNonconformityClose>>, NonconformityNonconformityCloseMutationVariables> = (props) => {
          const {ncId,data} = props ?? {};

          return  nonconformityNonconformityClose(ncId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type NonconformityNonconformityCloseMutationResult = NonNullable<Awaited<ReturnType<typeof nonconformityNonconformityClose>>>
    export type NonconformityNonconformityCloseMutationBody = CloseNonconformity
    export type NonconformityNonconformityCloseMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type NonconformityNonconformityCloseMutationVariables = {ncId: string;data: CloseNonconformity}

    /**
 * @summary Закрыть несоответствие
 */
export const useNonconformityNonconformityClose = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof nonconformityNonconformityClose>>, TError,NonconformityNonconformityCloseMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof nonconformityNonconformityClose>>,
        TError,
        NonconformityNonconformityCloseMutationVariables,
        TContext
      > => {
      return useMutation(getNonconformityNonconformityCloseMutationOptions(options), queryClient);
    }

export type nonconformityNonconformityConfirmResponse200 = {
  data: Receipt
  status: 200
}

export type nonconformityNonconformityConfirmResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type nonconformityNonconformityConfirmResponseSuccess = (nonconformityNonconformityConfirmResponse200) & {
  headers: Headers;
};
export type nonconformityNonconformityConfirmResponseError = (nonconformityNonconformityConfirmResponseDefault) & {
  headers: Headers;
};

export const getNonconformityNonconformityConfirmUrl = (ncId: string,) => {




  return `/api/v1/nonconformities/${ncId}/confirm`
}

/**
 * FR-52: контролёр подтверждает несоответствие по сигналам — защитное действие, критическое (AD-28), подпись уровня 2.
 * @summary Подтвердить несоответствие
 */
export const nonconformityNonconformityConfirm = async (ncId: string,
    confirmNonconformity: ConfirmNonconformity, options?: RequestInit): Promise<nonconformityNonconformityConfirmResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getNonconformityNonconformityConfirmUrl(ncId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(confirmNonconformity)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: nonconformityNonconformityConfirmResponseError['data'], status?: number} = new globalThis.Error();
    const data : nonconformityNonconformityConfirmResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: nonconformityNonconformityConfirmResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as nonconformityNonconformityConfirmResponseSuccess
}





export const getNonconformityNonconformityConfirmMutationKey = () => ['nonconformityNonconformityConfirm'] as const;

export const getNonconformityNonconformityConfirmMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof nonconformityNonconformityConfirm>>, TError,NonconformityNonconformityConfirmMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof nonconformityNonconformityConfirm>>, TError,NonconformityNonconformityConfirmMutationVariables, TContext> => {

const mutationKey = getNonconformityNonconformityConfirmMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof nonconformityNonconformityConfirm>>, NonconformityNonconformityConfirmMutationVariables> = (props) => {
          const {ncId,data} = props ?? {};

          return  nonconformityNonconformityConfirm(ncId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type NonconformityNonconformityConfirmMutationResult = NonNullable<Awaited<ReturnType<typeof nonconformityNonconformityConfirm>>>
    export type NonconformityNonconformityConfirmMutationBody = ConfirmNonconformity
    export type NonconformityNonconformityConfirmMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type NonconformityNonconformityConfirmMutationVariables = {ncId: string;data: ConfirmNonconformity}

    /**
 * @summary Подтвердить несоответствие
 */
export const useNonconformityNonconformityConfirm = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof nonconformityNonconformityConfirm>>, TError,NonconformityNonconformityConfirmMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof nonconformityNonconformityConfirm>>,
        TError,
        NonconformityNonconformityConfirmMutationVariables,
        TContext
      > => {
      return useMutation(getNonconformityNonconformityConfirmMutationOptions(options), queryClient);
    }

export type nonconformityDispositionSetResponse200 = {
  data: Receipt
  status: 200
}

export type nonconformityDispositionSetResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type nonconformityDispositionSetResponseSuccess = (nonconformityDispositionSetResponse200) & {
  headers: Headers;
};
export type nonconformityDispositionSetResponseError = (nonconformityDispositionSetResponseDefault) & {
  headers: Headers;
};

export const getNonconformityDispositionSetUrl = (ncId: string,) => {




  return `/api/v1/nonconformities/${ncId}/disposition`
}

/**
 * FR-53: переделка / ремонт / как есть / списать / вернуть поставщику — необратимое решение уполномоченного по маршруту (AD-27, AD-43); ремонт и «как есть» — только с действующим разрешением на отклонение (FR-54, nonconformity.concession_required).
 * @summary Решение по несоответствию
 */
export const nonconformityDispositionSet = async (ncId: string,
    setDisposition: SetDisposition, options?: RequestInit): Promise<nonconformityDispositionSetResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getNonconformityDispositionSetUrl(ncId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(setDisposition)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: nonconformityDispositionSetResponseError['data'], status?: number} = new globalThis.Error();
    const data : nonconformityDispositionSetResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: nonconformityDispositionSetResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as nonconformityDispositionSetResponseSuccess
}





export const getNonconformityDispositionSetMutationKey = () => ['nonconformityDispositionSet'] as const;

export const getNonconformityDispositionSetMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof nonconformityDispositionSet>>, TError,NonconformityDispositionSetMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof nonconformityDispositionSet>>, TError,NonconformityDispositionSetMutationVariables, TContext> => {

const mutationKey = getNonconformityDispositionSetMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof nonconformityDispositionSet>>, NonconformityDispositionSetMutationVariables> = (props) => {
          const {ncId,data} = props ?? {};

          return  nonconformityDispositionSet(ncId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type NonconformityDispositionSetMutationResult = NonNullable<Awaited<ReturnType<typeof nonconformityDispositionSet>>>
    export type NonconformityDispositionSetMutationBody = SetDisposition
    export type NonconformityDispositionSetMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type NonconformityDispositionSetMutationVariables = {ncId: string;data: SetDisposition}

    /**
 * @summary Решение по несоответствию
 */
export const useNonconformityDispositionSet = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof nonconformityDispositionSet>>, TError,NonconformityDispositionSetMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof nonconformityDispositionSet>>,
        TError,
        NonconformityDispositionSetMutationVariables,
        TContext
      > => {
      return useMutation(getNonconformityDispositionSetMutationOptions(options), queryClient);
    }

export type nonconformityDispositionVerifyResponse200 = {
  data: Receipt
  status: 200
}

export type nonconformityDispositionVerifyResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type nonconformityDispositionVerifyResponseSuccess = (nonconformityDispositionVerifyResponse200) & {
  headers: Headers;
};
export type nonconformityDispositionVerifyResponseError = (nonconformityDispositionVerifyResponseDefault) & {
  headers: Headers;
};

export const getNonconformityDispositionVerifyUrl = (ncId: string,) => {




  return `/api/v1/nonconformities/${ncId}/disposition/verify`
}

/**
 * Повторная проверка после переделки или ремонта подтверждает выполнение решения — разрешающее действие.
 * @summary Подтвердить выполнение решения
 */
export const nonconformityDispositionVerify = async (ncId: string,
    verifyDisposition: VerifyDisposition, options?: RequestInit): Promise<nonconformityDispositionVerifyResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getNonconformityDispositionVerifyUrl(ncId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(verifyDisposition)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: nonconformityDispositionVerifyResponseError['data'], status?: number} = new globalThis.Error();
    const data : nonconformityDispositionVerifyResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: nonconformityDispositionVerifyResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as nonconformityDispositionVerifyResponseSuccess
}





export const getNonconformityDispositionVerifyMutationKey = () => ['nonconformityDispositionVerify'] as const;

export const getNonconformityDispositionVerifyMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof nonconformityDispositionVerify>>, TError,NonconformityDispositionVerifyMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof nonconformityDispositionVerify>>, TError,NonconformityDispositionVerifyMutationVariables, TContext> => {

const mutationKey = getNonconformityDispositionVerifyMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof nonconformityDispositionVerify>>, NonconformityDispositionVerifyMutationVariables> = (props) => {
          const {ncId,data} = props ?? {};

          return  nonconformityDispositionVerify(ncId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type NonconformityDispositionVerifyMutationResult = NonNullable<Awaited<ReturnType<typeof nonconformityDispositionVerify>>>
    export type NonconformityDispositionVerifyMutationBody = VerifyDisposition
    export type NonconformityDispositionVerifyMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type NonconformityDispositionVerifyMutationVariables = {ncId: string;data: VerifyDisposition}

    /**
 * @summary Подтвердить выполнение решения
 */
export const useNonconformityDispositionVerify = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof nonconformityDispositionVerify>>, TError,NonconformityDispositionVerifyMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof nonconformityDispositionVerify>>,
        TError,
        NonconformityDispositionVerifyMutationVariables,
        TContext
      > => {
      return useMutation(getNonconformityDispositionVerifyMutationOptions(options), queryClient);
    }

export type analysisHypothesisListResponse200 = {
  data: Hypotheses
  status: 200
}

export type analysisHypothesisListResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type analysisHypothesisListResponseSuccess = (analysisHypothesisListResponse200) & {
  headers: Headers;
};
export type analysisHypothesisListResponseError = (analysisHypothesisListResponseDefault) & {
  headers: Headers;
};

export const getAnalysisHypothesisListUrl = (ncId: string,
    params?: AnalysisHypothesisListParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/nonconformities/${ncId}/hypotheses?${stringifiedParams}` : `/api/v1/nonconformities/${ncId}/hypotheses`
}

/**
 * FR-59, FR-60: версия вывода incident.hypothesis.computed и гипотезы, записанные людьми, с доводами «за» и «против»; нехватка сведений; похожие случаи. Гипотеза ≠ причина.
 * @summary Гипотезы причины и похожие случаи
 */
export const analysisHypothesisList = async (ncId: string,
    params?: AnalysisHypothesisListParams, options?: RequestInit): Promise<analysisHypothesisListResponseSuccess> => {

  const res = await fetch(getAnalysisHypothesisListUrl(ncId,params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: analysisHypothesisListResponseError['data'], status?: number} = new globalThis.Error();
    const data : analysisHypothesisListResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: analysisHypothesisListResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as analysisHypothesisListResponseSuccess
}





export const getAnalysisHypothesisListQueryKey = (ncId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<AnalysisHypothesisListParams>,) => {
    return [
    'api','v1','nonconformities',ncId,'hypotheses', ...(params ? [params] : [])
    ] as const;
    }


export const getAnalysisHypothesisListQueryOptions = <TData = Awaited<ReturnType<typeof analysisHypothesisList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(ncId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<AnalysisHypothesisListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof analysisHypothesisList>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getAnalysisHypothesisListQueryKey(ncId,params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof analysisHypothesisList>>> = ({ signal }) => analysisHypothesisList(toValue(ncId),toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, enabled: computed(() => toValue(ncId) !== null && toValue(ncId) !== undefined), ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof analysisHypothesisList>>, TError, TData>
}

export type AnalysisHypothesisListQueryResult = NonNullable<Awaited<ReturnType<typeof analysisHypothesisList>>>
export type AnalysisHypothesisListQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Гипотезы причины и похожие случаи
 */

export function useAnalysisHypothesisList<TData = Awaited<ReturnType<typeof analysisHypothesisList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 ncId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<AnalysisHypothesisListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof analysisHypothesisList>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getAnalysisHypothesisListQueryOptions(ncId,params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type analysisHypothesisRecordResponse200 = {
  data: Receipt
  status: 200
}

export type analysisHypothesisRecordResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type analysisHypothesisRecordResponseSuccess = (analysisHypothesisRecordResponse200) & {
  headers: Headers;
};
export type analysisHypothesisRecordResponseError = (analysisHypothesisRecordResponseDefault) & {
  headers: Headers;
};

export const getAnalysisHypothesisRecordUrl = (ncId: string,) => {




  return `/api/v1/nonconformities/${ncId}/hypotheses`
}

/**
 * FR-59: гипотеза причины человеком (почему возник / почему пропустили).
 * @summary Записать гипотезу
 */
export const analysisHypothesisRecord = async (ncId: string,
    recordHypothesis: RecordHypothesis, options?: RequestInit): Promise<analysisHypothesisRecordResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getAnalysisHypothesisRecordUrl(ncId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(recordHypothesis)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: analysisHypothesisRecordResponseError['data'], status?: number} = new globalThis.Error();
    const data : analysisHypothesisRecordResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: analysisHypothesisRecordResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as analysisHypothesisRecordResponseSuccess
}





export const getAnalysisHypothesisRecordMutationKey = () => ['analysisHypothesisRecord'] as const;

export const getAnalysisHypothesisRecordMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof analysisHypothesisRecord>>, TError,AnalysisHypothesisRecordMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof analysisHypothesisRecord>>, TError,AnalysisHypothesisRecordMutationVariables, TContext> => {

const mutationKey = getAnalysisHypothesisRecordMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof analysisHypothesisRecord>>, AnalysisHypothesisRecordMutationVariables> = (props) => {
          const {ncId,data} = props ?? {};

          return  analysisHypothesisRecord(ncId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type AnalysisHypothesisRecordMutationResult = NonNullable<Awaited<ReturnType<typeof analysisHypothesisRecord>>>
    export type AnalysisHypothesisRecordMutationBody = RecordHypothesis
    export type AnalysisHypothesisRecordMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type AnalysisHypothesisRecordMutationVariables = {ncId: string;data: RecordHypothesis}

    /**
 * @summary Записать гипотезу
 */
export const useAnalysisHypothesisRecord = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof analysisHypothesisRecord>>, TError,AnalysisHypothesisRecordMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof analysisHypothesisRecord>>,
        TError,
        AnalysisHypothesisRecordMutationVariables,
        TContext
      > => {
      return useMutation(getAnalysisHypothesisRecordMutationOptions(options), queryClient);
    }

export type analysisHypothesisRejectResponse200 = {
  data: Receipt
  status: 200
}

export type analysisHypothesisRejectResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type analysisHypothesisRejectResponseSuccess = (analysisHypothesisRejectResponse200) & {
  headers: Headers;
};
export type analysisHypothesisRejectResponseError = (analysisHypothesisRejectResponseDefault) & {
  headers: Headers;
};

export const getAnalysisHypothesisRejectUrl = (ncId: string,) => {




  return `/api/v1/nonconformities/${ncId}/hypotheses/reject`
}

/**
 * FR-59: гипотеза отклоняется с основанием; вывод системы не переписывается — отклонение записывается решением человека.
 * @summary Отклонить гипотезу
 */
export const analysisHypothesisReject = async (ncId: string,
    rejectHypothesis: RejectHypothesis, options?: RequestInit): Promise<analysisHypothesisRejectResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getAnalysisHypothesisRejectUrl(ncId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(rejectHypothesis)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: analysisHypothesisRejectResponseError['data'], status?: number} = new globalThis.Error();
    const data : analysisHypothesisRejectResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: analysisHypothesisRejectResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as analysisHypothesisRejectResponseSuccess
}





export const getAnalysisHypothesisRejectMutationKey = () => ['analysisHypothesisReject'] as const;

export const getAnalysisHypothesisRejectMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof analysisHypothesisReject>>, TError,AnalysisHypothesisRejectMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof analysisHypothesisReject>>, TError,AnalysisHypothesisRejectMutationVariables, TContext> => {

const mutationKey = getAnalysisHypothesisRejectMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof analysisHypothesisReject>>, AnalysisHypothesisRejectMutationVariables> = (props) => {
          const {ncId,data} = props ?? {};

          return  analysisHypothesisReject(ncId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type AnalysisHypothesisRejectMutationResult = NonNullable<Awaited<ReturnType<typeof analysisHypothesisReject>>>
    export type AnalysisHypothesisRejectMutationBody = RejectHypothesis
    export type AnalysisHypothesisRejectMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type AnalysisHypothesisRejectMutationVariables = {ncId: string;data: RejectHypothesis}

    /**
 * @summary Отклонить гипотезу
 */
export const useAnalysisHypothesisReject = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof analysisHypothesisReject>>, TError,AnalysisHypothesisRejectMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof analysisHypothesisReject>>,
        TError,
        AnalysisHypothesisRejectMutationVariables,
        TContext
      > => {
      return useMutation(getAnalysisHypothesisRejectMutationOptions(options), queryClient);
    }

export type analysisMeasurementRequestResponse200 = {
  data: Receipt
  status: 200
}

export type analysisMeasurementRequestResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type analysisMeasurementRequestResponseSuccess = (analysisMeasurementRequestResponse200) & {
  headers: Headers;
};
export type analysisMeasurementRequestResponseError = (analysisMeasurementRequestResponseDefault) & {
  headers: Headers;
};

export const getAnalysisMeasurementRequestUrl = (ncId: string,) => {




  return `/api/v1/nonconformities/${ncId}/measurements`
}

/**
 * Проверка гипотезы измерением: задачу исполнителю ставит notifications по записи запроса (тип записи — предложение контракта, см. docs/codegen.md).
 * @summary Запросить измерение
 */
export const analysisMeasurementRequest = async (ncId: string,
    requestMeasurement: RequestMeasurement, options?: RequestInit): Promise<analysisMeasurementRequestResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getAnalysisMeasurementRequestUrl(ncId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(requestMeasurement)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: analysisMeasurementRequestResponseError['data'], status?: number} = new globalThis.Error();
    const data : analysisMeasurementRequestResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: analysisMeasurementRequestResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as analysisMeasurementRequestResponseSuccess
}





export const getAnalysisMeasurementRequestMutationKey = () => ['analysisMeasurementRequest'] as const;

export const getAnalysisMeasurementRequestMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof analysisMeasurementRequest>>, TError,AnalysisMeasurementRequestMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof analysisMeasurementRequest>>, TError,AnalysisMeasurementRequestMutationVariables, TContext> => {

const mutationKey = getAnalysisMeasurementRequestMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof analysisMeasurementRequest>>, AnalysisMeasurementRequestMutationVariables> = (props) => {
          const {ncId,data} = props ?? {};

          return  analysisMeasurementRequest(ncId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type AnalysisMeasurementRequestMutationResult = NonNullable<Awaited<ReturnType<typeof analysisMeasurementRequest>>>
    export type AnalysisMeasurementRequestMutationBody = RequestMeasurement
    export type AnalysisMeasurementRequestMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type AnalysisMeasurementRequestMutationVariables = {ncId: string;data: RequestMeasurement}

    /**
 * @summary Запросить измерение
 */
export const useAnalysisMeasurementRequest = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof analysisMeasurementRequest>>, TError,AnalysisMeasurementRequestMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof analysisMeasurementRequest>>,
        TError,
        AnalysisMeasurementRequestMutationVariables,
        TContext
      > => {
      return useMutation(getAnalysisMeasurementRequestMutationOptions(options), queryClient);
    }

export type analysisSimilarListResponse200 = {
  data: SimilarCaseList
  status: 200
}

export type analysisSimilarListResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type analysisSimilarListResponseSuccess = (analysisSimilarListResponse200) & {
  headers: Headers;
};
export type analysisSimilarListResponseError = (analysisSimilarListResponseDefault) & {
  headers: Headers;
};

export const getAnalysisSimilarListUrl = (ncId: string,
    params?: AnalysisSimilarListParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/nonconformities/${ncId}/similar?${stringifiedParams}` : `/api/v1/nonconformities/${ncId}/similar`
}

/**
 * FR-60: прошлые несоответствия по виду дефекта, операции и оборудованию — причина, мера и её результат.
 * @summary Похожие случаи
 */
export const analysisSimilarList = async (ncId: string,
    params?: AnalysisSimilarListParams, options?: RequestInit): Promise<analysisSimilarListResponseSuccess> => {

  const res = await fetch(getAnalysisSimilarListUrl(ncId,params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: analysisSimilarListResponseError['data'], status?: number} = new globalThis.Error();
    const data : analysisSimilarListResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: analysisSimilarListResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as analysisSimilarListResponseSuccess
}





export const getAnalysisSimilarListQueryKey = (ncId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<AnalysisSimilarListParams>,) => {
    return [
    'api','v1','nonconformities',ncId,'similar', ...(params ? [params] : [])
    ] as const;
    }


export const getAnalysisSimilarListQueryOptions = <TData = Awaited<ReturnType<typeof analysisSimilarList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(ncId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<AnalysisSimilarListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof analysisSimilarList>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getAnalysisSimilarListQueryKey(ncId,params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof analysisSimilarList>>> = ({ signal }) => analysisSimilarList(toValue(ncId),toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, enabled: computed(() => toValue(ncId) !== null && toValue(ncId) !== undefined), ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof analysisSimilarList>>, TError, TData>
}

export type AnalysisSimilarListQueryResult = NonNullable<Awaited<ReturnType<typeof analysisSimilarList>>>
export type AnalysisSimilarListQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Похожие случаи
 */

export function useAnalysisSimilarList<TData = Awaited<ReturnType<typeof analysisSimilarList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 ncId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<AnalysisSimilarListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof analysisSimilarList>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getAnalysisSimilarListQueryOptions(ncId,params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type notificationsSummaryReadResponse200 = {
  data: NotificationSummary
  status: 200
}

export type notificationsSummaryReadResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type notificationsSummaryReadResponseSuccess = (notificationsSummaryReadResponse200) & {
  headers: Headers;
};
export type notificationsSummaryReadResponseError = (notificationsSummaryReadResponseDefault) & {
  headers: Headers;
};

export const getNotificationsSummaryReadUrl = (params?: NotificationsSummaryReadParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/notifications/summary?${stringifiedParams}` : `/api/v1/notifications/summary`
}

/**
 * FR-57: непрочитанные по видам — информация, тревога, задача, запрос решения.
 * @summary Сводка уведомлений для шапки
 */
export const notificationsSummaryRead = async (params?: NotificationsSummaryReadParams, options?: RequestInit): Promise<notificationsSummaryReadResponseSuccess> => {

  const res = await fetch(getNotificationsSummaryReadUrl(params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: notificationsSummaryReadResponseError['data'], status?: number} = new globalThis.Error();
    const data : notificationsSummaryReadResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: notificationsSummaryReadResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as notificationsSummaryReadResponseSuccess
}





export const getNotificationsSummaryReadQueryKey = (params?: MaybeRefOrGetter<NotificationsSummaryReadParams>,) => {
    return [
    'api','v1','notifications','summary', ...(params ? [params] : [])
    ] as const;
    }


export const getNotificationsSummaryReadQueryOptions = <TData = Awaited<ReturnType<typeof notificationsSummaryRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(params?: MaybeRefOrGetter<NotificationsSummaryReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof notificationsSummaryRead>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getNotificationsSummaryReadQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof notificationsSummaryRead>>> = ({ signal }) => notificationsSummaryRead(toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof notificationsSummaryRead>>, TError, TData>
}

export type NotificationsSummaryReadQueryResult = NonNullable<Awaited<ReturnType<typeof notificationsSummaryRead>>>
export type NotificationsSummaryReadQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Сводка уведомлений для шапки
 */

export function useNotificationsSummaryRead<TData = Awaited<ReturnType<typeof notificationsSummaryRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 params?: MaybeRefOrGetter<NotificationsSummaryReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof notificationsSummaryRead>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getNotificationsSummaryReadQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type processOperationFinishResponse200 = {
  data: Receipt
  status: 200
}

export type processOperationFinishResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type processOperationFinishResponseSuccess = (processOperationFinishResponse200) & {
  headers: Headers;
};
export type processOperationFinishResponseError = (processOperationFinishResponseDefault) & {
  headers: Headers;
};

export const getProcessOperationFinishUrl = (runId: string,) => {




  return `/api/v1/operation-runs/${runId}/finish`
}

/**
 * FR-137: завершена или прервана.
 * @summary Завершить операцию
 */
export const processOperationFinish = async (runId: string,
    finishOperation: FinishOperation, options?: RequestInit): Promise<processOperationFinishResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getProcessOperationFinishUrl(runId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(finishOperation)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: processOperationFinishResponseError['data'], status?: number} = new globalThis.Error();
    const data : processOperationFinishResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: processOperationFinishResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as processOperationFinishResponseSuccess
}





export const getProcessOperationFinishMutationKey = () => ['processOperationFinish'] as const;

export const getProcessOperationFinishMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof processOperationFinish>>, TError,ProcessOperationFinishMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof processOperationFinish>>, TError,ProcessOperationFinishMutationVariables, TContext> => {

const mutationKey = getProcessOperationFinishMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof processOperationFinish>>, ProcessOperationFinishMutationVariables> = (props) => {
          const {runId,data} = props ?? {};

          return  processOperationFinish(runId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type ProcessOperationFinishMutationResult = NonNullable<Awaited<ReturnType<typeof processOperationFinish>>>
    export type ProcessOperationFinishMutationBody = FinishOperation
    export type ProcessOperationFinishMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type ProcessOperationFinishMutationVariables = {runId: string;data: FinishOperation}

    /**
 * @summary Завершить операцию
 */
export const useProcessOperationFinish = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof processOperationFinish>>, TError,ProcessOperationFinishMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof processOperationFinish>>,
        TError,
        ProcessOperationFinishMutationVariables,
        TContext
      > => {
      return useMutation(getProcessOperationFinishMutationOptions(options), queryClient);
    }

export type processOperationPauseResponse200 = {
  data: Receipt
  status: 200
}

export type processOperationPauseResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type processOperationPauseResponseSuccess = (processOperationPauseResponse200) & {
  headers: Headers;
};
export type processOperationPauseResponseError = (processOperationPauseResponseDefault) & {
  headers: Headers;
};

export const getProcessOperationPauseUrl = (runId: string,) => {




  return `/api/v1/operation-runs/${runId}/pause`
}

/**
 * FR-137: причина паузы.
 * @summary Приостановить операцию
 */
export const processOperationPause = async (runId: string,
    pauseOperation: PauseOperation, options?: RequestInit): Promise<processOperationPauseResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getProcessOperationPauseUrl(runId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(pauseOperation)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: processOperationPauseResponseError['data'], status?: number} = new globalThis.Error();
    const data : processOperationPauseResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: processOperationPauseResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as processOperationPauseResponseSuccess
}





export const getProcessOperationPauseMutationKey = () => ['processOperationPause'] as const;

export const getProcessOperationPauseMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof processOperationPause>>, TError,ProcessOperationPauseMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof processOperationPause>>, TError,ProcessOperationPauseMutationVariables, TContext> => {

const mutationKey = getProcessOperationPauseMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof processOperationPause>>, ProcessOperationPauseMutationVariables> = (props) => {
          const {runId,data} = props ?? {};

          return  processOperationPause(runId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type ProcessOperationPauseMutationResult = NonNullable<Awaited<ReturnType<typeof processOperationPause>>>
    export type ProcessOperationPauseMutationBody = PauseOperation
    export type ProcessOperationPauseMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type ProcessOperationPauseMutationVariables = {runId: string;data: PauseOperation}

    /**
 * @summary Приостановить операцию
 */
export const useProcessOperationPause = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof processOperationPause>>, TError,ProcessOperationPauseMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof processOperationPause>>,
        TError,
        ProcessOperationPauseMutationVariables,
        TContext
      > => {
      return useMutation(getProcessOperationPauseMutationOptions(options), queryClient);
    }

export type machinelogsRunProfileReadResponse200 = {
  data: RunProfile
  status: 200
}

export type machinelogsRunProfileReadResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type machinelogsRunProfileReadResponseSuccess = (machinelogsRunProfileReadResponse200) & {
  headers: Headers;
};
export type machinelogsRunProfileReadResponseError = (machinelogsRunProfileReadResponseDefault) & {
  headers: Headers;
};

export const getMachinelogsRunProfileReadUrl = (runId: string,
    params?: MachinelogsRunProfileReadParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/operation-runs/${runId}/profile?${stringifiedParams}` : `/api/v1/operation-runs/${runId}/profile`
}

/**
 * FR-148: оборудование, программа, инструмент, сводки циклов против уставки и отклонения на окне выполнения; привязка — межизделийная стадия (AD-29).
 * @summary Профиль выполнения операции
 */
export const machinelogsRunProfileRead = async (runId: string,
    params?: MachinelogsRunProfileReadParams, options?: RequestInit): Promise<machinelogsRunProfileReadResponseSuccess> => {

  const res = await fetch(getMachinelogsRunProfileReadUrl(runId,params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: machinelogsRunProfileReadResponseError['data'], status?: number} = new globalThis.Error();
    const data : machinelogsRunProfileReadResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: machinelogsRunProfileReadResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as machinelogsRunProfileReadResponseSuccess
}





export const getMachinelogsRunProfileReadQueryKey = (runId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<MachinelogsRunProfileReadParams>,) => {
    return [
    'api','v1','operation-runs',runId,'profile', ...(params ? [params] : [])
    ] as const;
    }


export const getMachinelogsRunProfileReadQueryOptions = <TData = Awaited<ReturnType<typeof machinelogsRunProfileRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(runId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<MachinelogsRunProfileReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof machinelogsRunProfileRead>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getMachinelogsRunProfileReadQueryKey(runId,params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof machinelogsRunProfileRead>>> = ({ signal }) => machinelogsRunProfileRead(toValue(runId),toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, enabled: computed(() => toValue(runId) !== null && toValue(runId) !== undefined), ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof machinelogsRunProfileRead>>, TError, TData>
}

export type MachinelogsRunProfileReadQueryResult = NonNullable<Awaited<ReturnType<typeof machinelogsRunProfileRead>>>
export type MachinelogsRunProfileReadQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Профиль выполнения операции
 */

export function useMachinelogsRunProfileRead<TData = Awaited<ReturnType<typeof machinelogsRunProfileRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 runId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<MachinelogsRunProfileReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof machinelogsRunProfileRead>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getMachinelogsRunProfileReadQueryOptions(runId,params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type processOperationResumeResponse200 = {
  data: Receipt
  status: 200
}

export type processOperationResumeResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type processOperationResumeResponseSuccess = (processOperationResumeResponse200) & {
  headers: Headers;
};
export type processOperationResumeResponseError = (processOperationResumeResponseDefault) & {
  headers: Headers;
};

export const getProcessOperationResumeUrl = (runId: string,) => {




  return `/api/v1/operation-runs/${runId}/resume`
}

/**
 * FR-137.
 * @summary Продолжить операцию
 */
export const processOperationResume = async (runId: string,
    resumeOperation: ResumeOperation, options?: RequestInit): Promise<processOperationResumeResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getProcessOperationResumeUrl(runId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(resumeOperation)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: processOperationResumeResponseError['data'], status?: number} = new globalThis.Error();
    const data : processOperationResumeResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: processOperationResumeResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as processOperationResumeResponseSuccess
}





export const getProcessOperationResumeMutationKey = () => ['processOperationResume'] as const;

export const getProcessOperationResumeMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof processOperationResume>>, TError,ProcessOperationResumeMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof processOperationResume>>, TError,ProcessOperationResumeMutationVariables, TContext> => {

const mutationKey = getProcessOperationResumeMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof processOperationResume>>, ProcessOperationResumeMutationVariables> = (props) => {
          const {runId,data} = props ?? {};

          return  processOperationResume(runId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type ProcessOperationResumeMutationResult = NonNullable<Awaited<ReturnType<typeof processOperationResume>>>
    export type ProcessOperationResumeMutationBody = ResumeOperation
    export type ProcessOperationResumeMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type ProcessOperationResumeMutationVariables = {runId: string;data: ResumeOperation}

    /**
 * @summary Продолжить операцию
 */
export const useProcessOperationResume = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof processOperationResume>>, TError,ProcessOperationResumeMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof processOperationResume>>,
        TError,
        ProcessOperationResumeMutationVariables,
        TContext
      > => {
      return useMutation(getProcessOperationResumeMutationOptions(options), queryClient);
    }

export type opsHealthReadResponse200 = {
  data: OpsHealth
  status: 200
}

export type opsHealthReadResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type opsHealthReadResponseSuccess = (opsHealthReadResponse200) & {
  headers: Headers;
};
export type opsHealthReadResponseError = (opsHealthReadResponseDefault) & {
  headers: Headers;
};

export const getOpsHealthReadUrl = () => {




  return `/api/v1/ops/health`
}

/**
 * FR-127: сервисы и роли, очереди и отставание потребителей, карантин, интеграции, остановленные изделия, последний отчёт верификатора «по данным сервера».
 * @summary Состояние системы
 */
export const opsHealthRead = async ( options?: RequestInit): Promise<opsHealthReadResponseSuccess> => {

  const res = await fetch(getOpsHealthReadUrl(),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: opsHealthReadResponseError['data'], status?: number} = new globalThis.Error();
    const data : opsHealthReadResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: opsHealthReadResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as opsHealthReadResponseSuccess
}





export const getOpsHealthReadQueryKey = () => {
    return [
    'api','v1','ops','health'
    ] as const;
    }


export const getOpsHealthReadQueryOptions = <TData = Awaited<ReturnType<typeof opsHealthRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>( options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof opsHealthRead>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getOpsHealthReadQueryKey();



    const queryFn: QueryFunction<Awaited<ReturnType<typeof opsHealthRead>>> = ({ signal }) => opsHealthRead({ signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof opsHealthRead>>, TError, TData>
}

export type OpsHealthReadQueryResult = NonNullable<Awaited<ReturnType<typeof opsHealthRead>>>
export type OpsHealthReadQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Состояние системы
 */

export function useOpsHealthRead<TData = Awaited<ReturnType<typeof opsHealthRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof opsHealthRead>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getOpsHealthReadQueryOptions(options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type opsSettingListResponse200 = {
  data: SettingList
  status: 200
}

export type opsSettingListResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type opsSettingListResponseSuccess = (opsSettingListResponse200) & {
  headers: Headers;
};
export type opsSettingListResponseError = (opsSettingListResponseDefault) & {
  headers: Headers;
};

export const getOpsSettingListUrl = () => {




  return `/api/v1/ops/settings`
}

/**
 * AD-35, AD-36: адаптер каждого ведомого порта по ключу конфигурации, режим fixtures | live по модулям, включённые внешние системы.
 * @summary Настройки адаптеров
 */
export const opsSettingList = async ( options?: RequestInit): Promise<opsSettingListResponseSuccess> => {

  const res = await fetch(getOpsSettingListUrl(),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: opsSettingListResponseError['data'], status?: number} = new globalThis.Error();
    const data : opsSettingListResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: opsSettingListResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as opsSettingListResponseSuccess
}





export const getOpsSettingListQueryKey = () => {
    return [
    'api','v1','ops','settings'
    ] as const;
    }


export const getOpsSettingListQueryOptions = <TData = Awaited<ReturnType<typeof opsSettingList>>, TError = globalThis.Error & { info?: Problem; status?: number }>( options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof opsSettingList>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getOpsSettingListQueryKey();



    const queryFn: QueryFunction<Awaited<ReturnType<typeof opsSettingList>>> = ({ signal }) => opsSettingList({ signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof opsSettingList>>, TError, TData>
}

export type OpsSettingListQueryResult = NonNullable<Awaited<ReturnType<typeof opsSettingList>>>
export type OpsSettingListQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Настройки адаптеров
 */

export function useOpsSettingList<TData = Awaited<ReturnType<typeof opsSettingList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof opsSettingList>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getOpsSettingListQueryOptions(options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type opsSourceDisableResponse200 = {
  data: Receipt
  status: 200
}

export type opsSourceDisableResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type opsSourceDisableResponseSuccess = (opsSourceDisableResponse200) & {
  headers: Headers;
};
export type opsSourceDisableResponseError = (opsSourceDisableResponseDefault) & {
  headers: Headers;
};

export const getOpsSourceDisableUrl = (sourceId: string,) => {




  return `/api/v1/ops/sources/${sourceId}/disable`
}

/**
 * AD-28: отключение источника — критическое действие администратора (группа admin_security); защитное.
 * @summary Отключить источник
 */
export const opsSourceDisable = async (sourceId: string,
    switchSource: SwitchSource, options?: RequestInit): Promise<opsSourceDisableResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getOpsSourceDisableUrl(sourceId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(switchSource)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: opsSourceDisableResponseError['data'], status?: number} = new globalThis.Error();
    const data : opsSourceDisableResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: opsSourceDisableResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as opsSourceDisableResponseSuccess
}





export const getOpsSourceDisableMutationKey = () => ['opsSourceDisable'] as const;

export const getOpsSourceDisableMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof opsSourceDisable>>, TError,OpsSourceDisableMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof opsSourceDisable>>, TError,OpsSourceDisableMutationVariables, TContext> => {

const mutationKey = getOpsSourceDisableMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof opsSourceDisable>>, OpsSourceDisableMutationVariables> = (props) => {
          const {sourceId,data} = props ?? {};

          return  opsSourceDisable(sourceId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type OpsSourceDisableMutationResult = NonNullable<Awaited<ReturnType<typeof opsSourceDisable>>>
    export type OpsSourceDisableMutationBody = SwitchSource
    export type OpsSourceDisableMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type OpsSourceDisableMutationVariables = {sourceId: string;data: SwitchSource}

    /**
 * @summary Отключить источник
 */
export const useOpsSourceDisable = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof opsSourceDisable>>, TError,OpsSourceDisableMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof opsSourceDisable>>,
        TError,
        OpsSourceDisableMutationVariables,
        TContext
      > => {
      return useMutation(getOpsSourceDisableMutationOptions(options), queryClient);
    }

export type opsSourceEnableResponse200 = {
  data: Receipt
  status: 200
}

export type opsSourceEnableResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type opsSourceEnableResponseSuccess = (opsSourceEnableResponse200) & {
  headers: Headers;
};
export type opsSourceEnableResponseError = (opsSourceEnableResponseDefault) & {
  headers: Headers;
};

export const getOpsSourceEnableUrl = (sourceId: string,) => {




  return `/api/v1/ops/sources/${sourceId}/enable`
}

/**
 * AD-28: включение источника — критическое разрешающее действие администратора.
 * @summary Включить источник
 */
export const opsSourceEnable = async (sourceId: string,
    switchSource: SwitchSource, options?: RequestInit): Promise<opsSourceEnableResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getOpsSourceEnableUrl(sourceId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(switchSource)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: opsSourceEnableResponseError['data'], status?: number} = new globalThis.Error();
    const data : opsSourceEnableResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: opsSourceEnableResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as opsSourceEnableResponseSuccess
}





export const getOpsSourceEnableMutationKey = () => ['opsSourceEnable'] as const;

export const getOpsSourceEnableMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof opsSourceEnable>>, TError,OpsSourceEnableMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof opsSourceEnable>>, TError,OpsSourceEnableMutationVariables, TContext> => {

const mutationKey = getOpsSourceEnableMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof opsSourceEnable>>, OpsSourceEnableMutationVariables> = (props) => {
          const {sourceId,data} = props ?? {};

          return  opsSourceEnable(sourceId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type OpsSourceEnableMutationResult = NonNullable<Awaited<ReturnType<typeof opsSourceEnable>>>
    export type OpsSourceEnableMutationBody = SwitchSource
    export type OpsSourceEnableMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type OpsSourceEnableMutationVariables = {sourceId: string;data: SwitchSource}

    /**
 * @summary Включить источник
 */
export const useOpsSourceEnable = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof opsSourceEnable>>, TError,OpsSourceEnableMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof opsSourceEnable>>,
        TError,
        OpsSourceEnableMutationVariables,
        TContext
      > => {
      return useMutation(getOpsSourceEnableMutationOptions(options), queryClient);
    }

export type opsStoppedItemListResponse200 = {
  data: StoppedItemList
  status: 200
}

export type opsStoppedItemListResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type opsStoppedItemListResponseSuccess = (opsStoppedItemListResponse200) & {
  headers: Headers;
};
export type opsStoppedItemListResponseError = (opsStoppedItemListResponseDefault) & {
  headers: Headers;
};

export const getOpsStoppedItemListUrl = (params?: OpsStoppedItemListParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/ops/stopped-items?${stringifiedParams}` : `/api/v1/ops/stopped-items`
}

/**
 * AD-45: ошибка свёртки или проекции на записи → ops.processing.failed; изделие «обработка остановлена», партиция продолжает.
 * @summary Остановленные изделия
 */
export const opsStoppedItemList = async (params?: OpsStoppedItemListParams, options?: RequestInit): Promise<opsStoppedItemListResponseSuccess> => {

  const res = await fetch(getOpsStoppedItemListUrl(params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: opsStoppedItemListResponseError['data'], status?: number} = new globalThis.Error();
    const data : opsStoppedItemListResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: opsStoppedItemListResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as opsStoppedItemListResponseSuccess
}





export const getOpsStoppedItemListQueryKey = (params?: MaybeRefOrGetter<OpsStoppedItemListParams>,) => {
    return [
    'api','v1','ops','stopped-items', ...(params ? [params] : [])
    ] as const;
    }


export const getOpsStoppedItemListQueryOptions = <TData = Awaited<ReturnType<typeof opsStoppedItemList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(params?: MaybeRefOrGetter<OpsStoppedItemListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof opsStoppedItemList>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getOpsStoppedItemListQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof opsStoppedItemList>>> = ({ signal }) => opsStoppedItemList(toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof opsStoppedItemList>>, TError, TData>
}

export type OpsStoppedItemListQueryResult = NonNullable<Awaited<ReturnType<typeof opsStoppedItemList>>>
export type OpsStoppedItemListQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Остановленные изделия
 */

export function useOpsStoppedItemList<TData = Awaited<ReturnType<typeof opsStoppedItemList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 params?: MaybeRefOrGetter<OpsStoppedItemListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof opsStoppedItemList>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getOpsStoppedItemListQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type opsProcessingRetryResponse200 = {
  data: Receipt
  status: 200
}

export type opsProcessingRetryResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type opsProcessingRetryResponseSuccess = (opsProcessingRetryResponse200) & {
  headers: Headers;
};
export type opsProcessingRetryResponseError = (opsProcessingRetryResponseDefault) & {
  headers: Headers;
};

export const getOpsProcessingRetryUrl = (itemId: string,) => {




  return `/api/v1/ops/stopped-items/${itemId}/retry`
}

/**
 * AD-45: повтор свёртки изделия (как ant rebuild --item).
 * @summary Повторить обработку изделия
 */
export const opsProcessingRetry = async (itemId: string,
    retryProcessing: RetryProcessing, options?: RequestInit): Promise<opsProcessingRetryResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getOpsProcessingRetryUrl(itemId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(retryProcessing)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: opsProcessingRetryResponseError['data'], status?: number} = new globalThis.Error();
    const data : opsProcessingRetryResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: opsProcessingRetryResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as opsProcessingRetryResponseSuccess
}





export const getOpsProcessingRetryMutationKey = () => ['opsProcessingRetry'] as const;

export const getOpsProcessingRetryMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof opsProcessingRetry>>, TError,OpsProcessingRetryMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof opsProcessingRetry>>, TError,OpsProcessingRetryMutationVariables, TContext> => {

const mutationKey = getOpsProcessingRetryMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof opsProcessingRetry>>, OpsProcessingRetryMutationVariables> = (props) => {
          const {itemId,data} = props ?? {};

          return  opsProcessingRetry(itemId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type OpsProcessingRetryMutationResult = NonNullable<Awaited<ReturnType<typeof opsProcessingRetry>>>
    export type OpsProcessingRetryMutationBody = RetryProcessing
    export type OpsProcessingRetryMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type OpsProcessingRetryMutationVariables = {itemId: string;data: RetryProcessing}

    /**
 * @summary Повторить обработку изделия
 */
export const useOpsProcessingRetry = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof opsProcessingRetry>>, TError,OpsProcessingRetryMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof opsProcessingRetry>>,
        TError,
        OpsProcessingRetryMutationVariables,
        TContext
      > => {
      return useMutation(getOpsProcessingRetryMutationOptions(options), queryClient);
    }

export type accessPermissionListResponse200 = {
  data: PermissionList
  status: 200
}

export type accessPermissionListResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type accessPermissionListResponseSuccess = (accessPermissionListResponse200) & {
  headers: Headers;
};
export type accessPermissionListResponseError = (accessPermissionListResponseDefault) & {
  headers: Headers;
};

export const getAccessPermissionListUrl = (params?: AccessPermissionListParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/permissions?${stringifiedParams}` : `/api/v1/permissions`
}

/**
 * AD-15 «Для фронтенда»: без subject — плоский список «действие → объект» для @casl/vue; с subject и id — допустимые действия по конкретному объекту. Сервер вычисляет его тем же Enforce, что проверяет команды («в списке ⇔ разрешено»). На момент as_of (воспроизведение) команд в списке нет.
 * @summary Разрешённые действия
 */
export const accessPermissionList = async (params?: AccessPermissionListParams, options?: RequestInit): Promise<accessPermissionListResponseSuccess> => {

  const res = await fetch(getAccessPermissionListUrl(params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: accessPermissionListResponseError['data'], status?: number} = new globalThis.Error();
    const data : accessPermissionListResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: accessPermissionListResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as accessPermissionListResponseSuccess
}





export const getAccessPermissionListQueryKey = (params?: MaybeRefOrGetter<AccessPermissionListParams>,) => {
    return [
    'api','v1','permissions', ...(params ? [params] : [])
    ] as const;
    }


export const getAccessPermissionListQueryOptions = <TData = Awaited<ReturnType<typeof accessPermissionList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(params?: MaybeRefOrGetter<AccessPermissionListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof accessPermissionList>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getAccessPermissionListQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof accessPermissionList>>> = ({ signal }) => accessPermissionList(toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof accessPermissionList>>, TError, TData>
}

export type AccessPermissionListQueryResult = NonNullable<Awaited<ReturnType<typeof accessPermissionList>>>
export type AccessPermissionListQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Разрешённые действия
 */

export function useAccessPermissionList<TData = Awaited<ReturnType<typeof accessPermissionList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 params?: MaybeRefOrGetter<AccessPermissionListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof accessPermissionList>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getAccessPermissionListQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type accessPermissionExplainResponse200 = {
  data: Explanation
  status: 200
}

export type accessPermissionExplainResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type accessPermissionExplainResponseSuccess = (accessPermissionExplainResponse200) & {
  headers: Headers;
};
export type accessPermissionExplainResponseError = (accessPermissionExplainResponseDefault) & {
  headers: Headers;
};

export const getAccessPermissionExplainUrl = (params: AccessPermissionExplainParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/permissions/explain?${stringifiedParams}` : `/api/v1/permissions/explain`
}

/**
 * FR-136, FR-146: решение по действию над объектом с объяснением (роль, область, полномочие, клеймо, разделение обязанностей) и тем, что можно сделать вместо («Запросить решение»).
 * @summary Почему вы можете или не можете
 */
export const accessPermissionExplain = async (params: AccessPermissionExplainParams, options?: RequestInit): Promise<accessPermissionExplainResponseSuccess> => {

  const res = await fetch(getAccessPermissionExplainUrl(params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: accessPermissionExplainResponseError['data'], status?: number} = new globalThis.Error();
    const data : accessPermissionExplainResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: accessPermissionExplainResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as accessPermissionExplainResponseSuccess
}





export const getAccessPermissionExplainQueryKey = (params?: MaybeRefOrGetter<AccessPermissionExplainParams>,) => {
    return [
    'api','v1','permissions','explain', ...(params ? [params] : [])
    ] as const;
    }


export const getAccessPermissionExplainQueryOptions = <TData = Awaited<ReturnType<typeof accessPermissionExplain>>, TError = globalThis.Error & { info?: Problem; status?: number }>(params: MaybeRefOrGetter<AccessPermissionExplainParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof accessPermissionExplain>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getAccessPermissionExplainQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof accessPermissionExplain>>> = ({ signal }) => accessPermissionExplain(toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof accessPermissionExplain>>, TError, TData>
}

export type AccessPermissionExplainQueryResult = NonNullable<Awaited<ReturnType<typeof accessPermissionExplain>>>
export type AccessPermissionExplainQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Почему вы можете или не можете
 */

export function useAccessPermissionExplain<TData = Awaited<ReturnType<typeof accessPermissionExplain>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 params: MaybeRefOrGetter<AccessPermissionExplainParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof accessPermissionExplain>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getAccessPermissionExplainQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type nonconformityProcessHoldSetResponse200 = {
  data: Receipt
  status: 200
}

export type nonconformityProcessHoldSetResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type nonconformityProcessHoldSetResponseSuccess = (nonconformityProcessHoldSetResponse200) & {
  headers: Headers;
};
export type nonconformityProcessHoldSetResponseError = (nonconformityProcessHoldSetResponseDefault) & {
  headers: Headers;
};

export const getNonconformityProcessHoldSetUrl = () => {




  return `/api/v1/process-holds`
}

/**
 * FR-49: стоп точки процесса или критическая остановка (сдерживание процесса — отдельный объект от сдерживания изделий).
 * @summary Остановить точку процесса
 */
export const nonconformityProcessHoldSet = async (setProcessHold: SetProcessHold, options?: RequestInit): Promise<nonconformityProcessHoldSetResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getNonconformityProcessHoldSetUrl(),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(setProcessHold)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: nonconformityProcessHoldSetResponseError['data'], status?: number} = new globalThis.Error();
    const data : nonconformityProcessHoldSetResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: nonconformityProcessHoldSetResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as nonconformityProcessHoldSetResponseSuccess
}





export const getNonconformityProcessHoldSetMutationKey = () => ['nonconformityProcessHoldSet'] as const;

export const getNonconformityProcessHoldSetMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof nonconformityProcessHoldSet>>, TError,NonconformityProcessHoldSetMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof nonconformityProcessHoldSet>>, TError,NonconformityProcessHoldSetMutationVariables, TContext> => {

const mutationKey = getNonconformityProcessHoldSetMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof nonconformityProcessHoldSet>>, NonconformityProcessHoldSetMutationVariables> = (props) => {
          const {data} = props ?? {};

          return  nonconformityProcessHoldSet(data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type NonconformityProcessHoldSetMutationResult = NonNullable<Awaited<ReturnType<typeof nonconformityProcessHoldSet>>>
    export type NonconformityProcessHoldSetMutationBody = SetProcessHold
    export type NonconformityProcessHoldSetMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type NonconformityProcessHoldSetMutationVariables = {data: SetProcessHold}

    /**
 * @summary Остановить точку процесса
 */
export const useNonconformityProcessHoldSet = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof nonconformityProcessHoldSet>>, TError,NonconformityProcessHoldSetMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof nonconformityProcessHoldSet>>,
        TError,
        NonconformityProcessHoldSetMutationVariables,
        TContext
      > => {
      return useMutation(getNonconformityProcessHoldSetMutationOptions(options), queryClient);
    }

export type nonconformityProcessHoldReleaseResponse200 = {
  data: Receipt
  status: 200
}

export type nonconformityProcessHoldReleaseResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type nonconformityProcessHoldReleaseResponseSuccess = (nonconformityProcessHoldReleaseResponse200) & {
  headers: Headers;
};
export type nonconformityProcessHoldReleaseResponseError = (nonconformityProcessHoldReleaseResponseDefault) & {
  headers: Headers;
};

export const getNonconformityProcessHoldReleaseUrl = (holdId: string,) => {




  return `/api/v1/process-holds/${holdId}/release`
}

/**
 * FR-49: снятие остановки — разрешающее действие уполномоченного, с числом изделий после точки чистоты.
 * @summary Снять остановку точки процесса
 */
export const nonconformityProcessHoldRelease = async (holdId: string,
    releaseProcessHold: ReleaseProcessHold, options?: RequestInit): Promise<nonconformityProcessHoldReleaseResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getNonconformityProcessHoldReleaseUrl(holdId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(releaseProcessHold)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: nonconformityProcessHoldReleaseResponseError['data'], status?: number} = new globalThis.Error();
    const data : nonconformityProcessHoldReleaseResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: nonconformityProcessHoldReleaseResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as nonconformityProcessHoldReleaseResponseSuccess
}





export const getNonconformityProcessHoldReleaseMutationKey = () => ['nonconformityProcessHoldRelease'] as const;

export const getNonconformityProcessHoldReleaseMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof nonconformityProcessHoldRelease>>, TError,NonconformityProcessHoldReleaseMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof nonconformityProcessHoldRelease>>, TError,NonconformityProcessHoldReleaseMutationVariables, TContext> => {

const mutationKey = getNonconformityProcessHoldReleaseMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof nonconformityProcessHoldRelease>>, NonconformityProcessHoldReleaseMutationVariables> = (props) => {
          const {holdId,data} = props ?? {};

          return  nonconformityProcessHoldRelease(holdId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type NonconformityProcessHoldReleaseMutationResult = NonNullable<Awaited<ReturnType<typeof nonconformityProcessHoldRelease>>>
    export type NonconformityProcessHoldReleaseMutationBody = ReleaseProcessHold
    export type NonconformityProcessHoldReleaseMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type NonconformityProcessHoldReleaseMutationVariables = {holdId: string;data: ReleaseProcessHold}

    /**
 * @summary Снять остановку точки процесса
 */
export const useNonconformityProcessHoldRelease = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof nonconformityProcessHoldRelease>>, TError,NonconformityProcessHoldReleaseMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof nonconformityProcessHoldRelease>>,
        TError,
        NonconformityProcessHoldReleaseMutationVariables,
        TContext
      > => {
      return useMutation(getNonconformityProcessHoldReleaseMutationOptions(options), queryClient);
    }

export type processVersionListResponse200 = {
  data: ProcessVersionList
  status: 200
}

export type processVersionListResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type processVersionListResponseSuccess = (processVersionListResponse200) & {
  headers: Headers;
};
export type processVersionListResponseError = (processVersionListResponseDefault) & {
  headers: Headers;
};

export const getProcessVersionListUrl = (params?: ProcessVersionListParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/process/versions?${stringifiedParams}` : `/api/v1/process/versions`
}

/**
 * FR-22: черновик → на утверждении → действующая → выведена; кворум, изделия в работе по версии.
 * @summary Версии процесса
 */
export const processVersionList = async (params?: ProcessVersionListParams, options?: RequestInit): Promise<processVersionListResponseSuccess> => {

  const res = await fetch(getProcessVersionListUrl(params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: processVersionListResponseError['data'], status?: number} = new globalThis.Error();
    const data : processVersionListResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: processVersionListResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as processVersionListResponseSuccess
}





export const getProcessVersionListQueryKey = (params?: MaybeRefOrGetter<ProcessVersionListParams>,) => {
    return [
    'api','v1','process','versions', ...(params ? [params] : [])
    ] as const;
    }


export const getProcessVersionListQueryOptions = <TData = Awaited<ReturnType<typeof processVersionList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(params?: MaybeRefOrGetter<ProcessVersionListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof processVersionList>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getProcessVersionListQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof processVersionList>>> = ({ signal }) => processVersionList(toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof processVersionList>>, TError, TData>
}

export type ProcessVersionListQueryResult = NonNullable<Awaited<ReturnType<typeof processVersionList>>>
export type ProcessVersionListQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Версии процесса
 */

export function useProcessVersionList<TData = Awaited<ReturnType<typeof processVersionList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 params?: MaybeRefOrGetter<ProcessVersionListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof processVersionList>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getProcessVersionListQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type processVersionDraftResponse200 = {
  data: Receipt
  status: 200
}

export type processVersionDraftResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type processVersionDraftResponseSuccess = (processVersionDraftResponse200) & {
  headers: Headers;
};
export type processVersionDraftResponseError = (processVersionDraftResponseDefault) & {
  headers: Headers;
};

export const getProcessVersionDraftUrl = () => {




  return `/api/v1/process/versions`
}

/**
 * FR-22, FR-25: черновик из редактора; проверка описания при загрузке (FR-13) — отказ с кодом и id элемента. В журнал черновик не пишется.
 * @summary Создать черновик версии
 */
export const processVersionDraft = async (draftVersion: DraftVersion, options?: RequestInit): Promise<processVersionDraftResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getProcessVersionDraftUrl(),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(draftVersion)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: processVersionDraftResponseError['data'], status?: number} = new globalThis.Error();
    const data : processVersionDraftResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: processVersionDraftResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as processVersionDraftResponseSuccess
}





export const getProcessVersionDraftMutationKey = () => ['processVersionDraft'] as const;

export const getProcessVersionDraftMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof processVersionDraft>>, TError,ProcessVersionDraftMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof processVersionDraft>>, TError,ProcessVersionDraftMutationVariables, TContext> => {

const mutationKey = getProcessVersionDraftMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof processVersionDraft>>, ProcessVersionDraftMutationVariables> = (props) => {
          const {data} = props ?? {};

          return  processVersionDraft(data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type ProcessVersionDraftMutationResult = NonNullable<Awaited<ReturnType<typeof processVersionDraft>>>
    export type ProcessVersionDraftMutationBody = DraftVersion
    export type ProcessVersionDraftMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type ProcessVersionDraftMutationVariables = {data: DraftVersion}

    /**
 * @summary Создать черновик версии
 */
export const useProcessVersionDraft = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof processVersionDraft>>, TError,ProcessVersionDraftMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof processVersionDraft>>,
        TError,
        ProcessVersionDraftMutationVariables,
        TContext
      > => {
      return useMutation(getProcessVersionDraftMutationOptions(options), queryClient);
    }

export type processVersionReadResponse200 = {
  data: ProcessVersion
  status: 200
}

export type processVersionReadResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type processVersionReadResponseSuccess = (processVersionReadResponse200) & {
  headers: Headers;
};
export type processVersionReadResponseError = (processVersionReadResponseDefault) & {
  headers: Headers;
};

export const getProcessVersionReadUrl = (versionId: string,
    params?: ProcessVersionReadParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/process/versions/${versionId}?${stringifiedParams}` : `/api/v1/process/versions/${versionId}`
}

/**
 * FR-24: элементы в порядке маршрута с нашими свойствами и порогами карты реакций.
 * @summary Версия процесса в читаемом виде
 */
export const processVersionRead = async (versionId: string,
    params?: ProcessVersionReadParams, options?: RequestInit): Promise<processVersionReadResponseSuccess> => {

  const res = await fetch(getProcessVersionReadUrl(versionId,params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: processVersionReadResponseError['data'], status?: number} = new globalThis.Error();
    const data : processVersionReadResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: processVersionReadResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as processVersionReadResponseSuccess
}





export const getProcessVersionReadQueryKey = (versionId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<ProcessVersionReadParams>,) => {
    return [
    'api','v1','process','versions',versionId, ...(params ? [params] : [])
    ] as const;
    }


export const getProcessVersionReadQueryOptions = <TData = Awaited<ReturnType<typeof processVersionRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(versionId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<ProcessVersionReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof processVersionRead>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getProcessVersionReadQueryKey(versionId,params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof processVersionRead>>> = ({ signal }) => processVersionRead(toValue(versionId),toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, enabled: computed(() => toValue(versionId) !== null && toValue(versionId) !== undefined), ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof processVersionRead>>, TError, TData>
}

export type ProcessVersionReadQueryResult = NonNullable<Awaited<ReturnType<typeof processVersionRead>>>
export type ProcessVersionReadQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Версия процесса в читаемом виде
 */

export function useProcessVersionRead<TData = Awaited<ReturnType<typeof processVersionRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 versionId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<ProcessVersionReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof processVersionRead>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getProcessVersionReadQueryOptions(versionId,params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type processVersionActivateResponse200 = {
  data: Receipt
  status: 200
}

export type processVersionActivateResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type processVersionActivateResponseSuccess = (processVersionActivateResponse200) & {
  headers: Headers;
};
export type processVersionActivateResponseError = (processVersionActivateResponseDefault) & {
  headers: Headers;
};

export const getProcessVersionActivateUrl = (versionId: string,) => {




  return `/api/v1/process/versions/${versionId}/activate`
}

/**
 * FR-23, AD-17: только после закрытия маршрута кворума; движок не исполняет версию без полного набора действительных подписей. Изделия в работе остаются на своей версии.
 * @summary Ввести версию в действие
 */
export const processVersionActivate = async (versionId: string,
    activateVersion: ActivateVersion, options?: RequestInit): Promise<processVersionActivateResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getProcessVersionActivateUrl(versionId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(activateVersion)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: processVersionActivateResponseError['data'], status?: number} = new globalThis.Error();
    const data : processVersionActivateResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: processVersionActivateResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as processVersionActivateResponseSuccess
}





export const getProcessVersionActivateMutationKey = () => ['processVersionActivate'] as const;

export const getProcessVersionActivateMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof processVersionActivate>>, TError,ProcessVersionActivateMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof processVersionActivate>>, TError,ProcessVersionActivateMutationVariables, TContext> => {

const mutationKey = getProcessVersionActivateMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof processVersionActivate>>, ProcessVersionActivateMutationVariables> = (props) => {
          const {versionId,data} = props ?? {};

          return  processVersionActivate(versionId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type ProcessVersionActivateMutationResult = NonNullable<Awaited<ReturnType<typeof processVersionActivate>>>
    export type ProcessVersionActivateMutationBody = ActivateVersion
    export type ProcessVersionActivateMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type ProcessVersionActivateMutationVariables = {versionId: string;data: ActivateVersion}

    /**
 * @summary Ввести версию в действие
 */
export const useProcessVersionActivate = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof processVersionActivate>>, TError,ProcessVersionActivateMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof processVersionActivate>>,
        TError,
        ProcessVersionActivateMutationVariables,
        TContext
      > => {
      return useMutation(getProcessVersionActivateMutationOptions(options), queryClient);
    }

export type processVersionBpmnResponse200 = {
  data: ProcessBpmn
  status: 200
}

export type processVersionBpmnResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type processVersionBpmnResponseSuccess = (processVersionBpmnResponse200) & {
  headers: Headers;
};
export type processVersionBpmnResponseError = (processVersionBpmnResponseDefault) & {
  headers: Headers;
};

export const getProcessVersionBpmnUrl = (versionId: string,) => {




  return `/api/v1/process/versions/${versionId}/bpmn`
}

/**
 * AD-17: подписываемая версия — XML целиком (схема, свойства, BPMNDI, documentation); хеш — H(байты как загружены).
 * @summary BPMN версии
 */
export const processVersionBpmn = async (versionId: string, options?: RequestInit): Promise<processVersionBpmnResponseSuccess> => {

  const res = await fetch(getProcessVersionBpmnUrl(versionId),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: processVersionBpmnResponseError['data'], status?: number} = new globalThis.Error();
    const data : processVersionBpmnResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: processVersionBpmnResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as processVersionBpmnResponseSuccess
}





export const getProcessVersionBpmnQueryKey = (versionId: MaybeRefOrGetter<string>,) => {
    return [
    'api','v1','process','versions',versionId,'bpmn'
    ] as const;
    }


export const getProcessVersionBpmnQueryOptions = <TData = Awaited<ReturnType<typeof processVersionBpmn>>, TError = globalThis.Error & { info?: Problem; status?: number }>(versionId: MaybeRefOrGetter<string>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof processVersionBpmn>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getProcessVersionBpmnQueryKey(versionId);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof processVersionBpmn>>> = ({ signal }) => processVersionBpmn(toValue(versionId), { signal, ...fetchOptions });





   return  { queryKey, queryFn, enabled: computed(() => toValue(versionId) !== null && toValue(versionId) !== undefined), ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof processVersionBpmn>>, TError, TData>
}

export type ProcessVersionBpmnQueryResult = NonNullable<Awaited<ReturnType<typeof processVersionBpmn>>>
export type ProcessVersionBpmnQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary BPMN версии
 */

export function useProcessVersionBpmn<TData = Awaited<ReturnType<typeof processVersionBpmn>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 versionId: MaybeRefOrGetter<string>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof processVersionBpmn>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getProcessVersionBpmnQueryOptions(versionId,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type processVersionDiffResponse200 = {
  data: ProcessVersionDiff
  status: 200
}

export type processVersionDiffResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type processVersionDiffResponseSuccess = (processVersionDiffResponse200) & {
  headers: Headers;
};
export type processVersionDiffResponseError = (processVersionDiffResponseDefault) & {
  headers: Headers;
};

export const getProcessVersionDiffUrl = (versionId: string,
    params?: ProcessVersionDiffParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/process/versions/${versionId}/diff?${stringifiedParams}` : `/api/v1/process/versions/${versionId}/diff`
}

/**
 * FR-24: читаемая разница с действующей (или указанной) версией — входит в лист утверждения кворума.
 * @summary Разница версий
 */
export const processVersionDiff = async (versionId: string,
    params?: ProcessVersionDiffParams, options?: RequestInit): Promise<processVersionDiffResponseSuccess> => {

  const res = await fetch(getProcessVersionDiffUrl(versionId,params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: processVersionDiffResponseError['data'], status?: number} = new globalThis.Error();
    const data : processVersionDiffResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: processVersionDiffResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as processVersionDiffResponseSuccess
}





export const getProcessVersionDiffQueryKey = (versionId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<ProcessVersionDiffParams>,) => {
    return [
    'api','v1','process','versions',versionId,'diff', ...(params ? [params] : [])
    ] as const;
    }


export const getProcessVersionDiffQueryOptions = <TData = Awaited<ReturnType<typeof processVersionDiff>>, TError = globalThis.Error & { info?: Problem; status?: number }>(versionId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<ProcessVersionDiffParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof processVersionDiff>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getProcessVersionDiffQueryKey(versionId,params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof processVersionDiff>>> = ({ signal }) => processVersionDiff(toValue(versionId),toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, enabled: computed(() => toValue(versionId) !== null && toValue(versionId) !== undefined), ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof processVersionDiff>>, TError, TData>
}

export type ProcessVersionDiffQueryResult = NonNullable<Awaited<ReturnType<typeof processVersionDiff>>>
export type ProcessVersionDiffQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Разница версий
 */

export function useProcessVersionDiff<TData = Awaited<ReturnType<typeof processVersionDiff>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 versionId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<ProcessVersionDiffParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof processVersionDiff>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getProcessVersionDiffQueryOptions(versionId,params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type processNodeReadResponse200 = {
  data: ProcessNodeCard
  status: 200
}

export type processNodeReadResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type processNodeReadResponseSuccess = (processNodeReadResponse200) & {
  headers: Headers;
};
export type processNodeReadResponseError = (processNodeReadResponseDefault) & {
  headers: Headers;
};

export const getProcessNodeReadUrl = (versionId: string,
    stepKey: string,
    params?: ProcessNodeReadParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/process/versions/${versionId}/nodes/${stepKey}?${stringifiedParams}` : `/api/v1/process/versions/${versionId}/nodes/${stepKey}`
}

/**
 * FR-154: описание шага из documentation версии изделия, наши свойства, нормативные опоры (FR-156) и счётчики с переходом к изделиям и несоответствиям.
 * @summary Карточка узла
 */
export const processNodeRead = async (versionId: string,
    stepKey: string,
    params?: ProcessNodeReadParams, options?: RequestInit): Promise<processNodeReadResponseSuccess> => {

  const res = await fetch(getProcessNodeReadUrl(versionId,stepKey,params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: processNodeReadResponseError['data'], status?: number} = new globalThis.Error();
    const data : processNodeReadResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: processNodeReadResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as processNodeReadResponseSuccess
}





export const getProcessNodeReadQueryKey = (versionId: MaybeRefOrGetter<string>,
    stepKey: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<ProcessNodeReadParams>,) => {
    return [
    'api','v1','process','versions',versionId,'nodes',stepKey, ...(params ? [params] : [])
    ] as const;
    }


export const getProcessNodeReadQueryOptions = <TData = Awaited<ReturnType<typeof processNodeRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(versionId: MaybeRefOrGetter<string>,
    stepKey: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<ProcessNodeReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof processNodeRead>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getProcessNodeReadQueryKey(versionId,stepKey,params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof processNodeRead>>> = ({ signal }) => processNodeRead(toValue(versionId),toValue(stepKey),toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, enabled: computed(() => toValue(versionId) !== null && toValue(versionId) !== undefined && toValue(stepKey) !== null && toValue(stepKey) !== undefined), ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof processNodeRead>>, TError, TData>
}

export type ProcessNodeReadQueryResult = NonNullable<Awaited<ReturnType<typeof processNodeRead>>>
export type ProcessNodeReadQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Карточка узла
 */

export function useProcessNodeRead<TData = Awaited<ReturnType<typeof processNodeRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 versionId: MaybeRefOrGetter<string>,
    stepKey: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<ProcessNodeReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof processNodeRead>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getProcessNodeReadQueryOptions(versionId,stepKey,params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type processVersionRetireResponse200 = {
  data: Receipt
  status: 200
}

export type processVersionRetireResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type processVersionRetireResponseSuccess = (processVersionRetireResponse200) & {
  headers: Headers;
};
export type processVersionRetireResponseError = (processVersionRetireResponseDefault) & {
  headers: Headers;
};

export const getProcessVersionRetireUrl = (versionId: string,) => {




  return `/api/v1/process/versions/${versionId}/retire`
}

/**
 * FR-22: изделия прежних версий показываются на карте с пометкой версии.
 * @summary Вывести версию
 */
export const processVersionRetire = async (versionId: string,
    retireVersion: RetireVersion, options?: RequestInit): Promise<processVersionRetireResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getProcessVersionRetireUrl(versionId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(retireVersion)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: processVersionRetireResponseError['data'], status?: number} = new globalThis.Error();
    const data : processVersionRetireResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: processVersionRetireResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as processVersionRetireResponseSuccess
}





export const getProcessVersionRetireMutationKey = () => ['processVersionRetire'] as const;

export const getProcessVersionRetireMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof processVersionRetire>>, TError,ProcessVersionRetireMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof processVersionRetire>>, TError,ProcessVersionRetireMutationVariables, TContext> => {

const mutationKey = getProcessVersionRetireMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof processVersionRetire>>, ProcessVersionRetireMutationVariables> = (props) => {
          const {versionId,data} = props ?? {};

          return  processVersionRetire(versionId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type ProcessVersionRetireMutationResult = NonNullable<Awaited<ReturnType<typeof processVersionRetire>>>
    export type ProcessVersionRetireMutationBody = RetireVersion
    export type ProcessVersionRetireMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type ProcessVersionRetireMutationVariables = {versionId: string;data: RetireVersion}

    /**
 * @summary Вывести версию
 */
export const useProcessVersionRetire = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof processVersionRetire>>, TError,ProcessVersionRetireMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof processVersionRetire>>,
        TError,
        ProcessVersionRetireMutationVariables,
        TContext
      > => {
      return useMutation(getProcessVersionRetireMutationOptions(options), queryClient);
    }

export type processVersionSubmitResponse200 = {
  data: Receipt
  status: 200
}

export type processVersionSubmitResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type processVersionSubmitResponseSuccess = (processVersionSubmitResponse200) & {
  headers: Headers;
};
export type processVersionSubmitResponseError = (processVersionSubmitResponseDefault) & {
  headers: Headers;
};

export const getProcessVersionSubmitUrl = (versionId: string,) => {




  return `/api/v1/process/versions/${versionId}/submit`
}

/**
 * FR-23: лист утверждения версии (документ с маршрутом кворума: технолог-автор → контролёр с полномочием «кворум», руководитель производства).
 * @summary Отправить на утверждение
 */
export const processVersionSubmit = async (versionId: string,
    submitVersion: SubmitVersion, options?: RequestInit): Promise<processVersionSubmitResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getProcessVersionSubmitUrl(versionId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(submitVersion)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: processVersionSubmitResponseError['data'], status?: number} = new globalThis.Error();
    const data : processVersionSubmitResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: processVersionSubmitResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as processVersionSubmitResponseSuccess
}





export const getProcessVersionSubmitMutationKey = () => ['processVersionSubmit'] as const;

export const getProcessVersionSubmitMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof processVersionSubmit>>, TError,ProcessVersionSubmitMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof processVersionSubmit>>, TError,ProcessVersionSubmitMutationVariables, TContext> => {

const mutationKey = getProcessVersionSubmitMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof processVersionSubmit>>, ProcessVersionSubmitMutationVariables> = (props) => {
          const {versionId,data} = props ?? {};

          return  processVersionSubmit(versionId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type ProcessVersionSubmitMutationResult = NonNullable<Awaited<ReturnType<typeof processVersionSubmit>>>
    export type ProcessVersionSubmitMutationBody = SubmitVersion
    export type ProcessVersionSubmitMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type ProcessVersionSubmitMutationVariables = {versionId: string;data: SubmitVersion}

    /**
 * @summary Отправить на утверждение
 */
export const useProcessVersionSubmit = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof processVersionSubmit>>, TError,ProcessVersionSubmitMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof processVersionSubmit>>,
        TError,
        ProcessVersionSubmitMutationVariables,
        TContext
      > => {
      return useMutation(getProcessVersionSubmitMutationOptions(options), queryClient);
    }

export type ingestQuarantineListResponse200 = {
  data: QuarantineList
  status: 200
}

export type ingestQuarantineListResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type ingestQuarantineListResponseSuccess = (ingestQuarantineListResponse200) & {
  headers: Headers;
};
export type ingestQuarantineListResponseError = (ingestQuarantineListResponseDefault) & {
  headers: Headers;
};

export const getIngestQuarantineListUrl = (params?: IngestQuarantineListParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/quarantine?${stringifiedParams}` : `/api/v1/quarantine`
}

/**
 * FR-30: содержимое хранится вне журнала, факт помещения — служебная запись ingest.message.quarantined с кодом причины.
 * @summary Карантин сообщений
 */
export const ingestQuarantineList = async (params?: IngestQuarantineListParams, options?: RequestInit): Promise<ingestQuarantineListResponseSuccess> => {

  const res = await fetch(getIngestQuarantineListUrl(params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: ingestQuarantineListResponseError['data'], status?: number} = new globalThis.Error();
    const data : ingestQuarantineListResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: ingestQuarantineListResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as ingestQuarantineListResponseSuccess
}





export const getIngestQuarantineListQueryKey = (params?: MaybeRefOrGetter<IngestQuarantineListParams>,) => {
    return [
    'api','v1','quarantine', ...(params ? [params] : [])
    ] as const;
    }


export const getIngestQuarantineListQueryOptions = <TData = Awaited<ReturnType<typeof ingestQuarantineList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(params?: MaybeRefOrGetter<IngestQuarantineListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof ingestQuarantineList>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getIngestQuarantineListQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof ingestQuarantineList>>> = ({ signal }) => ingestQuarantineList(toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof ingestQuarantineList>>, TError, TData>
}

export type IngestQuarantineListQueryResult = NonNullable<Awaited<ReturnType<typeof ingestQuarantineList>>>
export type IngestQuarantineListQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Карантин сообщений
 */

export function useIngestQuarantineList<TData = Awaited<ReturnType<typeof ingestQuarantineList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 params?: MaybeRefOrGetter<IngestQuarantineListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof ingestQuarantineList>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getIngestQuarantineListQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type ingestQuarantineReadResponse200 = {
  data: QuarantineEntry
  status: 200
}

export type ingestQuarantineReadResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type ingestQuarantineReadResponseSuccess = (ingestQuarantineReadResponse200) & {
  headers: Headers;
};
export type ingestQuarantineReadResponseError = (ingestQuarantineReadResponseDefault) & {
  headers: Headers;
};

export const getIngestQuarantineReadUrl = (quarantineId: string,) => {




  return `/api/v1/quarantine/${quarantineId}`
}

/**
 * Исходное содержимое, код причины и история переобработки.
 * @summary Сообщение в карантине
 */
export const ingestQuarantineRead = async (quarantineId: string, options?: RequestInit): Promise<ingestQuarantineReadResponseSuccess> => {

  const res = await fetch(getIngestQuarantineReadUrl(quarantineId),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: ingestQuarantineReadResponseError['data'], status?: number} = new globalThis.Error();
    const data : ingestQuarantineReadResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: ingestQuarantineReadResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as ingestQuarantineReadResponseSuccess
}





export const getIngestQuarantineReadQueryKey = (quarantineId: MaybeRefOrGetter<string>,) => {
    return [
    'api','v1','quarantine',quarantineId
    ] as const;
    }


export const getIngestQuarantineReadQueryOptions = <TData = Awaited<ReturnType<typeof ingestQuarantineRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(quarantineId: MaybeRefOrGetter<string>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof ingestQuarantineRead>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getIngestQuarantineReadQueryKey(quarantineId);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof ingestQuarantineRead>>> = ({ signal }) => ingestQuarantineRead(toValue(quarantineId), { signal, ...fetchOptions });





   return  { queryKey, queryFn, enabled: computed(() => toValue(quarantineId) !== null && toValue(quarantineId) !== undefined), ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof ingestQuarantineRead>>, TError, TData>
}

export type IngestQuarantineReadQueryResult = NonNullable<Awaited<ReturnType<typeof ingestQuarantineRead>>>
export type IngestQuarantineReadQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Сообщение в карантине
 */

export function useIngestQuarantineRead<TData = Awaited<ReturnType<typeof ingestQuarantineRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 quarantineId: MaybeRefOrGetter<string>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof ingestQuarantineRead>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getIngestQuarantineReadQueryOptions(quarantineId,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type ingestMessageReprocessResponse200 = {
  data: Receipt
  status: 200
}

export type ingestMessageReprocessResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type ingestMessageReprocessResponseSuccess = (ingestMessageReprocessResponse200) & {
  headers: Headers;
};
export type ingestMessageReprocessResponseError = (ingestMessageReprocessResponseDefault) & {
  headers: Headers;
};

export const getIngestMessageReprocessUrl = (quarantineId: string,) => {




  return `/api/v1/quarantine/${quarantineId}/reprocess`
}

/**
 * FR-29, AD-20: после появления повышателя или исправления — принять, оставить или отбросить; итог — ingest.message.reprocessed.
 * @summary Переобработать из карантина
 */
export const ingestMessageReprocess = async (quarantineId: string,
    reprocessMessage: ReprocessMessage, options?: RequestInit): Promise<ingestMessageReprocessResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getIngestMessageReprocessUrl(quarantineId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(reprocessMessage)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: ingestMessageReprocessResponseError['data'], status?: number} = new globalThis.Error();
    const data : ingestMessageReprocessResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: ingestMessageReprocessResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as ingestMessageReprocessResponseSuccess
}





export const getIngestMessageReprocessMutationKey = () => ['ingestMessageReprocess'] as const;

export const getIngestMessageReprocessMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof ingestMessageReprocess>>, TError,IngestMessageReprocessMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof ingestMessageReprocess>>, TError,IngestMessageReprocessMutationVariables, TContext> => {

const mutationKey = getIngestMessageReprocessMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof ingestMessageReprocess>>, IngestMessageReprocessMutationVariables> = (props) => {
          const {quarantineId,data} = props ?? {};

          return  ingestMessageReprocess(quarantineId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type IngestMessageReprocessMutationResult = NonNullable<Awaited<ReturnType<typeof ingestMessageReprocess>>>
    export type IngestMessageReprocessMutationBody = ReprocessMessage
    export type IngestMessageReprocessMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type IngestMessageReprocessMutationVariables = {quarantineId: string;data: ReprocessMessage}

    /**
 * @summary Переобработать из карантина
 */
export const useIngestMessageReprocess = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof ingestMessageReprocess>>, TError,IngestMessageReprocessMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof ingestMessageReprocess>>,
        TError,
        IngestMessageReprocessMutationVariables,
        TContext
      > => {
      return useMutation(getIngestMessageReprocessMutationOptions(options), queryClient);
    }

export type qualityReactionMapReadResponse200 = {
  data: ReactionMap
  status: 200
}

export type qualityReactionMapReadResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type qualityReactionMapReadResponseSuccess = (qualityReactionMapReadResponse200) & {
  headers: Headers;
};
export type qualityReactionMapReadResponseError = (qualityReactionMapReadResponseDefault) & {
  headers: Headers;
};

export const getQualityReactionMapReadUrl = (params?: QualityReactionMapReadParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/reaction-map?${stringifiedParams}` : `/api/v1/reaction-map`
}

/**
 * FR-48, FR-50: действующая карта реакций — правила с режимом автоматизации 1–5, классом действия, владельцем, сроком и порогами.
 * @summary Карта реакций
 */
export const qualityReactionMapRead = async (params?: QualityReactionMapReadParams, options?: RequestInit): Promise<qualityReactionMapReadResponseSuccess> => {

  const res = await fetch(getQualityReactionMapReadUrl(params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: qualityReactionMapReadResponseError['data'], status?: number} = new globalThis.Error();
    const data : qualityReactionMapReadResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: qualityReactionMapReadResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as qualityReactionMapReadResponseSuccess
}





export const getQualityReactionMapReadQueryKey = (params?: MaybeRefOrGetter<QualityReactionMapReadParams>,) => {
    return [
    'api','v1','reaction-map', ...(params ? [params] : [])
    ] as const;
    }


export const getQualityReactionMapReadQueryOptions = <TData = Awaited<ReturnType<typeof qualityReactionMapRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(params?: MaybeRefOrGetter<QualityReactionMapReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof qualityReactionMapRead>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getQualityReactionMapReadQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof qualityReactionMapRead>>> = ({ signal }) => qualityReactionMapRead(toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof qualityReactionMapRead>>, TError, TData>
}

export type QualityReactionMapReadQueryResult = NonNullable<Awaited<ReturnType<typeof qualityReactionMapRead>>>
export type QualityReactionMapReadQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Карта реакций
 */

export function useQualityReactionMapRead<TData = Awaited<ReturnType<typeof qualityReactionMapRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 params?: MaybeRefOrGetter<QualityReactionMapReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof qualityReactionMapRead>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getQualityReactionMapReadQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type simulationRunListResponse200 = {
  data: RunList
  status: 200
}

export type simulationRunListResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type simulationRunListResponseSuccess = (simulationRunListResponse200) & {
  headers: Headers;
};
export type simulationRunListResponseError = (simulationRunListResponseDefault) & {
  headers: Headers;
};

export const getSimulationRunListUrl = (params?: SimulationRunListParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/runs?${stringifiedParams}` : `/api/v1/runs`
}

/**
 * AD-38: прогоны — отдельные пространства имён run_id.
 * @summary Прогоны сценариев
 */
export const simulationRunList = async (params?: SimulationRunListParams, options?: RequestInit): Promise<simulationRunListResponseSuccess> => {

  const res = await fetch(getSimulationRunListUrl(params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: simulationRunListResponseError['data'], status?: number} = new globalThis.Error();
    const data : simulationRunListResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: simulationRunListResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as simulationRunListResponseSuccess
}





export const getSimulationRunListQueryKey = (params?: MaybeRefOrGetter<SimulationRunListParams>,) => {
    return [
    'api','v1','runs', ...(params ? [params] : [])
    ] as const;
    }


export const getSimulationRunListQueryOptions = <TData = Awaited<ReturnType<typeof simulationRunList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(params?: MaybeRefOrGetter<SimulationRunListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof simulationRunList>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getSimulationRunListQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof simulationRunList>>> = ({ signal }) => simulationRunList(toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof simulationRunList>>, TError, TData>
}

export type SimulationRunListQueryResult = NonNullable<Awaited<ReturnType<typeof simulationRunList>>>
export type SimulationRunListQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Прогоны сценариев
 */

export function useSimulationRunList<TData = Awaited<ReturnType<typeof simulationRunList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 params?: MaybeRefOrGetter<SimulationRunListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof simulationRunList>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getSimulationRunListQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type simulationRunReadResponse200 = {
  data: Run
  status: 200
}

export type simulationRunReadResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type simulationRunReadResponseSuccess = (simulationRunReadResponse200) & {
  headers: Headers;
};
export type simulationRunReadResponseError = (simulationRunReadResponseDefault) & {
  headers: Headers;
};

export const getSimulationRunReadUrl = (runId: string,
    params?: SimulationRunReadParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/runs/${runId}?${stringifiedParams}` : `/api/v1/runs/${runId}`
}

/**
 * Шаг сценария, доменные часы (AD-37), пауза, скорость, решение человека, которого ждёт сценарий, счёт табло.
 * @summary Состояние прогона
 */
export const simulationRunRead = async (runId: string,
    params?: SimulationRunReadParams, options?: RequestInit): Promise<simulationRunReadResponseSuccess> => {

  const res = await fetch(getSimulationRunReadUrl(runId,params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: simulationRunReadResponseError['data'], status?: number} = new globalThis.Error();
    const data : simulationRunReadResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: simulationRunReadResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as simulationRunReadResponseSuccess
}





export const getSimulationRunReadQueryKey = (runId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<SimulationRunReadParams>,) => {
    return [
    'api','v1','runs',runId, ...(params ? [params] : [])
    ] as const;
    }


export const getSimulationRunReadQueryOptions = <TData = Awaited<ReturnType<typeof simulationRunRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(runId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<SimulationRunReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof simulationRunRead>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getSimulationRunReadQueryKey(runId,params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof simulationRunRead>>> = ({ signal }) => simulationRunRead(toValue(runId),toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, enabled: computed(() => toValue(runId) !== null && toValue(runId) !== undefined), ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof simulationRunRead>>, TError, TData>
}

export type SimulationRunReadQueryResult = NonNullable<Awaited<ReturnType<typeof simulationRunRead>>>
export type SimulationRunReadQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Состояние прогона
 */

export function useSimulationRunRead<TData = Awaited<ReturnType<typeof simulationRunRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 runId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<SimulationRunReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof simulationRunRead>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getSimulationRunReadQueryOptions(runId,params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type simulationBoardReadResponse200 = {
  data: Board
  status: 200
}

export type simulationBoardReadResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type simulationBoardReadResponseSuccess = (simulationBoardReadResponse200) & {
  headers: Headers;
};
export type simulationBoardReadResponseError = (simulationBoardReadResponseDefault) & {
  headers: Headers;
};

export const getSimulationBoardReadUrl = (runId: string,
    params?: SimulationBoardReadParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/runs/${runId}/board?${stringifiedParams}` : `/api/v1/runs/${runId}/board`
}

/**
 * AD-26, кейс §5.1: утверждения scenarios/expected над теми же operationId проверяются теми же Queries; ожидаемое хранится отдельно от входных событий.
 * @summary Табло «ожидалось → получилось»
 */
export const simulationBoardRead = async (runId: string,
    params?: SimulationBoardReadParams, options?: RequestInit): Promise<simulationBoardReadResponseSuccess> => {

  const res = await fetch(getSimulationBoardReadUrl(runId,params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: simulationBoardReadResponseError['data'], status?: number} = new globalThis.Error();
    const data : simulationBoardReadResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: simulationBoardReadResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as simulationBoardReadResponseSuccess
}





export const getSimulationBoardReadQueryKey = (runId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<SimulationBoardReadParams>,) => {
    return [
    'api','v1','runs',runId,'board', ...(params ? [params] : [])
    ] as const;
    }


export const getSimulationBoardReadQueryOptions = <TData = Awaited<ReturnType<typeof simulationBoardRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(runId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<SimulationBoardReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof simulationBoardRead>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getSimulationBoardReadQueryKey(runId,params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof simulationBoardRead>>> = ({ signal }) => simulationBoardRead(toValue(runId),toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, enabled: computed(() => toValue(runId) !== null && toValue(runId) !== undefined), ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof simulationBoardRead>>, TError, TData>
}

export type SimulationBoardReadQueryResult = NonNullable<Awaited<ReturnType<typeof simulationBoardRead>>>
export type SimulationBoardReadQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Табло «ожидалось → получилось»
 */

export function useSimulationBoardRead<TData = Awaited<ReturnType<typeof simulationBoardRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 runId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<SimulationBoardReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof simulationBoardRead>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getSimulationBoardReadQueryOptions(runId,params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type simulationInjectionListResponse200 = {
  data: InjectionList
  status: 200
}

export type simulationInjectionListResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type simulationInjectionListResponseSuccess = (simulationInjectionListResponse200) & {
  headers: Headers;
};
export type simulationInjectionListResponseError = (simulationInjectionListResponseDefault) & {
  headers: Headers;
};

export const getSimulationInjectionListUrl = (runId: string,) => {




  return `/api/v1/runs/${runId}/injections`
}

/**
 * FR-152: повтор события, опоздавшее событие, испорченный кадр, сбой станка, потеря куска данных, подделка в обход системы (только профили fixtures и demo).
 * @summary Кнопки цифрового стенда
 */
export const simulationInjectionList = async (runId: string, options?: RequestInit): Promise<simulationInjectionListResponseSuccess> => {

  const res = await fetch(getSimulationInjectionListUrl(runId),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: simulationInjectionListResponseError['data'], status?: number} = new globalThis.Error();
    const data : simulationInjectionListResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: simulationInjectionListResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as simulationInjectionListResponseSuccess
}





export const getSimulationInjectionListQueryKey = (runId: MaybeRefOrGetter<string>,) => {
    return [
    'api','v1','runs',runId,'injections'
    ] as const;
    }


export const getSimulationInjectionListQueryOptions = <TData = Awaited<ReturnType<typeof simulationInjectionList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(runId: MaybeRefOrGetter<string>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof simulationInjectionList>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getSimulationInjectionListQueryKey(runId);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof simulationInjectionList>>> = ({ signal }) => simulationInjectionList(toValue(runId), { signal, ...fetchOptions });





   return  { queryKey, queryFn, enabled: computed(() => toValue(runId) !== null && toValue(runId) !== undefined), ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof simulationInjectionList>>, TError, TData>
}

export type SimulationInjectionListQueryResult = NonNullable<Awaited<ReturnType<typeof simulationInjectionList>>>
export type SimulationInjectionListQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Кнопки цифрового стенда
 */

export function useSimulationInjectionList<TData = Awaited<ReturnType<typeof simulationInjectionList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 runId: MaybeRefOrGetter<string>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof simulationInjectionList>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getSimulationInjectionListQueryOptions(runId,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type simulationInjectionApplyResponse200 = {
  data: Receipt
  status: 200
}

export type simulationInjectionApplyResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type simulationInjectionApplyResponseSuccess = (simulationInjectionApplyResponse200) & {
  headers: Headers;
};
export type simulationInjectionApplyResponseError = (simulationInjectionApplyResponseDefault) & {
  headers: Headers;
};

export const getSimulationInjectionApplyUrl = (runId: string,) => {




  return `/api/v1/runs/${runId}/injections`
}

/**
 * FR-152, AD-26: инъекция через служебный порт stand-ов и обычный приём; «подделка в обход системы» — только демо-инструментом cmd/tamper. Результат виден на столах и табло.
 * @summary Нажать кнопку цифрового стенда
 */
export const simulationInjectionApply = async (runId: string,
    applyInjection: ApplyInjection, options?: RequestInit): Promise<simulationInjectionApplyResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getSimulationInjectionApplyUrl(runId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(applyInjection)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: simulationInjectionApplyResponseError['data'], status?: number} = new globalThis.Error();
    const data : simulationInjectionApplyResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: simulationInjectionApplyResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as simulationInjectionApplyResponseSuccess
}





export const getSimulationInjectionApplyMutationKey = () => ['simulationInjectionApply'] as const;

export const getSimulationInjectionApplyMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof simulationInjectionApply>>, TError,SimulationInjectionApplyMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof simulationInjectionApply>>, TError,SimulationInjectionApplyMutationVariables, TContext> => {

const mutationKey = getSimulationInjectionApplyMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof simulationInjectionApply>>, SimulationInjectionApplyMutationVariables> = (props) => {
          const {runId,data} = props ?? {};

          return  simulationInjectionApply(runId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type SimulationInjectionApplyMutationResult = NonNullable<Awaited<ReturnType<typeof simulationInjectionApply>>>
    export type SimulationInjectionApplyMutationBody = ApplyInjection
    export type SimulationInjectionApplyMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type SimulationInjectionApplyMutationVariables = {runId: string;data: ApplyInjection}

    /**
 * @summary Нажать кнопку цифрового стенда
 */
export const useSimulationInjectionApply = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof simulationInjectionApply>>, TError,SimulationInjectionApplyMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof simulationInjectionApply>>,
        TError,
        SimulationInjectionApplyMutationVariables,
        TContext
      > => {
      return useMutation(getSimulationInjectionApplyMutationOptions(options), queryClient);
    }

export type simulationRunPauseResponse200 = {
  data: Receipt
  status: 200
}

export type simulationRunPauseResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type simulationRunPauseResponseSuccess = (simulationRunPauseResponse200) & {
  headers: Headers;
};
export type simulationRunPauseResponseError = (simulationRunPauseResponseDefault) & {
  headers: Headers;
};

export const getSimulationRunPauseUrl = (runId: string,) => {




  return `/api/v1/runs/${runId}/pause`
}

/**
 * AD-26: пауза останавливает поток событий и доменные часы; столы показывают состояние на этот момент.
 * @summary Пауза
 */
export const simulationRunPause = async (runId: string,
    runControl: RunControl, options?: RequestInit): Promise<simulationRunPauseResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getSimulationRunPauseUrl(runId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(runControl)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: simulationRunPauseResponseError['data'], status?: number} = new globalThis.Error();
    const data : simulationRunPauseResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: simulationRunPauseResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as simulationRunPauseResponseSuccess
}





export const getSimulationRunPauseMutationKey = () => ['simulationRunPause'] as const;

export const getSimulationRunPauseMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof simulationRunPause>>, TError,SimulationRunPauseMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof simulationRunPause>>, TError,SimulationRunPauseMutationVariables, TContext> => {

const mutationKey = getSimulationRunPauseMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof simulationRunPause>>, SimulationRunPauseMutationVariables> = (props) => {
          const {runId,data} = props ?? {};

          return  simulationRunPause(runId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type SimulationRunPauseMutationResult = NonNullable<Awaited<ReturnType<typeof simulationRunPause>>>
    export type SimulationRunPauseMutationBody = RunControl
    export type SimulationRunPauseMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type SimulationRunPauseMutationVariables = {runId: string;data: RunControl}

    /**
 * @summary Пауза
 */
export const useSimulationRunPause = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof simulationRunPause>>, TError,SimulationRunPauseMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof simulationRunPause>>,
        TError,
        SimulationRunPauseMutationVariables,
        TContext
      > => {
      return useMutation(getSimulationRunPauseMutationOptions(options), queryClient);
    }

export type simulationRunResumeResponse200 = {
  data: Receipt
  status: 200
}

export type simulationRunResumeResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type simulationRunResumeResponseSuccess = (simulationRunResumeResponse200) & {
  headers: Headers;
};
export type simulationRunResumeResponseError = (simulationRunResumeResponseDefault) & {
  headers: Headers;
};

export const getSimulationRunResumeUrl = (runId: string,) => {




  return `/api/v1/runs/${runId}/resume`
}

/**
 * Продолжение без потерь и дублей.
 * @summary Продолжить
 */
export const simulationRunResume = async (runId: string,
    runControl: RunControl, options?: RequestInit): Promise<simulationRunResumeResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getSimulationRunResumeUrl(runId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(runControl)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: simulationRunResumeResponseError['data'], status?: number} = new globalThis.Error();
    const data : simulationRunResumeResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: simulationRunResumeResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as simulationRunResumeResponseSuccess
}





export const getSimulationRunResumeMutationKey = () => ['simulationRunResume'] as const;

export const getSimulationRunResumeMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof simulationRunResume>>, TError,SimulationRunResumeMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof simulationRunResume>>, TError,SimulationRunResumeMutationVariables, TContext> => {

const mutationKey = getSimulationRunResumeMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof simulationRunResume>>, SimulationRunResumeMutationVariables> = (props) => {
          const {runId,data} = props ?? {};

          return  simulationRunResume(runId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type SimulationRunResumeMutationResult = NonNullable<Awaited<ReturnType<typeof simulationRunResume>>>
    export type SimulationRunResumeMutationBody = RunControl
    export type SimulationRunResumeMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type SimulationRunResumeMutationVariables = {runId: string;data: RunControl}

    /**
 * @summary Продолжить
 */
export const useSimulationRunResume = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof simulationRunResume>>, TError,SimulationRunResumeMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof simulationRunResume>>,
        TError,
        SimulationRunResumeMutationVariables,
        TContext
      > => {
      return useMutation(getSimulationRunResumeMutationOptions(options), queryClient);
    }

export type simulationRunSetSpeedResponse200 = {
  data: Receipt
  status: 200
}

export type simulationRunSetSpeedResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type simulationRunSetSpeedResponseSuccess = (simulationRunSetSpeedResponse200) & {
  headers: Headers;
};
export type simulationRunSetSpeedResponseError = (simulationRunSetSpeedResponseDefault) & {
  headers: Headers;
};

export const getSimulationRunSetSpeedUrl = (runId: string,) => {




  return `/api/v1/runs/${runId}/speed`
}

/**
 * Ускорение доменных часов ×1…×1000 (AD-37); тики — time.clock.ticked.
 * @summary Изменить скорость
 */
export const simulationRunSetSpeed = async (runId: string,
    setSpeed: SetSpeed, options?: RequestInit): Promise<simulationRunSetSpeedResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getSimulationRunSetSpeedUrl(runId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(setSpeed)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: simulationRunSetSpeedResponseError['data'], status?: number} = new globalThis.Error();
    const data : simulationRunSetSpeedResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: simulationRunSetSpeedResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as simulationRunSetSpeedResponseSuccess
}





export const getSimulationRunSetSpeedMutationKey = () => ['simulationRunSetSpeed'] as const;

export const getSimulationRunSetSpeedMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof simulationRunSetSpeed>>, TError,SimulationRunSetSpeedMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof simulationRunSetSpeed>>, TError,SimulationRunSetSpeedMutationVariables, TContext> => {

const mutationKey = getSimulationRunSetSpeedMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof simulationRunSetSpeed>>, SimulationRunSetSpeedMutationVariables> = (props) => {
          const {runId,data} = props ?? {};

          return  simulationRunSetSpeed(runId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type SimulationRunSetSpeedMutationResult = NonNullable<Awaited<ReturnType<typeof simulationRunSetSpeed>>>
    export type SimulationRunSetSpeedMutationBody = SetSpeed
    export type SimulationRunSetSpeedMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type SimulationRunSetSpeedMutationVariables = {runId: string;data: SetSpeed}

    /**
 * @summary Изменить скорость
 */
export const useSimulationRunSetSpeed = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof simulationRunSetSpeed>>, TError,SimulationRunSetSpeedMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof simulationRunSetSpeed>>,
        TError,
        SimulationRunSetSpeedMutationVariables,
        TContext
      > => {
      return useMutation(getSimulationRunSetSpeedMutationOptions(options), queryClient);
    }

export type simulationRunStopResponse200 = {
  data: Receipt
  status: 200
}

export type simulationRunStopResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type simulationRunStopResponseSuccess = (simulationRunStopResponse200) & {
  headers: Headers;
};
export type simulationRunStopResponseError = (simulationRunStopResponseDefault) & {
  headers: Headers;
};

export const getSimulationRunStopUrl = (runId: string,) => {




  return `/api/v1/runs/${runId}/stop`
}

/**
 * Прогон завершается с итогом stopped.
 * @summary Остановить прогон
 */
export const simulationRunStop = async (runId: string,
    runControl: RunControl, options?: RequestInit): Promise<simulationRunStopResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getSimulationRunStopUrl(runId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(runControl)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: simulationRunStopResponseError['data'], status?: number} = new globalThis.Error();
    const data : simulationRunStopResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: simulationRunStopResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as simulationRunStopResponseSuccess
}





export const getSimulationRunStopMutationKey = () => ['simulationRunStop'] as const;

export const getSimulationRunStopMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof simulationRunStop>>, TError,SimulationRunStopMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof simulationRunStop>>, TError,SimulationRunStopMutationVariables, TContext> => {

const mutationKey = getSimulationRunStopMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof simulationRunStop>>, SimulationRunStopMutationVariables> = (props) => {
          const {runId,data} = props ?? {};

          return  simulationRunStop(runId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type SimulationRunStopMutationResult = NonNullable<Awaited<ReturnType<typeof simulationRunStop>>>
    export type SimulationRunStopMutationBody = RunControl
    export type SimulationRunStopMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type SimulationRunStopMutationVariables = {runId: string;data: RunControl}

    /**
 * @summary Остановить прогон
 */
export const useSimulationRunStop = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof simulationRunStop>>, TError,SimulationRunStopMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof simulationRunStop>>,
        TError,
        SimulationRunStopMutationVariables,
        TContext
      > => {
      return useMutation(getSimulationRunStopMutationOptions(options), queryClient);
    }

export type simulationScenarioListResponse200 = {
  data: ScenarioList
  status: 200
}

export type simulationScenarioListResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type simulationScenarioListResponseSuccess = (simulationScenarioListResponse200) & {
  headers: Headers;
};
export type simulationScenarioListResponseError = (simulationScenarioListResponseDefault) & {
  headers: Headers;
};

export const getSimulationScenarioListUrl = () => {




  return `/api/v1/scenarios`
}

/**
 * AD-26: определения scenarios/definitions — 8 ситуаций §4.2, 9 проверок §5.1, демо-сценарии, сбои; число утверждений табло и остановок на решениях.
 * @summary Сценарии
 */
export const simulationScenarioList = async ( options?: RequestInit): Promise<simulationScenarioListResponseSuccess> => {

  const res = await fetch(getSimulationScenarioListUrl(),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: simulationScenarioListResponseError['data'], status?: number} = new globalThis.Error();
    const data : simulationScenarioListResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: simulationScenarioListResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as simulationScenarioListResponseSuccess
}





export const getSimulationScenarioListQueryKey = () => {
    return [
    'api','v1','scenarios'
    ] as const;
    }


export const getSimulationScenarioListQueryOptions = <TData = Awaited<ReturnType<typeof simulationScenarioList>>, TError = globalThis.Error & { info?: Problem; status?: number }>( options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof simulationScenarioList>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getSimulationScenarioListQueryKey();



    const queryFn: QueryFunction<Awaited<ReturnType<typeof simulationScenarioList>>> = ({ signal }) => simulationScenarioList({ signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof simulationScenarioList>>, TError, TData>
}

export type SimulationScenarioListQueryResult = NonNullable<Awaited<ReturnType<typeof simulationScenarioList>>>
export type SimulationScenarioListQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Сценарии
 */

export function useSimulationScenarioList<TData = Awaited<ReturnType<typeof simulationScenarioList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof simulationScenarioList>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getSimulationScenarioListQueryOptions(options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type simulationRunStartResponse200 = {
  data: Receipt
  status: 200
}

export type simulationRunStartResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type simulationRunStartResponseSuccess = (simulationRunStartResponse200) & {
  headers: Headers;
};
export type simulationRunStartResponseError = (simulationRunStartResponseDefault) & {
  headers: Headers;
};

export const getSimulationRunStartUrl = (scenarioId: string,) => {




  return `/api/v1/scenarios/${scenarioId}/runs`
}

/**
 * AD-38: новый прогон с run_id, seed, скоростью и режимом; события идут через stand-ы → edge-агент → обычный приём (AD-26).
 * @summary Запустить сценарий
 */
export const simulationRunStart = async (scenarioId: string,
    startRun: StartRun, options?: RequestInit): Promise<simulationRunStartResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getSimulationRunStartUrl(scenarioId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(startRun)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: simulationRunStartResponseError['data'], status?: number} = new globalThis.Error();
    const data : simulationRunStartResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: simulationRunStartResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as simulationRunStartResponseSuccess
}





export const getSimulationRunStartMutationKey = () => ['simulationRunStart'] as const;

export const getSimulationRunStartMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof simulationRunStart>>, TError,SimulationRunStartMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof simulationRunStart>>, TError,SimulationRunStartMutationVariables, TContext> => {

const mutationKey = getSimulationRunStartMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof simulationRunStart>>, SimulationRunStartMutationVariables> = (props) => {
          const {scenarioId,data} = props ?? {};

          return  simulationRunStart(scenarioId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type SimulationRunStartMutationResult = NonNullable<Awaited<ReturnType<typeof simulationRunStart>>>
    export type SimulationRunStartMutationBody = StartRun
    export type SimulationRunStartMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type SimulationRunStartMutationVariables = {scenarioId: string;data: StartRun}

    /**
 * @summary Запустить сценарий
 */
export const useSimulationRunStart = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof simulationRunStart>>, TError,SimulationRunStartMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof simulationRunStart>>,
        TError,
        SimulationRunStartMutationVariables,
        TContext
      > => {
      return useMutation(getSimulationRunStartMutationOptions(options), queryClient);
    }

export type qualitySignalListResponse200 = {
  data: QualitySignalList
  status: 200
}

export type qualitySignalListResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type qualitySignalListResponseSuccess = (qualitySignalListResponse200) & {
  headers: Headers;
};
export type qualitySignalListResponseError = (qualitySignalListResponseDefault) & {
  headers: Headers;
};

export const getQualitySignalListUrl = (params?: QualitySignalListParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/signals?${stringifiedParams}` : `/api/v1/signals`
}

/**
 * Сигналы quality.signal.raised по изделию и состоянию рассмотрения. Сигнал ≠ брак (NFR-UI-4).
 * @summary Сигналы о признаке дефекта
 */
export const qualitySignalList = async (params?: QualitySignalListParams, options?: RequestInit): Promise<qualitySignalListResponseSuccess> => {

  const res = await fetch(getQualitySignalListUrl(params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: qualitySignalListResponseError['data'], status?: number} = new globalThis.Error();
    const data : qualitySignalListResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: qualitySignalListResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as qualitySignalListResponseSuccess
}





export const getQualitySignalListQueryKey = (params?: MaybeRefOrGetter<QualitySignalListParams>,) => {
    return [
    'api','v1','signals', ...(params ? [params] : [])
    ] as const;
    }


export const getQualitySignalListQueryOptions = <TData = Awaited<ReturnType<typeof qualitySignalList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(params?: MaybeRefOrGetter<QualitySignalListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof qualitySignalList>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getQualitySignalListQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof qualitySignalList>>> = ({ signal }) => qualitySignalList(toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof qualitySignalList>>, TError, TData>
}

export type QualitySignalListQueryResult = NonNullable<Awaited<ReturnType<typeof qualitySignalList>>>
export type QualitySignalListQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Сигналы о признаке дефекта
 */

export function useQualitySignalList<TData = Awaited<ReturnType<typeof qualitySignalList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 params?: MaybeRefOrGetter<QualitySignalListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof qualitySignalList>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getQualitySignalListQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type qualitySignalReadResponse200 = {
  data: QualitySignal
  status: 200
}

export type qualitySignalReadResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type qualitySignalReadResponseSuccess = (qualitySignalReadResponse200) & {
  headers: Headers;
};
export type qualitySignalReadResponseError = (qualitySignalReadResponseDefault) & {
  headers: Headers;
};

export const getQualitySignalReadUrl = (signalId: string,
    params?: QualitySignalReadParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/signals/${signalId}?${stringifiedParams}` : `/api/v1/signals/${signalId}`
}

/**
 * FR-38: сигнал VisionQC как цепочка ступеней ансамбля (где дефект / какой тип), уверенность и качество наблюдения в б. п., вектор версий наблюдения (AD-29), иллюстрация — адрес материала. Уверенность ≠ вероятность брака.
 * @summary Исходный сигнал
 */
export const qualitySignalRead = async (signalId: string,
    params?: QualitySignalReadParams, options?: RequestInit): Promise<qualitySignalReadResponseSuccess> => {

  const res = await fetch(getQualitySignalReadUrl(signalId,params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: qualitySignalReadResponseError['data'], status?: number} = new globalThis.Error();
    const data : qualitySignalReadResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: qualitySignalReadResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as qualitySignalReadResponseSuccess
}





export const getQualitySignalReadQueryKey = (signalId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<QualitySignalReadParams>,) => {
    return [
    'api','v1','signals',signalId, ...(params ? [params] : [])
    ] as const;
    }


export const getQualitySignalReadQueryOptions = <TData = Awaited<ReturnType<typeof qualitySignalRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(signalId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<QualitySignalReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof qualitySignalRead>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getQualitySignalReadQueryKey(signalId,params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof qualitySignalRead>>> = ({ signal }) => qualitySignalRead(toValue(signalId),toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, enabled: computed(() => toValue(signalId) !== null && toValue(signalId) !== undefined), ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof qualitySignalRead>>, TError, TData>
}

export type QualitySignalReadQueryResult = NonNullable<Awaited<ReturnType<typeof qualitySignalRead>>>
export type QualitySignalReadQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Исходный сигнал
 */

export function useQualitySignalRead<TData = Awaited<ReturnType<typeof qualitySignalRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 signalId: MaybeRefOrGetter<string>,
    params?: MaybeRefOrGetter<QualitySignalReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof qualitySignalRead>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getQualitySignalReadQueryOptions(signalId,params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type ingestSourceListResponse200 = {
  data: SourceList
  status: 200
}

export type ingestSourceListResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type ingestSourceListResponseSuccess = (ingestSourceListResponse200) & {
  headers: Headers;
};
export type ingestSourceListResponseError = (ingestSourceListResponseDefault) & {
  headers: Headers;
};

export const getIngestSourceListUrl = (params?: IngestSourceListParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/sources?${stringifiedParams}` : `/api/v1/sources`
}

/**
 * AD-7, AD-9: устройства, шлюзы, терминалы — ключ, состояние, последний source_seq, дыры последовательности, расхождение часов, объём карантина.
 * @summary Источники событий
 */
export const ingestSourceList = async (params?: IngestSourceListParams, options?: RequestInit): Promise<ingestSourceListResponseSuccess> => {

  const res = await fetch(getIngestSourceListUrl(params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: ingestSourceListResponseError['data'], status?: number} = new globalThis.Error();
    const data : ingestSourceListResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: ingestSourceListResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as ingestSourceListResponseSuccess
}





export const getIngestSourceListQueryKey = (params?: MaybeRefOrGetter<IngestSourceListParams>,) => {
    return [
    'api','v1','sources', ...(params ? [params] : [])
    ] as const;
    }


export const getIngestSourceListQueryOptions = <TData = Awaited<ReturnType<typeof ingestSourceList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(params?: MaybeRefOrGetter<IngestSourceListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof ingestSourceList>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getIngestSourceListQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof ingestSourceList>>> = ({ signal }) => ingestSourceList(toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof ingestSourceList>>, TError, TData>
}

export type IngestSourceListQueryResult = NonNullable<Awaited<ReturnType<typeof ingestSourceList>>>
export type IngestSourceListQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Источники событий
 */

export function useIngestSourceList<TData = Awaited<ReturnType<typeof ingestSourceList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 params?: MaybeRefOrGetter<IngestSourceListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof ingestSourceList>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getIngestSourceListQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type journalStreamSubscribeResponse200 = {
  data: string
  status: 200
}

export type journalStreamSubscribeResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type journalStreamSubscribeResponseSuccess = (journalStreamSubscribeResponse200) & {
  headers: Headers;
};
export type journalStreamSubscribeResponseError = (journalStreamSubscribeResponseDefault) & {
  headers: Headers;
};

export const getJournalStreamSubscribeUrl = (params?: JournalStreamSubscribeParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/stream?${stringifiedParams}` : `/api/v1/stream`
}

/**
 * Канал sse из contracts/events/asyncapi.yaml: `id:` = seq, `event: entity_changed`, `data:` — EntityChanged (components). Фронтенд инвалидирует ключ Vue Query [сущность, id]; переподключение — с Last-Event-ID (AD-21). Бюджет «событие → экран» ≤ 2 с (FR-2).
 * @summary Живые обновления столов (SSE)
 */
export const journalStreamSubscribe = async (params?: JournalStreamSubscribeParams, options?: RequestInit): Promise<journalStreamSubscribeResponseSuccess> => {

  const res = await fetch(getJournalStreamSubscribeUrl(params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: journalStreamSubscribeResponseError['data'], status?: number} = new globalThis.Error();
    const data : journalStreamSubscribeResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: journalStreamSubscribeResponseSuccess['data'] = body !== null ? body : ''
  return { data, status: res.status, headers: res.headers } as journalStreamSubscribeResponseSuccess
}





export const getJournalStreamSubscribeQueryKey = (params?: MaybeRefOrGetter<JournalStreamSubscribeParams>,) => {
    return [
    'api','v1','stream', ...(params ? [params] : [])
    ] as const;
    }


export const getJournalStreamSubscribeQueryOptions = <TData = Awaited<ReturnType<typeof journalStreamSubscribe>>, TError = globalThis.Error & { info?: Problem; status?: number }>(params?: MaybeRefOrGetter<JournalStreamSubscribeParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof journalStreamSubscribe>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getJournalStreamSubscribeQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof journalStreamSubscribe>>> = ({ signal }) => journalStreamSubscribe(toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof journalStreamSubscribe>>, TError, TData>
}

export type JournalStreamSubscribeQueryResult = NonNullable<Awaited<ReturnType<typeof journalStreamSubscribe>>>
export type JournalStreamSubscribeQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Живые обновления столов (SSE)
 */

export function useJournalStreamSubscribe<TData = Awaited<ReturnType<typeof journalStreamSubscribe>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 params?: MaybeRefOrGetter<JournalStreamSubscribeParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof journalStreamSubscribe>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getJournalStreamSubscribeQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type notificationsTaskListResponse200 = {
  data: TaskList
  status: 200
}

export type notificationsTaskListResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type notificationsTaskListResponseSuccess = (notificationsTaskListResponse200) & {
  headers: Headers;
};
export type notificationsTaskListResponseError = (notificationsTaskListResponseDefault) & {
  headers: Headers;
};

export const getNotificationsTaskListUrl = (params?: NotificationsTaskListParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/tasks?${stringifiedParams}` : `/api/v1/tasks`
}

/**
 * FR-57: задачи пользователя и его роли — физические, доп. проверка, эскалации, запросы решения; срок и цена задержки.
 * @summary Задачи и уведомления
 */
export const notificationsTaskList = async (params?: NotificationsTaskListParams, options?: RequestInit): Promise<notificationsTaskListResponseSuccess> => {

  const res = await fetch(getNotificationsTaskListUrl(params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: notificationsTaskListResponseError['data'], status?: number} = new globalThis.Error();
    const data : notificationsTaskListResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: notificationsTaskListResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as notificationsTaskListResponseSuccess
}





export const getNotificationsTaskListQueryKey = (params?: MaybeRefOrGetter<NotificationsTaskListParams>,) => {
    return [
    'api','v1','tasks', ...(params ? [params] : [])
    ] as const;
    }


export const getNotificationsTaskListQueryOptions = <TData = Awaited<ReturnType<typeof notificationsTaskList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(params?: MaybeRefOrGetter<NotificationsTaskListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof notificationsTaskList>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getNotificationsTaskListQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof notificationsTaskList>>> = ({ signal }) => notificationsTaskList(toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof notificationsTaskList>>, TError, TData>
}

export type NotificationsTaskListQueryResult = NonNullable<Awaited<ReturnType<typeof notificationsTaskList>>>
export type NotificationsTaskListQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Задачи и уведомления
 */

export function useNotificationsTaskList<TData = Awaited<ReturnType<typeof notificationsTaskList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 params?: MaybeRefOrGetter<NotificationsTaskListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof notificationsTaskList>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getNotificationsTaskListQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type notificationsTaskAcknowledgeResponse200 = {
  data: Receipt
  status: 200
}

export type notificationsTaskAcknowledgeResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type notificationsTaskAcknowledgeResponseSuccess = (notificationsTaskAcknowledgeResponse200) & {
  headers: Headers;
};
export type notificationsTaskAcknowledgeResponseError = (notificationsTaskAcknowledgeResponseDefault) & {
  headers: Headers;
};

export const getNotificationsTaskAcknowledgeUrl = (taskId: string,) => {




  return `/api/v1/tasks/${taskId}/acknowledge`
}

/**
 * FR-57: задача выполнена, принята или отклонена с примечанием (task.task.acknowledged).
 * @summary Отметить задачу
 */
export const notificationsTaskAcknowledge = async (taskId: string,
    acknowledgeTask: AcknowledgeTask, options?: RequestInit): Promise<notificationsTaskAcknowledgeResponseSuccess> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
const res = await fetch(getNotificationsTaskAcknowledgeUrl(taskId),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(acknowledgeTask)
  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: notificationsTaskAcknowledgeResponseError['data'], status?: number} = new globalThis.Error();
    const data : notificationsTaskAcknowledgeResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: notificationsTaskAcknowledgeResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as notificationsTaskAcknowledgeResponseSuccess
}





export const getNotificationsTaskAcknowledgeMutationKey = () => ['notificationsTaskAcknowledge'] as const;

export const getNotificationsTaskAcknowledgeMutationOptions = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof notificationsTaskAcknowledge>>, TError,NotificationsTaskAcknowledgeMutationVariables, TContext>, fetch?: RequestInit}
): UseMutationOptions<Awaited<ReturnType<typeof notificationsTaskAcknowledge>>, TError,NotificationsTaskAcknowledgeMutationVariables, TContext> => {

const mutationKey = getNotificationsTaskAcknowledgeMutationKey();
const {mutation: mutationOptions, fetch: fetchOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, fetch: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof notificationsTaskAcknowledge>>, NotificationsTaskAcknowledgeMutationVariables> = (props) => {
          const {taskId,data} = props ?? {};

          return  notificationsTaskAcknowledge(taskId,data,fetchOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type NotificationsTaskAcknowledgeMutationResult = NonNullable<Awaited<ReturnType<typeof notificationsTaskAcknowledge>>>
    export type NotificationsTaskAcknowledgeMutationBody = AcknowledgeTask
    export type NotificationsTaskAcknowledgeMutationError = globalThis.Error & { info?: Problem; status?: number }
    export type NotificationsTaskAcknowledgeMutationVariables = {taskId: string;data: AcknowledgeTask}

    /**
 * @summary Отметить задачу
 */
export const useNotificationsTaskAcknowledge = <TError = globalThis.Error & { info?: Problem; status?: number },
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof notificationsTaskAcknowledge>>, TError,NotificationsTaskAcknowledgeMutationVariables, TContext>, fetch?: RequestInit}
 , queryClient?: QueryClient): UseMutationReturnType<
        Awaited<ReturnType<typeof notificationsTaskAcknowledge>>,
        TError,
        NotificationsTaskAcknowledgeMutationVariables,
        TContext
      > => {
      return useMutation(getNotificationsTaskAcknowledgeMutationOptions(options), queryClient);
    }

export type journalTimelineReadResponse200 = {
  data: TimelineData
  status: 200
}

export type journalTimelineReadResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type journalTimelineReadResponseSuccess = (journalTimelineReadResponse200) & {
  headers: Headers;
};
export type journalTimelineReadResponseError = (journalTimelineReadResponseDefault) & {
  headers: Headers;
};

export const getJournalTimelineReadUrl = (params?: JournalTimelineReadParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/timeline?${stringifiedParams}` : `/api/v1/timeline`
}

/**
 * FR-4, FR-155: диапазон доступной истории (или прогона) и метки значимых событий — эскалации, стоп точки процесса, всплески, новые версии процесса; ось — из параметра axis.
 * @summary Таймлайн живой карты
 */
export const journalTimelineRead = async (params?: JournalTimelineReadParams, options?: RequestInit): Promise<journalTimelineReadResponseSuccess> => {

  const res = await fetch(getJournalTimelineReadUrl(params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: journalTimelineReadResponseError['data'], status?: number} = new globalThis.Error();
    const data : journalTimelineReadResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: journalTimelineReadResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as journalTimelineReadResponseSuccess
}





export const getJournalTimelineReadQueryKey = (params?: MaybeRefOrGetter<JournalTimelineReadParams>,) => {
    return [
    'api','v1','timeline', ...(params ? [params] : [])
    ] as const;
    }


export const getJournalTimelineReadQueryOptions = <TData = Awaited<ReturnType<typeof journalTimelineRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(params?: MaybeRefOrGetter<JournalTimelineReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof journalTimelineRead>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getJournalTimelineReadQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof journalTimelineRead>>> = ({ signal }) => journalTimelineRead(toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof journalTimelineRead>>, TError, TData>
}

export type JournalTimelineReadQueryResult = NonNullable<Awaited<ReturnType<typeof journalTimelineRead>>>
export type JournalTimelineReadQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Таймлайн живой карты
 */

export function useJournalTimelineRead<TData = Awaited<ReturnType<typeof journalTimelineRead>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 params?: MaybeRefOrGetter<JournalTimelineReadParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof journalTimelineRead>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getJournalTimelineReadQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







export type accessWorkplaceListResponse200 = {
  data: PostList
  status: 200
}

export type accessWorkplaceListResponseDefault = {
  data: Problem
  status: Exclude<HTTPStatusCodes, 200>
}

export type accessWorkplaceListResponseSuccess = (accessWorkplaceListResponse200) & {
  headers: Headers;
};
export type accessWorkplaceListResponseError = (accessWorkplaceListResponseDefault) & {
  headers: Headers;
};

export const getAccessWorkplaceListUrl = (params?: AccessWorkplaceListParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/workplaces?${stringifiedParams}` : `/api/v1/workplaces`
}

/**
 * Панель «Посты» стола руководителя и участок мастера (PRD §3a, FR-6, FR-81): участок — кто назначен — на месте ли (СКУД, ключ) — текущая деталь.
 * @summary Посты
 */
export const accessWorkplaceList = async (params?: AccessWorkplaceListParams, options?: RequestInit): Promise<accessWorkplaceListResponseSuccess> => {

  const res = await fetch(getAccessWorkplaceListUrl(params),
  {
    ...options,
    method: 'GET'


  }
)


  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  if (!res.ok) {

    const err: globalThis.Error & {info?: accessWorkplaceListResponseError['data'], status?: number} = new globalThis.Error();
    const data : accessWorkplaceListResponseError['data'] = body ? JSON.parse(body) : {}
    err.info = data;
    err.status = res.status;
    throw err;
  }
  const data: accessWorkplaceListResponseSuccess['data'] = body ? JSON.parse(body) : {}
  return { data, status: res.status, headers: res.headers } as accessWorkplaceListResponseSuccess
}





export const getAccessWorkplaceListQueryKey = (params?: MaybeRefOrGetter<AccessWorkplaceListParams>,) => {
    return [
    'api','v1','workplaces', ...(params ? [params] : [])
    ] as const;
    }


export const getAccessWorkplaceListQueryOptions = <TData = Awaited<ReturnType<typeof accessWorkplaceList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(params?: MaybeRefOrGetter<AccessWorkplaceListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof accessWorkplaceList>>, TError, TData>>, fetch?: RequestInit}
) => {

const {query: queryOptions, fetch: fetchOptions} = options ?? {};

  const queryKey =  getAccessWorkplaceListQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof accessWorkplaceList>>> = ({ signal }) => accessWorkplaceList(toValue(params), { signal, ...fetchOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof accessWorkplaceList>>, TError, TData>
}

export type AccessWorkplaceListQueryResult = NonNullable<Awaited<ReturnType<typeof accessWorkplaceList>>>
export type AccessWorkplaceListQueryError = globalThis.Error & { info?: Problem; status?: number }


/**
 * @summary Посты
 */

export function useAccessWorkplaceList<TData = Awaited<ReturnType<typeof accessWorkplaceList>>, TError = globalThis.Error & { info?: Problem; status?: number }>(
 params?: MaybeRefOrGetter<AccessWorkplaceListParams>, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof accessWorkplaceList>>, TError, TData>>, fetch?: RequestInit}
 , queryClient?: QueryClient
 ): UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getAccessWorkplaceListQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as UseQueryReturnType<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  query.queryKey = unref(queryOptions).queryKey as DataTag<QueryKey, TData, TError>;

  return query;
}







