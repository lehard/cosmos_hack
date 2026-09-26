<script setup lang="ts">
/**
 * Наложение над узлом схемы (FR-2, FR-5): счётчики очередь / в работе / прошло /
 * дефекты за период, метки ограничения линии, аномалий и «оценка невозможна».
 * Числа считает сервер (AD-21) — здесь только показ.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Bottleneck, NodeAnomaly, NodeCounters } from '@/entities/live-map'
import { statusPalette } from '@/shared/api/generated/statuses'
import { codeToKey } from '@/shared/i18n'

const props = defineProps<{
  counters?: NodeCounters
  bottleneck?: Bottleneck | null
  anomalies?: readonly NodeAnomaly[]
  dataGap?: boolean
  selected?: boolean
}>()
const emit = defineEmits<{ select: [] }>()
const { t } = useI18n()

/** Счётчик → подпись и тон; дефекты красные только когда они есть. */
const cells = computed(() => {
  const c = props.counters
  if (!c) return []
  return [
    { kind: 'queue', value: c.queue, label: t('liveMap.counters.inQueue'), color: statusPalette.neutral },
    { kind: 'in_progress', value: c.in_progress, label: t('liveMap.counters.inWork'), color: statusPalette.info },
    { kind: 'passed', value: c.passed, label: t('liveMap.counters.passed'), color: statusPalette.muted },
    { kind: 'defects', value: c.defects, label: t('liveMap.counters.defects'), color: c.defects > 0 ? statusPalette.danger : statusPalette.muted },
  ]
})

const anomalyText = (a: NodeAnomaly) =>
  a.kind === 'downtime_over_threshold'
    ? t('liveMap.anomalies.downtimeOverThreshold', { threshold: a.threshold ?? '—' })
    : t(`liveMap.anomalies.${codeToKey(a.kind)}`)

const anomalyTitle = computed(() => (props.anomalies ?? []).map(anomalyText).join('\n'))
</script>

<template>
  <div class="node-badge" :data-selected="selected || undefined" @click.stop="emit('select')">
    <span v-if="bottleneck" class="flag bottleneck" data-flag="bottleneck" :title="bottleneck.wait ? `${t('liveMap.bottleneck')}: ${bottleneck.wait}` : t('liveMap.bottleneck')">
      {{ t('liveMap.bottleneckShort') }}
    </span>
    <span v-if="anomalies?.length" class="flag anomaly" data-flag="anomaly" :title="anomalyTitle">!</span>
    <span v-if="dataGap" class="flag gap" data-flag="data-gap" :title="t('inspection.outcome.unableToAssess')">?</span>
    <span
      v-for="c in cells"
      :key="c.kind"
      class="cell"
      :data-counter="c.kind"
      :title="`${c.label}: ${c.value}`"
      :style="{ borderColor: c.color, color: c.kind === 'defects' && c.value > 0 ? c.color : undefined }"
    >{{ c.value }}</span>
  </div>
</template>

<style scoped>
.node-badge {
  display: inline-flex;
  gap: 2px;
  align-items: center;
  transform: translateY(-100%);
  cursor: pointer;
  font: 600 11px/1 'PT Sans', sans-serif;
  white-space: nowrap;
}

.cell {
  min-width: 16px;
  padding: 2px 3px;
  border: 1px solid;
  border-radius: 3px;
  background: #fff;
  color: #1f2937;
  text-align: center;
}

.flag {
  padding: 2px 4px;
  border-radius: 3px;
  color: #fff;
}

.bottleneck {
  background: #d64545;
}

.anomaly {
  background: #e0a100;
}

.gap {
  background: #8a8f98;
}

.node-badge[data-selected] .cell {
  box-shadow: 0 0 0 1px #2f6fdb;
}
</style>
