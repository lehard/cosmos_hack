<script setup lang="ts">
/**
 * Страница или стол с заголовком: заголовок (одна строка, многоточие и полный
 * текст в подсказке), подзаголовок, место под действия справа, тело.
 *
 * Когда брать: корень любой страницы в оболочке (стол роли, паспорт изделия,
 * карточка несоответствия, справка). Внутри — SectionPanel и виджеты.
 */
defineProps<{
  /** Заголовок страницы. */
  title: string
  /** Пояснение под заголовком. */
  subtitle?: string
}>()
</script>

<template>
  <section class="app-page">
    <header class="head">
      <div class="titles ant-box">
        <h1 class="title ant-ellipsis" :title="title">{{ title }}</h1>
        <p v-if="subtitle" class="subtitle ant-clamp-2">{{ subtitle }}</p>
        <div v-if="$slots.meta" class="meta ant-box"><slot name="meta" /></div>
      </div>
      <div v-if="$slots.actions" class="actions ant-box"><slot name="actions" /></div>
    </header>
    <div class="body ant-box"><slot /></div>
  </section>
</template>

<style scoped>
.app-page {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-4);
  min-width: 0;
}

.head {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-3) var(--ant-space-6);
  align-items: flex-end;
  justify-content: space-between;
}

.titles {
  display: flex;
  flex: 1 1 320px;
  flex-direction: column;
  gap: var(--ant-space-1);
}

.title {
  font-size: var(--ant-fs-xl);
}

.subtitle,
.meta {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-sm);
}

.actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-2);
  align-items: center;
}

.body {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-4);
}
</style>
