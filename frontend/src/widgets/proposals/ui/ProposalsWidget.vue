<script setup lang="ts">
/**
 * Страница «Предложения» (эпик 42; FR-63, UJ-1): предложения генераторов —
 * ограничение линии, область риска, кандидаты в правила реакции, адаптация
 * VisionQC, цифровизация по карте дефицита. Стол показывает список; щелчок по
 * строке открывает предложение в правом окне, кнопки решения — в его нижней
 * панели (Д-70). «Сформировать предложения» прогоняет подключённые генераторы;
 * ничего не применяется автоматически.
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NAlert } from 'naive-ui'
import { useSession } from '@/entities/session'
import { useOpenRecord, useSuggestionCommand, useSuggestions } from '@/entities/suggestion'
import { backendModeOf } from '@/shared/api/response'
import { naiveSizeOf, type WidgetProps } from '@/shared/config/widget'
import { useProblemText } from '@/shared/i18n/problem'
import { useMomentStore } from '@/shared/model/moment'
import { ActionButton, DataTable, EmptyState, SectionPanel, WidgetFrame } from '@/shared/ui'
import SuggestionDrawer from './SuggestionDrawer.vue'

const props = defineProps<WidgetProps>()
const { t, d } = useI18n()
const problemText = useProblemText()
const moment = useMomentStore()
const session = useSession()

const q = useSuggestions()
const list = computed(() => q.data.value?.data ?? null)
const items = computed(() => list.value?.items ?? [])

const { open, setOpen } = useOpenRecord(['suggestion'] as const)
const current = computed(() => (open.value ? (items.value.find((s) => s.suggestion_id === open.value?.id) ?? null) : null))

const command = useSuggestionCommand(() => {
  const s = session.data.value?.data
  return { policySeq: s?.policy_seq ?? 0, workplaceId: s?.workplace?.id }
})
const generated = ref<number | null>(null)
async function generate(): Promise<void> {
  try {
    const r = await command.mutateAsync({ kind: 'generate' })
    generated.value = (r.data as { event_ids?: string[] } | undefined)?.event_ids?.length ?? 0
  } catch {
    // Отказ — в command.error.
  }
}

const time = (iso: string) => d(new Date(iso), 'dateTime')
const role = (r?: string | null) => (r ? t(`widgets.suggestions.role.${r}`) : '—')
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="backendModeOf(q.data.value)"
    :loading="q.isPending.value && !list"
    :error="list ? undefined : q.error.value"
    :data-widget="widgetId"
  >
    <template #actions>
      <ActionButton
        type="primary"
        :label="t('widgets.suggestions.generate')"
        :size="naiveSizeOf(props.density)"
        :disabled="moment.isReplay"
        :loading="command.isPending.value"
        data-testid="generate"
        @click="generate"
      />
    </template>
    <div v-if="list" class="proposals" data-testid="proposals">
      <p class="principle ant-wrap" data-testid="principle">{{ t('widgets.suggestions.principle') }}</p>
      <p v-if="generated !== null" class="ok ant-wrap" data-testid="generated">{{ t('widgets.suggestions.generated', { n: generated }) }}</p>
      <NAlert v-if="command.error.value" type="error" :bordered="false" :show-icon="false" data-testid="command-error">
        <span class="ant-wrap">{{ problemText(command.error.value) }}</span>
      </NAlert>

      <SectionPanel :title="t('widgets.suggestions.listTitle')" :subtitle="t('widgets.suggestions.listSubtitle')">
        <EmptyState v-if="!items.length" compact :title="t('widgets.suggestions.empty')" data-testid="no-proposals" />
        <DataTable v-else :caption="t('widgets.suggestions.listTitle')">
          <thead>
            <tr>
              <th scope="col">{{ t('widgets.suggestions.col.kind') }}</th>
              <th scope="col">{{ t('widgets.suggestions.col.title') }}</th>
              <th scope="col">{{ t('widgets.suggestions.col.responsible') }}</th>
              <th scope="col">{{ t('widgets.suggestions.col.status') }}</th>
              <th scope="col">{{ t('widgets.suggestions.col.at') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="s in items"
              :key="s.suggestion_id"
              class="row"
              tabindex="0"
              :data-suggestion="s.suggestion_id"
              :data-status="s.status"
              :aria-selected="open?.id === s.suggestion_id"
              @click="setOpen({ kind: 'suggestion', id: s.suggestion_id })"
              @keydown.enter.prevent="setOpen({ kind: 'suggestion', id: s.suggestion_id })"
            >
              <td class="ant-box"><span class="ant-wrap">{{ t(`widgets.suggestions.kind.${s.kind}`) }}</span></td>
              <td class="ant-box">
                <span class="ant-clamp-2" :title="s.statement">{{ s.title || s.statement }}</span>
                <span v-if="s.estimate" class="muted ant-clamp-2">{{ s.estimate }}</span>
              </td>
              <td class="ant-box"><span class="ant-wrap">{{ s.responsible_id ?? role(s.responsible_role) }}</span></td>
              <td class="ant-box"><span class="status ant-wrap" :data-status="s.status">{{ t(`widgets.suggestions.status.${s.status}`) }}</span></td>
              <td class="ant-box"><span class="ant-ellipsis">{{ time(s.recorded_at) }}</span></td>
            </tr>
          </tbody>
        </DataTable>
      </SectionPanel>

    </div>
    <SuggestionDrawer :suggestion="current" :density="density" @close="setOpen(null)" />
  </WidgetFrame>
</template>

<style scoped>
.proposals {
  display: flex;
  flex-direction: column;
  gap: var(--ant-gap);
  font-size: var(--ant-fs-body);
}

.principle {
  margin: 0;
  padding: var(--ant-space-2) var(--ant-space-3);
  border-left: 3px solid var(--ant-status-neutral);
  background: var(--ant-surface-subtle);
}

.ok {
  margin: 0;
  color: var(--ant-status-success);
}

.row {
  cursor: pointer;
}

.row:hover {
  background: var(--ant-surface-hover);
}

.row[aria-selected='true'] {
  background: var(--ant-accent-soft);
}

.row[data-status='new'] td:first-child {
  box-shadow: inset 3px 0 0 var(--ant-accent);
}

.muted {
  display: block;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}



.status {
  font-size: var(--ant-fs-meta);
}
</style>
