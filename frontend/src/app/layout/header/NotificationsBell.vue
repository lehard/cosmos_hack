<script setup lang="ts">
/** Уведомления (FR-57): число непрочитанных; список — виджет tasks (эпик 13). */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NBadge, NButton, NIcon, NTooltip } from 'naive-ui'
import { Bell } from '@vicons/tabler'
import { useNotificationSummary } from '@/entities/notification'

const { t } = useI18n()
const summary = useNotificationSummary()
const unread = computed(() => summary.data.value?.data.unread ?? 0)
</script>

<template>
  <NTooltip>
    <template #trigger>
      <NBadge :value="unread" :max="99" :show="unread > 0">
        <NButton quaternary circle :aria-label="t('common.header.notifications')">
          <template #icon><NIcon><Bell /></NIcon></template>
        </NButton>
      </NBadge>
    </template>
    {{ t('common.header.notifications') }} · {{ t('shell.header.unread', { n: unread }) }}
  </NTooltip>
</template>
