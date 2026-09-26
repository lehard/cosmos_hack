<script setup lang="ts">
/**
 * «Остановить пост» — стоп точки процесса или критическая остановка на
 * оборудовании поста (FR-49): основание обязательно, условие снятия — по
 * желанию; подпись уровня 2 касанием токена (AD-13, AD-14). Остановку,
 * поставленную здесь, можно снять здесь же («Снять остановку»: сколько первых
 * изделий после снятия — на усиленный контроль). «Перевод поста» — остановка
 * одного источника и допуск исполнителя к другому, отдельной операции нет.
 * Кнопки — только по праву сервера над оборудованием (AD-15); в
 * воспроизведении — только чтение.
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NInput, NInputNumber, NRadio, NRadioGroup } from 'naive-ui'
import { useMutation, useQueryClient } from '@tanstack/vue-query'
import type { SummaryField } from '@/entities/document'
import { useJournalHead } from '@/entities/integrity'
import { useSession } from '@/entities/session'
import { useObjectActions } from '@/features/decision-authority'
import { SignDialog, payloadTypeOf, toPayloadB64, useSigningPort } from '@/features/sign-decision'
import { nonconformityProcessHoldRelease, nonconformityProcessHoldSet } from '@/shared/api/generated/client'
import type { Receipt, ReleaseProcessHold, SetProcessHold, SetProcessHoldLevel } from '@/shared/api/generated/model'
import { useProblemText } from '@/shared/i18n/problem'
import { newCommandId } from '@/shared/lib/command-id'
import { useMomentStore } from '@/shared/model/moment'
import { ActionButton, FormField } from '@/shared/ui'

const props = defineProps<{
  equipmentId: string
  /** Название оборудования для людей. */
  equipmentTitle: string
}>()

const SET = { operation: 'nonconformity.process_hold.set', eventType: 'decision.process_hold.set' }
const RELEASE = { operation: 'nonconformity.process_hold.release', eventType: 'decision.process_hold.released' }
const LEVELS: { value: SetProcessHoldLevel; key: string }[] = [
  { value: 'process_point_stop', key: 'statuses.containment.processPointStop' },
  { value: 'critical_stop', key: 'statuses.containment.criticalStop' },
]

const { t } = useI18n()
const problemText = useProblemText()
const moment = useMomentStore()
const session = useSession()
const head = useJournalHead()
const port = useSigningPort()
const queryClient = useQueryClient()
const { allowed } = useObjectActions('equipment', () => props.equipmentId)

const canSet = computed(() => !moment.isReplay && !!allowed.value?.has(SET.operation))
/** Остановка, поставленная здесь (номер задаёт клиент — её же и снимаем). */
const holdId = ref<string | null>(null)
const mode = ref<'set' | 'release' | null>(null)
const level = ref<SetProcessHoldLevel>('process_point_stop')
const reason = ref('')
const releaseCondition = ref('')
const cleanItems = ref<number>(3)
const pending = ref<{ commandId: string; kind: 'set' | 'release' } | null>(null)
const signing = ref(false)
const dialogError = ref<unknown>(undefined)
const receipt = ref<{ kind: 'set' | 'release'; seq: number } | null>(null)

const levelText = computed(() => t(LEVELS.find((l) => l.value === level.value)?.key ?? ''))
const summary = computed<SummaryField[]>(() =>
  pending.value?.kind === 'release'
    ? [
        { labelKey: 'widgets.signing.fields.action', value: t('processHoldAction.release') },
        { labelKey: 'processHoldAction.equipment', value: props.equipmentTitle },
        { labelKey: 'processHoldAction.cleanItems', value: String(cleanItems.value) },
        { labelKey: 'processHoldAction.reason', value: reason.value.trim() },
        { labelKey: 'widgets.signing.fields.level', valueKey: 'decisions.signature.level2' },
      ]
    : [
        { labelKey: 'widgets.signing.fields.action', value: t('processHoldAction.title') },
        { labelKey: 'processHoldAction.equipment', value: props.equipmentTitle },
        { labelKey: 'processHoldAction.level', value: levelText.value },
        { labelKey: 'processHoldAction.reason', value: reason.value.trim() },
        ...(releaseCondition.value.trim() ? [{ labelKey: 'processHoldAction.releaseCondition', value: releaseCondition.value.trim() }] : []),
        { labelKey: 'widgets.signing.fields.level', valueKey: 'decisions.signature.level2' },
      ],
)
const demoUnsigned = computed(() => session.data.value?.data?.demo === true)

const command = useMutation({
  mutationFn: (c: { kind: 'set'; body: SetProcessHold } | { kind: 'release'; id: string; body: ReleaseProcessHold }) =>
    c.kind === 'set' ? nonconformityProcessHoldSet(c.body) : nonconformityProcessHoldRelease(c.id, c.body),
  onSuccess: () => {
    for (const key of [['equipment'], ['workplace'], ['live_map'], ['item'], ['nonconformity']]) void queryClient.invalidateQueries({ queryKey: key })
  },
})

function begin(kind: 'set' | 'release'): void {
  receipt.value = null
  dialogError.value = undefined
  reason.value = ''
  mode.value = kind
}

function sign(): void {
  if (!mode.value) return
  dialogError.value = undefined
  pending.value = { commandId: newCommandId(), kind: mode.value }
}

async function submit(withAgent: boolean): Promise<void> {
  const p = pending.value
  const s = session.data.value?.data
  if (!p || !s) return
  signing.value = true
  dialogError.value = undefined
  try {
    // Что клиент видел — последняя запись журнала (AD-39): после неё новых записей гарда нет.
    const meta = {
      command_id: p.commandId,
      basis_seq: head.data.value?.data.seq ?? 0,
      policy_seq: s.policy_seq,
      ...(s.workplace?.id ? { workplace_id: s.workplace.id } : {}),
    }
    const def = p.kind === 'set' ? SET : RELEASE
    if (p.kind === 'set') {
      const id = holdId.value ?? `HOLD-${props.equipmentId}-${p.commandId.slice(0, 8)}`
      const body: SetProcessHold = {
        ...meta,
        hold_id: id,
        equipment_id: props.equipmentId,
        level: level.value,
        reason: { text: reason.value.trim() },
        ...(releaseCondition.value.trim() ? { release_condition: releaseCondition.value.trim() } : {}),
      }
      if (withAgent) body.signature = await signBody(def, body, {})
      const res = await command.mutateAsync({ kind: 'set', body })
      holdId.value = id
      receipt.value = { kind: 'set', seq: (res.data as Receipt).seq }
    } else {
      const id = holdId.value
      if (!id) return
      const body: ReleaseProcessHold = { ...meta, clean_point_items: cleanItems.value, reason: { text: reason.value.trim() } }
      if (withAgent) body.signature = await signBody(def, body, { hold_id: id })
      const res = await command.mutateAsync({ kind: 'release', id, body })
      holdId.value = null
      receipt.value = { kind: 'release', seq: (res.data as Receipt).seq }
    }
    pending.value = null
    mode.value = null
  } catch (err) {
    if ((err as { info?: { code?: string } } | null)?.info?.code !== 'signing.cancelled') dialogError.value = err
  } finally {
    signing.value = false
  }
}

/** Уровень 2: окно расширения показывает сводку и ждёт касания токена (AD-14). */
function signBody(def: { operation: string; eventType: string }, body: object, params: Record<string, string>) {
  return port.sign({
    level: 2,
    payload_type: payloadTypeOf('event'),
    payload_b64: toPayloadB64(body),
    event_type: def.eventType,
    command_request: { operation: def.operation, params },
  })
}
</script>

<template>
  <div v-if="canSet || receipt" class="hold ant-box" :data-equipment="equipmentId" data-testid="process-hold">
    <div v-if="canSet && !mode" class="buttons ant-box">
      <ActionButton v-if="!holdId" type="error" data-action="set-process-hold" :label="t('processHoldAction.title')" @click="begin('set')" />
      <ActionButton v-else type="primary" secondary data-action="release-process-hold" :label="t('processHoldAction.release')" @click="begin('release')" />
    </div>

    <div v-if="mode" class="form ant-box">
      <strong class="ant-ellipsis">{{ mode === 'set' ? t('processHoldAction.title') : t('processHoldAction.release') }} — {{ equipmentTitle }}</strong>
      <FormField v-if="mode === 'set'" :label="t('processHoldAction.level')">
        <NRadioGroup v-model:value="level" name="hold-level">
          <NRadio v-for="l in LEVELS" :key="l.value" :value="l.value">
            <span class="ant-ellipsis">{{ t(l.key) }}</span>
          </NRadio>
        </NRadioGroup>
      </FormField>
      <FormField v-if="mode === 'release'" :label="t('processHoldAction.cleanItems')">
        <NInputNumber v-model:value="cleanItems" :min="0" :max="100" />
      </FormField>
      <FormField :label="t('processHoldAction.reason')" required>
        <NInput v-model:value="reason" type="textarea" :autosize="{ minRows: 2, maxRows: 5 }" :placeholder="t('processHoldAction.reasonPlaceholder')" data-testid="hold-reason" />
      </FormField>
      <FormField v-if="mode === 'set'" :label="t('processHoldAction.releaseCondition')">
        <NInput v-model:value="releaseCondition" :placeholder="t('processHoldAction.releaseConditionPlaceholder')" />
      </FormField>
      <div class="buttons ant-box">
        <ActionButton type="primary" :disabled="!reason.trim()" data-action="sign-process-hold" :label="t('processHoldAction.sign')" @click="sign" />
        <ActionButton quaternary :label="t('common.actions.cancel')" @click="mode = null" />
      </div>
    </div>

    <p v-if="receipt" class="receipt ant-wrap" data-testid="process-hold-receipt">
      {{ receipt.kind === 'set' ? t('processHoldAction.done', { seq: receipt.seq }) : t('processHoldAction.released', { seq: receipt.seq }) }}
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
.hold,
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
