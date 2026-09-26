<script setup lang="ts">
/**
 * Часы прогона в шапке всех столов (Д-85): пока идёт прогон тестового
 * сценария — компактно доменное время и скорость «23.09 11:08 · ×60». Между
 * опросами часы идущего прогона досчитываются по скорости; пока прогон ждёт
 * человека или стоит на паузе — стоят. Без прогона и при просмотре прошлого
 * шапку не трогают. Активный прогон — shared/api/active-run (его опрашивает
 * оболочка; своего опроса здесь нет).
 */
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { NTooltip } from 'naive-ui'
import { formatRunClock, runClockAt, runClockTicking, type RunState } from '@/entities/run'
import { activeRun } from '@/shared/api/active-run'
import { useMomentStore } from '@/shared/model/moment'

const moment = useMomentStore()

const run = computed(() => {
  const r = activeRun.value
  return r?.clock_at ? { run_id: r.run_id, clock_at: r.clock_at, speed: r.speed ?? 1, state: r.state as RunState } : null
})
const visible = computed(() => !moment.isReplay && !!run.value)
const ticking = computed(() => !!run.value && runClockTicking(run.value))

/** Почему время стоит: кто и что должен сделать (из waiting_for активного прогона) — в подсказке. */
const WHO: Record<string, string> = {
  site_foreman: 'мастера',
  performer: 'сварщика',
  quality_inspector: 'контролёра ОТК',
  technologist: 'технолога',
  chief_welder: 'главного сварщика',
  head_of_qc: 'начальника ОТК',
}
const waitText = computed(() => {
  const r = activeRun.value
  if (r?.state === 'paused') return 'Пауза'
  const w = r?.state === 'waiting_for_decision' ? r.waiting_for : null
  if (!w) return ''
  const what = (w.title ?? '').replace(/^[^:«]*:\s*/, '')
  return `Время стоит — ждём ${WHO[w.role] ?? w.role}${what ? `: ${what}` : ''}`
})

// Каждый ответ опроса — новая точка отсчёта.
const observedAt = ref(Date.now())
const now = ref(Date.now())
watch(activeRun, () => (observedAt.value = now.value = Date.now()))
const timer = setInterval(() => {
  if (visible.value && ticking.value) now.value = Date.now()
}, 500)
onBeforeUnmount(() => clearInterval(timer))

// Сервер ведёт часы шагами по доменной минуте: свежий ответ может быть чуть позади
// досчитанного — пока прогон идёт, часы назад не отступают; смена прогона или состояния — снова по серверу.
let shown = { key: '', at: NaN }
const clock = computed(() => {
  const r = run.value
  if (!r) return ''
  let at = runClockAt(r, observedAt.value, now.value)
  const key = `${r.run_id}:${r.state}:${r.speed}`
  if (ticking.value && shown.key === key && shown.at > at) at = shown.at
  shown = { key, at }
  return formatRunClock(at)
})
</script>

<template>
  <span v-if="visible && run" class="run-clock" data-testid="run-clock" :data-state="run.state" :data-ticking="ticking || undefined">
    <span class="dot" aria-hidden="true" />
    <span class="time" data-testid="run-clock-time">{{ clock }}</span>
    <span class="speed" aria-hidden="true">·</span>
    <span class="speed" data-testid="run-clock-speed">×{{ run.speed }}</span>
    <NTooltip v-if="waitText" placement="bottom">
      <template #trigger>
        <span class="halt" tabindex="0" :aria-label="waitText" data-testid="run-clock-wait">⏸</span>
      </template>
      {{ waitText }}
    </NTooltip>
  </span>
</template>

<style scoped>
.run-clock {
  display: inline-flex;
  flex: none;
  gap: var(--ant-space-2);
  align-items: center;
  padding: 2px var(--ant-space-2);
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-sm);
  background: var(--ant-surface-subtle);
  color: var(--ant-text);
  white-space: nowrap;
}

.time {
  font-family: var(--ant-font-mono);
  font-weight: var(--ant-fw-bold);
}

.speed {
  color: var(--ant-text-2);
}

.halt {
  color: var(--ant-status-attention-text);
  cursor: help;
}

.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--ant-text-3);
}

.run-clock[data-ticking] .dot {
  background: var(--ant-status-success);
  animation: run-clock-pulse 1s ease-in-out infinite;
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
