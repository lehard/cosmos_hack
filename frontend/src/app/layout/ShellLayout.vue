<script setup lang="ts">
/**
 * Оболочка после входа: шапка + страница. Здесь же живут канал живых обновлений
 * (SSE, AD-21) и синхронизация прав @casl/vue с сервером (AD-15).
 */
import { computed, onBeforeUnmount, onMounted, shallowRef } from 'vue'
import { NLayout, NLayoutContent, NLayoutHeader } from 'naive-ui'
import { useQueryClient } from '@tanstack/vue-query'
import { startLiveUpdates, type LiveUpdates } from '@/shared/api/sse'
import { useAbilitySync } from '../providers/access'
import AppHeader from './AppHeader.vue'

const queryClient = useQueryClient()
useAbilitySync()

const updates = shallowRef<LiveUpdates | null>(null)
const live = computed(() => updates.value?.status.value ?? 'connecting')
onMounted(() => {
  updates.value = startLiveUpdates(queryClient)
})
onBeforeUnmount(() => updates.value?.stop())
</script>

<template>
  <NLayout class="shell">
    <NLayoutHeader bordered class="shell-header">
      <AppHeader :live="live" />
    </NLayoutHeader>
    <NLayoutContent class="shell-content" content-class="shell-page">
      <RouterView />
    </NLayoutContent>
  </NLayout>
</template>

<style scoped>
.shell {
  height: 100%;
}

.shell-header {
  height: var(--ant-w-header);
}

.shell-content {
  height: calc(100% - var(--ant-w-header));
}

.shell-content :deep(.shell-page) {
  min-width: 0;
  padding: var(--ant-space-5) var(--ant-space-6) var(--ant-space-8);
}
</style>
