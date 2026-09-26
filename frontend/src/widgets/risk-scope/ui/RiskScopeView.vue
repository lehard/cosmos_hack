<script setup lang="ts">
/**
 * Область риска — «тающая область» (FR-61, FR-62, FR-9): сверху крупно —
 * сколько изделий в области сейчас и путь 34 → 13 → 6; ступени — каждое
 * изменение с основанием (текст сервера), автором, временем и тем, где изделия;
 * изделия — группами по тому, что о них известно: подтверждено / под
 * подозрением / нет данных / исключено с основанием. Серое (нет данных) — не
 * зелёное: без доказательства изделие из области не выходит. Изделия в области
 * «подвергались условиям, способным вызвать дефект» — это не брак. Расширяет
 * область правило, сужает только человек по доказательствам.
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NAlert } from 'naive-ui'
import {
  FACTOR_TEXT,
  SCOPE_LOCATIONS,
  currentVersion,
  hasBasis,
  scopeIssues,
  scopeReduction,
  recordLabel,
  type IncidentKnown,
  type JournalRecordRef,
  type RiskScopeModel,
  type ScopeIssue,
  type ScopeItem,
  type ScopeLocation,
  type ScopeVersion,
} from '@/entities/incident'
import { statusAxes } from '@/shared/api/generated/statuses'
import { codeToKey } from '@/shared/i18n'
import { naiveSizeOf, type Density } from '@/shared/config/widget'
import { useMomentStore } from '@/shared/model/moment'
import { ActionButton } from '@/shared/ui'

const props = withDefaults(
  defineProps<{
    model: RiskScopeModel
    density?: Density
    /** Доступно ли сужение области. */
    canNarrow?: boolean
    /** Доступно ли расширение области. */
    canExpand?: boolean
    /** Команда отправляется — формы выключены. */
    busy?: boolean
    /** Записи, на которые можно сослаться как на доказательство (дорожки несоответствия расследования). */
    evidenceOptions?: readonly JournalRecordRef[]
  }>(),
  { density: 'compact', canNarrow: true, canExpand: true, busy: false, evidenceOptions: () => [] },
)
const emit = defineEmits<{
  /** Сузить область: исключаемые изделия, доказательства и основание (FR-61, AD-27). */
  narrow: [input: { item_ids: string[]; reason: string; evidence_event_ids: string[] }]
  /** Расширить область: добавляемые изделия, основание и (по желанию) доказательства. */
  expand: [input: { item_ids: string[]; reason: string; evidence_event_ids: string[] }]
  /** Открыть изделие. */
  'open-item': [itemId: string]
}>()
defineSlots<{
  /** Автор версии (id персоны) — контейнер показывает имя. */
  author(props: { id: string; authorName: string | null }): unknown
}>()

const { t, d } = useI18n()
const moment = useMomentStore()

const versions = computed(() => props.model.versions)
const current = computed(() => currentVersion(props.model))
const reduction = computed(() => scopeReduction(props.model))
const issues = computed(() => scopeIssues(props.model))

/** Раскрытая ступень пути: выбранная щелчком, иначе последняя (текущая). */
const picked_version = ref<number | null>(null)
const openIndex = computed(() => {
  const i = versions.value.findIndex((v) => v.scope_version === picked_version.value)
  return i >= 0 ? i : versions.value.length - 1
})
const openVersion = computed(() => versions.value[openIndex.value] ?? null)
const openDelta = computed(() => (openVersion.value ? deltaOf(openVersion.value, openIndex.value) : null))
/** Под числом — одна строка: кто изменил область. */
function shortTitle(v: ScopeVersion): string {
  if (v.change === 'computed') return t('widgets.analysis.riskScope.shortComputed')
  if (v.change === 'narrowed') return v.trigger?.kind === 'late_event' ? t('widgets.analysis.riskScope.shortLate') : t('widgets.analysis.riskScope.shortNarrowed')
  return t('widgets.analysis.riskScope.shortExpanded')
}

const dateTime = (x: string) => d(new Date(x), 'dateTime')

const LOCATION_TEXT: Record<ScopeLocation, string> = {
  in_production: 'riskScope.breakdown.inProduction',
  moved_on: 'riskScope.breakdown.movedOn',
  assembled: 'riskScope.breakdown.assembled',
  shipped: 'riskScope.breakdown.shipped',
}

/** Порядок групп: от того, что требует действий, к исключённому. */
const KNOWN_ORDER: readonly IncidentKnown[] = ['confirmed', 'suspect', 'unknown', 'excluded']
const KNOWN_TEXT: Record<IncidentKnown, string> = {
  confirmed: 'statuses.incident.confirmed',
  suspect: 'statuses.incident.suspect',
  unknown: 'riskScope.noData',
  excluded: 'widgets.analysis.riskScope.excludedWithBasis',
}
const KNOWN_HINT: Record<IncidentKnown, string> = {
  confirmed: 'widgets.analysis.riskScope.hintConfirmed',
  suspect: 'widgets.analysis.riskScope.hintSuspect',
  unknown: 'widgets.analysis.riskScope.hintUnknown',
  excluded: 'widgets.analysis.riskScope.hintExcluded',
}
/** Тон статуса из словаря сервера → CSS-переменная темы. */
const toneOf = (k: IncidentKnown, variant?: 'soft') =>
  `var(--ant-status-${statusAxes.incident.values[k].tone}${variant ? `-${variant}` : ''})`

const byKnown = computed(() => {
  const out = { confirmed: [], suspect: [], unknown: [], excluded: [] } as Record<IncidentKnown, ScopeItem[]>
  for (const it of props.model.items) out[it.known].push(it)
  return out
})

/** Изменение размера относительно прошлой ступени: «−21», «+12». */
function deltaOf(v: ScopeVersion, i: number): string | null {
  const prev = versions.value[i - 1]
  if (!prev || prev.size === v.size) return null
  return prev.size > v.size ? `−${prev.size - v.size}` : `+${v.size - prev.size}`
}

/** Что произошло на ступени: система собрала / человек сузил / расширено (правилом или человеком). */
function stepTitle(v: ScopeVersion): string {
  if (v.change === 'computed') return t('widgets.analysis.riskScope.stepComputed')
  if (v.change === 'narrowed') return t('widgets.analysis.riskScope.stepNarrowed')
  return t(v.author ? 'widgets.analysis.riskScope.stepExpanded' : 'widgets.analysis.riskScope.stepExpandedBySystem')
}

/** Где изделия ступени — только ненулевые места. */
const whereText = (v: ScopeVersion) =>
  SCOPE_LOCATIONS.filter((loc) => v.breakdown[loc] > 0)
    .map((loc) => `${t(LOCATION_TEXT[loc])} ${v.breakdown[loc]}`)
    .join(' · ')

const ISSUE_TEXT: Record<ScopeIssue['kind'], string> = {
  basis_missing: 'widgets.analysis.riskScope.issueBasisMissing',
  breakdown_mismatch: 'widgets.analysis.riskScope.issueBreakdownMismatch',
  version_order: 'widgets.analysis.riskScope.issueVersionOrder',
  narrowed_grew: 'widgets.analysis.riskScope.issueNarrowedGrew',
}

/** Открытая форма правки области. */
const form = ref<'narrow' | 'expand' | null>(null)
const picked = ref<string[]>([])
const typed = ref('')
const reason = ref('')
const evidence = ref<string[]>([])

/** Кандидаты на исключение — изделия, ещё не исключённые. */
const narrowable = computed(() => props.model.items.filter((i) => i.known !== 'excluded'))

function openForm(kind: 'narrow' | 'expand'): void {
  form.value = kind
  picked.value = []
  typed.value = ''
  reason.value = ''
  evidence.value = []
}

/** Записи-доказательства — свежие сверху. */
const evidenceList = computed(() => [...props.evidenceOptions].sort((a, b) => b.occurred_at.localeCompare(a.occurred_at)))
const recordText = (r: JournalRecordRef) => recordLabel(r, t)
const labelOf = (id: string) => props.model.items.find((i) => i.item_id === id)?.label ?? id

const typedIds = computed(() =>
  typed.value
    .split(/[\s,;]+/)
    .map((x) => x.trim())
    .filter(Boolean),
)
const formIds = computed(() => (form.value === 'narrow' ? picked.value : typedIds.value))
// Без основания изделие из области не выходит (FR-61) — и не входит без него; сужение — только с доказательством (AD-27).
const missing = computed(() => {
  const out: string[] = []
  if (!formIds.value.length) out.push(t(form.value === 'narrow' ? 'widgets.analysis.riskScope.needItems' : 'widgets.analysis.riskScope.needItemsExpand'))
  if (form.value === 'narrow' && !evidence.value.length) out.push(t('widgets.analysis.riskScope.needEvidence'))
  if (!reason.value.trim()) out.push(t('widgets.analysis.riskScope.needReason'))
  return out
})
const formReady = computed(() => missing.value.length === 0)

function submit(): void {
  if (!form.value || !formReady.value) return
  const input = { item_ids: [...formIds.value], reason: reason.value.trim(), evidence_event_ids: [...evidence.value] }
  if (form.value === 'narrow') emit('narrow', input)
  else emit('expand', input)
  form.value = null
}
</script>

<template>
  <div class="scope" :class="`density-${density}`" data-testid="risk-scope">
    <NAlert v-if="issues.length" type="error" :bordered="false" :show-icon="false" class="note" data-testid="issues">
      <div v-for="(x, i) in issues" :key="i">{{ t(ISSUE_TEXT[x.kind], { version: x.version }) }}</div>
      <div>{{ t('riskScope.basisRequired') }}</div>
    </NAlert>

    <!-- Герой: путь области 34 → 13 → 6; число — кнопка, под ним одна строка, что произошло. -->
    <ol class="path" :aria-label="t('riskScope.meltingScope')" data-testid="scope-path">
      <li v-for="v in versions" :key="v.scope_version" class="path-item" :data-change="v.change">
        <button
          type="button"
          class="step-btn"
          :aria-pressed="v.scope_version === openVersion?.scope_version"
          :data-version="v.scope_version"
          :data-basis="hasBasis(v) || v.change !== 'narrowed' ? undefined : 'missing'"
          data-testid="path-step"
          @click="picked_version = v.scope_version"
        >
          <span class="size" data-testid="size">{{ v.size }}</span>
          <span class="step-short ant-clamp-2">{{ shortTitle(v) }}</span>
        </button>
      </li>
    </ol>
    <p class="muted ant-wrap" data-testid="not-defective">
      <template v-if="reduction && versions.length > 1"><span data-testid="reduction">{{ t('riskScope.reduction', { from: reduction.from, to: reduction.to }) }}</span> · </template>{{ t('widgets.analysis.riskScope.onlyByEvidence') }}
    </p>

    <!-- Выбранная ступень: почему такой размер — основание, повод, доказательства, кто исключён, кто и когда. -->
    <section v-if="openVersion" class="step" :data-version="openVersion.scope_version" :data-change="openVersion.change" data-testid="version-line">
      <p class="step-title">
        {{ stepTitle(openVersion) }}
        <span v-if="openDelta" class="delta">{{ openDelta }}</span>
      </p>
      <p v-if="openVersion.trigger && openVersion.trigger.kind !== 'human' && openVersion.trigger.kind !== 'computed'" class="step-trigger ant-wrap" :data-kind="openVersion.trigger.kind" data-testid="step-trigger">
        {{ t(`widgets.analysis.riskScope.trigger.${codeToKey(openVersion.trigger.kind)}`) }}: {{ openVersion.trigger.label }}
      </p>
      <p class="step-basis ant-wrap" :class="{ bad: openVersion.change === 'narrowed' && !hasBasis(openVersion) }">
        <template v-if="openVersion.reason?.text.trim()">{{ openVersion.reason.text }}</template>
        <template v-else-if="openVersion.change === 'narrowed'">{{ t('riskScope.basis') }}: {{ t('widgets.analysis.riskScope.noBasis') }}</template>
      </p>
      <p v-if="openVersion.change === 'computed' && (model.common_factor || model.last_known_good)" class="muted ant-wrap">
        <template v-if="model.common_factor">{{ t('riskScope.commonFactor', { factor: `${t(FACTOR_TEXT[model.common_factor.factor])}: ${model.common_factor.label || model.common_factor.value}` }) }}</template>
        <template v-if="model.last_known_good"> · {{ t('riskScope.lastKnownGood', { what: model.last_known_good.label, time: dateTime(model.last_known_good.at) }) }}</template>
      </p>
      <ul v-if="openVersion.evidence?.length" class="step-evidence" data-testid="step-evidence">
        <li v-for="r in openVersion.evidence" :key="r.event_id" class="ant-wrap">{{ d(new Date(r.occurred_at), 'dateTime') }} · {{ recordText(r) }}</li>
      </ul>
      <p v-if="openVersion.change !== 'computed' && (openVersion.items_removed?.length || openVersion.items_added?.length)" class="step-items ant-wrap" data-testid="step-items">
        <template v-if="openVersion.items_removed?.length">{{ t('widgets.analysis.riskScope.itemsRemoved') }}: </template>
        <button v-for="id in openVersion.items_removed ?? []" :key="`r-${id}`" type="button" class="item-link" @click="emit('open-item', id)">{{ labelOf(id) }}</button>
        <template v-if="openVersion.items_added?.length">{{ t('widgets.analysis.riskScope.itemsAdded') }}: </template>
        <button v-for="id in openVersion.items_added ?? []" :key="`a-${id}`" type="button" class="item-link" @click="emit('open-item', id)">{{ labelOf(id) }}</button>
      </p>
      <p class="step-meta muted ant-wrap">
        <template v-if="openVersion.author"><slot name="author" :id="openVersion.author" :author-name="openVersion.author_name ?? null">{{ openVersion.author_name || openVersion.author }}</slot></template>
        <template v-else>{{ t('widgets.analysis.riskScope.systemAuthor') }}</template>
        · {{ dateTime(openVersion.recorded_at) }}
        <template v-if="openVersion.evidence_event_ids.length"> · {{ t('widgets.analysis.riskScope.evidence', { n: openVersion.evidence_event_ids.length }) }}</template>
        <template v-if="whereText(openVersion)"> · {{ t('riskScope.breakdown.title') }}: {{ whereText(openVersion) }}</template>
      </p>
      <p v-if="model.shipped_to_partners" class="muted">{{ t('riskScope.partners.shippedToPartners', { n: model.shipped_to_partners }) }}</p>
    </section>

    <!-- Изделия области — по раскрытию: серое (нет данных) — не зелёное (исключено с основанием). -->
    <details v-if="model.items.length" class="items" data-testid="items">
      <summary>{{ t('widgets.analysis.riskScope.itemsSummary', { n: current?.size ?? 0 }) }}</summary>
      <template v-for="k in KNOWN_ORDER" :key="k">
        <component
          :is="k === 'excluded' ? 'details' : 'div'"
          v-if="byKnown[k].length"
          class="group"
          :data-known="k"
          :style="{ '--tone': toneOf(k), '--tone-soft': toneOf(k, 'soft') }"
        >
          <component :is="k === 'excluded' ? 'summary' : 'p'" class="group-title ant-wrap">
            <strong>{{ t(KNOWN_TEXT[k]) }} · {{ byKnown[k].length }}</strong>
            <span class="muted"> — {{ t(KNOWN_HINT[k]) }}</span>
          </component>
          <ul class="chips">
            <li v-for="it in byKnown[k]" :key="it.item_id" :data-item="it.item_id">
              <button type="button" class="chip" :title="`${t(`statuses.incidentAction.${codeToKey(it.action)}`)} · ${t(LOCATION_TEXT[it.location])}`" @click="emit('open-item', it.item_id)">
                <span class="chip-label ant-ellipsis">{{ it.label }}</span>
                <span class="chip-meta ant-clamp-2">{{ t(`statuses.incidentAction.${codeToKey(it.action)}`) }} · {{ t(LOCATION_TEXT[it.location]) }}</span>
              </button>
            </li>
          </ul>
        </component>
      </template>
    </details>

    <!-- Готовые сужения по данным: изделия и основание из журнала — человек решает одним щелчком (FR-61, AD-27). -->
    <section v-if="model.narrow_options?.length && canNarrow" class="options" data-testid="narrow-options">
      <p class="options-title">{{ t('widgets.analysis.riskScope.optionsTitle') }}</p>
      <article v-for="(o, oi) in model.narrow_options" :key="oi" class="option" :data-option="oi">
        <p class="option-label ant-wrap">{{ o.label }}</p>
        <p class="option-reason ant-wrap">{{ t('riskScope.basis') }}: {{ o.reason_text }}</p>
        <ul v-if="o.evidence.length" class="step-evidence">
          <li v-for="r in o.evidence" :key="r.event_id" class="ant-wrap">{{ d(new Date(r.occurred_at), 'dateTime') }} · {{ recordText(r) }}</li>
        </ul>
        <ActionButton
          overflow="wrap"
          :size="naiveSizeOf(density)"
          type="primary"
          :disabled="busy || moment.isReplay || !o.item_ids.length || !o.evidence.length"
          data-testid="narrow-option"
          @click="emit('narrow', { item_ids: [...o.item_ids], reason: o.reason_text, evidence_event_ids: o.evidence.map((r) => r.event_id) })"
          :label="t('widgets.analysis.riskScope.optionApply', { n: o.item_ids.length })"
        />
      </article>
    </section>

    <footer class="actions">
      <ActionButton overflow="wrap" :size="naiveSizeOf(density)" :type="model.narrow_options?.length ? 'default' : 'primary'" :quaternary="Boolean(model.narrow_options?.length)" :secondary="!model.narrow_options?.length" :disabled="!canNarrow || busy || moment.isReplay" data-testid="narrow" @click="openForm('narrow')" :label="t('riskScope.narrow')" />
      <ActionButton overflow="wrap" :size="naiveSizeOf(density)" quaternary :disabled="!canExpand || busy || moment.isReplay" data-testid="expand" @click="openForm('expand')" :label="t('riskScope.expand')" />
    </footer>

    <form v-if="form" class="form" :data-form="form" @submit.prevent="submit">
      <fieldset v-if="form === 'narrow'" class="picks">
        <legend>{{ t('widgets.analysis.riskScope.narrowItems') }}</legend>
        <label v-for="it in narrowable" :key="it.item_id">
          <input v-model="picked" type="checkbox" :value="it.item_id" :data-pick="it.item_id" />
          {{ it.label }}
        </label>
      </fieldset>
      <label v-else>
        <span>{{ t('widgets.analysis.riskScope.expandItems') }}</span>
        <textarea v-model="typed" rows="2" data-testid="expand-items" />
      </label>
      <fieldset class="evidence" data-testid="evidence-picks">
        <legend>{{ t(form === 'narrow' ? 'widgets.analysis.riskScope.evidenceLegend' : 'widgets.analysis.riskScope.evidenceLegendOptional') }}</legend>
        <label v-for="r in evidenceList" :key="r.event_id" class="evidence-row">
          <input v-model="evidence" type="checkbox" :value="r.event_id" :data-evidence="r.event_id" />
          <span class="ant-wrap"><span class="muted">{{ d(new Date(r.occurred_at), 'dateTime') }}</span> · {{ recordText(r) }}</span>
        </label>
        <p v-if="!evidenceList.length" class="muted ant-wrap" data-testid="no-evidence">{{ t('widgets.analysis.riskScope.noEvidence') }}</p>
      </fieldset>
      <label>
        <span>{{ t('widgets.analysis.riskScope.reason') }}</span>
        <textarea v-model="reason" rows="2" required data-testid="scope-reason" />
      </label>
      <p v-if="missing.length" class="missing ant-wrap" data-testid="form-missing">{{ t('widgets.analysis.riskScope.stillNeeded', { what: missing.join(' · ') }) }}</p>
      <p class="muted">{{ t(form === 'narrow' ? 'riskScope.narrowOnlyByHuman' : 'riskScope.expandIsCautious') }}</p>
      <div class="actions">
        <ActionButton overflow="wrap" :size="naiveSizeOf(density)" type="primary" attr-type="submit" :disabled="!formReady || busy || moment.isReplay" data-testid="scope-submit" :label="t('common.actions.send')" />
        <ActionButton overflow="wrap" :size="naiveSizeOf(density)" quaternary @click="form = null" :label="t('common.actions.cancel')" />
      </div>
    </form>
  </div>
</template>

<style scoped>
.scope {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-5);
  min-width: 0;
  font-size: var(--ant-fs-body);
}

.density-large {
  font-size: var(--ant-fs-lg);
}

p {
  margin: 0;
}

.muted {
  color: var(--ant-text-3);
}

.note {
  padding: var(--ant-space-2) var(--ant-space-3);
}

h4 {
  margin: 0 0 var(--ant-space-2);
  font-size: 1em;
}

/* Итог: число крупно, путь сужения, счётчики «что известно». */












/* Ступени области. */










.step-basis.bad {
  color: var(--ant-status-danger-text);
  font-weight: var(--ant-fw-bold);
}

.step-trigger {
  color: var(--ant-status-attention-text);
  font-weight: var(--ant-fw-bold);
}

.step[data-change='expanded'] .step-trigger {
  color: var(--ant-status-danger-text);
}

.step-evidence {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin: 0;
  padding-left: var(--ant-space-4);
  color: var(--ant-text-2);
  font-size: var(--ant-fs-meta);
}

.step-items {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-1);
  align-items: baseline;
  font-size: var(--ant-fs-meta);
}

.item-link {
  padding: 0 var(--ant-space-1);
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-sm);
  background: var(--ant-surface);
  color: var(--ant-accent);
  font: inherit;
  cursor: pointer;
}

.item-link:hover {
  background: var(--ant-surface-hover);
}

.evidence {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  max-height: 240px;
  margin: 0;
  padding: var(--ant-space-1) var(--ant-space-2);
  overflow-y: auto;
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-sm);
}

.form .evidence-row {
  flex-direction: row;
  gap: var(--ant-space-2);
  align-items: baseline;
}

.missing {
  color: var(--ant-status-attention-text);
  font-weight: var(--ant-fw-bold);
}

.step-meta,
.step-where {
  font-size: var(--ant-fs-meta);
}

/* Изделия группами по тому, что известно. */

.group {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
  min-width: 0;
}

.group-title {
  cursor: default;
}

summary.group-title {
  cursor: pointer;
}

.chips {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: var(--ant-space-2);
  margin: 0;
  padding: 0;
  list-style: none;
}

.chip {
  display: flex;
  flex-direction: column;
  width: 100%;
  min-width: 0;
  padding: var(--ant-space-1) var(--ant-space-2);
  border: 1px solid var(--ant-border);
  border-left: 4px solid var(--tone);
  border-radius: var(--ant-radius-sm);
  background: var(--ant-surface);
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.chip:hover {
  background: var(--ant-surface-hover);
}

.chip-label {
  font-weight: var(--ant-fw-bold);
}

.chip-meta {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-2);
}

.form {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
  padding: var(--ant-space-2);
  border: 1px solid var(--ant-border-strong);
  border-radius: var(--ant-radius-md);
  background: var(--ant-surface-subtle);
}

.form label {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.picks {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-1) var(--ant-space-4);
  margin: 0;
  padding: var(--ant-space-1) var(--ant-space-2);
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-sm);
}

.picks label {
  flex-direction: row;
  gap: var(--ant-space-1);
  align-items: center;
}

.form textarea {
  padding: var(--ant-space-1) var(--ant-space-2);
  border: 1px solid var(--ant-border-strong);
  border-radius: var(--ant-radius-sm);
  font: inherit;
  resize: vertical;
}

/* Путь области: числа-кнопки со стрелками; выбранная ступень выделена. */
.path {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-2);
  align-items: stretch;
  margin: 0;
  padding: 0;
  list-style: none;
}

.path-item {
  display: flex;
  align-items: center;
}

.path-item + .path-item::before {
  margin-right: var(--ant-space-2);
  color: var(--ant-text-3);
  content: '→';
}

.step-btn {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  min-width: 90px;
  max-width: 200px;
  padding: 0 0 var(--ant-space-1);
  border: 0;
  border-bottom: 2px solid transparent;
  background: none;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.step-btn:hover {
  border-bottom-color: var(--ant-border-strong);
}

.step-btn[aria-pressed='true'] {
  border-bottom-color: var(--ant-accent);
}

.step-btn[data-basis='missing'] {
  border-bottom-color: var(--ant-status-danger);
}

.size {
  font-size: var(--ant-fs-title);
  font-weight: var(--ant-fw-bold);
  line-height: 1.2;
  font-variant-numeric: tabular-nums;
}

.path-item[data-change='narrowed'] .size {
  color: var(--ant-status-success-text);
}

.path-item[data-change='expanded'] .size {
  color: var(--ant-status-danger-text);
}

.step-short {
  color: var(--ant-text-2);
  font-size: var(--ant-fs-meta);
}

/* Раскрытая ступень. */
.step {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  min-width: 0;
  padding-left: var(--ant-space-4);
  border-left: 3px solid var(--ant-accent);
}

.step-title {
  font-weight: var(--ant-fw-bold);
}

.delta {
  margin-left: var(--ant-space-2);
  color: var(--ant-status-success-text);
}

.step[data-change='expanded'] .delta,
.step[data-change='expanded'] .step-title {
  color: var(--ant-status-danger-text);
}

.options {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
}

.options-title {
  font-weight: var(--ant-fw-bold);
}

.option {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  align-items: flex-start;
  padding-left: var(--ant-space-4);
  border-left: 3px solid var(--ant-accent);
}

.option-label {
  font-weight: var(--ant-fw-bold);
}

.option-reason {
  color: var(--ant-text-2);
}

.items summary {
  font-weight: var(--ant-fw-bold);
  cursor: pointer;
}

.items {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-3);
}
</style>
