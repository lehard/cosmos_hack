<script setup lang="ts">
/**
 * Статус паспорта допуска: допущен / приостановлен / выведен, стадия (тень,
 * пилот, работа) и уровень доверия 0–4 (AD-29). Тексты — statuses.analyzerPassport,
 * тон — токены статусов (приостановлен — danger, тень и пилот — attention).
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{ status: string; stage?: string | null; level?: number | null }>()
const { t, te } = useI18n()

const label = computed(() => {
  const key = props.status === 'active' && props.stage && props.stage !== 'active' ? props.stage : props.status
  const k = `statuses.analyzerPassport.${key === 'not_admitted' ? 'notAdmitted' : key}`
  return te(k) ? t(k) : t('widgets.visionAdaptation.notAdmitted')
})
const tone = computed(() => {
  if (props.status === 'suspended') return 'danger'
  if (props.status !== 'active') return 'muted'
  return props.stage === 'active' ? 'success' : 'attention'
})
</script>

<template>
  <span class="trust" :data-tone="tone" :title="label">
    <span class="dot" aria-hidden="true" />
    <span class="ant-ellipsis">{{ label }}</span>
    <span v-if="level !== undefined && level !== null && status === 'active'" class="level">{{ t('widgets.visionAdaptation.level', { n: level }) }}</span>
  </span>
</template>

<style scoped>
.trust {
  display: inline-flex;
  gap: var(--ant-space-1);
  align-items: center;
  max-width: 100%;
  padding: 0 var(--ant-space-2);
  border-radius: var(--ant-radius-pill);
  background: var(--ant-status-neutral-soft);
  font-size: var(--ant-fs-meta);
}

.dot {
  flex: none;
  width: var(--ant-space-2);
  height: var(--ant-space-2);
  border-radius: var(--ant-radius-pill);
  background: var(--ant-status-neutral);
}

.level {
  flex: none;
  color: var(--ant-text-2);
}

.trust[data-tone='danger'] {
  background: var(--ant-status-danger-soft);
}

.trust[data-tone='danger'] .dot {
  background: var(--ant-status-danger);
}

.trust[data-tone='success'] {
  background: var(--ant-status-success-soft);
}

.trust[data-tone='success'] .dot {
  background: var(--ant-status-success);
}

.trust[data-tone='attention'] {
  background: var(--ant-status-attention-soft);
}

.trust[data-tone='attention'] .dot {
  background: var(--ant-status-attention);
}

.trust[data-tone='muted'] .dot {
  background: var(--ant-status-muted);
}
</style>
