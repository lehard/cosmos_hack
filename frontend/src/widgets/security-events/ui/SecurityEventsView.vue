<script setup lang="ts">
/**
 * Шина событий безопасности — представление стола Аудитора ИБ (только чтение;
 * FR-106; AD-28; кейс Т3, О4): ошибки входа, недействительные подписи,
 * нарушения целостности, отказы в доступе и допуске, конфликты номеров,
 * тревоги по ключам; важность, объект, источник и связанное критическое действие.
 */
import { useI18n } from 'vue-i18n'
import { SECURITY_EVENT_TYPES, type SecurityEvent } from '@/entities/integrity'
import { statusPalette, type StatusTone } from '@/shared/api/generated/statuses'
import type { Density } from '@/shared/config/widget'
import { codeToKey } from '@/shared/i18n'
import { EmptyState } from '@/shared/ui'

withDefaults(defineProps<{ events: SecurityEvent[]; eventType: string | null; density?: Density }>(), { density: 'compact' })
const emit = defineEmits<{ 'update:eventType': [eventType: string | null] }>()
const { t, te, d } = useI18n()

const TONE: Record<SecurityEvent['severity'], StatusTone> = { info: 'info', warning: 'attention', alarm: 'critical' }
const typeKey = (type: string) => `widgets.audit.eventTypes.${codeToKey(type).replace(/\.([a-z])/g, (_, c: string) => c.toUpperCase())}`
const typeText = (type: string) => (te(typeKey(type)) ? t(typeKey(type)) : type)
const typeOptions = SECURITY_EVENT_TYPES.map((v) => ({ label: typeText(v), value: v }))
const time = (iso: string) => d(new Date(iso), 'dateTime')
</script>

<template>
  <div class="events" :class="`density-${density}`" data-testid="security-events">
    <p class="muted">{{ t('widgets.audit.readOnly') }}</p>
    <label class="filter">
      <span>{{ t('widgets.audit.eventType') }}</span>
      <select :value="eventType ?? ''" data-testid="event-type" @change="(e) => emit('update:eventType', (e.target as HTMLSelectElement).value || null)">
        <option value="">{{ t('widgets.audit.allTypes') }}</option>
        <option v-for="o in typeOptions" :key="o.value" :value="o.value">{{ o.label }}</option>
      </select>
    </label>
    <EmptyState v-if="!events.length" compact :title="t('empty.noRecords')" />
    <ol class="rows">
      <li v-for="e in events" :key="e.event_id" class="row" :data-seq="e.seq" :data-severity="e.severity" :style="{ borderLeftColor: statusPalette[TONE[e.severity]] }">
        <div class="line">
          <strong>{{ typeText(e.event_type) }}</strong>
          <span class="sev">{{ t(`widgets.audit.severity.${e.severity}`) }}</span>
          <span class="muted">{{ time(e.occurred_at) }} · № {{ e.seq }}</span>
        </div>
        <div>{{ e.summary }}</div>
        <div class="line muted">
          <span v-if="e.object">{{ t('audit.record.object') }}: {{ e.object.entity }}:{{ e.object.id }}</span>
          <span v-if="e.source_id">{{ t('common.words.source') }}: {{ e.source_id }}</span>
          <span v-if="e.ca_ref">{{ t('audit.caNumber', { caId: e.ca_ref }) }}</span>
        </div>
      </li>
    </ol>
  </div>
</template>

<style scoped>
.events {
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: var(--ant-fs-body);
}

.filter {
  display: flex;
  gap: 8px;
  align-items: center;
  max-width: 420px;
}

.filter span {
  white-space: nowrap;
}

.filter select {
  flex: 1;
  padding: 3px 6px;
  border: 1px solid var(--ant-border-strong);
  border-radius: var(--ant-radius-sm);
  background: var(--ant-surface);
  font: inherit;
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
  padding: 4px 8px;
  border: 1px solid var(--ant-border);
  border-left-width: 3px;
  border-radius: var(--ant-radius-md);
}

.line {
  display: flex;
  flex-wrap: wrap;
  gap: 2px 10px;
  align-items: baseline;
}

.row[data-severity='alarm'] .sev {
  color: var(--ant-status-critical);
  font-weight: var(--ant-fw-bold);
}

.muted {
  margin: 0;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}
</style>
