<script setup lang="ts">
/**
 * «Почему система это предлагает» (FR-50, FR-51, PRD §3a) и версии вывода
 * (FR-32): правило и режим автоматизации, основания — записи журнала и
 * пояснения сервера, альтернативные объяснения, чего не хватает. Это
 * предложение системы, а не решение. Пересмотр — новая версия с причиной;
 * прежняя остаётся в журнале и видна здесь.
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { OUTCOME_TEXT, codeText, conclusionVersions, type NCConclusionVersion, type NCSystemAnalysis } from '@/entities/nonconformity'
import { codeToKey } from '@/shared/i18n'
import RecordLine from './RecordLine.vue'
import { ActionButton } from '@/shared/ui'

const props = defineProps<{ analysis: NCSystemAnalysis }>()
const { t, te, d } = useI18n()

const versions = computed(() => conclusionVersions(props.analysis.versions))
const showPrevious = ref(false)
const time = (x: string) => d(new Date(x), 'dateTime')
const mode = (v: NCConclusionVersion) => (te(`decisions.automationMode.mode${v.automation_mode}`) ? t(`decisions.automationMode.mode${v.automation_mode}`) : `UNKNOWN(${v.automation_mode})`)
/** Нехватка сведений: код из перечня разбора — своим текстом, иначе как прислал сервер. */
const missing = (m: string) => {
  const key = `widgets.analysis.missing.${codeToKey(m)}`
  return te(key) ? t(key) : m
}
</script>

<template>
  <section class="why" data-testid="why-system">
    <h4>{{ t('ncCard.whySystemSuggests.title') }}</h4>
    <p class="note">{{ t('ncCard.whySystemSuggests.notAVerdict') }}</p>

    <article v-if="versions.current" class="version" data-testid="conclusion-current" :data-version="versions.current.version">
      <p class="head">
        <strong>{{ codeText(OUTCOME_TEXT, versions.current.outcome, t) }}</strong>
        <span class="muted">{{ t('widgets.ncCard.conclusionVersion', { version: versions.current.version }) }} · {{ time(versions.current.recorded_at) }}</span>
      </p>
      <p class="muted">{{ t('widgets.ncCard.rule', { ruleId: versions.current.rule_id }) }} · {{ t('decisions.automationMode.title') }}: {{ mode(versions.current) }}</p>
      <p v-if="versions.current.revised_due_to" class="revised" data-testid="revised">
        {{ t('timeline.marks.revised', { eventId: versions.current.revised_due_to }) }}
      </p>
      <h5>{{ t('ncCard.whySystemSuggests.basis') }}</h5>
      <ul class="list">
        <li v-for="w in analysis.why" :key="w">{{ w }}</li>
        <li v-for="c in versions.current.causes" :key="c.event_id"><RecordLine :record="c" /></li>
      </ul>
    </article>
    <p v-else class="muted">{{ t('empty.noRecords') }}</p>

    <template v-if="analysis.alternatives.length">
      <h5>{{ t('ncCard.whySystemSuggests.alternatives') }}</h5>
      <ul class="list" data-testid="alternatives">
        <li v-for="a in analysis.alternatives" :key="a">{{ a }}</li>
      </ul>
    </template>
    <template v-if="analysis.missing_information.length">
      <h5>{{ t('ncCard.whySystemSuggests.missingData') }}</h5>
      <ul class="list" data-testid="missing">
        <li v-for="m in analysis.missing_information" :key="m">{{ missing(m) }}</li>
      </ul>
      <p class="muted">{{ t('ncCard.circumstances.noCategoricalConclusion') }}</p>
    </template>

    <template v-if="versions.previous.length">
      <ActionButton size="tiny" quaternary data-testid="toggle-previous" @click="showPrevious = !showPrevious" :label="`${t('common.actions.showPreviousVersions')} (${versions.previous.length})`" />
      <div v-if="showPrevious" class="previous" data-testid="conclusion-previous">
        <article v-for="v in versions.previous" :key="v.event_id" class="version old" :data-version="v.version">
          <p class="head">
            <strong>{{ codeText(OUTCOME_TEXT, v.outcome, t) }}</strong>
            <span class="muted">{{ t('widgets.ncCard.conclusionVersion', { version: v.version }) }} · {{ time(v.recorded_at) }}</span>
          </p>
          <p class="muted">{{ t('timeline.marks.oldConclusionKept') }}</p>
        </article>
      </div>
    </template>
  </section>
</template>

<style scoped>
.why {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 8px 10px;
  border-left: 3px solid var(--ant-accent);
  background: var(--ant-surface-subtle);
}

h4,
h5 {
  margin: 0;
}

.note,
.muted,
.head,
.revised {
  margin: 0;
}

.note,
.muted {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.head {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: baseline;
}

.revised {
  color: var(--ant-status-attention-text);
  font-size: var(--ant-fs-meta);
}

.list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin: 0;
  padding-left: 16px;
}

.version.old {
  opacity: 0.75;
}
</style>
