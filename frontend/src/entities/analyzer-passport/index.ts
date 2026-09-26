/**
 * Сущность «analyzer_passport» — анализаторы VisionQC и OperatorVision, паспорта
 * допуска и контур адаптации (эпики 33, 40; FR-97…FR-101, AD-29). Ключи кэша —
 * по соглашению shared/api/keys.ts: сообщение SSE `analyzer_passport` (допуск,
 * автооткат, возврат) перечитывает списки и паспорт.
 *
 * Операции — `vision.analyzer.list`, `vision.passport.read|admit|reinstate|retire`,
 * `vision.check.list`, `vision.escape.list`, `vision.observation.read`,
 * `vision.example.list` (contracts/openapi.yaml) — сгенерированным клиентом.
 * Прогон сценария (AD-38) — параметр `run_id`: автооткат в прогоне виден только в нём.
 */
import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import {
  visionAnalyzerList,
  visionCheckList,
  visionEscapeList,
  visionExampleList,
  visionObservationRead,
  visionPassportAdmit,
  visionPassportRead,
  visionPassportReinstate,
  visionPassportRetire,
} from '@/shared/api/generated/client'
import type {
  AdaptationEscape,
  AdmitPassport,
  AnalyzerCheck,
  AnalyzerPassport,
  AnalyzerSummary,
  LabeledExample,
  ObservationAccount,
  ReinstatePassport,
  RetirePassport,
} from '@/shared/api/generated/model'
import { entityKeys } from '@/shared/api/keys'
import type { ApiError } from '@/shared/api/problem'
import { useMomentStore } from '@/shared/model/moment'

export type { AdaptationEscape, AdmitPassport, AnalyzerCheck, AnalyzerPassport, AnalyzerSummary, LabeledExample, ObservationAccount }

export const analyzerPassportKeys = entityKeys('analyzer_passport')

/** Параметры чтения: момент (AD-22) и прогон сценария (AD-38). */
function useParams(run: MaybeRefOrGetter<string | undefined>) {
  const moment = useMomentStore()
  return computed(() => {
    const r = toValue(run)
    return r ? { ...moment.params, run_id: r } : { ...moment.params }
  })
}

/** Анализаторы и их действующие паспорта (`vision.analyzer.list`). */
export function useAnalyzers(run: MaybeRefOrGetter<string | undefined> = undefined) {
  const params = useParams(run)
  return useQuery({
    queryKey: computed(() => analyzerPassportKeys.list('analyzers', params.value)),
    queryFn: ({ signal }) => visionAnalyzerList(params.value, { signal }),
    retry: false,
    placeholderData: keepPreviousData,
  })
}

/** Паспорт допуска с приостановкой, историей и контролем дрейфа (`vision.passport.read`). */
export function useAnalyzerPassport(id: MaybeRefOrGetter<string | null>, run: MaybeRefOrGetter<string | undefined> = undefined) {
  const params = useParams(run)
  return useQuery({
    queryKey: computed(() => analyzerPassportKeys.one(toValue(id) ?? '', params.value)),
    queryFn: ({ signal }) => visionPassportRead(toValue(id) ?? '', params.value, { signal }),
    enabled: computed(() => !!toValue(id)),
    retry: false,
  })
}

/** Отчёты проверки анализатора (`vision.check.list`): экзамен, эталонный набор, тень, дрейф. */
export function useAnalyzerChecks(id: MaybeRefOrGetter<string | null>, run: MaybeRefOrGetter<string | undefined> = undefined) {
  const params = useParams(run)
  return useQuery({
    queryKey: computed(() => analyzerPassportKeys.one(toValue(id) ?? '', 'checks', params.value)),
    queryFn: ({ signal }) => visionCheckList(toValue(id) ?? '', { ...params.value, limit: 100 }, { signal }),
    enabled: computed(() => !!toValue(id)),
    retry: false,
  })
}

/** Пропуски брака и изделия на перепроверку (`vision.escape.list`, FR-100). */
export function useVisionEscapes(run: MaybeRefOrGetter<string | undefined> = undefined) {
  const params = useParams(run)
  return useQuery({
    queryKey: computed(() => analyzerPassportKeys.list('escapes', params.value)),
    queryFn: ({ signal }) => visionEscapeList(params.value, { signal }),
    retry: false,
    placeholderData: keepPreviousData,
  })
}

/** Размеченные примеры из подтверждённых решений (`vision.example.list`, FR-99). */
export function useLabeledExamples(run: MaybeRefOrGetter<string | undefined> = undefined) {
  const params = useParams(run)
  return useQuery({
    queryKey: computed(() => analyzerPassportKeys.list('examples', params.value)),
    queryFn: ({ signal }) => visionExampleList(params.value, { signal }),
    retry: false,
    placeholderData: keepPreviousData,
  })
}

/** «Какими версиями и почему» по наблюдению анализатора (`vision.observation.read`, FR-98). */
export function useObservationAccount(eventId: MaybeRefOrGetter<string | null>) {
  const moment = useMomentStore()
  return useQuery({
    queryKey: computed(() => analyzerPassportKeys.list('observation', toValue(eventId) ?? '', moment.params)),
    queryFn: ({ signal }) => visionObservationRead(toValue(eventId) ?? '', moment.params, { signal }),
    enabled: computed(() => !!toValue(eventId)),
    retry: false,
  })
}

/** Команда над паспортом допуска: допуск новой версии, возврат после отката, вывод. */
export type PassportCommand =
  | { kind: 'admit'; body: AdmitPassport }
  | { kind: 'reinstate'; passportId: string; body: ReinstatePassport }
  | { kind: 'retire'; passportId: string; body: RetirePassport }

function send(cmd: PassportCommand) {
  switch (cmd.kind) {
    case 'admit':
      return visionPassportAdmit(cmd.body)
    case 'reinstate':
      return visionPassportReinstate(cmd.passportId, cmd.body)
    case 'retire':
      return visionPassportRetire(cmd.passportId, cmd.body)
  }
}

/**
 * Команда над паспортом (FR-98, FR-101): возврат после автоотката — только
 * начальник ОТК (иначе отказ `analyzer.reinstate_requires_head_of_qc`). После
 * успеха перечитываются анализаторы и паспорта.
 */
export function usePassportCommand() {
  const queryClient = useQueryClient()
  return useMutation<Awaited<ReturnType<typeof send>>, ApiError, PassportCommand>({
    mutationFn: send,
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: analyzerPassportKeys.all }),
  })
}
