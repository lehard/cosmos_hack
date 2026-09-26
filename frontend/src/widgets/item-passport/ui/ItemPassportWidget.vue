<script setup lang="ts">
/**
 * Виджет «Паспорт изделия» (FR-42, FR-43, FR-45, FR-46, FR-140) — контейнер:
 * данные через entities/item (`item.passport.read`, `item.history.list`,
 * `item.genealogy.read`) на момент из useMomentStore, рамка WidgetFrame.
 *
 * Изделие — из среза (`item_id`): страница паспорта или окно записи (Д-70).
 * Срез `view: compact` — узкая панель; `full` — страница и окно. Состояние рамки — по оси качества:
 * «заблокировано» не делает паспорт «дефектным» (NFR-UI-4).
 */
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { qualityState, useItemGenealogy, useItemHistory, usePassport } from '@/entities/item'
import { useDrillDown } from '@/features/drill-down'
import { backendModeOf } from '@/shared/api/response'
import type { WidgetProps } from '@/shared/config/widget'
import { WidgetFrame } from '@/shared/ui'
import ItemPassportView from './ItemPassportView.vue'

const props = defineProps<WidgetProps>()
const route = useRoute()
const drill = useDrillDown()

const str = (v: unknown): string | undefined => (typeof v === 'string' && v ? v : undefined)

const itemId = computed(() => str(props.slice.item_id) ?? null)
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

// Ошибка чтения паспорта держится до первых данных: перечитывание по SSE во
// время прогона сбрасывает ошибку запроса без данных обратно в «загрузку», и
// окно изделия при 404 крутилось бы бесконечно вместо понятной ошибки.
const lastError = ref<unknown>(null)
watch(
  () => passportQ.error.value,
  (e) => {
    if (e) lastError.value = e
  },
)
watch([passport, itemId], ([p], [, prevId]) => {
  if (p || itemId.value !== prevId) lastError.value = null
})
const passportError = computed(() => (passport.value ? undefined : (passportQ.error.value ?? lastError.value ?? undefined)))
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="backendModeOf(passportQ.data.value)"
    :state="state"
    :loading="!!itemId && passportQ.isPending.value && !passport && !passportError"
    :error="passportError"
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
      @open-full="drill.openPage({ entity: 'item', id: passport.item_id })"
    />
  </WidgetFrame>
</template>
