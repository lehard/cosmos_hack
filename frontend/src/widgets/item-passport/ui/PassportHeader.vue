<script setup lang="ts">
/**
 * Шапка паспорта изделия (FR-42, PRD §3b): кто это, по какой версии процесса
 * запущено, пять осей статуса и статус в каждом инциденте. Оси показываются
 * раздельно — «в изоляции» (положение) ≠ «блок» (сдерживание) ≠ «несоответствие
 * подтверждено» (качество).
 */
import { useI18n } from 'vue-i18n'
import { NAlert } from 'naive-ui'
import { AXIS_TEXT, STATUS_AXES_ORDER, SummaryTag, type ItemPassport } from '@/entities/item'
import { StatusTag } from '@/shared/ui'

defineProps<{ passport: ItemPassport }>()
const { t } = useI18n()
</script>

<template>
  <header class="head" data-testid="passport-head">
    <div class="title">
      <strong class="label">{{ passport.label }}</strong>
      <span class="muted">{{ t('passport.identifier') }}: <code>{{ passport.item_id }}</code></span>
      <SummaryTag :code="passport.summary" data-testid="summary" />
    </div>
    <p class="muted">
      {{ t('common.words.itemType') }}: {{ passport.item_type_id }} · {{ t('common.words.revision') }}: {{ passport.item_revision }} ·
      {{ t('passport.processVersion', { version: passport.process_version }) }}
    </p>

    <dl class="axes" data-testid="axes">
      <template v-for="axis in STATUS_AXES_ORDER" :key="axis">
        <dt>{{ t(AXIS_TEXT[axis]) }}</dt>
        <dd><StatusTag :axis="axis" :code="passport.statuses[axis]" /></dd>
      </template>
      <template v-for="inc in passport.incidents" :key="inc.incident_id">
        <dt>{{ t('common.words.incident') }} {{ inc.label }}</dt>
        <dd><StatusTag axis="incident" :code="inc.status" /></dd>
      </template>
    </dl>

    <p v-if="passport.statuses.containment === 'item_hold' || passport.statuses.containment === 'lot_hold'" class="muted" data-testid="hold-hint">
      {{ t('hints.hold') }}
    </p>
    <p v-if="passport.statuses.quality === 'unable_to_assess'" class="muted" data-testid="unable-hint">{{ t('hints.unableToAssess') }}</p>
    <p v-if="passport.statuses.quality === 'accepted_with_concession'" class="muted" data-testid="concession-hint">
      {{ t('decisions.concession.resultStatus') }}
    </p>
    <NAlert v-if="passport.processing_stopped" type="error" :bordered="false" :show-icon="false" data-testid="processing-stopped">
      {{ t('passport.processingStopped') }}
    </NAlert>
    <NAlert v-if="passport.identification_questioned" type="warning" :bordered="false" :show-icon="false" data-testid="identification-questioned">
      {{ t('widgets.passport.identificationQuestioned') }}
    </NAlert>
    <p class="muted" data-testid="documents-from-history">{{ t('passport.documentsFromHistory', { n: passport.documents_from_history }) }}</p>
  </header>
</template>

<style scoped>
.head {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.title {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 12px;
  align-items: baseline;
}

.label {
  font-size: 1.2em;
}

.muted {
  margin: 0;
  color: #6b7280;
  font-size: 12px;
}

.axes {
  display: grid;
  grid-template-columns: max-content minmax(0, 1fr);
  gap: 2px 12px;
  margin: 0;
}

.axes dt {
  color: #6b7280;
}

.axes dd {
  margin: 0;
}
</style>
