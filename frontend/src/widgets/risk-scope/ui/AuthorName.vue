<script setup lang="ts">
/**
 * Автор версии области: имя из карточки человека (`access.person.card`), а не
 * код персоны; щелчок — окно человека. Пока имя не пришло — код, как есть.
 */
import { computed } from 'vue'
import { usePersonCard } from '@/entities/person'

const props = defineProps<{ id: string }>()
const emit = defineEmits<{ open: [id: string] }>()

const card = usePersonCard(() => props.id)
const name = computed(() => card.data.value?.data.display_name || props.id)
</script>

<template>
  <button type="button" class="author ant-wrap" data-testid="version-author" @click="emit('open', id)">{{ name }}</button>
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
