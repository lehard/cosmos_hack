<script setup lang="ts">
/**
 * Виджет «Тревоги» (FR-8) — правая панель руководителя: просроченные изоляции и
 * сроки на точках предъявления, эскалации с ценой задержки, аномалии узлов,
 * нарушения целостности. Клик по тревоге — к объекту тревоги (FR-7).
 * Тексты — ключами liveMap.alerts; вид аномалии — liveMap.anomalies.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAlerts, type AlertEntry } from '@/entities/notification'
import { useDrillDown } from '@/features/drill-down'
import { backendModeOf } from '@/shared/api/response'
import { statusPalette, type StatusTone } from '@/shared/api/generated/statuses'
import type { WidgetProps } from '@/shared/config/widget'
import { codeToKey } from '@/shared/i18n'
import { formatMinutes } from '@/shared/lib/duration'
import { WidgetFrame } from '@/shared/ui'

const props = defineProps<WidgetProps>()
const { t, te, d } = useI18n()
const drill = useDrillDown()

const runId = computed(() => (typeof props.slice.run_id === 'string' && props.slice.run_id ? props.slice.run_id : undefined))
const query = useAlerts(computed(() => (runId.value ? { run_id: runId.value } : {})))
const alerts = computed(() => query.data.value?.data ?? null)

const TONE: Record<AlertEntry['kind'], StatusTone> = {
  overdue_isolation: 'danger',
  gate_overdue: 'attention',
  not_moved_to_isolator: 'danger',
  anomaly: 'attention',
  escalation: 'danger',
  integrity_violation: 'critical',
}

const dash = '—'

/** Текст тревоги по виду; неизвестный вид аномалии — UNKNOWN(код), а не похожий текст. */
function text(a: AlertEntry): string {
  switch (a.kind) {
    case 'overdue_isolation':
      return t('liveMap.alerts.overdueIsolation', { item: a.item ?? dash })
    case 'gate_overdue':
      return t('liveMap.alerts.gateOverdue', { gate: a.gate ?? dash })
    case 'not_moved_to_isolator':
      return t('liveMap.alerts.notMovedToIsolator', { item: a.item ?? dash })
    case 'anomaly': {
      const key = a.anomaly ? `liveMap.anomalies.${codeToKey(a.anomaly)}` : ''
      const what = key && te(key) ? t(key, { threshold: dash }) : `UNKNOWN(${a.anomaly ?? ''})`
      // Имя шага из BPMN; код узла — только если имени нет.
      return t('liveMap.alerts.anomaly', { node: a.node_name || a.node || dash, what })
    }
    case 'escalation':
      return `${t('common.notifications.escalation')}: ${t('liveMap.attention.overdueDecision', {
        target: a.target ?? dash,
        overdue: formatMinutes(t, a.overdue_minutes ?? 0),
        items: t('plural.items', { n: a.items ?? 0 }, a.items ?? 0),
        operations: t('plural.operations', { n: a.operations ?? 0 }, a.operations ?? 0),
      })}`
    case 'integrity_violation':
      return t('liveMap.alerts.integrityViolation')
  }
}

const rows = computed(() =>
  (alerts.value ?? []).map((a) => ({
    a,
    text: text(a),
    color: statusPalette[TONE[a.kind]],
    time: d(new Date(a.at), 'time'),
    link: a.ref && drill.canOpen(a.ref) ? a.ref : null,
  })),
)
/** Признак дефекта — тревоги об изделиях в изоляции; прочие тревоги не про годность (NFR-UI-4). */
const state = computed(() =>
  alerts.value?.some((a) => a.kind === 'overdue_isolation' || a.kind === 'not_moved_to_isolator') ? 'defect_indication' : 'normal',
)
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="backendModeOf(query.data.value)"
    :state="state"
    :loading="query.isPending.value && !alerts"
    :error="alerts ? undefined : query.error.value"
    :empty="!!alerts && !alerts.length"
    :data-widget="widgetId"
  >
    <ol class="alerts">
      <li v-for="r in rows" :key="r.a.alert_id" :data-kind="r.a.kind" :style="{ borderColor: r.color }">
        <time class="time" :datetime="r.a.at">{{ r.time }}</time>
        <button v-if="r.link" type="button" class="link" @click="drill.open(r.link)">{{ r.text }}</button>
        <span v-else>{{ r.text }}</span>
      </li>
    </ol>
  </WidgetFrame>
</template>

<style scoped>
.alerts {
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-height: 420px;
  margin: 0;
  padding: 0;
  overflow: auto;
  list-style: none;
}

.alerts li {
  display: flex;
  gap: 8px;
  padding: 4px 8px;
  border-left: 3px solid;
  font-size: var(--ant-fs-body);
}

.time {
  flex: none;
  color: var(--ant-text-3);
  font-family: var(--ant-font-mono);
}

.link {
  padding: 0;
  border: 0;
  background: none;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.link:hover {
  text-decoration: underline;
}
</style>
