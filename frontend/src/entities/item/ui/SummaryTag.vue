<script setup lang="ts">
/**
 * Сводный статус изделия (словарь `item_summary`, AD-30) — для списков и
 * генеалогии. Код вне словаря показывается как UNKNOWN(код).
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { statusDictionaries, statusPalette, type StatusTone } from '@/shared/api/generated/statuses'
import { codeToKey } from '@/shared/i18n'

const props = defineProps<{ code: string }>()
const { t, te } = useI18n()

const entry = computed(() => (statusDictionaries.item_summary.values as Record<string, { label: string; tone: StatusTone } | undefined>)[props.code])
const label = computed(() => {
  const key = `statuses.itemSummary.${codeToKey(props.code)}`
  return te(key) ? t(key) : (entry.value?.label ?? `UNKNOWN(${props.code})`)
})
</script>

<template>
  <span class="summary-tag" :data-code="code" :title="label">
    <span class="dot" :style="{ background: statusPalette[entry?.tone ?? 'neutral'] }" aria-hidden="true" />
    <span class="text">{{ label }}</span>
  </span>
</template>

<style scoped>
.summary-tag {
  display: inline-flex;
  gap: 6px;
  align-items: center;
  max-width: 100%;
  min-width: 0;
  white-space: nowrap;
}

.text {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}

.dot {
  flex: none;
  width: 8px;
  height: 8px;
  border-radius: 50%;
}
</style>
