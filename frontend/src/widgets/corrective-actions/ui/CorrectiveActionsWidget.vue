<script setup lang="ts">
/**
 * «Меры и качество» — взгляд руководителя по качеству (эпик 42; FR-138, в MVP
 * совмещён со столами руководителя производства и технолога) и корректирующие
 * меры с эффективностью (FR-64). Сверху — сводка: открытые, просроченные, не
 * помогшие меры, висящий временно усиленный контроль, «пора оценить»,
 * повторяющиеся проблемы; затем список мер (щелчок — правое окно меры с
 * кнопками в нижней панели, Д-70); повторяющиеся проблемы; организационная
 * память — что пробовали и с каким результатом.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useNcGroups } from '@/entities/incident'
import { useCorrectiveActions, useOpenRecord } from '@/entities/suggestion'
import { backendModeOf } from '@/shared/api/response'
import type { WidgetDataState, WidgetProps } from '@/shared/config/widget'
import { DataTable, EmptyState, SectionPanel, WidgetFrame } from '@/shared/ui'
import ActionDrawer from './ActionDrawer.vue'

defineProps<WidgetProps>()
const { t, d } = useI18n()
const q = useCorrectiveActions()
const data = computed(() => q.data.value?.data ?? null)
const items = computed(() => data.value?.items ?? [])
const { open, setOpen } = useOpenRecord(['action'] as const)
const current = computed(() => (open.value ? (items.value.find((a) => a.action_id === open.value?.id) ?? null) : null))

/**
 * Повторяющаяся проблема словами: вид дефекта и шаг — из групп разбора (названия
 * сервера), если такая группа есть; иначе — код, выдумывать не будем.
 */
const groups = useNcGroups()
function recurringText(defect: string, step: string): { defect: string; step: string } {
  const g = (groups.data.value ?? []).find((x) => x.defect_type === defect && x.operation === step)
  return { defect: g?.defect_type_label || defect, step: g?.operation_label || step }
}

/** Просроченные или не помогшие меры — «признак дефекта» рамки: требует внимания. */
const state = computed<WidgetDataState>(() => (data.value && (data.value.summary.overdue || data.value.summary.ineffective) ? 'defect_indication' : 'normal'))

const tiles = computed(() => {
  const s = data.value?.summary
  if (!s) return []
  return [
    { id: 'open', value: s.open, warn: false },
    { id: 'overdue', value: s.overdue, warn: s.overdue > 0 },
    { id: 'ineffective', value: s.ineffective, warn: s.ineffective > 0 },
    { id: 'hanging', value: s.hanging_temporary, warn: s.hanging_temporary > 0 },
    { id: 'evaluationDue', value: s.evaluation_due, warn: s.evaluation_due > 0 },
    { id: 'recurring', value: s.recurring, warn: false },
  ]
})
const time = (iso?: string | null) => (iso ? d(new Date(iso), 'dateTime') : '—')
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="backendModeOf(q.data.value)"
    :state="state"
    :loading="q.isPending.value && !data"
    :error="data ? undefined : q.error.value"
    :data-widget="widgetId"
  >
    <div v-if="data" class="quality" data-testid="quality">
      <ul class="tiles" data-testid="summary">
        <li v-for="x in tiles" :key="x.id" class="tile" :data-tile="x.id" :data-warn="x.warn">
          <span class="value">{{ x.value }}</span>
          <span class="label ant-clamp-2">{{ t(`widgets.quality.summary.${x.id}`) }}</span>
        </li>
      </ul>

      <SectionPanel :title="t('widgets.quality.actionsTitle')" :subtitle="t('widgets.quality.actionsSubtitle')">
        <EmptyState v-if="!items.length" compact :title="t('widgets.quality.empty')" data-testid="no-actions" />
        <DataTable v-else :caption="t('widgets.quality.actionsTitle')">
          <thead>
            <tr>
              <th scope="col">{{ t('widgets.quality.col.action') }}</th>
              <th scope="col">{{ t('widgets.quality.col.type') }}</th>
              <th scope="col">{{ t('widgets.quality.col.owner') }}</th>
              <th scope="col">{{ t('widgets.quality.col.status') }}</th>
              <th scope="col">{{ t('widgets.quality.col.flags') }}</th>
              <th scope="col">{{ t('widgets.quality.col.due') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="a in items"
              :key="a.action_id"
              class="row"
              tabindex="0"
              :data-action="a.action_id"
              :data-status="a.status"
              :aria-selected="open?.id === a.action_id"
              @click="setOpen({ kind: 'action', id: a.action_id })"
              @keydown.enter.prevent="setOpen({ kind: 'action', id: a.action_id })"
            >
              <td class="ant-box">
                <span class="ant-clamp-2">{{ a.title ?? a.plan.metric }}</span>
                <span class="muted ant-ellipsis">{{ a.action_id }} · {{ a.incident_id }}</span>
              </td>
              <td class="ant-box"><span class="ant-wrap">{{ t(`widgets.quality.type.${a.action_type}`) }}</span></td>
              <td class="ant-box ant-mono"><span class="ant-wrap">{{ a.owner }}</span></td>
              <td class="ant-box"><span class="ant-wrap">{{ t(`widgets.quality.status.${a.status}`) }}</span></td>
              <td class="ant-box">
                <span v-for="f in a.flags" :key="f" class="flag" :data-flag="f">{{ t(`widgets.quality.flag.${f}`) }}</span>
              </td>
              <td class="ant-box"><span class="ant-ellipsis">{{ time(a.due_at) }}</span></td>
            </tr>
          </tbody>
        </DataTable>
      </SectionPanel>

      <div class="columns">
        <SectionPanel variant="subtle" :title="t('widgets.quality.recurringTitle')" :subtitle="t('widgets.quality.recurringSubtitle')">
          <EmptyState v-if="!data.recurring.length" compact :title="t('widgets.quality.recurringEmpty')" />
          <ul v-else class="list" data-testid="recurring">
            <li v-for="r in data.recurring" :key="r.defect_type + r.step_key" class="ant-wrap">
              {{ t('widgets.quality.recurringRow', { ...recurringText(r.defect_type, r.step_key), n: r.count }) }} ·
              <span :class="r.with_action ? 'muted' : 'warn'">{{ r.with_action ? t('widgets.quality.withAction') : t('widgets.quality.withoutAction') }}</span>
            </li>
          </ul>
        </SectionPanel>
        <SectionPanel variant="subtle" :title="t('widgets.quality.memoryTitle')" :subtitle="t('widgets.quality.memorySubtitle')">
          <EmptyState v-if="!data.memory.length" compact :title="t('widgets.quality.memoryEmpty')" />
          <ul v-else class="list" data-testid="memory">
            <li v-for="m in data.memory" :key="m.action_id" class="ant-wrap" :data-outcome="m.outcome">
              {{ m.title }} — <strong>{{ t(`widgets.quality.outcome.${m.outcome}`) }}</strong
              ><template v-if="m.factor"> · {{ m.factor }}</template>
            </li>
          </ul>
        </SectionPanel>
      </div>
    </div>
    <ActionDrawer :action="current" :as-of="data?.as_of" :density="density" @close="setOpen(null)" />
  </WidgetFrame>
</template>

<style scoped>
.quality {
  display: flex;
  flex-direction: column;
  gap: var(--ant-gap);
  font-size: var(--ant-fs-body);
}

.tiles {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: var(--ant-space-2);
  margin: 0;
  padding: 0;
  list-style: none;
}

.tile {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  padding: var(--ant-space-2) var(--ant-space-3);
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-md);
  background: var(--ant-surface);
}

.tile[data-warn='true'] {
  border-color: var(--ant-status-attention);
  box-shadow: inset 3px 0 0 var(--ant-status-attention);
}

.value {
  font-size: var(--ant-fs-title);
  font-weight: var(--ant-fw-bold);
}

.label,
.muted {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.muted {
  display: block;
}

.row {
  cursor: pointer;
}

.row:hover {
  background: var(--ant-surface-hover);
}

.row[aria-selected='true'] {
  background: var(--ant-accent-soft);
}

.flag {
  display: inline-block;
  margin: 0 var(--ant-space-1) var(--ant-space-1) 0;
  padding: 0 var(--ant-space-2);
  border-radius: var(--ant-radius-pill);
  background: var(--ant-status-attention-soft);
  font-size: var(--ant-fs-meta);
}

.flag[data-flag='awaiting_window'] {
  background: var(--ant-surface-subtle);
}

.columns {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: var(--ant-gap);
}

.list {
  margin: 0;
  padding-left: var(--ant-space-5);
}

.warn {
  color: var(--ant-status-danger);
}
</style>
