<script setup lang="ts">
/**
 * Часы прогона в шапке всех столов (Д-85, AD-37): доменное «сейчас» прогона
 * тестового сценария и скорость — «23.09 11:08 · ×60». Между ответами сервера
 * часы идущего прогона досчитываются по скорости (точка пульсирует — время
 * идёт); пока прогон ждёт решения человека или стоит на паузе, часы стоят, и
 * рядом сказано, чего ждём. Нет идущего прогона или смотрим прошлое — ничего
 * не показывает (Д-70: время в шапке постоянно не висит).
 *
 * Данные — `simulation.run.list` (текущий прогон) и `simulation.run.read`
 * (`clock_at`, `speed`, `state`, `waiting_for`); повод перечитать — SSE `run`.
 */
import { computed, onScopeDispose, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NTooltip } from 'naive-ui'
import { formatRunClock, runClockAt, runClockTicking, runClockTickMs, showsRunClock, useCurrentRunId, useRun } from '@/entities/run'
import { backendModeOf } from '@/shared/api/response'
import { codeToKey } from '@/shared/i18n'
import { useMomentStore } from '@/shared/model/moment'

const { t, te } = useI18n()
const moment = useMomentStore()

const { runId } = useCurrentRunId()
const runQ = useRun(runId)
const run = computed(() => runQ.data.value?.data ?? null)
const visible = computed(() => !moment.isReplay && showsRunClock(run.value))
// Мир заготовок двигает часы шагами пульта — досчитывать между ответами нечего.
const ticking = computed(() => !!run.value && runClockTicking(run.value) && backendModeOf(runQ.data.value) !== 'fixtures')

const now = ref(Date.now())
let timer: ReturnType<typeof setInterval> | null = null
function stop(): void {
  if (timer) clearInterval(timer)
  timer = null
}
watch(
  () => [visible.value && ticking.value, run.value?.speed ?? 1] as const,
  ([on, speed]) => {
    stop()
    now.value = Date.now()
    if (on) timer = setInterval(() => (now.value = Date.now()), runClockTickMs(speed))
  },
  { immediate: true },
)
onScopeDispose(stop)

// Сервер ведёт часы шагами по доменной минуте: свежий ответ может быть чуть
// позади досчитанного. Пока прогон идёт, часы на экране назад не отступают;
// смена прогона или состояния (встал в ожидание) — снова по серверу.
let shown = { key: '', at: NaN }
const at = computed(() => {
  const r = run.value
  if (!r) return NaN
  if (!ticking.value) return Date.parse(r.clock_at)
  const key = `${r.run_id}:${r.state}:${r.speed}`
  let v = runClockAt(r, runQ.dataUpdatedAt.value, now.value)
  if (shown.key === key && shown.at > v) v = shown.at
  shown = { key, at: v }
  return v
})
const clock = computed(() => formatRunClock(at.value))

const roleText = (role: string) => (te(`roles.${codeToKey(role)}`) ? t(`roles.${codeToKey(role)}`) : role)

/** Состояние словами: идут / стоят — ждём ‹роль› / пауза. */
const stateText = computed(() => {
  const r = run.value
  if (!r) return ''
  if (r.state === 'waiting_for_decision') {
    return r.waiting_for ? t('shell.header.runClock.waitingRole', { role: roleText(r.waiting_for.role) }) : t('shell.header.runClock.waiting')
  }
  if (r.state === 'paused') return t('shell.header.runClock.paused')
  return ''
})
const hint = computed(() => {
  const r = run.value
  if (!r) return ''
  const lines = [t('shell.header.runClock.title', { run: r.run_id, scenario: r.scenario_id })]
  lines.push(ticking.value ? t('shell.header.runClock.ticking', { speed: r.speed }) : t('shell.header.runClock.stopped'))
  if (r.state === 'waiting_for_decision' && r.waiting_for) {
    lines.push(t('shell.header.runClock.waitingFor', { role: roleText(r.waiting_for.role), what: r.waiting_for.title || r.waiting_for.action }))
  }
  return lines.join('\n')
})
</script>

<template>
  <NTooltip v-if="visible && run">
    <template #trigger>
      <span class="run-clock" data-testid="run-clock" :data-state="run.state" :data-ticking="ticking || undefined" tabindex="0" :aria-label="hint">
        <span class="dot" aria-hidden="true" />
        <span class="time" data-testid="run-clock-time">{{ clock }}</span>
        <span class="speed" data-testid="run-clock-speed">×{{ run.speed }}</span>
        <span v-if="stateText" class="state ant-ellipsis" data-testid="run-clock-state">{{ stateText }}</span>
      </span>
    </template>
    <span class="hint ant-wrap">{{ hint }}</span>
  </NTooltip>
</template>

<style scoped>
.run-clock {
  display: inline-flex;
  flex: 0 1 auto;
  gap: var(--ant-space-2);
  align-items: center;
  min-width: 0;
  padding: 2px var(--ant-space-2);
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-sm);
  background: var(--ant-surface-subtle);
  color: var(--ant-text);
  white-space: nowrap;
}

.time {
  flex: none;
  font-family: var(--ant-font-mono);
  font-weight: var(--ant-fw-bold);
}

.speed {
  flex: none;
  color: var(--ant-text-2);
}

.state {
  min-width: 0;
  max-width: 220px;
  color: var(--ant-text-2);
}

.dot {
  flex: none;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--ant-text-3);
}

.run-clock[data-ticking] .dot {
  background: var(--ant-status-success);
  animation: run-clock-pulse 1s ease-in-out infinite;
}

.run-clock[data-state='waiting_for_decision'] {
  background: var(--ant-status-attention-soft);
}

.run-clock[data-state='waiting_for_decision'] .dot {
  background: var(--ant-status-attention);
}

.hint {
  white-space: pre-line;
}

@keyframes run-clock-pulse {
  50% {
    opacity: 0.3;
  }
}

@media (prefers-reduced-motion: reduce) {
  .run-clock[data-ticking] .dot {
    animation: none;
  }
}
</style>
