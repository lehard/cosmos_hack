<script setup lang="ts">
/**
 * Меры по причине (FR-64, UI-32): у значимой проблемы две причины — и меры по
 * каждой. Под «почему возник» — меры, предотвращающие появление
 * (`prevent_occurrence`), под «почему не остановили» — улучшающие обнаружение
 * (`improve_detection`). Список мер этого направления со статусом и сроком и
 * «Назначить меру»: что сделать, вид меры, план проверки эффективности —
 * критерий успеха и окно наблюдения обязательны («внедрено ≠ результативно»).
 * Ответственный — тот, кто назначает (свой сеанс).
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NInput, NInputNumber, NSelect } from 'naive-ui'
import type { CorrectiveActionView } from '@/shared/api/generated/model'
import { codeToKey } from '@/shared/i18n'
import { naiveSizeOf, type Density } from '@/shared/config/widget'
import { useMomentStore } from '@/shared/model/moment'
import { ActionButton, FormField } from '@/shared/ui'

export interface AssignInput {
  title: string
  action_type: 'correction' | 'corrective_action' | 'preventive_action'
  metric: string
  success_criterion: string
  window_days: number
}

const props = withDefaults(
  defineProps<{
    /** Направление мер этой причины. */
    direction: 'prevent_occurrence' | 'improve_detection'
    /** Меры инцидента этого направления. */
    actions: readonly CorrectiveActionView[]
    /** Право `analysis.action.assign`. */
    canAssign?: boolean
    busy?: boolean
    /** Кто будет ответственным (имя из сеанса). */
    ownerName?: string | null
    density?: Density
  }>(),
  { canAssign: false, busy: false, ownerName: null, density: 'compact' },
)
const emit = defineEmits<{ assign: [input: AssignInput] }>()

const { t, te, d } = useI18n()
const moment = useMomentStore()
const size = computed(() => naiveSizeOf(props.density))

const statusText = (s: string) => {
  const key = `statuses.correctiveAction.${codeToKey(s)}`
  return te(key) ? t(key) : s
}
const typeText = (x: string) => t(`widgets.analysis.actions.kind.${codeToKey(x)}`)

const open = ref(false)
const title = ref('')
const kind = ref<AssignInput['action_type']>('corrective_action')
const metric = ref('')
const success = ref('')
const windowDays = ref<number | null>(14)
const KINDS: AssignInput['action_type'][] = ['correction', 'corrective_action', 'preventive_action']
const kindOptions = computed(() => KINDS.map((k) => ({ value: k, label: t(`widgets.analysis.actions.kind.${codeToKey(k)}`) })))

const missing = computed(() => {
  const out: string[] = []
  if (!title.value.trim()) out.push(t('widgets.analysis.actions.needTitle'))
  if (!success.value.trim()) out.push(t('widgets.analysis.actions.needSuccess'))
  if (!windowDays.value || windowDays.value < 1) out.push(t('widgets.analysis.actions.needWindow'))
  return out
})

function start(): void {
  open.value = true
  title.value = ''
  kind.value = 'corrective_action'
  metric.value = ''
  success.value = ''
  windowDays.value = 14
}

function submit(): void {
  if (missing.value.length || !windowDays.value) return
  emit('assign', { title: title.value.trim(), action_type: kind.value, metric: metric.value.trim(), success_criterion: success.value.trim(), window_days: windowDays.value })
  open.value = false
}
</script>

<template>
  <div class="measures" :data-direction="direction" data-testid="branch-actions">
    <p class="title">{{ t(direction === 'prevent_occurrence' ? 'widgets.analysis.actions.titleMade' : 'widgets.analysis.actions.titleMissed') }}</p>
    <ul v-if="actions.length" class="list">
      <li v-for="a in actions" :key="a.action_id" class="item" :data-status="a.status">
        <span class="ant-wrap"><strong>{{ a.title || typeText(a.action_type) }}</strong></span>
        <span class="meta ant-wrap">
          {{ statusText(a.status) }}<template v-if="a.due_at"> · {{ t('widgets.analysis.actions.due', { time: d(new Date(a.due_at), 'date') }) }}</template>
          <template v-if="a.plan?.success_criterion"> · {{ t('widgets.analysis.actions.success', { what: a.plan.success_criterion }) }}</template>
        </span>
      </li>
    </ul>
    <p v-else class="none ant-wrap" data-testid="no-actions">{{ t('widgets.analysis.actions.none') }}</p>

    <ActionButton
      v-if="canAssign && !open"
      overflow="wrap"
      :size="size"
      quaternary
      :disabled="busy || moment.isReplay"
      data-testid="assign-open"
      @click="start"
      :label="t('widgets.analysis.actions.assign')"
    />

    <form v-if="open" class="form" data-testid="assign-form" @submit.prevent="submit">
      <FormField :label="t('widgets.analysis.actions.what')" required>
        <NInput v-model:value="title" :size="size" :maxlength="300" data-testid="assign-title" />
      </FormField>
      <FormField :label="t('widgets.analysis.actions.kindLabel')">
        <NSelect v-model:value="kind" :options="kindOptions" :size="size" data-testid="assign-kind" />
      </FormField>
      <p class="plan-title">{{ t('ncCard.investigation.effectivenessPlan') }}</p>
      <FormField :label="t('ncCard.investigation.metric')">
        <NInput v-model:value="metric" :size="size" :maxlength="300" data-testid="assign-metric" />
      </FormField>
      <FormField :label="t('ncCard.investigation.successCriterion')" required>
        <NInput v-model:value="success" :size="size" :maxlength="300" data-testid="assign-success" />
      </FormField>
      <FormField :label="t('widgets.analysis.actions.windowDays')" required>
        <NInputNumber v-model:value="windowDays" :size="size" :min="1" :max="365" :precision="0" data-testid="assign-window" />
      </FormField>
      <p v-if="ownerName" class="meta">{{ t('widgets.analysis.actions.owner', { name: ownerName }) }}</p>
      <p v-if="missing.length" class="missing ant-wrap" data-testid="assign-missing">{{ t('widgets.analysis.riskScope.stillNeeded', { what: missing.join(' · ') }) }}</p>
      <div class="buttons">
        <ActionButton overflow="wrap" :size="size" type="primary" attr-type="submit" :disabled="missing.length > 0 || busy || moment.isReplay" data-testid="assign-submit" :label="t('widgets.analysis.actions.submit')" />
        <ActionButton overflow="wrap" :size="size" quaternary @click="open = false" :label="t('common.actions.cancel')" />
      </div>
    </form>
  </div>
</template>

<style scoped>
.measures {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
  align-items: flex-start;
  min-width: 0;
}

p {
  margin: 0;
}

.title,
.plan-title {
  font-weight: var(--ant-fw-bold);
}

.list {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  width: 100%;
  margin: 0;
  padding: 0;
  list-style: none;
}

.item {
  display: flex;
  flex-direction: column;
  min-width: 0;
  padding-left: var(--ant-space-3);
  border-left: 3px solid var(--ant-status-info);
}

.item[data-status='effective'] {
  border-left-color: var(--ant-status-success);
}

.item[data-status='reopened'] {
  border-left-color: var(--ant-status-danger);
}

.meta,
.none {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.form {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-3);
  width: 100%;
  padding-left: var(--ant-space-3);
  border-left: 3px solid var(--ant-border-strong);
}

.missing {
  color: var(--ant-status-attention-text);
  font-weight: var(--ant-fw-bold);
}

.buttons {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-2);
}
</style>
