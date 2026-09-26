<script setup lang="ts">
/**
 * Стол роли (AD-21, NFR-EXT-1): всё, что на странице, пришло данными
 * normative/desks/‹роль›.yaml через access.desk.read. Кода под конкретную роль
 * здесь нет — новый стол собирается правкой yaml.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { NAlert, NSpin, NTabPane, NTabs } from 'naive-ui'
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
  <NSpin v-if="desk.isPending.value" class="center" />
  <NAlert v-else-if="desk.isError.value" type="error" :bordered="false">{{ problemText(desk.error.value) }}</NAlert>
  <section v-else-if="d && active" class="desk" :data-role="d.role" :data-density="d.density">
    <NTabs v-if="d.tabs.length > 1" type="line" :value="active.id" @update:value="select">
      <NTabPane v-for="tab in d.tabs" :key="tab.id" :name="tab.id" :tab="t(tab.title_key)" display-directive="if">
        <DeskTabView :tab="tab" :density="d.density" />
      </NTabPane>
    </NTabs>
    <template v-else>
      <h2 class="title">{{ t(active.title_key) }}</h2>
      <DeskTabView :tab="active" :density="d.density" />
    </template>
  </section>
  <NAlert v-else type="default" :bordered="false">{{ t('shell.desk.empty') }}</NAlert>
</template>

<style scoped>
.title {
  margin: 0 0 16px;
  font-size: 18px;
}

.center {
  display: flex;
  justify-content: center;
  padding: 48px 0;
}
</style>
