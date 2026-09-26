<script setup lang="ts">
/**
 * Виджет «Ждут моего решения» (PRD §3a, FR-55) — контейнер: очередь через
 * entities/nonconformity (`nonconformity.queue.list`, порядок считает сервер).
 * Щелчок по строке открывает запись в правом окне (Д-70, UI-7): строка с
 * несоответствием — окно несоответствия (карточка, паспорт, «Что решить»),
 * точка предъявления без несоответствия — окно изделия. Выделена строка,
 * открытая сейчас. Срез: `sort: [risk, deadline]`.
 */
import { computed, onBeforeUnmount, ref } from 'vue'
import { useRoute } from 'vue-router'
import { queueSortOf, rowKey, useDecisionQueue, type DecisionQueueRow, type QueueSort } from '@/entities/nonconformity'
import { useDrillDown } from '@/features/drill-down'
import { backendModeOf } from '@/shared/api/response'
import type { WidgetProps } from '@/shared/config/widget'
import { useMomentStore } from '@/shared/model/moment'
import { useRecordLink } from '@/shared/model/record'
import { WidgetFrame } from '@/shared/ui'
import DecisionQueueView from './DecisionQueueView.vue'

const props = defineProps<WidgetProps>()
const route = useRoute()
const drill = useDrillDown()
const record = useRecordLink()
const moment = useMomentStore()

const str = (v: unknown): string | undefined => (typeof v === 'string' && v ? v : undefined)
const sort = ref<QueueSort>(queueSortOf(props.slice.sort))
const runId = computed(() => str(route?.query.run) ?? str(props.slice.run_id))

const query = useDecisionQueue(sort, runId)
const rows = computed(() => query.data.value?.data?.items ?? null)

/** Что открыть по строке: несоответствие, если оно есть, иначе изделие. */
const refOf = (row: DecisionQueueRow) => (row.nc_id ? { entity: 'nonconformity' as const, id: row.nc_id } : { entity: 'item' as const, id: row.item_id })

/** Строка, открытая в окне сейчас. */
const selected = computed(() => {
  const open = record.current.value
  const row = open && rows.value?.find((r) => {
    const ref = refOf(r)
    return ref.entity === open.entity && ref.id === open.id
  })
  return row ? rowKey(row) : null
})

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
      :selected="selected"
      :now="now"
      :density="density"
      @select="(row) => drill.open(refOf(row))"
    />
  </WidgetFrame>
</template>
