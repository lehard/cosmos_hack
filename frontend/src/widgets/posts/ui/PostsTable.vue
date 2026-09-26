<script setup lang="ts">
/**
 * Панель «Посты» (FR-6): участок — кто назначен — на месте ли (СКУД, ключ) —
 * текущее изделие. Люди на схеме не показываются — только здесь (PRD §4.1).
 * Щелчок по строке (или по названию поста) — окно поста; по назначенному —
 * окно сотрудника; по изделию — окно изделия (Д-70).
 */
import { useI18n } from 'vue-i18n'
import type { PostRow } from '@/entities/workplace'
import { statusPalette } from '@/shared/api/generated/statuses'
import { PRESENCE } from '../model/presence'
import { DataTable } from '@/shared/ui'

withDefaults(
  defineProps<{
    rows: readonly PostRow[]
    /** Есть окно поста (оболочка поставила окно записи). */
    canOpenPost?: boolean
    /** Есть окно сотрудника. */
    canOpenPerson?: boolean
  }>(),
  { canOpenPost: false, canOpenPerson: false },
)
const emit = defineEmits<{
  'open-post': [workplaceId: string]
  'open-person': [personId: string]
  'open-item': [itemId: string]
}>()
const { t } = useI18n()
</script>

<template>
  <DataTable class="posts">
    <thead>
      <tr>
        <th>{{ t('liveMap.posts.station') }}</th>
        <th>{{ t('liveMap.posts.assigned') }}</th>
        <th>{{ t('liveMap.posts.presence') }}</th>
        <th>{{ t('liveMap.posts.currentItem') }}</th>
      </tr>
    </thead>
    <tbody>
      <tr
        v-for="r in rows"
        :key="r.workplace_id"
        :class="{ row: canOpenPost }"
        :data-workplace="r.workplace_id"
        :data-presence="r.presence"
        @click="canOpenPost && emit('open-post', r.workplace_id)"
      >
        <td>
          <button
            v-if="canOpenPost"
            type="button"
            class="link post ant-wrap"
            :title="t('shell.record.workplace.open')"
            @click.stop="emit('open-post', r.workplace_id)"
          >
            {{ r.station }}
          </button>
          <span v-else class="ant-wrap">{{ r.station }}</span>
        </td>
        <td>
          <button
            v-if="r.assigned && canOpenPerson"
            type="button"
            class="link person ant-wrap"
            :title="t('shell.record.person.open')"
            @click.stop="emit('open-person', r.assigned.person_id)"
          >
            {{ r.assigned.display }}
          </button>
          <span v-else-if="r.assigned" class="ant-wrap">{{ r.assigned.display }}</span>
          <span v-else class="none ant-wrap">{{ t('liveMap.posts.notAssigned') }}</span>
        </td>
        <td>
          <span class="presence">
            <span class="dot" :style="{ background: statusPalette[PRESENCE[r.presence].tone] }" aria-hidden="true" />
            <span class="ant-wrap">{{ t(PRESENCE[r.presence].key) }}</span>
          </span>
        </td>
        <td>
          <button
            v-if="r.current_item"
            type="button"
            class="link item ant-wrap"
            :title="t('common.actions.openPassport')"
            @click.stop="emit('open-item', r.current_item.item_id)"
          >
            {{ r.current_item.label }}
          </button>
          <span v-else class="none">—</span>
        </td>
      </tr>
    </tbody>
  </DataTable>
</template>

<style scoped>

.presence {
  display: inline-flex;
  min-width: 0;
  gap: 6px;
  align-items: center;
}

.dot {
  flex: none;
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.row {
  cursor: pointer;
}

.link {
  padding: 0;
  border: 0;
  background: none;
  color: var(--ant-text);
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.link:hover,
.link:focus-visible {
  color: var(--ant-accent);
  text-decoration: underline;
}

.person,
.item {
  color: var(--ant-accent);
}

.item {
  font-family: var(--ant-font-mono);
}

.none {
  color: var(--ant-n-400);
}
</style>
