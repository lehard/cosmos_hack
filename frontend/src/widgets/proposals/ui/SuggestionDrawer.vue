<script setup lang="ts">
/**
 * Правое окно предложения (Д-70; FR-63, UJ-1): суть, оценка эффекта, узел
 * процесса (ссылка на карту процесса), основания — записи журнала, история
 * решений. Кнопки — в нижней панели окна: «Передать ответственному» (задачу
 * ставит notifications), «Принять в работу», «Отклонить» — с основанием.
 * Система ничего не применяет сама.
 */
import { computed, inject, ref, watch } from 'vue'
import { routerKey } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { NAlert, NInput, NSelect } from 'naive-ui'
import { useSession } from '@/entities/session'
import { useSuggestionCommand, type Suggestion } from '@/entities/suggestion'
import { naiveSizeOf, type Density } from '@/shared/config/widget'
import { useProblemText } from '@/shared/i18n/problem'
import { useMomentStore } from '@/shared/model/moment'
import { ActionButton, FormField, KeyValue, KeyValueList, RecordDrawer, SectionPanel } from '@/shared/ui'

const props = withDefaults(defineProps<{ suggestion: Suggestion | null; density?: Density }>(), { density: 'comfortable' })
const emit = defineEmits<{ close: [] }>()
const { t, d } = useI18n()
const problemText = useProblemText()
const moment = useMomentStore()
const session = useSession()
const router = inject(routerKey, null)

const command = useSuggestionCommand(() => {
  const s = session.data.value?.data
  return { policySeq: s?.policy_seq ?? 0, workplaceId: s?.workplace?.id }
})

type Action = 'forward' | 'accept' | 'reject'
const action = ref<Action | null>(null)
const responsibleId = ref('')
const responsibleRole = ref<string | null>(null)
const text = ref('')
const done = ref<'forward' | 'resolve' | null>(null)

watch(
  () => props.suggestion?.suggestion_id,
  () => {
    action.value = null
    responsibleId.value = ''
    responsibleRole.value = props.suggestion?.responsible_role ?? null
    text.value = ''
    done.value = null
    command.reset()
  },
  { immediate: true },
)

const ROLES = ['site_foreman', 'technologist', 'head_of_qc', 'production_manager'] as const
const roles = computed(() => ROLES.map((r) => ({ value: r, label: t(`widgets.suggestions.role.${r}`) })))
const open = computed(() => props.suggestion?.status === 'new' || props.suggestion?.status === 'forwarded')
const time = (iso?: string | null) => (iso ? d(new Date(iso), 'dateTime') : '—')
const role = (r?: string | null) => (r ? t(`widgets.suggestions.role.${r}`) : '—')

const canSubmit = computed(() => {
  if (command.isPending.value || moment.isReplay || !action.value) return false
  if (action.value === 'forward') return !!responsibleId.value.trim()
  return !!text.value.trim()
})

async function submit(): Promise<void> {
  const s = props.suggestion
  if (!s || !action.value) return
  try {
    if (action.value === 'forward') {
      await command.mutateAsync({ kind: 'forward', suggestion: s, responsibleId: responsibleId.value.trim(), responsibleRole: responsibleRole.value ?? undefined, note: text.value.trim() || undefined })
      done.value = 'forward'
    } else {
      await command.mutateAsync({ kind: 'resolve', suggestion: s, resolution: action.value === 'accept' ? 'accepted' : 'rejected', reason: text.value.trim() })
      done.value = 'resolve'
    }
    action.value = null
  } catch {
    // Текст отказа (с кодом) — из command.error.
  }
}

/** Переход на карту процесса: раздел «Процессы» стола, узел — в адресе (FR-63: «ведёт на карту процесса»). */
function openMap(): void {
  const step = props.suggestion?.step_key
  if (step) void router?.push({ name: 'desk', params: { tab: 'processes' }, query: { step } })
}
</script>

<template>
  <RecordDrawer
    :show="!!suggestion"
    :kind-label="t('widgets.suggestions.drawerKind')"
    :number="suggestion?.suggestion_id ?? ''"
    :subtitle="suggestion?.title ?? ''"
    @close="emit('close')"
  >
    <template v-if="suggestion" #status>
      <span class="status" :data-status="suggestion.status">{{ t(`widgets.suggestions.status.${suggestion.status}`) }}</span>
    </template>
    <div v-if="suggestion" class="body" data-testid="suggestion-drawer" :data-suggestion="suggestion.suggestion_id">
      <p class="principle ant-wrap">{{ t('widgets.suggestions.principle') }}</p>
      <SectionPanel variant="subtle" :title="t('widgets.suggestions.statement')">
        <p class="ant-wrap" data-testid="statement">{{ suggestion.statement }}</p>
      </SectionPanel>
      <KeyValueList>
        <KeyValue :label="t('widgets.suggestions.col.kind')" :value="t(`widgets.suggestions.kind.${suggestion.kind}`)" />
        <KeyValue :label="t('widgets.suggestions.estimate')" :value="suggestion.estimate ?? null" />
        <KeyValue :label="t('widgets.suggestions.col.responsible')" :value="suggestion.responsible_id ? `${suggestion.responsible_id} · ${role(suggestion.responsible_role)}` : role(suggestion.responsible_role)" />
        <KeyValue :label="t('widgets.suggestions.step')" :value="suggestion.step_key ?? null" mono />
        <KeyValue :label="t('widgets.suggestions.incident')" :value="suggestion.incident_id ?? null" mono />
      </KeyValueList>
      <button v-if="suggestion.step_key" type="button" class="link" data-testid="map-link" @click="openMap">{{ t('widgets.suggestions.openMap') }}</button>
      <SectionPanel variant="subtle" :title="t('widgets.suggestions.basis')">
        <p v-if="!suggestion.basis.length" class="muted ant-wrap">{{ t('widgets.suggestions.basisNone') }}</p>
        <ul v-else class="list" data-testid="basis">
          <li v-for="b in suggestion.basis" :key="b" class="ant-mono ant-wrap">{{ b }}</li>
        </ul>
      </SectionPanel>
      <SectionPanel variant="subtle" :title="t('widgets.suggestions.historyTitle')">
        <ul class="list" data-testid="history">
          <li v-for="h in suggestion.history" :key="h.event_id" class="ant-wrap">
            {{ time(h.at) }} · {{ t(`widgets.suggestions.history.${h.type}`) }}<template v-if="h.actor"> · {{ h.actor }}</template
            ><template v-if="h.responsible_id"> → {{ h.responsible_id }}</template><template v-if="h.text"> · {{ h.text }}</template>
          </li>
        </ul>
      </SectionPanel>
      <p v-if="!open" class="muted ant-wrap">{{ t('widgets.suggestions.closedHint') }}</p>
      <form v-if="action" class="form" @submit.prevent="submit">
        <template v-if="action === 'forward'">
          <FormField :label="t('widgets.suggestions.responsibleId')" :hint="t('widgets.suggestions.responsibleIdHint')">
            <NInput v-model:value="responsibleId" :size="naiveSizeOf(density)" data-testid="responsible-id" />
          </FormField>
          <FormField :label="t('widgets.suggestions.responsibleRole')">
            <NSelect v-model:value="responsibleRole" :options="roles" :size="naiveSizeOf(density)" />
          </FormField>
          <FormField :label="t('widgets.suggestions.note')">
            <NInput v-model:value="text" type="textarea" :autosize="{ minRows: 2 }" :size="naiveSizeOf(density)" />
          </FormField>
        </template>
        <FormField v-else :label="t('widgets.suggestions.reason')">
          <NInput v-model:value="text" type="textarea" :autosize="{ minRows: 2 }" :size="naiveSizeOf(density)" data-testid="reason" />
        </FormField>
      </form>
      <p v-if="done" class="ok ant-wrap" data-testid="done">{{ t(`widgets.suggestions.done.${done}`) }}</p>
      <NAlert v-if="command.error.value" type="error" :bordered="false" :show-icon="false" data-testid="command-error">
        <span class="ant-wrap">{{ problemText(command.error.value) }}</span>
      </NAlert>
    </div>
    <template v-if="suggestion && open" #actions>
      <template v-if="action">
        <ActionButton :label="t('widgets.suggestions.cancel')" :size="naiveSizeOf(density)" @click="action = null" />
        <ActionButton
          type="primary"
          :label="t(action === 'forward' ? 'widgets.suggestions.confirmForward' : action === 'accept' ? 'widgets.suggestions.confirmAccept' : 'widgets.suggestions.confirmReject')"
          :size="naiveSizeOf(density)"
          :disabled="!canSubmit"
          :loading="command.isPending.value"
          data-testid="submit"
          @click="submit"
        />
      </template>
      <template v-else>
        <ActionButton type="primary" :label="t('widgets.suggestions.forward')" :size="naiveSizeOf(density)" :disabled="moment.isReplay" data-testid="forward" @click="action = 'forward'" />
        <ActionButton :label="t('widgets.suggestions.accept')" :size="naiveSizeOf(density)" :disabled="moment.isReplay" data-testid="accept" @click="action = 'accept'" />
        <ActionButton :label="t('widgets.suggestions.reject')" :size="naiveSizeOf(density)" :disabled="moment.isReplay" data-testid="reject" @click="action = 'reject'" />
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

.muted {
  margin: 0;
  color: var(--ant-text-3);
}

.ok {
  margin: 0;
  color: var(--ant-status-success);
}

.link {
  align-self: flex-start;
  padding: 0;
  border: 0;
  background: none;
  color: var(--ant-accent);
  cursor: pointer;
  font: inherit;
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
