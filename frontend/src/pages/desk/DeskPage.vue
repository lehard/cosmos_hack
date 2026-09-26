<script setup lang="ts">
/**
 * Стол роли (AD-21, NFR-EXT-1): всё, что на странице, пришло данными
 * normative/desks/‹роль›.yaml через access.desk.read. Кода под конкретную роль
 * здесь нет — новый стол собирается правкой yaml.
 * Вид — общие элементы shared/ui: заголовок — название раздела (AppPage);
 * разделы переключает левое меню оболочки (Д-73, app/layout/DeskNav).
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { NAlert, NSpin } from 'naive-ui'
import { AppPage, EmptyState } from '@/shared/ui'
import { useDesk } from '@/entities/desk'
import { useProblemText } from '@/shared/i18n/problem'
import DeskTabView from './ui/DeskTabView.vue'

const { t } = useI18n()
const route = useRoute()
const problemText = useProblemText()
const desk = useDesk()

const d = computed(() => desk.data.value?.data)
const active = computed(() => {
  const tabs = d.value?.tabs ?? []
  const wanted = typeof route.params.tab === 'string' ? route.params.tab : ''
  return tabs.find((tab) => tab.id === wanted) ?? tabs[0]
})
</script>

<template>
  <div v-if="desk.isPending.value" class="center"><NSpin /></div>
  <NAlert v-else-if="desk.isError.value" type="error" :bordered="false">{{ problemText(desk.error.value) }}</NAlert>
  <section v-else-if="d && active" class="desk" :data-role="d.role" :data-density="d.density">
    <AppPage :title="t(active.title_key)" :data-tab="active.id">
      <DeskTabView :key="active.id" :tab="active" :density="d.density" />
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
</style>
