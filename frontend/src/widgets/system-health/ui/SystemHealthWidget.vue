<script setup lang="ts">
/**
 * Виджет «Состояние компонентов» (FR-127, AD-45, AD-46) — контейнер: здоровье
 * (`ops.health.read`), метрики приёма (`ingest.metrics.read`, живые — эпик 06),
 * остановленные изделия (`ops.stopped_item.list`) и «повторить обработку»
 * (`ops.processing.retry`, command_id — UUIDv7, AD-7).
 */
import { computed, ref } from 'vue'
import { useOpsHealth, useRetryProcessing, useStoppedItems, type StoppedItem } from '@/entities/integrity'
import { useIngestMetrics } from '@/entities/quarantine'
import { useSession } from '@/entities/session'
import { backendModeOf } from '@/shared/api/response'
import type { WidgetProps } from '@/shared/config/widget'
import { newCommandId } from '@/shared/lib/command-id'
import { useMomentStore } from '@/shared/model/moment'
import { WidgetFrame } from '@/shared/ui'
import SystemHealthView from './SystemHealthView.vue'

defineProps<WidgetProps>()
const moment = useMomentStore()
const session = useSession()

const healthQ = useOpsHealth()
const health = computed(() => healthQ.data.value?.data ?? null)
const metricsQ = useIngestMetrics()
const stoppedQ = useStoppedItems()
const retry = useRetryProcessing()
const retried = ref<string | null>(null)

// Нарушение целостности или неработающий компонент — не «норма» (NFR-UI-4).
const state = computed(() => {
  const h = health.value
  if (!h) return 'normal'
  if (h.verifier?.verdict === 'violated' || h.components.some((c) => c.state === 'down')) return 'defect_indication'
  if (h.components.some((c) => c.state === 'unknown')) return 'unable_to_assess'
  return 'normal'
})

async function onRetry(item: StoppedItem): Promise<void> {
  const s = session.data.value?.data
  try {
    await retry.mutateAsync({
      itemId: item.item_id,
      body: {
        command_id: newCommandId(),
        basis_seq: item.failed_seq,
        policy_seq: s?.policy_seq ?? 0,
        failure_event_id: item.failure_event_id,
        ...(s?.workplace?.id ? { workplace_id: s.workplace.id } : {}),
      },
    })
    retried.value = item.item_id
  } catch {
    // Текст ошибки — из retry.error.
  }
}
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="backendModeOf(healthQ.data.value)"
    :state="state"
    :loading="healthQ.isPending.value && !health"
    :error="health ? undefined : healthQ.error.value"
    :data-widget="widgetId"
  >
    <SystemHealthView
      v-if="health"
      :health="health"
      :metrics="metricsQ.data.value?.data ?? null"
      :metrics-error="metricsQ.data.value ? undefined : (metricsQ.error.value ?? undefined)"
      :stopped="stoppedQ.data.value?.data?.items ?? null"
      :stopped-error="stoppedQ.data.value ? undefined : (stoppedQ.error.value ?? undefined)"
      :can-act="!moment.isReplay"
      :busy="retry.isPending.value"
      :error="retry.error.value ?? undefined"
      :retried="retried"
      :density="density"
      @retry="onRetry"
    />
  </WidgetFrame>
</template>
