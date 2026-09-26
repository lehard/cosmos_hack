<script setup lang="ts">
/**
 * Статус изделия по словарю статусов (AD-30, PRD §3b, NFR-UI-4): текст — только
 * своим ключом `statuses.‹ось›.‹код›`, цвет — тон из контракта. Код, которого нет
 * в словаре, показывается как UNKNOWN(код), а не подменяется похожим статусом.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { statusAxes, statusPalette, type StatusAxis, type StatusTone } from '@/shared/api/generated/statuses'
import { codeToKey } from '@/shared/i18n'

const props = defineProps<{
  /** Ось статуса (position, quality, …). */
  axis: StatusAxis
  /** Код значения из контракта. */
  code: string
}>()

const { t, te } = useI18n()

const entry = computed(() => (statusAxes[props.axis].values as Record<string, { label: string; tone: StatusTone } | undefined>)[props.code])

const label = computed(() => {
  const key = `statuses.${codeToKey(props.axis)}.${codeToKey(props.code)}`
  if (te(key)) return t(key)
  return entry.value?.label ?? `UNKNOWN(${props.code})`
})

const color = computed(() => statusPalette[entry.value?.tone ?? 'neutral'])
</script>

<template>
  <span class="status-tag" :data-axis="axis" :data-code="code" :data-unknown="entry ? undefined : 'true'">
    <span class="dot" :style="{ background: color }" aria-hidden="true" />
    {{ label }}
  </span>
</template>

<style scoped>
.status-tag {
  display: inline-flex;
  gap: 6px;
  align-items: center;
  white-space: nowrap;
}

.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex: none;
}
</style>
