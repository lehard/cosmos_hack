<script setup lang="ts">
/**
 * Правое окно строки карты дефицита (Д-70; FR-143): какие расследования и
 * инциденты, на каких записях построены выводы, какая цифровизация и оценка
 * сужения; предложение цифровизации — если записано, иначе кнопка в нижней
 * панели «Сформировать предложение» (прогон генераторов, FR-63).
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NAlert } from 'naive-ui'
import { useSession } from '@/entities/session'
import { useSuggestionCommand, type DataDeficitMap, type DeficitRow } from '@/entities/suggestion'
import { naiveSizeOf, type Density } from '@/shared/config/widget'
import { useProblemText } from '@/shared/i18n/problem'
import { useMomentStore } from '@/shared/model/moment'
import { ActionButton, KeyValue, KeyValueList, RecordDrawer, SectionPanel } from '@/shared/ui'

const props = withDefaults(defineProps<{ row: DeficitRow | null; map: DataDeficitMap | null; density?: Density }>(), { density: 'comfortable' })
const emit = defineEmits<{ close: [] }>()
const { t } = useI18n()
const problemText = useProblemText()
const moment = useMomentStore()
const session = useSession()
const command = useSuggestionCommand(() => {
  const s = session.data.value?.data
  return { policySeq: s?.policy_seq ?? 0, workplaceId: s?.workplace?.id }
})

const what = computed(() => (props.row ? t(`widgets.analysis.missing.${missingKey(props.row.kind)}`) : ''))

/** Код вида сведений → ключ текста разбора (эпик 12). */
function missingKey(kind: string): string {
  return kind.replace(/_([a-z])/g, (_, c: string) => c.toUpperCase())
}

async function generate(): Promise<void> {
  try {
    await command.mutateAsync({ kind: 'generate' })
  } catch {
    // Отказ — в command.error.
  }
}
</script>

<template>
  <RecordDrawer :show="!!row" :kind-label="t('widgets.deficit.drawerKind')" :number="what" :subtitle="row?.digitization ?? ''" @close="emit('close')">
    <div v-if="row" class="body" data-testid="deficit-drawer" :data-missing="row.kind">
      <KeyValueList>
        <KeyValue :label="t('widgets.deficit.col.count')" :value="t('widgets.deficit.ofTotal', { n: row.investigations, total: map?.investigations ?? 0 })" />
        <KeyValue :label="t('widgets.deficit.col.where')" :value="row.places.map((p) => `${p.place} (${p.count})`).join(', ') || null" />
        <KeyValue :label="t('widgets.deficit.col.digitization')" :value="row.digitization" />
        <KeyValue
          :label="t('widgets.deficit.col.estimate')"
          :value="row.estimate ? `${row.estimate.text} — ${t('widgets.deficit.estimateBasis', { n: row.estimate.incidents })}` : t('widgets.deficit.noEstimate')"
        />
      </KeyValueList>
      <p v-if="!row.estimate" class="muted ant-wrap">{{ t('widgets.deficit.noEstimateHint') }}</p>
      <SectionPanel variant="subtle" :title="t('widgets.deficit.ncs')">
        <p class="ant-mono ant-wrap" data-testid="ncs">{{ row.nc_ids.join(', ') || '—' }}</p>
      </SectionPanel>
      <SectionPanel variant="subtle" :title="t('widgets.deficit.incidents')">
        <p class="ant-mono ant-wrap">{{ row.incident_ids.join(', ') || '—' }}</p>
      </SectionPanel>
      <SectionPanel variant="subtle" :title="t('widgets.deficit.suggestion')">
        <p class="ant-wrap" data-testid="suggestion">
          <span v-if="row.suggestion_id" class="ant-mono">{{ row.suggestion_id }}</span>
          <span v-else class="muted">{{ t('widgets.deficit.suggestionNone') }}</span>
        </p>
      </SectionPanel>
      <NAlert v-if="command.error.value" type="error" :bordered="false" :show-icon="false">
        <span class="ant-wrap">{{ problemText(command.error.value) }}</span>
      </NAlert>
    </div>
    <template v-if="row && !row.suggestion_id" #actions>
      <ActionButton
        type="primary"
        :label="t('widgets.deficit.generate')"
        :size="naiveSizeOf(density)"
        :disabled="moment.isReplay"
        :loading="command.isPending.value"
        data-testid="generate"
        @click="generate"
      />
    </template>
  </RecordDrawer>
</template>

<style scoped>
.body {
  display: flex;
  flex-direction: column;
  gap: var(--ant-gap);
  font-size: var(--ant-fs-body);
}

.muted {
  margin: 0;
  color: var(--ant-text-3);
}

p {
  margin: 0;
}
</style>
