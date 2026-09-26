<script setup lang="ts">
/**
 * Плашка активного прогона (показ): на любом столе видно, что сейчас ждёт сценарий и от кого.
 * Своей роли — «Ждёт вас» и кнопка «Открыть» (окно объекта справа, `?open=`).
 * Смена прогона или шага — все запросы перечитываются в активном прогоне (shared/api/active-run).
 */
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useQueryClient } from '@tanstack/vue-query'
import { NButton } from 'naive-ui'
import { activeRun, refreshActiveRun } from '@/shared/api/active-run'
import { useSession } from '@/entities/session'
import { formatRunClock, runClockAt, type RunState } from '@/entities/run'

const emit = defineEmits<{ 'run-changed': [runId: string | null] }>()
const route = useRoute()
const router = useRouter()
const queryClient = useQueryClient()
const session = useSession()

let timer: ReturnType<typeof setInterval> | undefined
onMounted(() => {
  void refreshActiveRun()
  timer = setInterval(() => void refreshActiveRun(), 3000)
})
onBeforeUnmount(() => clearInterval(timer))

watch(
  () => activeRun.value?.run_id ?? null,
  (id) => {
    emit('run-changed', id)
    void queryClient.invalidateQueries()
  },
)
watch(
  () => [activeRun.value?.step, activeRun.value?.state],
  () => void queryClient.invalidateQueries(),
)

const myRole = computed(() => {
  const d = session.data.value as { role?: { id?: string; inherits?: string[] }; data?: { role?: { id?: string; inherits?: string[] } } } | undefined
  const r = d?.role ?? d?.data?.role
  return r ? [r.id, ...(r.inherits ?? [])] : []
})
const wait = computed(() => (activeRun.value?.state === 'waiting_for_decision' ? activeRun.value.waiting_for ?? null : null))
const mine = computed(() => !!wait.value && myRole.value.includes(wait.value.role))

const ROLE_TEXT: Record<string, string> = {
  site_foreman: 'Мастер',
  performer: 'Сварщик (исполнитель)',
  quality_inspector: 'Контролёр ОТК',
  technologist: 'Технолог',
  chief_welder: 'Главный сварщик',
  head_of_qc: 'Начальник ОТК',
  production_manager: 'Руководитель производства',
  administrator: 'Администратор',
}
const KIND_BY_ACTION: Record<string, string> = {
  'process.movement.receive': 'item',
  'process.operation.start': 'item',
  'nonconformity.item.isolate': 'item',
  'nonconformity.recheck.request': 'item',
  'nonconformity.presentation.resolve': 'item',
  'nonconformity.nonconformity.confirm': 'item',
  'nonconformity.disposition.set': 'nonconformity',
  'access.workplace.admit': 'workplace',
  'nonconformity.process_hold.set': 'workplace',
  'process.operation.finish': 'operation',
}
const openTarget = computed(() => {
  const w = wait.value
  if (!w?.object_id) return null
  const kind = KIND_BY_ACTION[w.action] ?? (/\/I-|:F-\d/.test(w.object_id) ? 'item' : null)
  return kind ? `${kind}:${w.object_id}` : null
})
function open(): void {
  if (openTarget.value) void router.push({ query: { ...route.query, open: openTarget.value } })
}
// Часы прогона идут между опросами по скорости (entities/run: runClockAt); в ожидании человека стоят.
const observedAt = ref(Date.now())
const now = ref(Date.now())
watch(activeRun, () => (observedAt.value = now.value = Date.now()))
const tick = setInterval(() => (now.value = Date.now()), 500)
onBeforeUnmount(() => clearInterval(tick))
const clock = computed(() => {
  const r = activeRun.value
  if (!r?.clock_at) return ''
  return formatRunClock(runClockAt({ clock_at: r.clock_at, speed: r.speed ?? 1, state: r.state as RunState }, observedAt.value, now.value))
})
</script>

<template>
  <div v-if="activeRun" class="run-banner" :class="{ mine, waiting: !!wait }" data-testid="run-banner">
    <span class="clock" title="Доменное время прогона">🕒 {{ clock }} · ×{{ activeRun.speed ?? 1 }}</span>
    <template v-if="wait">
      <strong v-if="mine">Ждёт вас:</strong>
      <span v-else>Сценарий ждёт — {{ ROLE_TEXT[wait.role] ?? wait.role }}:</span>
      <span>{{ wait.title ?? wait.action }}</span>
      <NButton v-if="openTarget" size="small" :type="mine ? 'primary' : 'default'" @click="open">Открыть</NButton>
    </template>
    <span v-else class="muted">Прогон идёт: {{ activeRun.step_title ?? '' }}</span>
  </div>
</template>

<style scoped>
.run-banner {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 12px;
  align-items: center;
  padding: 6px 16px;
  border-bottom: 1px solid var(--ant-border);
  background: var(--ant-accent-soft);
  font-size: var(--ant-fs-body);
}
.run-banner.waiting {
  background: #fff4d6;
}
.run-banner.mine {
  background: #ffe08a;
}
.clock {
  font-family: var(--ant-font-mono, monospace);
}
.muted {
  color: var(--ant-text-3);
}
</style>
