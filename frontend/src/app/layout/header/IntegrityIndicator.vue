<script setup lang="ts">
/**
 * Индикатор целостности журнала «по данным сервера» (AD-46, FR-73). Первичный
 * вердикт — отчёт верификатора у хранителя; здесь — то, что сообщил ant, с явной
 * пометкой. Нет свежего отчёта дольше двух интервалов — желтеет сам.
 * Потеря живых обновлений (SSE) — левой половиной той же точки (UI-25): одна
 * точка, в подсказке обе строки.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NTooltip } from 'naive-ui'
import { effectiveIntegrity, useIntegrity } from '@/entities/integrity'
import { statusPalette, type StatusTone } from '@/shared/api/generated/statuses'
import { useServerNow } from '@/shared/model/server-clock'

const props = withDefaults(defineProps<{ liveOff?: boolean }>(), { liveOff: false })
const { t, d } = useI18n()
const integrity = useIntegrity()
// Свежесть — от «сейчас» сервера (Ant-Now), не от часов браузера.
const now = useServerNow()

const report = computed(() => integrity.data.value?.data)
const status = computed(() => (integrity.isError.value ? 'unknown' : effectiveIntegrity(report.value, now.value)))

const TONE: Record<string, StatusTone> = { ok: 'success', violated: 'danger', stale: 'attention', unknown: 'neutral' }

/** Точка: целостность; нет живых обновлений — левая половина «внимание». */
const dotStyle = computed(() => {
  const own = statusPalette[TONE[status.value] ?? 'neutral']
  return props.liveOff ? { background: `linear-gradient(90deg, ${statusPalette.attention} 50%, ${own} 50%)` } : { background: own }
})

const text = computed(() => {
  switch (status.value) {
    case 'ok':
      return t('common.header.integrityOk', { time: report.value?.checked_at ? d(new Date(report.value.checked_at), 'time') : t('common.words.unknown') })
    case 'violated':
      return t('common.header.integrityViolated')
    case 'stale':
      return t('common.header.integrityStale')
    default:
      return t('common.words.unknown')
  }
})
</script>

<template>
  <NTooltip>
    <template #trigger>
      <span class="integrity" tabindex="0" :aria-label="t('common.header.integrity')" :data-status="status" :data-live-off="liveOff || undefined" data-testid="integrity">
        <span class="dot" :style="dotStyle" />
      </span>
    </template>
    <strong>{{ t('common.header.integrity') }}</strong><br />
    <span class="ant-wrap">{{ text }} · {{ t('common.header.integrityServerSide') }}</span>
    <template v-if="liveOff"><br /><span class="ant-wrap">{{ t('shell.header.liveOff') }}</span></template>
  </NTooltip>
</template>

<style scoped>
.integrity {
  display: inline-flex;
  gap: 6px;
  align-items: center;
  white-space: nowrap;
  padding: 6px;
  font-size: var(--ant-fs-sm);
  cursor: help;
}

.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}
</style>
