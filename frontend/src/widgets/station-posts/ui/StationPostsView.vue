<script setup lang="ts">
/**
 * Участок — представление (PRD §3a «Мастер участка — Участок», FR-81, UJ-7):
 * операции цеха — очередь, в работе, длительность против нормы, повторные
 * выполнения против лимита доработок, незавершённые операции, аномалии; рядом с
 * растущей очередью — факты участка («почему растёт очередь»); посты —
 * назначенный и присутствие, текущая деталь, оборудование и текущее выполнение.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NAlert, NButton, NDivider, NEllipsis, NFlex, NList, NListItem, NTag, NText } from 'naive-ui'
import type { ProcessStep } from '@/entities/live-map'
import { formatValue, MetricNumber, type MetricValue } from '@/entities/metric'
import { normText, overNorm } from '@/entities/live-map'
import type { RefEquipment } from '@/entities/equipment'
import { toolLifeOf } from '@/entities/equipment'
import type { Density } from '@/shared/config/widget'
import { naiveSizeOf } from '@/shared/config/widget'
import { useProblemText } from '@/shared/i18n/problem'
import { minutesOf } from '../model/norms'
import { queueFacts, type StationPost, type StationStep } from '../model/station'
import { PRESENCE_TEXT, presenceTagType } from '@/entities/workplace'
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
  }>(),
  { stepsError: undefined, postsError: undefined, registry: null, density: 'comfortable' },
)
const emit = defineEmits<{ item: [itemId: string]; node: [stepKey: string] }>()
const { t, te, n } = useI18n()
const problemText = useProblemText()
const size = computed(() => naiveSizeOf(props.density))
const tagSize = computed(() => (props.density === 'large' ? 'medium' : 'small'))

const value = (v: MetricValue) => formatValue({ t, n: (x, f) => n(x, f) }, v)
const durationOver = (s: StationStep) => overNorm(minutesOf(s.meanDuration), s.step.norm)
const facts = (s: StationStep) => queueFacts(s, props.posts ?? [], props.registry)
const factLine = (s: StationStep) => facts(s).map((f) => factText(t, f, value))

</script>

<template>
  <NFlex vertical :size="12" data-testid="station-view">
    <NText depth="3" data-testid="workshop">
      {{ workshopName ? t('widgets.shopFloor.station.workshop', { name: workshopName }) : t('widgets.shopFloor.station.noWorkshop') }}
    </NText>

    <section data-testid="steps">
      <NDivider title-placement="left">{{ t('widgets.shopFloor.station.steps') }}</NDivider>
      <NText v-if="stepsError && !steps" type="error">{{ problemText(stepsError) }}</NText>
      <NText v-else-if="steps && !steps.length" depth="3">{{ t('widgets.shopFloor.station.noSteps') }}</NText>
      <NList v-else-if="steps" :show-divider="true">
        <NListItem v-for="s in steps" :key="s.step.stepKey" :data-step="s.step.stepKey" :data-growing="s.growing || undefined">
          <NFlex vertical :size="6">
            <NFlex :size="8" align="center" :wrap="true">
              <NButton text :size="size" @click="emit('node', s.step.stepKey)">
                <NEllipsis :tooltip="{ width: 360 }">{{ s.step.name }}</NEllipsis>
              </NButton>
              <NText v-if="s.step.operationCode" depth="3">{{ t('widgets.shopFloor.station.operationCode', { code: s.step.operationCode }) }}</NText>
              <NTag v-if="s.step.specialProcess" :size="tagSize" :bordered="false" type="info">{{ t('widgets.shopFloor.station.specialProcess') }}</NTag>
            </NFlex>
            <NFlex :size="8" :wrap="true" data-testid="counters">
              <NTag :size="tagSize" :bordered="false" :type="s.growing ? 'warning' : 'default'" data-testid="queue">
                {{ t('liveMap.counters.inQueue') }}: {{ s.counters ? s.counters.queue : '—' }}
              </NTag>
              <NTag :size="tagSize" :bordered="false">{{ t('liveMap.counters.inWork') }}: {{ s.counters ? s.counters.in_progress : '—' }}</NTag>
              <NTag v-if="s.counters?.nonconformities" :size="tagSize" :bordered="false" type="error">
                {{ t('plural.nonconformities', { n: s.counters.nonconformities }, s.counters.nonconformities) }}
              </NTag>
              <NTag v-if="s.bottleneck" :size="tagSize" :bordered="false" type="warning" data-testid="bottleneck">
                {{ t('liveMap.bottleneck') }}<template v-if="s.bottleneck.wait"> · {{ s.bottleneck.wait }}</template>
              </NTag>
              <NTag v-for="a in s.anomalies" :key="a.kind" :size="tagSize" :bordered="false" type="warning" :data-anomaly="a.kind">
                <NEllipsis :tooltip="{ width: 360 }">{{ anomalyText(t, te, a) }}</NEllipsis>
              </NTag>
              <NTag v-if="s.dataGap" :size="tagSize" :bordered="false" data-testid="data-gap">{{ t('widgets.shopFloor.station.dataGap') }}</NTag>
            </NFlex>
            <NFlex vertical :size="2">
              <NFlex :size="6" align="baseline" :wrap="true" data-testid="duration">
                <NText depth="3">{{ t('widgets.shopFloor.station.meanDuration') }}:</NText>
                <MetricNumber v-if="s.meanDuration" :value="s.meanDuration" />
                <NText v-else depth="3">{{ t('empty.noDataUnknown') }}</NText>
                <NText depth="3">· {{ normText(t, s.step.norm) }}</NText>
                <NTag v-if="durationOver(s)" :size="tagSize" :bordered="false" type="error">{{ t('widgets.shopFloor.station.overNorm') }}</NTag>
              </NFlex>
              <NFlex :size="6" align="baseline" :wrap="true" data-testid="reworks">
                <NText depth="3">{{ t('widgets.shopFloor.station.reworkRuns') }}:</NText>
                <MetricNumber v-if="s.reworkRuns" :value="s.reworkRuns" :show-origin="false" />
                <NText v-else depth="3">{{ t('empty.noDataUnknown') }}</NText>
                <NText v-if="s.step.reworkLimit !== null" depth="3">
                  · {{ t('widgets.shopFloor.station.reworkLimit', { limit: s.step.reworkLimit, scope: t(`widgets.shopFloor.station.reworkScope.${s.step.reworkLimitScope ?? 'item'}`) }) }}
                </NText>
              </NFlex>
              <NFlex :size="6" align="baseline" :wrap="true" data-testid="unfinished">
                <NText depth="3">{{ t('analytics.metrics.unfinishedOperations.title') }}:</NText>
                <MetricNumber v-if="s.unfinished" :value="s.unfinished" :show-origin="false" />
                <NText v-else depth="3">{{ t('empty.noDataUnknown') }}</NText>
              </NFlex>
            </NFlex>
            <NAlert v-if="s.growing" type="warning" :bordered="false" :title="t('widgets.shopFloor.why.title', { step: s.step.name })" data-testid="why">
              <NFlex vertical :size="2">
                <NText depth="3">{{ t('widgets.shopFloor.why.factsNotCause') }}</NText>
                <NText v-for="(line, i) in factLine(s)" :key="i" data-testid="why-fact">{{ line }}</NText>
                <NText v-if="!factLine(s).length">{{ t('widgets.shopFloor.why.none') }}</NText>
              </NFlex>
            </NAlert>
          </NFlex>
        </NListItem>
      </NList>
    </section>

    <section data-testid="posts">
      <NDivider title-placement="left">{{ t('widgets.shopFloor.station.posts') }}</NDivider>
      <NText v-if="postsError && !posts" type="error">{{ problemText(postsError) }}</NText>
      <NText v-else-if="posts && !posts.length" depth="3">{{ t('empty.noRecords') }}</NText>
      <NList v-else-if="posts" :show-divider="true">
        <NListItem v-for="p in posts" :key="p.post.workplace_id" :data-workplace="p.post.workplace_id" :data-presence="p.post.presence">
          <NFlex vertical :size="4">
            <NFlex :size="8" align="center" :wrap="true">
              <NText strong><NEllipsis :tooltip="{ width: 360 }">{{ p.post.station }}</NEllipsis></NText>
              <NText>{{ p.post.assigned?.display ?? t('liveMap.posts.notAssigned') }}</NText>
              <NTag :size="tagSize" :bordered="false" :type="presenceTagType(p.post.presence)">{{ t(PRESENCE_TEXT[p.post.presence]) }}</NTag>
            </NFlex>
            <NFlex :size="6" align="center" :wrap="true">
              <NText depth="3">{{ t('liveMap.posts.currentItem') }}:</NText>
              <NButton v-if="p.post.current_item" text type="primary" :size="size" data-testid="current-item" @click="emit('item', p.post.current_item.item_id)">
                {{ p.post.current_item.label }}
              </NButton>
              <NText v-else depth="3">—</NText>
            </NFlex>
            <NFlex v-for="e in p.equipment" :key="e.equipment_id" vertical :size="2" :data-equipment="e.equipment_id">
              <NFlex :size="6" align="center" :wrap="true">
                <NText>{{ e.title }}</NText>
                <NTag :size="tagSize" :bordered="false" :type="e.condition === 'fault' ? 'error' : e.condition === 'warning' ? 'warning' : 'default'">
                  {{ t(`widgets.shopFloor.execution.${e.execution}`) }}
                </NTag>
                <NText v-if="toolLifeOf(e)" depth="3">{{ t('timeline.equipment.toolLife', toolLifeOf(e)!) }}</NText>
              </NFlex>
              <NText v-for="w in e.warnings" :key="`${w.kind}-${w.since}`" type="warning" data-testid="equipment-warning">
                <NEllipsis :line-clamp="2" :tooltip="{ width: 360 }">{{ w.text }}</NEllipsis>
              </NText>
              <PostRun v-if="e.current_run_id" :run-id="e.current_run_id" :steps="stepIndex" :at="at" @item="(id) => emit('item', id)" />
            </NFlex>
          </NFlex>
        </NListItem>
      </NList>
    </section>
  </NFlex>
</template>
