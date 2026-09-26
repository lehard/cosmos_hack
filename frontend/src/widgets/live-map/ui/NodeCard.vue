<script setup lang="ts">
/**
 * Карточка узла живой карты (FR-154, FR-7): описание шага из `documentation`
 * элемента BPMN версии изделия, счётчики узла (FR-2) и переход к изделиям и
 * несоответствиям узла; по изделию — паспорт.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NEmpty } from 'naive-ui'
import type { MapItem, NodeCounters } from '@/entities/live-map'
import type { StepNode } from '../model/bpmn'
import { dotLook, sortForDots } from '../model/overlays'
import { ActionButton } from '@/shared/ui'

const props = defineProps<{
  node: StepNode
  counters?: NodeCounters
  items: readonly MapItem[]
  incidentMode: boolean
}>()
const emit = defineEmits<{
  close: []
  'open-item': [itemId: string]
  /** Изделия и несоответствия узла (FR-7). */
  'open-node': [stepKey: string]
}>()
const { t } = useI18n()

const rows = computed(() => {
  const c = props.counters
  return [
    { kind: 'queue', label: t('liveMap.counters.inQueue'), value: c?.queue },
    { kind: 'in_progress', label: t('liveMap.counters.inWork'), value: c?.in_progress },
    { kind: 'passed', label: t('liveMap.counters.passed'), value: c?.passed },
    { kind: 'defects', label: t('liveMap.counters.defects'), value: c?.defects },
  ]
})

const list = computed(() =>
  sortForDots(props.items, props.incidentMode).map((item) => {
    const look = dotLook(item, props.incidentMode)
    return { item, look, status: look.statusKey ? t(look.statusKey) : '' }
  }),
)
</script>

<template>
  <aside class="node-card" :data-step="node.stepKey">
    <header class="head">
      <div>
        <h3 class="name">{{ node.name || node.stepKey }}</h3>
        <div class="meta">
          <span v-if="node.laneName">{{ t('liveMap.workshops.lane') }}: {{ node.laneName }}</span>
          <code class="key">{{ node.stepKey }}</code>
        </div>
      </div>
      <ActionButton overflow="wrap" size="tiny" quaternary @click="emit('close')" :label="t('common.actions.close')" />
    </header>

    <p v-if="node.documentation" class="doc" data-testid="node-doc">{{ node.documentation }}</p>

    <dl class="counters">
      <template v-for="r in rows" :key="r.kind">
        <dt>{{ r.label }}</dt>
        <dd :data-counter="r.kind">{{ r.value ?? t('empty.noDataUnknown') }}</dd>
      </template>
      <template v-if="counters?.nonconformities !== undefined">
        <dt>{{ t('common.words.nonconformity') }}</dt>
        <dd data-counter="nonconformities">{{ t('plural.nonconformities', { n: counters.nonconformities }, counters.nonconformities) }}</dd>
      </template>
    </dl>

    <ActionButton overflow="wrap" size="small" secondary block data-action="open-node" @click="emit('open-node', node.stepKey)" :label="t('liveMap.drillDown.nodeItems')" />

    <h4 class="sub">{{ t('common.words.items') }}</h4>
    <NEmpty v-if="!list.length" size="small" :description="t('empty.noRecords')" />
    <ul v-else class="items">
      <li v-for="r in list" :key="r.item.item_id">
        <button type="button" class="item" :data-item="r.item.item_id" :title="t('common.actions.openPassport')" @click="emit('open-item', r.item.item_id)">
          <span class="dot" :class="{ dimmed: r.look.dimmed }" :style="{ background: r.look.color }" aria-hidden="true" />
          <span class="label">{{ r.item.label }}</span>
          <span class="status">{{ r.status }}</span>
        </button>
      </li>
    </ul>
  </aside>
</template>

<style scoped>
.node-card {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 12px;
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-md);
  background: var(--ant-surface);
}

.head {
  display: flex;
  gap: 8px;
  align-items: flex-start;
  justify-content: space-between;
}

.name {
  margin: 0;
  font-size: var(--ant-fs-title);
}

.meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.key {
  font-family: var(--ant-font-mono);
}

.doc {
  margin: 0;
  white-space: pre-line;
  font-size: var(--ant-fs-body);
  line-height: 1.45;
}

.counters {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 4px 12px;
  margin: 0;
  font-size: var(--ant-fs-body);
}

.counters dd {
  margin: 0;
  font-weight: var(--ant-fw-bold);
  text-align: right;
}

.sub {
  margin: 4px 0 0;
  font-size: var(--ant-fs-body);
}

.items {
  display: flex;
  flex-direction: column;
  gap: 2px;
  max-height: 240px;
  margin: 0;
  padding: 0;
  overflow: auto;
  list-style: none;
}

.item {
  display: flex;
  gap: 8px;
  align-items: center;
  width: 100%;
  padding: 4px;
  border: 0;
  border-radius: var(--ant-radius-sm);
  background: transparent;
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.item:hover {
  background: var(--ant-n-100);
}

.dot {
  flex: none;
  width: 10px;
  height: 10px;
  border-radius: 50%;
}

.dot.dimmed {
  opacity: 0.35;
}

.label {
  font-family: var(--ant-font-mono);
}

.status {
  margin-left: auto;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}
</style>
