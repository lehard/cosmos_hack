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
const MAX_CHIPS = 6

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
    <!-- Инцидент: область риска и физическая локализация — одна строка над потоком. -->
    <section v-if="incident" class="incident" data-testid="flow-incident">
      <p class="incident-title ant-wrap">{{ incident.label }}</p>
      <p class="incident-line ant-wrap">
        <span>{{ t(`${K}.inScope`, { items: t('plural.items', { n: incident.size }, incident.size) }) }}</span>
        <span v-if="scopePath && scopePath.length > 1" class="path" data-testid="flow-path">{{ scopePath.join(' → ') }}</span>
        <span v-if="contained.inScope" class="contained" :data-all="contained.isolated === contained.inScope || undefined" data-testid="flow-contained">
          {{ t(`${K}.contained`, { n: contained.isolated, m: contained.inScope }) }}
        </span>
      </p>
      <p v-if="incident.basis" class="meta ant-wrap">{{ incident.basis }}</p>
    </section>

    <!-- Поток: участки в порядке схемы. -->
    <ol class="sections" :aria-label="t(`${K}.title`)">
      <li v-for="s in states" :key="s.id" class="section" :data-status="s.status" :data-section="s.id" :style="{ '--tone': tone(s.status) }">
        <button type="button" class="section-head" :title="t(`${K}.openScheme`)" @click="emit('open-section', s)">
          <span class="name ant-clamp-2">{{ s.name }}</span>
          <span class="status"><span class="dot" aria-hidden="true" />{{ t(`${K}.status.${s.status}`) }}</span>
        </button>
        <p class="main"><strong class="num">{{ s.inProgress }}</strong> {{ t(`${K}.inProgress`) }}</p>
        <p class="meta">{{ t(`${K}.counts`, { queue: s.queue, passed: s.passed }) }}<template v-if="s.defects"> · {{ t(`${K}.defects`, { n: s.defects }) }}</template></p>
        <p v-if="s.inScope.length" class="scope-line ant-wrap" data-testid="section-scope">{{ t(`${K}.sectionScope`, { m: s.inScope.length, n: s.isolated }) }}</p>
        <p v-if="s.bottleneck !== null" class="warn ant-wrap" data-testid="section-bottleneck">{{ t(`${K}.bottleneck`, { wait: s.bottleneck }) }}</p>
        <p v-for="(a, ai) in s.anomalies" :key="ai" class="warn ant-wrap">{{ a }}</p>
        <ul v-if="s.items.length" class="chips">
          <li v-for="it in s.items.slice(0, MAX_CHIPS)" :key="it.item_id">
            <button type="button" class="chip" :data-incident="it.incident_status || undefined" :data-isolated="it.position === 'isolated' || undefined" @click="emit('open-item', it.item_id)">{{ it.label }}</button>
          </li>
          <li v-if="s.items.length > MAX_CHIPS" class="more">+{{ s.items.length - MAX_CHIPS }}</li>
        </ul>
      </li>
    </ol>
    <p v-if="!states.length" class="meta">{{ t(`${K}.noSections`) }}</p>

    <!-- Что сейчас произошло. -->
    <section v-if="recent.length" class="feed" data-testid="flow-feed">
      <p class="feed-title">{{ t(`${K}.feed`) }}</p>
      <ul>
        <li v-for="m in recent" :key="m.mark_id" class="ant-wrap" :data-kind="m.kind"><span class="meta">{{ d(new Date(m.at), 'time') }}</span> {{ m.title }}</li>
      </ul>
    </section>
  </div>
</template>

<style scoped>
.flow {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-6);
  min-width: 0;
}

p {
  margin: 0;
}

.meta {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.incident {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  padding-left: var(--ant-space-4);
  border-left: 3px solid var(--ant-status-danger);
}

.incident-title {
  font-size: var(--ant-fs-title);
  font-weight: var(--ant-fw-bold);
}

.incident-line {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-1) var(--ant-space-5);
}

.path {
  font-weight: var(--ant-fw-bold);
  font-variant-numeric: tabular-nums;
}

.contained {
  color: var(--ant-status-danger-text);
  font-weight: var(--ant-fw-bold);
}

.contained[data-all] {
  color: var(--ant-status-success-text);
}

/* Участки — ровные колонки: раскладка не прыгает при изменении чисел. */
.sections {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: var(--ant-space-5);
  margin: 0;
  padding: 0;
  list-style: none;
}

.section {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  min-width: 0;
  padding-top: var(--ant-space-3);
  border-top: 3px solid var(--tone);
  transition: border-color 0.4s ease;
}

.section-head {
  display: flex;
  flex-direction: column;
  gap: 2px;
  align-items: flex-start;
  padding: 0;
  border: 0;
  background: none;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.section-head:hover .name {
  text-decoration: underline;
}

.name {
  font-weight: var(--ant-fw-bold);
}

.status {
  display: inline-flex;
  gap: var(--ant-space-1);
  align-items: center;
  color: var(--ant-text-2);
  font-size: var(--ant-fs-meta);
}

.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--tone);
}

.main {
  margin-top: var(--ant-space-2);
}

.num {
  font-size: var(--ant-fs-title);
  font-variant-numeric: tabular-nums;
}

.scope-line {
  color: var(--ant-status-danger-text);
  font-weight: var(--ant-fw-bold);
}

.warn {
  color: var(--ant-status-attention-text);
  font-size: var(--ant-fs-meta);
}

.chips {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-1) var(--ant-space-2);
  margin: var(--ant-space-1) 0 0;
  padding: 0;
  list-style: none;
  font-size: var(--ant-fs-meta);
}

.chip {
  padding: 0;
  border: 0;
  background: none;
  color: var(--ant-accent);
  font: inherit;
  cursor: pointer;
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

.feed {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
}

.feed-title {
  font-weight: var(--ant-fw-bold);
}

.feed ul {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin: 0;
  padding: 0;
  list-style: none;
}
</style>
