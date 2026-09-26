<script setup lang="ts">
/**
 * «Центр управления» руководителя производства: сверху — состояние предприятия
 * четырьмя словами (производство, качество, поток, решения); главное — «Требует
 * моего решения» (одна лента вместо «внимания» и «тревог»); рядом — главный
 * текущий инцидент: что случилось, область риска 34 → 13 → 6, сколько
 * локализовано, стадия и что дальше. Пустое не показываем — одна тихая фраза.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AlertEntry, AttentionEntry, IncidentSummary } from '@/shared/api/generated/model'
import { statusDictionaries } from '@/shared/api/generated/statuses'
import { codeToKey } from '@/shared/i18n'
import { formatMinutes } from '@/shared/lib/duration'
import type { DrillRef } from '@/shared/model/drill'
import type { DecisionRow, StateTile } from '../model/center'

const props = withDefaults(
  defineProps<{
    tiles: readonly StateTile[]
    decisions: readonly DecisionRow[]
    incident?: IncidentSummary | null
    scopePath?: readonly number[] | null
    contained?: { inScope: number; isolated: number } | null
    /** Можно ли открыть объект строки (есть окно или страница). */
    canOpen?: (ref: DrillRef) => boolean
  }>(),
  { incident: null, scopePath: null, contained: null, canOpen: () => false },
)
const emit = defineEmits<{ open: [ref: DrillRef]; 'open-map': [incidentId: string] }>()
const { t, te, d } = useI18n()
const K = 'widgets.controlCenter'
const dash = '—'

const TONE = { normal: 'success', attention: 'attention', danger: 'danger', critical: 'critical' } as const
const tone = (x: keyof typeof TONE) => `var(--ant-status-${TONE[x]})`

function tileMain(x: StateTile): string {
  const v = x.data
  switch (x.id) {
    case 'production':
      return t(`${K}.production.main`, { items: t('plural.items', { n: Number(v.inProgress) }, Number(v.inProgress)) })
    case 'quality':
      return v.incident ? t(`${K}.quality.mainIncident`, { items: t('plural.items', { n: Number(v.scope) }, Number(v.scope)) }) : t(`${K}.quality.mainNcs`, { n: v.ncs })
    case 'flow':
      return v.bottleneck ? t(`${K}.flow.mainBottleneck`, { step: v.bottleneck }) : t(`${K}.flow.mainOk`)
    default:
      return v.n ? t(`${K}.decisions.main`, { n: v.n }) : t(`${K}.decisions.none`)
  }
}
function tileSub(x: StateTile): string {
  const v = x.data
  switch (x.id) {
    case 'production':
      return [t(`${K}.production.queue`, { n: v.queue }), Number(v.delayed) ? t(`${K}.production.delayed`, { n: v.delayed }) : ''].filter(Boolean).join(' · ')
    case 'quality':
      return v.incident && Number(v.confirmed) ? t(`${K}.quality.confirmed`, { n: v.confirmed }) : ''
    case 'flow':
      return v.wait ? t(`${K}.flow.wait`, { wait: v.wait }) : Number(v.anomalies) ? t(`${K}.flow.anomalies`, { n: v.anomalies }) : ''
    default:
      return Number(v.overdue) ? t(`${K}.decisions.overdue`, { n: v.overdue }) : ''
  }
}

/** Текст строки — теми же ключами, что прежние «Требует внимания» и «Тревоги» (liveMap.*). */
function rowText(r: DecisionRow): string {
  const e = r.entry as Partial<Omit<AttentionEntry, 'kind'>> & Partial<Omit<AlertEntry, 'kind'>>
  const overdueDecision = () =>
    t('liveMap.attention.overdueDecision', {
      target: e.target ?? dash,
      overdue: formatMinutes(t, e.overdue_minutes ?? 0),
      items: t('plural.items', { n: e.items ?? 0 }, e.items ?? 0),
      operations: t('plural.operations', { n: e.operations ?? 0 }, e.operations ?? 0),
    })
  switch (r.kind) {
    case 'overdue_decision':
      return overdueDecision()
    case 'unverified_measures':
      return t('liveMap.attention.unverifiedMeasures', { n: e.n ?? 0 })
    case 'temporary_measures':
      return t('liveMap.attention.temporaryMeasures', { n: e.n ?? 0 })
    case 'overdue_isolation':
      return t('liveMap.alerts.overdueIsolation', { item: e.item ?? dash })
    case 'not_moved_to_isolator':
      return t('liveMap.alerts.notMovedToIsolator', { item: e.item ?? dash })
    case 'gate_overdue':
      return t('liveMap.alerts.gateOverdue', { gate: e.gate ?? dash })
    case 'escalation':
      return `${t('common.notifications.escalation')}: ${overdueDecision()}`
    case 'integrity_violation':
      return t('liveMap.alerts.integrityViolation')
    default: {
      const key = `liveMap.alerts.${codeToKey(String(r.kind))}`
      return te(key) ? t(key) : String(r.kind)
    }
  }
}

const STAGES = statusDictionaries.investigation_stage.values as Record<string, { label: string; tone: string }>
const stage = computed(() => (props.incident?.stage ? (STAGES[props.incident.stage] ?? null) : null))
const factor = computed(() => props.incident?.common_factor?.label || props.incident?.common_factor?.value || '')
</script>

<template>
  <div class="center" data-testid="control-center">
    <!-- Состояние предприятия — четыре слова. -->
    <ul class="tiles">
      <li v-for="x in tiles" :key="x.id" class="tile" :data-tile="x.id" :data-tone="x.tone" :style="{ '--tone': tone(x.tone) }">
        <p class="tile-title">{{ t(`${K}.${x.id}.title`) }}</p>
        <p class="tile-main ant-wrap">{{ tileMain(x) }}</p>
        <p v-if="tileSub(x)" class="meta ant-wrap">{{ tileSub(x) }}</p>
      </li>
    </ul>

    <div class="body">
      <!-- Требует моего решения — главное на экране. -->
      <section class="decisions" data-testid="decisions">
        <h3 class="h">{{ t(`${K}.decisionsTitle`) }}<span v-if="decisions.length" class="count"> · {{ decisions.length }}</span></h3>
        <p v-if="!decisions.length" class="meta">{{ t(`${K}.decisionsEmpty`) }}</p>
        <ol v-else class="rows">
          <li v-for="r in decisions" :key="r.id" class="row" :data-kind="r.kind" :style="{ '--tone': tone(r.tone) }">
            <time v-if="r.at" class="meta" :datetime="r.at">{{ d(new Date(r.at), 'time') }}</time>
            <button v-if="r.ref && canOpen(r.ref)" type="button" class="link ant-wrap" @click="emit('open', r.ref)">{{ rowText(r) }}</button>
            <span v-else class="ant-wrap">{{ rowText(r) }}</span>
          </li>
        </ol>
      </section>

      <!-- Главный текущий инцидент — связывает столы контролёра, мастера и технолога. -->
      <section v-if="incident" class="incident" data-testid="main-incident">
        <p class="meta">{{ t(`${K}.incidentTitle`) }}</p>
        <h3 class="h ant-wrap">{{ incident.label }}</h3>
        <p v-if="factor" class="meta ant-wrap">{{ factor }}</p>
        <dl class="facts">
          <div>
            <dt>{{ t(`${K}.scope`) }}</dt>
            <dd data-testid="incident-scope">{{ scopePath && scopePath.length > 1 ? scopePath.join(' → ') : incident.size }}</dd>
          </div>
          <div v-if="contained && contained.inScope">
            <dt>{{ t(`${K}.contained`) }}</dt>
            <dd data-testid="incident-contained" :data-all="contained.isolated === contained.inScope || undefined">{{ contained.isolated }} / {{ contained.inScope }}</dd>
          </div>
          <div v-if="stage">
            <dt>{{ t(`${K}.stage`) }}</dt>
            <dd>{{ stage.label }}</dd>
          </div>
        </dl>
        <p v-if="incident.next_step" class="next ant-wrap">{{ t('widgets.analysis.investigations.next', { what: incident.next_step }) }}</p>
        <button type="button" class="link" data-testid="open-map" @click="emit('open-map', incident.incident_id)">{{ t(`${K}.openMap`) }} →</button>
      </section>
    </div>
  </div>
</template>

<style scoped>
.center {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-8);
  min-width: 0;
}

p,
h3,
dl,
dd {
  margin: 0;
}

.meta {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.h {
  font-size: var(--ant-fs-title);
}

.count {
  color: var(--ant-text-3);
  font-weight: normal;
}

.tiles {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: var(--ant-space-6);
  margin: 0;
  padding: 0;
  list-style: none;
}

.tile {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  min-width: 0;
  padding-top: var(--ant-space-3);
  border-top: 3px solid var(--tone);
}

.tile-title {
  color: var(--ant-text-2);
  font-size: var(--ant-fs-meta);
}

.tile-main {
  font-size: var(--ant-fs-title);
  font-weight: var(--ant-fw-bold);
}

.body {
  display: grid;
  grid-template-columns: minmax(0, 3fr) minmax(260px, 2fr);
  gap: var(--ant-space-8);
}

@media (max-width: 1100px) {
  .body {
    grid-template-columns: minmax(0, 1fr);
  }
}

.decisions,
.incident {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-3);
  min-width: 0;
}

.rows {
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 0;
  list-style: none;
}

.row {
  display: flex;
  gap: var(--ant-space-3);
  align-items: baseline;
  min-width: 0;
  padding: var(--ant-space-3) 0 var(--ant-space-3) var(--ant-space-3);
  border-top: 1px solid var(--ant-border);
  box-shadow: inset 3px 0 0 var(--tone);
}

.link {
  padding: 0;
  border: 0;
  background: none;
  color: var(--ant-accent);
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.link:hover {
  text-decoration: underline;
}

.incident {
  padding-left: var(--ant-space-4);
  border-left: 3px solid var(--ant-status-danger);
}

.facts {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-3) var(--ant-space-6);
}

.facts dt {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.facts dd {
  font-size: var(--ant-fs-title);
  font-weight: var(--ant-fw-bold);
  font-variant-numeric: tabular-nums;
}

.facts dd[data-all] {
  color: var(--ant-status-success-text);
}

.next {
  font-weight: var(--ant-fw-bold);
}
</style>
