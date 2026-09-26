<script setup lang="ts">
/**
 * Текущее выполнение операции на посту (PRD §3a «длительность операции против
 * нормы», FR-121): профиль выполнения (`machinelogs.run_profile.read`) по
 * `current_run_id` оборудования поста; идёт с записи «операция начата» —
 * минуты против нормы шага из схемы процесса. Сведений нет — «неизвестно».
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NFlex, NTag, NText } from 'naive-ui'
import { useRunProfile } from '@/entities/equipment'
import { elapsedMinutes, normText, overNorm, type ProcessStep } from '@/entities/live-map'
import { formatMinutes } from '@/shared/lib/duration'

const props = defineProps<{
  runId: string
  steps: ReadonlyMap<string, ProcessStep>
  /** Момент просмотра. */
  at: Date
}>()
const emit = defineEmits<{ item: [itemId: string] }>()
const { t } = useI18n()
const profile = useRunProfile(() => props.runId)
const run = computed(() => profile.data.value?.data ?? null)
const step = computed(() => (run.value ? (props.steps.get(run.value.step_key) ?? null) : null))
const minutes = computed(() => (run.value && !run.value.finished_at ? elapsedMinutes(run.value.started_at, props.at) : null))
const over = computed(() => (step.value ? overNorm(minutes.value, step.value.norm) : null))
</script>

<template>
  <NFlex v-if="run" :size="6" align="center" :wrap="true" :data-run="runId" :data-over="over || undefined">
    <NText>{{ step?.name ?? run.step_key }}</NText>
    <NText v-if="minutes !== null" :type="over ? 'error' : undefined">
      {{ t('widgets.shopFloor.station.runFor', { time: formatMinutes(t, minutes) }) }}
    </NText>
    <NText depth="3">{{ step ? normText(t, step.norm) : t('widgets.shopFloor.station.noNorm') }}</NText>
    <NTag v-if="over" size="small" type="error" :bordered="false">{{ t('widgets.shopFloor.station.overNorm') }}</NTag>
    <NButton v-if="run.item_id" text type="primary" size="small" @click="emit('item', run.item_id)">{{ run.item_id }}</NButton>
  </NFlex>
  <NText v-else-if="profile.error.value" depth="3">{{ t('empty.noDataUnknown') }}</NText>
</template>
