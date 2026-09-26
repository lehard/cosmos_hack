<script setup lang="ts">
/**
 * «Остановить пост» — стоп точки процесса или критическая остановка на
 * оборудовании поста (FR-49): основание обязательно, условие снятия — по
 * желанию; подпись уровня 2 касанием токена (AD-13, AD-14). Действующие
 * остановки оборудования — с сервера (`nonconformity.station.read`: окно
 * операции участка с `equipment_id`), поэтому «Снять остановку» есть и после
 * перезагрузки и в другой вкладке: у каждой остановки — с какого момента,
 * уровень, основание, условие снятия и кнопка снятия с доступностью сервера
 * (нет полномочия — кнопка выключена, и сказано почему) и последствиями
 * (сколько первых изделий после снятия — на усиленный контроль). Окно
 * участка не отвечает — как раньше: снять можно поставленную здесь.
 * «Перевод поста» — остановка
 * одного источника и допуск исполнителя к другому, отдельной операции нет.
 * Кнопки — только по праву сервера над оборудованием (AD-15); в
 * воспроизведении — только чтение.
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NInput, NInputNumber, NRadio, NRadioGroup } from 'naive-ui'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import type { SummaryField } from '@/entities/document'
import { useJournalHead } from '@/entities/integrity'
import { useSession } from '@/entities/session'
import { useObjectActions } from '@/features/decision-authority'
import { SignDialog, payloadTypeOf, toPayloadB64, useSigningPort } from '@/features/sign-decision'
import { nonconformityProcessHoldRelease, nonconformityProcessHoldSet, nonconformityStationRead } from '@/shared/api/generated/client'
import type { Receipt, ReleaseProcessHold, SetProcessHold, SetProcessHoldLevel, StationAction, StationHold } from '@/shared/api/generated/model'
import { codeToKey } from '@/shared/i18n'
import { useProblemText } from '@/shared/i18n/problem'
import { newCommandId } from '@/shared/lib/command-id'
import { useMomentStore } from '@/shared/model/moment'
import { ActionButton, FormField } from '@/shared/ui'

const props = withDefaults(
  defineProps<{
    equipmentId: string
    /** Название оборудования для людей. */
    equipmentTitle: string
    /**
     * Шаг процесса для окна участка; нет — номер оборудования: остановки
     * оборудования сервер отдаёт по `equipment_id` при любом шаге.
     */
    stepKey?: string
  }>(),
  { stepKey: undefined },
)

const SET = { operation: 'nonconformity.process_hold.set', eventType: 'decision.process_hold.set' }
const RELEASE = { operation: 'nonconformity.process_hold.release', eventType: 'decision.process_hold.released' }
const LEVELS: { value: SetProcessHoldLevel; key: string }[] = [
  { value: 'process_point_stop', key: 'statuses.containment.processPointStop' },
  { value: 'critical_stop', key: 'statuses.containment.criticalStop' },
]

const { t, te, d } = useI18n()
const problemText = useProblemText()
const moment = useMomentStore()
const session = useSession()
const head = useJournalHead()
const port = useSigningPort()
const queryClient = useQueryClient()
const { allowed } = useObjectActions('equipment', () => props.equipmentId)

const canSet = computed(() => !moment.isReplay && !!allowed.value?.has(SET.operation))
/** Остановка, поставленная здесь (номер задаёт клиент), пока окно участка её не показало. */
const holdId = ref<string | null>(null)
const mode = ref<'set' | 'release' | null>(null)
/** Какую остановку снимаем. */
const releaseId = ref<string | null>(null)

// Окно участка (nonconformity.station.read): действующие остановки оборудования и
// действия с доступностью. Ключ начинается с ['equipment'] — команды его сбрасывают.
const stepKey = computed(() => props.stepKey || props.equipmentId)
const stationQ = useQuery({
  queryKey: computed(() => ['equipment', props.equipmentId, 'station-view', stepKey.value, moment.params] as const),
  queryFn: ({ signal }) => nonconformityStationRead(stepKey.value, { equipment_id: props.equipmentId, ...moment.params }, { signal }),
  retry: false,
})
const station = computed(() => stationQ.data.value?.data ?? null)

/** Действующая остановка в окне: с сервера (с действием снятия) или поставленная здесь. */
interface HoldRow {
  holdId: string
  hold?: StationHold
  release?: StationAction
}
const rows = computed<HoldRow[]>(() => {
  const v = station.value
  const out: HoldRow[] = (v?.active_holds ?? [])
    .filter((h) => !h.equipment_id || h.equipment_id === props.equipmentId)
    .map((h) => ({ holdId: h.hold_id, hold: h, release: v?.actions.find((a) => a.operation === RELEASE.operation && a.hold_id === h.hold_id) }))
  if (holdId.value && !out.some((r) => r.holdId === holdId.value)) out.push({ holdId: holdId.value })
  return out
})
const setAction = computed(() => station.value?.actions.find((a) => a.operation === SET.operation) ?? null)
const releasing = computed(() => rows.value.find((r) => r.holdId === releaseId.value) ?? null)
/** Кнопка «Снять» видна: сервер дал действие (доступное или нет) или остановка поставлена здесь. */
const showRelease = (r: HoldRow) => !moment.isReplay && (!!r.release || canSet.value)
const releaseAllowed = (r: HoldRow) => (r.release ? r.release.allowed : canSet.value)
const levelText = (level: string) => {
  const key = `statuses.containment.${codeToKey(level)}`
  return te(key) ? t(key) : level
}
/** Последствия команды — словами сервера. */
const consequences = computed(() => (mode.value === 'release' ? (releasing.value?.release?.consequences ?? []) : (setAction.value?.consequences ?? [])))
const level = ref<SetProcessHoldLevel>('process_point_stop')
const reason = ref('')
const releaseCondition = ref('')
const cleanItems = ref<number>(3)
const pending = ref<{ commandId: string; kind: 'set' | 'release' } | null>(null)
const signing = ref(false)
const dialogError = ref<unknown>(undefined)
const receipt = ref<{ kind: 'set' | 'release'; seq: number } | null>(null)

const chosenLevelText = computed(() => t(LEVELS.find((l) => l.value === level.value)?.key ?? ''))
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
        { labelKey: 'processHoldAction.level', value: chosenLevelText.value },
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

function begin(kind: 'set' | 'release', id: string | null = null): void {
  receipt.value = null
  dialogError.value = undefined
  reason.value = ''
  releaseId.value = id
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
      const id = releaseId.value
      if (!id) return
      const body: ReleaseProcessHold = { ...meta, clean_point_items: cleanItems.value, reason: { text: reason.value.trim() } }
      if (withAgent) body.signature = await signBody(def, body, { hold_id: id })
      const res = await command.mutateAsync({ kind: 'release', id, body })
      if (holdId.value === id) holdId.value = null
      releaseId.value = null
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
  <div v-if="canSet || receipt || rows.length" class="hold ant-box" :data-equipment="equipmentId" data-testid="process-hold">
    <ul v-if="rows.length" class="holds ant-box" data-testid="active-holds">
      <li v-for="r in rows" :key="r.holdId" class="active-hold ant-box" :data-hold="r.holdId">
        <template v-if="r.hold">
          <strong class="ant-wrap">{{ t('processHoldAction.activeSince', { since: d(new Date(r.hold.since), 'dateTime'), level: levelText(r.hold.level) }) }}</strong>
          <span class="ant-wrap">{{ t('processHoldAction.reason') }}: {{ r.hold.reason }}</span>
          <span v-if="r.hold.release_condition" class="muted ant-wrap">{{ t('processHoldAction.releaseCondition') }}: {{ r.hold.release_condition }}</span>
        </template>
        <strong v-else class="ant-wrap">{{ t('processHoldAction.activeLocal', { id: r.holdId }) }}</strong>
        <div v-if="showRelease(r) && !mode" class="buttons ant-box">
          <ActionButton
            type="primary"
            secondary
            data-action="release-process-hold"
            :disabled="!releaseAllowed(r)"
            :hint="r.release?.why_available"
            :label="t('processHoldAction.release')"
            @click="begin('release', r.holdId)"
          />
        </div>
        <span v-if="r.release && !r.release.allowed" class="muted ant-wrap" data-testid="release-why">{{ r.release.why_available }}</span>
      </li>
    </ul>

    <div v-if="canSet && !mode && !rows.length" class="buttons ant-box">
      <ActionButton
        type="error"
        data-action="set-process-hold"
        :disabled="setAction ? !setAction.allowed : false"
        :hint="setAction?.why_available"
        :label="t('processHoldAction.title')"
        @click="begin('set')"
      />
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
      <div v-if="consequences.length" class="consequences ant-box" data-testid="hold-consequences">
        <span class="muted">{{ t('processHoldAction.consequences') }}</span>
        <ul>
          <li v-for="(c, i) in consequences" :key="i" class="ant-wrap">{{ c }}</li>
        </ul>
      </div>
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

.holds {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
  margin: 0;
  padding: 0;
  list-style: none;
}

.active-hold {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  padding: var(--ant-space-2) var(--ant-space-3);
  border-left: 3px solid var(--ant-status-danger);
  border-radius: var(--ant-radius-sm);
  background: var(--ant-status-danger-soft);
}

.consequences ul {
  margin: var(--ant-space-1) 0 0;
  padding-left: var(--ant-space-5);
}

.muted {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
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
