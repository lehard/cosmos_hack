<script setup lang="ts">
/**
 * Шапка паспорта изделия (FR-42, PRD §3b): кто это, по какой версии процесса
 * запущено, уровень идентификации, пять осей статуса раздельно — «в изоляции»
 * (положение) ≠ «блок» (сдерживание) ≠ «несоответствие подтверждено»
 * (качество) — и связанные несоответствия и инциденты.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NAlert } from 'naive-ui'
import { AXIS_TEXT, IDENTIFICATION_TEXT, STATUS_AXES_ORDER, SummaryTag, identificationDoubtful, isHeld, type ItemPassport } from '@/entities/item'
import { StatusTag } from '@/shared/ui'

const props = withDefaults(
  defineProps<{
    passport: ItemPassport
    /** Есть ли экран карточки несоответствия (ссылки активны). */
    canOpenNc?: boolean
    /** Есть ли экран инцидента. */
    canOpenIncident?: boolean
  }>(),
  { canOpenNc: false, canOpenIncident: false },
)
const emit = defineEmits<{ 'open-nc': [ncId: string]; 'open-incident': [incidentId: string] }>()
const { t } = useI18n()

const status = computed(() => props.passport.status)
const doubtful = computed(() => identificationDoubtful(props.passport.identification))
</script>

<template>
  <header class="head" data-testid="passport-head">
    <div class="title">
      <strong class="label">{{ passport.label }}</strong>
      <span class="muted">{{ t('passport.identifier') }}: <code>{{ passport.item_id }}</code></span>
      <SummaryTag :code="status.summary" data-testid="summary" />
    </div>
    <p class="muted">
      {{ t('common.words.itemType') }}: {{ passport.item_type_id }} · {{ t('common.words.revision') }}: {{ passport.item_revision }}
      <template v-if="passport.order_id"> · {{ t('common.words.productionOrder') }}: {{ passport.order_id }}</template>
    </p>
    <p class="muted" :title="passport.process_version">{{ t('passport.processVersion', { version: passport.process_version }) }}</p>
    <p class="muted" data-testid="identification" :data-level="passport.identification">
      {{ t('widgets.passport.identification.title') }}: {{ t(IDENTIFICATION_TEXT[passport.identification]) }}
    </p>

    <dl class="axes" data-testid="axes">
      <template v-for="axis in STATUS_AXES_ORDER" :key="axis">
        <dt>{{ t(AXIS_TEXT[axis]) }}</dt>
        <dd><StatusTag :axis="axis" :code="status[axis]" /></dd>
      </template>
    </dl>

    <p v-if="isHeld(status)" class="muted" data-testid="hold-hint">{{ t('hints.hold') }}</p>
    <p v-if="status.position === 'isolated'" class="muted" data-testid="isolation-hint">{{ t('hints.isolation') }}</p>
    <p v-if="status.quality === 'unable_to_assess'" class="muted" data-testid="unable-hint">{{ t('hints.unableToAssess') }}</p>
    <p v-if="status.quality === 'accepted_with_concession'" class="muted" data-testid="concession-hint">{{ t('decisions.concession.resultStatus') }}</p>
    <NAlert v-if="doubtful" type="warning" :bordered="false" :show-icon="false" data-testid="identification-doubtful">
      {{ t('widgets.passport.identification.doubtful') }}
    </NAlert>

    <p v-if="passport.nonconformities.length" class="links" data-testid="nonconformities">
      <span class="muted">{{ t('common.words.nonconformity') }}:</span>
      <template v-for="nc in passport.nonconformities" :key="nc">
        <button v-if="canOpenNc" type="button" class="linklike" @click="emit('open-nc', nc)">{{ nc }}</button>
        <span v-else>{{ nc }}</span>
      </template>
    </p>
    <p v-if="passport.incidents.length" class="links" data-testid="incidents">
      <span class="muted">{{ t('common.words.incident') }}:</span>
      <template v-for="inc in passport.incidents" :key="inc">
        <button v-if="canOpenIncident" type="button" class="linklike" @click="emit('open-incident', inc)">{{ inc }}</button>
        <span v-else>{{ inc }}</span>
      </template>
    </p>
    <p class="muted" data-testid="documents-from-history">{{ t('passport.documentsFromHistory', { n: passport.documents.length }) }}</p>
  </header>
</template>

<style scoped>
.head {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.title,
.links {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 10px;
  align-items: baseline;
  margin: 0;
}

.label {
  font-size: 1.2em;
}

.muted {
  margin: 0;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.axes {
  display: grid;
  grid-template-columns: max-content minmax(0, 1fr);
  gap: 2px 12px;
  margin: 0;
}

.axes dt {
  color: var(--ant-text-3);
}

.axes dd {
  margin: 0;
}

.linklike {
  padding: 0;
  border: 0;
  background: none;
  color: var(--ant-accent);
  font: inherit;
  cursor: pointer;
}
</style>
