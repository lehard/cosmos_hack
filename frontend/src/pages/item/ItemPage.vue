<script setup lang="ts">
/**
 * Паспорт изделия на всю страницу (общий экран PRD §3a) — тот же виджет
 * item-passport, что на столах, со срезом «полный».
 */
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import WidgetHost from '@/widgets/WidgetHost.vue'

const route = useRoute()
const slice = computed(() => ({ item_id: String(route.params.id ?? ''), view: 'full' }))
</script>

<template>
  <WidgetHost widget="item-passport" slot-id="page" :slice="slice" density="comfortable" />
</template>
