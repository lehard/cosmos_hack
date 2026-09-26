<script setup lang="ts">
/**
 * «Карта производства» — режим руководителя по умолчанию (FR-1…9, FR-155): не
 * BPMN, а участки в порядке потока. У участка — главное число «в работе», тише —
 * очередь, прошло, дефекты; изделия инцидента на участке и сколько из них
 * локализовано; отклонение от нормы и узкое место; статус словом и цветом.
 * Сверху при инциденте — область риска (путь 34 → 13 → 6) и «локализовано N из M»;
 * снизу — что сейчас произошло (события полосы времени до текущего момента).
 * Раскладка постоянна: при ×60 меняются только числа и цвета. Щелчок по участку —
 * схема процесса (BPMN) — уровень глубже; по изделию — окно изделия.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { LiveMap, TimelineMark } from '@/shared/api/generated/model'
import { containment, sectionStates, sectionsOf, type SectionState, type SectionStatus } from '@/entities/live-map'

const props = withDefaults(
  defineProps<{
    map: LiveMap
    /** Путь области риска по версиям (34, 13, 6); нет — только текущий размер. */
    scopePath?: readonly number[] | null
    /** События полосы времени. */
    marks?: readonly TimelineMark[]
    /** Текущий момент (мс): «сейчас» или момент проигрывания. */
    now: number
  }>(),
  { scopePath: null, marks: () => [] },
)
const emit = defineEmits<{ 'open-section': [section: SectionState]; 'open-item': [itemId: string] }>()
const { t, d } = useI18n()

const sections = computed(() => sectionsOf(props.map.bpmn_xml))
const states = computed(() => sectionStates(props.map, sections.value))
const incident = computed(() => props.map.incident ?? null)
const contained = computed(() => containment(states.value))

const STATUS_TONE: Record<SectionStatus, string> = { normal: 'success', attention: 'attention', danger: 'danger', unknown: 'muted' }
const tone = (s: SectionStatus) => `var(--ant-status-${STATUS_TONE[s]})`
const K = 'widgets.liveMap.flow'
const MAX_CHIPS = 4

/** Одна фраза о главном: инцидент и локализация → узкое место → «штатно». */
const bottleneckSection = computed(() => states.value.find((x) => x.bottleneck !== null) ?? null)
/** Участок без движения — приглушён: пустое не кричит так же, как рабочее. */
const isIdle = (x: SectionState) => x.inProgress + x.queue + x.items.length === 0 && x.status === 'normal'
/** Под числом — одна строка: что не так на участке (изделия инцидента → узкое место → отклонения → дефекты). */
function note(x: SectionState): { text: string; tone: 'danger' | 'attention' } | null {
  if (x.inScope.length) return { text: t(`${K}.sectionScope`, { m: x.inScope.length, n: x.isolated }), tone: 'danger' }
  if (x.bottleneck !== null) return { text: t(`${K}.bottleneckShort`), tone: 'attention' }
  if (x.anomalies.length) return { text: t(`${K}.anomaliesShort`, { n: x.anomalies.length }), tone: 'attention' }
  if (x.defects) return { text: t(`${K}.defects`, { n: x.defects }), tone: 'attention' }
  return null
}

/** Что сейчас произошло: последние события до текущего момента, свежие сверху. */
const recent = computed(() =>
  [...props.marks]
    .filter((m) => m.title && Date.parse(m.at) <= props.now)
    .sort((a, b) => b.at.localeCompare(a.at))
    .slice(0, 5),
)
</script>

<template>
  <div class="flow" data-testid="production-flow">
    <!-- Одна фраза о главном. -->
    <header class="lead">
      <template v-if="incident">
        <p class="lead-line ant-wrap" data-testid="flow-incident">
          <span class="dot danger" aria-hidden="true" />
          <span class="lead-title">{{ incident.label }}</span>
          <span class="lead-sep">·</span>
          <span>{{ t(`${K}.inScope`, { items: t('plural.items', { n: incident.size }, incident.size) }) }}</span>
          <span v-if="contained.inScope" class="lead-contained" :data-all="contained.isolated === contained.inScope || undefined" data-testid="flow-contained">
            · {{ t(`${K}.contained`, { n: contained.isolated, m: contained.inScope }) }}
          </span>
        </p>
        <p class="lead-sub">
          <span v-if="scopePath && scopePath.length > 1" class="path" data-testid="flow-path">{{ scopePath.join(' → ') }}</span>
          <span v-if="incident.basis" class="ant-wrap"> {{ incident.basis }}</span>
        </p>
      </template>
      <p v-else-if="bottleneckSection" class="lead-line ant-wrap" data-testid="flow-lead">
        <span class="dot attention" aria-hidden="true" />
        <span class="lead-title">{{ t(`${K}.leadBottleneck`, { name: bottleneckSection.name }) }}</span>
        <span v-if="bottleneckSection.bottleneck" class="lead-muted"> · {{ t(`${K}.leadWait`, { wait: bottleneckSection.bottleneck }) }}</span>
      </p>
      <p v-else class="lead-line" data-testid="flow-lead"><span class="dot normal" aria-hidden="true" /><span class="lead-title">{{ t(`${K}.leadOk`) }}</span></p>
    </header>

    <!-- Поток: станции одной линией, слева направо. -->
    <ol class="stations" :style="{ '--n': Math.max(states.length, 1) }" :aria-label="t(`${K}.title`)">
      <li
        v-for="x in states"
        :key="x.id"
        class="station"
        :class="{ idle: isIdle(x) }"
        :data-status="x.status"
        :data-section="x.id"
        :style="{ '--tone': tone(x.status) }"
      >
        <button type="button" class="section-head" :title="t(`${K}.openScheme`)" @click="emit('open-section', x)">
          <span class="name ant-clamp-2">{{ x.name }}</span>
        </button>
        <p class="big"><span class="num">{{ x.inProgress }}</span></p>
        <p class="caption">{{ t(`${K}.inProgress`) }}</p>
        <p class="stats">{{ t(`${K}.counts`, { queue: x.queue, passed: x.passed }) }}</p>
        <p
          v-if="note(x)"
          class="note ant-wrap"
          :data-tone="note(x)!.tone"
          :title="x.anomalies.join('\n') || undefined"
          :data-testid="x.inScope.length ? 'section-scope' : x.bottleneck !== null ? 'section-bottleneck' : undefined"
        >{{ note(x)!.text }}</p>
        <ul v-if="x.items.length" class="chips">
          <li v-for="it in x.items.slice(0, MAX_CHIPS)" :key="it.item_id">
            <button type="button" class="chip" :data-incident="it.incident_status || undefined" :data-isolated="it.position === 'isolated' || undefined" @click="emit('open-item', it.item_id)">{{ it.label }}</button>
          </li>
          <li v-if="x.items.length > MAX_CHIPS" class="more">+{{ x.items.length - MAX_CHIPS }}</li>
        </ul>
      </li>
    </ol>
    <p v-if="!states.length" class="caption">{{ t(`${K}.noSections`) }}</p>

    <!-- Что сейчас произошло — тихо, ниже. -->
    <section v-if="recent.length" class="feed" data-testid="flow-feed">
      <p class="feed-title">{{ t(`${K}.feed`) }}</p>
      <ul>
        <li v-for="m in recent" :key="m.mark_id" class="ant-wrap" :data-kind="m.kind"><span class="time">{{ d(new Date(m.at), 'time') }}</span>{{ m.title }}</li>
      </ul>
    </section>
  </div>
</template>

<style scoped>
.flow {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-8);
  min-width: 0;
  padding: var(--ant-space-2) var(--ant-space-2) var(--ant-space-4);
}

p {
  margin: 0;
}

/* Одна фраза о главном. */
.lead {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
}

.lead-line {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-2);
  align-items: baseline;
  font-size: var(--ant-fs-title);
}

.lead-title {
  font-weight: var(--ant-fw-bold);
}

.lead-sep,
.lead-muted,
.lead-sub {
  color: var(--ant-text-3);
}

.lead-sub {
  padding-left: calc(10px + var(--ant-space-2));
}

.lead-contained {
  color: var(--ant-status-danger-text);
  font-weight: var(--ant-fw-bold);
}

.lead-contained[data-all] {
  color: var(--ant-status-success-text);
}

.path {
  color: var(--ant-text);
  font-weight: var(--ant-fw-bold);
  font-variant-numeric: tabular-nums;
}

.dot {
  flex: none;
  align-self: center;
  width: 10px;
  height: 10px;
  border-radius: 50%;
}

.dot.danger {
  background: var(--ant-status-danger);
}

.dot.attention {
  background: var(--ant-status-attention);
}

.dot.normal {
  background: var(--ant-status-success);
}

/* Поток — одна линия станций; раскладка постоянна, меняются только числа и цвета. */
.stations {
  display: grid;
  grid-template-columns: repeat(var(--n), minmax(0, 1fr));
  gap: var(--ant-space-8);
  margin: 0;
  padding: 0;
  list-style: none;
}

@media (max-width: 1100px) {
  .stations {
    grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  }
}

.station {
  position: relative;
  display: flex;
  flex-direction: column;
  min-width: 0;
  padding-top: var(--ant-space-4);
  border-top: 2px solid var(--tone);
  transition:
    border-color 0.6s ease,
    opacity 0.6s ease;
}

/* Тонкая стрелка потока между станциями. */
.station + .station::before {
  position: absolute;
  top: -11px;
  left: calc(-1 * var(--ant-space-8) / 2 - 6px);
  color: var(--ant-text-3);
  font-size: 14px;
  content: '›';
}

.station.idle {
  opacity: 0.45;
}

.station[data-status='normal'] {
  border-top-color: var(--ant-border-strong);
}

.section-head {
  align-self: flex-start;
  min-height: 2.6em;
  padding: 0;
  border: 0;
  background: none;
  color: var(--ant-text-2);
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.section-head:hover {
  color: var(--ant-text);
}

.big {
  margin-top: var(--ant-space-3);
  line-height: 1;
}

.num {
  font-size: var(--ant-fs-display);
  font-weight: var(--ant-fw-bold);
  font-variant-numeric: tabular-nums;
}

.caption {
  margin-top: var(--ant-space-1);
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.stats {
  margin-top: var(--ant-space-3);
  color: var(--ant-text-2);
  font-size: var(--ant-fs-meta);
  font-variant-numeric: tabular-nums;
}

.note {
  margin-top: var(--ant-space-2);
  font-size: var(--ant-fs-meta);
  font-weight: var(--ant-fw-bold);
}

.note[data-tone='danger'] {
  color: var(--ant-status-danger-text);
}

.note[data-tone='attention'] {
  color: var(--ant-status-attention-text);
}

.chips {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-1) var(--ant-space-2);
  margin: var(--ant-space-3) 0 0;
  padding: 0;
  list-style: none;
  font-size: var(--ant-fs-meta);
}

.chip {
  padding: 0;
  border: 0;
  background: none;
  color: var(--ant-text-3);
  font: inherit;
  cursor: pointer;
}

.chip:hover {
  color: var(--ant-accent);
}

.chip[data-incident='confirmed'],
.chip[data-incident='suspect'] {
  color: var(--ant-status-danger-text);
  font-weight: var(--ant-fw-bold);
}

.chip[data-isolated] {
  text-decoration: line-through;
}

.more {
  color: var(--ant-text-3);
}

/* Что сейчас произошло — тихий список. */
.feed {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
  max-width: 720px;
}

.feed-title {
  color: var(--ant-text-2);
  font-size: var(--ant-fs-meta);
}

.feed ul {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  margin: 0;
  padding: 0;
  list-style: none;
}

.time {
  display: inline-block;
  min-width: 4.5em;
  color: var(--ant-text-3);
  font-variant-numeric: tabular-nums;
}
</style>
