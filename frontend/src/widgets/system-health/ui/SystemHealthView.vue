<script setup lang="ts">
/**
 * Состояние компонентов — представление (FR-127, AD-45, AD-46; кейс О3/Т7):
 * сервисы и копии, очереди и отставание, интеграции, последний отчёт
 * верификатора, метрики приёма, изделия с остановленной обработкой и
 * «повторить обработку». «Неизвестно» и «ещё не подключён» не выдаются за «работает».
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NAlert } from 'naive-ui'
import type { IngestMetrics, OpsHealth, StoppedItem } from '@/shared/api/generated/model'
import { statusPalette, type StatusTone } from '@/shared/api/generated/statuses'
import type { Density } from '@/shared/config/widget'
import { codeToKey } from '@/shared/i18n'
import { useProblemText } from '@/shared/i18n/problem'
import { ActionButton, DataTable } from '@/shared/ui'

const props = withDefaults(
  defineProps<{
    health: OpsHealth
    metrics?: IngestMetrics | null
    metricsError?: unknown
    stopped?: StoppedItem[] | null
    stoppedError?: unknown
    canAct?: boolean
    busy?: boolean
    error?: unknown
    /** Изделие, для которого повтор принят. */
    retried?: string | null
    density?: Density
  }>(),
  { metrics: null, metricsError: undefined, stopped: null, stoppedError: undefined, canAct: true, busy: false, error: undefined, retried: null, density: 'comfortable' },
)
const emit = defineEmits<{ retry: [item: StoppedItem] }>()
const { t, te, d, n } = useI18n()
const problemText = useProblemText()

const TONE: Record<string, StatusTone> = { ok: 'success', degraded: 'attention', down: 'danger', not_implemented: 'muted', unknown: 'neutral', intact: 'success', intact_with_caveats: 'attention', violated: 'critical' }
const color = (code: string) => statusPalette[TONE[code] ?? 'neutral']
const time = (iso: string | null | undefined) => (iso ? d(new Date(iso), 'dateTime') : '—')
const completeness = computed(() => (props.metrics ? n(props.metrics.completeness_bp / 10000, 'percent') : ''))
</script>

<template>
  <div class="health" :class="`density-${density}`" data-testid="system-health">
    <p class="muted">
      {{ t('widgets.admin.health.version', { version: health.version }) }} · {{ t(`widgets.admin.health.profile.${health.profile}`) }}
    </p>

    <section>
      <h4>{{ t('widgets.admin.health.components') }}</h4>
      <DataTable>
        <thead>
          <tr>
            <th>{{ t('widgets.admin.health.component') }}</th>
            <th>{{ t('widgets.admin.health.state') }}</th>
            <th>{{ t('widgets.admin.health.instances') }}</th>
            <th>{{ t('widgets.admin.health.leader') }}</th>
            <th>{{ t('widgets.admin.health.checkedAt') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="c in health.components" :key="c.component" :data-component="c.component" :data-state="c.state">
            <td>
              <strong>{{ c.component }}</strong>
              <div v-if="c.detail" class="muted">{{ c.detail }}</div>
            </td>
            <td class="state"><span class="dot" :style="{ background: color(c.state) }" aria-hidden="true" />{{ t(`widgets.admin.health.componentState.${codeToKey(c.state)}`) }}</td>
            <td>{{ c.instances }}</td>
            <td>{{ c.leader ?? '—' }}</td>
            <td>{{ time(c.checked_at) }}</td>
          </tr>
        </tbody>
      </DataTable>
    </section>

    <section v-if="health.queues.length">
      <h4>{{ t('admin.queues') }}</h4>
      <DataTable>
        <thead>
          <tr>
            <th>{{ t('widgets.admin.health.queue') }}</th>
            <th>{{ t('widgets.admin.health.pending') }}</th>
            <th>{{ t('widgets.admin.health.lag') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="q in health.queues" :key="`${q.scope}:${q.name}`" :data-queue="q.name">
            <td>{{ q.name }} <span class="muted">{{ t(`widgets.admin.health.scope.${q.scope}`) }}</span></td>
            <td>{{ q.pending }}</td>
            <td>{{ q.lag_seq }}</td>
          </tr>
        </tbody>
      </DataTable>
    </section>

    <section v-if="health.integrations.length">
      <h4>{{ t('widgets.admin.health.integrations') }}</h4>
      <ul class="plain">
        <li v-for="i in health.integrations" :key="i.system" :data-integration="i.system" class="state">
          <span class="dot" :style="{ background: color(i.state) }" aria-hidden="true" />
          <strong>{{ te(`widgets.admin.system.${i.system}`) ? t(`widgets.admin.system.${i.system}`) : i.system }}</strong>
          {{ t(`widgets.admin.health.integrationState.${i.state}`) }}
          <span v-if="i.since" class="muted">{{ t('widgets.admin.health.since', { time: time(i.since) }) }}</span>
          <span v-if="i.detail" class="muted">{{ i.detail }}</span>
        </li>
      </ul>
    </section>

    <section data-testid="verifier">
      <h4>{{ t('audit.verifier.title') }}</h4>
      <p v-if="health.verifier" class="state">
        <span class="dot" :style="{ background: color(health.verifier.verdict) }" aria-hidden="true" />
        {{ t(`widgets.admin.health.verdict.${codeToKey(health.verifier.verdict)}`) }} ·
        {{ t('admin.lastIntegrityCheck', { time: time(health.verifier.checked_at) }) }}
      </p>
      <p v-else class="muted">{{ t('widgets.admin.health.noVerifier') }}</p>
    </section>

    <section data-testid="metrics">
      <h4>{{ t('widgets.admin.health.metrics') }}</h4>
      <p v-if="metricsError" class="error">{{ t('widgets.admin.health.partFailed', { what: t('widgets.admin.health.metrics') }) }}: {{ problemText(metricsError) }}</p>
      <dl v-else-if="metrics" class="facts">
        <dt>{{ t('admin.ingestMetrics.latency') }}</dt>
        <dd>{{ t('widgets.admin.health.p50p95', { p50: metrics.latency_p50_ms, p95: metrics.latency_p95_ms }) }}</dd>
        <dt>{{ t('widgets.admin.health.eventToScreen') }}</dt>
        <dd>{{ metrics.event_to_screen_p95_ms }} мс</dd>
        <dt>{{ t('admin.ingestMetrics.completeness') }}</dt>
        <dd data-testid="completeness">{{ completeness }}</dd>
        <dt>{{ t('admin.ingestMetrics.duplicates') }}</dt>
        <dd>{{ metrics.duplicates }}</dd>
        <dt>{{ t('admin.ingestMetrics.rejects') }}</dt>
        <dd>{{ metrics.rejected }}</dd>
        <dt>{{ t('admin.ingestMetrics.quarantineSize') }}</dt>
        <dd data-testid="quarantine-open">{{ metrics.quarantine_open }}</dd>
      </dl>
      <p v-if="metrics" class="muted">{{ t('widgets.admin.health.window', { from: time(metrics.window_from), to: time(metrics.window_to) }) }}</p>
    </section>

    <section data-testid="stopped">
      <h4>{{ t('widgets.admin.health.stopped') }}</h4>
      <p class="muted">{{ t('widgets.admin.health.stoppedCount', { n: health.stopped_items }) }}</p>
      <p v-if="stoppedError" class="error">{{ t('widgets.admin.health.partFailed', { what: t('widgets.admin.health.stopped') }) }}: {{ problemText(stoppedError) }}</p>
      <p v-else-if="stopped && !stopped.length" class="muted">{{ t('widgets.admin.health.noStopped') }}</p>
      <DataTable v-else-if="stopped">
        <thead>
          <tr>
            <th>{{ t('common.words.item') }}</th>
            <th>{{ t('widgets.admin.health.consumer') }}</th>
            <th>{{ t('widgets.admin.health.error') }}</th>
            <th>{{ t('widgets.admin.health.failedAt') }}</th>
            <th>{{ t('widgets.admin.health.retries') }}</th>
            <th />
          </tr>
        </thead>
        <tbody>
          <tr v-for="s in stopped" :key="s.failure_event_id" :data-item="s.item_id">
            <td>{{ s.item_id }}</td>
            <td>{{ s.consumer }}</td>
            <td class="err">{{ s.error }}</td>
            <td>{{ time(s.failed_at) }}</td>
            <td>{{ s.retries }}</td>
            <td>
              <span v-if="retried === s.item_id" class="ok">{{ t('widgets.admin.health.retried') }}</span>
              <ActionButton v-else size="tiny" :disabled="!canAct || busy" @click="emit('retry', s)" :label="t('widgets.admin.health.retry')" />
            </td>
          </tr>
        </tbody>
      </DataTable>
      <NAlert v-if="error" type="error" :bordered="false" :show-icon="false" data-testid="command-error">{{ problemText(error) }}</NAlert>
    </section>
  </div>
</template>

<style scoped>
.health {
  display: flex;
  flex-direction: column;
  gap: 12px;
  font-size: var(--ant-fs-body);
}

h4 {
  margin: 0 0 4px;
  font-size: 1em;
}

.state {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}

td.state {
  display: table-cell;
}

.dot {
  display: inline-block;
  width: 10px;
  height: 10px;
  margin-right: 6px;
  border-radius: 50%;
}

.plain {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.facts {
  display: grid;
  grid-template-columns: max-content 1fr;
  gap: 2px 12px;
  margin: 0;
}

.facts dt {
  color: var(--ant-text-3);
}

.facts dd {
  margin: 0;
}

.muted {
  margin: 0;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.error,
.err {
  color: var(--ant-status-danger);
}

.ok {
  color: var(--ant-status-success);
}
</style>
