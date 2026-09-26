<script setup lang="ts">
/**
 * Окно решения на точке предъявления и пересмотра (Д-70, Д-79, Д-81; UI-28):
 * рабочий экран решения, а не паспорт изделия. Один каркас:
 * контекст → доказательства → что предлагает система → решение человека →
 * последствия. Пересмотр (строка очереди «пересмотреть решение») — «тогда |
 * сейчас»: что было в основании при подписи и что пришло после, почему это
 * значимо. Предъявление — результаты методов контроля («оценка невозможна» — не
 * «годно»). Действия, их доступность, основание и последствия — только из ответа
 * сервера (`nonconformity.presentation.read`); выбранное — причина, последствия
 * перед подписью, подпись уровня 2. Паспорт — ссылкой «Подробности об изделии».
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { NAlert, NInput } from 'naive-ui'
import { SourceMark, entryText } from '@/entities/item'
import {
  buildPresentationCommand,
  presentationEventType,
  presentationSummary,
  reasonRequired,
  recordTone,
  usePresentation,
  usePresentationCommand,
  uuidv7,
  type NCPresentationAction,
  type Receipt,
} from '@/entities/nonconformity'
import { useSession } from '@/entities/session'
import { useDrillDown } from '@/features/drill-down'
import { SignDialog, payloadTypeOf, toPayloadB64, useSigningPort } from '@/features/sign-decision'
import { useProblemText } from '@/shared/i18n/problem'
import { useMomentStore } from '@/shared/model/moment'
import { ActionButton, RecordDrawer } from '@/shared/ui'
import RegimeBar from '@/widgets/nc-card/ui/RegimeBar.vue'

const props = defineProps<{ id: string; show: boolean }>()
const emit = defineEmits<{ close: [] }>()
const { t, d } = useI18n()
const route = useRoute()
const drill = useDrillDown()
const moment = useMomentStore()
const session = useSession()
const port = useSigningPort()
const problemText = useProblemText()

const runId = computed(() => (typeof route.query.run === 'string' && route.query.run ? route.query.run : undefined))
const query = usePresentation(() => props.id, runId)
const view = computed(() => (query.data.value?.status === 200 ? query.data.value.data : null))
const review = computed(() => view.value?.review ?? null)
const point = computed(() => view.value?.presentation ?? null)
const actions = computed(() => view.value?.actions ?? [])
const time = (x: string) => d(new Date(x), 'dateTime')

const kindLabel = computed(() => (review.value ? t('widgets.presentation.reviewKind') : t('widgets.presentation.kind')))
const subtitle = computed(() => {
  const p = point.value
  if (!p) return ''
  const parts = [p.closing_point_label ?? p.closing_point, p.step_label, t('decisions.gate.presentationNumber', { n: p.presentation_no })]
  return parts.filter(Boolean).join(' · ')
})

// ── выбор действия, причина, подпись ──
const chosen = ref<NCPresentationAction | null>(null)
const reason = ref('')
const pending = ref<{ action: NCPresentationAction; commandId: string } | null>(null)
const signing = ref(false)
const dialogError = ref<unknown>(undefined)
const receipt = ref<{ label: string; receipt: Receipt } | null>(null)
const command = usePresentationCommand()
watch(
  () => props.id,
  () => {
    chosen.value = null
    reason.value = ''
    receipt.value = null
  },
)
const needReason = computed(() => !!chosen.value && reasonRequired(chosen.value))
const canSign = computed(() => !!chosen.value && chosen.value.allowed && !moment.isReplay && !signing.value && (!needReason.value || !!reason.value.trim()))
const summary = computed(() => (pending.value && view.value ? presentationSummary(pending.value.action, view.value, reason.value) : []))
const demoUnsigned = computed(() => session.data.value?.data?.demo === true)

function choose(a: NCPresentationAction): void {
  if (!a.allowed) return
  chosen.value = a
  receipt.value = null
}

async function submit(withAgent: boolean): Promise<void> {
  const p = pending.value
  const v = view.value
  const s = session.data.value?.data
  if (!p || !v || !s) return
  dialogError.value = undefined
  signing.value = true
  try {
    const meta = { command_id: p.commandId, policy_seq: s.policy_seq, ...(s.workplace?.id ? { workplace_id: s.workplace.id } : {}) }
    let req = buildPresentationCommand(p.action, v, meta, reason.value)
    if (withAgent) {
      // Уровень 2: агент показывает доверенную сводку и ждёт касания токена (AD-14).
      const signature = await port.sign({
        level: 2,
        payload_type: payloadTypeOf('event'),
        payload_b64: toPayloadB64(req.body),
        event_type: presentationEventType(p.action),
        command_request: { operation: p.action.operation, params: { item_id: v.item_id }, item_id: v.item_id },
      })
      req = buildPresentationCommand(p.action, v, { ...meta, signature }, reason.value)
    }
    const res = await command.mutateAsync(req)
    receipt.value = { label: p.action.label, receipt: res.data as Receipt }
    pending.value = null
    chosen.value = null
    reason.value = ''
  } catch (err) {
    if ((err as { info?: { code?: string } } | null)?.info?.code !== 'signing.cancelled') dialogError.value = err
  } finally {
    signing.value = false
  }
}

// ── коротко на первом слое (UI-54) ──
const WHY_MAIN = 2
const whyAll = ref(false)
const whyShown = computed(() => (whyAll.value ? (review.value?.why_significant ?? []) : (review.value?.why_significant ?? []).slice(0, WHY_MAIN)))
/**
 * Последствия выбранного действия: бизнес — на виду (до трёх), остальное — под
 * «Технические последствия». Если сервер разделил их сам (technical_consequences) —
 * берём его разделение; иначе первые три — главные (сервер ставит их первыми).
 */
const MAIN_EFFECTS = 3
const techOpen = ref(false)
const effects = computed(() => {
  const a = chosen.value as (NCPresentationAction & { technical_consequences?: string[] }) | null
  if (!a) return { main: [] as string[], technical: [] as string[] }
  if (a.technical_consequences) return { main: a.consequences, technical: a.technical_consequences }
  return { main: a.consequences.slice(0, MAIN_EFFECTS), technical: a.consequences.slice(MAIN_EFFECTS) }
})
watch(chosen, () => (techOpen.value = false))

const OUTCOME_KEY: Record<string, string> = {
  accept: 'widgets.presentation.outcome.accept',
  accept_with_concession: 'widgets.presentation.outcome.acceptWithConcession',
  reject: 'widgets.presentation.outcome.reject',
  insufficient_data: 'widgets.presentation.outcome.insufficientData',
  upheld: 'widgets.presentation.outcome.upheld',
  revoked: 'widgets.presentation.outcome.revoked',
}
const outcomeText = (o: string) => (OUTCOME_KEY[o] ? t(OUTCOME_KEY[o]!) : `UNKNOWN(${o})`)
</script>

<template>
  <RecordDrawer :show="show" :kind-label="kindLabel" :number="view?.item_label ?? id" :subtitle="subtitle" :loading="query.isPending.value && !view" data-record="presentation" @close="emit('close')">
    <template #links>
      <ActionButton text type="primary" size="small" data-testid="open-item" :label="t('widgets.presentation.itemDetails')" @click="drill.open({ entity: 'item', id })" />
    </template>

    <NAlert v-if="query.error.value && !view" type="error" :bordered="false" data-testid="presentation-error">{{ problemText(query.error.value) }}</NAlert>

    <div v-if="view" class="decision" data-testid="presentation">
      <!-- 1. Контекст: пересмотр — «тогда | сейчас»; предъявление — что предъявлено и результаты контроля. -->
      <section v-if="review" class="hero review" data-zone="review">
        <p class="kicker">{{ t('widgets.presentation.reviewKicker') }}</p>
        <div class="compare">
          <div class="side then" data-testid="then">
            <h4>{{ t('widgets.presentation.then') }}</h4>
            <ul class="facts">
              <li v-for="r in review.known_at_decision" :key="r.event_id" class="fact" :data-tone="recordTone(r)">
                <span class="dot" :data-tone="recordTone(r)" aria-hidden="true" />
                <div class="fact-body">
                  <span class="ant-wrap">{{ entryText(r) }}</span>
                  <span class="fact-meta"><SourceMark :record="r" /> {{ time(r.occurred_at) }}</span>
                </div>
              </li>
              <li v-if="!review.known_at_decision.length" class="muted">{{ t('empty.noRecords') }}</li>
            </ul>
            <!-- Прежнее решение — итог колонки «тогда». -->
            <p class="decided ant-wrap" data-testid="previous-decision">
              <span class="muted">{{ t('widgets.presentation.decidedThen') }}:</span> {{ entryText(review.decision) }}
            </p>
            <p class="muted ant-wrap">{{ time(review.decision.occurred_at) }}<template v-if="review.decision.author"> · {{ review.decision.author }}</template></p>
          </div>
          <div class="side now" data-testid="now">
            <h4>{{ t('widgets.presentation.now') }}</h4>
            <ul class="facts">
              <li v-for="r in review.new_facts" :key="r.event_id" class="fact" :data-tone="recordTone(r)">
                <span class="dot" :data-tone="r.reading ? 'danger' : recordTone(r)" aria-hidden="true" />
                <div class="fact-body">
                  <span class="ant-wrap">{{ entryText(r) }}</span>
                  <span class="fact-meta"><SourceMark :record="r" /> {{ time(r.occurred_at) }}</span>
                  <RegimeBar v-if="r.reading" :reading="r.reading" />
                </div>
              </li>
            </ul>
          </div>
        </div>
        <p class="conclusion ant-wrap" data-testid="review-conclusion">{{ t('widgets.presentation.conclusion') }}</p>
        <p class="muted ant-wrap" data-testid="kept">{{ t('widgets.presentation.keptUnchanged') }}</p>
        <template v-if="review.why_significant?.length">
          <ul class="list" data-testid="why-significant">
            <li v-for="w in whyShown" :key="w" class="ant-wrap">{{ w }}</li>
          </ul>
          <button v-if="review.why_significant.length > WHY_MAIN" type="button" class="more" data-testid="toggle-why" @click="whyAll = !whyAll">
            {{ whyAll ? t('widgets.presentation.whyLess') : t('widgets.presentation.whyMore', { n: review.why_significant.length - WHY_MAIN }) }}
          </button>
        </template>
      </section>

      <section v-else class="hero" data-zone="presentation">
        <p class="kicker">{{ t('widgets.presentation.presentedAt', { point: point?.closing_point_label ?? point?.closing_point }) }}</p>
        <p class="lead ant-wrap">{{ point?.step_label ?? point?.step_key }}</p>
        <p class="meta ant-wrap">
          {{ t('decisions.gate.presentationNumber', { n: point?.presentation_no ?? 1 }) }}<template v-if="(point?.presentation_no ?? 1) > 1"> · {{ t('widgets.presentation.repeated') }}</template>
          <template v-if="point?.next_step_label"> · {{ t('widgets.presentation.nextStep', { step: point.next_step_label }) }}</template>
        </p>
      </section>

      <!-- 2. Доказательства: результаты методов контроля (у пересмотра они уже в колонке «тогда»). -->
      <section v-if="!review" class="block" data-zone="methods">
        <h4>{{ t('widgets.presentation.methodResults') }}</h4>
        <ul class="facts" data-testid="method-results">
          <li v-for="r in view.method_results" :key="r.event_id" class="fact" :data-tone="recordTone(r)">
            <span class="dot" :data-tone="recordTone(r)" aria-hidden="true" />
            <div class="fact-body">
              <span class="ant-wrap">{{ entryText(r) }}</span>
              <span class="fact-meta"><SourceMark :record="r" /> {{ time(r.occurred_at) }}</span>
              <span v-if="r.params?.outcome === 'unable_to_assess'" class="unable ant-wrap" data-testid="unable">{{ t('widgets.presentation.unableNotConforming') }}</span>
            </div>
          </li>
          <li v-if="!view.method_results.length" class="muted">{{ t('widgets.presentation.noMethods') }}</li>
        </ul>
      </section>

      <!-- 3. Что предлагает система — отдельно от решения человека. -->
      <section v-if="view.recommendation" class="block suggest" data-zone="recommendation" data-testid="recommendation">
        <h4>{{ t('ncCard.sections.systemSuggests') }}</h4>
        <p class="lead ant-wrap">{{ outcomeText(view.recommendation.outcome) }}</p>
        <ul class="list">
          <li v-for="w in view.recommendation.why" :key="w" class="ant-wrap">{{ w }}</li>
        </ul>
        <p class="muted ant-wrap">{{ t('ncCard.whySystemSuggests.notAVerdict') }}</p>
      </section>
    </div>

    <!-- 4–5. Решение человека и последствия — нижняя панель окна. -->
    <template v-if="view" #actions>
      <div class="panel" data-testid="presentation-actions">
        <p v-if="!actions.length" class="muted ant-wrap" data-testid="no-actions">{{ t('widgets.presentation.noActions') }}</p>
        <div v-else class="options">
          <button
            v-for="a in actions"
            :key="`${a.operation}:${a.resolution ?? a.outcome}`"
            type="button"
            class="option"
            :data-action="a.resolution ?? a.outcome"
            :aria-pressed="chosen === a"
            :disabled="!a.allowed || moment.isReplay || signing"
            @click="choose(a)"
          >
            <span class="option-title ant-wrap">{{ a.label }}</span>
            <span class="option-why ant-wrap" :data-allowed="a.allowed || undefined">{{ a.why_available }}</span>
          </button>
        </div>
        <div v-if="chosen" class="chosen" data-testid="chosen">
          <template v-if="effects.main.length">
            <p class="chosen-title">{{ t('widgets.presentation.afterDecision') }}</p>
            <ul class="list" data-testid="consequences">
              <li v-for="c in effects.main" :key="c" class="ant-wrap">{{ c }}</li>
            </ul>
            <template v-if="effects.technical.length">
              <button type="button" class="more" data-testid="toggle-technical" @click="techOpen = !techOpen">
                {{ techOpen ? t('widgets.presentation.technicalHide') : t('widgets.presentation.technicalShow', { n: effects.technical.length }) }}
              </button>
              <ul v-if="techOpen" class="list muted" data-testid="technical-consequences">
                <li v-for="c in effects.technical" :key="c" class="ant-wrap">{{ c }}</li>
              </ul>
            </template>
          </template>
          <label class="field">
            <span>{{ needReason ? t('widgets.presentation.reasonRequired') : t('widgets.presentation.reasonOptional') }}</span>
            <NInput v-model:value="reason" type="textarea" :autosize="{ minRows: 2, maxRows: 5 }" data-testid="reason" />
          </label>
          <ActionButton type="primary" :disabled="!canSign" :loading="signing" data-testid="sign" :label="t('decisions.signature.signClosingDecision')" @click="chosen && (pending = { action: chosen, commandId: uuidv7() })" />
        </div>
        <section v-if="receipt" class="receipt" data-testid="receipt">
          <p class="receipt-title">{{ t('widgets.decisions.recordedTitle') }}: {{ receipt.label }}</p>
          <p class="muted">{{ t('widgets.decisions.recorded', { seq: receipt.receipt.seq }) }}</p>
        </section>
        <NAlert v-if="command.error.value && !pending" type="error" :bordered="false" data-testid="command-error">{{ problemText(command.error.value) }}</NAlert>
      </div>
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
    </template>
  </RecordDrawer>
</template>

<style scoped>
.decision {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-6);
  min-width: 0;
  line-height: 1.5;
}

p,
h4 {
  margin: 0;
}

h4 {
  font-size: var(--ant-fs-title);
}

.hero,
.block {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
  min-width: 0;
}

.hero.review {
  padding: var(--ant-space-4) var(--ant-space-5);
  border-left: 4px solid var(--ant-status-attention);
  border-radius: 0 var(--ant-radius-lg) var(--ant-radius-lg) 0;
  background: var(--ant-status-attention-soft);
}

.kicker {
  color: var(--ant-text-2);
  font-size: var(--ant-fs-meta);
  font-weight: var(--ant-fw-bold);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.hero.review .kicker {
  color: var(--ant-status-attention-text);
}

.lead {
  font-size: var(--ant-fs-title);
  font-weight: var(--ant-fw-bold);
}

.meta {
  color: var(--ant-text-2);
  font-size: var(--ant-fs-meta);
}

/* «Тогда | сейчас» — две колонки, в узком окне друг под другом. */
.compare {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: var(--ant-space-4);
  margin-top: var(--ant-space-2);
}

.side {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
  min-width: 0;
  padding: var(--ant-space-3);
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-md);
  background: var(--ant-surface);
}

.side.now {
  border-color: var(--ant-status-danger);
}

.facts,
.list {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
  min-width: 0;
  margin: 0;
  padding: 0;
  list-style: none;
}

.list {
  gap: var(--ant-space-1);
  padding-left: var(--ant-space-5);
  list-style: disc;
}

.fact {
  display: flex;
  gap: var(--ant-space-2);
  min-width: 0;
}

.dot {
  flex: none;
  width: 10px;
  height: 10px;
  margin-top: 7px;
  border-radius: 50%;
  background: var(--ant-status-neutral);
}

.dot[data-tone='ok'] {
  background: var(--ant-status-success);
}

.dot[data-tone='warn'] {
  background: var(--ant-status-attention);
}

.dot[data-tone='danger'] {
  background: var(--ant-status-danger);
}

.fact-body {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  flex: 1 1 auto;
}

.fact-meta {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-2);
  align-items: center;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-xs);
}

.unable {
  color: var(--ant-status-attention-text);
  font-weight: var(--ant-fw-bold);
}

.suggest {
  padding: var(--ant-space-3) var(--ant-space-4);
  border-left: 3px solid var(--ant-accent);
  border-radius: 0 var(--ant-radius-md) var(--ant-radius-md) 0;
  background: var(--ant-surface-subtle);
}

.muted {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

/* Нижняя панель: действия карточками — надпись и почему доступно; выбранное — последствия, причина, подпись. */
.panel {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-3);
  width: 100%;
  min-width: 0;
}

.options {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: var(--ant-space-2);
}

.option {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  padding: var(--ant-space-2) var(--ant-space-3);
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-md);
  background: var(--ant-surface);
  color: var(--ant-text);
  font: inherit;
  text-align: start;
  cursor: pointer;
}

.option:hover:not(:disabled) {
  border-color: var(--ant-border-strong);
  background: var(--ant-surface-hover);
}

.option[aria-pressed='true'] {
  border-color: var(--ant-accent);
  background: var(--ant-accent-soft);
  box-shadow: inset 0 0 0 1px var(--ant-accent);
}

.option:disabled {
  cursor: not-allowed;
  opacity: 0.65;
}

.option-title {
  font-weight: var(--ant-fw-bold);
}

.option-why {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-xs);
}

.chosen {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
  padding: var(--ant-space-3);
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-md);
}

.chosen-title {
  font-weight: var(--ant-fw-bold);
}

.field {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
}

.conclusion {
  font-weight: var(--ant-fw-bold);
}

.decided {
  margin-top: var(--ant-space-2);
  padding-top: var(--ant-space-2);
  border-top: 1px solid var(--ant-border);
  font-weight: var(--ant-fw-bold);
}

.more {
  align-self: flex-start;
  padding: 0;
  border: 0;
  background: none;
  color: var(--ant-accent);
  font: inherit;
  font-size: var(--ant-fs-meta);
  cursor: pointer;
}

.receipt {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: var(--ant-space-2) var(--ant-space-3);
  border-left: 3px solid var(--ant-status-success);
  border-radius: 0 var(--ant-radius-md) var(--ant-radius-md) 0;
  background: var(--ant-status-success-soft);
}

.receipt-title {
  color: var(--ant-status-success-text);
  font-weight: var(--ant-fw-bold);
}
</style>
