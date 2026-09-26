<script setup lang="ts">
/**
 * Автор версии области: имя из карточки человека (`access.person.card`), щелчок —
 * окно человека. Нет права читать карточку (технологу политика её не даёт) —
 * просто текст, без ссылки в пустое окно и без запроса с отказом; имя в ответе
 * области (author_name) — запрос бэкенду.
 */
import { computed } from 'vue'
import { canPerform } from '@/entities/incident'
import { usePermissions } from '@/entities/permission'
import { usePersonCard } from '@/entities/person'

const props = defineProps<{ id: string; name?: string | null }>()
const emit = defineEmits<{ open: [id: string] }>()

const permissions = usePermissions()
const allowed = computed(() => canPerform(permissions.data.value?.data.items, 'access.person.card', 'person', props.id))
const card = usePersonCard(() => (allowed.value && !props.name ? props.id : null))
const text = computed(() => props.name || card.data.value?.data.display_name || props.id)
</script>

<template>
  <button v-if="allowed" type="button" class="author ant-wrap" data-testid="version-author" @click="emit('open', id)">{{ text }}</button>
  <span v-else class="ant-wrap" data-testid="version-author">{{ text }}</span>
</template>

<style scoped>
.author {
  padding: 0;
  border: 0;
  background: none;
  color: var(--ant-accent);
  font: inherit;
  cursor: pointer;
}

.author:hover {
  text-decoration: underline;
}
</style>
