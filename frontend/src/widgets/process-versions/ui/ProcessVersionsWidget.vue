<script setup lang="ts">
/**
 * Виджет «Процесс» (FR-24) — контейнер: версии процесса, читаемое
 * представление и разница с действующей. Срез стола `mode: diff` открывает
 * сразу отличия (стол технолога).
 */
import { computed, ref } from 'vue'
import type { WidgetProps } from '@/shared/config/widget'
import { WidgetFrame } from '@/shared/ui'
import { useProcessVersionsSource } from '../model/source'
import ProcessVersionsView from './ProcessVersionsView.vue'

const props = defineProps<WidgetProps>()

const src = useProcessVersionsSource()
const data = computed(() => src.data.value)
const selected = ref<string | null>(null)
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
    <ProcessVersionsView v-if="data?.length" v-model:selected="selected" :versions="data" :density="density" :initial-mode="initialMode" />
  </WidgetFrame>
</template>
