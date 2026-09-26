<script setup lang="ts">
/**
 * Заготовка виджета: оболочка (эпик 03) заранее кладёт в реестр все виджеты
 * PRD §3a, а фронт-эпики 10–15 заменяют заготовку содержимым, не трогая
 * оболочку. Показывает рамку, слот, плотность и срез из yaml стола — так видно,
 * что стол собран из данных.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { WidgetProps } from '@/shared/config/widget'
import EmptyState from './EmptyState.vue'
import WidgetFrame from './WidgetFrame.vue'

const props = defineProps<WidgetProps & { /** Эпик, который наполняет виджет. */ epic: number }>()
const { t } = useI18n()

const slice = computed(() => (Object.keys(props.slice).length ? JSON.stringify(props.slice) : null))
</script>

<template>
  <WidgetFrame :title-key="titleKey" :density="density" :data-widget="widgetId" data-stub="true">
    <EmptyState :title="t('shell.stub.title')" :description="t('shell.stub.body', { epic, slot: slotId, density: t(`shell.density.${density}`) })">
      <code v-if="slice" class="slice ant-wrap">{{ t('shell.stub.slice', { slice }) }}</code>
    </EmptyState>
  </WidgetFrame>
</template>

<style scoped>
.slice {
  display: block;
  max-width: 60ch;
  padding: var(--ant-space-1) var(--ant-space-2);
  border-radius: var(--ant-radius-sm);
  background: var(--ant-n-100);
  color: var(--ant-text-2);
  font-size: var(--ant-fs-xs);
}
</style>
