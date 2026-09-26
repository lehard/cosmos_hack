<script setup lang="ts">
/**
 * «Что передала камера» и «Почему этой версии можно доверять?» (разбор «Камеры +
 * ИИ»; FR-98): человеческий вид наблюдения — точка, исход, уверенность рядом с
 * качеством кадра, вектор версий (анализатор, камера, карта контроля, калибровка),
 * уровень доверия, с которым система реагировала, и что ей разрешено делать самой;
 * по одному щелчку — паспорт допуска анализатора: стадия и статус, когда допущен,
 * история, «демо-паспорт без экзамена — не промышленная валидация». Только чтение:
 * управление допуском — на столе начальника ОТК.
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAnalyzerPassport, useObservationAccount } from '@/entities/analyzer-passport'
import { bpToFraction } from '@/entities/nonconformity'
import { KeyValue, KeyValueList } from '@/shared/ui'

const props = defineProps<{ eventId: string }>()
const { t, te, n, d } = useI18n()

const accountQ = useObservationAccount(() => props.eventId)
const account = computed(() => accountQ.data.value?.data ?? null)
const trustOpen = ref(false)
const passportQ = useAnalyzerPassport(() => (trustOpen.value ? (account.value?.passport_id ?? null) : null))
const passport = computed(() => passportQ.data.value?.data ?? null)

const dec = (bp: number) => n(bpToFraction(bp), 'decimal2')
const time = (x: string) => d(new Date(x), 'dateTime')
const outcome = (o: string) => (te(`widgets.visionAdaptation.outcome.${o}`) ? t(`widgets.visionAdaptation.outcome.${o}`) : o)
const status = (s: string, stage?: string) => {
  const key = s === 'active' && stage && stage !== 'active' ? stage : s
  const k = `statuses.analyzerPassport.${key === 'not_admitted' ? 'notAdmitted' : key}`
  return te(k) ? t(k) : t('widgets.visionAdaptation.notAdmitted')
}
const action = (a: string) => (te(`widgets.visionAdaptation.actions.${a}`) ? t(`widgets.visionAdaptation.actions.${a}`) : a)
/** Ключи вектора версий → текст (как в «Подробностях» сигнала); неизвестный — как есть. */
const VERSION_TEXT: Record<string, string> = {
  analyzer: 'inspection.versions.analyzer',
  analyzer_version: 'inspection.versions.analyzer',
  camera: 'inspection.versions.cameraConfig',
  camera_config: 'inspection.versions.cameraConfig',
  recipe: 'inspection.versions.recipe',
  recipe_ref: 'inspection.versions.recipe',
  calibration: 'inspection.versions.calibration',
  threshold_profile: 'inspection.versions.thresholdProfile',
  item_revision: 'inspection.versions.itemRevision',
}
const versions = computed(() =>
  Object.entries(account.value?.versions ?? {})
    .filter(([k]) => VERSION_TEXT[k])
    .map(([k, v]) => ({ k, label: t(VERSION_TEXT[k]!), v: String(v) })),
)
</script>

<template>
  <section class="camera" data-testid="camera-observation">
    <p v-if="accountQ.error.value && !account" class="muted ant-wrap">{{ t('ncCard.camera.unavailable') }}</p>
    <template v-if="account">
      <KeyValueList>
        <KeyValue v-if="account.point" :label="t('ncCard.camera.point')" :value="account.point" />
        <KeyValue :label="t('ncCard.camera.when')" :value="time(account.occurred_at)" />
        <KeyValue :label="t('ncCard.camera.outcome')" :value="outcome(account.outcome)" />
        <KeyValue :label="t('ncCard.camera.confidenceQuality')">
          <span data-testid="camera-cq">{{ account.confidence_bp != null ? dec(account.confidence_bp) : '—' }} · {{ account.quality_bp != null ? dec(account.quality_bp) : '—' }}</span>
        </KeyValue>
        <KeyValue v-for="v in versions" :key="v.k" :label="v.label" :value="v.v" />
        <KeyValue :label="t('widgets.visionAdaptation.observation.levelThen')">
          <span data-testid="camera-level">{{ t(`widgets.visionAdaptation.levels.${account.level_then}`) }}</span>
        </KeyValue>
      </KeyValueList>
      <p v-if="account.allowed_auto_actions.length" class="ant-wrap" data-testid="camera-auto">
        <span class="muted">{{ t('widgets.visionAdaptation.allowed') }}:</span> {{ account.allowed_auto_actions.map(action).join(' · ') }}
      </p>
      <p class="muted ant-wrap">{{ t('widgets.visionAdaptation.forbidden') }}</p>
      <p v-if="account.status_then !== account.status_now" class="warn ant-wrap" data-testid="camera-status-changed">
        {{ t('ncCard.camera.statusChanged', { then: status(account.status_then), now: status(account.status_now) }) }}
      </p>

      <button v-if="account.passport_id" type="button" class="more" data-testid="toggle-trust" @click="trustOpen = !trustOpen">
        {{ trustOpen ? t('ncCard.camera.trustHide') : t('ncCard.camera.trustShow') }}
      </button>
      <div v-if="trustOpen && passport" class="trust" data-testid="trust">
        <p class="trust-head ant-wrap">
          <strong>{{ passport.title ?? passport.analyzer_id }}</strong>
          <span>{{ status(passport.status, passport.stage) }}</span>
          <span class="muted">{{ t('widgets.visionAdaptation.level', { n: passport.trust_level }) }}</span>
        </p>
        <p class="ant-wrap">{{ t('ncCard.camera.admittedAt', { time: time(passport.admitted_at), recipe: passport.recipe_ref }) }}</p>
        <p v-if="passport.provenance === 'genesis'" class="warn ant-wrap" data-testid="trust-genesis">{{ t('widgets.visionAdaptation.genesis') }}</p>
        <p v-if="passport.suspension" class="warn ant-wrap">{{ t('ncCard.camera.suspended') }}</p>
        <p v-if="passport.history?.length" class="muted ant-wrap">{{ t('ncCard.camera.history', { n: passport.history.length }) }}</p>
      </div>
    </template>
  </section>
</template>

<style scoped>
.camera,
.trust {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
  min-width: 0;
}

p {
  margin: 0;
}

.trust {
  padding: var(--ant-space-2) var(--ant-space-3);
  border-left: 3px solid var(--ant-accent);
  border-radius: 0 var(--ant-radius-md) var(--ant-radius-md) 0;
  background: var(--ant-surface-subtle);
}

.trust-head {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-1) var(--ant-space-3);
  align-items: baseline;
}

.muted {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.warn {
  color: var(--ant-status-attention-text);
}

.more {
  align-self: flex-start;
  padding: 0;
  border: 0;
  background: none;
  color: var(--ant-accent);
  font: inherit;
  cursor: pointer;
}
</style>
