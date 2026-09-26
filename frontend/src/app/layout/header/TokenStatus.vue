<script setup lang="ts">
/** Статус токена на рабочем месте (AD-14): порт shared/lib/token-agent. */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NIcon, NTooltip } from 'naive-ui'
import { Key } from '@vicons/tabler'
import { useTokenStatus } from '@/shared/lib/token-agent'

const { t } = useI18n()
const status = useTokenStatus()

const text = computed(() =>
  status.value === 'inserted'
    ? t('common.header.tokenInserted')
    : status.value === 'missing'
      ? t('common.header.tokenMissing')
      : t('common.header.tokenAgentMissing'),
)
</script>

<template>
  <NTooltip>
    <template #trigger>
      <span class="token" :data-status="status" data-testid="token-status">
        <NIcon :depth="status === 'inserted' ? 1 : 3"><Key /></NIcon>
        {{ t('common.header.tokenStatus') }}
      </span>
    </template>
    {{ text }}
  </NTooltip>
</template>

<style scoped>
.token {
  display: inline-flex;
  gap: 4px;
  align-items: center;
  font-size: var(--ant-fs-sm);
  white-space: nowrap;
}
</style>
