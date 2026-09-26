<script setup lang="ts">
/**
 * Исходный сигнал целиком — второй слой карточки, «Подробности» (FR-51, FR-38,
 * кейс §2.3): на чём основан, вид дефекта, тяжесть, зона, ступени и вектор
 * версий, адреса материалов. Уверенность и качество наблюдения — в первом слое
 * карточки, рядом друг с другом. Сигнал — сообщение о признаке дефекта,
 * а не брак; решения людей его не меняют. Уверенность 0,87 — не «вероятность
 * брака 87 %» (NFR-UI-4).
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { BASIS_KIND_TEXT, SEVERITY_TEXT, bpToFraction, codeText, type NCSourceSignal } from '@/entities/nonconformity'
import { KeyValue, KeyValueList } from '@/shared/ui'
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
    <RecordLine :record="signal.record" technical />
    <KeyValueList>
      <KeyValue :label="t('common.words.basis')" :value="codeText(BASIS_KIND_TEXT, signal.basis_kind, t)" />
      <KeyValue :label="t('common.words.defectType')">
        <span data-testid="defect-type">
          <template v-if="!signal.defect_type_code">{{ t('common.words.unknown') }}</template>
          <template v-else-if="!signal.defect_type_known">{{ t('inspection.unknownDefectType', { code: signal.defect_type_code }) }}</template>
          <template v-else-if="signal.defect_type_label">{{ signal.defect_type_label }} <span class="muted">({{ signal.defect_type_code }})</span></template>
          <template v-else>{{ signal.defect_type_code }}</template>
        </span>
      </KeyValue>
      <KeyValue :label="t('common.words.severity')" :value="codeText(SEVERITY_TEXT, signal.severity, t)" />
      <KeyValue v-if="signal.zone_id" :label="t('common.words.zone')">
        <span data-testid="zone-detail">{{ signal.zone_label ?? signal.zone_id }}<span v-if="signal.zone_label" class="muted"> ({{ signal.zone_id }})</span></span>
      </KeyValue>
    </KeyValueList>

    <section v-if="signal.stages.length" class="block">
      <h5 class="ant-wrap">{{ t('inspection.stages.title') }}</h5>
      <ul>
        <li v-for="s in signal.stages" :key="s.stage" class="ant-wrap" :data-stage="s.stage">
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
      <h5 class="ant-wrap">{{ t('inspection.versions.title') }}</h5>
      <KeyValueList>
        <KeyValue v-for="v in versions" :key="v.k" :label="v.label" :value="v.v" mono />
      </KeyValueList>
    </section>

    <section class="block">
      <h5 class="ant-wrap">{{ t('common.words.evidence') }}</h5>
      <ul v-if="signal.evidence_refs.length">
        <li v-for="ref in signal.evidence_refs" :key="ref" class="ant-wrap"><code>{{ ref }}</code></li>
      </ul>
      <p v-else class="muted ant-wrap">{{ t('empty.materialNotProvided') }}</p>
    </section>
    <p class="muted ant-wrap">{{ t('inspection.outcomeNote.defectFound') }} · {{ t('decisions.signal.originalSignalKept') }}</p>
  </article>
</template>

<style scoped>
/* Карточка сигнала — на всю ширину колонки зоны. */
.signal {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
  min-width: 0;
  padding: var(--ant-space-3) var(--ant-space-4);
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-md);
}

.block {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  min-width: 0;
}

h5 {
  margin: 0;
  color: var(--ant-text-2);
}

ul {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin: 0;
  padding-left: var(--ant-space-4);
}

code {
  font-family: var(--ant-font-mono);
}

.muted {
  margin: 0;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}
</style>
