<script setup lang="ts">
/**
 * Виджет «Тестовые сценарии» — пульт демонстрации (FR-129, FR-152; AD-26,
 * AD-36, AD-37, AD-38; кейс §5.1) — контейнер.
 *
 * Данные — entities/run: сценарии (`simulation.scenario.list`), прогоны
 * (`simulation.run.list`), состояние текущего прогона (`simulation.run.read`),
 * кнопки стенда (`simulation.injection.list`); сеанс — policy_seq и рабочее
 * место для команд (AD-39, AD-15). Команды — `simulation.run.start|pause|
 * resume|set_speed|stop` и `simulation.injection.apply`; `command_id` —
 * UUIDv7, один на нажатие (AD-7).
 *
 * На паузе сервер останавливает поток событий и доменные часы — все столы
 * показывают состояние на этот момент (AD-26); пульт лишь показывает это.
 * Выбор прогона — в хранилище выбора (Pinia): табло на том же столе смотрит
 * тот же прогон. Срез стола: `run_id` — закрепить прогон.
 */
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import {
  clampSpeed,
  defaultConfig,
  isActive,
  isIdle,
  useCurrentRunId,
  useInjections,
  useRun,
  useRunCommand,
  useRunFocusStore,
  useRunPlan,
  useScenarios,
  type Injection,
  type RunCommand,
  type RunConfig,
  type RunControlKind,
} from '@/entities/run'
import { useSession } from '@/entities/session'
import { backendModeOf } from '@/shared/api/response'
import type { WidgetProps } from '@/shared/config/widget'
import { newCommandId } from '@/shared/lib/command-id'
import { useMomentStore } from '@/shared/model/moment'
import { WidgetFrame } from '@/shared/ui'
import ScenarioConsoleView from './ScenarioConsoleView.vue'

const props = defineProps<WidgetProps>()
const route = useRoute()
const focus = useRunFocusStore()
const moment = useMomentStore()
const session = useSession()

const str = (v: unknown): string | undefined => (typeof v === 'string' && v ? v : undefined)
const fixedRun = computed(() => str(route?.query.run) ?? str(props.slice.run_id))

const scenariosQ = useScenarios()
const scenarios = computed(() => scenariosQ.data.value?.data?.items ?? null)

const { runId } = useCurrentRunId(fixedRun)
const runQ = useRun(runId)
const run = computed(() => runQ.data.value?.data ?? null)
const live = computed(() => !!run.value && !isIdle(run.value) && isActive(run.value.state))
// План прогона (Д-85): чего ждём и что дальше; сервер плана не отдал — пульт работает как раньше.
const planQ = useRunPlan(computed(() => (run.value && !isIdle(run.value) ? runId.value : null)), live)
const plan = computed(() => planQ.data.value?.data ?? null)
const injectionsQ = useInjections(computed(() => (live.value ? runId.value : null)))
const injections = computed(() => injectionsQ.data.value?.data?.items ?? [])

// Выбранный сценарий: выбор на пульте, иначе сценарий текущего прогона, иначе первый.
const selectedScenario = computed(() => focus.scenarioId ?? run.value?.scenario_id ?? scenarios.value?.[0]?.scenario_id ?? null)
const config = ref<RunConfig>(defaultConfig(null))
// Новый сценарий — конфигурация из его определения; перечитывание списка её не сбрасывает.
watch(
  () => [selectedScenario.value, !!scenarios.value] as const,
  ([id]) => {
    config.value = defaultConfig(scenarios.value?.find((s) => s.scenario_id === id) ?? null)
  },
  { immediate: true },
)

const command = useRunCommand()
const injected = ref<number | null>(null)
watch(runId, () => (injected.value = null))

/** Общие поля команды: id, версия политики, рабочее место, seq, на котором видели прогон (AD-7, AD-39). */
function meta() {
  const s = session.data.value?.data
  return {
    command_id: newCommandId(),
    basis_seq: run.value?.basis_seq ?? 0,
    policy_seq: s?.policy_seq ?? 0,
    ...(s?.workplace?.id ? { workplace_id: s.workplace.id } : {}),
  }
}

async function exec(cmd: RunCommand): Promise<void> {
  try {
    const res = await command.mutateAsync(cmd)
    if (cmd.kind === 'inject') injected.value = res.data.event_ids.length
  } catch {
    // Текст ошибки показывает представление (command.error).
  }
}

function start(): void {
  const id = selectedScenario.value
  if (!id) return
  const c = config.value
  void exec({
    kind: 'start',
    scenarioId: id,
    body: {
      ...meta(),
      mode: c.mode,
      speed: clampSpeed(c.speed),
      ...(c.items !== null ? { items: c.items } : {}),
      ...(c.seed !== null ? { seed: c.seed } : {}),
    },
  })
}

function control(kind: RunControlKind): void {
  if (runId.value) void exec({ kind, runId: runId.value, body: meta() })
}

function speed(v: number): void {
  if (runId.value) void exec({ kind: 'speed', runId: runId.value, body: { ...meta(), speed: clampSpeed(v) } })
}

function inject(injection: Injection['injection'], target: string | undefined): void {
  if (!runId.value) return
  void exec({ kind: 'inject', runId: runId.value, body: { ...meta(), injection, ...(target ? { target_event_id: target } : {}) } })
}
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="backendModeOf(runQ.data.value ?? scenariosQ.data.value)"
    :loading="scenariosQ.isPending.value && !scenarios"
    :error="scenarios ? undefined : scenariosQ.error.value"
    :empty="!!scenarios && !scenarios.length && !run"
    empty-key="empty.scenarioNotStarted"
    :data-widget="widgetId"
  >
    <ScenarioConsoleView
      v-if="scenarios"
      v-model:config="config"
      :scenarios="scenarios"
      :selected-scenario="selectedScenario"
      :run="run"
      :plan="plan"
      :injections="injections"
      :can-act="!moment.isReplay"
      :busy="command.isPending.value"
      :error="command.error.value ?? runQ.error.value ?? undefined"
      :injected="injected"
      :density="density"
      @select-scenario="focus.selectScenario"
      @start="start"
      @control="control"
      @speed="speed"
      @inject="inject"
    />
  </WidgetFrame>
</template>
