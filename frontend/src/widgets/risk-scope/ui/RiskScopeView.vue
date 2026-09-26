<script setup lang="ts">
/**
 * Область риска — «тающая область» (FR-61, FR-62, FR-9): версии области с
 * основанием каждого сужения, разбивка «в производстве / ушли дальше /
 * собраны / отгружены», две оси статуса изделия в инциденте. Изделия в
 * области «подвергались условиям, способным вызвать дефект» — это не брак.
 * Расширяет область правило, сужает только человек по доказательствам.
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NAlert, NButton } from 'naive-ui'
import {
  FACTOR_TEXT,
  SCOPE_LOCATIONS,
  currentVersion,
  hasBasis,
  scopeIssues,
  scopeReduction,
  type RiskScopeModel,
  type ScopeIssue,
  type ScopeLocation,
  type ScopeVersion,
} from '@/entities/incident'
import { statusDictionaries, statusPalette } from '@/shared/api/generated/statuses'
import { codeToKey } from '@/shared/i18n'
import { naiveSizeOf, type Density } from '@/shared/config/widget'
import { useMomentStore } from '@/shared/model/moment'
import { StatusTag } from '@/shared/ui'

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
  }>(),
  { density: 'compact', canNarrow: true, canExpand: true, busy: false },
)
const emit = defineEmits<{
  /** Сузить область: исключаемые изделия и основание (FR-61). */
  narrow: [input: { item_ids: string[]; reason: string }]
  /** Расширить область: добавляемые изделия и основание. */
  expand: [input: { item_ids: string[]; reason: string }]
  /** Открыть изделие. */
  'open-item': [itemId: string]
}>()

const { t, n, d } = useI18n()
const moment = useMomentStore()

const versions = computed(() => props.model.versions)
const current = computed(() => currentVersion(props.model))
const reduction = computed(() => scopeReduction(props.model))
const issues = computed(() => scopeIssues(props.model))
const maxSize = computed(() => Math.max(1, ...versions.value.map((v) => v.size)))

const dateTime = (x: string) => d(new Date(x), 'dateTime')

const LOCATION_TEXT: Record<ScopeLocation, string> = {
  in_production: 'riskScope.breakdown.inProduction',
  moved_on: 'riskScope.breakdown.movedOn',
  assembled: 'riskScope.breakdown.assembled',
  shipped: 'riskScope.breakdown.shipped',
}

/** Строка изменения версии: «Размер при создании: 34», «Сужено: 34 → 13». */
function changeText(v: ScopeVersion, i: number): string {
  const prev = versions.value[i - 1]
  if (v.change === 'computed' || !prev) return t('riskScope.sizeAtCreation', { n: v.size })
  return t(v.change === 'narrowed' ? 'riskScope.narrowedBy' : 'riskScope.expandedBy', { from: prev.size, to: v.size })
}

/** Основание: текст причины и число доказательств; без основания — явно. */
function basisText(v: ScopeVersion): string {
  if (!hasBasis(v)) return t('widgets.analysis.riskScope.noBasis')
  const parts = [v.reason?.text.trim()].filter(Boolean) as string[]
  if (v.evidence_event_ids.length) parts.push(t('widgets.analysis.riskScope.evidence', { n: v.evidence_event_ids.length }))
  return parts.join(', ')
}

const versionLine = (v: ScopeVersion, i: number) =>
  t('riskScope.versionLine', {
    version: v.scope_version,
    change: changeText(v, i),
    basis: basisText(v),
    author: v.author ?? t('widgets.analysis.riskScope.systemAuthor'),
    time: dateTime(v.recorded_at),
  })

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

/** Кандидаты на исключение — изделия, ещё не исключённые. */
const narrowable = computed(() => props.model.items.filter((i) => i.known !== 'excluded'))

function openForm(kind: 'narrow' | 'expand'): void {
  form.value = kind
  picked.value = []
  typed.value = ''
  reason.value = ''
}

const typedIds = computed(() =>
  typed.value
    .split(/[\s,;]+/)
    .map((x) => x.trim())
    .filter(Boolean),
)
const formIds = computed(() => (form.value === 'narrow' ? picked.value : typedIds.value))
// Без основания изделие из области не выходит (FR-61) — и не входит без него.
const formReady = computed(() => formIds.value.length > 0 && reason.value.trim() !== '')

function submit(): void {
  if (!form.value || !formReady.value) return
  const input = { item_ids: [...formIds.value], reason: reason.value.trim() }
  if (form.value === 'narrow') emit('narrow', input)
  else emit('expand', input)
  form.value = null
}

const actionColor = (code: string) =>
  statusPalette[(statusDictionaries.incident_action.values as Record<string, { tone: keyof typeof statusPalette }>)[code]?.tone ?? 'neutral']
</script>

<template>
  <div class="scope" :class="`density-${density}`" data-testid="risk-scope">
    <header class="head">
      <strong>{{ model.incident_label }}</strong>
      <span v-if="model.common_factor">
        · {{ t('riskScope.commonFactor', { factor: `${t(FACTOR_TEXT[model.common_factor.factor])}: ${model.common_factor.value}` }) }}
      </span>
      <span v-if="model.window"> · {{ t('riskScope.window', { from: dateTime(model.window.start), to: dateTime(model.window.end) }) }}</span>
    </header>
    <p class="muted" :title="t('hints.lastKnownGood')">
      <template v-if="model.last_known_good">
        {{ t('riskScope.lastKnownGood', { what: model.last_known_good.label, time: dateTime(model.last_known_good.at) }) }}
      </template>
      <template v-else>{{ t('ncCard.causalWindow.lowerBound') }}: {{ t('empty.noDataUnknown') }}</template>
    </p>

    <NAlert type="info" :bordered="false" :show-icon="false" class="note" data-testid="not-defective">
      {{ t('riskScope.notDefective') }}. {{ t('riskScope.narrowOnlyByHuman') }}
    </NAlert>
    <NAlert v-if="issues.length" type="error" :bordered="false" :show-icon="false" class="note" data-testid="issues">
      <div v-for="(x, i) in issues" :key="i">{{ t(ISSUE_TEXT[x.kind], { version: x.version }) }}</div>
      <div>{{ t('riskScope.basisRequired') }}</div>
    </NAlert>

    <!-- Тающая область: размер каждой версии. -->
    <section class="melting" :aria-label="t('riskScope.meltingScope')">
      <h4>
        {{ t('riskScope.meltingScope') }}
        <template v-if="reduction && versions.length > 1">
          · <span data-testid="reduction">{{ t('riskScope.reduction', { from: reduction.from, to: reduction.to }) }}</span>
          · {{ t('riskScope.reductionPercent', { percent: n(reduction.ratio, 'percent') }) }}
        </template>
      </h4>
      <ol class="bars">
        <li v-for="(v, i) in versions" :key="v.scope_version" :data-version="v.scope_version" :data-change="v.change" :data-basis="hasBasis(v) || v.change !== 'narrowed' ? undefined : 'missing'">
          <span class="ver">{{ t('riskScope.version', { version: v.scope_version }) }}</span>
          <span class="track"><span class="fill" :style="{ width: `${(v.size / maxSize) * 100}%` }" /></span>
          <span class="size" data-testid="size">{{ v.size }}</span>
          <span class="change">{{ changeText(v, i) }}</span>
        </li>
      </ol>
    </section>

    <!-- Основания каждой правки (FR-61). -->
    <section class="history">
      <h4>{{ t('riskScope.versions') }}</h4>
      <ul>
        <li v-for="(v, i) in versions" :key="v.scope_version" :class="{ bad: v.change === 'narrowed' && !hasBasis(v) }" data-testid="version-line">
          {{ versionLine(v, i) }}
        </li>
      </ul>
    </section>

    <!-- Разбивка текущей версии. -->
    <section v-if="current" class="breakdown">
      <h4>{{ t('riskScope.breakdown.title') }}</h4>
      <div class="tiles">
        <div v-for="loc in SCOPE_LOCATIONS" :key="loc" class="tile" :data-location="loc">
          <span class="tile-n">{{ current.breakdown[loc] }}</span>
          <span class="tile-l">{{ t(LOCATION_TEXT[loc]) }}</span>
        </div>
      </div>
      <p v-if="model.shipped_to_partners" class="muted">{{ t('riskScope.partners.shippedToPartners', { n: model.shipped_to_partners }) }}</p>
    </section>

    <!-- Изделия: что известно / что делать (FR-62). -->
    <section v-if="model.items.length" class="items">
      <h4>{{ t('widgets.analysis.riskScope.items') }} · {{ t('plural.items', { n: model.items.length }, model.items.length) }}</h4>
      <table class="table">
        <thead>
          <tr>
            <th>{{ t('common.words.item') }}</th>
            <th>{{ t('widgets.analysis.riskScope.known') }}</th>
            <th>{{ t('widgets.analysis.riskScope.action') }}</th>
            <th>{{ t('widgets.analysis.riskScope.location') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="it in model.items" :key="it.item_id" :data-item="it.item_id">
            <td><button type="button" class="linklike" @click="emit('open-item', it.item_id)">{{ it.label }}</button></td>
            <td><StatusTag axis="incident" :code="it.known" /></td>
            <td>
              <span class="action"><span class="dot" :style="{ background: actionColor(it.action) }" aria-hidden="true" />{{ t(`statuses.incidentAction.${codeToKey(it.action)}`) }}</span>
            </td>
            <td>{{ t(LOCATION_TEXT[it.location]) }}</td>
          </tr>
        </tbody>
      </table>
    </section>

    <footer class="actions">
      <NButton :size="naiveSizeOf(density)" type="primary" secondary :disabled="!canNarrow || busy || moment.isReplay" data-testid="narrow" @click="openForm('narrow')">
        {{ t('riskScope.narrow') }}
      </NButton>
      <NButton :size="naiveSizeOf(density)" :disabled="!canExpand || busy || moment.isReplay" data-testid="expand" @click="openForm('expand')">
        {{ t('riskScope.expand') }}
      </NButton>
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
      <label>
        <span>{{ t('widgets.analysis.riskScope.reason') }}</span>
        <textarea v-model="reason" rows="2" required data-testid="scope-reason" />
      </label>
      <p class="muted">{{ t(form === 'narrow' ? 'riskScope.narrowOnlyByHuman' : 'riskScope.expandIsCautious') }}</p>
      <div class="actions">
        <NButton :size="naiveSizeOf(density)" type="primary" attr-type="submit" :disabled="!formReady || busy || moment.isReplay" data-testid="scope-submit">
          {{ t('common.actions.send') }}
        </NButton>
        <NButton :size="naiveSizeOf(density)" quaternary @click="form = null">{{ t('common.actions.cancel') }}</NButton>
      </div>
    </form>
  </div>
</template>

<style scoped>
.scope {
  display: flex;
  flex-direction: column;
  gap: 10px;
  font-size: 13px;
}

.density-large {
  font-size: 16px;
}

.head {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.muted {
  margin: 0;
  color: #6b7280;
}

.note {
  padding: 6px 10px;
}

h4 {
  margin: 0 0 6px;
  font-size: 1em;
}

.bars {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.bars li {
  display: grid;
  grid-template-columns: 124px minmax(120px, 1fr) 40px minmax(0, 1fr);
  gap: 8px;
  align-items: center;
}

.track {
  height: 14px;
  border-radius: 3px;
  background: #f3f4f6;
  overflow: hidden;
}

.fill {
  display: block;
  height: 100%;
  background: #e0a100;
  transition: width 0.4s ease;
}

.bars li:last-child .fill {
  background: #d64545;
}

.bars li[data-basis='missing'] .change {
  color: #d64545;
  font-weight: 700;
}

.size {
  font-family: 'PT Mono', monospace;
  font-weight: 700;
  text-align: right;
}

.change {
  color: #6b7280;
}

.history ul {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin: 0;
  padding-left: 18px;
}

.history li.bad {
  color: #d64545;
}

.tiles {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 8px;
}

.tile {
  display: flex;
  flex-direction: column;
  padding: 6px 10px;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
}

.tile-n {
  font-size: 1.6em;
  font-weight: 700;
}

.tile-l {
  color: #6b7280;
}

.table {
  width: 100%;
  border-collapse: collapse;
}

.table th {
  padding: 4px 6px;
  color: #6b7280;
  font-weight: 400;
  text-align: left;
  border-bottom: 1px solid #e5e7eb;
}

.table td {
  padding: 3px 6px;
  border-bottom: 1px solid #f3f4f6;
}

.action {
  display: inline-flex;
  gap: 6px;
  align-items: center;
}

.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.linklike {
  padding: 0;
  border: 0;
  background: none;
  color: inherit;
  font: inherit;
  cursor: pointer;
}

.linklike:hover {
  text-decoration: underline;
}

.actions {
  display: flex;
  gap: 8px;
}

.form {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 8px;
  border: 1px solid #cbd5e1;
  border-radius: 6px;
  background: #f8fafc;
}

.form label {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.picks {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 14px;
  margin: 0;
  padding: 4px 8px;
  border: 1px solid #e5e7eb;
  border-radius: 4px;
}

.picks label {
  flex-direction: row;
  gap: 4px;
  align-items: center;
}

.form textarea {
  font: inherit;
  padding: 4px 6px;
  border: 1px solid #cbd5e1;
  border-radius: 4px;
  resize: vertical;
}
</style>
