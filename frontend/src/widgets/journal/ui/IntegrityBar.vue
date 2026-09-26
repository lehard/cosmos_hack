<script setup lang="ts">
/**
 * Полоса целостности над журналом (AD-44, AD-46; FR-106, FR-108; кейс Т3):
 * вердикт независимого верификатора словами и тоном, когда проверено и до
 * какой записи, отпечаток подписанного отчёта (сокращённо, полный — в
 * подсказке), голова журнала — сколько записей и последний номер критического
 * действия. Коротко поясняет человеку, что это значит.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { IntegrityStatus, JournalHead } from '@/entities/integrity'
import type { StatusTone } from '@/shared/api/generated/statuses'
import { shortDigest } from '../model/labels'
import ToneTag from './ToneTag.vue'

const props = withDefaults(
  defineProps<{
    /** Вердикт с учётом свежести (effectiveIntegrity); null — ответа ещё нет. */
    status: IntegrityStatus['status'] | null
    checkedAt?: string | null
    reportRef?: string | null
    /** До какой записи проверил последний отчёт; null — неизвестно. */
    checkedUpTo?: number | null
    head?: JournalHead | null
    headError?: boolean
  }>(),
  { checkedAt: null, reportRef: null, checkedUpTo: null, head: null, headError: false },
)
const { t, d, n } = useI18n()

const VERDICT: Record<IntegrityStatus['status'], { key: string; tone: StatusTone }> = {
  ok: { key: 'widgets.journal.integrity.ok', tone: 'success' },
  violated: { key: 'widgets.journal.integrity.violated', tone: 'critical' },
  stale: { key: 'widgets.journal.integrity.stale', tone: 'attention' },
  unknown: { key: 'widgets.journal.integrity.unknown', tone: 'neutral' },
}
const verdict = computed(() => VERDICT[props.status ?? 'unknown'] ?? VERDICT.unknown)
const time = (iso: string) => d(new Date(iso), 'dateTimeSec')
</script>

<template>
  <section class="integrity-bar ant-box" :data-status="status ?? 'unknown'" data-testid="integrity-bar" :aria-label="t('widgets.journal.integrity.title')">
    <div class="line ant-box">
      <ToneTag class="verdict" :label="t(verdict.key)" :tone="verdict.tone" data-testid="verdict" />
      <span v-if="checkedAt" class="meta ant-wrap" data-testid="checked-at">
        {{ t('widgets.journal.integrity.checkedAt', { time: time(checkedAt) }) }}<template v-if="checkedUpTo != null"> · {{ t('widgets.journal.integrity.checkedUpTo', { seq: n(checkedUpTo, 'integer') }) }}</template>
      </span>
      <span v-if="reportRef" class="meta digest ant-box" :title="t('widgets.journal.integrity.reportHint', { ref: reportRef })" data-testid="report-ref">
        {{ t('widgets.journal.integrity.report') }}: <code class="ant-mono ant-ellipsis">{{ shortDigest(reportRef) }}</code>
      </span>
    </div>
    <div class="line ant-box">
      <template v-if="head">
        <strong class="count ant-wrap" data-testid="head-seq">{{ t('widgets.journal.integrity.records', { n: n(head.seq, 'integer') }) }}</strong>
        <strong class="count ant-wrap" data-testid="head-ca">{{
          head.ca_seq > 0 ? t('widgets.journal.integrity.criticalActions', { n: head.ca_seq }) : t('widgets.journal.integrity.noCriticalActions')
        }}</strong>
        <span v-if="head.clock_mode === 'scenario'" class="meta ant-wrap">{{ t('widgets.journal.integrity.scenarioClock') }}</span>
      </template>
      <span v-else-if="headError" class="meta ant-wrap">{{ t('widgets.journal.integrity.headUnavailable') }}</span>
      <span class="meta explain ant-wrap">{{ t('widgets.journal.integrity.explain') }}</span>
    </div>
  </section>
</template>

<style scoped>
.integrity-bar {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  padding: var(--ant-space-2) var(--ant-space-3);
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-md);
  background: var(--ant-surface-subtle);
}

.integrity-bar[data-status='violated'] {
  border-color: var(--ant-status-critical);
}

.integrity-bar[data-status='stale'] {
  border-color: var(--ant-status-attention);
}

.line {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-1) var(--ant-space-4);
  align-items: center;
}

.verdict {
  font-weight: var(--ant-fw-bold);
}

.meta {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.digest {
  display: inline-flex;
  gap: var(--ant-space-1);
  align-items: baseline;
  max-width: 100%;
}

.count {
  font-size: var(--ant-fs-body);
}

.explain {
  flex: 1 1 20rem;
}
</style>
