<script setup lang="ts">
/**
 * Раздел «Процесс» стола технолога (FR-24): версии описания процесса,
 * читаемое представление выбранной версии и перечень отличий от действующей
 * («добавлена точка предъявления после сварки», «порог уверенности для вида X
 * изменён»). Схема BPMN — просмотрщик эпика 10, редактор и кворум — эпик 39;
 * здесь только текст.
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton } from 'naive-ui'
import {
  activeVersion,
  diffVersions,
  type ProcessDiffEntry,
  type ProcessPropertyKey,
  type ProcessPropertyValue,
  type ProcessVersion,
} from '@/entities/process-version'
import { codeToKey } from '@/shared/i18n'
import type { Density } from '@/shared/config/widget'

type Mode = 'view' | 'diff'

const props = withDefaults(
  defineProps<{
    versions: ProcessVersion[]
    density?: Density
    /** Какую вкладку открыть: читаемое представление или отличия (срез стола `mode`). */
    initialMode?: Mode
    /** Готовый перечень отличий от сервера; нет — считается здесь. */
    diff?: ProcessDiffEntry[] | null
  }>(),
  { density: 'compact', initialMode: 'view', diff: null },
)
/** Выбранная версия (`version_id`). */
const selected = defineModel<string | null>('selected', { default: null })

const { t, n, d } = useI18n()

const mode = ref<Mode>(props.initialMode)
const active = computed(() => activeVersion(props.versions))

/** По умолчанию — самая новая не действующая (черновик на утверждении), иначе действующая. */
const fallback = computed(() => props.versions.find((v) => v.status === 'on_approval' || v.status === 'draft') ?? active.value ?? props.versions[0])
watch(
  () => props.versions,
  () => {
    if (!props.versions.some((v) => v.version_id === selected.value)) selected.value = fallback.value?.version_id ?? null
  },
  { immediate: true },
)
const current = computed(() => props.versions.find((v) => v.version_id === selected.value) ?? fallback.value)

const entries = computed<ProcessDiffEntry[] | null>(() => {
  if (!current.value || !active.value || current.value.version_id === active.value.version_id) return null
  return props.diff ?? diffVersions(active.value, current.value)
})

const statusText = (s: string) => t(`statuses.processVersion.${codeToKey(s)}`)
const dateTime = (x: string) => d(new Date(x), 'dateTime')

/** Значение свойства в читаемом виде. */
function valueText(v: ProcessPropertyValue | undefined): string {
  if (v === true) return t('widgets.analysis.process.valueYes')
  if (v === false) return t('widgets.analysis.process.valueNo')
  if (v === null || v === undefined || v === '') return t('widgets.analysis.process.valueEmpty')
  return typeof v === 'number' ? n(v) : v
}

const propText = (p: ProcessPropertyKey) => t(`process.properties.${p}`)
const threshold = (bp: number | null) => (bp === null ? t('widgets.analysis.process.valueEmpty') : n(bp / 10_000, 'decimal2'))

/** Строка отличия (process.diff.*). */
function diffText(e: ProcessDiffEntry): string {
  switch (e.kind) {
    case 'elementAdded':
      return t('process.diff.elementAdded', { element: e.element })
    case 'elementRemoved':
      return t('process.diff.elementRemoved', { element: e.element })
    case 'presentationPointAdded':
      return t('process.diff.presentationPointAdded', { step: e.step })
    case 'thresholdChanged':
      return t('process.diff.thresholdChanged', { defectType: e.defectType, from: threshold(e.from), to: threshold(e.to) })
    case 'propertyChanged':
      return t('process.diff.propertyChanged', { element: e.element, property: propText(e.property), from: valueText(e.from), to: valueText(e.to) })
  }
  return ''
}

const propsOf = (p: Partial<Record<ProcessPropertyKey, ProcessPropertyValue>>) =>
  Object.entries(p) as [ProcessPropertyKey, ProcessPropertyValue][]
</script>

<template>
  <div class="process" :class="`density-${density}`" data-testid="process-versions">
    <nav class="list" :aria-label="t('process.versions')">
      <h4>{{ t('process.versions') }}</h4>
      <button
        v-for="v in versions"
        :key="v.version_id"
        type="button"
        class="version"
        :class="{ on: current?.version_id === v.version_id }"
        :data-version="v.version_id"
        :data-status="v.status"
        :aria-current="current?.version_id === v.version_id || undefined"
        @click="selected = v.version_id"
      >
        <span class="v-label">{{ t('common.words.version') }} {{ v.label }}</span>
        <span class="v-status">{{ statusText(v.status) }}</span>
        <span class="v-date">{{ dateTime(v.created_at) }}</span>
      </button>
    </nav>

    <section v-if="current" class="body">
      <div class="modes" role="tablist">
        <NButton size="small" :type="mode === 'view' ? 'primary' : 'default'" secondary role="tab" :aria-selected="mode === 'view'" data-testid="mode-view" @click="mode = 'view'">
          {{ t('process.readableView') }}
        </NButton>
        <NButton size="small" :type="mode === 'diff' ? 'primary' : 'default'" secondary role="tab" :aria-selected="mode === 'diff'" data-testid="mode-diff" @click="mode = 'diff'">
          {{ t('process.diffTitle') }}
        </NButton>
      </div>

      <p class="meta">
        <strong>{{ t('common.words.version') }} {{ current.label }}</strong> · {{ statusText(current.status) }}
        <template v-if="current.author"> · {{ t('widgets.analysis.process.author', { who: current.author }) }}</template>
        <template v-if="current.effective_from"> · {{ t('widgets.analysis.process.effectiveFrom', { time: dateTime(current.effective_from) }) }}</template>
        <template v-if="current.quorum"> · <span :title="t('hints.quorum')">{{ t('process.quorum', current.quorum) }}</span></template>
      </p>
      <p v-if="current.status !== 'active' && current.status !== 'retired'" class="muted">{{ t('process.futureItemsOnly') }}</p>

      <!-- Отличия от действующей (FR-24). -->
      <div v-if="mode === 'diff'" class="diff" data-testid="diff">
        <p v-if="!active" class="muted">{{ t('widgets.analysis.process.noActive') }}</p>
        <p v-else-if="entries === null" class="muted">{{ t('widgets.analysis.process.isActive') }}</p>
        <p v-else-if="!entries.length" class="muted">{{ t('widgets.analysis.process.noDiff') }}</p>
        <ul v-else>
          <li v-for="(e, i) in entries" :key="i" :data-kind="e.kind">{{ diffText(e) }}</li>
        </ul>
      </div>

      <!-- Читаемое представление. -->
      <ol v-else class="elements" data-testid="readable">
        <li v-for="e in current.elements" :key="e.id" class="element" :data-kind="e.kind">
          <div class="el-head">
            <span class="kind">{{ t(`process.elements.${e.kind}`) }}</span>
            <strong>{{ e.name }}</strong>
            <span v-if="e.lane" class="muted">· {{ t('process.elements.lane') }}: {{ e.lane }}</span>
            <code v-if="e.step_key" class="step">{{ t('widgets.analysis.process.step', { key: e.step_key }) }}</code>
          </div>
          <dl v-if="propsOf(e.properties).length || e.thresholds" class="props">
            <template v-for="[k, v] in propsOf(e.properties)" :key="k">
              <dt>{{ propText(k) }}</dt>
              <dd>{{ valueText(v) }}</dd>
            </template>
            <template v-if="e.thresholds && Object.keys(e.thresholds).length">
              <dt>{{ propText('reactionMap') }}</dt>
              <dd>
                <span v-for="(bp, type) in e.thresholds" :key="type" class="thr">{{ type }}: {{ threshold(bp) }}</span>
              </dd>
            </template>
          </dl>
        </li>
      </ol>
    </section>
  </div>
</template>

<style scoped>
.process {
  display: grid;
  grid-template-columns: minmax(180px, 240px) minmax(0, 1fr);
  gap: 16px;
  font-size: 13px;
}

.density-large {
  font-size: 16px;
}

h4 {
  margin: 0 0 6px;
  font-size: 1em;
}

.list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.version {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 2px 8px;
  padding: 6px 8px;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  background: #fff;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.version.on {
  border-color: #2f6fdb;
  background: #eff6ff;
}

.v-label {
  font-weight: 700;
}

.v-status {
  color: #6b7280;
}

.v-date {
  grid-column: 1 / -1;
  color: #6b7280;
  font-size: 12px;
}

.body {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}

.modes {
  display: flex;
  gap: 6px;
}

.meta,
.muted {
  margin: 0;
}

.muted {
  color: #6b7280;
}

.diff ul {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin: 0;
  padding-left: 18px;
}

.diff li[data-kind='presentationPointAdded'],
.diff li[data-kind='thresholdChanged'] {
  font-weight: 700;
}

.elements {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin: 0;
  padding-left: 22px;
}

.el-head {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: baseline;
}

.kind {
  padding: 0 6px;
  border-radius: 8px;
  background: #f3f4f6;
  font-size: 11px;
}

.step {
  color: #6b7280;
  font-family: 'PT Mono', monospace;
  font-size: 11px;
}

.props {
  display: grid;
  grid-template-columns: max-content 1fr;
  gap: 2px 12px;
  margin: 4px 0 0;
}

.props dt {
  color: #6b7280;
}

.props dd {
  margin: 0;
}

.thr {
  margin-right: 10px;
  font-family: 'PT Mono', monospace;
}
</style>
