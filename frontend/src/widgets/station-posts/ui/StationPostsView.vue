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
import { ReceiveAction } from '@/features/item-receive'
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
    <p class="ant-muted ant-wrap" data-testid="workshop">
      {{ workshopName ? t('widgets.shopFloor.station.workshop', { name: workshopName }) : t('widgets.shopFloor.station.noWorkshop') }}
    </p>

    <!-- Главное наверху: что пришло и ждёт, что в работе на постах. -->
    <SectionPanel :title="t('stationDesk.incoming')" variant="plain" data-testid="incoming">
      <p v-if="!incoming.length && !entryItems.length" class="ant-muted" data-testid="nothing-incoming">{{ t('stationDesk.nothingIncoming') }}</p>
      <ul v-else class="incoming">
        <li v-for="r in incoming" :key="r.taskId" class="incoming-row" :data-incoming="r.itemId">
          <strong class="ant-wrap">{{ r.title }}</strong>
          <ReceiveAction v-if="canAct" :item-id="r.itemId" :basis-seq="basisSeq" />
        </li>
        <li v-if="entryItems.length" class="incoming-row" data-testid="entry-items">
          <span class="ant-muted">{{ t('stationDesk.atEntry') }}:</span>
          <div class="row">
            <ActionButton
              v-for="i in entryItems"
              :key="i.item_id"
              text
              type="primary"
              :size="size"
              :label="i.label"
              :hint="t('common.actions.openPassport')"
              @click="emit('item', i.item_id)"
            />
          </div>
        </li>
      </ul>
    </SectionPanel>

    <SectionPanel :title="t('stationDesk.atPosts')" variant="plain" data-testid="posts">
      <NAlert v-if="postsError && !posts" type="error" :bordered="false">{{ problemText(postsError) }}</NAlert>
      <EmptyState v-else-if="posts && !posts.length" compact :title="t('empty.noRecords')" />
      <DataTable v-else-if="posts" :caption="t('widgets.shopFloor.station.posts')">
        <thead>
          <tr>
            <th>{{ t('liveMap.posts.station') }}</th>
            <th>{{ t('liveMap.posts.assigned') }}</th>
            <th>{{ t('liveMap.posts.currentItem') }}</th>
            <th>{{ t('common.words.equipment') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="p in posts" :key="p.post.workplace_id" :data-workplace="p.post.workplace_id" :data-presence="p.post.presence">
            <th scope="row">
              <ActionButton text type="primary" :size="size" :label="p.post.station" data-testid="open-post" @click="emit('workplace', p.post.workplace_id)" />
            </th>
            <td class="assigned">
              <div class="cell">
                <ActionButton v-if="p.post.assigned" text type="primary" :size="size" :label="p.post.assigned.display" data-testid="open-person" @click="emit('person', p.post.assigned.person_id)" />
                <NTag v-if="p.post.presence !== 'unknown'" size="small" :bordered="false" :type="presenceTagType(p.post.presence)">
                  <span class="ant-wrap">{{ t(PRESENCE_TEXT[p.post.presence]) }}</span>
                </NTag>
              </div>
            </td>
            <td>
              <ActionButton
                v-if="p.post.current_item"
                text
                type="primary"
                :size="size"
                :label="p.post.current_item.label"
                :hint="t('common.actions.openPassport')"
                data-testid="current-item"
                @click="emit('item', p.post.current_item.item_id)"
              />
              <span v-else class="ant-muted">—</span>
            </td>
            <td>
              <span v-if="!p.equipment.length" class="ant-muted">—</span>
              <div v-for="e in p.equipment" :key="e.equipment_id" class="cell" :data-equipment="e.equipment_id">
                <div class="row">
                  <span class="ant-wrap">{{ e.title }}</span>
                  <NTag v-if="e.execution !== 'unknown'" size="small" :bordered="false" :type="e.condition === 'fault' ? 'error' : e.condition === 'warning' ? 'warning' : 'default'">
                    {{ t(`widgets.shopFloor.execution.${e.execution}`) }}
                  </NTag>
                  <span v-if="toolLifeOf(e)" class="ant-muted">{{ t('timeline.equipment.toolLife', toolLifeOf(e)!) }}</span>
                </div>
                <p v-for="w in e.warnings" :key="`${w.kind}-${w.since}`" class="warning ant-clamp-2" :title="w.text" data-testid="equipment-warning">{{ w.text }}</p>
                <PostRun v-if="e.current_run_id" :run-id="e.current_run_id" :steps="stepIndex" :at="at" :current-item="p.post.current_item ?? null" @item="(id) => emit('item', id)" />
              </div>
            </td>
          </tr>
        </tbody>
      </DataTable>
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
            <span v-if="s.dataGap" class="ant-muted" data-testid="data-gap">{{ t('stationDesk.noSource') }}</span>
          </div>
          <div v-if="hasFacts(s)" class="facts">
            <div v-if="s.meanDuration" class="row" data-testid="duration">
              <span class="ant-muted">{{ t('widgets.shopFloor.station.meanDuration') }}:</span>
              <MetricNumber :value="s.meanDuration" />
              <span class="ant-muted">· {{ normText(t, s.step.norm) }}</span>
              <NTag v-if="durationOver(s)" size="small" :bordered="false" type="error">{{ t('widgets.shopFloor.station.overNorm') }}</NTag>
            </div>
            <div v-if="s.reworkRuns" class="row" data-testid="reworks">
              <span class="ant-muted">{{ t('widgets.shopFloor.station.reworkRuns') }}:</span>
              <MetricNumber :value="s.reworkRuns" :show-origin="false" />
              <span v-if="s.step.reworkLimit !== null" class="ant-muted">
                · {{ t('widgets.shopFloor.station.reworkLimit', { limit: s.step.reworkLimit, scope: t(`widgets.shopFloor.station.reworkScope.${s.step.reworkLimitScope ?? 'item'}`) }) }}
              </span>
            </div>
            <div v-if="s.unfinished" class="row" data-testid="unfinished">
              <span class="ant-muted">{{ t('analytics.metrics.unfinishedOperations.title') }}:</span>
              <MetricNumber :value="s.unfinished" :show-origin="false" />
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
.assigned {
  min-width: 12em;
}

.assigned :deep(.ant-wrap) {
  overflow-wrap: normal;
  word-break: normal;
  white-space: nowrap;
}

.station,
.list,
.incoming,
.incoming-row,
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

.warning {
  color: var(--ant-status-attention-text);
  font-size: var(--ant-fs-meta);
}
.incoming {
  margin: 0;
  padding: 0;
  list-style: none;
}

.incoming-row {
  padding: var(--ant-space-2) 0;
  border-bottom: 1px solid var(--ant-border);
}

.incoming-row:last-child {
  border-bottom: 0;
}
</style>
