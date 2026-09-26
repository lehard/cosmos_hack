<script setup lang="ts">
/**
 * Окно процесса справа (Д-70; эпик 39, FR-22…FR-25, UJ-4). Без выбранной
 * версии — процесс: действующая версия, изделия в работе, список версий;
 * кнопки «Новая версия» (модельер на копии действующей) и «Загрузить .bpmn
 * как новую версию». С выбранной версией — вкладки «Схема и свойства»
 * (модельер bpmn-js с панелью наших свойств, черновик правится), «Отличия»
 * (читаемая разница с действующей, FR-24) и «Лист утверждения» (маршрут
 * кворума, FR-23); кнопки пути версии и «Скачать .bpmn» — в нижней панели.
 * Подписи кворума ставят подписанты в «Запросах решения» своих столов
 * (документ эпика 28); здесь — прогресс и ввод в действие.
 *
 * Окно — общий `RecordDrawer` (shared/ui, Д-70): заголовок, вкладки, тело,
 * нижняя панель действий.
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NInput, NSpin, useMessage } from 'naive-ui'
import { ProcessModeler } from '@/features/process-editor'
import { toDiffEntry, type ProcessDiffEntry } from '@/entities/process-version'
import { codeToKey } from '@/shared/i18n'
import { useProblemText } from '@/shared/i18n/problem'
import { problemOf } from '@/shared/api/problem'
import { useMomentStore } from '@/shared/model/moment'
import { DataTable, EmptyState, FormField, KeyValue, KeyValueList, RecordDrawer, SectionPanel, type RecordDrawerTab } from '@/shared/ui'
import { bpmnFileName, BpmnFileError, downloadText, readBpmnFile } from '../model/bpmn-file'
import { useCanOnVersion } from '../model/access'
import { actionsFor, nextLabel, type VersionAction } from '../model/lifecycle'
import { fetchBpmn, useApprovalDocument, useBpmn, useDiff, useLifecycle, useVersions, type ProcessSummary } from '../model/source'

const props = defineProps<{
  /** Открытый процесс; null — окно закрыто. */
  process: ProcessSummary | null
  /** Выбранная версия (из адреса); null — обзор процесса. */
  versionId: string | null
}>()
const emit = defineEmits<{ close: []; 'select-version': [id: string | null] }>()

const { t, d } = useI18n()
const canOnVersion = useCanOnVersion()
const moment = useMomentStore()
const message = useMessage()
const problemText = useProblemText()

type Tab = 'scheme' | 'diff' | 'approval'
const tab = ref<Tab>('scheme')
const editing = ref(false)
const dirty = ref(false)
const label = ref('')
const note = ref('')
const reason = ref('')
const asking = ref<VersionAction | null>(null)
const details = ref<string | null>(null)
const modeler = ref<InstanceType<typeof ProcessModeler> | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)

const processId = computed(() => props.process?.process_id ?? null)
const list = useVersions(processId, computed(() => Boolean(processId.value)))
const versions = computed(() => [...(list.versions.value ?? [])].sort((a, b) => b.created_at.localeCompare(a.created_at)))
const version = computed(() => versions.value.find((v) => v.version_id === props.versionId) ?? null)
const active = computed(() => versions.value.find((v) => v.status === 'active') ?? null)
const labels = computed(() => versions.value.map((v) => v.label))

const id = computed(() => version.value?.version_id ?? null)
const bpmn = useBpmn(id)
const diff = useDiff(computed(() => (version.value && version.value.status !== 'active' ? version.value.version_id : null)))
const doc = useApprovalDocument(computed(() => version.value?.approval_document_id ?? null))
const cmd = useLifecycle()

/** «Новая версия» из обзора процесса: открыть действующую сразу в правке. */
let pendingEdit = false
watch(
  () => [props.versionId, processId.value],
  () => {
    tab.value = 'scheme'
    dirty.value = false
    asking.value = null
    details.value = null
    note.value = ''
    reason.value = ''
    if (!pendingEdit) editing.value = false
    pendingEdit = false
    label.value = version.value?.status === 'draft' ? version.value.label : nextLabel(labels.value)
  },
)
const editable = computed(() => !moment.isReplay && (version.value?.status === 'draft' || editing.value))
const xml = computed(() => bpmn.data.value?.data.bpmn_xml ?? '')
const entries = computed<ProcessDiffEntry[]>(() => (diff.data.value?.data.entries ?? []).map(toDiffEntry).filter((e): e is ProcessDiffEntry => e !== null))
const document = computed(() => doc.data.value?.data ?? null)
const routeClosedId = computed(() => document.value?.route_closed_event_id ?? null)
const can = (a: string) => !moment.isReplay && canOnVersion(a)

const buttons = computed(() =>
  version.value && !moment.isReplay
    ? actionsFor(version.value.status, can, { dirty: dirty.value, routeClosed: Boolean(routeClosedId.value), editing: editing.value })
    : [],
)

const statusText = (s: string) => t(`statuses.processVersion.${codeToKey(s)}`)
const when = (x?: string | null) => (x ? d(new Date(x), 'dateTime') : '—')

function diffText(e: ProcessDiffEntry): string {
  switch (e.kind) {
    case 'elementAdded':
    case 'elementRemoved':
      return t(`process.diff.${e.kind}`, { element: e.element })
    case 'presentationPointAdded':
      return t('process.diff.presentationPointAdded', { step: e.step })
    case 'thresholdChanged':
      return t('process.diff.thresholdChanged', { defectType: e.defectType, from: e.from ?? '—', to: e.to ?? '—' })
    case 'propertyChanged':
      return t('process.diff.propertyChanged', { element: e.element, property: e.property, from: e.from ?? '—', to: e.to ?? '—' })
  }
  return ''
}

/** Ошибка команды: текст по коду и, у отказа загрузки (FR-13), перечень нарушений. */
function fail(e: unknown): void {
  message.error(problemText(e))
  const p = problemOf(e)
  details.value = p?.detail && String(p.code ?? '').startsWith('process.') ? String(p.detail) : null
}

/** После сохранения черновика — открыть самый новый черновик процесса. */
async function openNewestDraft(): Promise<void> {
  const r = await list.query.refetch()
  const drafts = (r.data?.data.items ?? []).filter((v) => v.status === 'draft').sort((a, b) => b.created_at.localeCompare(a.created_at))
  if (drafts[0]) emit('select-version', drafts[0].version_id)
}

async function saveDraft(body: string, lbl: string): Promise<void> {
  await cmd.mutateAsync({ kind: 'draft', label: lbl, base_version_id: active.value?.version_id, bpmn_xml: body })
  message.success(t('processEditor.done.saved'))
  editing.value = false
  dirty.value = false
  details.value = null
  await openNewestDraft()
}

function newVersion(): void {
  if (!active.value) return
  pendingEdit = true
  editing.value = true
  emit('select-version', active.value.version_id)
}

async function onFile(ev: Event): Promise<void> {
  const input = ev.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file || !processId.value) return
  try {
    const text = await file.text()
    const info = readBpmnFile(text)
    if (info.processId !== processId.value) {
      message.error(t('processEditor.fileErrors.otherProcess', { id: info.processId }))
      return
    }
    await saveDraft(text, nextLabel(labels.value))
  } catch (e) {
    if (e instanceof BpmnFileError) message.error(t(`processEditor.fileErrors.${e.key}`))
    else fail(e)
  }
}

async function download(): Promise<void> {
  if (!version.value || !processId.value) return
  try {
    downloadText(await fetchBpmn(version.value.version_id), bpmnFileName(processId.value, version.value.label))
  } catch (e) {
    fail(e)
  }
}

async function run(action: VersionAction): Promise<void> {
  const v = version.value
  if (!v) return
  try {
    switch (action) {
      case 'newDraft':
        editing.value = true
        dirty.value = false
        label.value = nextLabel(labels.value)
        tab.value = 'scheme'
        return
      case 'saveDraft':
        await saveDraft(await modeler.value!.saveXML(), label.value.trim() || nextLabel(labels.value))
        return
      case 'submit':
      case 'retire':
        if (asking.value !== action) {
          asking.value = action
          return
        }
        if (action === 'submit') {
          await cmd.mutateAsync({ kind: 'submit', version_id: v.version_id, note: note.value })
          message.success(t('processEditor.done.submitted'))
          tab.value = 'approval'
        } else {
          await cmd.mutateAsync({ kind: 'retire', version_id: v.version_id, reason: reason.value })
          message.success(t('processEditor.done.retired'))
        }
        asking.value = null
        return
      case 'activate':
        if (!routeClosedId.value) return
        await cmd.mutateAsync({ kind: 'activate', version_id: v.version_id, route_closed_event_id: routeClosedId.value })
        message.success(t('processEditor.done.activated'))
        return
    }
  } catch (e) {
    fail(e)
  }
}

const primary = (a: VersionAction) => a === 'submit' || a === 'activate' || a === 'saveDraft'
const tabs = computed<RecordDrawerTab[]>(() =>
  version.value ? (['scheme', 'diff', 'approval'] as Tab[]).map((id) => ({ id, label: t(`processEditor.tabs.${id}`) })) : [],
)
</script>

<template>
  <RecordDrawer
    :show="Boolean(process)"
    :kind-label="version ? t('processEditor.drawerKind') : t('processEditor.registry.kind')"
    :number="process ? (version ? `${process.name} · ${version.label}` : process.name) : ''"
    :subtitle="process?.process_id ?? ''"
    :tabs="tabs"
    :tab="tab"
    data-record="process"
    @update:tab="(x: string) => (tab = x as Tab)"
    @close="emit('close')"
  >
    <template v-if="version" #status>
      <span class="status" :data-status="version.status">{{ statusText(version.status) }}</span>
    </template>
    <template v-if="version" #links>
      <button type="button" class="back" data-testid="back" @click="emit('select-version', null)">← {{ t('processEditor.registry.backToVersions') }}</button>
    </template>

    <template v-if="process">
      <!-- Обзор процесса: версии -->
      <div v-if="!version" class="body ant-box" data-testid="process-overview">
        <KeyValueList>
          <KeyValue :label="t('processEditor.registry.columns.active')" :value="process.active_version?.label ?? t('processEditor.registry.noActive')" />
          <KeyValue :label="t('processEditor.registry.columns.items')" :value="process.items_in_work" />
        </KeyValueList>
        <div v-if="list.query.isLoading.value" class="center"><NSpin size="small" /></div>
        <DataTable v-else :caption="t('processEditor.versions')">
          <thead>
            <tr>
              <th scope="col">{{ t('processEditor.columns.label') }}</th>
              <th scope="col">{{ t('processEditor.columns.status') }}</th>
              <th scope="col">{{ t('processEditor.columns.created') }}</th>
              <th scope="col">{{ t('processEditor.columns.effective') }}</th>
              <th scope="col">{{ t('processEditor.columns.quorum') }}</th>
              <th scope="col" class="num">{{ t('processEditor.columns.items') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="v in versions"
              :key="v.version_id"
              class="row"
              tabindex="0"
              :data-version="v.version_id"
              :data-status="v.status"
              @click="emit('select-version', v.version_id)"
              @keydown.enter="emit('select-version', v.version_id)"
            >
              <td><strong>{{ v.label }}</strong></td>
              <td>{{ statusText(v.status) }}</td>
              <td>{{ when(v.created_at) }}</td>
              <td>{{ when(v.effective_from) }}</td>
              <td>{{ v.quorum ? `${v.quorum.have} / ${v.quorum.need}` : '—' }}</td>
              <td class="num">{{ v.items_in_work }}</td>
            </tr>
          </tbody>
        </DataTable>
        <p class="note">{{ t('processEditor.futureOnly') }}</p>
      </div>

      <!-- Версия -->
      <div v-else class="body ant-box" :data-tab="tab">
        <template v-if="tab === 'scheme'">
          <KeyValueList>
            <KeyValue :label="t('processEditor.columns.created')" :value="when(version.created_at)" />
            <KeyValue :label="t('processEditor.columns.effective')" :value="when(version.effective_from)" />
            <KeyValue :label="t('processEditor.hash')" :value="version.hash || '—'" mono />
          </KeyValueList>
          <p v-if="editing" class="note">{{ t('processEditor.newDraftHint') }}</p>
          <div v-if="bpmn.isLoading.value" class="center"><NSpin size="small" /></div>
          <ProcessModeler v-else-if="xml" ref="modeler" :xml="xml" :editable="editable" @dirty="dirty = editable" />
        </template>

        <SectionPanel
          v-else-if="tab === 'diff'"
          variant="plain"
          :title="active ? t('processEditor.diffAgainst', { label: active.label }) : t('process.diffTitle')"
        >
          <EmptyState v-if="!entries.length" compact :title="t('processEditor.noDiff')" />
          <ul v-else class="diff" data-testid="diff">
            <li v-for="(e, i) in entries" :key="i" :data-kind="e.kind" class="ant-wrap">{{ diffText(e) }}</li>
          </ul>
          <p class="note">{{ t('processEditor.futureOnly') }}</p>
        </SectionPanel>

        <template v-else>
          <EmptyState v-if="!version.approval_document_id" compact :title="t('processEditor.approval.none')" />
          <SectionPanel v-else variant="plain" :title="t('processEditor.approval.document', { id: version.approval_document_id })">
            <p v-if="version.quorum" class="strong">{{ t('processEditor.approval.open', { have: version.quorum.have, need: version.quorum.need }) }}</p>
            <p v-if="routeClosedId" class="strong" data-testid="route-closed">{{ t('processEditor.approval.closed') }}</p>
            <ol v-if="document?.route" class="route">
              <li v-for="st in document.route" :key="st.stage" :data-done="st.done">
                <strong class="ant-wrap">{{ st.title || st.authority_label }}</strong>
                <span v-if="st.signatures.some((s) => s.counted)" class="signed">
                  {{ t('processEditor.approval.signed', { who: st.signatures.filter((s) => s.counted).map((s) => s.signer_id).join(', ') }) }}
                </span>
                <span v-else class="waiting">{{ t('processEditor.approval.waiting') }}</span>
              </li>
            </ol>
            <p class="note">{{ t('processEditor.approval.signHint') }}</p>
          </SectionPanel>
        </template>
      </div>
    </template>

    <template #actions>
      <div class="actions ant-box" data-testid="drawer-actions">
        <SectionPanel v-if="details" variant="subtle" :title="t('processEditor.fileErrors.details')">
          <ul class="diff">
            <li v-for="(x, i) in details.split('; ')" :key="i" class="ant-wrap ant-mono">{{ x }}</li>
          </ul>
        </SectionPanel>
        <template v-if="version">
          <FormField v-if="editing || version.status === 'draft'" :label="t('processEditor.labelField')">
            <NInput v-model:value="label" size="small" :placeholder="nextLabel(labels)" :maxlength="64" />
          </FormField>
          <FormField v-if="asking === 'submit'" :label="t('processEditor.noteField')">
            <NInput v-model:value="note" type="textarea" size="small" :maxlength="1800" :autosize="{ minRows: 2, maxRows: 4 }" />
          </FormField>
          <FormField v-if="asking === 'retire'" :label="t('processEditor.reasonField')" required>
            <NInput v-model:value="reason" size="small" :maxlength="1000" />
          </FormField>
        </template>
        <div class="buttons">
          <template v-if="!version">
            <input ref="fileInput" type="file" accept=".bpmn,.xml,application/xml,text/xml" class="ant-sr-only" data-testid="upload-version" @change="onFile" />
            <NButton v-if="can('process.version.draft')" :disabled="cmd.isPending.value" data-action="uploadVersion" @click="fileInput?.click()">
              {{ t('processEditor.actions.uploadVersion') }}
            </NButton>
            <NButton v-if="can('process.version.draft') && active" type="primary" data-action="newVersion" @click="newVersion">
              {{ t('processEditor.actions.newVersion') }}
            </NButton>
          </template>
          <template v-else>
            <NButton data-action="download" @click="download">{{ t('processEditor.actions.download') }}</NButton>
            <NButton v-if="asking" quaternary @click="asking = null">{{ t('processEditor.actions.cancel') }}</NButton>
            <NButton
              v-for="b in buttons"
              :key="b.action"
              :type="primary(b.action) ? 'primary' : 'default'"
              :disabled="!b.enabled || cmd.isPending.value || (asking === 'retire' && b.action === 'retire' && !reason.trim())"
              :loading="cmd.isPending.value && (asking === b.action || b.action === 'saveDraft' || b.action === 'activate')"
              :data-action="b.action"
              @click="run(b.action)"
            >
              {{ t(`processEditor.actions.${b.action}`) }}
            </NButton>
          </template>
        </div>
      </div>
    </template>
  </RecordDrawer>
</template>

<style scoped>
.status,
.note {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
  font-weight: normal;
}

.status[data-status='active'] {
  color: var(--ant-accent);
}

.back {
  padding: 0;
  border: 0;
  background: none;
  color: var(--ant-accent);
  font: inherit;
  font-size: var(--ant-fs-meta);
  cursor: pointer;
}

.body {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-4);
  min-width: 0;
}

.center {
  display: flex;
  justify-content: center;
  padding: var(--ant-space-8) 0;
}

.row {
  cursor: pointer;
}

.num {
  text-align: right;
}

.diff,
.route {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
  margin: 0;
  padding-left: var(--ant-space-5);
}

.route li {
  display: flex;
  flex-direction: column;
}

.signed {
  color: var(--ant-accent);
}

.waiting {
  color: var(--ant-text-3);
}

.strong {
  margin: 0;
  font-weight: var(--ant-fw-bold);
}

.actions {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
  width: 100%;
}

.buttons {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-2);
  justify-content: flex-end;
}
</style>
