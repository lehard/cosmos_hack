<script setup lang="ts">
/**
 * Пометка источника факта (FR-140): ручной ввод / импорт / станок / датчик /
 * камера / внешняя система / вывод системы / решение человека — и надёжность.
 * Ручная отметка выделена и никогда не выдаётся за данные станка.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { sourceMark } from '../model/passport'
import type { SourcedRecord } from '../model/types'

const props = defineProps<{ record: SourcedRecord }>()
const { t } = useI18n()
const mark = computed(() => sourceMark(props.record))
</script>

<template>
  <span class="source" :data-source="mark.code" :data-manual="mark.manual || undefined" :title="t('hints.sourceKind')">
    {{ t(mark.key) }}<template v-if="mark.reliabilityKey"> · {{ t(mark.reliabilityKey) }}</template>
  </span>
</template>

<style scoped>
.source {
  display: inline-block;
  padding: 0 6px;
  border: 1px solid var(--ant-border-strong);
  border-radius: var(--ant-radius-lg);
  color: var(--ant-text-2);
  font-size: var(--ant-fs-xs);
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  vertical-align: middle;
}

.source[data-manual] {
  border-style: dashed;
  border-color: var(--ant-status-neutral);
}
</style>
