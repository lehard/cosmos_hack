<script setup lang="ts">
/**
 * Пометка происхождения значения времени (кейс §5.2, FR-88, FR-140): «Передано
 * источником» / «Вычислено системой» / смешанное. Длительность без
 * происхождения помечена явно — а не выдана за данные источника.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { originOf, type MetricValue } from '../model/value'

const props = defineProps<{ value: Pick<MetricValue, 'origin' | 'unit'> }>()
const { t } = useI18n()
const info = computed(() => originOf(props.value))
</script>

<template>
  <span v-if="info" class="origin" :data-origin="info.code" :data-warn="info.warn || undefined" :title="t('widgets.analytics.origin.hint')">
    {{ t(info.key) }}
  </span>
</template>

<style scoped>
.origin {
  display: inline-block;
  padding: 0 6px;
  border: 1px solid #d1d5db;
  border-radius: 8px;
  color: #4b5563;
  font-size: 11px;
  font-weight: 400;
  white-space: nowrap;
}

.origin[data-origin='computed_by_system'] {
  border-style: dashed;
}

.origin[data-warn] {
  border-color: #e0a100;
  color: #7a5a00;
}
</style>
