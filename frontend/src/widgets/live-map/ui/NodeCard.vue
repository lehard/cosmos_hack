<script setup lang="ts">
/**
 * Содержимое окна узла живой карты (FR-154, FR-7; Д-70 — правое окно записи):
 * описание шага из `documentation` элемента BPMN версии изделия, счётчики узла
 * (FR-2) и изделия в узле; по изделию — паспорт. Заголовок (шаг, цех, код) и
 * кнопка «Изделия и несоответствия узла» — у окна (LiveMapView).
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NEmpty } from 'naive-ui'
import type { MapItem, NodeCounters } from '@/entities/live-map'
import type { StepNode } from '../model/bpmn'
import { dotLook, sortForDots } from '../model/overlays'

const props = defineProps<{
  node: StepNode
  counters?: NodeCounters
  items: readonly MapItem[]
  incidentMode: boolean
}>()
const emit = defineEmits<{
  'open-item': [itemId: string]
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
  <div class="node-card" :data-step="node.stepKey">
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
  </div>
</template>

<style scoped>
.node-card {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-3);
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
  margin: 0;
  padding: 0;
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
