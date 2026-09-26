<script setup lang="ts">
/**
 * Виджет «Общие факторы» (FR-135) — контейнер: что общего у несоответствий
 * группы в разборе (группа — из фокуса разбора, иначе самая крупная).
 */
import { computed } from 'vue'
import { factorsState } from '@/entities/incident'
import type { WidgetProps } from '@/shared/config/widget'
import { WidgetFrame } from '@/shared/ui'
import { useCommonFactorsSource } from '../model/source'
import CommonFactorsView from './CommonFactorsView.vue'

defineProps<WidgetProps>()

const src = useCommonFactorsSource()
const data = computed(() => src.data.value)
const state = computed(() => (data.value ? factorsState(data.value) : 'normal'))
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="src.mode.value"
    :state="state"
    :loading="src.isPending.value"
    :error="src.error.value"
    :empty="!data"
    empty-key="widgets.analysis.factors.noGroup"
    :data-widget="widgetId"
  >
    <CommonFactorsView v-if="data" :model="data" :density="density" />
  </WidgetFrame>
</template>
