<script setup lang="ts">
/**
 * Виджет «Посты» (FR-6) — контейнер: читает посты через entities/workplace на
 * момент из useMomentStore; вынутый ключ приходит событием SSE `workplace` и
 * перечитывает панель (проверка FR-6: не позднее 2 с). Срез: `workshop` — цех,
 * `run_id` — прогон сценария.
 */
import { computed } from 'vue'
import { usePosts } from '@/entities/workplace'
import { useDrillDown } from '@/features/drill-down'
import { backendModeOf } from '@/shared/api/response'
import type { WidgetProps } from '@/shared/config/widget'
import { WidgetFrame } from '@/shared/ui'
import { postsState } from '../model/presence'
import PostsTable from './PostsTable.vue'

const props = defineProps<WidgetProps>()
const drill = useDrillDown()

const str = (v: unknown) => (typeof v === 'string' && v ? v : undefined)
const query = usePosts(
  computed(() => {
    const p: { workshop?: string; run_id?: string } = {}
    const workshop = str(props.slice.workshop)
    const run = str(props.slice.run_id)
    if (workshop) p.workshop = workshop
    if (run) p.run_id = run
    return p
  }),
)
const rows = computed(() => query.data.value?.data ?? null)
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="backendModeOf(query.data.value)"
    :state="postsState(rows)"
    :loading="query.isPending.value && !rows"
    :error="rows ? undefined : query.error.value"
    :empty="!!rows && !rows.length"
    :data-widget="widgetId"
  >
    <PostsTable v-if="rows" :rows="rows" @open-item="(id) => drill.open({ entity: 'item', id })" />
  </WidgetFrame>
</template>
