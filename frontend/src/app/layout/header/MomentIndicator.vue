<script setup lang="ts">
/**
 * Момент просмотра (AD-21, Д-70, UI-4): в обычной работе ничего не показывает —
 * «сейчас» не нужно подписывать. Только при просмотре прошлого: «На момент:
 * ‹дата›» (в подсказке — как было / что мы знали) и кнопка «Вернуться к текущему».
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NTooltip } from 'naive-ui'
import { useMomentStore } from '@/shared/model/moment'

const { t, d } = useI18n()
const moment = useMomentStore()

const label = computed(() => (moment.asOf ? t('shell.header.pastView', { time: d(new Date(moment.asOf), 'dateTime') }) : ''))
const axis = computed(() => t(moment.axis === 'recorded' ? 'common.modes.asOfRecorded' : 'common.modes.asOfOccurred'))
</script>

<template>
  <span v-if="moment.asOf" class="moment" data-testid="moment" :data-axis="moment.axis">
    <NTooltip>
      <template #trigger>
        <span class="past ant-ellipsis">{{ label }}</span>
      </template>
      <span class="ant-wrap">{{ axis }} · {{ t('shell.header.pastViewHint') }}</span>
    </NTooltip>
    <NButton size="small" type="primary" secondary data-testid="go-live" @click="moment.goLive()">
      <span class="ant-ellipsis">{{ t('shell.header.goLive') }}</span>
    </NButton>
  </span>
</template>

<style scoped>
.moment {
  display: inline-flex;
  flex: 0 1 auto;
  gap: var(--ant-space-2);
  align-items: center;
  min-width: 0;
}

.past {
  padding: 2px 8px;
  border-radius: var(--ant-radius-sm);
  background: var(--ant-status-attention-soft);
  color: var(--ant-text);
  font-weight: var(--ant-fw-bold);
}
</style>
