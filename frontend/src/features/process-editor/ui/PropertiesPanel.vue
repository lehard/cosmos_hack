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
const { t } = useI18n()

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

const optionsOf = (f: PanelField) => (f.options ?? []).map((v) => ({ value: v, label: v }))

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
      <SectionPanel variant="plain" :title="t('processEditor.panel.element')">
        <p class="meta ant-wrap">{{ element.type }} · {{ element.id }}</p>
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
