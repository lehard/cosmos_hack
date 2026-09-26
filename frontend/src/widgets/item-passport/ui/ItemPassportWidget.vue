<script setup lang="ts">
/**
 * Виджет «Паспорт изделия» (FR-42, FR-43, FR-45, FR-46, FR-140) — контейнер:
 * данные через entities/item (`item.passport.read`, `item.history.list`,
 * `item.genealogy.read`) на момент из useMomentStore, рамка WidgetFrame.
 *
 * Изделие — из среза стола (`item_id`, страница паспорта) или из выбора в
 * очереди «Ждут моего решения» (стол контролёра). Срез `view: compact` —
 * правая панель; `full` — страница. Состояние рамки — по оси качества:
 * «заблокировано» не делает паспорт «дефектным» (NFR-UI-4).
 */
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { qualityState, useItemGenealogy, useItemHistory, usePassport } from '@/entities/item'
import { useDecisionFocusStore } from '@/entities/nonconformity'
import { useDrillDown } from '@/features/drill-down'
import { backendModeOf } from '@/shared/api/response'
import type { WidgetProps } from '@/shared/config/widget'
import { WidgetFrame } from '@/shared/ui'
import ItemPassportView from './ItemPassportView.vue'

const props = defineProps<WidgetProps>()
const route = useRoute()
const drill = useDrillDown()
const focus = useDecisionFocusStore()

const str = (v: unknown): string | undefined => (typeof v === 'string' && v ? v : undefined)

const itemId = computed(() => str(props.slice.item_id) ?? focus.itemId ?? null)
const view = computed<'full' | 'compact'>(() => (props.slice.view === 'compact' ? 'compact' : 'full'))
const opts = computed(() => {
  const run = str(route?.query.run) ?? str(props.slice.run_id)
  return run ? { run_id: run } : {}
})

const passportQ = usePassport(itemId, opts)
const historyQ = useItemHistory(itemId, () => ({ ...opts.value, enabled: view.value === 'full' }))
const genealogyQ = useItemGenealogy(itemId, () => ({ ...opts.value, enabled: view.value === 'full' }))

const passport = computed(() => passportQ.data.value?.data ?? null)
const state = computed(() => (passport.value ? qualityState(passport.value.status) : 'normal'))
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="backendModeOf(passportQ.data.value)"
    :state="state"
    :loading="!!itemId && passportQ.isPending.value && !passport"
    :error="passport ? undefined : passportQ.error.value"
    :empty="!itemId"
    empty-key="widgets.passport.noItem"
    :data-widget="widgetId"
  >
    <ItemPassportView
      v-if="passport"
      :passport="passport"
      :view="view"
      :density="density"
      :history="historyQ.data.value?.data ?? null"
      :history-loading="historyQ.isPending.value && historyQ.fetchStatus.value !== 'idle'"
      :history-error="historyQ.error.value"
      :genealogy="genealogyQ.data.value?.data ?? null"
      :genealogy-loading="genealogyQ.isPending.value && genealogyQ.fetchStatus.value !== 'idle'"
      :genealogy-error="genealogyQ.error.value"
      :can-open-nc="drill.canOpen({ entity: 'nonconformity', id: 'x' })"
      :can-open-incident="drill.canOpen({ entity: 'incident', id: 'x' })"
      @open-item="(id) => drill.open({ entity: 'item', id })"
      @open-nc="(id) => drill.open({ entity: 'nonconformity', id })"
      @open-incident="(id) => drill.open({ entity: 'incident', id })"
      @open-full="drill.open({ entity: 'item', id: passport.item_id })"
    />
  </WidgetFrame>
</template>
