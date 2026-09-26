<script setup lang="ts">
/**
 * Пульт тестовых сценариев — представление (FR-129, FR-152; AD-26, AD-37, AD-38;
 * кейс §5.1). Слева — список сценариев и конфигурация прогона (изделия, скорость,
 * seed, режим), справа — текущий прогон: состояние, шаг, доменное время,
 * скорость, пауза и продолжение, план прогона («сейчас ждём», ближайшие события,
 * запланированные сбои — Д-85), и кнопки цифрового стенда.
 *
 * Данных не читает и команд не шлёт — только показывает и сообщает о намерениях
 * (контейнер ScenarioConsoleWidget). В воспроизведении (`canAct = false`)
 * команды выключены (FR-4).
 */
import { computed, reactive, watch } from 'vue'
import { headMode, setHeadMode } from '@/shared/api/active-run'
import { useI18n } from 'vue-i18n'
import { NAlert, NButton, NInput, NInputNumber, NProgress, NRadioButton, NRadioGroup } from 'naive-ui'
import {
  RUN_STATE_TONE,
  SPEED_MAX,
  SPEED_MIN,
  SPEED_PRESETS,
  actionKey,
  clampSpeed,
  configError,
  controlsOf,
  displayStep,
  isActive,
  isIdle,
  progressOf,
  type Injection,
  type Run,
  type RunConfig,
  type RunPlan,
  type RunControlKind,
  type Scenario,
} from '@/entities/run'
import { statusPalette } from '@/shared/api/generated/statuses'
import type { Density } from '@/shared/config/widget'
import { codeToKey } from '@/shared/i18n'
import { useProblemText } from '@/shared/i18n/problem'
import { ActionButton } from '@/shared/ui'
import RunPlanPanel from './RunPlanPanel.vue'

const props = withDefaults(
  defineProps<{
    scenarios: Scenario[]
    selectedScenario?: string | null
    config: RunConfig
    run?: Run | null
    /** План прогона (`simulation.run.plan`, Д-85); нет — только «ждёт решения». */
    plan?: RunPlan | null
    injections?: Injection[]
    /** Команды разрешены (не воспроизведение). */
    canAct?: boolean
    /** Команда выполняется. */
    busy?: boolean
    /** Ошибка последней команды. */
    error?: unknown
    /** Итог последней команды стенда: сколько записей журнала внесено. */
    injected?: number | null
    density?: Density
  }>(),
  { selectedScenario: null, run: null, plan: null, injections: () => [], canAct: true, busy: false, error: undefined, injected: null, density: 'comfortable' },
)
const emit = defineEmits<{
  'select-scenario': [id: string]
  'update:config': [config: RunConfig]
  start: []
  control: [kind: RunControlKind]
  speed: [speed: number]
  inject: [injection: Injection['injection'], target: string | undefined]
}>()
const { t, te, d } = useI18n()
const problemText = useProblemText()

const scenario = computed(() => props.scenarios.find((s) => s.scenario_id === props.selectedScenario) ?? null)
const cfgError = computed(() => configError(props.config))
const patch = (p: Partial<RunConfig>) => emit('update:config', { ...props.config, ...p })

const idle = computed(() => (props.run ? isIdle(props.run) : true))
const controls = computed(() => (props.run ? controlsOf(props.run) : { pause: false, resume: false, stop: false }))
const live = computed(() => !!props.run && !idle.value && isActive(props.run.state))
const stateColor = computed(() => (props.run ? statusPalette[RUN_STATE_TONE[props.run.state]] : undefined))
const scenarioTitle = (id: string) => props.scenarios.find((s) => s.scenario_id === id)?.title ?? id

const time = (iso: string | null | undefined) => (iso ? d(new Date(iso), 'dateTime') : '—')
const roleText = (role: string) => (te(`roles.${codeToKey(role)}`) ? t(`roles.${codeToKey(role)}`) : role)
const actionText = (action: string) => {
  const key = `widgets.scenarios.waitAction.${actionKey(action)}`
  return te(key) ? t(key) : action
}

/** Своя скорость: команда уходит, когда значение в границах и отличается от текущей. */
function customSpeed(v: number | null): void {
  if (v !== null && v !== props.run?.speed && v === clampSpeed(v)) emit('speed', v)
}

// Целевые события для кнопок стенда, которым нужна запись (повтор, опоздание, подделка).
const targets = reactive<Record<string, string>>({})
/**
 * Кнопки, которые принимают целевое событие, даже если оно необязательно
 * (FR-152, эпик 36): пусто — сервер берёт последнее подходящее событие прогона.
 */
const TARGETABLE: ReadonlySet<Injection['injection']> = new Set(['duplicate_event', 'late_event', 'corrupt_frame', 'machine_fault', 'tamper_outside', 'light_change'])
const hasTarget = (item: Injection) => item.needs_target || TARGETABLE.has(item.injection)
watch(
  () => props.run?.run_id,
  () => Object.keys(targets).forEach((k) => delete targets[k]),
)
function inject(item: Injection): void {
  const target = targets[item.injection]?.trim() || undefined
  emit('inject', item.injection, target)
}

// Режим этого браузера: «голова» — без активного прогона (shared/api/active-run).
const head = headMode()
</script>

<template>
  <div class="console" :class="`density-${density}`" data-testid="scenario-console">
    <div class="mode" data-testid="browser-mode">
      <span class="muted">{{ t('widgets.scenarios.mode') }}</span>
      <NRadioGroup :value="head ? 'head' : 'simulation'" size="small" @update:value="(v: string) => setHeadMode(v === 'head')">
        <NRadioButton value="simulation">{{ t('widgets.scenarios.modeSimulation') }}</NRadioButton>
        <NRadioButton value="head">{{ t('widgets.scenarios.modeHead') }}</NRadioButton>
      </NRadioGroup>
      <span class="muted note">{{ t('widgets.scenarios.modeNote') }}</span>
    </div>
    <section class="scenarios" :aria-label="t('widgets.scenarios.list')">
      <h4>{{ t('widgets.scenarios.list') }}</h4>
      <ul class="list">
        <li
          v-for="s in scenarios"
          :key="s.scenario_id"
          class="scenario"
          :data-id="s.scenario_id"
          :aria-selected="s.scenario_id === selectedScenario"
          tabindex="0"
          @click="emit('select-scenario', s.scenario_id)"
          @keydown.enter.prevent="emit('select-scenario', s.scenario_id)"
        >
          <strong>{{ s.title }}</strong>
          <span class="muted">{{ s.scenario_id }} · {{ t('widgets.scenarios.version', { version: s.version }) }}</span>
          <span v-if="s.case_refs?.length" class="muted">{{ t('widgets.scenarios.caseRefs', { refs: s.case_refs.join(', ') }) }}</span>
          <span class="muted">
            {{ t('widgets.scenarios.decisions', { n: s.decisions }) }}<template v-if="s.assertions"> · {{ t('widgets.scenarios.assertions', { n: s.assertions }) }}</template>
          </span>
        </li>
      </ul>

      <form v-if="scenario" class="config" data-testid="config" @submit.prevent="!cfgError && canAct && emit('start')">
        <h4>{{ t('widgets.scenarios.config.title') }}</h4>
        <p v-if="scenario.description" class="muted">{{ scenario.description }}</p>
        <div class="field">
          <span>{{ t('testStand.config.timeSpeed', { speed: config.speed }) }}</span>
          <div class="speeds">
            <NButton
              v-for="p in SPEED_PRESETS"
              :key="p"
              size="tiny"
              :type="p === config.speed ? 'primary' : 'default'"
              :data-testid="`cfg-speed-${p}`"
              @click="patch({ speed: p })"
              >×{{ p }}</NButton
            >
          </div>
        </div>
        <div class="field">
          <span>{{ t('widgets.scenarios.config.mode') }}</span>
          <NRadioGroup :value="config.mode" size="small" @update:value="(v: RunConfig['mode']) => patch({ mode: v })">
            <NRadioButton value="interactive" data-testid="cfg-mode-interactive">Интерактивно</NRadioButton>
            <NRadioButton value="autocheck" data-testid="cfg-mode-autocheck">Автопроверка</NRadioButton>
          </NRadioGroup>
          <span class="muted">{{ t(`widgets.scenarios.mode.${config.mode}`) }}</span>
        </div>
        <p v-if="cfgError" class="error" data-testid="cfg-error">{{ t(cfgError) }}</p>
        <ActionButton type="primary" attr-type="submit" :disabled="!canAct || busy || !!cfgError" data-testid="start" :label="t('testStand.controls.start')" />
      </form>
      <p v-else class="muted">{{ t('widgets.scenarios.selectHint') }}</p>
    </section>

    <section class="run" :aria-label="t('widgets.scenarios.current')" data-testid="run">
      <h4>{{ t('widgets.scenarios.current') }}</h4>
      <p v-if="!run" class="muted" data-testid="no-run">{{ t('widgets.scenarios.noRun') }}</p>
      <template v-else>
        <p v-if="idle" class="muted" data-testid="idle">{{ t('widgets.scenarios.idle', { scenario: scenarioTitle(run.scenario_id) }) }}</p>
        <div v-else class="line">
          <strong data-testid="run-id">{{ t('testStand.run', { run: run.run_id }) }}</strong>
          <span class="muted">{{ scenarioTitle(run.scenario_id) }}</span>
        </div>
        <div class="line">
          <span class="state" :data-state="run.state" data-testid="run-state">
            <span class="dot" :style="{ background: stateColor }" aria-hidden="true" />
            {{ t(`widgets.scenarios.state.${codeToKey(run.state)}`) }}
          </span>
          <span data-testid="run-step">{{ t('widgets.scenarios.step', { step: displayStep(run.step), steps: run.steps }) }}</span>
        </div>
        <NProgress type="line" :percentage="progressOf(run)" :show-indicator="false" :height="6" />
        <dl class="facts">
          <dt>{{ t('widgets.scenarios.clock') }}</dt>
          <dd data-testid="run-clock">{{ time(run.clock_at) }}</dd>
          <dt>{{ t('widgets.scenarios.speed') }}</dt>
          <dd data-testid="run-speed">×{{ run.speed }}</dd>
          <template v-if="!idle">
            <dt>{{ t('widgets.scenarios.config.mode') }}</dt>
            <dd>{{ run.mode === 'autocheck' ? 'Автопроверка' : 'Интерактивно' }}</dd>
            <dt>{{ t('widgets.scenarios.startedAt') }}</dt>
            <dd>{{ time(run.started_at) }}</dd>
            <template v-if="run.finished_at">
              <dt>{{ t('widgets.scenarios.finishedAt') }}</dt>
              <dd>{{ time(run.finished_at) }}</dd>
            </template>
          </template>
          <dt>{{ t('desks.verificationBoard') }}</dt>
          <dd data-testid="run-board">{{ t('widgets.scenarios.boardScore', { passed: run.board_passed, total: run.board_total }) }}</dd>
        </dl>

        <NAlert v-if="run.state === 'paused' && !idle" type="info" :bordered="false" :show-icon="false" data-testid="paused">
          {{ t('testStand.pausedState', { time: time(run.clock_at) }) }}
        </NAlert>
        <RunPlanPanel v-if="plan && !idle && plan.run_id === run.run_id" :plan="plan" />
        <NAlert v-else-if="run.state === 'waiting_for_decision' && run.waiting_for" type="warning" :bordered="false" :show-icon="false" data-testid="waiting">
          <div>{{ t('testStand.waitingForDecision', { role: roleText(run.waiting_for.role), what: actionText(run.waiting_for.action) }) }}</div>
          <div class="muted">{{ t('widgets.scenarios.waitingObject', { object: run.waiting_for.object_id }) }}</div>
          <div class="muted">{{ t('testStand.waitingHint') }}</div>
        </NAlert>

        <div class="controls">
          <ActionButton v-if="controls.pause" :disabled="!canAct || busy" data-testid="pause" @click="emit('control', 'pause')" :label="t('testStand.controls.pause')" />
          <ActionButton v-if="controls.resume" type="primary" :disabled="!canAct || busy" data-testid="resume" @click="emit('control', 'resume')" :label="t('testStand.controls.resume')" />
          <ActionButton v-if="controls.stop" :disabled="!canAct || busy" data-testid="stop" @click="emit('control', 'stop')" :label="t('testStand.controls.stop')" />
        </div>
        <div v-if="live" class="field">
          <span>{{ t('widgets.scenarios.speed') }}</span>
          <div class="speeds">
            <NButton
              v-for="p in SPEED_PRESETS"
              :key="p"
              size="tiny"
              :type="p === run.speed ? 'primary' : 'default'"
              :disabled="!canAct || busy || p === run.speed"
              :data-testid="`speed-${p}`"
              @click="emit('speed', p)"
              >×{{ p }}</NButton
            >
            <NInputNumber
              :value="run.speed"
              :min="SPEED_MIN"
              :max="SPEED_MAX"
              :precision="0"
              size="tiny"
              class="custom-speed"
              :update-value-on-input="false"
              :disabled="!canAct || busy"
              :aria-label="t('widgets.scenarios.config.speedCustom')"
              data-testid="speed-custom"
              @update:value="customSpeed"
            />
          </div>
        </div>
      </template>

      <NAlert v-if="error" type="error" :bordered="false" :show-icon="false" data-testid="command-error">
        {{ t('widgets.scenarios.commandFailed') }}: {{ problemText(error) }}
      </NAlert>

      <section class="stand" :aria-label="t('testStand.injections.title')" data-testid="stand">
        <h4>{{ t('testStand.title') }} — {{ t('testStand.injections.title') }}</h4>
        <p class="muted">{{ live ? t('widgets.scenarios.stand.hint') : t('widgets.scenarios.stand.noRun') }}</p>
        <ul v-if="live" class="injections">
          <li v-for="item in injections" :key="item.injection" :data-injection="item.injection">
            <ActionButton size="small" :disabled="!canAct || busy || !item.available || (item.needs_target && !targets[item.injection]?.trim())" @click="inject(item)" :label="item.title" />
            <NInput
              v-if="hasTarget(item)"
              v-model:value="targets[item.injection]"
              size="small"
              class="target"
              :placeholder="t(item.needs_target ? 'widgets.scenarios.stand.target' : 'widgets.scenarios.stand.targetOptional')"
              :aria-label="t('widgets.scenarios.stand.target')"
            />
            <span v-if="item.description" class="muted">{{ item.description }}</span>
            <span v-if="!item.available" class="muted">{{ t('widgets.scenarios.stand.unavailable') }}</span>
            <span v-if="item.injection === 'tamper_outside'" class="muted">{{ t('testStand.demoOnly') }}</span>
          </li>
        </ul>
        <p v-if="injected !== null" class="ok" data-testid="injected">{{ t('widgets.scenarios.stand.applied', { n: injected }) }}</p>
      </section>
    </section>
  </div>
</template>

<style scoped>
.field :deep(.n-radio-group) {
  display: flex;
  flex-wrap: wrap;
}

.console {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 40px;
  font-size: var(--ant-fs-body);
}

.mode {
  grid-column: 1 / -1;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
}

.mode .note {
  font-size: var(--ant-fs-meta);
}

.density-large {
  font-size: var(--ant-fs-lg);
}

h4 {
  margin: 0 0 6px;
  font-size: 1em;
}

.list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin: 0 0 12px;
  padding: 0;
  list-style: none;
}

.scenario {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 6px 8px;
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-md);
  cursor: pointer;
}

.scenario[aria-selected='true'] {
  border-color: var(--ant-accent);
  background: var(--ant-accent-soft);
}

.config,
.run,
.stand {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.config label,
.field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.speeds,
.controls,
.line {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}

.custom-speed {
  width: 110px;
}

.state {
  display: inline-flex;
  gap: 6px;
  align-items: center;
  font-weight: var(--ant-fw-bold);
}

.dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
}

.facts {
  display: grid;
  grid-template-columns: 9em 1fr;
  gap: 6px 16px;
  margin: 0;
}

.facts dt {
  color: var(--ant-text-3);
}

.facts dd {
  margin: 0;
}

.injections {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.injections li {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 8px;
  align-items: center;
}

.target {
  width: 160px;
}

.muted {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.error {
  margin: 0;
  color: var(--ant-status-danger);
}

.ok {
  margin: 0;
  color: var(--ant-status-success);
}
</style>
