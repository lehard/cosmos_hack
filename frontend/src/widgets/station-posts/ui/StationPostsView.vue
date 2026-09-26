<script setup lang="ts">
/**
 * Участок — представление (PRD §3a «Мастер участка — Участок», FR-81, UJ-7):
 * операции цеха — очередь, в работе, длительность против нормы, повторные
 * выполнения против лимита доработок, незавершённые операции, аномалии; рядом с
 * растущей очередью — факты участка («почему растёт очередь»); посты —
 * назначенный и присутствие, текущая деталь, оборудование и текущее выполнение.
 * Элементы и токены дизайн-системы «Главный» (shared/ui/README.md).
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NAlert, NTag } from 'naive-ui'
import type { RefEquipment } from '@/entities/equipment'
import { toolLifeOf } from '@/entities/equipment'
import { normText, overNorm, type ProcessStep } from '@/entities/live-map'
import { formatValue, MetricNumber, type MetricValue } from '@/entities/metric'
import { PRESENCE_TEXT, presenceTagType } from '@/entities/workplace'
import type { Density } from '@/shared/config/widget'
import { naiveSizeOf } from '@/shared/config/widget'
import { useProblemText } from '@/shared/i18n/problem'
import { ActionButton, DataTable, EmptyState, SectionPanel } from '@/shared/ui'
import { minutesOf } from '../model/norms'
import { queueFacts, type EntryItem, type IncomingTask, type StationPost, type StationStep } from '../model/station'
import { anomalyText, factText } from '../model/texts'
import PostRun from './PostRun.vue'


const props = withDefaults(
  defineProps<{
    /** Название цеха участка; null — цех не определён, показан весь завод. */
    workshopName: string | null
    steps: readonly StationStep[] | null
    stepsError?: unknown
    posts: readonly StationPost[] | null
    postsError?: unknown
    registry?: readonly RefEquipment[] | null
    stepIndex: ReadonlyMap<string, ProcessStep>
    /** Момент просмотра (для «идёт N мин»). */
    at: Date
    density?: Density
    /** Задачи приёмки — форма «Принять в цех» прямо здесь. */
    incoming?: readonly IncomingTask[]
    /** Изделия на входе участка. */
    entryItems?: readonly EntryItem[]
    /** seq для basis_seq приёмки (голова журнала при чтении задач, AD-39). */
    basisSeq?: number
    canAct?: boolean
  }>(),
  { stepsError: undefined, postsError: undefined, registry: null, density: 'comfortable', incoming: () => [], entryItems: () => [], basisSeq: 0, canAct: true },
)
const emit = defineEmits<{ item: [itemId: string]; node: [stepKey: string]; workplace: [workplaceId: string]; person: [personId: string] }>()
const { t, te, n } = useI18n()
const problemText = useProblemText()
const size = computed(() => naiveSizeOf(props.density))

const value = (v: MetricValue) => formatValue({ t, n: (x, f) => n(x, f) }, v)
const durationOver = (s: StationStep) => overNorm(minutesOf(s.meanDuration), s.step.norm)
/** Показатели операции, которые пришли: пустые строки «нет данных» не показываем. */
const hasFacts = (s: StationStep) => !!(s.meanDuration || s.reworkRuns || s.unfinished)
const factLine = (s: StationStep) => queueFacts(s, props.posts ?? [], props.registry).map((f) => factText(t, f, value))
</script>

<template>
  <div class="station" data-testid="station-view">
    <p class="meta" data-testid="workshop">
      {{ workshopName ? t('widgets.shopFloor.station.workshop', { name: workshopName }) : t('widgets.shopFloor.station.noWorkshop') }}
    </p>

    <!-- Пришло и ждёт: что поступило на участок. Принимают в «Задачах». -->
    <section class="block" data-testid="incoming">
      <h3 class="heading">{{ t('stationDesk.incoming') }}</h3>
      <p v-if="!incoming.length && !entryItems.length" class="quiet" data-testid="nothing-incoming">{{ t('stationDesk.nothingIncoming') }}</p>
      <ul v-else class="lines">
        <li v-for="r in incoming" :key="r.taskId" class="line" :data-incoming="r.itemId">
          <button type="button" class="name" :title="t('common.actions.openPassport')" @click="emit('item', r.itemId)">{{ r.title.replace(/^Принять в цех\s*/, '') }}</button>
          <span class="quiet">ждёт приёмки — в «Задачах»</span>
        </li>
        <li v-for="i in entryItems" :key="i.item_id" class="line" data-testid="entry-items">
          <button type="button" class="name" :title="t('common.actions.openPassport')" @click="emit('item', i.item_id)">{{ i.label }}</button>
          <span class="quiet">{{ t('stationDesk.atEntry') }}</span>
        </li>
      </ul>
    </section>

    <!-- Посты: кто стоит, что на посту, как оборудование. -->
    <section class="block" data-testid="posts">
      <h3 class="heading">{{ t('stationDesk.atPosts') }}</h3>
      <NAlert v-if="postsError && !posts" type="error" :bordered="false">{{ problemText(postsError) }}</NAlert>
      <p v-else-if="posts && !posts.length" class="quiet">{{ t('empty.noRecords') }}</p>
      <ul v-else-if="posts" class="lines">
        <li v-for="p in posts" :key="p.post.workplace_id" class="line post" :data-workplace="p.post.workplace_id" :data-presence="p.post.presence">
          <button type="button" class="name" data-testid="open-post" @click="emit('workplace', p.post.workplace_id)">{{ p.post.station }}</button>
          <span class="cols">
            <button v-if="p.post.assigned" type="button" class="link" data-testid="open-person" @click="emit('person', p.post.assigned.person_id)">{{ p.post.assigned.display }}</button>
            <span v-else class="quiet">свободен</span>
            <button v-if="p.post.current_item" type="button" class="link" data-testid="current-item" @click="emit('item', p.post.current_item.item_id)">{{ p.post.current_item.label }}</button>
            <template v-for="e in p.equipment" :key="e.equipment_id">
              <span v-if="e.execution !== 'unknown'" class="state" :data-tone="e.condition" :data-equipment="e.equipment_id">{{ t(`widgets.shopFloor.execution.${e.execution}`) }}</span>
            </template>
          </span>
          <template v-for="e in p.equipment" :key="`w-${e.equipment_id}`">
            <p v-for="w in e.warnings" :key="`${w.kind}-${w.since}`" class="warning" :title="w.text" data-testid="equipment-warning">{{ w.text }}</p>
            <PostRun v-if="e.current_run_id" :run-id="e.current_run_id" :steps="stepIndex" :at="at" :current-item="p.post.current_item ?? null" @item="(id) => emit('item', id)" />
          </template>
        </li>
      </ul>
    </section>

    <!-- Операции участка: одна строка на операцию, лишнего нет. -->
    <section class="block" data-testid="steps">
      <h3 class="heading">{{ t('widgets.shopFloor.station.steps') }}</h3>
      <NAlert v-if="stepsError && !steps" type="error" :bordered="false">{{ problemText(stepsError) }}</NAlert>
      <p v-else-if="steps && !steps.length" class="quiet">{{ t('widgets.shopFloor.station.noSteps') }}</p>
      <ul v-else-if="steps" class="lines">
        <li v-for="s in steps" :key="s.step.stepKey" class="line step" :data-step="s.step.stepKey" :data-growing="s.growing || undefined">
          <button type="button" class="name" data-testid="open-operation" @click="emit('node', s.step.stepKey)">{{ s.step.name }}</button>
          <span class="cols" data-testid="counters">
            <span v-if="s.counters" class="count" :data-tone="s.growing ? 'warn' : undefined" data-testid="queue">в очереди {{ s.counters.queue }}</span>
            <span v-if="s.counters" class="count" data-testid="in-work">в работе {{ s.counters.in_progress }}</span>
            <span v-if="s.counters?.nonconformities" class="state" data-tone="fault">{{ t('plural.nonconformities', { n: s.counters.nonconformities }, s.counters.nonconformities) }}</span>
            <span v-if="s.bottleneck" class="state" data-tone="warning" data-testid="bottleneck">{{ t('liveMap.bottleneck') }}</span>
            <span v-for="a in s.anomalies" :key="a.kind" class="state" data-tone="warning" :data-anomaly="a.kind" :title="anomalyText(t, te, a)">{{ anomalyText(t, te, a) }}</span>
            <span v-if="s.step.specialProcess" class="quiet">{{ t('widgets.shopFloor.station.specialProcess') }}</span>
          </span>
          <p v-if="s.meanDuration" class="quiet" data-testid="duration">
            {{ t('widgets.shopFloor.station.meanDuration') }} <MetricNumber :value="s.meanDuration" /> · {{ normText(t, s.step.norm) }}
            <span v-if="durationOver(s)" class="state" data-tone="fault">{{ t('widgets.shopFloor.station.overNorm') }}</span>
          </p>
          <NAlert v-if="s.growing" type="warning" :bordered="false" :title="t('widgets.shopFloor.why.title', { step: s.step.name })" data-testid="why">
            <p v-for="(line, i) in factLine(s)" :key="i" class="ant-wrap" data-testid="why-fact">{{ line }}</p>
            <p v-if="!factLine(s).length" class="ant-wrap">{{ t('widgets.shopFloor.why.none') }}</p>
          </NAlert>
        </li>
      </ul>
    </section>
  </div>
</template>

<style scoped>
.station {
  display: flex;
  flex-direction: column;
  gap: calc(var(--ant-space-3) * 3);
  min-width: 0;
}

.meta,
.quiet {
  margin: 0;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.block {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
  min-width: 0;
}

.heading {
  margin: 0;
  color: var(--ant-text-2);
  font-size: var(--ant-fs-meta);
  font-weight: var(--ant-fw-bold);
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

.lines {
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 0;
  list-style: none;
}

.line {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-1) var(--ant-space-3);
  align-items: baseline;
  justify-content: space-between;
  padding: var(--ant-space-3) 0;
  border-bottom: 1px solid var(--ant-border);
}

.line:last-child {
  border-bottom: 0;
}

.line > .quiet,
.line > .warning,
.line > :deep(.n-alert) {
  flex-basis: 100%;
}

.cols {
  display: inline-flex;
  flex-wrap: wrap;
  gap: var(--ant-space-1) var(--ant-space-3);
  align-items: baseline;
  color: var(--ant-text-2);
}

.name,
.link {
  padding: 0;
  border: 0;
  background: none;
  color: var(--ant-text);
  font: inherit;
  text-align: start;
  cursor: pointer;
}

.name {
  font-weight: var(--ant-fw-bold);
}

.link {
  color: var(--ant-accent);
}

.name:hover,
.link:hover {
  color: var(--ant-accent);
  text-decoration: underline;
}

.count[data-tone='warn'] {
  color: var(--ant-status-attention-text);
}

.state {
  color: var(--ant-text-2);
}

.state[data-tone='warning'] {
  color: var(--ant-status-attention-text);
}

.state[data-tone='fault'] {
  color: var(--ant-status-danger-text);
}

.warning {
  margin: 0;
  color: var(--ant-status-attention-text);
  font-size: var(--ant-fs-meta);
}
</style>
