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
import { ActionButton, EmptyState } from '@/shared/ui'

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
const mode = computed(() => {
  const key = `decisions.automationMode.mode${props.request.escalation.automation_mode}`
  return te(key) ? t(key) : `UNKNOWN(${props.request.escalation.automation_mode})`
})
const quorum = (q: string, k: number | undefined, n: number) =>
  q === 'k_of_n' ? t('documents.route.kOfN', { k: k ?? '?', n }) : q === 'all' ? t('widgets.decisionRequest.quorumAll') : t('widgets.decisionRequest.quorumOne')

const declining = ref(false)
const comment = ref('')
</script>

<template>
  <article class="request" :class="`density-${density}`" :data-document="request.document.document_id" data-testid="decision-request">
    <section class="block">
      <h4>{{ t('decisions.decisionCard.whatProposed') }}</h4>
      <p class="lead" data-testid="proposal">{{ request.proposal.summary }}</p>
      <p class="muted">
        {{ t('common.words.item') }} {{ request.proposal.item_label }}
        <template v-if="request.proposal.nc_number"> · {{ t('ncCard.number', { number: request.proposal.nc_number }) }}</template>
        · {{ request.document.template_ref }}
      </p>
      <ActionButton text type="primary" :size="size" data-testid="open-item" :label="t('common.actions.openPassport')" @click="emit('open-item', request.proposal.item_id)" />
      <p v-if="request.proposal.kind === 'concession' || request.proposal.code === 'use_as_is' || request.proposal.code === 'repair'" class="muted" data-testid="concession-note">
        {{ t('decisions.concession.resultStatus') }}
      </p>
    </section>

    <section class="block">
      <h4>{{ t('decisions.decisionCard.whyEscalated') }}</h4>
      <p>{{ request.escalation.reason }}</p>
      <p class="muted">{{ t('decisions.automationMode.title') }}: {{ mode }}</p>
      <p v-if="request.escalation.rule_id" class="muted">{{ t('widgets.ncCard.rule', { ruleId: request.escalation.rule_id }) }}</p>
    </section>

    <section class="block" data-testid="evidence">
      <h4>{{ t('decisions.decisionCard.evidence') }}</h4>
      <EmptyState v-if="!request.evidence.length" compact :title="t('empty.noRecords')" />
      <ul>
        <li v-for="e in request.evidence" :key="e.event_id">
          <span class="muted">{{ time(e.occurred_at) }}</span>
          {{ catalog[e.event_type]?.title ?? `UNKNOWN(${e.event_type})` }}
          <template v-for="m in e.evidence_refs ?? []" :key="m.material_address">
            <span class="material" :data-illustration="m.is_illustration || undefined">
              <template v-if="m.is_illustration">{{ t('empty.illustrationBadge') }}</template>
              <template v-else><code>{{ m.material_address }}</code></template>
            </span>
          </template>
        </li>
      </ul>
    </section>

    <section class="block similar">
      <div data-testid="similar-accepted">
        <h5>{{ t('decisions.decisionCard.similarAccepted') }}</h5>
        <p v-if="!request.similar_accepted.length" class="muted">{{ t('empty.noSimilarCases') }}</p>
        <ul>
          <li v-for="c in request.similar_accepted" :key="c.ref_id">{{ c.number }} — {{ c.summary }}</li>
        </ul>
      </div>
      <div data-testid="similar-rejected">
        <h5>{{ t('decisions.decisionCard.similarRejected') }}</h5>
        <p v-if="!request.similar_rejected.length" class="muted">{{ t('empty.noSimilarCases') }}</p>
        <ul>
          <li v-for="c in request.similar_rejected" :key="c.ref_id">{{ c.number }} — {{ c.summary }}</li>
        </ul>
      </div>
    </section>

    <section class="block" data-testid="route">
      <h4>{{ t('documents.route.title') }} · {{ t('widgets.decisionRequest.progress', { have: progress.have, need: progress.need }) }}</h4>
      <ol class="stages">
        <li
          v-for="s in request.document.route"
          :key="s.stage"
          :data-stage="s.stage"
          :data-mine="s.stage === request.my_stage || undefined"
          :data-done="countedOf(s).length >= s.required || undefined"
        >
          <p>
            <strong>{{ t('documents.route.stage', { n: s.stage }) }}</strong> · {{ s.authority_label }} ·
            {{ quorum(s.quorum, s.k, s.required) }} ·
            {{ t(countedOf(s).length >= s.required ? 'documents.route.signed' : 'documents.route.waiting') }}
            <template v-if="s.external_party === 'customer_representative'"> · {{ t('widgets.decisionRequest.customerRepresentative') }}</template>
          </p>
          <ul>
            <li v-for="sig in s.signatures" :key="sig.event_id" :data-counted="sig.counted || undefined">
              <SignatureMark :signature="sig" />
              <span v-if="sig.previous_version" class="muted">{{ t('documents.route.previousVersionSignatures') }}</span>
            </li>
          </ul>
        </li>
      </ol>
    </section>

    <p v-if="remaining" class="deadline" :data-overdue="deadline?.overdue || undefined" data-testid="remaining">{{ remaining }}</p>
    <p v-if="!myTurn" class="muted" data-testid="not-my-turn">{{ t('widgets.decisionRequest.notYourStage') }}</p>

    <footer class="buttons">
      <ActionButton overflow="wrap" type="primary" :size="size" :disabled="!canAct || busy || !myTurn" :loading="busy" data-testid="sign" @click="emit('sign')" :label="t('decisions.decisionCard.sign', { decision: decisionWord })" />
      <ActionButton overflow="wrap" :size="size" :disabled="!canAct || busy || !myTurn" data-testid="decline-open" @click="declining = !declining" :label="t('decisions.decisionCard.decline')" />
    </footer>
    <div v-if="declining" class="decline" data-testid="decline">
      <label>
        <span>{{ t('decisions.decisionCard.declineCommentLabel') }}</span>
        <NInput v-model:value="comment" type="textarea" :autosize="{ minRows: 2, maxRows: 4 }" :size="size" data-testid="decline-comment" />
      </label>
      <ActionButton overflow="wrap" :size="size" :disabled="!comment.trim() || busy" data-testid="decline-send" @click="emit('decline', comment.trim())" :label="t('common.actions.send')" />
    </div>
  </article>
</template>

<style scoped>
.request {
  display: flex;
  flex-direction: column;
  gap: 10px;
  font-size: var(--ant-fs-md);
}

.density-large {
  font-size: var(--ant-fs-lg);
}

.block h4,
.block h5,
.block p {
  margin: 0 0 4px;
}

.lead {
  font-size: 1.1em;
  font-weight: var(--ant-fw-bold);
}

.muted {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

ul,
ol {
  margin: 0;
  padding-left: 18px;
}

.similar {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.stages li[data-mine] > p {
  color: var(--ant-accent);
}

.material {
  margin-left: 6px;
  font-size: var(--ant-fs-meta);
}

.material[data-illustration] {
  color: var(--ant-status-attention-text);
  font-weight: var(--ant-fw-bold);
}

.deadline {
  margin: 0;
  color: var(--ant-text-3);
}

.deadline[data-overdue] {
  color: var(--ant-status-danger);
  font-weight: var(--ant-fw-bold);
}

.buttons,
.decline {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: flex-end;
}

.decline label {
  display: flex;
  flex: 1 1 280px;
  flex-direction: column;
  gap: 2px;
}
</style>
