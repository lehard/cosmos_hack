<script setup lang="ts">
// Пустая стартовая страница каркаса (эпик 01): проверяет, что интерфейс
// встроен в бинарник и открывается. Столы ролей заменят её в эпике 03.
import { NEmpty, NIcon, NLayout, NLayoutContent, NLayoutHeader, NText } from 'naive-ui'
import { ClipboardCheck } from '@vicons/tabler'
</script>

<template>
  <NLayout class="page">
    <NLayoutHeader bordered class="header">
      <NText strong>ant</NText>
      <NText depth="3">контроль качества деталей</NText>
    </NLayoutHeader>
    <NLayoutContent class="content">
      <NEmpty description="Система запущена. Столы ролей появятся здесь.">
        <template #icon>
          <NIcon><ClipboardCheck /></NIcon>
        </template>
      </NEmpty>
    </NLayoutContent>
  </NLayout>
</template>

<style scoped>
.page {
  height: 100%;
}

.header {
  display: flex;
  gap: 12px;
  align-items: baseline;
  padding: 12px 24px;
}

.content {
  display: flex;
  align-items: center;
  justify-content: center;
  height: calc(100% - 49px);
}
</style>
