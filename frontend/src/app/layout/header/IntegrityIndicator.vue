<script setup lang="ts">
/**
 * Индикатор целостности журнала «по данным сервера» (AD-46, FR-73). Первичный
 * вердикт — отчёт верификатора у хранителя; здесь — то, что сообщил ant, с явной
 * пометкой. Нет свежего отчёта дольше двух интервалов — желтеет сам.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NTooltip } from 'naive-ui'
import { effectiveIntegrity, useIntegrity } from '@/entities/integrity'
import { statusPalette, type StatusTone } from '@/shared/api/generated/statuses'

const { t, d } = useI18n()
const integrity = useIntegrity()

const report = computed(() => integrity.data.value?.data)
const status = computed(() => (integrity.isError.value ? 'unknown' : effectiveIntegrity(report.value, Date.now())))

const TONE: Record<string, StatusTone> = { ok: 'success', violated: 'danger', stale: 'attention', unknown: 'neutral' }

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
      <span class="integrity" :data-status="status" data-testid="integrity">
        <span class="dot" :style="{ background: statusPalette[TONE[status] ?? 'neutral'] }" />
        {{ t('common.header.integrity') }}
      </span>
    </template>
    {{ text }} · {{ t('common.header.integrityServerSide') }}
  </NTooltip>
</template>

<style scoped>
.integrity {
  display: inline-flex;
  gap: 6px;
  align-items: center;
  white-space: nowrap;
  font-size: 13px;
}

.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}
</style>
