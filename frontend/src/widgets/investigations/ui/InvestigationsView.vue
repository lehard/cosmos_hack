<script setup lang="ts">
/**
 * «Мои расследования» (UI-32): карточка на инцидент — что расследуем, сколько
 * изделий в области сейчас и сколько было при открытии, идёт ли расследование.
 * Выбранная карточка — рабочее пространство раздела (область риска, дорожки,
 * гипотезы). Открытые — первыми, свежие — раньше; карточки — полосой над
 * рабочим пространством (на узком экране — одна под другой).
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { FACTOR_TEXT, type IncidentKnown } from '@/entities/incident'
import type { IncidentSummary } from '@/shared/api/generated/model'
import { statusAxes, statusDictionaries } from '@/shared/api/generated/statuses'

const props = defineProps<{ incidents: readonly IncidentSummary[]; selected: string | null }>()
const emit = defineEmits<{ select: [incidentId: string] }>()
defineSlots<{
  /** Несоответствия расследования — контейнер рисует ссылки на их окна. */
  ncs(props: { incident: IncidentSummary }): unknown
  /** Главное действие «что делать дальше» — кнопка контейнера. */
  cta(props: { incident: IncidentSummary }): unknown
}>()

const { t, d } = useI18n()

const sorted = computed(() =>
  [...props.incidents].sort((a, b) => Number(a.status !== 'open') - Number(b.status !== 'open') || b.opened_at.localeCompare(a.opened_at)),
)
/** Выбранное — шапкой; нет выбора — первое открытое (порядок sorted). */
const current = computed(() => sorted.value.find((i) => i.incident_id === props.selected) ?? sorted.value[0] ?? null)
const others = computed(() => sorted.value.filter((i) => i !== current.value))
const factor = (i: IncidentSummary) =>
  i.common_factor ? t('riskScope.commonFactor', { factor: `${t(FACTOR_TEXT[i.common_factor.factor])}: ${i.common_factor.label || i.common_factor.value}` }) : null

/** Стадия расследования — словарь контракта investigation_stage (подпись и тон). */
const STAGES = statusDictionaries.investigation_stage.values as Record<string, { label: string; tone: string }>
const stage = (i: IncidentSummary) => (i.stage ? (STAGES[i.stage] ?? null) : null)

/** Счётчики «что известно» — только ненулевые, в порядке важности. */
const KNOWN: readonly IncidentKnown[] = ['confirmed', 'suspect', 'unknown', 'excluded']
const KNOWN_TEXT: Record<IncidentKnown, string> = {
  confirmed: 'statuses.incident.confirmed',
  suspect: 'statuses.incident.suspect',
  unknown: 'riskScope.noData',
  excluded: 'widgets.analysis.riskScope.excludedWithBasis',
}
const counts = (i: IncidentSummary) =>
  i.counts ? KNOWN.filter((k) => i.counts[k] > 0).map((k) => ({ k, n: i.counts[k], tone: `var(--ant-status-${statusAxes.incident.values[k].tone})` })) : []
</script>

<template>
  <div class="investigations" data-testid="investigations">
    <!-- Выбранное расследование — шапка рабочего пространства: что расследуем, где мы, что делать дальше. -->
    <section
      v-if="current"
      class="hero"
      :data-incident="current.incident_id"
      :data-status="current.status"
      aria-current="true"
    >
      <header class="head">
        <div class="what">
          <h3 class="label ant-wrap">{{ current.label }}</h3>
          <p v-if="factor(current)" class="muted ant-wrap">{{ factor(current) }}</p>
        </div>
        <span v-if="stage(current)" class="stage" :style="{ '--tone': `var(--ant-status-${stage(current)!.tone})`, '--tone-soft': `var(--ant-status-${stage(current)!.tone}-soft)` }" data-testid="stage">{{ stage(current)!.label }}</span>
      </header>

      <div class="scope">
        <span v-if="counts(current).length" class="counts" data-testid="counts">
          <span v-for="c in counts(current)" :key="c.k" class="count" :data-known="c.k" :style="{ '--tone': c.tone }"><strong>{{ c.n }}</strong> {{ t(KNOWN_TEXT[c.k]).toLowerCase() }}</span>
        </span>
        <span class="now">
          {{ t('widgets.analysis.investigations.nowInScope', { items: t('plural.items', { n: current.size }, current.size) }) }}
          <span v-if="current.initial_size !== current.size" class="muted" data-testid="path">{{ t('widgets.analysis.investigations.path', { from: current.initial_size, to: current.size }) }}</span>
        </span>
      </div>

      <div v-if="current.status === 'open' && (current.next_step || $slots.cta)" class="next" data-testid="next-block">
        <p class="next-title">{{ t('widgets.analysis.investigations.whatNext') }}</p>
        <p v-if="current.next_step" class="next-text ant-wrap" data-testid="next-step">{{ current.next_step }}</p>
        <slot name="cta" :incident="current" />
      </div>

      <div v-if="current.nc_ids?.length" class="ncs" data-testid="incident-ncs">
        <span class="ncs-title">{{ t('widgets.analysis.investigations.ncs') }}</span>
        <slot name="ncs" :incident="current" />
      </div>

      <details v-if="current.status === 'open' && current.close_blockers?.length" class="blockers" data-testid="close-blockers">
        <summary>{{ t('widgets.analysis.investigations.cannotClose') }} ({{ current.close_blockers.length }})</summary>
        <p v-for="(b, bi) in current.close_blockers" :key="bi" class="blocker ant-wrap">{{ b.text }}</p>
      </details>

      <p class="state muted" data-testid="state">
        {{ t(current.status === 'open' ? 'widgets.analysis.investigations.open' : 'widgets.analysis.investigations.closed') }}
        · {{ t('widgets.analysis.investigations.opened', { time: d(new Date(current.opened_at), 'dateTime') }) }}
        <template v-if="current.last_event_at"> · {{ t('widgets.analysis.investigations.lastEvent', { time: d(new Date(current.last_event_at), 'dateTime') }) }}</template>
      </p>
    </section>

    <!-- Другие расследования — переключателями, не второй копией. -->
    <nav v-if="others.length" class="others" :aria-label="t('widgets.analysis.investigations.others')">
      <span class="muted">{{ t('widgets.analysis.investigations.others') }}:</span>
      <button v-for="i in others" :key="i.incident_id" type="button" class="other" :data-incident="i.incident_id" :data-status="i.status" @click="emit('select', i.incident_id)">
        <span class="ant-ellipsis">{{ i.label }}</span>
        <span class="muted">· {{ i.status === 'open' ? t('plural.items', { n: i.size }, i.size) : t('widgets.analysis.investigations.closed') }}</span>
      </button>
    </nav>
  </div>
</template>

<style scoped>
.investigations {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-3);
  min-width: 0;
}

p,
h3 {
  margin: 0;
}

.muted {
  color: var(--ant-text-3);
}

.hero {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-3);
  min-width: 0;
}

.head {
  display: flex;
  gap: var(--ant-space-3);
  align-items: flex-start;
  justify-content: space-between;
}

.what {
  min-width: 0;
}

.label {
  font-size: var(--ant-fs-title);
}

.stage {
  flex: none;
  padding: 2px var(--ant-space-3);
  border: 1px solid var(--tone);
  border-radius: var(--ant-radius-pill);
  background: var(--tone-soft);
  font-weight: var(--ant-fw-bold);
  white-space: nowrap;
}

.scope {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-2) var(--ant-space-4);
  align-items: center;
}

.counts {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-2);
}

.count {
  padding: 2px var(--ant-space-2);
  border-left: 4px solid var(--tone);
  border-radius: var(--ant-radius-sm);
  background: var(--ant-surface-subtle);
}

.now {
  font-weight: var(--ant-fw-bold);
}

/* Что делать дальше — главное действие страницы. */
.next {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
  align-items: flex-start;
  padding: var(--ant-space-3) var(--ant-space-4);
  border-left: 4px solid var(--ant-accent);
  border-radius: var(--ant-radius-md);
  background: var(--ant-accent-soft);
}

.next-title {
  color: var(--ant-text-2);
  font-size: var(--ant-fs-meta);
  font-weight: var(--ant-fw-bold);
}

.next-text {
  font-size: var(--ant-fs-title);
  font-weight: var(--ant-fw-bold);
}

.ncs {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-1);
  align-items: baseline;
}

.ncs-title {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.blockers summary {
  color: var(--ant-status-attention-text);
  font-size: var(--ant-fs-meta);
  cursor: pointer;
}

.blocker {
  padding-left: var(--ant-space-4);
  color: var(--ant-text-2);
  font-size: var(--ant-fs-meta);
}

.state {
  font-size: var(--ant-fs-meta);
}

.others {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-2);
  align-items: center;
  padding-top: var(--ant-space-2);
  border-top: 1px solid var(--ant-border);
  font-size: var(--ant-fs-meta);
}

.other {
  display: inline-flex;
  gap: var(--ant-space-1);
  max-width: 320px;
  padding: 2px var(--ant-space-2);
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-pill);
  background: var(--ant-surface);
  color: inherit;
  font: inherit;
  cursor: pointer;
}

.other:hover {
  background: var(--ant-surface-hover);
}
</style>
