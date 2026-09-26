/**
 * Прогон тестового сценария (эпик 14; FR-129, FR-152; AD-26, AD-36, AD-37, AD-38):
 * пульт демонстрации и табло «ожидалось → получилось».
 *
 * Операции модуля `simulation` (contracts/openapi.yaml) — сгенерированным
 * клиентом: `simulation.scenario.list`, `simulation.run.list`,
 * `simulation.run.read`, `simulation.board.read`, `simulation.injection.list`,
 * `simulation.run.start|pause|resume|set_speed|stop`, `simulation.injection.apply`.
 * Ключи кэша — `[run, id, …]` и `[run, '@list', …]`: их инвалидирует SSE `run`.
 *
 * Пока прогон идёт, состояние и табло ещё и перечитываются по таймеру —
 * доменные часы двигаются и без событий по самому прогону (AD-37).
 */
import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import {
  simulationBoardRead,
  simulationInjectionApply,
  simulationInjectionList,
  simulationRunList,
  simulationRunPause,
  simulationRunRead,
  simulationRunResume,
  simulationRunSetSpeed,
  simulationRunStart,
  simulationRunStop,
  simulationScenarioList,
} from '@/shared/api/generated/client'
import type { ApplyInjection, Receipt, RunControl, SetSpeed, StartRun } from '@/shared/api/generated/model'
import { entityKeys } from '@/shared/api/keys'
import { statusOf, type ApiError } from '@/shared/api/problem'
import { useMomentStore } from '@/shared/model/moment'
import { isActive, isIdle, pickCurrentRun } from './model/run'
import { useRunFocusStore } from './model/focus'
import { simulationRunPlan } from './model/plan'

export * from './model/run'
export * from './model/focus'
export * from './model/clock'
export * from './model/plan'
export type { ApplyInjection, Receipt, RunControl, SetSpeed, StartRun }

export const runKeys = entityKeys('run')

/** Период перечитывания идущего прогона, мс (SSE — основной канал). */
export const RUN_POLL_MS = 2000

/** Сценарии пульта (`simulation.scenario.list`). */
export function useScenarios() {
  return useQuery({
    queryKey: runKeys.list('scenarios'),
    queryFn: ({ signal }) => simulationScenarioList({ signal }),
    retry: false,
    staleTime: 60_000,
  })
}

/** Период перечитывания списка прогонов, мс: пока есть идущий прогон — чаще. */
export const RUNS_POLL_MS = 5000
export const RUNS_IDLE_POLL_MS = 15_000

/**
 * Прогоны (`simulation.run.list`). Список читают пульт и часы в шапке всех
 * столов (Д-85): без идущего прогона — реже, чтобы не нагружать сервер.
 */
export function useRuns() {
  return useQuery({
    queryKey: runKeys.list('runs'),
    queryFn: ({ signal }) => simulationRunList({ limit: 50 }, { signal }),
    retry: false,
    // Нет права читать прогоны (роль «только карточка») — не переспрашивать.
    refetchInterval: (q) =>
      statusOf(q.state.error) === 403 ? false : (q.state.data?.data?.items ?? []).some((r) => isActive(r.state) && !isIdle(r)) ? RUNS_POLL_MS : RUNS_IDLE_POLL_MS,
  })
}

/**
 * Текущий прогон пульта: заданный срезом стола или адресом (`?run=`), иначе
 * выбранный на пульте, иначе по правилу pickCurrentRun.
 * @param fixed — run_id из среза или адреса
 */
export function useCurrentRunId(fixed: MaybeRefOrGetter<string | undefined> = undefined) {
  const focus = useRunFocusStore()
  const runs = useRuns()
  const runId = computed<string | null>(() => {
    const f = toValue(fixed)
    if (f) return f
    return pickCurrentRun(runs.data.value?.data?.items ?? [], focus.runId)?.run_id ?? null
  })
  return { runId, runs }
}

/** Параметры момента для чтения прогона. */
function useMomentParams() {
  const moment = useMomentStore()
  return computed(() => ({ ...moment.params }))
}

/** Состояние прогона на момент (`simulation.run.read`): шаг, часы, пауза, скорость, ожидание. */
export function useRun(runId: MaybeRefOrGetter<string | null>) {
  const params = useMomentParams()
  return useQuery({
    queryKey: computed(() => runKeys.one(toValue(runId) ?? '', 'state', params.value)),
    queryFn: ({ signal }) => simulationRunRead(toValue(runId) ?? '', params.value, { signal }),
    enabled: computed(() => !!toValue(runId)),
    retry: false,
    placeholderData: (prev, prevQuery) => (prevQuery?.queryKey[1] === toValue(runId) ? prev : undefined),
    refetchInterval: (q) => (q.state.data?.data && isActive(q.state.data.data.state) && q.state.data.data.state !== 'paused' ? RUN_POLL_MS : false),
  })
}

/**
 * Табло «ожидалось → получилось» (`simulation.board.read`, AD-26, кейс §5.1).
 * @param live — прогон идёт: табло перечитывается по таймеру
 */
export function useBoard(runId: MaybeRefOrGetter<string | null>, live: MaybeRefOrGetter<boolean> = false) {
  const params = useMomentParams()
  return useQuery({
    queryKey: computed(() => runKeys.one(toValue(runId) ?? '', 'board', params.value)),
    queryFn: ({ signal }) => simulationBoardRead(toValue(runId) ?? '', params.value, { signal }),
    enabled: computed(() => !!toValue(runId)),
    retry: false,
    placeholderData: keepPreviousData,
    refetchInterval: () => (toValue(live) ? RUN_POLL_MS : false),
  })
}

/**
 * План прогона (`simulation.run.plan`, Д-85): чего ждём и что дальше. Пока
 * прогон идёт — перечитывается вместе с состоянием; операции ещё нет на
 * сервере (501) — пульт просто не показывает план.
 */
export function useRunPlan(runId: MaybeRefOrGetter<string | null>, live: MaybeRefOrGetter<boolean> = false) {
  return useQuery({
    queryKey: computed(() => runKeys.one(toValue(runId) ?? '', 'plan')),
    queryFn: ({ signal }) => simulationRunPlan(toValue(runId) ?? '', { limit: 50 }, { signal }),
    enabled: computed(() => !!toValue(runId)),
    retry: false,
    placeholderData: (prev, prevQuery) => (prevQuery?.queryKey[1] === toValue(runId) ? prev : undefined),
    refetchInterval: () => (toValue(live) ? RUN_POLL_MS : false),
  })
}

/** Кнопки цифрового стенда (`simulation.injection.list`, FR-152). */
export function useInjections(runId: MaybeRefOrGetter<string | null>) {
  return useQuery({
    queryKey: computed(() => runKeys.one(toValue(runId) ?? '', 'injections')),
    queryFn: ({ signal }) => simulationInjectionList(toValue(runId) ?? '', { signal }),
    enabled: computed(() => !!toValue(runId)),
    retry: false,
  })
}

/** Команда пульта: что сделать и с каким телом. */
export type RunCommand =
  | { kind: 'start'; scenarioId: string; body: StartRun }
  | { kind: 'pause' | 'resume' | 'stop'; runId: string; body: RunControl }
  | { kind: 'speed'; runId: string; body: SetSpeed }
  | { kind: 'inject'; runId: string; body: ApplyInjection }

/** Выполнить команду пульта сгенерированным клиентом. */
function send(cmd: RunCommand) {
  switch (cmd.kind) {
    case 'start':
      return simulationRunStart(cmd.scenarioId, cmd.body)
    case 'pause':
      return simulationRunPause(cmd.runId, cmd.body)
    case 'resume':
      return simulationRunResume(cmd.runId, cmd.body)
    case 'stop':
      return simulationRunStop(cmd.runId, cmd.body)
    case 'speed':
      return simulationRunSetSpeed(cmd.runId, cmd.body)
    case 'inject':
      return simulationInjectionApply(cmd.runId, cmd.body)
  }
}

/**
 * Команда пульта (FR-129, FR-152). Запуск и остановка меняют весь мир прогона
 * (AD-38: данные в пространстве имён прогона) — перечитываются все запросы;
 * остальные команды — только прогон и табло (столы обновит SSE).
 */
export function useRunCommand() {
  const queryClient = useQueryClient()
  const focus = useRunFocusStore()
  return useMutation<Awaited<ReturnType<typeof send>>, ApiError, RunCommand>({
    mutationFn: send,
    onSuccess: (_res, cmd) => {
      if (cmd.kind === 'start' || cmd.kind === 'stop') {
        // Квитанция не несёт run_id: новый прогон пульт найдёт в списке прогонов.
        if (cmd.kind === 'start') focus.selectRun(null)
        void queryClient.invalidateQueries()
        return
      }
      void queryClient.invalidateQueries({ queryKey: runKeys.all })
    },
  })
}
