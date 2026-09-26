<script setup lang="ts">
/**
 * Текущее выполнение операции на посту (PRD §3a «длительность операции против
 * нормы», FR-121): профиль выполнения (`machinelogs.run_profile.read`) по
 * `current_run_id` оборудования поста; идёт с записи «операция начата» —
 * минуты против нормы шага из схемы процесса. Сведений нет — «неизвестно».
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NTag } from 'naive-ui'
import { useRunProfile } from '@/entities/equipment'
import { elapsedMinutes, normText, overNorm, type ProcessStep } from '@/entities/live-map'
import { formatMinutes } from '@/shared/lib/duration'
import { ActionButton } from '@/shared/ui'

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
  <div v-if="run" class="run" :data-run="runId" :data-over="over || undefined">
    <span class="ant-wrap">{{ step?.name ?? run.step_key }}</span>
    <span v-if="minutes !== null" :class="{ over }">{{ t('widgets.shopFloor.station.runFor', { time: formatMinutes(t, minutes) }) }}</span>
    <span class="ant-muted">{{ normText(t, step?.norm) }}</span>
    <NTag v-if="over" size="small" type="error" :bordered="false">{{ t('widgets.shopFloor.station.overNorm') }}</NTag>
    <ActionButton v-if="run.item_id" text type="primary" size="small" :label="run.item_id" :hint="t('common.actions.openPassport')" @click="emit('item', run.item_id)" />
  </div>
  <span v-else-if="profile.error.value" class="ant-muted">{{ t('empty.noDataUnknown') }}</span>
</template>

<style scoped>
.run {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-1) var(--ant-space-2);
  align-items: center;
  min-width: 0;
}

.over {
  color: var(--ant-status-danger-text);
}
</style>
