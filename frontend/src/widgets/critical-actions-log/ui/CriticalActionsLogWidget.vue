<script setup lang="ts">
/**
 * Виджет «Журнал критических действий» стола Аудитора ИБ (только чтение;
 * FR-79, FR-106; AD-28) — контейнер: записи на момент
 * (`security.critical_action.list`, фильтр по группе) и выбранная целиком
 * (`security.critical_action.read`). Срез: `group` — группа по умолчанию.
 */
import { computed, ref } from 'vue'
import { CA_GROUPS, useCriticalAction, useCriticalActions, type CriticalActionCaGroup } from '@/entities/integrity'
import { backendModeOf } from '@/shared/api/response'
import type { WidgetProps } from '@/shared/config/widget'
import { WidgetFrame } from '@/shared/ui'
import CriticalActionsLogView from './CriticalActionsLogView.vue'

const props = defineProps<WidgetProps>()
const isGroup = (v: unknown): v is CriticalActionCaGroup => typeof v === 'string' && (CA_GROUPS as readonly string[]).includes(v)
const group = ref<CriticalActionCaGroup | null>(isGroup(props.slice.group) ? props.slice.group : null)

const listQ = useCriticalActions(computed(() => (group.value ? { group: group.value } : {})))
const actions = computed(() => listQ.data.value?.data?.items ?? null)
const selected = ref<string | null>(null)
const detailQ = useCriticalAction(selected)
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="backendModeOf(listQ.data.value)"
    :loading="listQ.isPending.value && !actions"
    :error="actions ? undefined : listQ.error.value"
    :data-widget="widgetId"
  >
    <CriticalActionsLogView
      v-if="actions"
      v-model:group="group"
      :actions="actions"
      :selected="selected"
      :detail="detailQ.data.value?.data ?? null"
      :detail-error="detailQ.error.value ?? undefined"
      :density="density"
      @select="(id) => (selected = id)"
    />
  </WidgetFrame>
</template>
