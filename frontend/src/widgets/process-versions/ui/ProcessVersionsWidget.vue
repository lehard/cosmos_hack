<script setup lang="ts">
/**
 * Виджет «Процесс» (FR-24) — контейнер: версии процесса, читаемое
 * представление и отличия от действующей через `process.version.*`. Срез стола
 * `mode: diff` открывает сразу отличия (стол технолога).
 */
import { computed } from 'vue'
import type { WidgetProps } from '@/shared/config/widget'
import { WidgetFrame } from '@/shared/ui'
import { useProcessVersionsSource } from '../model/source'
import ProcessVersionsView from './ProcessVersionsView.vue'

const props = defineProps<WidgetProps>()

const src = useProcessVersionsSource()
const data = computed(() => src.versions.value)
const initialMode = computed(() => (props.slice.mode === 'diff' ? 'diff' : 'view'))
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="src.mode.value"
    :loading="src.isPending.value"
    :error="src.error.value"
    :empty="!data?.length"
    empty-key="empty.noRecords"
    :data-widget="widgetId"
  >
    <ProcessVersionsView
      v-if="data?.length"
      v-model:selected="src.selected.value"
      :versions="data"
      :diff="src.diff.value"
      :density="density"
      :initial-mode="initialMode"
    />
  </WidgetFrame>
</template>
