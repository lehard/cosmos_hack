<script setup lang="ts">
/**
 * Паспорт изделия (FR-42, FR-43, FR-45, FR-46, FR-140; кейс «история изделия»):
 * шапка с осями статуса, записи «кто · что · когда» с пометкой источника и
 * проверкой подписи, журнал изменений, генеалогия, зоны, носители, документы.
 *
 * «Одна правда — разные взгляды» (PRD §3a): тот же виджет на всех столах;
 * срез `view: compact` — правая панель контролёра и подписанта: шапка и
 * последние записи; `full` — страница паспорта.
 */
import { useI18n } from 'vue-i18n'
import { NAlert, NButton, NSpin, NTabPane, NTabs } from 'naive-ui'
import type { ItemGenealogy, ItemHistory, ItemPassport } from '@/entities/item'
import { naiveSizeOf, type Density } from '@/shared/config/widget'
import { useProblemText } from '@/shared/i18n/problem'
import PassportCarriers from './PassportCarriers.vue'
import PassportChanges from './PassportChanges.vue'
import PassportDocuments from './PassportDocuments.vue'
import PassportEntries from './PassportEntries.vue'
import PassportGenealogy from './PassportGenealogy.vue'
import PassportHeader from './PassportHeader.vue'
import PassportZones from './PassportZones.vue'

withDefaults(
  defineProps<{
    passport: ItemPassport
    view?: 'full' | 'compact'
    density?: Density
    /** Журнал изменений (FR-43); null — ещё не пришёл. */
    history?: ItemHistory | null
    historyLoading?: boolean
    historyError?: unknown
    /** Генеалогия (FR-45); null — ещё не пришла. */
    genealogy?: ItemGenealogy | null
    genealogyLoading?: boolean
    genealogyError?: unknown
    /** Есть экран карточки несоответствия. */
    canOpenNc?: boolean
    /** Есть экран инцидента. */
    canOpenIncident?: boolean
  }>(),
  {
    view: 'full',
    density: 'comfortable',
    history: null,
    historyLoading: false,
    historyError: undefined,
    genealogy: null,
    genealogyLoading: false,
    genealogyError: undefined,
    canOpenNc: false,
    canOpenIncident: false,
  },
)
const emit = defineEmits<{
  'open-item': [itemId: string]
  'open-nc': [ncId: string]
  'open-incident': [incidentId: string]
  /** Открыть полный паспорт (из компактного). */
  'open-full': []
}>()

const { t } = useI18n()
const problemText = useProblemText()

/** Сколько последних записей в компактном паспорте. */
const COMPACT_ENTRIES = 5
</script>

<template>
  <div class="passport" :class="`density-${density}`" :data-view="view" data-testid="item-passport">
    <PassportHeader
      :passport="passport"
      :can-open-nc="canOpenNc"
      :can-open-incident="canOpenIncident"
      @open-nc="(id) => emit('open-nc', id)"
      @open-incident="(id) => emit('open-incident', id)"
    />

    <template v-if="view === 'compact'">
      <PassportEntries :entries="passport.entries" :limit="COMPACT_ENTRIES" :filterable="false" />
      <NButton :size="naiveSizeOf(density)" secondary data-testid="open-full" @click="emit('open-full')">{{ t('common.actions.openPassport') }}</NButton>
    </template>

    <NTabs v-else type="line" default-value="entries" animated>
      <NTabPane name="entries" :tab="t('timeline.title')">
        <PassportEntries :entries="passport.entries" />
      </NTabPane>
      <NTabPane name="changes" :tab="t('widgets.passport.tabs.changes')">
        <NSpin v-if="historyLoading" size="small" />
        <NAlert v-else-if="historyError" type="error" :bordered="false" :show-icon="false">{{ problemText(historyError) }}</NAlert>
        <PassportChanges v-else :changes="history?.items ?? []" />
      </NTabPane>
      <NTabPane name="genealogy" :tab="t('passport.genealogy')">
        <NSpin v-if="genealogyLoading" size="small" />
        <NAlert v-else-if="genealogyError" type="error" :bordered="false" :show-icon="false">{{ problemText(genealogyError) }}</NAlert>
        <PassportGenealogy v-else-if="genealogy" :genealogy="genealogy" @open-item="(id) => emit('open-item', id)" />
      </NTabPane>
      <NTabPane name="zones" :tab="t('passport.zones')">
        <PassportZones :zones="passport.zones" />
      </NTabPane>
      <NTabPane name="carriers" :tab="t('widgets.passport.tabs.carriers')">
        <PassportCarriers :carriers="passport.carriers" />
      </NTabPane>
      <NTabPane name="documents" :tab="t('passport.documents')">
        <PassportDocuments :documents="passport.documents" />
      </NTabPane>
    </NTabs>
  </div>
</template>

<style scoped>
.passport {
  display: flex;
  flex-direction: column;
  gap: 10px;
  font-size: 13px;
}

.density-large {
  font-size: 16px;
}
</style>
