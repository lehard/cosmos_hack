/**
 * Приём событий глазами администратора (эпик 14; FR-30, FR-33, FR-41; AD-26):
 * карантин сообщений, источники с ключами и дырами в номерах, метрики приёма.
 * Карантин и метрики живые (эпик 06), команды источников — модуль ops.
 *
 * Операции — `ingest.quarantine.list|read`, `ingest.message.reprocess`,
 * `ingest.source.list`, `ingest.metrics.read`, `ops.source.disable|enable`
 * (contracts/openapi.yaml) — сгенерированным клиентом. Ключи — `[quarantine, …]`:
 * сообщение SSE `quarantine` перечитывает и карантин, и источники, и метрики.
 */
import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import {
  ingestMessageReprocess,
  ingestMetricsRead,
  ingestQuarantineList,
  ingestQuarantineRead,
  ingestSourceList,
  opsSourceDisable,
  opsSourceEnable,
} from '@/shared/api/generated/client'
import type {
  IngestMetrics,
  IngestQuarantineListParams,
  QuarantineEntry,
  QuarantineEntryState,
  ReprocessMessage,
  SourceView,
  SourceViewState,
  SwitchSource,
} from '@/shared/api/generated/model'
import { entityKeys } from '@/shared/api/keys'
import type { ApiError } from '@/shared/api/problem'

export type { IngestMetrics, QuarantineEntry, QuarantineEntryState, ReprocessMessage, SourceView, SourceViewState, SwitchSource }

export const quarantineKeys = entityKeys('quarantine')

/** Период перечитывания метрик и источников, мс: они меняются и без событий по объекту. */
export const INGEST_POLL_MS = 15_000

/** Фильтр карантина: состояние записи, код причины, источник. */
export type QuarantineFilter = Pick<IngestQuarantineListParams, 'state' | 'code' | 'source_id'>

/** Карантин сообщений (`ingest.quarantine.list`, FR-41): записи с кодами причин. */
export function useQuarantine(filter: MaybeRefOrGetter<QuarantineFilter> = {}) {
  const params = computed(() => ({ limit: 200, ...toValue(filter) }))
  return useQuery({
    queryKey: computed(() => quarantineKeys.list('entries', params.value)),
    queryFn: ({ signal }) => ingestQuarantineList(params.value, { signal }),
    retry: false,
    placeholderData: keepPreviousData,
  })
}

/** Запись карантина с исходным содержимым (`ingest.quarantine.read`). */
export function useQuarantineEntry(id: MaybeRefOrGetter<string | null>) {
  return useQuery({
    queryKey: computed(() => quarantineKeys.one(toValue(id) ?? '')),
    queryFn: ({ signal }) => ingestQuarantineRead(toValue(id) ?? '', { signal }),
    enabled: computed(() => !!toValue(id)),
    retry: false,
  })
}

/** Источники событий (`ingest.source.list`, FR-33): состояние, ключ, последний номер, дыры. */
export function useSources() {
  return useQuery({
    queryKey: quarantineKeys.list('sources'),
    queryFn: ({ signal }) => ingestSourceList({ limit: 200 }, { signal }),
    retry: false,
    refetchInterval: INGEST_POLL_MS,
  })
}

/** Метрики приёма (`ingest.metrics.read`): задержка, полнота, повторы, отказы, карантин. */
export function useIngestMetrics() {
  return useQuery({
    queryKey: quarantineKeys.list('metrics'),
    queryFn: ({ signal }) => ingestMetricsRead({ signal }),
    retry: false,
    refetchInterval: INGEST_POLL_MS,
  })
}

/** Команда приёма: повторная обработка записи карантина или отключение/включение источника. */
export type IngestCommand =
  | { kind: 'reprocess'; quarantineId: string; body: ReprocessMessage }
  | { kind: 'disable' | 'enable'; sourceId: string; body: SwitchSource }

function send(cmd: IngestCommand) {
  switch (cmd.kind) {
    case 'reprocess':
      return ingestMessageReprocess(cmd.quarantineId, cmd.body)
    case 'disable':
      return opsSourceDisable(cmd.sourceId, cmd.body)
    case 'enable':
      return opsSourceEnable(cmd.sourceId, cmd.body)
  }
}

/**
 * Команда администратора над приёмом (FR-41: переобработка принимается один раз —
 * AD-7; номер изделия без человека не подставляется). После успеха
 * перечитываются карантин, источники и метрики.
 */
export function useIngestCommand() {
  const queryClient = useQueryClient()
  return useMutation<Awaited<ReturnType<typeof send>>, ApiError, IngestCommand>({
    mutationFn: send,
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: quarantineKeys.all }),
  })
}
