<script setup lang="ts">
/**
 * План прогона на пульте (Д-85): «Сейчас ждём: ‹роль› — ‹действие›
 * (‹изделие›)», запланированные сбои — заранее и заметно, ближайшие события с
 * доменным временем; остановки (прогон ждёт нажатия человека на его столе)
 * выделены. Данные — `simulation.run.plan`; представление без запросов.
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { formatRunClock, planWait, plannedAlerts, upcoming, type PlanEntry, type RunPlan } from '@/entities/run'
import { codeToKey } from '@/shared/i18n'
import { ActionButton, EmptyState, SectionPanel } from '@/shared/ui'

const props = withDefaults(defineProps<{ plan: RunPlan; limit?: number }>(), { limit: 8 })
const { t, te } = useI18n()

const all = ref(false)
const wait = computed(() => planWait(props.plan))
const alerts = computed(() => plannedAlerts(props.plan))
const ahead = computed(() => upcoming(props.plan, Number.MAX_SAFE_INTEGER))
const shown = computed(() => (all.value ? ahead.value : ahead.value.slice(0, props.limit)))
const hidden = computed(() => ahead.value.length - shown.value.length)

const roleText = (role: string | undefined) => (role ? (te(`roles.${codeToKey(role)}`) ? t(`roles.${codeToKey(role)}`) : role) : '')
const time = (iso: string | undefined) => (iso ? formatRunClock(Date.parse(iso)) : '—')
/** «23.09 11:08» или «23.09 11:08–11:48» для группы событий. */
const span = (e: PlanEntry) => (e.until ? `${time(e.at)}–${time(e.until).slice(-5)}` : time(e.at))
/** Кто: роль и персона решения. */
const who = (e: PlanEntry) => [roleText(e.role), e.persona].filter(Boolean).join(' · ')
</script>

<template>
  <SectionPanel variant="subtle" :title="t('widgets.scenarios.plan.title')" data-testid="run-plan">
    <div class="plan">
      <p v-if="wait" class="wait ant-wrap" data-testid="plan-wait">
        <strong>{{ t('widgets.scenarios.plan.waitingNow') }}</strong>
        {{ roleText(wait.role) }} — {{ String(wait.what).replace(/^[^:«]*:\s*/, '') }}
      </p>
      <p v-else-if="plan.state === 'running'" class="meta ant-wrap" data-testid="plan-machines">{{ t('widgets.scenarios.plan.machinesGo') }}</p>

      <div v-for="(a, i) in alerts" :key="`alert-${i}`" class="alert ant-wrap" data-testid="plan-alert">
        <strong>{{ t('widgets.scenarios.plan.alert') }}</strong>
        <span>{{ a.title }}</span>
        <span class="meta">{{ t('widgets.scenarios.plan.at', { time: span(a) }) }}</span>
      </div>

      <EmptyState v-if="!ahead.length" compact :title="t('widgets.scenarios.plan.empty')" />
      <ol v-else class="entries" data-testid="plan-entries">
        <li v-for="(e, i) in shown" :key="i" class="entry" :data-kind="e.kind" :data-stop="e.stop || undefined" :data-waiting="e.waiting || undefined">
          <span class="at">{{ span(e) }}</span>
          <span class="what ant-box">
            <span class="title ant-wrap">{{ e.title }}</span>
            <span v-if="who(e) || e.item_id" class="meta ant-wrap">{{ [who(e), e.item_id].filter(Boolean).join(' · ') }}</span>
          </span>
          <span v-if="e.waiting" class="tag tag-wait">{{ t('widgets.scenarios.plan.waitingTag') }}</span>
          <span v-else-if="e.stop" class="tag tag-stop">{{ t('widgets.scenarios.plan.stopTag') }}</span>
          <span v-else-if="e.kind === 'alert'" class="tag tag-alert">{{ t('widgets.scenarios.plan.alertTag') }}</span>
        </li>
      </ol>
      <ActionButton v-if="hidden > 0 || all" size="small" quaternary data-testid="plan-more" :label="all ? t('widgets.scenarios.plan.less') : t('widgets.scenarios.plan.more', { n: hidden })" @click="all = !all" />
      <p class="meta ant-wrap">{{ t('widgets.scenarios.plan.clockHint') }}</p>
    </div>
  </SectionPanel>
</template>

<style scoped>
.plan {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
  min-width: 0;
}

.wait {
  margin: 0;
  padding: var(--ant-space-2) var(--ant-space-3);
  border-radius: var(--ant-radius-md);
  background: var(--ant-status-attention-soft);
}

.alert {
  display: flex;
  flex-wrap: wrap;
  gap: 2px var(--ant-space-2);
  padding: var(--ant-space-2) var(--ant-space-3);
  border-left: 3px solid var(--ant-status-danger);
  border-radius: var(--ant-radius-sm);
  background: var(--ant-status-danger-soft);
}

.entries {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.entry {
  display: grid;
  grid-template-columns: max-content minmax(0, 1fr) max-content;
  gap: var(--ant-space-3);
  align-items: baseline;
  padding: var(--ant-space-1) var(--ant-space-2);
  border-radius: var(--ant-radius-sm);
}

.entry[data-stop] {
  background: var(--ant-surface);
  box-shadow: inset 3px 0 0 var(--ant-status-attention);
}

.entry[data-stop] .title {
  font-weight: var(--ant-fw-bold);
}

.entry[data-waiting] {
  background: var(--ant-status-attention-soft);
}

.entry[data-kind='alert'] {
  box-shadow: inset 3px 0 0 var(--ant-status-danger);
}

.entry[data-kind='alert'] .title {
  color: var(--ant-status-danger-text);
  font-weight: var(--ant-fw-bold);
}

.at {
  color: var(--ant-text-2);
  font-family: var(--ant-font-mono);
  font-size: var(--ant-fs-meta);
  white-space: nowrap;
}

.what {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.meta {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

p.meta {
  margin: 0;
}

.tag {
  padding: 0 var(--ant-space-2);
  border-radius: var(--ant-radius-pill);
  font-size: var(--ant-fs-meta);
  white-space: nowrap;
}

.tag-stop,
.tag-wait {
  background: var(--ant-status-attention-soft);
  color: var(--ant-status-attention-text);
}

.tag-wait {
  font-weight: var(--ant-fw-bold);
}

.tag-alert {
  background: var(--ant-status-danger-soft);
  color: var(--ant-status-danger-text);
}
</style>
