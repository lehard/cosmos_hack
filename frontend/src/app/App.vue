<script setup lang="ts">
// Корневой компонент: русская локаль Naive UI и общая тема (NFR-UI-2:
// минималистично, одноцветно; цвет — только для статуса и главного действия).
import { dateRuRU, NConfigProvider, NMessageProvider, ruRU, type GlobalThemeOverrides } from 'naive-ui'

const themeOverrides: GlobalThemeOverrides = {
  common: {
    fontFamily: "'PT Sans', system-ui, sans-serif",
    fontFamilyMono: "'PT Mono', ui-monospace, monospace",
    primaryColor: '#1f2937',
    primaryColorHover: '#374151',
    primaryColorPressed: '#111827',
  },
}
</script>

<template>
  <NConfigProvider :locale="ruRU" :date-locale="dateRuRU" :theme-overrides="themeOverrides">
    <NMessageProvider>
      <RouterView />
    </NMessageProvider>
  </NConfigProvider>
</template>
