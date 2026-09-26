<script setup lang="ts">
/**
 * Виджет «Карточка несоответствия» (FR-51) — контейнер: данные через
 * entities/nonconformity (`nonconformity.card.read`) на момент из
 * useMomentStore, рамка WidgetFrame. Несоответствие — из среза (`nc_id`):
 * страница карточки или окно записи (Д-70), куда его открывает очередь.
 * Срок решения считается от «сейчас» сервера (Ant-Now): в воспроизведении — от момента.
 */
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { ncCardState, useNcCard } from '@/entities/nonconformity'
import { useDrillDown } from '@/features/drill-down'
import { backendModeOf } from '@/shared/api/response'
import type { WidgetProps } from '@/shared/config/widget'
import { useMomentStore } from '@/shared/model/moment'
import { useServerNow } from '@/shared/model/server-clock'
import { WidgetFrame } from '@/shared/ui'
import NcCardView from './NcCardView.vue'

const props = defineProps<WidgetProps>()
const route = useRoute()
const drill = useDrillDown()
const moment = useMomentStore()

const str = (v: unknown): string | undefined => (typeof v === 'string' && v ? v : undefined)
const ncId = computed(() => str(props.slice.nc_id) ?? null)
const runId = computed(() => str(route?.query.run) ?? str(props.slice.run_id))
const view = computed<'evidence' | 'full'>(() => (props.slice.view === 'evidence' ? 'evidence' : 'full'))

const query = useNcCard(ncId, runId)
const card = computed(() => query.data.value?.data ?? null)
const state = computed(() => (card.value ? ncCardState(card.value) : 'normal'))

// Обратный отсчёт срока (FR-55): раз в 30 с; в воспроизведении — момент воспроизведения.
// «Сейчас» — по часам сервера (Ant-Now); в воспроизведении — момент воспроизведения.
const serverNow = useServerNow()
const now = computed(() => (moment.asOf ? Date.parse(moment.asOf) : serverNow.value))
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="backendModeOf(query.data.value)"
    :state="state"
    :loading="!!ncId && query.isPending.value && !card"
    :error="card ? undefined : query.error.value"
    :empty="!ncId"
    empty-key="widgets.ncCard.noSelection"
    :data-widget="widgetId"
  >
    <NcCardView
      v-if="card"
      :card="card"
      :now="now"
      :density="density"
      :view="view"
      @open-item="(id) => drill.open({ entity: 'item', id })"
    />
  </WidgetFrame>
</template>
