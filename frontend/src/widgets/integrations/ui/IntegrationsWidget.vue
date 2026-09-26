<script setup lang="ts">
/**
 * Виджет «Интеграции» (FR-157, AD-47) — контейнер: `ops.integration.list`,
 * щелчок по строке открывает окно записи справа (Д-70). Кнопки «Включить»,
 * «Выключить», «Стенд ↔ реальная система», «Проверить соединение» и
 * переотправка из карантина — в окне записи.
 */
import { computed } from 'vue'
import { useIntegrations } from '@/entities/integration'
import { useDrillDown, type DrillRef } from '@/features/drill-down'
import { backendModeOf } from '@/shared/api/response'
import type { WidgetProps } from '@/shared/config/widget'
import { WidgetFrame } from '@/shared/ui'
import IntegrationsView from './IntegrationsView.vue'

defineProps<WidgetProps>()
const q = useIntegrations()
const list = computed(() => q.data.value?.data ?? null)
const drill = useDrillDown()

// Сбой канала или выключенная установленная система — не «норма».
const state = computed(() => (list.value?.items.some((e) => e.installed && (e.channel === 'degraded' || e.state === 'disabled')) ? 'defect_indication' : 'normal'))

/** Окно записи интеграции (тип записи вне EntityKind контракта — только оболочки). */
const open = (system: string) => drill.open({ entity: 'integration', id: system } as unknown as DrillRef)
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="backendModeOf(q.data.value)"
    :state="state"
    :loading="q.isPending.value && !list"
    :error="list ? undefined : q.error.value"
    :data-widget="widgetId"
  >
    <IntegrationsView v-if="list" :list="list" @open="open" />
  </WidgetFrame>
</template>
