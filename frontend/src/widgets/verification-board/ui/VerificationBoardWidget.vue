<script setup lang="ts">
/**
 * Виджет «Табло проверки» — «ожидалось → получилось» (AD-26, кейс §5.1,
 * FR-129) — контейнер. Табло того же прогона, что на пульте (хранилище выбора
 * entities/run); срез стола `run_id` или адрес `?run=` закрепляют прогон.
 * Операция — `simulation.board.read` на момент (AD-37); пока прогон идёт,
 * табло перечитывается, на паузе — показывает состояние на момент паузы.
 */
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import { isActive, useBoard, useCurrentRunId, useRun, type BoardFilter } from '@/entities/run'
import { backendModeOf } from '@/shared/api/response'
import type { WidgetProps } from '@/shared/config/widget'
import { WidgetFrame } from '@/shared/ui'
import VerificationBoardView from './VerificationBoardView.vue'

const props = defineProps<WidgetProps>()
const route = useRoute()

const str = (v: unknown): string | undefined => (typeof v === 'string' && v ? v : undefined)
const fixedRun = computed(() => str(route?.query.run) ?? str(props.slice.run_id))
const { runId, runs } = useCurrentRunId(fixedRun)
const runQ = useRun(runId)
const live = computed(() => {
  const r = runQ.data.value?.data
  return !!r && isActive(r.state) && r.state !== 'paused'
})

const boardQ = useBoard(runId, live)
const board = computed(() => boardQ.data.value?.data ?? null)
const filter = ref<BoardFilter>(props.slice.filter === 'failed' || props.slice.filter === 'open' ? props.slice.filter : 'all')
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="backendModeOf(boardQ.data.value)"
    :loading="(!!runId && boardQ.isPending.value && !board) || (!runId && runs.isPending.value)"
    :error="board ? undefined : (boardQ.error.value ?? (runId ? undefined : runs.error.value))"
    :empty="(!runId && !runs.isPending.value) || (!!board && !board.rows.length)"
    :empty-key="runId ? 'empty.noRecords' : 'widgets.board.noRun'"
    :data-widget="widgetId"
  >
    <VerificationBoardView v-if="board" v-model:filter="filter" :board="board" :density="density" />
  </WidgetFrame>
</template>
