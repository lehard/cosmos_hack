<script setup lang="ts">
/**
 * Три исхода наблюдения (кейс §1.4, §4.5): признаков нет · признак обнаружен ·
 * оценка невозможна; «оценка невозможна» никогда не «годно».
 */
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const OUTCOMES = [
  { key: 'ok', mark: '✓', text: 'inspection.outcomeShort.noDefectFound' },
  { key: 'danger', mark: '!', text: 'inspection.outcomeShort.defectFound' },
  { key: 'warn', mark: '?', text: 'inspection.outcomeShort.unableToAssess' },
] as const
</script>

<template>
  <p class="legend ant-wrap" data-testid="outcome-legend">
    <span v-for="o in OUTCOMES" :key="o.key" class="item" :data-tone="o.key"><span class="mark" aria-hidden="true">{{ o.mark }}</span>{{ t(o.text) }}</span>
    <span class="rule">{{ t('ncCard.outcomes.rule') }}</span>
  </p>
</template>

<style scoped>
.legend {
  display: flex;
  flex-wrap: wrap;
  gap: 2px var(--ant-space-3);
  align-items: center;
  margin: 0;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-xs);
}

.item {
  display: inline-flex;
  gap: 4px;
  align-items: center;
}

.mark {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 14px;
  height: 14px;
  border-radius: 50%;
  background: var(--ant-status-neutral);
  color: var(--ant-surface);
  font-weight: var(--ant-fw-bold);
  font-size: 10px;
}

.item[data-tone='ok'] .mark {
  background: var(--ant-status-success);
}

.item[data-tone='danger'] .mark {
  background: var(--ant-status-danger);
}

.item[data-tone='warn'] .mark {
  background: var(--ant-status-attention);
}

.rule {
  color: var(--ant-status-attention-text);
  font-weight: var(--ant-fw-bold);
}
</style>
