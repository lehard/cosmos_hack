<script setup lang="ts">
/**
 * Виджет «Предложения» — каркас страницы (FR-63; наполнение — эпик 42). Читает
 * только уже существующий вход генератора «ограничение линии»
 * (`analytics.node_counters.read`); списка предложений в контракте v1 нет.
 */
import { computed } from 'vue'
import { useNodeCounterSet } from '@/entities/metric'
import type { WidgetProps } from '@/shared/config/widget'
import { WidgetFrame } from '@/shared/ui'
import ProposalsView from './ProposalsView.vue'

defineProps<WidgetProps>()
const nodes = useNodeCounterSet()
const bottleneck = computed(() => nodes.data.value?.bottleneck ?? null)
</script>

<template>
  <WidgetFrame :title-key="titleKey" :density="density" :mode="nodes.mode.value" :loading="nodes.isPending.value" :data-widget="widgetId">
    <ProposalsView :bottleneck="bottleneck" />
  </WidgetFrame>
</template>
