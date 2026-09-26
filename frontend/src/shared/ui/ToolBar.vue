<script setup lang="ts">
/**
 * Панель инструментов: слева — переключатели, фильтры, поиск; справа (слот
 * `end`) — действия. Переносится на новую строку, а не вылезает за край.
 *
 * Когда брать: ряд элементов управления над таблицей, списком или графиком.
 */
</script>

<template>
  <div class="tool-bar" role="toolbar">
    <div class="start ant-box"><slot /></div>
    <div v-if="$slots.end" class="end ant-box"><slot name="end" /></div>
  </div>
</template>

<style scoped>
.tool-bar {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-2) var(--ant-space-4);
  align-items: center;
  justify-content: space-between;
  min-width: 0;
}

.start,
.end {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-2);
  align-items: center;
}

.end {
  margin-left: auto;
}
</style>
