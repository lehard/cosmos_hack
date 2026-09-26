<script setup lang="ts">
/**
 * «Рабочий стол» технолога: что сейчас требует действий — крупно и по важности.
 * Каждая задача — что случилось, почему важно, что сделать дальше, и кнопка
 * перехода в раздел, где это делается. Ничего не требует — так и сказано.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { statusDictionaries } from '@/shared/api/generated/statuses'
import type { InboxTask } from '../model/tasks'

const props = defineProps<{ tasks: readonly InboxTask[] }>()
const emit = defineEmits<{ go: [task: InboxTask] }>()
const { t } = useI18n()

const STAGES = statusDictionaries.investigation_stage.values as Record<string, { label: string }>
const K = 'widgets.analysis.inbox'

function title(x: InboxTask): string {
  const d = x.data
  switch (x.kind) {
    case 'investigation':
      return d.factor ? `${d.label} · ${d.factor}` : String(d.label)
    case 'recurring':
      return t(`${K}.recurringTitle`, { defect: d.defect, step: d.step })
    case 'actions':
      return t(`${K}.actionsTitle`)
    default:
      return t(`${K}.versionTitle`, { process: d.process, label: d.label })
  }
}
function lines(x: InboxTask): string[] {
  const d = x.data
  switch (x.kind) {
    case 'investigation':
      return [STAGES[String(d.stage)]?.label ?? '', t(`${K}.inScope`, { items: t('plural.items', { n: Number(d.size) }, Number(d.size)), initial: d.initial })].filter(Boolean)
    case 'recurring':
      return [t(`${K}.recurringCount`, { n: d.count }), t(`${K}.noActions`)]
    case 'actions':
      return [
        Number(d.overdue) ? t(`${K}.overdue`, { n: d.overdue }) : '',
        Number(d.ineffective) ? t(`${K}.ineffective`, { n: d.ineffective }) : '',
        Number(d.evaluation) ? t(`${K}.evaluation`, { n: d.evaluation }) : '',
      ].filter(Boolean)
    default:
      return [t(d.status === 'on_approval' ? `${K}.onApproval` : `${K}.draft`)]
  }
}
const next = (x: InboxTask) => (x.kind === 'investigation' && x.data.next ? String(x.data.next) : null)
const GO: Record<InboxTask['kind'], string> = {
  investigation: `${K}.goInvestigation`,
  recurring: `${K}.goRecurring`,
  actions: `${K}.goActions`,
  version: `${K}.goVersion`,
}
const count = computed(() => props.tasks.length)
</script>

<template>
  <div class="inbox" data-testid="inbox">
    <p class="head">
      <template v-if="count">{{ t(`${K}.needAction`) }} <strong data-testid="inbox-count">{{ count }}</strong></template>
      <template v-else>{{ t(`${K}.nothing`) }}</template>
    </p>
    <ul class="tasks">
      <li v-for="x in tasks" :key="x.id" class="task" :data-kind="x.kind" :data-tone="x.tone" :style="{ '--tone': `var(--ant-status-${x.tone})` }">
        <p class="title ant-wrap">{{ title(x) }}</p>
        <p v-for="(l, i) in lines(x)" :key="i" class="line ant-wrap">{{ l }}</p>
        <p v-if="next(x)" class="next ant-wrap" data-testid="inbox-next">{{ t(`${K}.next`, { what: next(x) }) }}</p>
        <button type="button" class="go" :data-go="x.tab" @click="emit('go', x)">{{ t(GO[x.kind]) }} →</button>
      </li>
    </ul>
  </div>
</template>

<style scoped>
.inbox {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-4);
  min-width: 0;
}

p {
  margin: 0;
}

.head {
  color: var(--ant-text-2);
}

.tasks {
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 0;
  list-style: none;
}

.task {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  min-width: 0;
  padding: var(--ant-space-5) 0 var(--ant-space-5) var(--ant-space-4);
  border-top: 1px solid var(--ant-border);
  box-shadow: inset 3px 0 0 var(--tone);
}

.title {
  font-size: var(--ant-fs-title);
  font-weight: var(--ant-fw-bold);
}

.line {
  color: var(--ant-text-2);
  font-size: var(--ant-fs-meta);
}

.next {
  font-weight: var(--ant-fw-bold);
}

.go {
  align-self: flex-start;
  margin-top: var(--ant-space-2);
  padding: var(--ant-space-1) var(--ant-space-4);
  border: 0;
  border-radius: var(--ant-radius-md);
  background: var(--ant-accent);
  color: var(--ant-surface);
  font: inherit;
  font-weight: var(--ant-fw-bold);
  cursor: pointer;
}

.go:hover {
  opacity: 0.9;
}
</style>
