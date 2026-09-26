<script setup lang="ts">
/**
 * Число показателя: только значение в своих единицах, без плашек рядом.
 * Оговорки времени (происхождение — кейс §5.2, FR-88; смысл интервала) — в
 * подсказке: по умолчанию во всплывающем `title` самого числа; `show-origin=false`
 * — когда оговорки уже в подсказке значка у заголовка показателя (MetricNotes),
 * чтобы не повторять их у каждой строки. При подсказке у самого числа
 * происхождение ещё и в атрибутах `data-origin` / `data-meaning` (для проверок).
 * «Оценка невозможна» (`unknown`) показывается словами и прочерком —
 * это не ноль и не «норма» (NFR-UI-4).
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { formatValue, meaningOf, originOf, valueNotes, type MetricValue } from '../model/value'

const props = withDefaults(defineProps<{ value: MetricValue; unknown?: boolean; showOrigin?: boolean }>(), { unknown: false, showOrigin: true })
const { t, n } = useI18n()
const text = computed(() => formatValue({ t, n: (v, f) => n(v, f) }, props.value))
const origin = computed(() => originOf(props.value))
const meaning = computed(() => meaningOf(props.value))
/** Подсказка числа: значение и оговорки; без оговорок — не ставим. */
const hint = computed(() => {
  if (!props.showOrigin) return undefined
  const notes = valueNotes(t, props.value)
  return notes.length ? [props.unknown ? t('widgets.analytics.value.unknown') : text.value, ...notes].join('\n') : undefined
})
</script>

<template>
  <span
    class="metric-number"
    :data-unknown="unknown || undefined"
    :data-origin="showOrigin ? origin?.code : undefined"
    :data-warn="(showOrigin && origin?.warn) || undefined"
    :data-meaning="showOrigin ? meaning?.code : undefined"
    :title="hint"
  >
    <template v-if="unknown">
      <span class="dash" aria-hidden="true">—</span>
      <span class="unknown" data-testid="unknown">{{ t('widgets.analytics.value.unknown') }}</span>
    </template>
    <span v-else class="value" data-testid="value">{{ text }}</span>
  </span>
</template>

<style scoped>
.metric-number {
  display: inline-flex;
  flex-wrap: wrap;
  gap: var(--ant-space-1);
  align-items: baseline;
  min-width: 0;
}

.value {
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.unknown {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
  font-weight: 400;
}
</style>
