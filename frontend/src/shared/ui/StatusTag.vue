<script setup lang="ts">
/**
 * Статус изделия по словарю статусов (AD-30, PRD §3b, NFR-UI-4): текст — только
 * своим ключом `statuses.‹ось›.‹код›`, цвет — тон из контракта. Код, которого нет
 * в словаре, показывается как UNKNOWN(код), а не подменяется похожим статусом.
 *
 * Вид (UI-3): мягкая плашка тона статуса с цветной точкой; длинная подпись —
 * многоточие, полный текст — в подсказке. Когда брать: любой статус из словаря.
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

const tone = computed<StatusTone>(() => entry.value?.tone ?? 'neutral')
const color = computed(() => statusPalette[tone.value])
</script>

<template>
  <span class="status-tag" :data-axis="axis" :data-code="code" :data-tone="tone" :data-unknown="entry ? undefined : 'true'" :title="label">
    <span class="dot" :style="{ background: color }" aria-hidden="true" />
    <span class="text ant-ellipsis">{{ label }}</span>
  </span>
</template>

<style scoped>
.status-tag {
  display: inline-flex;
  gap: 6px;
  align-items: center;
  max-width: 100%;
  min-width: 0;
  padding: 1px 8px 1px 6px;
  border-radius: var(--ant-radius-pill);
  background: var(--ant-n-100);
  color: var(--ant-n-800);
  font-size: var(--ant-fs-meta);
  line-height: 18px;
  vertical-align: middle;
}

.status-tag[data-tone='info'] {
  background: var(--ant-status-info-soft);
}

.status-tag[data-tone='attention'] {
  background: var(--ant-status-attention-soft);
}

.status-tag[data-tone='danger'] {
  background: var(--ant-status-danger-soft);
}

.status-tag[data-tone='critical'] {
  background: var(--ant-status-critical-soft);
}

.status-tag[data-tone='success'] {
  background: var(--ant-status-success-soft);
}

.status-tag[data-tone='qualified'] {
  background: var(--ant-status-qualified-soft);
}

.dot {
  flex: none;
  width: 7px;
  height: 7px;
  border-radius: 50%;
}
</style>
