<script setup lang="ts">
// Заготовка виджета «metric-drilldown» (эпик 03). Эпик 15 заменяет её содержимым,
// оставаясь в рамке WidgetFrame (четыре состояния, момент, метка fixtures | live).
import type { WidgetProps } from '@/shared/config/widget'
import { WidgetStub } from '@/shared/ui'

const props = defineProps<WidgetProps>()
</script>

<template>
  <WidgetStub v-bind="props" :epic="15" />
</template>
