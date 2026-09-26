<script setup lang="ts">
// Корневой компонент: русская локаль Naive UI (календарь, пагинация) и общая
// тема дизайн-системы из токенов (NFR-UI-2, shared/ui/theme); провайдеры сообщений и диалогов для всех страниц.
import { dateRuRU, NConfigProvider, NDialogProvider, NMessageProvider, ruRU } from 'naive-ui'
import { themeOverrides } from '@/shared/ui/theme'
</script>

<template>
  <NConfigProvider :locale="ruRU" :date-locale="dateRuRU" :theme-overrides="themeOverrides">
    <NMessageProvider>
      <NDialogProvider>
        <RouterView />
      </NDialogProvider>
    </NMessageProvider>
  </NConfigProvider>
</template>
