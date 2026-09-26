<script setup lang="ts">
/**
 * Модельер процесса (FR-25, AD-17): bpmn-js Modeler с расширением
 * `urn:ant:bpmn-ext:1` (дескриптор moddle — model/ant-moddle.json, копия
 * контракта). Черновик правится (`editable`), остальные версии — только
 * просмотр той же схемы. Итог — `saveXML()`: BPMN 2.0 XML целиком (схема,
 * свойства, BPMNDI, documentation) для операции `process.version.draft`.
 * Водяной знак bpmn.io остаётся видимым (AD-21).
 *
 * Два вида (UI-34):
 * - `preview` — схема в окне записи: без палитры и свойств, только чтобы увидеть
 *   процесс целиком; щелчок — «открыть на весь экран»;
 * - `workspace` — рабочее место на всю высоту родителя: кнопки масштаба
 *   («вся схема», 100 %, ±), палитра слева, свойства элемента справа — только
 *   когда элемент выбран. Схема вписывается в окно, когда холст получил размер,
 *   и заново при изменении размера — пока человек сам не сдвинул или не
 *   приблизил её.
 */
import { markRaw, nextTick, onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Modeler from 'bpmn-js/lib/Modeler'
import 'bpmn-js/dist/assets/diagram-js.css'
import 'bpmn-js/dist/assets/bpmn-js.css'
import 'bpmn-js/dist/assets/bpmn-font/css/bpmn-embedded.css'
import descriptor from '../model/ant-moddle.json'
import { createEditor, type EditorElement, type ElementEditor, type ModelerServices } from '../model/editor'
import PropertiesPanel from './PropertiesPanel.vue'

const props = withDefaults(
  defineProps<{
    /** BPMN 2.0 XML версии. */
    xml: string
    /** Черновик — правка разрешена. */
    editable?: boolean
    /** Вид: схема в окне записи или рабочее место. */
    view?: 'preview' | 'workspace'
  }>(),
  { editable: false, view: 'workspace' },
)
const emit = defineEmits<{
  /** Схема изменена после открытия. */
  dirty: []
  'import-error': [error: Error]
  /** Просмотр: открыть на весь экран. */
  expand: []
}>()
const { t, n } = useI18n()

const container = ref<HTMLElement | null>(null)
let modeler: Modeler | null = null
const editor = shallowRef<ElementEditor | null>(null)
const selected = shallowRef<EditorElement | null>(null)
const revision = ref(0)
const failed = ref(false)
const zoomLevel = ref(1)
/** Человек сам двигал или приближал схему — не вписывать заново. */
let userMoved = false

interface EventBusLike {
  on(e: string, cb: (ev: { newSelection?: EditorElement[]; elements?: EditorElement[]; viewbox?: { scale: number } }) => void): void
}
interface CanvasLike {
  zoom(v?: string | number, center?: 'auto' | { x: number; y: number }): number
  resized(): void
}
const canvas = (): CanvasLike | null => (modeler ? (modeler.get('canvas') as CanvasLike) : null)

/** Вписать всю схему в окно. */
function fit(): void {
  const c = canvas()
  if (!c || !container.value || container.value.clientWidth === 0 || container.value.clientHeight === 0) return
  c.resized()
  c.zoom('fit-viewport', 'auto')
  zoomLevel.value = c.zoom()
}
function zoomBy(k: number): void {
  const c = canvas()
  if (!c) return
  userMoved = true
  c.zoom(Math.min(Math.max(c.zoom() * k, 0.2), 4), 'auto')
  zoomLevel.value = c.zoom()
}
function zoomActual(): void {
  const c = canvas()
  if (!c) return
  userMoved = true
  c.zoom(1, 'auto')
  zoomLevel.value = 1
}
function fitAll(): void {
  userMoved = false
  fit()
}

let importSeq = 0
async function load(xml: string): Promise<void> {
  if (!modeler) return
  const seq = ++importSeq
  failed.value = false
  try {
    await modeler.importXML(xml)
    if (seq !== importSeq) return
    selected.value = null
    revision.value++
    userMoved = false
    await nextTick()
    // Холст мог ещё не получить размер (окно выезжает) — вписываем после кадра, и ResizeObserver повторит.
    requestAnimationFrame(fit)
  } catch (e) {
    failed.value = true
    emit('import-error', e instanceof Error ? e : new Error(String(e)))
  }
}

let observer: ResizeObserver | null = null

onMounted(() => {
  modeler = markRaw(new Modeler({ container: container.value!, moddleExtensions: { ant: descriptor } }))
  const bus = modeler.get('eventBus') as EventBusLike
  bus.on('selection.changed', (ev) => {
    selected.value = ev.newSelection?.[0] ?? null
    revision.value++
  })
  bus.on('elements.changed', () => {
    revision.value++
    emit('dirty')
  })
  bus.on('canvas.viewbox.changed', (ev) => {
    if (ev.viewbox) zoomLevel.value = ev.viewbox.scale
  })
  editor.value = createEditor({ modeling: modeler.get('modeling') as ModelerServices['modeling'], moddle: modeler.get('moddle') as ModelerServices['moddle'] })
  if (typeof ResizeObserver !== 'undefined') {
    observer = new ResizeObserver(() => {
      canvas()?.resized()
      if (!userMoved) fit()
    })
    observer.observe(container.value!)
  }
  void load(props.xml)
})

watch(
  () => props.xml,
  (x) => void load(x),
)

onBeforeUnmount(() => {
  observer?.disconnect()
  modeler?.destroy()
  modeler = null
})

/** Колесо, перетаскивание — человек сам ведёт схему. */
const markMoved = () => (userMoved = true)

/** BPMN XML схемы как сейчас в модельере (с раскладкой BPMNDI). */
async function saveXML(): Promise<string> {
  if (!modeler) throw new Error('модельер не открыт')
  const { xml } = await modeler.saveXML({ format: true })
  if (!xml) throw new Error('пустая схема')
  return xml
}

defineExpose({ saveXML, fit: fitAll })
</script>

<template>
  <div class="modeler" :class="[`view-${view}`, { readonly: !editable || view === 'preview', 'has-side': view === 'workspace' && selected }]" data-testid="process-modeler">
    <div class="canvas-wrap">
      <p v-if="failed" class="error" role="alert">{{ t('errors.loadFailed') }}</p>
      <div ref="container" class="canvas" @wheel.passive="markMoved" @pointerdown="markMoved" />

      <!-- Просмотр в окне: вся схема — щелчком на весь экран. -->
      <button v-if="view === 'preview'" type="button" class="expand-cover" :aria-label="t('processEditor.workspace.openFull')" data-testid="open-full" @click="emit('expand')">
        <span class="expand-label">{{ t('processEditor.workspace.openFull') }}</span>
      </button>

      <!-- Масштаб: вся схема, 100 %, ближе, дальше. -->
      <div v-else class="zoom" role="toolbar" :aria-label="t('processEditor.workspace.zoom')">
        <button type="button" data-testid="zoom-fit" @click="fitAll">{{ t('processEditor.workspace.fit') }}</button>
        <button type="button" data-testid="zoom-actual" @click="zoomActual">{{ n(zoomLevel, 'percent') }}</button>
        <button type="button" :aria-label="t('processEditor.workspace.zoomOut')" data-testid="zoom-out" @click="zoomBy(1 / 1.25)">−</button>
        <button type="button" :aria-label="t('processEditor.workspace.zoomIn')" data-testid="zoom-in" @click="zoomBy(1.25)">+</button>
      </div>
      <p v-if="view === 'workspace'" class="hint">{{ t(editable ? 'processEditor.workspace.hintEdit' : 'processEditor.workspace.hintView') }}</p>
    </div>

    <aside v-if="view === 'workspace' && selected" class="side">
      <PropertiesPanel :element="selected" :editor="editable ? editor : null" :revision="revision" @close="selected = null" />
    </aside>
  </div>
</template>

<style scoped>
.modeler {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  height: 100%;
  min-height: 0;
}

.modeler.has-side {
  grid-template-columns: minmax(0, 1fr) minmax(300px, 380px);
}

.view-preview {
  height: 320px;
}

.canvas-wrap {
  position: relative;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  background: var(--ant-surface);
}

.view-preview .canvas-wrap {
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-md);
}

.canvas {
  position: absolute;
  inset: 0;
}

/* Только просмотр: палитра и контекстное меню правки скрыты. */
.readonly :deep(.djs-palette),
.readonly :deep(.djs-context-pad) {
  display: none;
}

/* Палитра — у левого края рабочего места, не поверх схемы в узком окне. */
.view-workspace :deep(.djs-palette) {
  top: var(--ant-space-3);
  left: var(--ant-space-3);
  border-color: var(--ant-border);
  border-radius: var(--ant-radius-md);
  background: var(--ant-surface);
  box-shadow: var(--ant-shadow-sm);
}

.expand-cover {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: flex-end;
  justify-content: center;
  padding: var(--ant-space-3);
  border: 0;
  background: transparent;
  cursor: zoom-in;
}

.expand-label {
  padding: var(--ant-space-1) var(--ant-space-3);
  border-radius: var(--ant-radius-pill);
  background: var(--ant-surface);
  box-shadow: var(--ant-shadow-sm);
  color: var(--ant-accent);
  font-weight: var(--ant-fw-bold);
}

.zoom {
  position: absolute;
  right: var(--ant-space-3);
  bottom: var(--ant-space-3);
  display: flex;
  overflow: hidden;
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-md);
  background: var(--ant-surface);
  box-shadow: var(--ant-shadow-sm);
}

.zoom button {
  display: inline-flex;
  align-items: center;
  min-width: 36px;
  height: 32px;
  padding: 0 var(--ant-space-2);
  border: 0;
  border-left: 1px solid var(--ant-border);
  background: none;
  color: var(--ant-text);
  font: inherit;
  font-variant-numeric: tabular-nums;
  cursor: pointer;
}

.zoom button:first-child {
  border-left: 0;
}

.zoom button:hover {
  background: var(--ant-surface-hover);
}

.hint {
  position: absolute;
  bottom: var(--ant-space-3);
  left: var(--ant-space-3);
  max-width: calc(100% - 320px);
  margin: 0;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.side {
  min-height: 0;
  overflow: auto;
  padding: var(--ant-space-4);
  border-left: 1px solid var(--ant-border);
  background: var(--ant-surface);
}

.error {
  position: absolute;
  z-index: 1;
  margin: var(--ant-space-3);
  color: var(--ant-status-danger-text);
}
</style>
