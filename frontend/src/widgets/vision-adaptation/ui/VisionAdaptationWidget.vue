<script setup lang="ts">
/**
 * Страница «Адаптация VisionQC» (эпик 40; FR-98…FR-101, AD-29) — контейнер:
 * списки анализаторов с паспортами допуска, пропусков брака и размеченных
 * примеров; щелчок по строке открывает запись в правом окне, кнопки действий —
 * в его нижней панели (Д-70). Открытое окно — в адресе `?open=‹вид›:‹id›`.
 * Прогон сценария — `?run=` или срез стола `run_id` (AD-38): автооткат в
 * прогоне виден только в нём.
 */
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAnalyzers, useLabeledExamples, useVisionEscapes } from '@/entities/analyzer-passport'
import { backendModeOf } from '@/shared/api/response'
import type { WidgetProps } from '@/shared/config/widget'
import { DataTable, EmptyState, SectionPanel, WidgetFrame } from '@/shared/ui'
import AdaptationDrawer, { type OpenRef } from './AdaptationDrawer.vue'
import TrustStatus from './TrustStatus.vue'

const props = defineProps<WidgetProps>()
const { t, d } = useI18n()
const route = useRoute()
const router = useRouter()

const str = (v: unknown): string | undefined => (typeof v === 'string' && v ? v : undefined)
const runId = computed(() => str(route?.query.run) ?? str(props.slice.run_id))

const analyzersQ = useAnalyzers(runId)
const escapesQ = useVisionEscapes(runId)
const examplesQ = useLabeledExamples(runId)

const analyzers = computed(() => analyzersQ.data.value?.data?.items ?? null)
const escapes = computed(() => escapesQ.data.value?.data?.items ?? [])
const examples = computed(() => examplesQ.data.value?.data?.items ?? [])

/** Открытая запись: из адреса `?open=passport:AP-…` (ссылки из уведомлений открывают то же окно). */
const open = computed<OpenRef | null>(() => {
  const raw = str(route?.query.open)
  if (!raw) return null
  const [kind, ...rest] = raw.split(':')
  const id = rest.join(':')
  return (kind === 'passport' || kind === 'escape' || kind === 'observation') && id ? { kind, id } : null
})

function openRecord(ref: OpenRef | null): void {
  const query = { ...route?.query }
  if (ref) query.open = `${ref.kind}:${ref.id}`
  else delete query.open
  void router?.replace({ query })
}

const time = (iso: string) => d(new Date(iso), 'dateTime')
const share = (bp?: number | null) => (bp === undefined || bp === null ? '—' : (bp / 10000).toFixed(2).replace('.', ','))
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="backendModeOf(analyzersQ.data.value)"
    :loading="analyzersQ.isPending.value && !analyzers"
    :error="analyzers ? undefined : analyzersQ.error.value"
    :data-widget="widgetId"
  >
    <div v-if="analyzers" class="adaptation" data-testid="vision-adaptation">
      <SectionPanel :title="t('widgets.visionAdaptation.passports.title')" :subtitle="t('widgets.visionAdaptation.passports.subtitle')">
        <EmptyState v-if="!analyzers.length" compact :title="t('widgets.visionAdaptation.passports.empty')" />
        <DataTable v-else :caption="t('widgets.visionAdaptation.passports.title')">
          <thead>
            <tr>
              <th scope="col">{{ t('widgets.visionAdaptation.col.analyzer') }}</th>
              <th scope="col">{{ t('widgets.visionAdaptation.col.recipe') }}</th>
              <th scope="col">{{ t('widgets.visionAdaptation.col.version') }}</th>
              <th scope="col">{{ t('widgets.visionAdaptation.col.status') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="a in analyzers"
              :key="a.analyzer_id"
              class="row"
              tabindex="0"
              :data-passport="a.passport_id"
              :data-status="a.status"
              :aria-selected="open?.kind === 'passport' && open.id === a.passport_id"
              @click="a.passport_id && openRecord({ kind: 'passport', id: a.passport_id })"
              @keydown.enter.prevent="a.passport_id && openRecord({ kind: 'passport', id: a.passport_id })"
            >
              <td class="ant-box"><span class="ant-wrap">{{ a.title }}</span></td>
              <td class="ant-box ant-mono"><span class="ant-wrap">{{ a.versions?.recipe_ref ?? '—' }}</span></td>
              <td class="ant-box ant-mono"><span class="ant-wrap">{{ a.versions?.analyzer_version ?? '—' }}</span></td>
              <td class="ant-box"><TrustStatus :status="a.status" :stage="a.stage" :level="a.trust_level" /></td>
            </tr>
          </tbody>
        </DataTable>
      </SectionPanel>

      <SectionPanel :title="t('widgets.visionAdaptation.escapes.title')" :subtitle="t('widgets.visionAdaptation.escapes.subtitle')">
        <EmptyState v-if="!escapes.length" compact :title="t('widgets.visionAdaptation.escapes.empty')" />
        <DataTable v-else :caption="t('widgets.visionAdaptation.escapes.title')">
          <thead>
            <tr>
              <th scope="col">{{ t('widgets.visionAdaptation.col.item') }}</th>
              <th scope="col">{{ t('widgets.visionAdaptation.col.version') }}</th>
              <th scope="col">{{ t('widgets.visionAdaptation.col.missed') }}</th>
              <th scope="col">{{ t('widgets.visionAdaptation.col.recheck') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="e in escapes"
              :key="e.event_id"
              class="row"
              tabindex="0"
              :data-escape="e.event_id"
              :aria-selected="open?.kind === 'escape' && open.id === e.event_id"
              @click="openRecord({ kind: 'escape', id: e.event_id })"
              @keydown.enter.prevent="openRecord({ kind: 'escape', id: e.event_id })"
            >
              <td class="ant-box ant-mono"><span class="ant-wrap">{{ e.item_id }}</span></td>
              <td class="ant-box ant-mono"><span class="ant-wrap">{{ e.analyzer_version ?? '—' }}</span></td>
              <td class="ant-box is-num">{{ e.missed.length }}</td>
              <td class="ant-box">
                <span v-if="e.method_covers_defect" class="ant-wrap">{{ t('widgets.visionAdaptation.escapes.recheckCount', { n: e.recheck.length }) }}</span>
                <span v-else class="muted ant-wrap">{{ t('widgets.visionAdaptation.escapes.methodBlind') }}</span>
              </td>
            </tr>
          </tbody>
        </DataTable>
      </SectionPanel>

      <SectionPanel :title="t('widgets.visionAdaptation.examples.title')" :subtitle="t('widgets.visionAdaptation.examples.subtitle')">
        <EmptyState v-if="!examples.length" compact :title="t('widgets.visionAdaptation.examples.empty')" />
        <DataTable v-else :caption="t('widgets.visionAdaptation.examples.title')">
          <thead>
            <tr>
              <th scope="col">{{ t('widgets.visionAdaptation.col.item') }}</th>
              <th scope="col">{{ t('widgets.visionAdaptation.col.analyzerAnswer') }}</th>
              <th scope="col">{{ t('widgets.visionAdaptation.col.expertAnswer') }}</th>
              <th scope="col">{{ t('widgets.visionAdaptation.col.decidedAt') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="x in examples"
              :key="x.decision_event_id + x.observation_event_id"
              class="row"
              tabindex="0"
              :aria-selected="open?.kind === 'observation' && open.id === x.observation_event_id"
              @click="openRecord({ kind: 'observation', id: x.observation_event_id })"
              @keydown.enter.prevent="openRecord({ kind: 'observation', id: x.observation_event_id })"
            >
              <td class="ant-box ant-mono"><span class="ant-wrap">{{ x.item_id ?? '—' }}</span></td>
              <td class="ant-box">
                <span class="ant-wrap">{{ t(`widgets.visionAdaptation.outcome.${x.analyzer_outcome}`) }}<template v-if="x.analyzer_defect"> · {{ x.analyzer_defect }}</template> · {{ share(x.analyzer_confidence_bp) }}</span>
              </td>
              <td class="ant-box"><span class="ant-wrap">{{ t(`widgets.visionAdaptation.verdict.${x.verdict}`) }}<template v-if="x.expert_defect"> · {{ x.expert_defect }}</template></span></td>
              <td class="ant-box"><span class="ant-ellipsis">{{ time(x.decided_at) }}</span></td>
            </tr>
          </tbody>
        </DataTable>
      </SectionPanel>
    </div>
    <AdaptationDrawer :open="open" :run-id="runId" :escapes="escapes" :density="density" @close="openRecord(null)" @open="openRecord" />
  </WidgetFrame>
</template>

<style scoped>
.adaptation {
  display: flex;
  flex-direction: column;
  gap: var(--ant-gap);
  font-size: var(--ant-fs-body);
}

.row {
  cursor: pointer;
}

.row:hover {
  background: var(--ant-surface-hover);
}

.row[aria-selected='true'] {
  background: var(--ant-accent-soft);
}

.row[data-status='suspended'] td:first-child {
  box-shadow: inset 3px 0 0 var(--ant-status-danger);
}

.is-num {
  text-align: right;
}

.muted {
  color: var(--ant-text-3);
}
</style>
