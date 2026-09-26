<script setup lang="ts">
/**
 * Виджет «Источники» (FR-30, FR-33) — контейнер: источники событий
 * (`ingest.source.list`), каналы обмена (`erp.channel.list`), отключение и
 * включение источника с основанием (`ops.source.disable|enable`, command_id —
 * UUIDv7, AD-7). В воспроизведении команды выключены (FR-4).
 */
import { computed, ref } from 'vue'
import { useErpChannels } from '@/entities/erp-message'
import { useIngestCommand, useSources, type SourceView } from '@/entities/quarantine'
import { useSession } from '@/entities/session'
import { backendModeOf } from '@/shared/api/response'
import type { WidgetProps } from '@/shared/config/widget'
import { newCommandId } from '@/shared/lib/command-id'
import { useMomentStore } from '@/shared/model/moment'
import { WidgetFrame } from '@/shared/ui'
import SourcesView from './SourcesView.vue'

defineProps<WidgetProps>()
const moment = useMomentStore()
const session = useSession()

const sourcesQ = useSources()
const sources = computed(() => sourcesQ.data.value?.data?.items ?? null)
const channelsQ = useErpChannels()
const command = useIngestCommand()
const done = ref<string | null>(null)

// Подозрение на потерю или незарегистрированный ключ — не «норма».
const state = computed(() => (sources.value?.some((s) => s.state === 'loss_suspected' || s.state === 'unknown_key' || s.gap_count > 0) ? 'defect_indication' : 'normal'))

async function onSwitch(source: SourceView, to: 'disable' | 'enable', reason: string): Promise<void> {
  const s = session.data.value?.data
  done.value = null
  try {
    await command.mutateAsync({
      kind: to,
      sourceId: source.source_id,
      body: {
        command_id: newCommandId(),
        // Список источников не несёт basis_seq (AD-39) — сверка состояния на сервере по текущему.
        basis_seq: 0,
        policy_seq: s?.policy_seq ?? 0,
        reason: { text: reason },
        ...(s?.workplace?.id ? { workplace_id: s.workplace.id } : {}),
      },
    })
    done.value = source.source_id
  } catch {
    // Текст ошибки — из command.error.
  }
}
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="backendModeOf(sourcesQ.data.value)"
    :state="state"
    :loading="sourcesQ.isPending.value && !sources"
    :error="sources ? undefined : sourcesQ.error.value"
    :data-widget="widgetId"
  >
    <SourcesView
      v-if="sources"
      :sources="sources"
      :channels="channelsQ.data.value?.data?.items ?? null"
      :channels-error="channelsQ.data.value ? undefined : (channelsQ.error.value ?? undefined)"
      :can-act="!moment.isReplay"
      :busy="command.isPending.value"
      :error="command.error.value ?? undefined"
      :done="done"
      :density="density"
      @switch="onSwitch"
    />
  </WidgetFrame>
</template>
