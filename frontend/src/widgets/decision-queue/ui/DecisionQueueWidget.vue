<script setup lang="ts">
/**
 * Виджет «Ждут моего решения» (PRD §3a, FR-55) — контейнер: очередь через
 * entities/nonconformity (`nonconformity.queue.list`, порядок считает сервер),
 * выбор строки — в хранилище выбора стола (Pinia): карточка, панель решений и
 * паспорт на том же столе показывают выбранное. Срез: `sort: [risk, deadline]`.
 */
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { queueSortOf, useDecisionFocusStore, useDecisionQueue, type QueueSort } from '@/entities/nonconformity'
import { backendModeOf } from '@/shared/api/response'
import type { WidgetProps } from '@/shared/config/widget'
import { useMomentStore } from '@/shared/model/moment'
import { WidgetFrame } from '@/shared/ui'
import DecisionQueueView from './DecisionQueueView.vue'

const props = defineProps<WidgetProps>()
const route = useRoute()
const focus = useDecisionFocusStore()
const moment = useMomentStore()

const str = (v: unknown): string | undefined => (typeof v === 'string' && v ? v : undefined)
const sort = ref<QueueSort>(queueSortOf(props.slice.sort))
const runId = computed(() => str(route?.query.run) ?? str(props.slice.run_id))

const query = useDecisionQueue(sort, runId)
const rows = computed(() => query.data.value?.data?.items ?? null)

// Ничего не выбрано — берём первую строку, чтобы карточка не была пустой.
watch(
  rows,
  (list) => {
    if (list?.length && !focus.rowId) focus.select(list[0]!)
  },
  { immediate: true },
)

const tick = ref(Date.now())
const timer = setInterval(() => (tick.value = Date.now()), 30_000)
onBeforeUnmount(() => clearInterval(timer))
const now = computed(() => (moment.asOf ? Date.parse(moment.asOf) : tick.value))
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="backendModeOf(query.data.value)"
    :loading="query.isPending.value && !rows"
    :error="rows ? undefined : query.error.value"
    :empty="!!rows && !rows.length"
    empty-key="empty.queueEmpty"
    :data-widget="widgetId"
  >
    <DecisionQueueView
      v-if="rows"
      v-model:sort="sort"
      :rows="rows"
      :selected="focus.rowId"
      :now="now"
      :density="density"
      @select="(row) => focus.select(row)"
    />
  </WidgetFrame>
</template>
