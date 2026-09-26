<script setup lang="ts">
/**
 * Окно операции участка (Д-70, UI-42; SITE_FOREMAN: «увидеть, что застряло»):
 * id — шаг процесса. Что на операции сейчас (очередь, в работе, прошло,
 * несоответствия; норма; аномалии и «ограничение линии»), что от мастера нужно
 * (очередь растёт — факты участка рядом, это факты, а не причина; иначе —
 * «в норме, действий не требуется»), изделия на операции по положению (в очереди,
 * в работе, прочие) — щелчок открывает изделие, посты цеха — щелчок открывает
 * окно поста. Годность мастер не решает — решения по изделию в окне изделия.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { NTag } from 'naive-ui'
import { useEquipmentRegistry, useEquipmentStates } from '@/entities/equipment'
import { SummaryTag } from '@/entities/item'
import { normText, parseProcessSteps, useLiveMap } from '@/entities/live-map'
import { formatValue, type MetricValue } from '@/entities/metric'
import { useItemsAtStep } from '@/entities/operation'
import { PRESENCE_TEXT, presenceTagType, usePosts } from '@/entities/workplace'
import { useDrillDown } from '@/features/drill-down'
import type { DrillRef } from '@/shared/model/drill'
import { ActionButton, RecordDrawer, StatusTag } from '@/shared/ui'
import { buildPosts, buildSteps, queueFacts } from '@/widgets/station-posts/model/station'
import { anomalyText, factText } from '@/widgets/station-posts/model/texts'

const props = defineProps<{ id: string; show: boolean }>()
const emit = defineEmits<{ close: [] }>()
const { t, te, n } = useI18n()
const route = useRoute()
const drill = useDrillDown()

const run = computed(() => (typeof route.query.run === 'string' && route.query.run ? route.query.run : undefined))
const mapQ = useLiveMap(computed(() => ({ period: 'shift' as const, ...(run.value ? { run_id: run.value } : {}) })))
const parsed = computed(() => parseProcessSteps(mapQ.data.value?.data.bpmn_xml))
const step = computed(() => parsed.value.byKey.get(props.id) ?? null)
const info = computed(() => {
  const map = mapQ.data.value?.data
  return map && step.value ? (buildSteps([step.value], map, null)[0] ?? null) : null
})

const postsQ = usePosts(computed(() => ({ ...(step.value?.workshop ? { workshop: step.value.workshop } : {}), ...(run.value ? { run_id: run.value } : {}) })))
const equipmentQ = useEquipmentStates()
const registryQ = useEquipmentRegistry()
const posts = computed(() => (postsQ.data.value?.data ? buildPosts(postsQ.data.value.data, equipmentQ.data.value?.data) : []))
const value = (v: MetricValue) => formatValue({ t, n: (x, f) => n(x, f) }, v)
const facts = computed(() => (info.value ? queueFacts(info.value, posts.value, registryQ.data.value?.data).map((f) => factText(t, f, value)) : []))

const itemsQ = useItemsAtStep(() => props.id)
const items = computed(() => itemsQ.data.value?.data ?? [])
const groups = computed(() => {
  const queue = items.value.filter((i) => i.status.position === 'in_queue')
  const work = items.value.filter((i) => i.status.position === 'in_progress')
  const other = items.value.filter((i) => i.status.position !== 'in_queue' && i.status.position !== 'in_progress')
  return [
    { key: 'queue', title: t('liveMap.counters.inQueue'), rows: queue },
    { key: 'work', title: t('liveMap.counters.inWork'), rows: work },
    { key: 'other', title: t('widgets.operation.otherItems'), rows: other },
  ].filter((g) => g.rows.length)
})

const subtitle = computed(() => {
  const s = step.value
  if (!s) return ''
  return [s.operationCode ? t('widgets.shopFloor.station.operationCode', { code: s.operationCode }) : null, s.specialProcess ? t('widgets.shopFloor.station.specialProcess') : null, normText(t, s.norm)]
    .filter(Boolean)
    .join(' · ')
})
const counters = computed(() => {
  const c = info.value?.counters
  return [
    { key: 'queue', label: t('liveMap.counters.inQueue'), value: c?.queue },
    { key: 'in_progress', label: t('liveMap.counters.inWork'), value: c?.in_progress },
    { key: 'passed', label: t('liveMap.counters.passed'), value: c?.passed },
    { key: 'nc', label: t('common.words.nonconformity'), value: c?.nonconformities },
  ]
})

/** Изделие и пост — в своём окне (Д-70): окно операции уступает место. */
function open(ref: DrillRef): void {
  drill.open(ref)
}
</script>

<template>
  <RecordDrawer
    :show="show"
    :kind-label="t('widgets.operation.kind')"
    :number="step?.name ?? id"
    :subtitle="subtitle"
    :loading="mapQ.isPending.value && !step"
    data-record="operation"
    @close="emit('close')"
  >
    <div class="operation" data-testid="operation-window">
      <p v-if="!step && !mapQ.isPending.value" class="muted ant-wrap">{{ t('widgets.operation.notFound') }}</p>

      <!-- Что на операции сейчас. -->
      <section v-if="step" class="block" data-zone="now">
        <h4>{{ t('widgets.operation.now') }}</h4>
        <div class="tiles" data-testid="counters">
          <div v-for="c in counters" :key="c.key" class="tile" :data-counter="c.key">
            <span class="tile-label ant-wrap">{{ c.label }}</span>
            <span class="tile-value">{{ c.value ?? '—' }}</span>
          </div>
        </div>
        <div v-if="info && (info.anomalies.length || info.bottleneck || info.dataGap)" class="row">
          <NTag v-if="info.bottleneck" size="small" :bordered="false" type="warning">{{ t('liveMap.bottleneck') }}<template v-if="info.bottleneck.wait"> · {{ info.bottleneck.wait }}</template></NTag>
          <NTag v-for="a in info.anomalies" :key="a.kind" size="small" :bordered="false" type="warning" :data-anomaly="a.kind">
            <span class="ant-wrap">{{ anomalyText(t, te, a) }}</span>
          </NTag>
          <NTag v-if="info.dataGap" size="small" :bordered="false">{{ t('widgets.shopFloor.station.dataGap') }}</NTag>
        </div>
      </section>

      <!-- Что от вас нужно. -->
      <section v-if="step" class="block task" :data-growing="info?.growing || undefined" data-zone="task" data-testid="what-to-do">
        <p class="kicker">{{ t('widgets.operation.whatToDo') }}</p>
        <template v-if="info?.growing">
          <p class="lead ant-wrap">{{ t('widgets.operation.growing') }}</p>
          <p class="muted ant-wrap">{{ t('widgets.shopFloor.why.factsNotCause') }}</p>
          <ul class="list" data-testid="facts">
            <li v-for="(line, i) in facts" :key="i" class="ant-wrap">{{ line }}</li>
            <li v-if="!facts.length" class="ant-wrap">{{ t('widgets.shopFloor.why.none') }}</li>
          </ul>
        </template>
        <p v-else-if="info?.anomalies.length" class="lead ant-wrap">{{ t('widgets.operation.anomaly') }}</p>
        <p v-else class="lead ant-wrap">{{ t('widgets.operation.normal') }}</p>
        <p class="muted ant-wrap">{{ t('widgets.operation.notQuality') }}</p>
      </section>

      <!-- Изделия на операции. -->
      <section v-if="step" class="block" data-zone="items" data-testid="op-items">
        <h4>{{ t('common.words.items') }}</h4>
        <p v-if="!groups.length" class="muted">{{ t('widgets.operation.noItems') }}</p>
        <div v-for="g in groups" :key="g.key" class="group" :data-group="g.key">
          <p class="group-title">{{ g.title }} · {{ g.rows.length }}</p>
          <ul class="items">
            <li v-for="i in g.rows" :key="i.item_id" class="item-row">
              <ActionButton text type="primary" size="small" :label="i.label" :hint="t('common.actions.openPassport')" @click="open({ entity: 'item', id: i.item_id })" />
              <StatusTag axis="position" :code="i.status.position" />
              <SummaryTag v-if="i.status.summary !== 'in_process'" :code="i.status.summary" />
            </li>
          </ul>
        </div>
      </section>

      <!-- Посты цеха: кто назначен, на месте ли; щелчок — окно поста. -->
      <section v-if="step && posts.length" class="block" data-zone="posts" data-testid="op-posts">
        <h4>{{ t('widgets.shopFloor.station.posts') }}</h4>
        <ul class="items">
          <li v-for="p in posts" :key="p.post.workplace_id" class="item-row">
            <ActionButton text type="primary" size="small" :label="p.post.station" @click="open({ entity: 'workplace', id: p.post.workplace_id })" />
            <span class="muted ant-wrap">{{ p.post.assigned?.display ?? t('liveMap.posts.notAssigned') }}</span>
            <NTag size="small" :bordered="false" :type="presenceTagType(p.post.presence)">{{ t(PRESENCE_TEXT[p.post.presence]) }}</NTag>
          </li>
        </ul>
      </section>
    </div>
  </RecordDrawer>
</template>

<style scoped>
.operation {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-6);
  min-width: 0;
  line-height: 1.5;
}

p,
h4 {
  margin: 0;
}

h4 {
  font-size: var(--ant-fs-title);
}

.block {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
  min-width: 0;
}

.tiles {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
  gap: var(--ant-space-2);
}

.tile {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: var(--ant-space-2) var(--ant-space-3);
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-md);
  background: var(--ant-surface-subtle);
}

.tile-label {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.tile-value {
  font-size: var(--ant-fs-title);
  font-weight: var(--ant-fw-bold);
  font-variant-numeric: tabular-nums;
}

.row {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-1) var(--ant-space-2);
}

.task {
  padding: var(--ant-space-3) var(--ant-space-4);
  border-left: 4px solid var(--ant-status-success);
  border-radius: 0 var(--ant-radius-md) var(--ant-radius-md) 0;
  background: var(--ant-surface-subtle);
}

.task[data-growing] {
  border-left-color: var(--ant-status-attention);
  background: var(--ant-status-attention-soft);
}

.kicker {
  color: var(--ant-text-2);
  font-size: var(--ant-fs-meta);
  font-weight: var(--ant-fw-bold);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.lead {
  font-weight: var(--ant-fw-bold);
}

.list {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  margin: 0;
  padding-left: var(--ant-space-5);
}

.group {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
}

.group-title {
  color: var(--ant-text-2);
  font-size: var(--ant-fs-meta);
  font-weight: var(--ant-fw-bold);
}

.items {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  margin: 0;
  padding: 0;
  list-style: none;
}

.item-row {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-1) var(--ant-space-2);
  align-items: center;
  min-width: 0;
}

.muted {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}
</style>
