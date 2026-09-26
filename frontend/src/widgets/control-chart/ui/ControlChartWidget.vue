<script setup lang="ts">
/**
 * Виджет «Контрольные карты» — контейнер: выбор узла процесса (узлы и аномалии —
 * `analytics.node_counters.read`), карта узла — `analytics.control_chart.read`.
 * Узел по умолчанию: из среза стола (`step_key`), иначе узел с аномалией
 * «доля дефектов вне контрольных границ», иначе ограничение линии, иначе первый.
 * Точка вне границ → состояние «признак дефекта» (FR-5), а не «брак».
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NSelect } from 'naive-ui'
import { useDrillDown } from '@/features/drill-down'
import { PeriodPicker, useControlChart, useNodeCounterSet } from '@/entities/metric'
import { naiveSizeOf, type WidgetDataState, type WidgetProps } from '@/shared/config/widget'
import { WidgetFrame } from '@/shared/ui'
import ControlChartView from './ControlChartView.vue'

const props = defineProps<WidgetProps>()
const { t } = useI18n()
const drill = useDrillDown()
const nodes = useNodeCounterSet()

const sliceStep = typeof props.slice.step_key === 'string' ? props.slice.step_key : null
const sliceMetric = typeof props.slice.metric_id === 'string' ? props.slice.metric_id : undefined
const chosen = ref<string | null>(null)

const steps = computed(() => (nodes.data.value?.counters ?? []).map((c) => c.step_key))
const defaultStep = computed(() => {
  if (sliceStep) return sliceStep
  const set = nodes.data.value
  if (!set) return null
  return set.anomalies.find((a) => a.kind === 'defect_rate_out_of_control')?.step_key ?? set.bottleneck?.step_key ?? steps.value[0] ?? null
})
const step = computed(() => chosen.value ?? defaultStep.value)
const options = computed(() => {
  const keys = steps.value.includes(step.value ?? '') || !step.value ? steps.value : [step.value, ...steps.value]
  return keys.map((k) => ({ label: k, value: k }))
})

const src = useControlChart(step, sliceMetric)
const chart = computed(() => src.data.value)
const state = computed<WidgetDataState>(() => (chart.value?.points.some((p) => p.out_of_control) ? 'defect_indication' : 'normal'))
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="src.mode.value ?? nodes.mode.value"
    :state="state"
    :loading="nodes.isPending.value || (!!step && src.isPending.value)"
    :error="src.error.value ?? (step ? undefined : nodes.error.value)"
    :empty="!step"
    :data-widget="widgetId"
  >
    <div class="head">
      <NSelect
        class="step"
        :size="naiveSizeOf(density)"
        :value="step"
        :options="options"
        :aria-label="t('analytics.slices.operation')"
        data-testid="step-pick"
        @update:value="(v: string) => (chosen = v)"
      />
      <PeriodPicker :size="naiveSizeOf(density)" />
    </div>
    <p class="hint">{{ t('widgets.analytics.chart.hint') }}</p>
    <ControlChartView v-if="chart" :chart="chart" @open="(r) => drill.open(r)" />
  </WidgetFrame>
</template>

<style scoped>
.head {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  margin-bottom: 4px;
}

.step {
  min-width: 220px;
  max-width: 320px;
}

.hint {
  margin: 0 0 8px;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}
</style>
