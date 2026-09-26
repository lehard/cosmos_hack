<script setup lang="ts">
/**
 * Уведомления (FR-57): число непрочитанных; по нажатию — виды (информация,
 * тревога, задача, запрос решения) и открытые задачи с подтверждением
 * (эпик 13). Полный список — виджет «Задачи» на столе роли.
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NBadge, NButton, NIcon, NPopover } from 'naive-ui'
import { Bell } from '@vicons/tabler'
import { useNotificationSummary } from '@/entities/notification'
import NotificationsPanel from './NotificationsPanel.vue'

const { t } = useI18n()
const summary = useNotificationSummary()
const unread = computed(() => summary.data.value?.data.unread ?? 0)
const show = ref(false)
</script>

<template>
  <NPopover v-model:show="show" trigger="click" placement="bottom-end">
    <template #trigger>
      <NBadge :value="unread" :max="99" :show="unread > 0">
        <NButton quaternary circle :aria-label="`${t('common.header.notifications')} · ${t('shell.header.unread', { n: unread })}`" data-testid="notifications-bell">
          <template #icon><NIcon><Bell /></NIcon></template>
        </NButton>
      </NBadge>
    </template>
    <NotificationsPanel :summary="summary.data.value?.data ?? null" @close="show = false" />
  </NPopover>
</template>
