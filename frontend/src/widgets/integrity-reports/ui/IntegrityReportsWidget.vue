<script setup lang="ts">
/**
 * Виджет «Верификатор» стола Аудитора ИБ (только чтение; FR-106, FR-108;
 * AD-46) — контейнер: состояние целостности (`security.integrity.read`, тот же
 * кэш, что у индикатора шапки), отчёты верификатора
 * (`security.verifier_report.list`) и выбранный отчёт (`security.verifier_report.read`).
 * Индикатор желтеет сам без свежего отчёта (effectiveIntegrity).
 */
import { computed, ref, watch } from 'vue'
import { effectiveIntegrity, useIntegrity, useVerifierReport, useVerifierReports } from '@/entities/integrity'
import { backendModeOf } from '@/shared/api/response'
import type { WidgetProps } from '@/shared/config/widget'
import { WidgetFrame } from '@/shared/ui'
import IntegrityReportsView from './IntegrityReportsView.vue'

defineProps<WidgetProps>()

const integrityQ = useIntegrity()
const integrity = computed(() => integrityQ.data.value?.data)
const status = computed(() => (integrity.value ? effectiveIntegrity(integrity.value, Date.now()) : null))

const reportsQ = useVerifierReports()
const reports = computed(() => reportsQ.data.value?.data?.items ?? null)
const selected = ref<string | null>(null)
// Первый (последний по времени) отчёт открыт сразу.
watch(
  reports,
  (list) => {
    if (list?.length && !list.some((r) => r.report_digest === selected.value)) selected.value = list[0]!.report_digest
  },
  { immediate: true },
)
const reportQ = useVerifierReport(selected)

const state = computed(() => (status.value === 'violated' ? 'defect_indication' : status.value === 'stale' || status.value === 'unknown' ? 'unable_to_assess' : 'normal'))
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="backendModeOf(reportsQ.data.value ?? integrityQ.data.value)"
    :state="state"
    :loading="reportsQ.isPending.value && integrityQ.isPending.value"
    :error="reports || integrity ? undefined : (reportsQ.error.value ?? integrityQ.error.value)"
    :data-widget="widgetId"
  >
    <IntegrityReportsView
      :status="status"
      :checked-at="integrity?.checked_at ?? null"
      :reports="reports ?? []"
      :selected="selected"
      :report="reportQ.data.value?.data ?? null"
      :reports-error="reports ? undefined : (reportsQ.error.value ?? undefined)"
      :report-error="reportQ.error.value ?? undefined"
      :density="density"
      @select="(id) => (selected = id)"
    />
  </WidgetFrame>
</template>
