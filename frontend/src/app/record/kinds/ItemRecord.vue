<script setup lang="ts">
/**
 * Окно изделия (Д-70): паспорт изделия целиком — статусы, записи, генеалогия,
 * документы (виджет item-passport со срезом `item_id`). Отсюда же — переход на
 * страницу паспорта и в окна несоответствий изделия.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { SummaryTag, usePassport } from '@/entities/item'
import { useDrillDown } from '@/features/drill-down'
import { ActionButton, RecordDrawer } from '@/shared/ui'
import WidgetHost from '@/widgets/WidgetHost.vue'

const props = defineProps<{ id: string; show: boolean }>()
const emit = defineEmits<{ close: [] }>()
const { t } = useI18n()
const route = useRoute()
const drill = useDrillDown()

const runId = computed(() => (typeof route.query.run === 'string' && route.query.run ? route.query.run : undefined))
const opts = computed(() => (runId.value ? { run_id: runId.value } : {}))
const query = usePassport(() => props.id, opts)
const passport = computed(() => query.data.value?.data ?? null)

/** Страница паспорта: новый адрес без `open` — окно закрывается само. */
const openPage = () => drill.openPage({ entity: 'item', id: props.id })
</script>

<template>
  <RecordDrawer
    :show="show"
    :kind-label="t('common.words.item')"
    :number="passport?.label ?? id"
    data-record="item"
    @close="emit('close')"
  >
    <template v-if="passport" #status>
      <SummaryTag :code="passport.status.summary" />
    </template>
    <template #links>
      <ActionButton text type="primary" size="small" data-testid="open-item-page" :label="t('shell.record.openPage')" @click="openPage" />
    </template>

    <WidgetHost
      widget="item-passport"
      slot-id="record-passport"
      :slice="{ item_id: id, view: 'full', ...opts }"
      :frame="{ hideTitle: true, plain: true }"
    />
  </RecordDrawer>
</template>
