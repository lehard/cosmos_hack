<script setup lang="ts">
/**
 * Число показателя: значение в своих единицах и пометка происхождения для
 * времени. «Оценка невозможна» (`unknown`) показывается словами и прочерком —
 * это не ноль и не «норма» (NFR-UI-4).
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { formatValue, type MetricValue } from '../model/value'
import OriginMark from './OriginMark.vue'

const props = withDefaults(defineProps<{ value: MetricValue; unknown?: boolean; showOrigin?: boolean }>(), { unknown: false, showOrigin: true })
const { t, n } = useI18n()
const text = computed(() => formatValue({ t, n: (v, f) => n(v, f) }, props.value))
</script>

<template>
  <span class="metric-number" :data-unknown="unknown || undefined">
    <template v-if="unknown">
      <span class="dash" aria-hidden="true">—</span>
      <span class="unknown" data-testid="unknown">{{ t('widgets.analytics.value.unknown') }}</span>
    </template>
    <span v-else class="value" data-testid="value">{{ text }}</span>
    <OriginMark v-if="showOrigin" :value="value" />
  </span>
</template>

<style scoped>
.metric-number {
  display: inline-flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: baseline;
}

.value {
  font-variant-numeric: tabular-nums;
}

.unknown {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
  font-weight: 400;
}
</style>
