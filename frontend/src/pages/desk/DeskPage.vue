<script setup lang="ts">
/**
 * Стол роли (AD-21, NFR-EXT-1): всё, что на странице, пришло данными
 * normative/desks/‹роль›.yaml через access.desk.read. Кода под конкретную роль
 * здесь нет — новый стол собирается правкой yaml.
 * Вид — общие элементы shared/ui: заголовок стола (AppPage) или вкладки.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { NAlert, NSpin, NTabPane, NTabs } from 'naive-ui'
import { AppPage, EmptyState } from '@/shared/ui'
import { useDesk } from '@/entities/desk'
import { useProblemText } from '@/shared/i18n/problem'
import DeskTabView from './ui/DeskTabView.vue'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const problemText = useProblemText()
const desk = useDesk()

const d = computed(() => desk.data.value?.data)
const active = computed(() => {
  const tabs = d.value?.tabs ?? []
  const wanted = typeof route.params.tab === 'string' ? route.params.tab : ''
  return tabs.find((tab) => tab.id === wanted) ?? tabs[0]
})

const select = (id: string) => router.replace({ name: 'desk', params: { tab: id } })
</script>

<template>
  <div v-if="desk.isPending.value" class="center"><NSpin /></div>
  <NAlert v-else-if="desk.isError.value" type="error" :bordered="false">{{ problemText(desk.error.value) }}</NAlert>
  <section v-else-if="d && active" class="desk" :data-role="d.role" :data-density="d.density">
    <NTabs v-if="d.tabs.length > 1" type="line" size="large" class="desk-tabs" :value="active.id" @update:value="select">
      <NTabPane v-for="tab in d.tabs" :key="tab.id" :name="tab.id" :tab="t(tab.title_key)" display-directive="if">
        <DeskTabView :tab="tab" :density="d.density" />
      </NTabPane>
    </NTabs>
    <AppPage v-else :title="t(active.title_key)">
      <DeskTabView :tab="active" :density="d.density" />
    </AppPage>
  </section>
  <EmptyState v-else :title="t('shell.desk.empty')" />
</template>

<style scoped>
.center {
  display: flex;
  justify-content: center;
  padding: var(--ant-space-12) 0;
}

.desk-tabs :deep(.n-tabs-nav) {
  margin-bottom: var(--ant-space-5);
}

.desk-tabs :deep(.n-tabs-tab) {
  max-width: 320px;
}

.desk-tabs :deep(.n-tabs-tab__label) {
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
</style>
