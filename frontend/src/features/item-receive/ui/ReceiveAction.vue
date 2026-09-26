<script setup lang="ts">
/**
 * «Принять в цех» (FR-16, FR-137): мастер подтверждает, что изделие физически
 * пришло в цех (на участок) — куда принято и что показал осмотр при приёмке.
 * Команда — `process.movement.receive` (как подтверждение перемещения в изолятор,
 * но с назначением «цех / участок»). Кнопка есть, только если сервер разрешает
 * действие над изделием (AD-15); в воспроизведении — только чтение.
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NRadio, NRadioGroup, NSelect } from 'naive-ui'
import { useOperationCommand } from '@/entities/operation'
import { useLocations, workshopByScope, workshopOf } from '@/entities/reference'
import { useSession } from '@/entities/session'
import { useObjectActions } from '@/features/decision-authority'
import type { ReceiveMovementDestinationKind, ReceiveMovementInspectionOnReceipt } from '@/shared/api/generated/model'
import { codeToKey } from '@/shared/i18n'
import { useProblemText } from '@/shared/i18n/problem'
import { newCommandId } from '@/shared/lib/command-id'
import { useMomentStore } from '@/shared/model/moment'
import { ActionButton, FormField } from '@/shared/ui'

const props = defineProps<{
  itemId: string
  /** seq, на котором показан паспорт (AD-39). */
  basisSeq: number
  /** Куда принимать — по процессу (место шага из задачи); задано — выбора нет. */
  toLocationId?: string | null
  /** Шаг процесса из задачи (например, welding.receive) — приёмка относится к нему. */
  stepKey?: string | null
}>()

const OPERATION = 'process.movement.receive'
const INSPECTION: readonly ReceiveMovementInspectionOnReceipt[] = ['no_damage', 'damage_found', 'not_inspected']
/** Куда можно принять: цех и участки (изолятор — отдельное подтверждение). */
const KINDS = new Set(['workshop', 'station'])

const { t } = useI18n()
const problemText = useProblemText()
const moment = useMomentStore()
const session = useSession()
const locations = useLocations()
const command = useOperationCommand()
const { allowed } = useObjectActions('item', () => props.itemId)

const canReceive = computed(() => !moment.isReplay && !!allowed.value?.has(OPERATION))
const open = ref(false)
const toLocation = ref('')
const inspection = ref<ReceiveMovementInspectionOnReceipt>('no_damage')
/** id команды — один на намерение: повтор после ошибки уходит с тем же id (AD-7). */
const commandId = ref(newCommandId())
const recordedSeq = ref<number | null>(null)

const all = computed(() => locations.data.value?.data ?? [])
/** Цех мастера: по рабочему месту сеанса, иначе по области роли. */
const ownWorkshop = computed(() => {
  const s = session.data.value?.data
  const wp = s?.workplace?.id
  return (wp ? workshopOf(all.value, wp) : null) ?? workshopByScope(all.value, s?.scope)
})
const options = computed(() =>
  all.value
    .filter((l) => KINDS.has(l.kind))
    .map((l) => ({ label: l.name, value: l.location_id, kind: l.kind })),
)
watch(
  () => [props.toLocationId, ownWorkshop.value] as const,
  ([fixed, w]) => {
    if (fixed) toLocation.value = fixed
    else if (!toLocation.value && w) toLocation.value = w.location_id
  },
  { immediate: true },
)
const destinationKind = computed<ReceiveMovementDestinationKind>(() =>
  options.value.find((o) => o.value === toLocation.value)?.kind === 'station' ? 'station' : 'workshop',
)
const placeName = computed(() => options.value.find((o) => o.value === toLocation.value)?.label ?? toLocation.value)
const ready = computed(() => canReceive.value && !command.isPending.value && !!toLocation.value)

function start(): void {
  recordedSeq.value = null
  commandId.value = newCommandId()
  open.value = true
}

async function confirm(): Promise<void> {
  const s = session.data.value?.data
  if (!ready.value || !s) return
  const res = await command.mutateAsync({
    kind: 'receive',
    item_id: props.itemId,
    body: {
      to_location_id: toLocation.value,
      ...(props.stepKey ? { step_key: props.stepKey } : {}),
      destination_kind: destinationKind.value,
      inspection_on_receipt: inspection.value,
      command_id: commandId.value,
      basis_seq: props.basisSeq,
      policy_seq: s.policy_seq,
      ...(s.workplace?.id ? { workplace_id: s.workplace.id } : {}),
    },
  })
  recordedSeq.value = res.data.seq
  open.value = false
}
</script>

<template>
  <div v-if="canReceive || recordedSeq !== null" class="receive ant-box" data-testid="receive">
    <ActionButton v-if="canReceive && !open" type="primary" data-action="receive-item" :label="t('receiveAction.title')" @click="start" />

    <div v-if="open" class="form ant-box">
      <strong class="title ant-ellipsis">{{ t('receiveAction.title') }}</strong>
      <p v-if="toLocationId" class="ant-muted ant-wrap" data-testid="receive-place-fixed">{{ t('receiveAction.place') }}: {{ placeName }}</p>
      <FormField v-else :label="t('receiveAction.place')" required>
        <NSelect v-model:value="toLocation" :options="options" filterable data-testid="receive-place" />
      </FormField>
      <FormField :label="t('widgets.shopFloor.isolator.inspection')">
        <NRadioGroup v-model:value="inspection" data-testid="receive-inspection">
          <NRadio v-for="v in INSPECTION" :key="v" :value="v">
            <span class="ant-ellipsis">{{ t(`widgets.shopFloor.isolator.receipt.${codeToKey(v)}`) }}</span>
          </NRadio>
        </NRadioGroup>
      </FormField>
      <div class="buttons ant-box">
        <ActionButton type="primary" :disabled="!ready" :loading="command.isPending.value" data-action="confirm-receive" :label="t('receiveAction.confirm')" @click="confirm" />
        <ActionButton quaternary :label="t('common.actions.cancel')" @click="open = false" />
      </div>
      <p v-if="command.error.value" class="error ant-wrap">{{ problemText(command.error.value) }}</p>
    </div>

    <p v-if="recordedSeq !== null" class="receipt ant-wrap" data-testid="receive-receipt">
      {{ t('receiveAction.done', { place: placeName, seq: recordedSeq }) }}
    </p>
  </div>
</template>

<style scoped>
.receive,
.form {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-3);
}

.buttons {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-2);
}

.receipt {
  margin: 0;
  color: var(--ant-status-success);
  font-size: var(--ant-fs-meta);
}

.error {
  margin: 0;
  color: var(--ant-status-danger);
  font-size: var(--ant-fs-meta);
}
</style>
