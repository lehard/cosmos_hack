/**
 * Чтение показателей через сгенерированный клиент (AD-20, AD-21): операции
 * модуля analytics из contracts/openapi.yaml — `analytics.tile.list`,
 * `analytics.overview.read`, `analytics.metric.drilldown`,
 * `analytics.control_chart.read`, `analytics.node_counters.read`.
 *
 * Ключи кэша — по соглашению shared/api/keys.ts под видом сущности `live_map`
 * (объект прав этих операций — `live_map`): `[live_map, @list, analytics, …,
 * момент]`. Так сообщение SSE `live_map` обновляет показатели вместе со
 * счётчиками карты — после подтверждения, отклонения и позднего события
 * (кейс §5.2) экран перечитывает сервер сам.
 */
import { computed, toValue, type MaybeRefOrGetter, type Ref } from 'vue'
import { keepPreviousData, useInfiniteQuery, useQuery } from '@tanstack/vue-query'
import {
  analyticsControlChartRead,
  analyticsMetricDrilldown,
  analyticsNodeCountersRead,
  analyticsOverviewRead,
  analyticsTileList,
} from '@/shared/api/generated/client'
import type { AnalyticsOverview, BackendMode, ControlChart, MetricDrilldown, MetricTileList, NodeCounterSet } from '@/shared/api/generated/model'
import { entityKeys } from '@/shared/api/keys'
import { backendModeOf } from '@/shared/api/response'
import { useMomentStore } from '@/shared/model/moment'
import { useMetricFocusStore } from './model/focus'

export type { AnalyticsOverview, ControlChart, MetricDrilldown, MetricTileList, NodeCounterSet }

/** Ключи кэша аналитики — под видом сущности `live_map`. */
export const analyticsKeys = entityKeys('live_map')

/** Источник данных виджета: то, что контейнер передаёт рамке и представлению. */
export interface MetricSource<T> {
  data: Readonly<Ref<T | null>>
  isPending: Readonly<Ref<boolean>>
  error: Readonly<Ref<unknown>>
  /** Режим fixtures | live из заголовка ответа (FR-150). */
  mode: Readonly<Ref<BackendMode | null>>
}

type Enveloped<A> = { data: A; headers?: Headers }

function toSource<A>(q: { data: Readonly<Ref<Enveloped<A> | undefined>>; isLoading: Readonly<Ref<boolean>>; error: Readonly<Ref<unknown>> }): MetricSource<A> {
  return {
    data: computed(() => q.data.value?.data ?? null),
    isPending: computed(() => q.isLoading.value),
    error: computed(() => q.error.value ?? undefined),
    mode: computed(() => backendModeOf(q.data.value)),
  }
}

/** Параметры чтения: период из фокуса аналитики и момент (AD-21). */
function useReadParams() {
  const moment = useMomentStore()
  const focus = useMetricFocusStore()
  return computed(() => ({ period: focus.period, ...moment.params }))
}

/** Плитки показателей стола руководителя (`analytics.tile.list`). */
export function useMetricTiles(): MetricSource<MetricTileList> {
  const params = useReadParams()
  const q = useQuery({
    queryKey: computed(() => analyticsKeys.list('analytics', 'tiles', params.value)),
    queryFn: ({ signal }) => analyticsTileList(params.value, { signal }),
    placeholderData: keepPreviousData,
  })
  return toSource(q)
}

/** Полный набор показателей кейса (`analytics.overview.read`). */
export function useAnalyticsOverview(): MetricSource<AnalyticsOverview> {
  const params = useReadParams()
  const q = useQuery({
    queryKey: computed(() => analyticsKeys.list('analytics', 'overview', params.value)),
    queryFn: ({ signal }) => analyticsOverviewRead(params.value, { signal }),
    placeholderData: keepPreviousData,
  })
  return toSource(q)
}

/** Счётчики узлов: список узлов, аномалии и узлы без данных (`analytics.node_counters.read`). */
export function useNodeCounterSet(): MetricSource<NodeCounterSet> {
  const params = useReadParams()
  const q = useQuery({
    queryKey: computed(() => analyticsKeys.list('analytics', 'node-counters', params.value)),
    queryFn: ({ signal }) => analyticsNodeCountersRead(params.value, { signal }),
    placeholderData: keepPreviousData,
  })
  return toSource(q)
}

/** Контрольная карта узла (`analytics.control_chart.read`); пустой узел — запроса нет. */
export function useControlChart(stepKey: MaybeRefOrGetter<string | null>, metricId: MaybeRefOrGetter<string | undefined> = undefined): MetricSource<ControlChart> {
  const params = useReadParams()
  const q = useQuery({
    queryKey: computed(() => analyticsKeys.list('analytics', 'control-chart', toValue(stepKey) ?? '', toValue(metricId) ?? '', params.value)),
    queryFn: ({ signal }) => {
      const metric = toValue(metricId)
      return analyticsControlChartRead(toValue(stepKey)!, metric ? { ...params.value, metric_id: metric } : params.value, { signal })
    },
    enabled: computed(() => Boolean(toValue(stepKey))),
  })
  return toSource(q)
}

/** Раскрытие показателя: страницы строк вклада и итог. */
export interface DrilldownSource extends MetricSource<MetricDrilldown> {
  /** Есть следующая страница. */
  hasMore: Readonly<Ref<boolean>>
  /** Загрузить следующую страницу. */
  loadMore: () => void
  loadingMore: Readonly<Ref<boolean>>
}

/**
 * Раскрытие числа до строк вклада изделий и id исходных записей
 * (`analytics.metric.drilldown`, AD-45). Страницы склеиваются в один ответ.
 * @param metricId — показатель; null — ничего не выбрано
 * @param sliceKey — срез; нет — итог
 */
export function useMetricDrilldown(metricId: MaybeRefOrGetter<string | null>, sliceKey: MaybeRefOrGetter<string | undefined>): DrilldownSource {
  const params = useReadParams()
  const q = useInfiniteQuery({
    queryKey: computed(() => analyticsKeys.list('analytics', 'drilldown', toValue(metricId) ?? '', toValue(sliceKey) ?? '', params.value)),
    queryFn: ({ signal, pageParam }) => {
      const slice = toValue(sliceKey)
      return analyticsMetricDrilldown(
        toValue(metricId)!,
        { ...params.value, ...(slice ? { slice } : {}), ...(pageParam ? { cursor: pageParam } : {}) },
        { signal },
      )
    },
    initialPageParam: '' as string,
    getNextPageParam: (last) => last.data.next_cursor || undefined,
    enabled: computed(() => Boolean(toValue(metricId))),
    retry: false,
  })
  const pages = computed(() => q.data.value?.pages ?? [])
  return {
    data: computed(() => {
      const [first, ...rest] = pages.value
      if (!first) return null
      const last = rest.at(-1) ?? first
      return { ...first.data, items: pages.value.flatMap((p) => p.data.items), next_cursor: last.data.next_cursor }
    }),
    isPending: computed(() => q.isLoading.value),
    error: computed(() => q.error.value ?? undefined),
    mode: computed(() => backendModeOf(pages.value[0])),
    hasMore: computed(() => q.hasNextPage.value),
    loadingMore: computed(() => q.isFetchingNextPage.value),
    loadMore: () => void q.fetchNextPage(),
  }
}
