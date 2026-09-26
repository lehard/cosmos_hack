<script setup lang="ts">
/**
 * Задачи — представление (FR-57, FR-8): открытые задачи и запросы решения с
 * отметкой, тревоги и просроченные решения с ценой задержки. Выполненные —
 * свёрнуты. Заголовок «Задачи» несёт рамка виджета — внутри не повторяется;
 * пустые разделы тревог и просроченных решений не показываются (UI-38).
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NAlert, NCollapse, NCollapseItem } from 'naive-ui'
import type { NotificationSummary } from '@/entities/notification'
import type { TaskEntry } from '@/entities/task'
import { TaskInbox } from '@/features/task-actions'
import type { DrillRef } from '@/shared/model/drill'
import type { Density } from '@/shared/config/widget'
import { naiveSizeOf } from '@/shared/config/widget'
import { useProblemText } from '@/shared/i18n/problem'
import { ActionButton, SectionPanel } from '@/shared/ui'
import type { NoticeRow, TaskSection } from '../model/slice'

const props = withDefaults(
  defineProps<{
    sections: readonly TaskSection[]
    summary?: NotificationSummary | null
    open: readonly TaskEntry[] | null
    closed: readonly TaskEntry[]
    basisSeq: number
    tasksError?: unknown
    alerts: readonly NoticeRow[] | null
    alertsError?: unknown
    escalations: readonly NoticeRow[] | null
    escalationsError?: unknown
    canAct?: boolean
    density?: Density
    canOpen?: (ref: DrillRef) => boolean
  }>(),
  { summary: null, tasksError: undefined, alertsError: undefined, escalationsError: undefined, canAct: true, density: 'comfortable', canOpen: () => true },
)
const emit = defineEmits<{ open: [ref: DrillRef] }>()
const { t } = useI18n()
const problemText = useProblemText()
const size = computed(() => naiveSizeOf(props.density))

</script>

<template>
  <div class="tasks" data-testid="tasks-view">

    <SectionPanel v-if="sections.includes('task')" variant="plain" data-testid="section-tasks">
      <NAlert v-if="tasksError && !open" type="error" :bordered="false">{{ problemText(tasksError) }}</NAlert>
      <!-- Нет открытых — одной строкой, выполненные ниже (UI-44). -->
      <p v-else-if="open && !open.length" class="ant-muted" data-testid="no-open-tasks">{{ t('widgets.shopFloor.tasks.noOpen') }}</p>
      <TaskInbox v-else-if="open" :tasks="open" :basis-seq="basisSeq" :can-act="canAct" :density="density" :can-open="canOpen" @open="(r) => emit('open', r)" />
      <NCollapse v-if="closed.length" data-testid="closed-tasks">
        <NCollapseItem :title="t('widgets.shopFloor.tasks.closed', { n: closed.length })" name="closed">
          <TaskInbox :tasks="closed" :basis-seq="basisSeq" :can-act="false" :density="density" :can-open="canOpen" @open="(r) => emit('open', r)" />
        </NCollapseItem>
      </NCollapse>
    </SectionPanel>

    <SectionPanel v-if="sections.includes('escalation') && (escalationsError || escalations?.length)" :title="t('widgets.shopFloor.tasks.escalations')" variant="plain" data-testid="section-escalations">
      <NAlert v-if="escalationsError && !escalations" type="error" :bordered="false">{{ problemText(escalationsError) }}</NAlert>
      <ul v-else-if="escalations" class="notices">
        <li v-for="r in escalations" :key="r.id" class="notice" :data-escalation="r.id">
          <p class="ant-clamp-2" :class="r.severe ? 'severe' : 'mild'" :title="r.text">{{ r.text }}</p>
          <div v-if="r.ref && canOpen(r.ref)">
            <ActionButton text type="primary" :size="size" :label="t('common.actions.open')" @click="emit('open', r.ref)" />
          </div>
        </li>
      </ul>
    </SectionPanel>

    <SectionPanel v-if="sections.includes('alarm') && (alertsError || alerts?.length)" :title="t('liveMap.alerts.title')" variant="plain" data-testid="section-alarms">
      <NAlert v-if="alertsError && !alerts" type="error" :bordered="false">{{ problemText(alertsError) }}</NAlert>
      <ul v-else-if="alerts" class="notices">
        <li v-for="r in alerts" :key="r.id" class="notice" :data-alert="r.id">
          <!-- Тревога с объектом — нажимается целиком: окно операции, изделия… (UI-45). -->
          <button v-if="r.ref && canOpen(r.ref)" type="button" class="notice-link" data-testid="open-alert" @click="emit('open', r.ref)">
            <span v-if="r.time" class="ant-muted">{{ r.time }} · </span><span :class="r.severe ? 'severe' : 'mild'">{{ r.text }}</span>
            <span class="go">{{ t('widgets.shopFloor.tasks.whatToDo') }} →</span>
          </button>
          <p v-else class="ant-clamp-2" :title="r.text">
            <span v-if="r.time" class="ant-muted">{{ r.time }} · </span><span :class="r.severe ? 'severe' : 'mild'">{{ r.text }}</span>
          </p>
        </li>
      </ul>
    </SectionPanel>
  </div>
</template>

<style scoped>
.tasks {
  display: flex;
  flex-direction: column;
  gap: var(--ant-gap);
  min-width: 0;
}

.line {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-1) var(--ant-space-2);
  align-items: center;
  min-width: 0;
}

.notices {
  display: flex;
  flex-direction: column;
  min-width: 0;
  margin: 0;
  padding: 0;
  list-style: none;
}

.notice {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  min-width: 0;
  padding: var(--ant-space-2) 0;
  border-bottom: 1px solid var(--ant-border);
}

.notice:last-child {
  border-bottom: 0;
}

.notice-link {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: var(--ant-space-1) 0;
  border: 0;
  background: none;
  font: inherit;
  text-align: start;
  cursor: pointer;
}

.notice-link:hover .go {
  text-decoration: underline;
}

.go {
  color: var(--ant-accent);
  font-size: var(--ant-fs-meta);
}

.severe {
  color: var(--ant-status-danger-text);
}

.mild {
  color: var(--ant-status-attention-text);
}
</style>
