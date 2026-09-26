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
import { ActionButton, EmptyState, SectionPanel } from '@/shared/ui'
import { minutesOf } from '../model/norms'
import { queueFacts, type StationPost, type StationStep } from '../model/station'
import { anomalyText, factText } from '../model/texts'
import { postTone, shiftSummary, type NowAction } from '../model/now'
import type { DrillRef } from '@/shared/model/drill'
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
    /** Смена сеанса — в строке состояния. */
    shiftTitle?: string | null
    /** «Требует действий»: задачи и несостыковки (model/now). */
    actions?: readonly NowAction[]
  }>(),
  { stepsError: undefined, postsError: undefined, registry: null, density: 'comfortable', shiftTitle: null, actions: () => [] },
)
const emit = defineEmits<{
  item: [itemId: string]
  node: [stepKey: string]
  workplace: [workplaceId: string]
  person: [personId: string]
  assign: [workplaceId: string, station: string]
  open: [ref: DrillRef]
}>()
const { t, te, n } = useI18n()
const problemText = useProblemText()
const size = computed(() => naiveSizeOf(props.density))

const value = (v: MetricValue) => formatValue({ t, n: (x, f) => n(x, f) }, v)
const durationOver = (s: StationStep) => overNorm(minutesOf(s.meanDuration), s.step.norm)
const summary = computed(() => shiftSummary(props.actions, (props.posts ?? []).map((p) => p.post), props.steps))
/** Статистики смены по операции нет совсем — одной строкой, а не тремя «нет данных». */
const noStats = (s: StationStep) => !s.meanDuration && !s.reworkRuns && !s.unfinished
function actionText(a: NowAction): string {
  switch (a.kind) {
    case 'not_moved':
      return t('widgets.shopFloor.now.notMoved', { item: a.item ?? '—' })
    case 'presence':
      return t('widgets.shopFloor.now.presence', { person: a.person, post: a.post, presence: t(PRESENCE_TEXT[a.presence!]) })
    case 'unassigned':
      return t('widgets.shopFloor.now.unassigned', { post: a.post })
    default:
      return a.title ?? ''
  }
}
function act(a: NowAction): void {
  if (a.kind === 'unassigned' && a.workplaceId) emit('assign', a.workplaceId, a.post ?? '')
  else if (a.kind === 'presence' && a.workplaceId) emit('workplace', a.workplaceId)
  else if (a.ref) emit('open', a.ref)
}
const actLabel = (a: NowAction) => t(`widgets.shopFloor.now.act.${a.kind}`)
const factLine = (s: StationStep) => queueFacts(s, props.posts ?? [], props.registry).map((f) => factText(t, f, value))
</script>

<template>
  <div class="station" data-testid="station-view">
    <!-- Строка состояния: «как у меня дела?» одной строкой. -->
    <div class="status-line" data-testid="status-line">
      <strong class="ant-wrap" data-testid="workshop">{{ workshopName ?? t('widgets.shopFloor.station.noWorkshop') }}<template v-if="shiftTitle"> · {{ shiftTitle }}</template></strong>
      <span :data-tone="summary.actions ? 'warn' : undefined">{{ t('widgets.shopFloor.now.sumActions', { n: summary.actions }, summary.actions) }}</span>
      <span>{{ t('widgets.shopFloor.now.sumWorking', { n: summary.working }, summary.working) }}</span>
      <span v-if="summary.unassigned" data-tone="warn">{{ t('widgets.shopFloor.now.sumUnassigned', { n: summary.unassigned }, summary.unassigned) }}</span>
      <span>{{ t('widgets.shopFloor.now.sumInWork', { n: summary.inWork }, summary.inWork) }}</span>
    </div>

    <!-- Требует действий: задачи и несостыковки цифрового и физического. -->
    <section class="actions-block" :data-empty="!actions.length || undefined" data-testid="now-actions">
      <h3 v-if="actions.length" class="block-title">{{ t('widgets.shopFloor.now.title') }} · {{ actions.length }}</h3>
      <p v-else class="calm" data-testid="now-calm">✓ {{ t('widgets.shopFloor.now.calm') }}</p>
      <ul v-if="actions.length" class="action-list">
        <li v-for="a in actions" :key="a.key" class="action" :data-tone="a.tone" :data-kind="a.kind">
          <span class="dot" aria-hidden="true" />
          <span class="action-text ant-wrap">
            <span v-if="a.kind === 'not_moved' || a.kind === 'presence'" class="kind">{{ t('widgets.shopFloor.now.mismatch') }}</span>
            {{ actionText(a) }}
          </span>
          <ActionButton text type="primary" :size="size" :label="actLabel(a)" :data-testid="`act-${a.kind}`" @click="act(a)" />
        </li>
      </ul>
    </section>

    <SectionPanel :title="t('widgets.shopFloor.station.posts')" variant="plain" data-testid="posts">
      <NAlert v-if="postsError && !posts" type="error" :bordered="false">{{ problemText(postsError) }}</NAlert>
      <EmptyState v-else-if="posts && !posts.length" compact :title="t('empty.noRecords')" />
      <!-- Посты карточками: состояние поста целиком за один взгляд. -->
      <div v-else-if="posts" class="post-cards">
        <article v-for="p in posts" :key="p.post.workplace_id" class="post-card" :data-tone="postTone(p.post)" :data-workplace="p.post.workplace_id" :data-presence="p.post.presence">
          <header class="post-head">
            <span class="dot" aria-hidden="true" />
            <ActionButton text type="primary" :size="size" :label="p.post.station" data-testid="open-post" @click="emit('workplace', p.post.workplace_id)" />
          </header>
          <p class="post-line">
            <ActionButton v-if="p.post.assigned" text :size="size" :label="p.post.assigned.display" data-testid="open-person" @click="emit('person', p.post.assigned.person_id)" />
            <span v-else class="muted">{{ t('liveMap.posts.notAssigned') }}</span>
            <NTag v-if="p.post.assigned" size="small" :bordered="false" :type="presenceTagType(p.post.presence)"><span class="ant-wrap">{{ t(PRESENCE_TEXT[p.post.presence]) }}</span></NTag>
          </p>
          <p v-if="p.post.current_item" class="post-line">
            <span class="muted">{{ t('liveMap.posts.currentItem') }}:</span>
            <ActionButton text type="primary" :size="size" :label="p.post.current_item.label" :hint="t('common.actions.openPassport')" data-testid="current-item" @click="emit('item', p.post.current_item.item_id)" />
          </p>
          <div v-for="e in p.equipment" :key="e.equipment_id" class="post-eq" :data-equipment="e.equipment_id">
            <p class="post-line">
              <span class="ant-wrap">{{ e.title }}</span>
              <NTag size="small" :bordered="false" :type="e.condition === 'fault' ? 'error' : e.condition === 'warning' ? 'warning' : 'default'">{{ t(`widgets.shopFloor.execution.${e.execution}`) }}</NTag>
              <span v-if="toolLifeOf(e)" class="muted">{{ t('timeline.equipment.toolLife', toolLifeOf(e)!) }}</span>
            </p>
            <p v-for="w in e.warnings" :key="`${w.kind}-${w.since}`" class="warning ant-clamp-2" :title="w.text" data-testid="equipment-warning">{{ w.text }}</p>
            <PostRun v-if="e.current_run_id" :run-id="e.current_run_id" :steps="stepIndex" :at="at" @item="(id) => emit('item', id)" />
          </div>
          <footer class="post-foot">
            <ActionButton v-if="!p.post.assigned || p.post.presence === 'not_assigned'" size="small" type="primary" secondary :label="t('widgets.shopFloor.now.act.unassigned')" data-testid="assign-post" @click="emit('assign', p.post.workplace_id, p.post.station)" />
            <ActionButton size="small" quaternary :label="t('widgets.shopFloor.now.openPost')" @click="emit('workplace', p.post.workplace_id)" />
          </footer>
        </article>
      </div>
    </SectionPanel>

    <SectionPanel :title="t('widgets.shopFloor.station.steps')" variant="plain" data-testid="steps">
      <NAlert v-if="stepsError && !steps" type="error" :bordered="false">{{ problemText(stepsError) }}</NAlert>
      <EmptyState v-else-if="steps && !steps.length" compact :title="t('widgets.shopFloor.station.noSteps')" />
      <div v-else-if="steps" class="list">
        <SectionPanel v-for="s in steps" :key="s.step.stepKey" variant="subtle" :data-step="s.step.stepKey" :data-growing="s.growing || undefined">
          <div class="row">
            <ActionButton text type="primary" :size="size" :label="s.step.name" :hint="t('widgets.operation.open')" data-testid="open-operation" @click="emit('node', s.step.stepKey)" />
            <span v-if="s.step.operationCode" class="ant-muted">{{ t('widgets.shopFloor.station.operationCode', { code: s.step.operationCode }) }}</span>
            <NTag v-if="s.step.specialProcess" size="small" :bordered="false" type="info">{{ t('widgets.shopFloor.station.specialProcess') }}</NTag>
          </div>
          <div class="row" data-testid="counters">
            <!-- Счётчики — кнопки: открывают окно операции с изделиями (UI-42). -->
            <button type="button" class="chip" :data-tone="s.growing ? 'warn' : undefined" data-testid="queue" @click="emit('node', s.step.stepKey)">
              {{ t('liveMap.counters.inQueue') }}: {{ s.counters ? s.counters.queue : '—' }}
            </button>
            <button type="button" class="chip" data-testid="in-work" @click="emit('node', s.step.stepKey)">{{ t('liveMap.counters.inWork') }}: {{ s.counters ? s.counters.in_progress : '—' }}</button>
            <NTag v-if="s.counters?.nonconformities" size="small" :bordered="false" type="error">
              {{ t('plural.nonconformities', { n: s.counters.nonconformities }, s.counters.nonconformities) }}
            </NTag>
            <NTag v-if="s.bottleneck" size="small" :bordered="false" type="warning" data-testid="bottleneck">
              {{ t('liveMap.bottleneck') }}<template v-if="s.bottleneck.wait"> · {{ s.bottleneck.wait }}</template>
            </NTag>
            <NTag v-for="a in s.anomalies" :key="a.kind" size="small" :bordered="false" type="warning" :data-anomaly="a.kind">
              <span class="ant-ellipsis" :title="anomalyText(t, te, a)">{{ anomalyText(t, te, a) }}</span>
            </NTag>
            <NTag v-if="s.dataGap" size="small" :bordered="false" data-testid="data-gap">{{ t('widgets.shopFloor.station.dataGap') }}</NTag>
          </div>
          <p v-if="noStats(s)" class="ant-muted ant-wrap" data-testid="no-stats">{{ t('widgets.shopFloor.now.noStats') }} · {{ normText(t, s.step.norm) }}</p>
          <div v-else class="facts">
            <div class="row" data-testid="duration">
              <span class="ant-muted">{{ t('widgets.shopFloor.station.meanDuration') }}:</span>
              <MetricNumber v-if="s.meanDuration" :value="s.meanDuration" />
              <span v-else class="ant-muted">{{ t('empty.noDataUnknown') }}</span>
              <span class="ant-muted">· {{ normText(t, s.step.norm) }}</span>
              <NTag v-if="durationOver(s)" size="small" :bordered="false" type="error">{{ t('widgets.shopFloor.station.overNorm') }}</NTag>
            </div>
            <div class="row" data-testid="reworks">
              <span class="ant-muted">{{ t('widgets.shopFloor.station.reworkRuns') }}:</span>
              <MetricNumber v-if="s.reworkRuns" :value="s.reworkRuns" :show-origin="false" />
              <span v-else class="ant-muted">{{ t('empty.noDataUnknown') }}</span>
              <span v-if="s.step.reworkLimit !== null" class="ant-muted">
                · {{ t('widgets.shopFloor.station.reworkLimit', { limit: s.step.reworkLimit, scope: t(`widgets.shopFloor.station.reworkScope.${s.step.reworkLimitScope ?? 'item'}`) }) }}
              </span>
            </div>
            <div class="row" data-testid="unfinished">
              <span class="ant-muted">{{ t('analytics.metrics.unfinishedOperations.title') }}:</span>
              <MetricNumber v-if="s.unfinished" :value="s.unfinished" :show-origin="false" />
              <span v-else class="ant-muted">{{ t('empty.noDataUnknown') }}</span>
            </div>
          </div>
          <NAlert v-if="s.growing" type="warning" :bordered="false" :title="t('widgets.shopFloor.why.title', { step: s.step.name })" data-testid="why">
            <p class="ant-muted ant-wrap">{{ t('widgets.shopFloor.why.factsNotCause') }}</p>
            <p v-for="(line, i) in factLine(s)" :key="i" class="ant-wrap" data-testid="why-fact">{{ line }}</p>
            <p v-if="!factLine(s).length" class="ant-wrap">{{ t('widgets.shopFloor.why.none') }}</p>
          </NAlert>
        </SectionPanel>
      </div>
    </SectionPanel>
  </div>
</template>

<style scoped>
.station,
.list,
.facts,
.cell {
  display: flex;
  flex-direction: column;
  gap: var(--ant-gap);
  min-width: 0;
}

.facts,
.cell {
  gap: var(--ant-space-1);
}

.row {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-1) var(--ant-space-2);
  align-items: center;
  min-width: 0;
}

.chip {
  padding: 2px var(--ant-space-2);
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-sm);
  background: var(--ant-surface-subtle);
  color: var(--ant-text);
  font: inherit;
  font-size: var(--ant-fs-meta);
  cursor: pointer;
}

.chip:hover {
  border-color: var(--ant-accent);
  color: var(--ant-accent);
}

.chip[data-tone='warn'] {
  border-color: var(--ant-status-attention);
  background: var(--ant-status-attention-soft);
}

.status-line {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-1) var(--ant-space-4);
  align-items: baseline;
  color: var(--ant-text-2);
}

.status-line [data-tone='warn'] {
  color: var(--ant-status-attention-text);
  font-weight: var(--ant-fw-bold);
}

.actions-block {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
  padding: var(--ant-space-3) var(--ant-space-4);
  border: 1px solid var(--ant-status-attention);
  border-left-width: 4px;
  border-radius: var(--ant-radius-lg);
  background: var(--ant-status-attention-soft);
}

.actions-block[data-empty] {
  padding: var(--ant-space-2) var(--ant-space-4);
  border-color: var(--ant-status-success);
  background: var(--ant-status-success-soft);
}

.block-title,
.calm {
  margin: 0;
  font-size: var(--ant-fs-title);
}

.calm {
  color: var(--ant-status-success-text);
  font-size: var(--ant-fs-body);
  font-weight: var(--ant-fw-bold);
}

.action-list {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  margin: 0;
  padding: 0;
  list-style: none;
}

.action {
  display: flex;
  gap: var(--ant-space-2);
  align-items: center;
  min-width: 0;
  padding: var(--ant-space-1) var(--ant-space-2);
  border-radius: var(--ant-radius-md);
  background: var(--ant-surface);
}

.action-text {
  flex: 1 1 auto;
  min-width: 0;
}

.kind {
  margin-right: var(--ant-space-1);
  color: var(--ant-status-danger-text);
  font-size: var(--ant-fs-meta);
  font-weight: var(--ant-fw-bold);
  text-transform: uppercase;
}

.dot {
  flex: none;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--ant-status-neutral);
}

.action[data-tone='danger'] .dot {
  background: var(--ant-status-danger);
}

.action[data-tone='warn'] .dot {
  background: var(--ant-status-attention);
}

.post-cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: var(--ant-space-3);
}

.post-card {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  min-width: 0;
  padding: var(--ant-space-3);
  border: 1px solid var(--ant-border);
  border-top: 4px solid var(--ant-border-strong);
  border-radius: var(--ant-radius-lg);
  background: var(--ant-surface);
}

.post-card[data-tone='ok'] {
  border-top-color: var(--ant-status-success);
}

.post-card[data-tone='warn'] {
  border-top-color: var(--ant-status-attention);
}

.post-card[data-tone='ok'] .post-head .dot {
  background: var(--ant-status-success);
}

.post-card[data-tone='warn'] .post-head .dot {
  background: var(--ant-status-attention);
}

.post-head,
.post-line,
.post-foot {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-1) var(--ant-space-2);
  align-items: center;
  min-width: 0;
  margin: 0;
}

.post-foot {
  margin-top: auto;
  padding-top: var(--ant-space-2);
}

.post-eq {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.muted {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.warning {
  color: var(--ant-status-attention-text);
  font-size: var(--ant-fs-meta);
}
</style>
