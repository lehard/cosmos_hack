<script setup lang="ts">
/**
 * «Как машина пришла к выводу» — дорожка решения (Ф1 SHOW-IS2; кейс §1.4,
 * §2.2, §4.5, §5.4): где смотрели → качество наблюдения против порога (шкала)
 * → ответ анализатора и уверенность (шкала; уверенность — не вероятность брака)
 * → правило карты реакций (номер, редакция, режим) → уровень доверия
 * анализатора → что система сделала сама и что решает человек. Чего источник
 * не передал — так и сказано. Представление без запросов: данные — `trace`.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { codeToKey } from '@/shared/i18n'
import { confidenceBelow, OUTCOME_TEXT, qualityBelow, scalePercent, type DecisionTrace, type TraceAct, type TraceScale } from '../model/trace'

const props = defineProps<{ trace: DecisionTrace }>()
const { t, te, n, d } = useI18n()

const tr = computed(() => props.trace)
const lowQuality = computed(() => qualityBelow(tr.value.quality))
const lowConfidence = computed(() => confidenceBelow(tr.value.answer.confidence))
const bp = (v: number) => n(v / 10000, 'decimal2')
const outcomeText = (o: string | undefined) => (o && OUTCOME_TEXT[o] ? t(OUTCOME_TEXT[o]) : o)
const severityText = (s: string | undefined) => (s && te(`decisionTrace.severity.${s}`) ? t(`decisionTrace.severity.${s}`) : s)
const thresholdText = (s: TraceScale) => (s.threshold === undefined ? '' : t(`decisionTrace.thresholdFrom.${s.thresholdFrom ?? 'rule'}`, { value: bp(s.threshold) }))
const levelTitle = (l: number) => (te(`decisionTrace.levels.l${l}`) ? t(`decisionTrace.levels.l${l}`) : '')
/** Текст действия; уровень сдерживания — по словарю статусов. */
function actText(a: TraceAct, group: 'acts' | 'human'): string {
  const params = { ...(a.params ?? {}) }
  if (typeof params.level === 'string') {
    const key = `statuses.containment.${codeToKey(params.level)}`
    if (te(key)) params.level = t(key)
  }
  return t(`decisionTrace.${group}.${a.key}`, params)
}
const where = computed(() => [tr.value.frame.point, tr.value.frame.zone].filter(Boolean).join(' · '))
</script>

<template>
  <section class="trace ant-box" data-testid="decision-trace" :aria-label="t('decisionTrace.title')">
    <h4 class="head ant-wrap">{{ t('decisionTrace.title') }}</h4>
    <ol class="lane">
      <li class="step" data-step="where">
        <span class="name">{{ t('decisionTrace.steps.where') }}</span>
        <span class="ant-wrap">{{ where || t('decisionTrace.notSent') }}</span>
        <span v-if="tr.frame.at" class="muted">{{ d(new Date(tr.frame.at), 'dateTime') }}</span>
      </li>

      <li class="step" data-step="quality" :data-below="lowQuality || undefined">
        <span class="name">{{ t('decisionTrace.steps.quality') }}</span>
        <template v-if="tr.quality">
          <div class="scale" role="meter" :aria-valuenow="tr.quality.value" aria-valuemin="0" aria-valuemax="10000" :aria-label="t('decisionTrace.steps.quality')">
            <span class="fill" :style="{ width: `${scalePercent(tr.quality.value)}%` }" />
            <span v-if="tr.quality.threshold !== undefined" class="mark" :style="{ left: `${scalePercent(tr.quality.threshold)}%` }" />
          </div>
          <span class="values ant-wrap">
            <strong data-testid="trace-quality">{{ bp(tr.quality.value) }}</strong>
            <span v-if="tr.quality.threshold !== undefined" class="muted">{{ thresholdText(tr.quality) }}</span>
            <span v-else class="muted">{{ t('decisionTrace.noThreshold') }}</span>
          </span>
          <span v-if="lowQuality" class="verdict warn ant-wrap" data-testid="trace-quality-low">{{ t('decisionTrace.qualityLow') }}</span>
        </template>
        <span v-else class="muted ant-wrap">{{ t('decisionTrace.notSent') }}</span>
      </li>

      <li class="step" data-step="answer">
        <span class="name">{{ t('decisionTrace.steps.answer') }}</span>
        <span class="ant-wrap" data-testid="trace-answer">
          <strong>{{ outcomeText(tr.answer.outcome) || t('decisionTrace.notSent') }}</strong>
          <template v-if="tr.answer.defect"> · {{ tr.answer.defect }}</template>
          <template v-if="tr.answer.severity"> · {{ severityText(tr.answer.severity) }}</template>
        </span>
        <template v-if="tr.answer.confidence">
          <div class="scale" role="meter" :aria-valuenow="tr.answer.confidence.value" aria-valuemin="0" aria-valuemax="10000" :aria-label="t('decisionTrace.confidence')">
            <span class="fill" :style="{ width: `${scalePercent(tr.answer.confidence.value)}%` }" />
            <span v-if="tr.answer.confidence.threshold !== undefined" class="mark" :style="{ left: `${scalePercent(tr.answer.confidence.threshold)}%` }" />
          </div>
          <span class="values ant-wrap">
            <span>{{ t('decisionTrace.confidence') }}: <strong data-testid="trace-confidence">{{ bp(tr.answer.confidence.value) }}</strong></span>
            <span v-if="tr.answer.confidence.threshold !== undefined" class="muted">{{ thresholdText(tr.answer.confidence) }}</span>
            <span v-if="lowConfidence" class="warn">{{ t('decisionTrace.confidenceLow') }}</span>
          </span>
          <span class="muted ant-wrap">{{ t('decisionTrace.confidenceNote') }}</span>
        </template>
        <span v-if="tr.answer.analyzerVersion" class="muted ant-wrap">{{ t('decisionTrace.analyzerVersion', { version: tr.answer.analyzerVersion }) }}</span>
      </li>

      <li class="step" data-step="rule">
        <span class="name">{{ t('decisionTrace.steps.rule') }}</span>
        <span v-if="tr.rule.id" class="ant-wrap" data-testid="trace-rule">
          <strong>{{ tr.rule.id }}</strong>
          <template v-if="tr.rule.title"> — {{ tr.rule.title }}</template>
        </span>
        <span v-else class="muted ant-wrap">{{ t('decisionTrace.ruleNotSent') }}</span>
        <span v-if="tr.rule.rev || tr.rule.automationMode" class="muted ant-wrap">
          <template v-if="tr.rule.rev">{{ t('decisionTrace.ruleRev', { rev: tr.rule.rev }) }}</template>
          <template v-if="tr.rule.rev && tr.rule.automationMode"> · </template>
          <template v-if="tr.rule.automationMode">{{ t('decisionTrace.automationMode', { mode: tr.rule.automationMode }) }}</template>
        </span>
        <span v-if="tr.rule.trigger" class="muted ant-wrap">{{ t('decisionTrace.trigger', { text: tr.rule.trigger }) }}</span>
        <ul v-if="tr.rule.reasons.length" class="reasons">
          <li v-for="(r, i) in tr.rule.reasons" :key="i" class="ant-wrap">{{ r }}</li>
        </ul>
      </li>

      <li class="step" data-step="trust">
        <span class="name">{{ t('decisionTrace.steps.trust') }}</span>
        <span v-if="tr.trust.level !== undefined" class="ant-wrap" data-testid="trace-trust">
          <strong>{{ t('decisionTrace.level', { level: tr.trust.level }) }}</strong> — {{ levelTitle(tr.trust.level) }}
        </span>
        <span v-else class="muted ant-wrap">{{ t('decisionTrace.noPassport') }}</span>
        <span v-if="tr.trust.passportId" class="muted ant-wrap">
          {{ t('decisionTrace.passport', { id: tr.trust.passportId }) }}<template v-if="tr.trust.title"> · {{ tr.trust.title }}</template>
        </span>
        <span v-if="tr.trust.statusThen && tr.trust.statusThen !== 'active'" class="warn ant-wrap">{{ t(`decisionTrace.statusThen.${codeToKey(tr.trust.statusThen)}`) }}</span>
        <span v-if="tr.trust.provenance === 'genesis'" class="muted ant-wrap" data-testid="trace-genesis">{{ t('decisionTrace.genesis') }}</span>
      </li>
    </ol>

    <div class="split">
      <div class="side" data-testid="trace-system">
        <span class="name">{{ t('decisionTrace.systemDid') }}</span>
        <ul>
          <li v-for="(a, i) in tr.system" :key="i" class="ant-wrap">{{ actText(a, 'acts') }}</li>
        </ul>
      </div>
      <div class="side human" data-testid="trace-human">
        <span class="name">{{ t('decisionTrace.humanDecides') }}</span>
        <ul>
          <li v-for="(a, i) in tr.human" :key="i" class="ant-wrap">{{ actText(a, 'human') }}</li>
        </ul>
      </div>
    </div>
  </section>
</template>

<style scoped>
.trace {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-3);
  min-width: 0;
}

.head {
  margin: 0;
  font-size: var(--ant-fs-body);
}

.lane {
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 0;
  list-style: none;
  counter-reset: step;
}

.step {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  min-width: 0;
  padding: 0 0 var(--ant-space-3) var(--ant-space-8);
  counter-increment: step;
}

/* Номер шага и линия дорожки. */
.step::before {
  position: absolute;
  top: 0;
  left: 0;
  display: grid;
  place-items: center;
  width: 22px;
  height: 22px;
  border: 1px solid var(--ant-border-strong);
  border-radius: 50%;
  background: var(--ant-surface);
  color: var(--ant-text-2);
  font-size: var(--ant-fs-xs);
  content: counter(step);
}

.step:not(:last-child)::after {
  position: absolute;
  top: 24px;
  bottom: 2px;
  left: 11px;
  width: 1px;
  background: var(--ant-border);
  content: '';
}

.step[data-below]::before {
  border-color: var(--ant-status-attention);
  background: var(--ant-status-attention-soft);
}

.name {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.scale {
  position: relative;
  max-width: 320px;
  height: 8px;
  border-radius: var(--ant-radius-pill);
  background: var(--ant-n-100);
}

.fill {
  position: absolute;
  inset: 0 auto 0 0;
  border-radius: var(--ant-radius-pill);
  background: var(--ant-status-info);
}

.step[data-below] .fill {
  background: var(--ant-status-attention);
}

.mark {
  position: absolute;
  top: -3px;
  bottom: -3px;
  width: 2px;
  background: var(--ant-text);
}

.values {
  display: flex;
  flex-wrap: wrap;
  gap: 2px var(--ant-space-3);
  align-items: baseline;
}

.muted {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.warn {
  color: var(--ant-status-attention-text);
  font-size: var(--ant-fs-meta);
}

.verdict {
  font-weight: var(--ant-fw-bold);
}

.reasons,
.side ul {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin: 0;
  padding-left: var(--ant-space-5);
  font-size: var(--ant-fs-meta);
}

.split {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: var(--ant-space-3);
}

.side {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  min-width: 0;
  padding: var(--ant-space-2) var(--ant-space-3);
  border-radius: var(--ant-radius-md);
  background: var(--ant-surface-subtle);
}

.side.human {
  box-shadow: inset 3px 0 0 var(--ant-accent);
}
</style>
