<script setup lang="ts">
/**
 * Панель наших свойств элемента (FR-25, FR-12, FR-156): название, описание
 * элемента (`bpmn:documentation` — текст карточки узла, FR-154) и группы
 * расширения `urn:ant:bpmn-ext:1` по провайдеру model/properties.ts. У
 * черновика поля правятся командами модельера; у остальных версий — только
 * просмотр.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { codeToKey } from '@/shared/i18n'
import { NButton, NCheckbox, NInput, NInputNumber, NSelect } from 'naive-ui'
import { EmptyState, FormField, SectionPanel } from '@/shared/ui'
import type { EditorElement, ElementEditor } from '../model/editor'
import { documentationOf, entriesOf, fieldValue, PANEL_GROUPS, type FieldValue, type PanelField, type PanelGroup } from '../model/properties'

const props = defineProps<{
  element: EditorElement | null
  /** Правка; null — только просмотр. */
  editor: ElementEditor | null
  /** Счётчик изменений модельера — перечитать значения. */
  revision: number
}>()
const emit = defineEmits<{ close: [] }>()
const { t, te } = useI18n()

/** Вид элемента BPMN словами: bpmn:Task → «Операция»; нет перевода — тип как есть. */
const typeText = computed(() => {
  const type = props.element?.type ?? ''
  const key = `processEditor.panel.types.${type.replace(/^bpmn:/, '')}`
  return te(key) ? t(key) : type
})

/** Значения списков — словами из словарей версии процесса и контроля (UI-17); нет перевода — код. */
const VALUE_DICT: Record<string, string> = {
  stepKind: 'process.values.stepKind',
  reworkLimitScope: 'process.values.reworkLimitScope',
  erpAction: 'process.values.erpAction',
  timerScope: 'process.values.timerScope',
  outcome: 'process.values.outcome',
  method: 'inspection.method',
  phase: 'inspection.phase',
}
function optionText(field: string, code: string): string {
  const dict = VALUE_DICT[field]
  const key = dict ? `${dict}.${codeToKey(code)}` : ''
  return key && te(key) ? t(key) : code
}

const bo = computed(() => {
  void props.revision
  return props.element?.businessObject ?? null
})
const readonly = computed(() => !props.editor)

/** Группы, которые показать: заполненные — всегда, пустые — только в черновике. */
const groups = computed(() => {
  void props.revision
  const b = bo.value
  if (!b) return []
  return PANEL_GROUPS.map((g) => ({ g, entries: entriesOf(b, g.type) })).filter((x) => !readonly.value || x.entries.length > 0)
})

const optionsOf = (f: PanelField) => (f.options ?? []).map((v) => ({ value: v, label: optionText(f.name, v) }))

function set(g: PanelGroup, index: number, f: PanelField, raw: FieldValue): void {
  if (!props.editor || !props.element) return
  props.editor.setField(props.element, g.type, index, f.name, raw === null || raw === '' ? undefined : raw)
}
</script>

<template>
  <div class="panel ant-box" data-testid="properties-panel">
    <EmptyState v-if="!element || !bo" compact :title="t('processEditor.panel.noSelection')" />
    <template v-else>
      <p v-if="readonly" class="note">{{ t('processEditor.panel.readonly') }}</p>
      <header class="head">
        <p class="type ant-wrap" :title="`${element.type} · ${element.id}`" data-testid="element-type">{{ typeText }}</p>
        <button type="button" class="close" :aria-label="t('common.actions.close')" data-testid="panel-close" @click="emit('close')">×</button>
      </header>
      <SectionPanel variant="plain" :title="t('processEditor.panel.element')">
        <FormField :label="t('processEditor.panel.name')">
          <NInput
            :value="(bo.name as string | undefined) ?? ''"
            :disabled="readonly"
            size="small"
            data-field="name"
            @update:value="(v: string) => editor && editor.setName(element!, v)"
          />
        </FormField>
        <FormField :label="t('processEditor.panel.documentation')">
          <NInput
            type="textarea"
            :value="documentationOf(bo)"
            :disabled="readonly"
            :autosize="{ minRows: 3, maxRows: 10 }"
            size="small"
            data-field="documentation"
            @update:value="(v: string) => editor && editor.setDocumentation(element!, v)"
          />
        </FormField>
      </SectionPanel>

      <SectionPanel v-for="{ g, entries } in groups" :key="g.type" variant="subtle" :title="t(`processEditor.panel.groups.${g.type}`)" :data-group="g.type">
        <div v-for="(entry, idx) in entries.length ? entries : [null]" :key="idx" class="entry ant-box">
          <FormField v-for="f in g.fields" :key="f.name" :label="t(`processEditor.panel.fields.${f.name}`)">
            <NCheckbox
              v-if="f.kind === 'boolean'"
              :checked="entry ? fieldValue(entry, f) === true : false"
              :disabled="readonly"
              :data-field="f.name"
              @update:checked="(v: boolean) => set(g, idx, f, v)"
            />
            <NInputNumber
              v-else-if="f.kind === 'integer'"
              :value="entry ? (fieldValue(entry, f) as number | null) : null"
              :disabled="readonly"
              :precision="0"
              size="small"
              clearable
              :data-field="f.name"
              @update:value="(v: number | null) => set(g, idx, f, v)"
            />
            <NSelect
              v-else-if="f.options"
              :value="entry ? (fieldValue(entry, f) as string | null) : null"
              :options="optionsOf(f)"
              :disabled="readonly"
              size="small"
              clearable
              :data-field="f.name"
              @update:value="(v: string | null) => set(g, idx, f, v)"
            />
            <NInput
              v-else
              :type="f.kind === 'text' ? 'textarea' : 'text'"
              :value="entry ? ((fieldValue(entry, f) as string | null) ?? '') : ''"
              :disabled="readonly"
              :autosize="f.kind === 'text' ? { minRows: 2, maxRows: 6 } : undefined"
              size="small"
              :data-field="f.name"
              @change="(v: string) => set(g, idx, f, v)"
            />
          </FormField>
          <NButton v-if="!readonly && g.many && entry" size="tiny" quaternary @click="editor!.removeEntry(element!, g.type, idx)">
            {{ t('processEditor.panel.remove') }}
          </NButton>
        </div>
        <NButton v-if="!readonly && g.many && entries.length" size="tiny" secondary :data-add="g.type" @click="editor!.addEntry(element!, g.type)">
          {{ t('processEditor.panel.add') }}
        </NButton>
      </SectionPanel>
    </template>
  </div>
</template>

<style scoped>
.panel {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-3);
  min-width: 0;
}

.head {
  display: flex;
  gap: var(--ant-space-2);
  align-items: flex-start;
  justify-content: space-between;
}

.type {
  margin: 0;
  font-size: var(--ant-fs-title);
  font-weight: var(--ant-fw-bold);
}

.close {
  flex: none;
  width: 28px;
  height: 28px;
  border: 0;
  border-radius: var(--ant-radius-sm);
  background: none;
  color: var(--ant-text-2);
  font-size: 20px;
  line-height: 1;
  cursor: pointer;
}

.close:hover {
  background: var(--ant-surface-hover);
}

.note,
.meta {
  margin: 0;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.entry {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
  padding-bottom: var(--ant-space-2);
  border-bottom: 1px solid var(--ant-border);
}

.entry:last-of-type {
  border-bottom: 0;
}
</style>
