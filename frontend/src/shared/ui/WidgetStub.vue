<script setup lang="ts">
/**
 * Заготовка виджета: оболочка (эпик 03) заранее кладёт в реестр все виджеты
 * PRD §3a, а фронт-эпики 10–15 заменяют заготовку содержимым, не трогая
 * оболочку. Показывает рамку, слот, плотность и срез из yaml стола — так видно,
 * что стол собран из данных.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NEmpty, NText } from 'naive-ui'
import type { WidgetProps } from '@/shared/config/widget'
import WidgetFrame from './WidgetFrame.vue'

const props = defineProps<WidgetProps & { /** Эпик, который наполняет виджет. */ epic: number }>()
const { t } = useI18n()

const slice = computed(() => (Object.keys(props.slice).length ? JSON.stringify(props.slice) : null))
</script>

<template>
  <WidgetFrame :title-key="titleKey" :density="density" :data-widget="widgetId" data-stub="true">
    <NEmpty :description="t('shell.stub.title')">
      <template #extra>
        <NText depth="3">{{ t('shell.stub.body', { epic, slot: slotId, density: t(`shell.density.${density}`) }) }}</NText>
        <br />
        <NText v-if="slice" depth="3" code>{{ t('shell.stub.slice', { slice }) }}</NText>
      </template>
    </NEmpty>
  </WidgetFrame>
</template>
