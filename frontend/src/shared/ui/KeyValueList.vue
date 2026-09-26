<script setup lang="ts">
/**
 * Список пар «ключ — значение» (`<dl>`): в одну или несколько колонок;
 * `stacked` — подпись над значением (узкая правая панель).
 *
 * Когда брать: вместе с KeyValue для реквизитов объекта.
 */
withDefaults(
  defineProps<{
    /** Колонок пар. */
    columns?: 1 | 2 | 3
    /** Подпись над значением. */
    stacked?: boolean
  }>(),
  { columns: 1, stacked: false },
)
</script>

<template>
  <dl class="key-value-list ant-box" :class="{ 'is-stacked': stacked }" :style="{ '--kv-columns': columns }">
    <slot />
  </dl>
</template>

<style scoped>
.key-value-list {
  display: grid;
  grid-template-columns: repeat(var(--kv-columns), minmax(0, 1fr));
  gap: 0 var(--ant-space-6);
  margin: 0;
}
</style>
