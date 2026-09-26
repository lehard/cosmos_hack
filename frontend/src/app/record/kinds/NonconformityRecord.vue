<script setup lang="ts">
/**
 * Окно несоответствия (Д-70, UI-7, UI-8): заголовок — «Несоответствие НС-…»,
 * изделие и статус; вкладки «Карточка» и «Паспорт изделия»; внизу — панель
 * «Что решить» с кнопками решения, которые не уезжают при прокрутке.
 * Содержимое — виджеты nc-card, item-passport и decision-panel со срезом
 * `nc_id` / `item_id` (их логика не меняется).
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { NC_STATUS_TEXT, codeText, useNcCard } from '@/entities/nonconformity'
import { RecordDrawer, StatusTag, type RecordDrawerTab } from '@/shared/ui'
import WidgetHost from '@/widgets/WidgetHost.vue'

const props = defineProps<{ id: string; show: boolean }>()
const emit = defineEmits<{ close: [] }>()
const { t } = useI18n()
const route = useRoute()

const runId = computed(() => (typeof route.query.run === 'string' && route.query.run ? route.query.run : undefined))
const query = useNcCard(() => props.id, runId)
const card = computed(() => query.data.value?.data ?? null)

const tab = ref('card')
watch(
  () => props.id,
  () => (tab.value = 'card'),
)
const tabs = computed<RecordDrawerTab[]>(() => [
  { id: 'card', label: t('desks.ncCard') },
  ...(card.value ? [{ id: 'passport', label: t('desks.passport') }] : []),
])
const subtitle = computed(() => (card.value ? `${t('common.words.item')} ${card.value.item_label}` : ''))
const slice = computed(() => ({ nc_id: props.id, ...(runId.value ? { run_id: runId.value } : {}) }))
</script>

<template>
  <RecordDrawer
    v-model:tab="tab"
    :show="show"
    :kind-label="t('common.words.nonconformity')"
    :number="card?.number ?? id"
    :subtitle="subtitle"
    :tabs="tabs"
    data-record="nonconformity"
    @close="emit('close')"
  >
    <template v-if="card" #status>
      <StatusTag axis="quality" :code="card.axes.quality" />
      <span class="ant-ellipsis">{{ codeText(NC_STATUS_TEXT, card.status, t) }}</span>
    </template>

    <WidgetHost
      v-if="tab === 'card' || !card"
      widget="nc-card"
      slot-id="record-card"
      :slice="{ ...slice, view: 'full' }"
      :frame="{ hideTitle: true, plain: true }"
    />
    <WidgetHost
      v-else
      widget="item-passport"
      slot-id="record-passport"
      :slice="{ item_id: card.item_id, view: 'full', ...(runId ? { run_id: runId } : {}) }"
      :frame="{ hideTitle: true, plain: true }"
    />

    <template #actions>
      <WidgetHost widget="decision-panel" slot-id="record-decision" :slice="slice" :frame="{ plain: true }" />
    </template>
  </RecordDrawer>
</template>
