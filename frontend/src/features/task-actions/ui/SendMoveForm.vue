<script setup lang="ts">
/**
 * «Отправить» прямо в задаче мастера (FR-16, FR-137): после приёмки на
 * закрывающей точке (ЗТ-3 → «Отправить Ф-001: …») изделие уходит в следующий
 * цех — `process.movement.send` (POST /items/{id}/send). Откуда — цех задачи
 * (`location_id`), шаг — `step_key` задачи; куда — по процессу: шаг отправки
 * называет получателя (`…send_to_assembly` → сборочный цех), мастера не спрашиваем.
 * Выбор цеха остаётся только если из шага получатель не выводится. Задачу снимает
 * сервер: токен процесса ушёл с шага. Кнопка есть, только если сервер разрешает действие
 * над изделием (AD-15); в воспроизведении — только чтение.
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NAlert, NSelect } from 'naive-ui'
import { useOperationCommand } from '@/entities/operation'
import { useLocations } from '@/entities/reference'
import { useSession } from '@/entities/session'
import { useObjectActions } from '@/features/decision-authority'
import { naiveSizeOf, type Density } from '@/shared/config/widget'
import { useProblemText } from '@/shared/i18n/problem'
import { newCommandId } from '@/shared/lib/command-id'
import { useMomentStore } from '@/shared/model/moment'
import { ActionButton, FormField } from '@/shared/ui'

const props = withDefaults(
  defineProps<{
    itemId: string
    /** seq, на котором прочитан список задач (AD-39). */
    basisSeq?: number
    /** Откуда — цех шага из задачи. */
    fromLocationId?: string | null
    /** Шаг процесса из задачи (например, welding.send_to_assembly). */
    stepKey?: string | null
    density?: Density
  }>(),
  { basisSeq: 0, fromLocationId: null, stepKey: null, density: 'comfortable' },
)

const OPERATION = 'process.movement.send'

const { t } = useI18n()
const problemText = useProblemText()
const moment = useMomentStore()
const session = useSession()
const locations = useLocations()
const command = useOperationCommand()
const { allowed } = useObjectActions('item', () => props.itemId)
const size = computed(() => naiveSizeOf(props.density))

const canSend = computed(() => !moment.isReplay && !!props.fromLocationId && !!allowed.value?.has(OPERATION))
/** Получатель по шагу процесса: `…send_to_‹участок›` → цех дорожки этого участка. */
const TARGET_BY_SEGMENT: Record<string, string> = { assembly: 'WS-AC', welding: 'WS-WC', machining: 'WS-MC', final: 'WS-QA', qa: 'WS-QA', warehouse: 'WS-SK', incoming: 'WS-SK' }
const targetFromStep = computed(() => {
  const m = /send_to_([a-z]+)/.exec(props.stepKey ?? '')
  return m ? (TARGET_BY_SEGMENT[m[1]!] ?? null) : null
})
const to = ref('')
watch(targetFromStep, (v) => { if (v) to.value = v }, { immediate: true })
/** id команды — один на намерение: повтор после ошибки уходит с тем же id (AD-7). */
const commandId = ref(newCommandId())
const recordedSeq = ref<number | null>(null)

/** Цеха справочника, кроме цеха задачи. */
const workshops = computed(() => (locations.data.value?.data ?? []).filter((l) => l.kind === 'workshop'))
const options = computed(() => workshops.value.filter((l) => l.location_id !== props.fromLocationId).map((l) => ({ label: l.name, value: l.location_id })))
const fromName = computed(() => workshops.value.find((l) => l.location_id === props.fromLocationId)?.name ?? props.fromLocationId ?? '')
const toName = computed(() => options.value.find((o) => o.value === to.value)?.label ?? to.value)
const ready = computed(() => canSend.value && !command.isPending.value && !!to.value)

async function confirm(): Promise<void> {
  const s = session.data.value?.data
  if (!ready.value || !s || !props.fromLocationId) return
  const res = await command
    .mutateAsync({
      kind: 'send',
      item_id: props.itemId,
      body: {
        from_location_id: props.fromLocationId,
        to_location_id: to.value,
        ...(props.stepKey ? { step_key: props.stepKey } : {}),
        command_id: commandId.value,
        basis_seq: props.basisSeq,
        policy_seq: s.policy_seq,
        ...(s.workplace?.id ? { workplace_id: s.workplace.id } : {}),
      },
    })
    .catch(() => null)
  if (res) recordedSeq.value = res.data.seq
}
</script>

<template>
  <div v-if="canSend || recordedSeq !== null" class="send" data-testid="task-send">
    <template v-if="recordedSeq === null">
      <p class="ant-muted ant-wrap" data-testid="send-from">{{ t('sendAction.from') }}: {{ fromName }}</p>
      <p v-if="targetFromStep" class="ant-muted ant-wrap" data-testid="send-to-fixed">{{ t('sendAction.to') }}: {{ toName }}</p>
      <FormField v-else :label="t('sendAction.to')" required>
        <NSelect v-model:value="to" :size="size" :options="options" filterable data-testid="send-to" />
      </FormField>
      <NAlert v-if="command.error.value" type="error" :bordered="false">{{ problemText(command.error.value) }}</NAlert>
      <ActionButton
        :size="size"
        type="primary"
        :disabled="!ready"
        :loading="command.isPending.value"
        :label="t('sendAction.confirm')"
        data-action="confirm-send"
        @click="confirm"
      />
    </template>
    <p v-else class="receipt ant-wrap" data-testid="send-receipt">{{ t('sendAction.done', { place: toName, seq: recordedSeq }) }}</p>
  </div>
</template>

<style scoped>
.send {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
}

.receipt {
  margin: 0;
  color: var(--ant-status-success);
  font-size: var(--ant-fs-meta);
}
</style>
