<script setup lang="ts">
/**
 * Карточка «Требуется ваше решение» для редких подписантов — представителя
 * заказчика и согласующего (FR-136, PRD §3a): что предлагается; почему пришло к
 * вам; доказательства; похожие случаи — принятые и отклонённые; чьи подписи
 * нужны и чьи уже есть; остаток срока; кнопка подписи. Без их подписей решения
 * режимов 4–5 (ремонт, «как есть», разрешение на отклонение) не закрываются.
 * Подписывается документ с маршрутом (AD-13); засчитывает подписи модуль
 * documents — здесь только показ (AD-43).
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NInput } from 'naive-ui'
import { countedOf, routeProgress, type DecisionRequest } from '@/entities/document'
import { SignatureMark } from '@/entities/item'
import { DISPOSITION_STATUS_TEXT, deadlineOf } from '@/entities/nonconformity'
import { naiveSizeOf, type Density } from '@/shared/config/widget'
import { eventCatalog } from '@/shared/contracts/catalog'
import { formatMinutes } from '@/shared/lib/duration'
import { ActionButton } from '@/shared/ui'

const props = withDefaults(
  defineProps<{
    request: DecisionRequest
    /** «Сейчас» для остатка срока, мс. */
    now: number
    canAct?: boolean
    busy?: boolean
    density?: Density
  }>(),
  { canAct: true, busy: false, density: 'comfortable' },
)
const emit = defineEmits<{
  /** Подписать — открыть окно подписи уровня 2. */
  sign: []
  /** Не согласовать — с замечанием. */
  decline: [comment: string]
  /** Открыть паспорт изделия (Д-70: правым окном). */
  'open-item': [itemId: string]
}>()

const { t, te, d } = useI18n()
const size = computed(() => naiveSizeOf(props.density))
const time = (x: string) => d(new Date(x), 'dateTime')
const catalog = eventCatalog as Record<string, { title: string } | undefined>

const progress = computed(() => routeProgress(props.request.document.route))
const myTurn = computed(() => props.request.my_stage != null)
const deadline = computed(() => deadlineOf(props.request.due_at, props.now))
const remaining = computed(() => {
  const dl = deadline.value
  if (!dl) return null
  const x = formatMinutes(t, dl.minutes)
  return dl.overdue ? t('ncCard.deadline.overdueBy', { time: x }) : t('common.words.remaining', { time: x })
})

/** Что подписывается — словом статуса для решения по изделию, иначе сводкой сервера. */
const decisionWord = computed(() => {
  const p = props.request.proposal
  if (p.kind === 'disposition') {
    const key = (DISPOSITION_STATUS_TEXT as Record<string, string>)[p.code]
    return key ? t(key) : `UNKNOWN(${p.code})`
  }
  if (p.kind === 'concession') return t('decisions.concession.title')
  return p.summary
})
/** Режим автоматизации словом; кода без текста людям не показываем (UI-35). */
const mode = computed(() => {
  const key = `decisions.automationMode.mode${props.request.escalation.automation_mode}`
  return te(key) ? t(key) : null
})
/** Этап, который ждёт вас, и следующий за ним — «что будет после вашей подписи». */
const route = computed(() => [...props.request.document.route].sort((a, b) => a.stage - b.stage))
const mine = computed(() => route.value.find((s) => s.stage === props.request.my_stage) ?? null)
const next = computed(() => (mine.value ? (route.value.find((s) => s.stage > mine.value!.stage && countedOf(s).length < s.required) ?? null) : null))
/** Подпись: для решения по изделию — словом решения, иначе коротко «Подписать» (заголовок документа — в карточке). */
const signLabel = computed(() => {
  const k = props.request.proposal.kind
  return k === 'disposition' || k === 'concession' ? t('decisions.decisionCard.sign', { decision: decisionWord.value }) : t('widgets.decisionRequest.signShort')
})
const hasSimilar = computed(() => props.request.similar_accepted.length + props.request.similar_rejected.length > 0)
const detailsOpen = ref(false)
const quorum = (q: string, k: number | undefined, n: number) =>
  q === 'k_of_n' ? t('documents.route.kOfN', { k: k ?? '?', n }) : q === 'all' ? t('widgets.decisionRequest.quorumAll') : t('widgets.decisionRequest.quorumOne')

const declining = ref(false)
const comment = ref('')
</script>

<template>
  <article class="request" :class="`density-${density}`" :data-document="request.document.document_id" data-testid="decision-request">
    <!-- Что от вас нужно — главное, крупно, с действием рядом (UI-35, Д-79). -->
    <section class="task" :data-mine="myTurn || undefined" data-testid="task">
      <p class="kicker">{{ myTurn ? t('widgets.decisionRequest.needFromYou') : t('widgets.decisionRequest.notYourStageTitle') }}</p>
      <p v-if="mine" class="task-title ant-wrap" data-testid="my-stage">
        {{ t('widgets.decisionRequest.yourStage', { n: mine.stage, who: mine.authority_label }) }}
      </p>
      <h3 class="lead ant-wrap" data-testid="proposal">{{ request.proposal.summary }}</h3>
      <p class="object ant-wrap">
        <span>{{ request.proposal.item_label }}</span>
        <template v-if="request.proposal.nc_number"><span>{{ t('ncCard.number', { number: request.proposal.nc_number }) }}</span></template>
        <ActionButton text type="primary" :size="size" data-testid="open-item" :label="t('common.actions.openPassport')" @click="emit('open-item', request.proposal.item_id)" />
      </p>
      <p v-if="request.proposal.kind === 'concession' || request.proposal.code === 'use_as_is' || request.proposal.code === 'repair'" class="note ant-wrap" data-testid="concession-note">
        {{ t('decisions.concession.resultStatus') }}
      </p>
      <p class="after ant-wrap" data-testid="after-sign">
        <template v-if="next">{{ t('widgets.decisionRequest.afterYou', { n: next.stage, who: next.authority_label }) }}</template>
        <template v-else-if="mine">{{ t('widgets.decisionRequest.youAreLast') }}</template>
      </p>
      <p v-if="remaining" class="deadline" :data-overdue="deadline?.overdue || undefined" data-testid="remaining">{{ remaining }}</p>
      <p v-if="!myTurn" class="muted" data-testid="not-my-turn">{{ t('widgets.decisionRequest.notYourStage') }}</p>
      <footer class="buttons">
        <ActionButton overflow="wrap" type="primary" :size="size" :disabled="!canAct || busy || !myTurn" :loading="busy" data-testid="sign" @click="emit('sign')" :label="signLabel" />
        <ActionButton overflow="wrap" :size="size" :disabled="!canAct || busy || !myTurn" data-testid="decline-open" @click="declining = !declining" :label="t('decisions.decisionCard.decline')" />
      </footer>
      <div v-if="declining" class="decline" data-testid="decline">
        <label>
          <span>{{ t('decisions.decisionCard.declineCommentLabel') }}</span>
          <NInput v-model:value="comment" type="textarea" :autosize="{ minRows: 2, maxRows: 4 }" :size="size" data-testid="decline-comment" />
        </label>
        <ActionButton overflow="wrap" :size="size" :disabled="!comment.trim() || busy" data-testid="decline-send" @click="emit('decline', comment.trim())" :label="t('common.actions.send')" />
      </div>
    </section>

    <section class="block">
      <h4>{{ t('decisions.decisionCard.whyEscalated') }}</h4>
      <p class="ant-wrap">{{ request.escalation.reason }}</p>
    </section>

    <section class="block" data-testid="route">
      <h4>{{ t('documents.route.title') }} · {{ t('widgets.decisionRequest.progress', { have: progress.have, need: progress.need }) }}</h4>
      <ol class="stages">
        <li
          v-for="s in route"
          :key="s.stage"
          class="stage"
          :data-stage="s.stage"
          :data-mine="s.stage === request.my_stage || undefined"
          :data-done="countedOf(s).length >= s.required || undefined"
        >
          <span class="stage-no" aria-hidden="true">{{ s.stage }}</span>
          <div class="stage-body">
            <p class="ant-wrap">
              <strong>{{ s.authority_label }}</strong>
              <span class="muted"> · {{ t('documents.route.stage', { n: s.stage }) }} · {{ quorum(s.quorum, s.k, s.required) }}</span>
            </p>
            <p class="stage-state">
              {{ t(countedOf(s).length >= s.required ? 'documents.route.signed' : 'documents.route.waiting') }}
              <template v-if="s.stage === request.my_stage"> · {{ t('widgets.decisionRequest.yourTurn') }}</template>
              <template v-if="s.external_party === 'customer_representative'"> · {{ t('widgets.decisionRequest.customerRepresentative') }}</template>
            </p>
            <ul v-if="s.signatures.length" class="sigs">
              <li v-for="sig in s.signatures" :key="sig.event_id" :data-counted="sig.counted || undefined">
                <SignatureMark :signature="sig" />
                <span v-if="sig.previous_version" class="muted">{{ t('documents.route.previousVersionSignatures') }}</span>
              </li>
            </ul>
          </div>
        </li>
      </ol>
    </section>

    <section class="block" data-testid="evidence">
      <h4>{{ t('decisions.decisionCard.evidence') }}</h4>
      <p v-if="!request.evidence.length" class="muted">{{ t('widgets.decisionRequest.noMaterials') }}</p>
      <ul v-else class="list">
        <li v-for="e in request.evidence" :key="e.event_id">
          <span class="muted">{{ time(e.occurred_at) }}</span>
          {{ catalog[e.event_type]?.title ?? `UNKNOWN(${e.event_type})` }}
          <template v-for="m in e.evidence_refs ?? []" :key="m.material_address">
            <span class="material" :data-illustration="m.is_illustration || undefined">
              <template v-if="m.is_illustration">{{ t('empty.illustrationBadge') }}</template>
              <template v-else>{{ t('widgets.decisionRequest.materialAttached') }}</template>
            </span>
          </template>
        </li>
      </ul>
    </section>

    <section class="block">
      <h4>{{ t('widgets.decisionRequest.similarTitle') }}</h4>
      <p v-if="!hasSimilar" class="muted" data-testid="no-similar">{{ t('empty.noSimilarCases') }}</p>
      <div v-else class="similar">
        <div data-testid="similar-accepted">
          <h5>{{ t('decisions.decisionCard.similarAccepted') }}</h5>
          <p v-if="!request.similar_accepted.length" class="muted">{{ t('empty.noSimilarCases') }}</p>
          <ul class="list">
            <li v-for="c in request.similar_accepted" :key="c.ref_id">{{ c.number }} — {{ c.summary }}</li>
          </ul>
        </div>
        <div data-testid="similar-rejected">
          <h5>{{ t('decisions.decisionCard.similarRejected') }}</h5>
          <p v-if="!request.similar_rejected.length" class="muted">{{ t('empty.noSimilarCases') }}</p>
          <ul class="list">
            <li v-for="c in request.similar_rejected" :key="c.ref_id">{{ c.number }} — {{ c.summary }}</li>
          </ul>
        </div>
      </div>
    </section>

    <section class="block details">
      <ActionButton size="small" quaternary data-testid="toggle-details" :label="detailsOpen ? t('ncCard.details.hide') : t('widgets.decisionRequest.details')" @click="detailsOpen = !detailsOpen" />
      <dl v-if="detailsOpen" class="details-body" data-testid="details">
        <template v-if="mode"><dt>{{ t('decisions.automationMode.title') }}</dt><dd>{{ mode }}</dd></template>
        <template v-if="request.escalation.rule_id"><dt>{{ t('widgets.decisionRequest.ruleLabel') }}</dt><dd>{{ request.escalation.rule_id }}<template v-if="request.escalation.rule_rev">, {{ request.escalation.rule_rev }}</template></dd></template>
        <dt>{{ t('widgets.decisionRequest.template') }}</dt><dd>{{ request.document.template_ref }} · {{ t('widgets.decisionRequest.version', { n: request.document.version }) }}</dd>
      </dl>
    </section>
  </article>
</template>

<style scoped>
/* Воздух и читаемая ширина строки (UI-35): карточка не растягивается на весь экран. */
.request {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-6);
  max-width: 920px;
  padding: var(--ant-space-2) 0;
  font-size: var(--ant-fs-md);
  line-height: 1.5;
}

.density-large {
  font-size: var(--ant-fs-lg);
}

p,
h3,
h4,
h5 {
  margin: 0;
}

/* Что от вас нужно — выделенный блок с действием. */
.task {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
  padding: var(--ant-space-5) var(--ant-space-6);
  border: 1px solid var(--ant-border);
  border-left: 4px solid var(--ant-border-strong);
  border-radius: var(--ant-radius-lg);
  background: var(--ant-surface-subtle);
}

.task[data-mine] {
  border-left-color: var(--ant-accent);
  background: var(--ant-accent-soft);
}

.kicker {
  color: var(--ant-text-2);
  font-size: var(--ant-fs-meta);
  font-weight: var(--ant-fw-bold);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.task-title {
  font-size: var(--ant-fs-title);
  font-weight: var(--ant-fw-bold);
}

.lead {
  font-size: var(--ant-fs-title);
  font-weight: normal;
  line-height: 1.35;
}

.object {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-1) var(--ant-space-4);
  align-items: center;
  color: var(--ant-text-2);
}

.note {
  color: var(--ant-status-attention-text);
}

.after {
  color: var(--ant-text-2);
}

.deadline {
  color: var(--ant-text-2);
  font-weight: var(--ant-fw-bold);
}

.deadline[data-overdue] {
  color: var(--ant-status-danger);
}

.buttons,
.decline {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-3);
  align-items: flex-end;
  margin-top: var(--ant-space-2);
}

.decline label {
  display: flex;
  flex: 1 1 280px;
  flex-direction: column;
  gap: var(--ant-space-1);
}

.block {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
}

.block h4 {
  font-size: var(--ant-fs-title);
}

.muted {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.list {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  margin: 0;
  padding-left: var(--ant-space-5);
}

/* Маршрут — шагами: номер в кружке, кто подписывает, состояние, подписи. */
.stages {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-3);
  margin: 0;
  padding: 0;
  list-style: none;
}

.stage {
  display: flex;
  gap: var(--ant-space-3);
  min-width: 0;
}

.stage-no {
  display: inline-flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: 2px solid var(--ant-border-strong);
  border-radius: 50%;
  color: var(--ant-text-2);
  font-weight: var(--ant-fw-bold);
}

.stage[data-done] .stage-no {
  border-color: var(--ant-status-success);
  color: var(--ant-status-success);
}

.stage[data-mine] .stage-no {
  border-color: var(--ant-accent);
  background: var(--ant-accent);
  color: var(--ant-surface);
}

.stage-body {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.stage-state {
  color: var(--ant-text-2);
  font-size: var(--ant-fs-meta);
}

.stage[data-mine] .stage-state {
  color: var(--ant-accent);
  font-weight: var(--ant-fw-bold);
}

.sigs {
  margin: var(--ant-space-1) 0 0;
  padding: 0;
  list-style: none;
}

.similar {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: var(--ant-space-4);
}

.material {
  margin-left: var(--ant-space-2);
  font-size: var(--ant-fs-meta);
}

.material[data-illustration] {
  color: var(--ant-status-attention-text);
  font-weight: var(--ant-fw-bold);
}

.details {
  align-items: flex-start;
}

.details-body {
  display: grid;
  grid-template-columns: max-content minmax(0, 1fr);
  gap: var(--ant-space-1) var(--ant-space-4);
  margin: 0;
  color: var(--ant-text-2);
  font-size: var(--ant-fs-meta);
}

.details-body dd {
  margin: 0;
  overflow-wrap: anywhere;
}
</style>
