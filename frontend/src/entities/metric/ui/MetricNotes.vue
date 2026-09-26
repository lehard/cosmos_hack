<script setup lang="ts">
/**
 * Значок подсказки у заголовка показателя: оговорки числа (определение,
 * смысл интервала, происхождение времени — кейс §5.2, FR-88) словами во
 * всплывающей подсказке, а не плашками рядом с числом. Нет оговорок — нет
 * значка. `warn` — происхождение времени не передано: значок требует внимания.
 */
import { NIcon, NTooltip } from 'naive-ui'
import { InfoCircle } from '@vicons/tabler'

withDefaults(defineProps<{ notes: readonly string[]; warn?: boolean }>(), { warn: false })
</script>

<template>
  <NTooltip v-if="notes.length" :style="{ maxWidth: 'var(--ant-w-side)' }">
    <template #trigger>
      <!-- Щелчок по значку — только подсказка, не раскрытие числа. -->
      <NIcon
        class="metric-notes"
        role="img"
        tabindex="0"
        :aria-label="notes.join('. ')"
        :data-warn="warn || undefined"
        data-testid="metric-notes"
        @click.stop
      >
        <InfoCircle />
      </NIcon>
    </template>
    <span v-for="(line, i) in notes" :key="i" class="note ant-wrap">{{ line }}</span>
  </NTooltip>
</template>

<style scoped>
.metric-notes {
  flex: none;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
  line-height: var(--ant-lh-tight);
  cursor: help;
}

.metric-notes[data-warn] {
  color: var(--ant-status-attention-text);
}

.note {
  display: block;
}

.note + .note {
  margin-top: var(--ant-space-1);
}
</style>
