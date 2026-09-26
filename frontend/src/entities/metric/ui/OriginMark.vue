<script setup lang="ts">
/**
 * Пометка происхождения значения времени (кейс §5.2, FR-88, FR-140): «Передано
 * источником» / «Вычислено системой» / смешанное. Длительность без
 * происхождения помечена явно — а не выдана за данные источника.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { meaningOf, originOf, type MetricValue } from '../model/value'

const props = defineProps<{ value: Pick<MetricValue, 'origin' | 'unit' | 'meaning' | 'meaning_note'> }>()
const { t } = useI18n()
const info = computed(() => originOf(props.value))
const meaning = computed(() => meaningOf(props.value))
</script>

<template>
  <span v-if="info" class="origin" :data-origin="info.code" :data-warn="info.warn || undefined" :title="t('widgets.analytics.origin.hint')">
    {{ t(info.key) }}
  </span>
  <span v-if="meaning" class="origin meaning" :data-meaning="meaning.code" data-testid="meaning" :title="meaning.note ?? t('widgets.analytics.meaning.hint')">
    {{ t(meaning.key) }}
  </span>
</template>

<style scoped>
.origin {
  display: inline-block;
  padding: 0 6px;
  border: 1px solid var(--ant-border-strong);
  border-radius: var(--ant-radius-lg);
  color: var(--ant-text-2);
  font-size: var(--ant-fs-xs);
  font-weight: 400;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  vertical-align: middle;
}

.origin[data-origin='computed_by_system'] {
  border-style: dashed;
}

.origin[data-warn] {
  border-color: var(--ant-status-attention);
  color: var(--ant-status-attention-text);
}
</style>
