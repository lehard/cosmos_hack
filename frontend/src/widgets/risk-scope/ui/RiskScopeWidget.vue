<script setup lang="ts">
/**
 * Виджет «Область риска» (FR-61, FR-62) — контейнер: `analysis.risk_scope.read`
 * по инциденту в разборе; при нескольких инцидентах — выбор.
 *
 * Сужение (`analysis.scope.narrow`) и расширение (`analysis.scope.expand`) —
 * решения человека с основанием (группа критических действий `risk_scope`);
 * кнопки доступны по списку прав сервера (AD-15).
 */
import { computed, inject } from 'vue'
import { useI18n } from 'vue-i18n'
import { NAlert } from 'naive-ui'
import { routerKey } from 'vue-router'
import { scopeState, useAnalysisCommands, useAnalysisFocusStore, useCan } from '@/entities/incident'
import type { WidgetProps } from '@/shared/config/widget'
import { useProblemText } from '@/shared/i18n/problem'
import { WidgetFrame } from '@/shared/ui'
import { useRiskScopeSource } from '../model/source'
import RiskScopeView from './RiskScopeView.vue'

defineProps<WidgetProps>()

const { t } = useI18n()
const problemText = useProblemText()
// Роутер берём необязательно: виджет живёт и вне страницы (тесты столов).
const router = inject(routerKey, null)
const focus = useAnalysisFocusStore()
const src = useRiskScopeSource()
const data = computed(() => src.data.value)
const state = computed(() => (data.value ? scopeState(data.value) : 'normal'))

const can = useCan()
const cmd = useAnalysisCommands()
const mutations = [cmd.narrowScope, cmd.expandScope]
const busy = computed(() => mutations.some((m) => m.isPending.value))
const lastError = computed(() => mutations.map((m) => m.error.value).find(Boolean) ?? null)
const sent = computed(() => mutations.some((m) => m.isSuccess.value))

const canNarrow = computed(() => can('analysis.scope.narrow', 'incident', src.incidentId.value))
const canExpand = computed(() => can('analysis.scope.expand', 'incident', src.incidentId.value))

function change(kind: 'narrow' | 'expand', input: { item_ids: string[]; reason: string }): void {
  const incidentId = src.incidentId.value
  if (!incidentId) return
  mutations.forEach((m) => m.reset())
  const body = { item_ids: input.item_ids, reason: { text: input.reason }, evidence_event_ids: [], basis_seq: src.basisSeq.value ?? 0 }
  if (kind === 'narrow') cmd.narrowScope.mutate({ incidentId, body })
  else cmd.expandScope.mutate({ incidentId, body })
}

/** Изделие → паспорт изделия (FR-7). */
const openItem = (id: string) => router?.push({ name: 'item', params: { id } })

const pick = (e: Event) => (focus.incidentId = (e.target as HTMLSelectElement).value || null)
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
    <label v-if="src.incidents.value.length > 1" class="incident-pick">
      {{ t('widgets.analysis.riskScope.pickIncident') }}:
      <select :value="src.incidentId.value ?? ''" data-testid="incident-pick" @change="pick">
        <option v-for="i in src.incidents.value" :key="i.incident_id" :value="i.incident_id">{{ i.label }} · {{ i.size }}</option>
      </select>
    </label>
    <NAlert v-if="lastError" type="error" :bordered="false" :show-icon="false" class="cmd-note" data-testid="command-error">{{ problemText(lastError) }}</NAlert>
    <NAlert v-else-if="sent" type="success" :bordered="false" :show-icon="false" class="cmd-note" data-testid="command-sent">
      {{ t('widgets.analysis.commandSent') }}
    </NAlert>
    <RiskScopeView
      v-if="data"
      :model="data"
      :density="density"
      :can-narrow="canNarrow"
      :can-expand="canExpand"
      :busy="busy"
      @narrow="(input) => change('narrow', input)"
      @expand="(input) => change('expand', input)"
      @open-item="openItem"
    />
  </WidgetFrame>
</template>

<style scoped>
.incident-pick {
  display: flex;
  gap: 6px;
  align-items: center;
  margin-bottom: 8px;
  font-size: var(--ant-fs-body);
}

.cmd-note {
  margin-bottom: 8px;
  padding: 6px 10px;
}
</style>
