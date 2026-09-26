<script setup lang="ts">
/**
 * Виджет «Карантин сообщений» (FR-41; кейс §5.1 S12) — контейнер: записи
 * (`ingest.quarantine.list`, живые — эпик 06), выбранная целиком
 * (`ingest.quarantine.read`), переобработка (`ingest.message.reprocess`,
 * command_id — UUIDv7: повтор не примет сообщение дважды, AD-7).
 * Срез: `view: summary` — компактный список без содержимого и команд.
 */
import { computed, ref, watch } from 'vue'
import { useIngestCommand, useQuarantine, useQuarantineEntry, type QuarantineEntry } from '@/entities/quarantine'
import { useSession } from '@/entities/session'
import { backendModeOf } from '@/shared/api/response'
import type { WidgetProps } from '@/shared/config/widget'
import { newCommandId } from '@/shared/lib/command-id'
import { useMomentStore } from '@/shared/model/moment'
import { WidgetFrame } from '@/shared/ui'
import QuarantineView from './QuarantineView.vue'

const props = defineProps<WidgetProps>()
const moment = useMomentStore()
const session = useSession()

const summary = computed(() => props.slice.view === 'summary')
const filter = ref<'open' | 'all'>('open')
const listQ = useQuarantine(computed(() => (filter.value === 'open' ? { state: 'open' as const } : {})))
const entries = computed(() => listQ.data.value?.data?.items ?? null)

const selected = ref<string | null>(null)
watch(entries, (list) => {
  if (selected.value && !list?.some((e) => e.quarantine_id === selected.value)) selected.value = null
})
const detailQ = useQuarantineEntry(computed(() => (summary.value ? null : selected.value)))
const command = useIngestCommand()
const done = ref<string | null>(null)

async function onReprocess(entry: QuarantineEntry, reason: string, discard: boolean): Promise<void> {
  const s = session.data.value?.data
  try {
    await command.mutateAsync({
      kind: 'reprocess',
      quarantineId: entry.quarantine_id,
      body: {
        command_id: newCommandId(),
        // Запись карантина не несёт basis_seq (AD-39): сервер сверяет по текущему состоянию.
        basis_seq: 0,
        policy_seq: s?.policy_seq ?? 0,
        reason: { text: reason },
        ...(discard ? { discard: true } : {}),
        ...(s?.workplace?.id ? { workplace_id: s.workplace.id } : {}),
      },
    })
    done.value = entry.quarantine_id
  } catch {
    // Текст ошибки — из command.error.
  }
}
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="backendModeOf(listQ.data.value)"
    :loading="listQ.isPending.value && !entries"
    :error="entries ? undefined : listQ.error.value"
    :empty="!!entries && !entries.length && summary"
    empty-key="empty.quarantineEmpty"
    :data-widget="widgetId"
  >
    <QuarantineView
      v-if="entries"
      v-model:filter="filter"
      :entries="entries"
      :selected="selected"
      :detail="detailQ.data.value?.data ?? null"
      :detail-error="detailQ.error.value ?? undefined"
      :summary="summary"
      :can-act="!moment.isReplay"
      :busy="command.isPending.value"
      :error="command.error.value ?? undefined"
      :done="done"
      :density="density"
      @select="(id) => (selected = id)"
      @reprocess="onReprocess"
    />
  </WidgetFrame>
</template>
