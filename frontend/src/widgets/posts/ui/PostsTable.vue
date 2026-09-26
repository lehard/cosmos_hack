<script setup lang="ts">
/**
 * Панель «Посты» (FR-6): участок — кто назначен — на месте ли (СКУД, ключ) —
 * текущее изделие. Люди на схеме не показываются — только здесь (PRD §4.1).
 */
import { useI18n } from 'vue-i18n'
import type { PostRow } from '@/entities/workplace'
import { statusPalette } from '@/shared/api/generated/statuses'
import { PRESENCE } from '../model/presence'

defineProps<{ rows: readonly PostRow[] }>()
const emit = defineEmits<{ 'open-item': [itemId: string] }>()
const { t } = useI18n()
</script>

<template>
  <table class="posts">
    <thead>
      <tr>
        <th>{{ t('liveMap.posts.station') }}</th>
        <th>{{ t('liveMap.posts.assigned') }}</th>
        <th>{{ t('liveMap.posts.presence') }}</th>
        <th>{{ t('liveMap.posts.currentItem') }}</th>
      </tr>
    </thead>
    <tbody>
      <tr v-for="r in rows" :key="r.workplace_id" :data-workplace="r.workplace_id" :data-presence="r.presence">
        <td>{{ r.station }}</td>
        <td>{{ r.assigned?.display ?? t('liveMap.posts.notAssigned') }}</td>
        <td>
          <span class="presence">
            <span class="dot" :style="{ background: statusPalette[PRESENCE[r.presence].tone] }" aria-hidden="true" />
            {{ t(PRESENCE[r.presence].key) }}
          </span>
        </td>
        <td>
          <button v-if="r.current_item" type="button" class="item" :title="t('common.actions.openPassport')" @click="emit('open-item', r.current_item.item_id)">
            {{ r.current_item.label }}
          </button>
          <span v-else class="none">—</span>
        </td>
      </tr>
    </tbody>
  </table>
</template>

<style scoped>
.posts {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}

th {
  color: #6b7280;
  font-weight: 400;
  text-align: left;
}

th,
td {
  padding: 4px 8px;
  border-bottom: 1px solid #eef0f3;
}

.presence {
  display: inline-flex;
  gap: 6px;
  align-items: center;
}

.dot {
  flex: none;
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.item {
  padding: 0;
  border: 0;
  background: none;
  color: #2f6fdb;
  font: inherit;
  font-family: 'PT Mono', monospace;
  cursor: pointer;
}

.none {
  color: #9ca3af;
}
</style>
