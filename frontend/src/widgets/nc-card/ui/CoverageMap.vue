<script setup lang="ts">
/**
 * Покрытие контролем (разбор «Камеры + ИИ»; FR-36, кейс §4.5): по точкам контроля —
 * какими методами, что получено, что ждём; виды дефектов, не проверенные методом,
 * способным их выявить («камера не видит внутренние поры — нужен рентген»). Итог
 * словами: «визуальный контроль пройден, полный контроль ещё не завершён».
 * Ничего не досчитывается: статусы и исходы — от сервера (`quality.coverage.read`).
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { METHOD_TEXT, type InspectionMethod } from '@/entities/nonconformity'
import type { InspectionCoverage } from '@/shared/api/generated/model'

const props = defineProps<{ coverage: InspectionCoverage; /** Показывать только точки этих шагов (сварка — KT-3); пусто — все. */ focus?: readonly string[] }>()
const { t } = useI18n()

const points = computed(() => {
  const all = props.coverage.points.filter((p) => p.required)
  const list = props.focus?.length ? all.filter((p) => props.focus!.includes(p.inspection_point)) : all
  const by = new Map<string, typeof list>()
  for (const p of list) by.set(p.inspection_point, [...(by.get(p.inspection_point) ?? []), p])
  return [...by.entries()].map(([point, methods]) => ({ point, methods }))
})
const tone = (p: InspectionCoverage['points'][number]) =>
  p.status === 'received' ? (p.outcome === 'defect_indicated' ? 'danger' : p.outcome === 'unable_to_assess' ? 'warn' : 'ok') : p.status === 'missing' ? 'missing' : 'pending'
const notChecked = computed(() => (props.coverage.types ?? []).filter((x) => x.status === 'not_checked'))
const summary = computed(() => {
  const pts = points.value.flatMap((x) => x.methods)
  if (!pts.length) return null
  const cameraDone = pts.some((p) => p.method === 'camera' && p.status === 'received')
  const allDone = pts.every((p) => p.status === 'received')
  if (allDone) return 'ncCard.coverage.summary.complete'
  return cameraDone ? 'ncCard.coverage.summary.visualDone' : 'ncCard.coverage.summary.pending'
})
</script>

<template>
  <div class="coverage" data-testid="coverage-map">
    <ul class="points">
      <li v-for="p in points" :key="p.point" class="point" :data-point="p.point">
        <span class="point-name">{{ p.point }}</span>
        <span v-for="m in p.methods" :key="`${m.step_key}:${m.method}`" class="method" :data-tone="tone(m)" :data-method="m.method">
          {{ t(METHOD_TEXT[m.method as InspectionMethod] ?? 'widgets.codes.method.other') }} · {{ t(`ncCard.coverage.status.${tone(m)}`) }}
        </span>
      </li>
    </ul>
    <p v-if="summary" class="summary ant-wrap" data-testid="coverage-summary">{{ t(summary) }}</p>
    <p v-if="notChecked.length" class="not-checked ant-wrap" data-testid="coverage-not-checked">
      {{ t('ncCard.coverage.notChecked') }}: {{ notChecked.map((x) => x.name).join(', ') }}
    </p>
  </div>
</template>

<style scoped>
.coverage {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
  min-width: 0;
}

.points {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  margin: 0;
  padding: 0;
  list-style: none;
}

.point {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-1) var(--ant-space-2);
  align-items: center;
  min-width: 0;
}

.point-name {
  min-width: 3.5em;
  font-weight: var(--ant-fw-bold);
}

.method {
  padding: 1px var(--ant-space-2);
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-pill);
  color: var(--ant-text-2);
  font-size: var(--ant-fs-meta);
}

.method[data-tone='ok'] {
  border-color: var(--ant-status-success);
  background: var(--ant-status-success-soft);
  color: var(--ant-status-success-text);
}

.method[data-tone='danger'] {
  border-color: var(--ant-status-danger);
  background: var(--ant-status-danger-soft);
  color: var(--ant-status-danger-text);
}

.method[data-tone='warn'],
.method[data-tone='missing'] {
  border-color: var(--ant-status-attention);
  background: var(--ant-status-attention-soft);
  color: var(--ant-status-attention-text);
}

.method[data-tone='pending'] {
  border-style: dashed;
}

.summary {
  margin: 0;
  font-weight: var(--ant-fw-bold);
}

.not-checked {
  margin: 0;
  color: var(--ant-status-attention-text);
  font-size: var(--ant-fs-meta);
}
</style>
