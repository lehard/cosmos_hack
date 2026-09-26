<script setup lang="ts">
/**
 * Виджет «Мои расследования» — контейнер: инциденты (`analysis.incident.list`);
 * выбранный уходит в фокус разбора — его читают область риска и гипотезы
 * раздела «Расследование».
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NAlert } from 'naive-ui'
import { isOpen, useAnalysisCommands, useAnalysisFocusStore, useCan, useFocusedIncident, useFocusedNc, useHypotheses } from '@/entities/incident'
import { naiveSizeOf } from '@/shared/config/widget'
import { useProblemText } from '@/shared/i18n/problem'
import { useMomentStore } from '@/shared/model/moment'
import { ActionButton } from '@/shared/ui'
import type { WidgetProps } from '@/shared/config/widget'
import { WidgetFrame } from '@/shared/ui'
import { useRecordLink } from '@/shared/model/record'
import InvestigationsView from './InvestigationsView.vue'
import NcLink from './NcLink.vue'

const props = defineProps<WidgetProps>()
const { t } = useI18n()
const problemText = useProblemText()
const moment = useMomentStore()

const focus = useAnalysisFocusStore()
const { incidents, list, id } = useFocusedIncident()
const items = computed(() => list.value)
/** Выбрать расследование: его несоответствие (primary_nc_id) становится входом гипотез и дорожек. */
/** Несоответствие расследования → окно несоответствия (Д-70), выбор — вход гипотез и дорожек. */
const record = useRecordLink()
function openNc(incidentId: string, ncId: string): void {
  select(incidentId)
  focus.ncId = ncId
  record.open({ entity: 'nonconformity', id: ncId })
}

/**
 * Главное действие «что делать дальше»: у открытой гипотезы есть «что проверить
 * следующим» — «Запросить проверку» одним щелчком (analysis.measurement.request);
 * результат придёт записью и в историю гипотезы.
 */
const ncId = useFocusedNc()
const hyps = useHypotheses(ncId)
const target = computed(() =>
  [...(hyps.data.value?.hypotheses ?? [])]
    .filter((h) => isOpen(h) && (h.next_check?.text || h.measurement_hint))
    .sort((a, b) => (b.confidence_bp ?? 0) - (a.confidence_bp ?? 0))[0] ?? null,
)
const can = useCan()
const cmd = useAnalysisCommands()
const canMeasure = computed(() => Boolean(ncId.value) && can('analysis.measurement.request', 'nonconformity', ncId.value))
function requestCheck(): void {
  const h = target.value
  if (!h || !ncId.value) return
  cmd.requestMeasurement.reset()
  cmd.requestMeasurement.mutate({
    ncId: ncId.value,
    body: { hypothesis_id: h.hypothesis_id, what: h.next_check?.text || h.measurement_hint || '', basis_seq: hyps.basisSeq.value ?? 0 },
  })
}

function select(incidentId: string): void {
  focus.incidentId = incidentId
  focus.ncId = null
  focus.eventId = null
}
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :loading="incidents.isPending.value"
    :error="incidents.error.value"
    :empty="!items.length"
    empty-key="widgets.analysis.investigations.empty"
    :data-widget="widgetId"
  >
    <InvestigationsView v-if="items.length" :incidents="items" :selected="id" @select="select">
      <template v-if="target && canMeasure" #cta>
        <ActionButton
          overflow="wrap"
          :size="naiveSizeOf(props.density)"
          type="primary"
          :disabled="cmd.requestMeasurement.isPending.value || cmd.requestMeasurement.isSuccess.value || moment.isReplay"
          :loading="cmd.requestMeasurement.isPending.value"
          data-testid="cta-request-check"
          @click="requestCheck"
          :label="t('widgets.analysis.hypothesis.requestCheck')"
        />
        <NAlert v-if="cmd.requestMeasurement.error.value" type="error" :bordered="false" :show-icon="false" data-testid="cta-error">{{ problemText(cmd.requestMeasurement.error.value) }}</NAlert>
        <p v-else-if="cmd.requestMeasurement.isSuccess.value" class="cta-sent" data-testid="cta-sent">{{ t('widgets.analysis.investigations.checkRequested') }}</p>
      </template>
      <template #ncs="{ incident }">
        <NcLink v-for="nc in incident.nc_ids" :key="nc" :id="nc" @open="(ncId) => openNc(incident.incident_id, ncId)" />
      </template>
    </InvestigationsView>
  </WidgetFrame>
</template>

<style scoped>
.cta-sent {
  margin: 0;
  color: var(--ant-status-success-text);
  font-weight: var(--ant-fw-bold);
}
</style>
