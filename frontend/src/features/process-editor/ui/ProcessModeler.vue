<script setup lang="ts">
/**
 * Модельер процесса (FR-25, AD-17): bpmn-js Modeler с расширением
 * `urn:ant:bpmn-ext:1` (дескриптор moddle — model/ant-moddle.json, копия
 * контракта) и справа — панель наших свойств выбранного элемента. Черновик
 * правится (`editable`), остальные версии — только просмотр той же схемы.
 * Итог — `saveXML()`: BPMN 2.0 XML целиком (схема, свойства, BPMNDI,
 * documentation) для операции `process.version.draft`. Водяной знак bpmn.io
 * остаётся видимым (AD-21).
 */
import { markRaw, onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue'
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
  }>(),
  { editable: false },
)
const emit = defineEmits<{
  /** Схема изменена после открытия. */
  dirty: []
  'import-error': [error: Error]
}>()
const { t } = useI18n()

const container = ref<HTMLElement | null>(null)
let modeler: Modeler | null = null
const editor = shallowRef<ElementEditor | null>(null)
const selected = shallowRef<EditorElement | null>(null)
const revision = ref(0)
const failed = ref(false)

interface EventBusLike {
  on(e: string, cb: (ev: { newSelection?: EditorElement[]; elements?: EditorElement[] }) => void): void
}

let importSeq = 0
async function load(xml: string): Promise<void> {
  if (!modeler) return
  const seq = ++importSeq
  failed.value = false
  try {
    await modeler.importXML(xml)
    if (seq !== importSeq) return
    ;(modeler.get('canvas') as { zoom(v: string): void }).zoom('fit-viewport')
    selected.value = null
    revision.value++
  } catch (e) {
    failed.value = true
    emit('import-error', e instanceof Error ? e : new Error(String(e)))
  }
}

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
  editor.value = createEditor({ modeling: modeler.get('modeling') as ModelerServices['modeling'], moddle: modeler.get('moddle') as ModelerServices['moddle'] })
  void load(props.xml)
})

watch(
  () => props.xml,
  (x) => void load(x),
)

onBeforeUnmount(() => {
  modeler?.destroy()
  modeler = null
})

/** BPMN XML схемы как сейчас в модельере (с раскладкой BPMNDI). */
async function saveXML(): Promise<string> {
  if (!modeler) throw new Error('модельер не открыт')
  const { xml } = await modeler.saveXML({ format: true })
  if (!xml) throw new Error('пустая схема')
  return xml
}

defineExpose({ saveXML })
</script>

<template>
  <div class="modeler ant-box" :class="{ readonly: !editable }" data-testid="process-modeler">
    <div class="canvas-wrap ant-box">
      <p v-if="failed" class="error" role="alert">{{ t('errors.loadFailed') }}</p>
      <div ref="container" class="canvas" />
    </div>
    <aside class="side ant-box">
      <PropertiesPanel :element="selected" :editor="editable ? editor : null" :revision="revision" />
    </aside>
  </div>
</template>

<style scoped>
.modeler {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(260px, 340px);
  gap: var(--ant-space-3);
  min-height: 520px;
}

.canvas-wrap {
  position: relative;
  min-height: 520px;
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-md);
  background: var(--ant-surface);
  overflow: hidden;
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

.side {
  max-height: 70vh;
  overflow: auto;
}

.error {
  position: absolute;
  z-index: 1;
  margin: var(--ant-space-3);
  color: var(--ant-status-danger, var(--ant-text));
}

@media (max-width: 900px) {
  .modeler {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
