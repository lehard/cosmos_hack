<script setup lang="ts">
/**
 * Пара «ключ — значение»: приглушённая подпись и значение (переносится, коды —
 * моноширинно). Значение — свойством `value` или слотом (статус, ссылка).
 * Пустое значение — «—», а не пустое место (NFR-UI-4: отсутствие данных видно).
 *
 * Когда брать: реквизиты объекта (паспорт, карточка, узел карты). Несколько
 * пар — внутри KeyValueList.
 */
withDefaults(
  defineProps<{
    /** Подпись. */
    label: string
    value?: string | number | null
    /** Номер, код, отпечаток — моноширинно. */
    mono?: boolean
  }>(),
  { value: null, mono: false },
)
</script>

<template>
  <div class="key-value">
    <dt class="key ant-wrap">{{ label }}</dt>
    <dd class="value ant-wrap" :class="{ 'ant-mono': mono }">
      <slot>{{ value === null || value === '' ? '—' : value }}</slot>
    </dd>
  </div>
</template>

<style scoped>
.key-value {
  display: grid;
  grid-template-columns: minmax(96px, 40%) minmax(0, 1fr);
  gap: var(--ant-space-3);
  align-items: baseline;
  min-width: 0;
  padding: calc(var(--ant-pad-cell-y) / 2) 0;
}

.key {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.value {
  margin: 0;
  color: var(--ant-text);
}

:global(.key-value-list.is-stacked) .key-value {
  grid-template-columns: minmax(0, 1fr);
  gap: 2px;
}
</style>
