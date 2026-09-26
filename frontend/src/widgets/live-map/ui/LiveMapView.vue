<script setup lang="ts">
/**
 * Живая карта на готовых данных (FR-1…3, 5, 7, 9, 130, 154, 155): панель периода
 * и версии, полоса режима инцидента с легендой и счётчиком сокращения области,
 * схема с наложениями и карточка выбранного узла. Данные приходят свойствами —
 * компонент не ходит на сервер; контейнер — LiveMapWidget.vue.
 */
import { computed, ref, shallowRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NDatePicker, NRadioButton, NRadioGroup, NSelect, NSwitch } from 'naive-ui'
import { NormsPanel, parseNorms, type StepNorms } from '@/features/norms-layer'
import type { CounterPeriod, LiveMapData, NodeAnomaly, NodeCounters } from '@/entities/live-map'
import { naiveSizeOf, type Density } from '@/shared/config/widget'
import type { DiagramIndex } from '../model/bpmn'
import { INCIDENT_LEGEND, itemsByStep, itemsOfOtherVersions, itemsPerLane, scopeReduction } from '../model/overlays'
import BpmnMapViewer from './BpmnMapViewer.vue'
import NodeCard from './NodeCard.vue'

const props = withDefaults(
  defineProps<{
    data: LiveMapData
    period: CounterPeriod
    /** Границы произвольного периода, мс UTC. */
    range?: [number, number] | null
    density?: Density
  }>(),
  { range: null, density: 'comfortable' },
)
const emit = defineEmits<{
  'update:period': [period: CounterPeriod]
  'update:range': [range: [number, number] | null]
  'select-version': [processVersionId: string]
  'open-item': [itemId: string]
  'open-node': [stepKey: string]
}>()
const { t, n } = useI18n()

const PERIODS: CounterPeriod[] = ['shift', 'day', 'week', 'month', 'custom']

const index = shallowRef<DiagramIndex | null>(null)
const importError = ref<Error | null>(null)
const selected = ref<string | null>(null)

// Новая схема — прежний выбор узла может быть не из неё.
watch(
  () => props.data.bpmn_xml,
  () => {
    importError.value = null
  },
)

const versionId = computed(() => props.data.process_version.process_version_id)
const byStep = computed(() => itemsByStep(props.data.items, versionId.value))
const counters = computed(() => new Map<string, NodeCounters>(props.data.counters.map((c) => [c.step_key, c])))
const anomalies = computed(() => {
  const m = new Map<string, NodeAnomaly[]>()
  for (const a of props.data.anomalies) m.set(a.step_key, [...(m.get(a.step_key) ?? []), a])
  return m
})
const dataGaps = computed(() => new Set(props.data.data_gaps))
const laneCounts = computed(() => (index.value ? itemsPerLane(byStep.value, index.value) : new Map<string, number>()))
const otherVersions = computed(() => itemsOfOtherVersions(props.data.items, versionId.value))

// Необязательные объекты контракта могут не прийти — читаем как «нет» (null).
const incident = computed(() => props.data.incident ?? null)
const bottleneck = computed(() => props.data.bottleneck ?? null)
const reduction = computed(() => (incident.value ? scopeReduction(incident.value) : null))

const versionOptions = computed(() =>
  props.data.versions.map((v) => ({
    value: v.process_version_id,
    label: `${t('liveMap.versionLabel', { version: v.label })} · ${t('plural.items', { n: v.items }, v.items)}`,
  })),
)

// Слой «нормы» (FR-156, эпик 39): опоры шагов из XML показанной версии.
const normsOn = ref(false)
const norms = computed(() => (normsOn.value ? parseNorms(props.data.bpmn_xml) : new Map<string, StepNorms>()))
const normSteps = computed(() => new Set(norms.value.keys()))

const selectedNode = computed(() => (selected.value && index.value ? index.value.byStepKey.get(selected.value) ?? null : null))

function onReady(idx: DiagramIndex) {
  index.value = idx
  if (selected.value && !idx.byStepKey.has(selected.value)) selected.value = null
}
</script>

<template>
  <div class="live-map" :class="`density-${density}`" :data-version="versionId" :data-incident="incident?.incident_id">
    <div class="toolbar">
      <NRadioGroup
        :value="period"
        :size="naiveSizeOf(density)"
        name="period"
        data-testid="period"
        @update:value="(v: CounterPeriod) => emit('update:period', v)"
      >
        <NRadioButton v-for="p in PERIODS" :key="p" :value="p" :data-period="p">{{ t(`liveMap.period.${p}`) }}</NRadioButton>
      </NRadioGroup>
      <NDatePicker
        v-if="period === 'custom'"
        type="datetimerange"
        :size="naiveSizeOf(density)"
        :value="range"
        clearable
        @update:value="(v: [number, number] | null) => emit('update:range', v)"
      />
      <label class="norms-toggle" :title="t('normsLayer.hint')">
        <NSwitch v-model:value="normsOn" :size="density === 'large' ? 'large' : 'medium'" data-testid="norms-toggle" />
        {{ t('normsLayer.toggle') }}
      </label>
      <NSelect
        class="version"
        :size="naiveSizeOf(density)"
        :value="versionId"
        :options="versionOptions"
        :consistent-menu-width="false"
        data-testid="version"
        @update:value="(v: string) => emit('select-version', v)"
      />
    </div>
    <p v-if="otherVersions" class="note" data-testid="other-versions">
      {{ t('liveMap.ownVersionNote') }} · {{ t('plural.items', { n: otherVersions }, otherVersions) }}
    </p>

    <section v-if="incident && reduction" class="incident" data-testid="incident">
      <div class="incident-head">
        <strong>{{ t('liveMap.incident.modeTitle', { incident: incident.label }) }}</strong>
        <span data-testid="scope-version">{{ t('riskScope.version', { version: incident.scope_version }) }}</span>
        <span class="reduction" data-testid="scope-reduction">
          {{ t('riskScope.reduction', { from: reduction.from, to: reduction.to }) }}
          · {{ t('riskScope.reductionPercent', { percent: n(reduction.fraction, 'percent') }) }}
        </span>
      </div>
      <p v-if="incident.basis" class="basis">{{ t('riskScope.basis') }}: {{ incident.basis }}</p>
      <ul class="legend">
        <li v-for="l in INCIDENT_LEGEND" :key="l.status" :data-legend="l.status">
          <span class="swatch" :style="{ background: l.color }" aria-hidden="true" />{{ t(l.textKey) }}
        </li>
      </ul>
      <p class="note">{{ t('liveMap.incident.colorNote') }}</p>
    </section>

    <div class="body" :class="{ 'with-card': selectedNode }">
      <div class="map">
        <p v-if="importError" class="import-error" role="alert">{{ t('errors.loadFailed') }}</p>
        <BpmnMapViewer
          :xml="data.bpmn_xml"
          :counters="counters"
          :items-by-step="byStep"
          :lane-counts="laneCounts"
          :bottleneck="bottleneck"
          :anomalies="anomalies"
          :data-gaps="dataGaps"
          :incident-mode="!!incident"
          :selected="selected"
          :norm-steps="normSteps"
          @ready="onReady"
          @import-error="(e) => (importError = e)"
          @select-node="(k) => (selected = k)"
          @open-item="(id) => emit('open-item', id)"
        />
      </div>
      <div v-if="selectedNode" class="side">
        <NormsPanel v-if="normsOn" :norms="norms.get(selectedNode.stepKey) ?? null" />
        <NodeCard
          :node="selectedNode"
          :counters="counters.get(selectedNode.stepKey)"
          :items="byStep.get(selectedNode.stepKey) ?? []"
          :incident-mode="!!incident"
          @close="selected = null"
          @open-item="(id) => emit('open-item', id)"
          @open-node="(k) => emit('open-node', k)"
        />
      </div>
    </div>
    <p v-if="normsOn" class="note" data-testid="norms-hint">{{ t('normsLayer.hint') }} · {{ t('normsLayer.count', { n: normSteps.size }) }}</p>
    <p class="note">{{ t('liveMap.noPeopleOnMap') }}</p>
  </div>
</template>

<style scoped>
.live-map {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.version {
  width: auto;
  min-width: 260px;
  margin-left: auto;
}

.note {
  margin: 0;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.incident {
  padding: 8px 12px;
  border-left: 3px solid var(--ant-status-danger);
  border-radius: var(--ant-radius-sm);
  background: var(--ant-status-danger-soft);
}

.incident-head {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: baseline;
}

.reduction {
  font-weight: var(--ant-fw-bold);
}

.basis {
  margin: 4px 0 0;
  font-size: var(--ant-fs-body);
}

.legend {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 16px;
  margin: 6px 0 4px;
  padding: 0;
  list-style: none;
  font-size: var(--ant-fs-body);
}

.swatch {
  display: inline-block;
  width: 10px;
  height: 10px;
  margin-right: 6px;
  border-radius: 50%;
}

.body {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 12px;
}

.body.with-card {
  grid-template-columns: minmax(0, 1fr) minmax(280px, 360px);
}

.norms-toggle {
  display: inline-flex;
  gap: 6px;
  align-items: center;
  font-size: var(--ant-fs-body);
  cursor: pointer;
}

.side {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
}

.map {
  position: relative;
  height: 560px;
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-md);
  overflow: hidden;
}

.density-compact .map {
  height: 480px;
}

.density-large .map {
  height: 640px;
}

.import-error {
  position: absolute;
  z-index: 1;
  margin: 8px;
  color: var(--ant-status-danger);
}
</style>
