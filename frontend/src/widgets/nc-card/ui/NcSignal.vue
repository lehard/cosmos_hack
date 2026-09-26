<script setup lang="ts">
/**
 * Исходный сигнал (FR-51, FR-38, кейс §2.3): на чём основан, вид дефекта,
 * тяжесть, зона, уверенность анализатора и качество наблюдения — раздельно,
 * ступени и вектор версий, материалы. Сигнал — сообщение о признаке дефекта,
 * а не брак; решения людей его не меняют. Уверенность 0,87 — не «вероятность
 * брака 87 %» (NFR-UI-4).
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { BASIS_KIND_TEXT, SEVERITY_TEXT, bpToFraction, codeText, type NCSourceSignal } from '@/entities/nonconformity'
import RecordLine from './RecordLine.vue'

const props = defineProps<{ signal: NCSourceSignal }>()
const { t, n } = useI18n()

/** Ключи вектора версий (AD-29) → текст; неизвестный ключ показывается как есть. */
const VERSION_TEXT: Record<string, string> = {
  item_revision: 'inspection.versions.itemRevision',
  recipe_ref: 'inspection.versions.recipe',
  camera_config: 'inspection.versions.cameraConfig',
  calibration: 'inspection.versions.calibration',
  analyzer_version: 'inspection.versions.analyzer',
  threshold_profile: 'inspection.versions.thresholdProfile',
  contract_version: 'inspection.versions.dataContract',
  app_version: 'inspection.versions.application',
}
const versions = computed(() => Object.entries(props.signal.versions ?? {}).map(([k, v]) => ({ k, label: VERSION_TEXT[k] ? t(VERSION_TEXT[k]!) : k, v })))
const dec = (bp: number) => n(bpToFraction(bp), 'decimal2')
</script>

<template>
  <article class="signal" :data-signal="signal.signal_id" :data-severity="signal.severity" data-testid="source-signal">
    <RecordLine :record="signal.record" />
    <dl class="facts">
      <dt>{{ t('common.words.basis') }}</dt>
      <dd>{{ codeText(BASIS_KIND_TEXT, signal.basis_kind, t) }}</dd>
      <dt>{{ t('common.words.defectType') }}</dt>
      <dd data-testid="defect-type">
        <template v-if="!signal.defect_type_code">{{ t('common.words.unknown') }}</template>
        <template v-else-if="!signal.defect_type_known">{{ t('inspection.unknownDefectType', { code: signal.defect_type_code }) }}</template>
        <template v-else>{{ signal.defect_type_code }}</template>
      </dd>
      <dt>{{ t('common.words.severity') }}</dt>
      <dd>{{ codeText(SEVERITY_TEXT, signal.severity, t) }}</dd>
      <template v-if="signal.zone_id">
        <dt>{{ t('common.words.zone') }}</dt>
        <dd>{{ signal.zone_id }}</dd>
      </template>
      <dt>{{ t('inspection.analyzerConfidence') }}</dt>
      <dd data-testid="confidence" :title="t('hints.analyzerConfidence')">
        <template v-if="signal.analyzer_confidence_bp != null">
          {{ dec(signal.analyzer_confidence_bp) }} <span class="muted">— {{ t('inspection.confidenceIsNotProbability') }}</span>
        </template>
        <template v-else>{{ t('empty.noDataUnknown') }}</template>
      </dd>
      <dt>{{ t('inspection.observationQuality') }}</dt>
      <dd data-testid="observation-quality" :title="t('hints.observationQuality')">
        {{ signal.observation_quality_bp != null ? dec(signal.observation_quality_bp) : t('empty.noDataUnknown') }}
      </dd>
    </dl>

    <section v-if="signal.stages.length" class="block">
      <h5>{{ t('inspection.stages.title') }}</h5>
      <ul>
        <li v-for="s in signal.stages" :key="s.stage" :data-stage="s.stage">
          {{
            t('inspection.stages.stageLine', {
              stage: s.stage,
              result: s.output_note ?? t('common.words.unknown'),
              confidence: s.confidence_bp != null ? dec(s.confidence_bp) : t('common.words.unknown'),
              version: s.version,
            })
          }}
        </li>
      </ul>
    </section>

    <section v-if="versions.length" class="block">
      <h5>{{ t('inspection.versions.title') }}</h5>
      <dl class="facts">
        <template v-for="v in versions" :key="v.k">
          <dt>{{ v.label }}</dt>
          <dd><code>{{ v.v }}</code></dd>
        </template>
      </dl>
    </section>

    <section class="block">
      <h5>{{ t('common.words.evidence') }}</h5>
      <ul v-if="signal.evidence_refs.length">
        <li v-for="ref in signal.evidence_refs" :key="ref"><code>{{ ref }}</code></li>
      </ul>
      <p v-else class="muted" data-testid="no-material">{{ t('empty.materialNotProvided') }}</p>
    </section>
    <p class="muted">{{ t('inspection.outcomeNote.defectFound') }} · {{ t('decisions.signal.originalSignalKept') }}</p>
  </article>
</template>

<style scoped>
.signal {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 8px 10px;
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-md);
}

.facts {
  display: grid;
  grid-template-columns: max-content minmax(0, 1fr);
  gap: 2px 12px;
  margin: 0;
}

.facts dt {
  color: var(--ant-text-3);
}

.facts dd {
  margin: 0;
}

h5 {
  margin: 0 0 2px;
}

ul {
  margin: 0;
  padding-left: 16px;
}

.muted {
  margin: 0;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}
</style>
