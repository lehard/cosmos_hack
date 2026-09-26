<script setup lang="ts">
/**
 * Рабочее место схемы процесса на весь экран (UI-34): правка черновика и
 * просмотр версии не в узком окне записи, а поверх всего экрана. Сверху —
 * процесс и версия, метка новой версии, «Скачать .bpmn», «Сохранить черновик»
 * (только при праве и в правке), «Закрыть»; ниже — схема на всю оставшуюся
 * высоту (высота задана явно: экран минус верхняя панель), масштаб и свойства
 * элемента — в модельере. Закрыть с несохранёнными правками — только после
 * подтверждения.
 */
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NInput, NModal, NPopconfirm } from 'naive-ui'
import ProcessModeler from './ProcessModeler.vue'

const props = withDefaults(
  defineProps<{
    show: boolean
    /** BPMN 2.0 XML версии. */
    xml: string
    /** Название процесса. */
    title: string
    /** Метка открытой версии и её статус словами. */
    versionLabel: string
    statusText: string
    /** Правка: схема правится, есть «Сохранить черновик». */
    editable: boolean
    /** Метка, под которой сохранится черновик. */
    label: string
    /** Сохранение идёт. */
    saving?: boolean
  }>(),
  { saving: false },
)
const emit = defineEmits<{
  close: []
  download: []
  save: [xml: string, label: string]
  'update:label': [label: string]
}>()

const { t } = useI18n()
const modeler = ref<InstanceType<typeof ProcessModeler> | null>(null)
const dirty = ref(false)

watch(
  () => [props.show, props.xml],
  () => (dirty.value = false),
)

async function save(): Promise<void> {
  if (!modeler.value) return
  emit('save', await modeler.value.saveXML(), props.label)
}
function close(): void {
  dirty.value = false
  emit('close')
}
</script>

<template>
  <NModal :show="show" :mask-closable="false" :close-on-esc="!dirty" :auto-focus="false" @esc="close" @update:show="(v: boolean) => !v && close()">
    <div class="workspace" role="dialog" :aria-label="title" data-testid="process-workspace">
      <header class="bar">
        <div class="what">
          <p class="kind">{{ t(editable ? 'processEditor.workspace.editing' : 'processEditor.workspace.viewing') }}</p>
          <h2 class="title ant-ellipsis" :title="`${title} · ${versionLabel}`">{{ title }} · {{ versionLabel }} <span class="status">{{ statusText }}</span></h2>
          <p v-if="editable" class="note ant-wrap">{{ t('processEditor.newDraftHint') }}</p>
        </div>
        <div class="controls">
          <label v-if="editable" class="label-field">
            <span>{{ t('processEditor.labelField') }}</span>
            <NInput :value="label" size="small" :maxlength="64" data-testid="workspace-label" @update:value="(v: string) => emit('update:label', v)" />
          </label>
          <NButton data-action="download" @click="emit('download')">{{ t('processEditor.actions.download') }}</NButton>
          <NButton v-if="editable" type="primary" :disabled="!dirty || !label.trim()" :loading="saving" data-action="saveDraft" @click="save">
            {{ t('processEditor.actions.saveDraft') }}
          </NButton>
          <NPopconfirm v-if="dirty" :positive-text="t('processEditor.workspace.discard')" :negative-text="t('processEditor.actions.cancel')" @positive-click="close">
            <template #trigger>
              <NButton data-action="close">{{ t('common.actions.close') }}</NButton>
            </template>
            {{ t('processEditor.workspace.unsaved') }}
          </NPopconfirm>
          <NButton v-else data-action="close" @click="close">{{ t('common.actions.close') }}</NButton>
        </div>
      </header>
      <div class="stage">
        <ProcessModeler ref="modeler" :xml="xml" :editable="editable" view="workspace" @dirty="dirty = editable" />
      </div>
    </div>
  </NModal>
</template>

<style scoped>
.workspace {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  width: 100vw;
  height: 100vh;
  background: var(--ant-bg-app);
}

.bar {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-3) var(--ant-space-6);
  align-items: center;
  justify-content: space-between;
  padding: var(--ant-space-3) var(--ant-space-6);
  border-bottom: 1px solid var(--ant-border);
  background: var(--ant-surface);
}

.what {
  display: flex;
  flex: 1 1 360px;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.kind,
.note {
  margin: 0;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.title {
  margin: 0;
  font-size: var(--ant-fs-title);
}

.status {
  color: var(--ant-accent);
  font-size: var(--ant-fs-meta);
  font-weight: normal;
}

.controls {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-2);
  align-items: flex-end;
}

.label-field {
  display: flex;
  flex-direction: column;
  gap: 2px;
  width: 140px;
  color: var(--ant-text-2);
  font-size: var(--ant-fs-meta);
}

.stage {
  min-height: 0;
}
</style>
