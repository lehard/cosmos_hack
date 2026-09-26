<script setup lang="ts">
/**
 * Оболочка после входа: шапка, слева меню разделов стола (Д-73), страница.
 * Здесь же живут канал живых обновлений (SSE, AD-21) и синхронизация прав
 * @casl/vue с сервером (AD-15), и правое окно записи (Д-70): любая страница
 * открывает запись по `?open=‹тип›:‹id›`.
 */
import { computed, onBeforeUnmount, onMounted, provide, shallowRef, watch } from 'vue'
import { NLayout, NLayoutContent, NLayoutHeader } from 'naive-ui'
import { useQueryClient } from '@tanstack/vue-query'
import { startLiveUpdates, type LiveUpdates } from '@/shared/api/sse'
import { SSE_ADDRESS } from '@/shared/api/generated/stream'
import { RECORD_DRAWER } from '@/shared/model/record'
import { useAbilitySync } from '../providers/access'
import { RECORD_KINDS } from '../record/registry'
import RecordDrawerHost from '../record/RecordDrawerHost.vue'
import AppHeader from './AppHeader.vue'
import DeskNav from './DeskNav.vue'
import { activeRun, refreshActiveRun } from '@/shared/api/active-run'

const queryClient = useQueryClient()
useAbilitySync()
provide(RECORD_DRAWER, { kinds: RECORD_KINDS })

const updates = shallowRef<LiveUpdates | null>(null)
const live = computed(() => updates.value?.status.value ?? 'connecting')
let runTimer: ReturnType<typeof setInterval> | undefined
onMounted(() => {
  updates.value = startLiveUpdates(queryClient)
  void refreshActiveRun()
  runTimer = setInterval(() => void refreshActiveRun(), 3000)
})
onBeforeUnmount(() => {
  updates.value?.stop()
  clearInterval(runTimer)
})
/** Идёт прогон сценария — столы работают в нём: живые обновления переподключаются (SSE фильтрует по run_id), данные перечитываются. */
watch(
  () => activeRun.value?.run_id ?? null,
  (runId) => {
    updates.value?.stop()
    const url = runId ? `${SSE_ADDRESS}${SSE_ADDRESS.includes('?') ? '&' : '?'}run_id=${encodeURIComponent(runId)}` : SSE_ADDRESS
    updates.value = startLiveUpdates(queryClient, url)
    void queryClient.invalidateQueries()
  },
)
</script>

<template>
  <NLayout class="shell">
    <NLayoutHeader bordered class="shell-header">
      <AppHeader :live="live" />
    </NLayoutHeader>
    <NLayoutContent class="shell-content" content-class="shell-body">
      <DeskNav class="shell-nav" />
      <main class="shell-page">
        <RouterView />
      </main>
    </NLayoutContent>
    <RecordDrawerHost />
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

.shell-content :deep(.shell-body) {
  display: flex;
  height: 100%;
}

/* Меню не прокручивается вместе со страницей — всегда на виду. */
.shell-nav {
  flex: none;
}

.shell-page {
  flex: 1 1 auto;
  min-width: 0;
  overflow: auto;
  padding: var(--ant-space-5) var(--ant-space-6) var(--ant-space-8);
}
</style>
