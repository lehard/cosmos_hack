<script setup lang="ts">
/**
 * Зоны изделия (FR-46, FR-20, FR-21): закрыта ли зона и каким шагом, открыто ли
 * вмешательство — тогда прежние результаты зоны «устарели». «Проверена» экран
 * не утверждает, если сервер этого не сообщил.
 */
import { useI18n } from 'vue-i18n'
import { ZONE_STATUS_TEXT, zoneStatus, type ItemZone } from '@/entities/item'

defineProps<{ zones: ItemZone[] }>()
const { t } = useI18n()
</script>

<template>
  <section class="zones" data-testid="passport-zones">
    <p v-if="!zones.length" class="muted">{{ t('empty.noRecords') }}</p>
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
  border-radius: 8px;
  background: #f3f4f6;
  font-size: 12px;
}

.muted {
  margin: 0;
  color: #6b7280;
  font-size: 12px;
}

.warn {
  color: #b45309;
  font-size: 12px;
}
</style>
