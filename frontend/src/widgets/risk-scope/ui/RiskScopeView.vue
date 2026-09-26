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

const { t, n, d } = useI18n()
const moment = useMomentStore()

const versions = computed(() => props.model.versions)
const current = computed(() => currentVersion(props.model))
const reduction = computed(() => scopeReduction(props.model))
const issues = computed(() => scopeIssues(props.model))

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
    <!-- Итог крупно: сколько изделий сейчас в области и путь, которым она сузилась. -->
    <header class="summary">
      <p class="headline ant-wrap">
        <span class="now" data-testid="scope-now">{{ current?.size ?? 0 }}</span>
        <span class="now-text">{{ t('widgets.analysis.riskScope.nowInScope', { items: t('plural.items', { n: current?.size ?? 0 }, current?.size ?? 0) }) }}</span>
      </p>
      <ol v-if="versions.length > 1" class="path" :aria-label="t('riskScope.meltingScope')" data-testid="scope-path">
        <li v-for="v in versions" :key="v.scope_version" :data-change="v.change">{{ v.size }}</li>
      </ol>
      <p v-if="reduction && versions.length > 1" class="muted">
        <span data-testid="reduction">{{ t('riskScope.reduction', { from: reduction.from, to: reduction.to }) }}</span>
        · {{ t('riskScope.reductionPercent', { percent: n(reduction.ratio, 'percent') }) }}
      </p>
      <p class="context muted ant-wrap">
        <template v-if="model.common_factor">{{ t('riskScope.commonFactor', { factor: `${t(FACTOR_TEXT[model.common_factor.factor])}: ${model.common_factor.label || model.common_factor.value}` }) }}</template>
        <template v-if="model.last_known_good"> · {{ t('riskScope.lastKnownGood', { what: model.last_known_good.label, time: dateTime(model.last_known_good.at) }) }}</template>
        <template v-else> · {{ t('ncCard.causalWindow.lowerBound') }}: {{ t('empty.noDataUnknown') }}</template>
      </p>
      <!-- Что известно об изделиях: серое (нет данных) — не зелёное (исключено с основанием). -->
      <ul class="counts" data-testid="known-counts">
        <li v-for="k in KNOWN_ORDER" :key="k" :data-known="k" :style="{ '--tone': toneOf(k), '--tone-soft': toneOf(k, 'soft') }">
          <span class="count-n">{{ byKnown[k].length }}</span>
          <span class="count-l">{{ t(KNOWN_TEXT[k]) }}</span>
        </li>
      </ul>
      <p class="muted ant-wrap" data-testid="not-defective">{{ t('riskScope.notDefective') }}. {{ t('riskScope.narrowOnlyByHuman') }}</p>
    </header>

    <NAlert v-if="issues.length" type="error" :bordered="false" :show-icon="false" class="note" data-testid="issues">
      <div v-for="(x, i) in issues" :key="i">{{ t(ISSUE_TEXT[x.kind], { version: x.version }) }}</div>
      <div>{{ t('riskScope.basisRequired') }}</div>
    </NAlert>

    <!-- Ступени: каждое изменение области — сколько, почему, по чему, кто и когда (FR-61). -->
    <section class="steps" :aria-label="t('riskScope.versions')">
      <h4>{{ t('widgets.analysis.riskScope.whyThisSize', { n: current?.size ?? 0 }) }}</h4>
      <ol>
        <li
          v-for="(v, i) in versions"
          :key="v.scope_version"
          class="step"
          :data-version="v.scope_version"
          :data-change="v.change"
          :data-basis="hasBasis(v) || v.change !== 'narrowed' ? undefined : 'missing'"
          data-testid="version-line"
        >
          <div class="step-size">
            <span class="size" data-testid="size">{{ v.size }}</span>
            <span v-if="deltaOf(v, i)" class="delta">{{ deltaOf(v, i) }}</span>
          </div>
          <div class="step-body">
            <p class="step-title">{{ stepTitle(v) }}</p>
            <p v-if="v.trigger && v.trigger.kind !== 'human' && v.trigger.kind !== 'computed'" class="step-trigger ant-wrap" :data-kind="v.trigger.kind" data-testid="step-trigger">
              {{ t(`widgets.analysis.riskScope.trigger.${codeToKey(v.trigger.kind)}`) }}: {{ v.trigger.label }}
            </p>
            <p class="step-basis ant-wrap" :class="{ bad: v.change === 'narrowed' && !hasBasis(v) }">
              <template v-if="v.reason?.text.trim()">{{ v.reason.text }}</template>
              <template v-else-if="v.change === 'narrowed'">{{ t('riskScope.basis') }}: {{ t('widgets.analysis.riskScope.noBasis') }}</template>
            </p>
            <p class="step-meta muted ant-wrap">
              <template v-if="v.author"><slot name="author" :id="v.author" :author-name="v.author_name ?? null">{{ v.author_name || v.author }}</slot></template>
              <template v-else>{{ t('widgets.analysis.riskScope.systemAuthor') }}</template>
              · {{ dateTime(v.recorded_at) }}
              <template v-if="v.evidence_event_ids.length"> · {{ t('widgets.analysis.riskScope.evidence', { n: v.evidence_event_ids.length }) }}</template>
            </p>
            <ul v-if="v.evidence?.length" class="step-evidence" data-testid="step-evidence">
              <li v-for="r in v.evidence" :key="r.event_id" class="ant-wrap">{{ d(new Date(r.occurred_at), 'dateTime') }} · {{ recordText(r) }}</li>
            </ul>
            <p v-if="v.change !== 'computed' && (v.items_removed?.length || v.items_added?.length)" class="step-items ant-wrap" data-testid="step-items">
              <template v-if="v.items_removed?.length">{{ t('widgets.analysis.riskScope.itemsRemoved') }}: </template>
              <button v-for="id in v.items_removed ?? []" :key="`r-${id}`" type="button" class="item-link" @click="emit('open-item', id)">{{ labelOf(id) }}</button>
              <template v-if="v.items_added?.length">{{ t('widgets.analysis.riskScope.itemsAdded') }}: </template>
              <button v-for="id in v.items_added ?? []" :key="`a-${id}`" type="button" class="item-link" @click="emit('open-item', id)">{{ labelOf(id) }}</button>
            </p>
            <p v-if="whereText(v)" class="step-where muted ant-wrap">{{ t('riskScope.breakdown.title') }}: {{ whereText(v) }}</p>
          </div>
        </li>
      </ol>
      <p v-if="model.shipped_to_partners" class="muted">{{ t('riskScope.partners.shippedToPartners', { n: model.shipped_to_partners }) }}</p>
    </section>

    <!-- Изделия по тому, что о них известно; щелчок — окно изделия (Д-70). -->
    <section v-if="model.items.length" class="items">
      <h4>{{ t('widgets.analysis.riskScope.items') }}</h4>
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
    </section>

    <footer class="actions">
      <ActionButton overflow="wrap" :size="naiveSizeOf(density)" type="primary" secondary :disabled="!canNarrow || busy || moment.isReplay" data-testid="narrow" @click="openForm('narrow')" :label="t('riskScope.narrow')" />
      <ActionButton overflow="wrap" :size="naiveSizeOf(density)" :disabled="!canExpand || busy || moment.isReplay" data-testid="expand" @click="openForm('expand')" :label="t('riskScope.expand')" />
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
  gap: var(--ant-space-4);
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
.summary {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
  min-width: 0;
}

.headline {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-2);
  align-items: baseline;
}

.now {
  font-size: var(--ant-fs-display);
  font-weight: var(--ant-fw-bold);
  line-height: 1;
}

.now-text {
  font-size: var(--ant-fs-title);
  font-weight: var(--ant-fw-bold);
}

.path {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-1);
  align-items: center;
  margin: 0;
  padding: 0;
  list-style: none;
  font-size: var(--ant-fs-title);
  font-weight: var(--ant-fw-bold);
}

.path li + li::before {
  margin-right: var(--ant-space-1);
  color: var(--ant-text-3);
  content: '→';
}

.path li[data-change='narrowed'] {
  color: var(--ant-status-success-text);
}

.path li[data-change='expanded'] {
  color: var(--ant-status-danger-text);
}

.counts {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(128px, 1fr));
  gap: var(--ant-space-2);
  margin: 0;
  padding: 0;
  list-style: none;
}

.counts li {
  display: flex;
  flex-direction: column;
  min-width: 0;
  padding: var(--ant-space-2) var(--ant-space-3);
  border-left: 4px solid var(--tone);
  border-radius: var(--ant-radius-md);
  background: var(--tone-soft);
}

.count-n {
  font-size: var(--ant-fs-title);
  font-weight: var(--ant-fw-bold);
}

.count-l {
  color: var(--ant-text-2);
}

/* Ступени области. */
.steps ol {
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 0;
  list-style: none;
}

.step {
  display: grid;
  grid-template-columns: 56px minmax(0, 1fr);
  gap: var(--ant-space-3);
  padding: var(--ant-space-3) 0;
  border-top: 1px solid var(--ant-border);
}

.step:first-child {
  border-top: 0;
}

.step-size {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
}

.size {
  font-size: var(--ant-fs-title);
  font-weight: var(--ant-fw-bold);
  font-variant-numeric: tabular-nums;
}

.delta {
  color: var(--ant-status-success-text);
  font-weight: var(--ant-fw-bold);
}

.step[data-change='expanded'] .delta {
  color: var(--ant-status-danger-text);
}

.step-body {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  min-width: 0;
}

.step-title {
  font-weight: var(--ant-fw-bold);
}

.step[data-change='expanded'] .step-title {
  color: var(--ant-status-danger-text);
}

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
.items {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-3);
}

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
</style>
