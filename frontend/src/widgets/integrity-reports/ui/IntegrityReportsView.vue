<script setup lang="ts">
/**
 * Целостность журнала — представление стола Аудитора ИБ (только чтение;
 * FR-106, FR-108; AD-44, AD-46; кейс Т3, О4): состояние «по данным сервера»,
 * отчёты верификатора (вердикт, до какой записи проверено, отпечаток программы)
 * и выбранный отчёт — проверки по видам, подписи по происхождению, бумажные
 * решения для сверки. «Цело» — только при нуле «не проверяемо».
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { IntegrityStatus, VerifierReport, VerifierReportSummary } from '@/entities/integrity'
import { statusPalette, type StatusTone } from '@/shared/api/generated/statuses'
import type { Density } from '@/shared/config/widget'
import { codeToKey } from '@/shared/i18n'
import { useProblemText } from '@/shared/i18n/problem'

const props = withDefaults(
  defineProps<{
    status?: IntegrityStatus['status'] | null
    checkedAt?: string | null
    reports: VerifierReportSummary[]
    /** Список отчётов не прочитан — это не «отчётов не было». */
    reportsError?: unknown
    selected?: string | null
    report?: VerifierReport | null
    reportError?: unknown
    density?: Density
  }>(),
  { status: null, checkedAt: null, reportsError: undefined, selected: null, report: null, reportError: undefined, density: 'compact' },
)
const emit = defineEmits<{ select: [digest: string] }>()
const { t, d } = useI18n()
const problemText = useProblemText()

const TONE: Record<string, StatusTone> = { ok: 'success', intact: 'success', intact_with_reservations: 'attention', unverifiable: 'attention', stale: 'attention', violated: 'critical', rejected: 'critical', unknown: 'neutral' }
const color = (code: string) => statusPalette[TONE[code] ?? 'neutral']
const time = (iso: string | null | undefined) => (iso ? d(new Date(iso), 'dateTime') : '—')
const statusText = computed(() => {
  switch (props.status) {
    case 'ok':
      return t('audit.integrity.indicatorOk')
    case 'violated':
      return t('audit.integrity.indicatorViolated')
    case 'stale':
      return t('audit.integrity.indicatorStale')
    default:
      return t('common.header.integrityStale')
  }
})
const short = (digest: string) => (digest.length > 16 ? `${digest.slice(0, 16)}…` : digest)
</script>

<template>
  <div class="integrity" :class="`density-${density}`" data-testid="integrity-reports">
    <p class="muted">{{ t('widgets.audit.readOnly') }} · {{ t('audit.protectedFromAdmin') }}</p>
    <p v-if="status" class="headline" :data-status="status" data-testid="status">
      <span class="dot" :style="{ background: color(status) }" aria-hidden="true" />
      <strong>{{ statusText }}</strong>
      <span class="muted">{{ t('widgets.audit.serverSide') }}<template v-if="checkedAt"> · {{ time(checkedAt) }}</template></span>
    </p>

    <h4>{{ t('widgets.audit.reports') }}</h4>
    <p v-if="reportsError" class="error" data-testid="reports-error">{{ problemText(reportsError) }}</p>
    <p v-else-if="!reports.length" class="muted">{{ t('widgets.admin.health.noVerifier') }}</p>
    <ol class="rows">
      <li
        v-for="r in reports"
        :key="r.report_digest"
        class="row"
        tabindex="0"
        :data-digest="r.report_digest"
        :data-verdict="r.verdict"
        :aria-selected="r.report_digest === selected"
        @click="emit('select', r.report_digest)"
        @keydown.enter.prevent="emit('select', r.report_digest)"
      >
        <span class="dot" :style="{ background: color(r.verdict) }" aria-hidden="true" />
        <strong>{{ t(`widgets.admin.health.verdict.${codeToKey(r.verdict)}`) }}</strong>
        <span>{{ time(r.checked_at) }}</span>
        <span class="muted">{{ t('widgets.audit.checkedUpTo', { seq: r.checked_up_to_seq }) }}</span>
        <code class="muted" :title="r.report_digest">{{ short(r.report_digest) }}</code>
        <span v-if="r.verifier_build" class="muted">{{ t('audit.verifier.binaryHash') }}: {{ short(r.verifier_build) }}</span>
      </li>
    </ol>

    <section v-if="selected" class="card" data-testid="report">
      <p v-if="reportError" class="error">{{ problemText(reportError) }}</p>
      <template v-else-if="report">
        <h4>{{ t('audit.verifier.report') }}</h4>
        <p class="muted">{{ t('audit.verifier.reportSigned') }} · {{ t('widgets.audit.signedBy', { key: report.signed_by }) }}</p>
        <p v-if="report.virtual_time" class="muted">{{ t('widgets.audit.virtualTime') }}</p>
        <table>
          <thead>
            <tr>
              <th>{{ t('widgets.audit.check') }}</th>
              <th>{{ t('common.words.status') }}</th>
              <th>{{ t('widgets.audit.count') }}</th>
              <th>{{ t('audit.verifier.place') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(c, i) in report.checks" :key="`${c.check}:${i}`" :data-check="c.check" :data-status="c.status">
              <td>{{ c.check }}</td>
              <td><span class="dot" :style="{ background: color(c.status) }" aria-hidden="true" />{{ t(`widgets.audit.checkStatus.${c.status}`) }}</td>
              <td>{{ c.count }}</td>
              <td>
                <span v-if="c.ca_ref">{{ c.ca_ref }}</span>
                <span v-if="c.details" class="muted">{{ c.details }}</span>
              </td>
            </tr>
          </tbody>
        </table>
        <h5>{{ t('widgets.audit.signatureClasses') }}</h5>
        <p class="muted">
          <span v-for="(n, cls) in report.signature_classes" :key="cls" class="cls">{{ cls }}: {{ n }}</span>
        </p>
        <p class="muted">{{ t('widgets.audit.paperDecisions', { n: report.paper_decisions.length }) }}</p>
      </template>
    </section>
    <p v-else-if="reports.length" class="muted">{{ t('widgets.audit.selectReport') }}</p>
  </div>
</template>

<style scoped>
.integrity {
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: 13px;
}

h4,
h5 {
  margin: 4px 0;
}

.headline {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  margin: 0;
}

.rows {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.row {
  display: flex;
  flex-wrap: wrap;
  gap: 2px 10px;
  align-items: center;
  padding: 4px 8px;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  cursor: pointer;
}

.row[aria-selected='true'] {
  border-color: #2f6fdb;
  background: #eff6ff;
}

.card {
  padding: 8px;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
}

table {
  width: 100%;
  border-collapse: collapse;
}

th,
td {
  padding: 3px 6px;
  border-bottom: 1px solid #e5e7eb;
  text-align: left;
  vertical-align: top;
}

th {
  color: #6b7280;
  font-weight: 400;
}

.dot {
  display: inline-block;
  width: 10px;
  height: 10px;
  margin-right: 6px;
  border-radius: 50%;
}

.cls {
  margin-right: 12px;
}

.muted {
  margin: 0;
  color: #6b7280;
  font-size: 12px;
}

.error {
  color: #d64545;
}
</style>
