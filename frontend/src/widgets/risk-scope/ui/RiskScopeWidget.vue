<script setup lang="ts">
/**
 * Виджет «Область риска» (FR-61, FR-62) — контейнер.
 *
 * Сужение и расширение — решения человека с основанием (`incident.scope.narrowed`
 * / `expanded`, группа критических действий `risk_scope`); пока их операций нет
 * в API, кнопки выключены (canAct=false).
 */
import { computed, inject } from 'vue'
import { routerKey } from 'vue-router'
import { scopeState } from '@/entities/incident'
import type { WidgetProps } from '@/shared/config/widget'
import { WidgetFrame } from '@/shared/ui'
import { useRiskScopeSource } from '../model/source'
import RiskScopeView from './RiskScopeView.vue'

defineProps<WidgetProps>()

// Роутер берём необязательно: виджет живёт и вне страницы (тесты столов).
const router = inject(routerKey, null)
const src = useRiskScopeSource()
const data = computed(() => src.data.value)
const state = computed(() => (data.value ? scopeState(data.value) : 'normal'))

/** Изделие → паспорт изделия (FR-7). */
const openItem = (id: string) => router?.push({ name: 'item', params: { id } })
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="src.mode.value"
    :state="state"
    :loading="src.isPending.value"
    :error="src.error.value"
    :empty="!data"
    empty-key="empty.noRiskScope"
    :data-widget="widgetId"
  >
    <RiskScopeView v-if="data" :model="data" :density="density" :can-act="false" @open-item="openItem" />
  </WidgetFrame>
</template>
