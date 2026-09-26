<script setup lang="ts">
/**
 * «Установить сдерживание» по изделию (FR-49): уровень — наблюдать, доп.
 * проверка, блок изделия или партии — и основание; подпись уровня 2 касанием
 * токена (AD-13, AD-14), как у решений контролёра. Кнопка есть, только если
 * сервер разрешает `nonconformity.containment.set` над изделием (права — с
 * сервера, AD-15); в воспроизведении — только чтение.
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NInput, NRadio, NRadioGroup } from 'naive-ui'
import { useMutation, useQueryClient } from '@tanstack/vue-query'
import type { SummaryField } from '@/entities/document'
import { newCommandId } from '@/shared/lib/command-id'
import { useSession } from '@/entities/session'
import { useObjectActions } from '@/features/decision-authority'
import { SignDialog, payloadTypeOf, toPayloadB64, useSigningPort } from '@/features/sign-decision'
import { nonconformityContainmentSet } from '@/shared/api/generated/client'
import type { Receipt, SetContainment, SetContainmentLevel } from '@/shared/api/generated/model'
import { useProblemText } from '@/shared/i18n/problem'
import { useMomentStore } from '@/shared/model/moment'
import { ActionButton, FormField } from '@/shared/ui'

const props = defineProps<{
  itemId: string
  /** Номер изделия для людей (сводка подписи). */
  itemLabel: string
  /** seq, на котором показан паспорт (AD-39). */
  basisSeq: number
}>()

const OPERATION = 'nonconformity.containment.set'
const EVENT_TYPE = 'decision.containment.set'
const LEVELS: { value: SetContainmentLevel; key: string }[] = [
  { value: 'observe', key: 'statuses.containment.observe' },
  { value: 'additional_check', key: 'statuses.containment.additionalCheck' },
  { value: 'item_hold', key: 'statuses.containment.itemHold' },
  { value: 'lot_hold', key: 'statuses.containment.batchHold' },
]

const { t } = useI18n()
const problemText = useProblemText()
const moment = useMomentStore()
const session = useSession()
const port = useSigningPort()
const queryClient = useQueryClient()
const { allowed } = useObjectActions('item', () => props.itemId)

const canSet = computed(() => !moment.isReplay && !!allowed.value?.has(OPERATION))
const open = ref(false)
const level = ref<SetContainmentLevel>('item_hold')
const reason = ref('')
const pending = ref<{ commandId: string } | null>(null)
const signing = ref(false)
const dialogError = ref<unknown>(undefined)
const receipt = ref<Receipt | null>(null)

const levelText = computed(() => t(LEVELS.find((l) => l.value === level.value)?.key ?? ''))
const summary = computed<SummaryField[]>(() => [
  { labelKey: 'widgets.signing.fields.action', value: t('containmentAction.title') },
  { labelKey: 'common.words.item', value: props.itemLabel },
  { labelKey: 'containmentAction.level', value: levelText.value },
  { labelKey: 'containmentAction.reason', value: reason.value.trim() },
  { labelKey: 'widgets.signing.fields.level', valueKey: 'decisions.signature.level2' },
])
const demoUnsigned = computed(() => session.data.value?.data?.demo === true)

const command = useMutation({
  mutationFn: (body: SetContainment) => nonconformityContainmentSet(props.itemId, body),
  onSuccess: () => {
    // Изменилось сдерживание — паспорт, очередь и карточки перечитываются.
    void queryClient.invalidateQueries({ queryKey: ['item'] })
    void queryClient.invalidateQueries({ queryKey: ['nonconformity'] })
  },
})

function start(): void {
  receipt.value = null
  dialogError.value = undefined
  open.value = true
}

function sign(): void {
  dialogError.value = undefined
  pending.value = { commandId: newCommandId() }
}

async function submit(withAgent: boolean): Promise<void> {
  const p = pending.value
  const s = session.data.value?.data
  if (!p || !s) return
  signing.value = true
  dialogError.value = undefined
  try {
    const body: SetContainment = {
      level: level.value,
      reason: { text: reason.value.trim() },
      command_id: p.commandId,
      basis_seq: props.basisSeq,
      policy_seq: s.policy_seq,
      ...(s.workplace?.id ? { workplace_id: s.workplace.id } : {}),
    }
    if (withAgent) {
      // Уровень 2: окно расширения показывает сводку и ждёт касания токена (AD-14).
      body.signature = await port.sign({
        level: 2,
        payload_type: payloadTypeOf('event'),
        payload_b64: toPayloadB64(body),
        event_type: EVENT_TYPE,
        command_request: { operation: OPERATION, params: { item_id: props.itemId }, item_id: props.itemId },
      })
    }
    const res = await command.mutateAsync(body)
    receipt.value = res.data as Receipt
    pending.value = null
    open.value = false
    reason.value = ''
  } catch (err) {
    // Отказ в окне подтверждения — не ошибка: форма остаётся.
    if ((err as { info?: { code?: string } } | null)?.info?.code !== 'signing.cancelled') dialogError.value = err
  } finally {
    signing.value = false
  }
}
</script>

<template>
  <div v-if="canSet || receipt" class="containment ant-box" data-testid="containment">
    <ActionButton v-if="canSet && !open" type="warning" data-action="set-containment" :label="t('containmentAction.title')" @click="start" />

    <div v-if="open" class="form ant-box">
      <strong class="title ant-ellipsis">{{ t('containmentAction.title') }}</strong>
      <FormField :label="t('containmentAction.level')">
        <NRadioGroup v-model:value="level" name="containment-level" data-testid="containment-level">
          <NRadio v-for="l in LEVELS" :key="l.value" :value="l.value" :data-level="l.value">
            <span class="ant-ellipsis">{{ t(l.key) }}</span>
          </NRadio>
        </NRadioGroup>
      </FormField>
      <FormField :label="t('containmentAction.reason')" required>
        <NInput v-model:value="reason" type="textarea" :autosize="{ minRows: 2, maxRows: 5 }" :placeholder="t('containmentAction.reasonPlaceholder')" data-testid="containment-reason" />
      </FormField>
      <div class="buttons ant-box">
        <ActionButton type="primary" :disabled="!reason.trim()" data-action="sign-containment" :label="t('containmentAction.sign')" @click="sign" />
        <ActionButton quaternary :label="t('common.actions.cancel')" @click="open = false" />
      </div>
    </div>

    <p v-if="receipt" class="receipt ant-wrap" data-testid="containment-receipt">
      {{ t('containmentAction.done', { level: levelText, seq: receipt.seq }) }}<template v-if="receipt.ca_ref"> · {{ receipt.ca_ref }}</template>
    </p>
    <p v-if="command.error.value && !pending" class="error ant-wrap">{{ problemText(command.error.value) }}</p>

    <SignDialog
      :show="!!pending"
      stage="confirm"
      :summary="summary"
      :token-status="port.status.value"
      :paper-allowed="false"
      :demo-unsigned="demoUnsigned"
      :busy="signing"
      :error="dialogError"
      @confirm-token="submit(true)"
      @confirm-unsigned="submit(false)"
      @close="pending = null"
    />
  </div>
</template>

<style scoped>
.containment {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
}

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
