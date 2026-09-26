<script setup lang="ts">
/**
 * Правое окно корректирующей меры (Д-70; FR-64): тип и направление («почему
 * возник» / «почему пропустили»), план проверки эффективности, окно
 * наблюдения, история по попыткам. Кнопки — в нижней панели окна: «Отметить
 * внедрение» (назначена или переоткрыта), «Эффективна» и «Не помогла»
 * (внедрена; «эффективна» — после окна наблюдения, иначе сервер отказывает
 * с кодом incident.action_state). Не помогла — мера переоткрывается.
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NAlert, NInput } from 'naive-ui'
import { useSession } from '@/entities/session'
import { useSuggestionCommand, type CorrectiveActionView } from '@/entities/suggestion'
import { naiveSizeOf, type Density } from '@/shared/config/widget'
import { useProblemText } from '@/shared/i18n/problem'
import { useMomentStore } from '@/shared/model/moment'
import { ActionButton, FormField, KeyValue, KeyValueList, RecordDrawer, SectionPanel } from '@/shared/ui'

const props = withDefaults(defineProps<{ action: CorrectiveActionView | null; asOf?: string; density?: Density }>(), { asOf: undefined, density: 'comfortable' })
const emit = defineEmits<{ close: [] }>()
const { t, d } = useI18n()
const problemText = useProblemText()
const moment = useMomentStore()
const session = useSession()
const command = useSuggestionCommand(() => {
  const s = session.data.value?.data
  return { policySeq: s?.policy_seq ?? 0, workplaceId: s?.workplace?.id }
})

type Step = 'implement' | 'effective' | 'failed'
const step = ref<Step | null>(null)
const text = ref('')
const done = ref(false)
watch(
  () => props.action?.action_id,
  () => {
    step.value = null
    text.value = ''
    done.value = false
    command.reset()
  },
)

const time = (iso?: string | null) => (iso ? d(new Date(iso), 'dateTime') : '—')
const canImplement = computed(() => props.action?.status === 'assigned' || props.action?.status === 'reopened')
const canEvaluate = computed(() => props.action?.status === 'implemented')
/** «Эффективна» — только после окна наблюдения (FR-64). */
const windowOpen = computed(() => {
  const due = props.action?.evaluation_due_at
  if (!due) return false
  const now = props.asOf ? new Date(props.asOf) : new Date()
  return now < new Date(due)
})

async function submit(): Promise<void> {
  const a = props.action
  if (!a || !step.value) return
  try {
    if (step.value === 'implement') await command.mutateAsync({ kind: 'implement', action: a, note: text.value.trim() || undefined })
    else await command.mutateAsync({ kind: 'evaluate', action: a, result: step.value, evidence: text.value.trim() || undefined })
    done.value = true
    step.value = null
  } catch {
    // Текст отказа (с кодом) — из command.error.
  }
}
</script>

<template>
  <RecordDrawer :show="!!action" :kind-label="t('widgets.quality.drawerKind')" :number="action?.action_id ?? ''" :subtitle="action?.title ?? action?.plan.metric ?? ''" @close="emit('close')">
    <template v-if="action" #status>
      <span class="status" :data-status="action.status">{{ t(`widgets.quality.status.${action.status}`) }}</span>
    </template>
    <div v-if="action" class="body" data-testid="action-drawer" :data-action="action.action_id">
      <p class="principle ant-wrap">{{ t('widgets.quality.principle') }}</p>
      <KeyValueList>
        <KeyValue :label="t('widgets.quality.col.type')" :value="`${t(`widgets.quality.type.${action.action_type}`)} · ${t(`widgets.quality.direction.${action.direction}`)}`" />
        <KeyValue :label="t('widgets.quality.incident')" :value="action.incident_id" mono />
        <KeyValue :label="t('widgets.quality.col.owner')" :value="action.owner" mono />
        <KeyValue :label="t('widgets.quality.col.due')" :value="action.due_at ? time(action.due_at) : null" />
        <KeyValue :label="t('widgets.quality.cycle')" :value="action.cycle" />
      </KeyValueList>
      <SectionPanel variant="subtle" :title="t('widgets.quality.plan')">
        <KeyValueList data-testid="plan">
          <KeyValue :label="t('widgets.quality.metric')" :value="action.plan.metric" />
          <KeyValue :label="t('widgets.quality.baseline')" :value="action.plan.baseline" />
          <KeyValue :label="t('widgets.quality.window')" :value="t('widgets.quality.windowDays', { n: action.plan.window_days })" />
          <KeyValue :label="t('widgets.quality.criterion')" :value="action.plan.success_criterion" />
          <KeyValue :label="t('widgets.quality.enhanced')" :value="action.plan.enhanced_control ?? null" />
          <KeyValue v-if="action.evaluation_due_at" :label="t('widgets.quality.evaluationFrom')" :value="time(action.evaluation_due_at)" />
        </KeyValueList>
      </SectionPanel>
      <SectionPanel variant="subtle" :title="t('widgets.quality.historyTitle')">
        <ul class="list" data-testid="history">
          <li v-for="h in action.history" :key="h.event_id" class="ant-wrap">
            {{ time(h.at) }} · {{ t(`widgets.quality.history.${h.type}`) }}<template v-if="h.result"> — {{ t(`widgets.quality.result.${h.result}`) }}</template>
            · {{ t('widgets.quality.cycle') }} {{ h.cycle }}<template v-if="h.actor"> · {{ h.actor }}</template><template v-if="h.text || h.evidence"> · {{ h.text ?? h.evidence }}</template>
          </li>
        </ul>
      </SectionPanel>
      <form v-if="step" class="form" @submit.prevent="submit">
        <FormField :label="t(step === 'implement' ? 'widgets.quality.note' : 'widgets.quality.evidence')">
          <NInput v-model:value="text" type="textarea" :autosize="{ minRows: 2 }" :size="naiveSizeOf(density)" data-testid="text" />
        </FormField>
      </form>
      <p v-if="done" class="ok ant-wrap" data-testid="done">{{ t('widgets.quality.done') }}</p>
      <NAlert v-if="command.error.value" type="error" :bordered="false" :show-icon="false" data-testid="command-error">
        <span class="ant-wrap">{{ problemText(command.error.value) }}</span>
      </NAlert>
    </div>
    <template v-if="action && (canImplement || canEvaluate)" #actions>
      <template v-if="step">
        <ActionButton :label="t('widgets.quality.cancel')" :size="naiveSizeOf(density)" @click="step = null" />
        <ActionButton
          type="primary"
          :label="t('widgets.quality.confirm')"
          :size="naiveSizeOf(density)"
          :disabled="command.isPending.value || moment.isReplay"
          :loading="command.isPending.value"
          data-testid="submit"
          @click="submit"
        />
      </template>
      <template v-else>
        <ActionButton v-if="canImplement" type="primary" :label="t('widgets.quality.implement')" :size="naiveSizeOf(density)" :disabled="moment.isReplay" data-testid="implement" @click="step = 'implement'" />
        <template v-if="canEvaluate">
          <ActionButton
            type="primary"
            :label="t('widgets.quality.effective')"
            :size="naiveSizeOf(density)"
            :disabled="moment.isReplay || windowOpen"
            :title="windowOpen ? `${t('widgets.quality.evaluationFrom')} ${time(action.evaluation_due_at)}` : undefined"
            data-testid="effective"
            @click="step = 'effective'"
          />
          <ActionButton :label="t('widgets.quality.failed')" :size="naiveSizeOf(density)" :disabled="moment.isReplay" data-testid="failed" @click="step = 'failed'" />
        </template>
      </template>
    </template>
  </RecordDrawer>
</template>

<style scoped>
.body {
  display: flex;
  flex-direction: column;
  gap: var(--ant-gap);
  font-size: var(--ant-fs-body);
}

.principle {
  margin: 0;
  padding: var(--ant-space-2) var(--ant-space-3);
  border-left: 3px solid var(--ant-status-neutral);
  background: var(--ant-surface-subtle);
  font-size: var(--ant-fs-meta);
}

.list {
  margin: 0;
  padding-left: var(--ant-space-5);
}

.ok {
  margin: 0;
  color: var(--ant-status-success);
}

.form {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-3);
}

.status {
  font-size: var(--ant-fs-meta);
  color: var(--ant-text-2);
}
</style>
