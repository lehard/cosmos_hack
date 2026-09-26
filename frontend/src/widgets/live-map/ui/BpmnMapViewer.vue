<script setup lang="ts">
/**
 * Просмотрщик схемы процесса с наложениями (FR-1, FR-2, FR-5, FR-9, FR-130; AD-21).
 *
 * bpmn-js NavigatedViewer (только чтение, перемещение и масштаб) показывает BPMN
 * версии с раскладкой BPMNDI: дорожки — цеха. Поверх узлов — наложения
 * diagram-js, в которые Vue телепортирует свои компоненты: счётчики узла, точки
 * изделий, метки ограничения и аномалий; над дорожкой — число изделий в цехе.
 * Узел-ограничение, аномалии, «оценка невозможна» и выбранный узел — ещё и
 * маркеры на самой фигуре. Водяной знак bpmn.io остаётся видимым (AD-21).
 *
 * Данные связываются с узлами по step_key (model/bpmn.ts), не по id элемента.
 */
import { markRaw, onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import NavigatedViewer from 'bpmn-js/lib/NavigatedViewer'
import 'bpmn-js/dist/assets/diagram-js.css'
import 'bpmn-js/dist/assets/bpmn-js.css'
import type { Bottleneck, MapItem, NodeAnomaly, NodeCounters } from '@/entities/live-map'
import { buildIndex, type BusinessObjectLike, type DiagramIndex, type StepNode } from '../model/bpmn'
import NodeBadge from './NodeBadge.vue'
import NodeDots from './NodeDots.vue'

const props = withDefaults(
  defineProps<{
    /** BPMN 2.0 XML показанной версии. */
    xml: string
    counters?: ReadonlyMap<string, NodeCounters>
    itemsByStep?: ReadonlyMap<string, MapItem[]>
    /** Изделий в дорожке-цехе по id дорожки (FR-130). */
    laneCounts?: ReadonlyMap<string, number>
    bottleneck?: Bottleneck | null
    anomalies?: ReadonlyMap<string, NodeAnomaly[]>
    dataGaps?: ReadonlySet<string>
    incidentMode?: boolean
    /** Выбранный узел (step_key). */
    selected?: string | null
    /** Слой «нормы» (FR-156, эпик 39): step_key узлов с нормативной опорой. */
    normSteps?: ReadonlySet<string>
  }>(),
  {
    counters: () => new Map(),
    itemsByStep: () => new Map(),
    laneCounts: () => new Map(),
    bottleneck: null,
    anomalies: () => new Map(),
    dataGaps: () => new Set(),
    incidentMode: false,
    selected: null,
    normSteps: () => new Set(),
  },
)

const emit = defineEmits<{
  /** Схема открыта: индекс узлов по step_key. */
  ready: [index: DiagramIndex]
  /** Схема не открылась (битый XML). */
  'import-error': [error: Error]
  'select-node': [stepKey: string]
  'open-item': [itemId: string]
}>()

const { t } = useI18n()

/** Минимум API bpmn-js/diagram-js, который нужен карте. */
interface ElementLike {
  id: string
  businessObject?: BusinessObjectLike
}
interface Services {
  elementRegistry: { getAll(): ElementLike[] }
  overlays: { add(id: string, type: string, o: { position: Record<string, number>; html: HTMLElement }): string; clear(): void }
  canvas: { addMarker(id: string, m: string): void; removeMarker(id: string, m: string): void; zoom(v: string): void }
  eventBus: { on(e: string, cb: (ev: { element: ElementLike }) => void): void }
}

const container = ref<HTMLElement | null>(null)
let viewer: NavigatedViewer | null = null
const svc = <K extends keyof Services>(name: K): Services[K] => viewer!.get(name) as Services[K]

/** Контейнеры наложений, куда телепортируются компоненты Vue. */
interface NodeSlot {
  node: StepNode
  top: HTMLElement
  bottom: HTMLElement
}
interface LaneSlot {
  bpmnId: string
  el: HTMLElement
}
const nodeSlots = shallowRef<NodeSlot[]>([])
const laneSlots = shallowRef<LaneSlot[]>([])
const index = shallowRef<DiagramIndex | null>(null)

/** Метки на фигурах: id элемента → маркеры, чтобы снимать только свои. */
const applied = new Map<string, Set<string>>()
const MARKERS = { bottleneck: 'ant-bottleneck', anomaly: 'ant-anomaly', gap: 'ant-data-gap', selected: 'ant-selected', norm: 'ant-norm' } as const

function applyMarkers(): void {
  if (!viewer || !index.value) return
  const canvas = svc('canvas')
  const want = new Map<string, Set<string>>()
  const mark = (stepKey: string | null | undefined, marker: string) => {
    const node = stepKey ? index.value!.byStepKey.get(stepKey) : undefined
    if (!node) return
    const set = want.get(node.bpmnId) ?? new Set<string>()
    set.add(marker)
    want.set(node.bpmnId, set)
  }
  mark(props.bottleneck?.step_key, MARKERS.bottleneck)
  for (const key of props.anomalies.keys()) mark(key, MARKERS.anomaly)
  for (const key of props.dataGaps) mark(key, MARKERS.gap)
  mark(props.selected, MARKERS.selected)
  for (const key of props.normSteps) mark(key, MARKERS.norm)
  for (const [id, set] of applied) for (const m of set) if (!want.get(id)?.has(m)) canvas.removeMarker(id, m)
  for (const [id, set] of want) for (const m of set) if (!applied.get(id)?.has(m)) canvas.addMarker(id, m)
  applied.clear()
  for (const [id, set] of want) applied.set(id, set)
}

/** Номер импорта: ответ устаревшего импорта не трогает холст. */
let importSeq = 0

async function load(xml: string): Promise<void> {
  if (!viewer) return
  const seq = ++importSeq
  try {
    await viewer.importXML(xml)
  } catch (e) {
    if (seq === importSeq) emit('import-error', e instanceof Error ? e : new Error(String(e)))
    return
  }
  if (seq !== importSeq) return
  const all = svc('elementRegistry').getAll()
  const idx = buildIndex(all.map((el) => el.businessObject).filter((bo): bo is BusinessObjectLike => !!bo))
  const overlays = svc('overlays')
  overlays.clear()
  applied.clear()
  const nodes: NodeSlot[] = []
  for (const node of idx.byBpmnId.values()) {
    const top = document.createElement('div')
    const bottom = document.createElement('div')
    overlays.add(node.bpmnId, 'ant-counters', { position: { top: -4, left: 0 }, html: top })
    overlays.add(node.bpmnId, 'ant-items', { position: { bottom: -4, left: 0 }, html: bottom })
    nodes.push({ node, top, bottom })
  }
  const lanes: LaneSlot[] = []
  for (const lane of idx.lanes) {
    const el = document.createElement('div')
    overlays.add(lane.bpmnId, 'ant-lane', { position: { top: 4, left: 36 }, html: el })
    lanes.push({ bpmnId: lane.bpmnId, el })
  }
  index.value = markRaw(idx)
  nodeSlots.value = nodes
  laneSlots.value = lanes
  try {
    svc('canvas').zoom('fit-viewport')
  } catch {
    // Холст без размеров (скрытая вкладка) — масштаб поставит пользователь.
  }
  applyMarkers()
  emit('ready', idx)
}

onMounted(() => {
  viewer = new NavigatedViewer({ container: container.value! })
  svc('eventBus').on('element.click', ({ element }) => {
    const node = index.value?.byBpmnId.get(element.id)
    if (node) emit('select-node', node.stepKey)
  })
  void load(props.xml)
})

onBeforeUnmount(() => {
  importSeq++
  viewer?.destroy()
  viewer = null
})

watch(
  () => props.xml,
  (xml, old) => {
    if (xml !== old) void load(xml)
  },
)
watch(() => [props.bottleneck, props.anomalies, props.dataGaps, props.selected, props.normSteps], applyMarkers)

/** Для тестов и родителя: индекс открытой схемы. */
defineExpose({ index })
</script>

<template>
  <div class="bpmn-map">
    <div ref="container" class="canvas" data-testid="bpmn-canvas" />
    <template v-for="s in nodeSlots" :key="s.node.bpmnId">
      <Teleport :to="s.top">
        <NodeBadge
          v-if="counters.has(s.node.stepKey) || bottleneck?.step_key === s.node.stepKey || anomalies.has(s.node.stepKey) || dataGaps.has(s.node.stepKey)"
          :data-step="s.node.stepKey"
          :counters="counters.get(s.node.stepKey)"
          :bottleneck="bottleneck?.step_key === s.node.stepKey ? bottleneck : null"
          :anomalies="anomalies.get(s.node.stepKey)"
          :data-gap="dataGaps.has(s.node.stepKey)"
          :selected="selected === s.node.stepKey"
          @select="emit('select-node', s.node.stepKey)"
        />
      </Teleport>
      <Teleport :to="s.bottom">
        <NodeDots
          v-if="itemsByStep.get(s.node.stepKey)?.length"
          :data-step="s.node.stepKey"
          :items="itemsByStep.get(s.node.stepKey)!"
          :incident-mode="incidentMode"
          @open="(id) => emit('open-item', id)"
          @more="emit('select-node', s.node.stepKey)"
        />
      </Teleport>
    </template>
    <Teleport v-for="l in laneSlots" :key="l.bpmnId" :to="l.el">
      <span v-if="laneCounts.get(l.bpmnId)" class="lane-count" :data-lane="l.bpmnId">
        {{ t('plural.items', { n: laneCounts.get(l.bpmnId) }, laneCounts.get(l.bpmnId)!) }}
      </span>
    </Teleport>
  </div>
</template>

<style scoped>
.bpmn-map {
  position: relative;
  height: 100%;
  min-height: 480px;
}

.canvas {
  position: absolute;
  inset: 0;
}

.lane-count {
  padding: 1px 6px;
  border-radius: var(--ant-radius-lg);
  background: color-mix(in srgb, var(--ant-accent) 12%, transparent);
  color: var(--ant-text);
  font: 600 11px/16px 'PT Sans', sans-serif;
  white-space: nowrap;
}

/* Маркеры на фигурах bpmn-js (цвета — палитра словаря статусов). */
.canvas :deep(.djs-element.ant-anomaly .djs-visual > :first-child) {
  stroke: var(--ant-status-attention) !important;
  stroke-width: 3px !important;
  stroke-dasharray: 6 3;
}

.canvas :deep(.djs-element.ant-bottleneck .djs-visual > :first-child) {
  stroke-dasharray: none !important;
  stroke: var(--ant-status-danger) !important;
  stroke-width: 4px !important;
}

.canvas :deep(.djs-element.ant-data-gap .djs-visual > :first-child) {
  fill: var(--ant-border) !important;
}

/* Слой «нормы» (FR-156): узел с нормативной опорой — пунктирная обводка акцентом. */
.canvas :deep(.djs-element.ant-norm .djs-visual > :first-child) {
  stroke: var(--ant-accent) !important;
  stroke-dasharray: 6 3;
  stroke-width: 3px !important;
}

.canvas :deep(.djs-element.ant-selected .djs-visual > :first-child) {
  stroke: var(--ant-accent) !important;
  stroke-width: 3px !important;
}
</style>
