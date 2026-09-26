<script setup lang="ts">
/**
 * Реестр документов — раздел «Документы» столов технолога, начальника ОТК,
 * руководителя производства, администратора и аудитора ИБ (FR-65, FR-66,
 * FR-139; AD-12, AD-43). Все документы, собранные из журнала: вид, номер,
 * объект (изделие, версия процесса, несоответствие, партия, пост и смена),
 * состояние (черновик, на подписи, на бумаге, возвращён, подписан,
 * аннулирован), кто должен подписать сейчас, дата. Отбор — по процессу,
 * изделию, виду, состоянию и поиск; отбор делает сервер
 * (`documents.document.list` с параметрами). Щелчок по строке — правое окно
 * документа (Д-70, `?open=document:‹id›`): содержимое, маршрут подписей,
 * версии и кнопки по правам.
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NInput, NSelect } from 'naive-ui'
import { useProcesses } from '@/entities/live-map'
import { useDrillDown } from '@/features/drill-down'
import { statusPalette } from '@/shared/api/generated/statuses'
import type { WidgetProps } from '@/shared/config/widget'
import { naiveSizeOf } from '@/shared/config/widget'
import { DataTable, EmptyState, ToolBar, WidgetFrame } from '@/shared/ui'
import { useDocumentRegistry } from '../model/api'
import {
  DOC_STATES,
  STATE_TONE,
  countByState,
  docState,
  itemLabel,
  itemsOf,
  kindKey,
  subjectKey,
  templateId,
  templatesOf,
  type DocState,
  type RegistryFilter,
} from '../model/registry'

const props = defineProps<WidgetProps>()
const { t, te, d } = useI18n()
const drill = useDrillDown()

const processId = ref<string | null>(null)
const itemId = ref<string | null>(null)
const template = ref<string | null>(null)
const state = ref<DocState | null>(null)
const search = ref('')

const filter = computed<RegistryFilter>(() => ({
  ...(processId.value ? { process_id: processId.value } : {}),
  ...(itemId.value ? { item_id: itemId.value } : {}),
  ...(template.value ? { template: template.value } : {}),
  ...(state.value ? { state: state.value } : {}),
  ...(search.value.trim() ? { q: search.value.trim() } : {}),
}))

// Весь реестр — для вариантов отбора и счётчиков; отобранный — для таблицы.
const all = useDocumentRegistry({})
const list = useDocumentRegistry(filter)
const processesQ = useProcesses()

const allRows = computed(() => all.rows.value ?? [])
const rows = computed(() => list.rows.value ?? [])
const counts = computed(() => countByState(allRows.value))

const kindLabel = (ref: string): string => (te(kindKey(ref)) ? t(kindKey(ref)) : templateId(ref))
const subjectLabel = (entity: string): string => (te(subjectKey(entity)) ? t(subjectKey(entity)) : entity)

const processOptions = computed(() => (processesQ.data.value ?? []).map((p) => ({ label: p.name, value: p.process_id })))
const itemOptions = computed(() => itemsOf(allRows.value).map((i) => ({ label: i.label, value: i.id })))
const kindOptions = computed(() =>
  templatesOf(allRows.value)
    .map((id) => ({ label: kindLabel(id), value: id }))
    .sort((a, b) => a.label.localeCompare(b.label, 'ru')),
)

const size = computed(() => naiveSizeOf(props.density))
const filtered = computed(() => Object.keys(filter.value).length > 0)
function reset(): void {
  processId.value = itemId.value = template.value = state.value = null
  search.value = ''
}

const open = (id: string) => drill.open({ entity: 'document', id })
const when = (iso?: string | null) => (iso ? d(new Date(iso), 'dateTime') : '—')
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="list.mode.value ?? all.mode.value"
    :loading="list.query.isPending.value && !list.rows.value"
    :error="list.rows.value ? undefined : list.query.error.value ?? undefined"
    :empty="false"
    :data-widget="widgetId"
  >
    <div class="registry ant-box">
      <div class="states ant-box" role="group" :aria-label="t('docRegistry.filters.state')">
        <button type="button" class="state-chip" :aria-pressed="!state" data-state="all" @click="state = null">
          <span class="ant-ellipsis">{{ t('docRegistry.filters.allStates') }}</span>
          <span class="count">{{ allRows.length }}</span>
        </button>
        <button
          v-for="s in DOC_STATES"
          :key="s"
          type="button"
          class="state-chip"
          :aria-pressed="state === s"
          :data-state="s"
          @click="state = state === s ? null : s"
        >
          <span class="dot" :style="{ background: statusPalette[STATE_TONE[s]] }" aria-hidden="true" />
          <span class="ant-ellipsis">{{ t(`docRegistry.states.${s}`) }}</span>
          <span class="count">{{ counts[s] }}</span>
        </button>
      </div>

      <ToolBar>
        <NSelect
          v-model:value="processId"
          class="f-process"
          :size="size"
          clearable
          :options="processOptions"
          :placeholder="t('docRegistry.filters.process')"
          data-filter="process"
        />
        <NSelect
          v-model:value="itemId"
          class="f-item"
          :size="size"
          clearable
          filterable
          :options="itemOptions"
          :placeholder="t('docRegistry.filters.item')"
          data-filter="item"
        />
        <NSelect
          v-model:value="template"
          class="f-kind"
          :size="size"
          clearable
          filterable
          :options="kindOptions"
          :placeholder="t('docRegistry.filters.kind')"
          data-filter="kind"
        />
        <NInput v-model:value="search" class="f-search" :size="size" clearable :placeholder="t('docRegistry.filters.search')" data-filter="search" />
        <template #end>
          <NButton v-if="filtered" :size="size" quaternary data-action="reset" @click="reset">{{ t('docRegistry.filters.reset') }}</NButton>
          <span class="muted total">{{ t('docRegistry.total', { n: rows.length }) }}</span>
        </template>
      </ToolBar>

      <EmptyState v-if="list.rows.value && !rows.length" compact :title="t(filtered ? 'docRegistry.emptyFiltered' : 'docRegistry.empty')" />
      <DataTable v-else-if="rows.length" :caption="t('docRegistry.title')">
        <thead>
          <tr>
            <th scope="col">{{ t('docRegistry.columns.document') }}</th>
            <th scope="col">{{ t('docRegistry.columns.number') }}</th>
            <th scope="col">{{ t('docRegistry.columns.subject') }}</th>
            <th scope="col">{{ t('docRegistry.columns.state') }}</th>
            <th scope="col">{{ t('docRegistry.columns.awaiting') }}</th>
            <th scope="col">{{ t('docRegistry.columns.date') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="r in rows"
            :key="r.document_id"
            class="row"
            tabindex="0"
            :data-document="r.document_id"
            :data-state="docState(r)"
            @click="open(r.document_id)"
            @keydown.enter="open(r.document_id)"
          >
            <td class="ant-box">
              <span class="title ant-clamp-2" :title="r.title">{{ r.title }}</span>
              <span class="muted ant-ellipsis kind">{{ kindLabel(r.template) }}</span>
            </td>
            <td class="ant-box">
              <span class="ant-mono ant-wrap">{{ r.document_id }}</span>
              <span v-if="r.version > 1" class="muted">{{ t('docRegistry.version', { n: r.version }) }}</span>
            </td>
            <td class="ant-box">
              <span class="muted ant-ellipsis">{{ subjectLabel(r.subject.entity) }}</span>
              <span class="ant-wrap">{{ r.subject_label || (r.subject.entity === 'item' ? itemLabel(r.subject.id) : r.subject.id) }}</span>
            </td>
            <td class="ant-box">
              <span class="state-tag" :data-tone="STATE_TONE[docState(r)]">
                <span class="dot" :style="{ background: statusPalette[STATE_TONE[docState(r)]] }" aria-hidden="true" />
                <span class="ant-ellipsis">{{ t(`docRegistry.states.${docState(r)}`) }}</span>
              </span>
              <span v-if="r.stages_total" class="muted progress">{{ t('docRegistry.progress', { done: r.stages_done ?? 0, total: r.stages_total }) }}</span>
            </td>
            <td class="ant-box">
              <template v-if="r.awaiting">
                <span class="ant-wrap">{{ r.awaiting.title }}</span>
                <span v-if="r.awaiting.candidates.length" class="muted ant-wrap">{{ r.awaiting.candidates.join(', ') }}</span>
              </template>
              <span v-else class="muted">—</span>
            </td>
            <td class="ant-box">
              <span class="ant-wrap">{{ when(r.updated_at ?? r.closed_at ?? r.drafted_at) }}</span>
            </td>
          </tr>
        </tbody>
      </DataTable>
    </div>
  </WidgetFrame>
</template>

<style scoped>
.registry {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-3);
}

.states {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-2);
}

.state-chip {
  display: inline-flex;
  align-items: center;
  gap: var(--ant-space-2);
  min-width: 0;
  max-width: 100%;
  padding: var(--ant-space-1) var(--ant-space-3);
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-pill);
  background: var(--ant-surface);
  color: var(--ant-text);
  font: inherit;
  font-size: var(--ant-fs-meta);
  cursor: pointer;
}

.state-chip[aria-pressed='true'] {
  border-color: var(--ant-accent);
  background: var(--ant-accent-soft);
}

.count {
  color: var(--ant-text-3);
  font-variant-numeric: tabular-nums;
}

.f-process {
  width: 16rem;
}

.f-item,
.f-kind {
  width: 14rem;
}

.f-search {
  width: 16rem;
}

.total {
  font-size: var(--ant-fs-meta);
}

.row {
  cursor: pointer;
}

.row:hover td {
  background: var(--ant-surface-hover);
}

td > span {
  display: block;
}

.muted {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.title {
  font-weight: var(--ant-fw-bold);
}

.state-tag {
  display: inline-flex !important;
  align-items: center;
  gap: var(--ant-space-2);
  max-width: 100%;
  padding: 0 var(--ant-space-2);
  border-radius: var(--ant-radius-pill);
  background: var(--ant-surface-subtle);
}

.dot {
  flex: none;
  width: 8px;
  height: 8px;
  border-radius: 50%;
}
</style>
