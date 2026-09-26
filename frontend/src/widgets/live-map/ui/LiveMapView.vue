<script setup lang="ts">
/**
 * Живая карта на готовых данных (FR-1…3, 5, 7, 9, 130, 154, 155): панель периода
 * и версии, полоса режима инцидента с легендой и счётчиком сокращения области,
 * схема с наложениями. Схема занимает всю оставшуюся высоту места (в своём
 * разделе — окна); щелчок по узлу открывает правое окно (Д-70, UI-12): шаг,
 * цех, описание, счётчики, изделия; внизу — «Изделия и несоответствия узла».
 * Данные приходят свойствами — компонент не ходит на сервер; контейнер —
 * LiveMapWidget.vue.
 */
import { computed, ref, shallowRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NDatePicker, NIcon, NRadioButton, NRadioGroup, NSelect, NTooltip } from 'naive-ui'
import { InfoCircle } from '@vicons/tabler'
import type { CounterPeriod, LiveMapData, NodeAnomaly, NodeCounters } from '@/entities/live-map'
import type { ProcessSummary } from '@/shared/api/generated/model'
import { naiveSizeOf, type Density } from '@/shared/config/widget'
import { ActionButton, RecordDrawer } from '@/shared/ui'
import type { DiagramIndex, StepNode } from '../model/bpmn'
import { INCIDENT_LEGEND, itemsByStep, itemsOfOtherVersions, itemsPerLane, scopeReduction } from '../model/overlays'
import BpmnMapViewer from './BpmnMapViewer.vue'
import NodeCard from './NodeCard.vue'

const props = withDefaults(
  defineProps<{
    data: LiveMapData
    period: CounterPeriod
    /** Границы произвольного периода, мс UTC. */
    range?: [number, number] | null
    /** Процессы предприятия для выбора (UI-11). */
    processes?: ProcessSummary[]
    density?: Density
  }>(),
  { range: null, processes: () => [], density: 'comfortable' },
)
const emit = defineEmits<{
  'update:period': [period: CounterPeriod]
  'update:range': [range: [number, number] | null]
  'select-process': [processId: string]
  'select-version': [processVersionId: string]
  'open-item': [itemId: string]
  'open-node': [stepKey: string]
}>()
const { t, n } = useI18n()

const PERIODS: CounterPeriod[] = ['shift', 'day', 'week', 'month', 'custom']

const index = shallowRef<DiagramIndex | null>(null)
const importError = ref<Error | null>(null)
const selected = ref<string | null>(null)
/** Плашка инцидента развёрнута; свёрнутая — одна строка поверх схемы. */
const incidentOpen = ref(true)

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

const processOptions = computed(() =>
  props.processes.map((p) => ({
    value: p.process_id,
    label: [p.name, t('plural.items', { n: p.items_in_work }, p.items_in_work), p.status === 'active' ? '' : t(`liveMap.processStatus.${p.status}`)].filter(Boolean).join(' · '),
  })),
)

const versionOptions = computed(() =>
  props.data.versions.map((v) => ({
    value: v.process_version_id,
    label: `${t('liveMap.versionLabel', { version: v.label })} · ${t('plural.items', { n: v.items }, v.items)}`,
  })),
)

const selectedNode = computed(() => (selected.value && index.value ? index.value.byStepKey.get(selected.value) ?? null : null))
/** Последний выбранный узел остаётся в окне, пока оно уезжает. */
const shownNode = shallowRef<StepNode | null>(null)
watch(selectedNode, (node) => {
  if (node) shownNode.value = node
})
const nodeSubtitle = computed(() => {
  const node = shownNode.value
  if (!node) return ''
  return [node.laneName ? `${t('liveMap.workshops.lane')}: ${node.laneName}` : '', node.stepKey].filter(Boolean).join(' · ')
})

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
      <div class="selects">
        <NSelect
          v-if="processOptions.length"
          class="process"
          :size="naiveSizeOf(density)"
          :value="data.process_id"
          :options="processOptions"
          :consistent-menu-width="false"
          data-testid="process"
          @update:value="(v: string) => emit('select-process', v)"
        />
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
      <NTooltip placement="bottom-end">
        <template #trigger>
          <NIcon class="hint" size="18" :aria-label="t('liveMap.noPeopleOnMap')" tabindex="0"><InfoCircle /></NIcon>
        </template>
        <span class="ant-wrap">{{ t('liveMap.noPeopleOnMap') }}</span>
        <span v-if="otherVersions" class="ant-wrap" data-testid="other-versions">
          <br />{{ t('liveMap.ownVersionNote') }} · {{ t('plural.items', { n: otherVersions }, otherVersions) }}
        </span>
      </NTooltip>
    </div>

    <div class="map">
      <div class="canvas-box">
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
          @ready="onReady"
          @import-error="(e) => (importError = e)"
          @select-node="(k) => (selected = k)"
          @open-item="(id) => emit('open-item', id)"
        />
      </div>

      <!-- Режим инцидента — плашкой поверх схемы: место под схему не меняется. -->
      <section v-if="incident && reduction" class="incident" :data-open="incidentOpen || undefined" data-testid="incident">
      <div class="incident-head">
        <ActionButton
          text
          size="tiny"
          class="incident-toggle"
          :label="incidentOpen ? '▾' : '▸'"
          :aria-expanded="incidentOpen"
          :aria-label="incidentOpen ? t('liveMap.incident.collapse') : t('liveMap.incident.expand')"
          @click="incidentOpen = !incidentOpen"
        />
        <strong>{{ t('liveMap.incident.modeTitle', { incident: incident.label }) }}</strong>
        <span data-testid="scope-version">{{ t('riskScope.version', { version: incident.scope_version }) }}</span>
        <span class="reduction" data-testid="scope-reduction">
          {{ t('riskScope.reduction', { from: reduction.from, to: reduction.to }) }}
          · {{ t('riskScope.reductionPercent', { percent: n(reduction.fraction, 'percent') }) }}
        </span>
      </div>
      <template v-if="incidentOpen">
        <p v-if="incident.basis" class="basis ant-clamp-2" :title="incident.basis">{{ t('riskScope.basis') }}: {{ incident.basis }}</p>
        <ul class="legend">
          <li v-for="l in INCIDENT_LEGEND" :key="l.status" :data-legend="l.status">
            <span class="swatch" :style="{ background: l.color }" aria-hidden="true" />{{ t(l.textKey) }}
          </li>
        </ul>
        <p class="note">{{ t('liveMap.incident.colorNote') }}</p>
      </template>
      </section>
    </div>

    <!-- Полоса времени — часть карты, постоянной высоты (UI-19). -->
    <div v-if="$slots.timeline" class="timeline-strip">
      <slot name="timeline" />
    </div>

    <RecordDrawer
      :show="!!selectedNode"
      :kind-label="t('liveMap.nodeKind')"
      :number="shownNode ? shownNode.name || shownNode.stepKey : ''"
      :subtitle="nodeSubtitle"
      data-record="node"
      @close="selected = null"
    >
      <NodeCard
        v-if="shownNode"
        :node="shownNode"
        :counters="counters.get(shownNode.stepKey)"
        :items="byStep.get(shownNode.stepKey) ?? []"
        :incident-mode="!!incident"
        @open-item="(id) => emit('open-item', id)"
      />
      <template #actions>
        <ActionButton
          v-if="shownNode"
          type="primary"
          data-action="open-node"
          :label="t('liveMap.drillDown.nodeItems')"
          @click="emit('open-node', shownNode.stepKey)"
        />
      </template>
    </RecordDrawer>
  </div>
</template>

<style scoped>
.live-map {
  display: flex;
  flex-direction: column;
  gap: 8px;
  height: 100%;
  min-height: 0;
}

.toolbar {
  display: flex;
  flex: none;
  flex-wrap: nowrap;
  gap: 8px;
  align-items: center;
  min-width: 0;
}

.selects {
  display: flex;
  flex: 1 1 auto;
  flex-wrap: nowrap;
  gap: 8px;
  justify-content: flex-end;
  min-width: 0;
}

.process,
.version {
  flex: 0 1 300px;
  min-width: 160px;
}

.hint {
  flex: none;
  color: var(--ant-text-3);
  cursor: help;
}

.note {
  margin: 0;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

/* Схема — вся оставшаяся высота и ширина, строго внутри своей рамки:
   двигаемся по схеме, не по странице; размер не зависит от инцидента и времени. */
.map {
  position: relative;
  flex: 1 1 auto;
  min-width: 0;
  min-height: 420px;
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-md);
  overflow: hidden;
}

.canvas-box {
  position: absolute;
  inset: 0;
}

.incident {
  position: absolute;
  top: var(--ant-space-2);
  left: var(--ant-space-2);
  z-index: 2;
  max-width: min(760px, calc(100% - var(--ant-space-4)));
  padding: var(--ant-space-2) var(--ant-space-3);
  border-left: 3px solid var(--ant-status-danger);
  border-radius: var(--ant-radius-sm);
  background: var(--ant-status-danger-soft);
  box-shadow: var(--ant-shadow-md);
  font-size: var(--ant-fs-meta);
}

.incident-head {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 10px;
  align-items: baseline;
}

.reduction {
  font-weight: var(--ant-fw-bold);
}

.basis {
  margin: 4px 0 0;
}

.legend {
  display: flex;
  flex-wrap: wrap;
  gap: 2px 12px;
  margin: 4px 0 2px;
  padding: 0;
  list-style: none;
}

.swatch {
  display: inline-block;
  width: 10px;
  height: 10px;
  margin-right: 6px;
  border-radius: 50%;
}

.timeline-strip {
  flex: none;
  padding-top: var(--ant-space-1);
}

.import-error {
  position: absolute;
  z-index: 1;
  margin: 8px;
  color: var(--ant-status-danger);
}
</style>
