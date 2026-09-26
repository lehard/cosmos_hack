<script setup lang="ts">
/**
 * Стол роли (AD-21, NFR-EXT-1): всё, что на странице, пришло данными
 * normative/desks/‹роль›.yaml через access.desk.read. Кода под конкретную роль
 * здесь нет — новый стол собирается правкой yaml.
 * Разделы переключает левое меню оболочки (Д-73, app/layout/DeskNav); своего
 * заголовка у раздела нет — название выделено в меню, заголовки несут рамки
 * виджетов (Д-77, UI-13). Раздел с виджетом «на всю высоту» (живая карта) —
 * страница без прокрутки.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { NAlert, NSpin } from 'naive-ui'
import { EmptyState } from '@/shared/ui'
import { useDesk } from '@/entities/desk'
import { fillsSection } from '@/widgets/registry'
import { useProblemText } from '@/shared/i18n/problem'
import DeskTabView from './ui/DeskTabView.vue'

const { t } = useI18n()
const route = useRoute()
const problemText = useProblemText()
const desk = useDesk()

const d = computed(() => desk.data.value?.data)
/** Раздел с виджетом «на всю высоту» (живая карта) — страница без прокрутки. */
const fill = computed(() => active.value?.layout === 'single' && active.value.slots.some((s) => fillsSection(s.widget)))
const active = computed(() => {
  const tabs = d.value?.tabs ?? []
  const wanted = typeof route.params.tab === 'string' ? route.params.tab : ''
  return tabs.find((tab) => tab.id === wanted) ?? tabs[0]
})
</script>

<template>
  <div v-if="desk.isPending.value" class="center"><NSpin /></div>
  <NAlert v-else-if="desk.isError.value" type="error" :bordered="false">{{ problemText(desk.error.value) }}</NAlert>
  <section v-else-if="d && active" class="desk" :class="{ 'desk--fill': fill }" :data-role="d.role" :data-density="d.density">
    <DeskTabView :key="active.id" :tab="active" :density="d.density" :data-tab="active.id" />
  </section>
  <EmptyState v-else :title="t('shell.desk.empty')" />
</template>

<style scoped>
.desk--fill {
  height: 100%;
}

.center {
  display: flex;
  justify-content: center;
  padding: var(--ant-space-12) 0;
}
</style>
