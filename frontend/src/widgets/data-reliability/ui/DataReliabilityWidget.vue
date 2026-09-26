<script setup lang="ts">
/**
 * Виджет «Надёжность данных» — первый раздел стола администратора: общий статус
 * одной строкой, три контура (источники, интеграции, VisionQC) строками и «Требует
 * внимания» — каждая строка ведёт в свой раздел стола, запись — справа (Д-70).
 * «Нет данных» не превращается в зелёный: пропуски, карантин, недоступность и
 * приостановка видны прямо. Минимализм: строки, воздух, одна кнопка-глагол.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAnalyzers, useVisionEscapes } from '@/entities/analyzer-passport'
import { useIntegrations } from '@/entities/integration'
import { useSources } from '@/entities/quarantine'
import { backendModeOf } from '@/shared/api/response'
import type { WidgetProps } from '@/shared/config/widget'
import { WidgetFrame } from '@/shared/ui'
import { areaSummary, integrationIssues, overallTone, sourceIssues, visionIssues, type Attention } from '../model/reliability'

const props = defineProps<WidgetProps>()
const { t, te } = useI18n()
const router = useRouter()

const sourcesQ = useSources()
const integrationsQ = useIntegrations()
const analyzersQ = useAnalyzers()
const escapesQ = useVisionEscapes()

const sources = computed(() => sourcesQ.data.value?.data?.items ?? [])
const integrations = computed(() => (integrationsQ.data.value?.data?.items ?? []).filter((i) => i.state !== 'disabled'))
const analyzers = computed(() => analyzersQ.data.value?.data?.items ?? [])
const escapes = computed(() => escapesQ.data.value?.data?.items ?? [])
const systemName = (s: string) => (te(`widgets.integrations.systems.${s}`) ? t(`widgets.integrations.systems.${s}`) : s)

const issues = computed<Attention[]>(() => [
  ...sourceIssues(sources.value),
  ...integrationIssues(integrations.value, systemName),
  ...visionIssues(analyzers.value, escapes.value),
])
const areas = computed(() => [
  areaSummary('sources', sources.value.filter((s) => s.state !== 'disabled').length, issues.value),
  areaSummary('integrations', integrations.value.length, issues.value),
  areaSummary('vision', analyzers.value.length, issues.value),
])
const overall = computed(() => overallTone(issues.value))
const AREA_TAB = { sources: 'sources', integrations: 'integrations', vision: 'vision-adaptation' } as const

function go(tab: string, open?: string): void {
  void router.push({ name: 'desk', params: { tab }, query: open ? { open } : {} })
}
const loading = computed(() => sourcesQ.isPending.value && integrationsQ.isPending.value && analyzersQ.isPending.value)
const allFailed = computed(() => !!sourcesQ.error.value && !!integrationsQ.error.value && !!analyzersQ.error.value)
</script>

<template>
  <WidgetFrame
    :title-key="props.titleKey"
    :density="props.density"
    :mode="backendModeOf(sourcesQ.data.value ?? integrationsQ.data.value)"
    :state="allFailed ? 'input_error' : 'normal'"
    :loading="loading"
    :data-widget="props.widgetId"
  >
    <div class="reliability" data-testid="data-reliability">
      <p class="overall" :data-tone="overall" data-testid="overall">{{ t(`dataReliability.overall.${overall}`) }}</p>

      <ul class="areas">
        <li v-for="a in areas" :key="a.area" class="area" :data-area="a.area" :data-tone="a.tone">
          <span class="dot" aria-hidden="true" />
          <button type="button" class="area-name" @click="go(AREA_TAB[a.area])">{{ t(`dataReliability.area.${a.area}`) }}</button>
          <span class="area-count" data-testid="area-count">{{ a.total ? t('dataReliability.okOf', { ok: a.ok, total: a.total }) : t('dataReliability.none') }}</span>
        </li>
      </ul>

      <section v-if="issues.length" class="attention" data-testid="attention">
        <h3 class="heading">{{ t('dataReliability.attention') }} · {{ issues.length }}</h3>
        <ul class="lines">
          <li v-for="x in issues" :key="x.key" class="line" :data-tone="x.tone" :data-area="x.area">
            <span class="dot" aria-hidden="true" />
            <span class="text">{{ t(x.text, x.params) }}</span>
            <button type="button" class="go" data-testid="attention-open" @click="go(x.tab, x.open)">{{ t('dataReliability.open') }} →</button>
          </li>
        </ul>
      </section>
      <p v-else class="calm" data-testid="calm">{{ t('dataReliability.calm') }}</p>
      <p class="principle">{{ t('dataReliability.principle') }}</p>
    </div>
  </WidgetFrame>
</template>

<style scoped>
.reliability {
  display: flex;
  flex-direction: column;
  gap: calc(var(--ant-space-3) * 3);
  max-width: 920px;
  min-width: 0;
}

p,
h3 {
  margin: 0;
}

.overall {
  font-size: var(--ant-fs-title);
  font-weight: var(--ant-fw-bold);
}

.overall[data-tone='danger'] {
  color: var(--ant-status-danger-text);
}

.overall[data-tone='warn'] {
  color: var(--ant-status-attention-text);
}

.overall[data-tone='ok'] {
  color: var(--ant-status-success-text);
}

.areas,
.lines {
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 0;
  list-style: none;
}

.area,
.line {
  display: flex;
  gap: var(--ant-space-3);
  align-items: baseline;
  min-width: 0;
  padding: var(--ant-space-3) 0;
  border-bottom: 1px solid var(--ant-border);
}

.dot {
  flex: none;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--ant-status-success);
}

[data-tone='warn'] > .dot {
  background: var(--ant-status-attention);
}

[data-tone='danger'] > .dot {
  background: var(--ant-status-danger);
}

.area-name,
.go {
  padding: 0;
  border: 0;
  background: none;
  font: inherit;
  cursor: pointer;
}

.area-name {
  flex: 1 1 auto;
  color: var(--ant-text);
  font-weight: var(--ant-fw-bold);
  text-align: start;
}

.area-count {
  color: var(--ant-text-2);
  font-variant-numeric: tabular-nums;
}

.heading {
  color: var(--ant-text-2);
  font-size: var(--ant-fs-meta);
  font-weight: var(--ant-fw-bold);
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

.text {
  flex: 1 1 auto;
  min-width: 0;
  overflow-wrap: anywhere;
}

.go {
  flex: none;
  color: var(--ant-accent);
}

.calm {
  color: var(--ant-status-success-text);
}

.principle {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}
</style>
