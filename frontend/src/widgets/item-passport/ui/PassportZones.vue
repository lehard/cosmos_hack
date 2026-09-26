<script setup lang="ts">
/**
 * Зоны изделия (FR-46, FR-20, FR-21): закрыта ли зона и каким шагом, открыто ли
 * вмешательство — тогда прежние результаты зоны «устарели». «Проверена» экран
 * не утверждает, если сервер этого не сообщил.
 */
import { useI18n } from 'vue-i18n'
import { ZONE_STATUS_TEXT, zoneStatus, type ItemZone } from '@/entities/item'
import { EmptyState } from '@/shared/ui'

defineProps<{ zones: ItemZone[] }>()
const { t } = useI18n()
</script>

<template>
  <section class="zones" data-testid="passport-zones">
    <EmptyState v-if="!zones.length" compact :title="t('empty.noRecords')" />
    <ul v-else>
      <li v-for="z in zones" :key="z.zone_id" :data-zone="z.zone_id" :data-status="zoneStatus(z) ?? 'none'">
        <strong>{{ z.title }}</strong>
        <span v-if="zoneStatus(z)" class="status">{{ t(ZONE_STATUS_TEXT[zoneStatus(z)!]) }}</span>
        <span v-if="z.closed && z.closed_by" class="muted">{{ t('widgets.passport.zones.closedBy', { step: z.closed_by }) }}</span>
        <span v-if="z.open_intervention" class="warn" data-testid="intervention">{{ t('errors.decision.interventionOpen') }}</span>
      </li>
    </ul>
  </section>
</template>

<style scoped>
ul {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin: 0;
  padding: 0;
  list-style: none;
}

li {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 10px;
  align-items: baseline;
}

.status {
  padding: 0 6px;
  border-radius: var(--ant-radius-lg);
  background: var(--ant-n-100);
  font-size: var(--ant-fs-meta);
}

.muted {
  margin: 0;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.warn {
  color: var(--ant-status-attention-text);
  font-size: var(--ant-fs-meta);
}
</style>
