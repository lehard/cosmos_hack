<script setup lang="ts">
/**
 * Знак и имя системы — «Главный» (решение пользователя Д-65: кодовое имя, как у
 * Главного конструктора). Знак монохромный: буква «Г» и точка на орбите в
 * квадрате; цвет — текущий цвет текста. Пояснение «Платформа управления качеством производства» —
 * мелко под именем, не вместо него.
 *
 * Когда брать: шапка оболочки (`size="sm"`), экран входа (`size="lg"`).
 */
import { useI18n } from 'vue-i18n'

withDefaults(
  defineProps<{
    size?: 'sm' | 'lg'
    /** Показать пояснение под именем. */
    tagline?: boolean
  }>(),
  { size: 'sm', tagline: false },
)

const { t } = useI18n()
</script>

<template>
  <span class="brand-mark" :class="`is-${size}`">
    <svg class="logo" viewBox="0 0 32 32" aria-hidden="true" focusable="false">
      <rect width="32" height="32" rx="7" fill="currentColor" />
      <path class="glyph" d="M11 24V9h11" />
      <circle class="orbit" cx="22" cy="21.5" r="2.25" />
    </svg>
    <span class="names ant-box">
      <span class="name ant-ellipsis">{{ t('shell.brand.name') }}</span>
      <span v-if="tagline" class="tagline ant-ellipsis">{{ t('shell.brand.tagline') }}</span>
    </span>
  </span>
</template>

<style scoped>
.brand-mark {
  display: inline-flex;
  gap: 10px;
  align-items: center;
  min-width: 0;
  color: var(--ant-n-900);
}

.logo {
  flex: none;
  width: 28px;
  height: 28px;
}

.glyph {
  fill: none;
  stroke: var(--ant-surface);
  stroke-width: 3;
  stroke-linecap: square;
}

.orbit {
  fill: var(--ant-surface);
}

.is-lg .logo {
  width: 44px;
  height: 44px;
}

.names {
  display: flex;
  flex-direction: column;
  line-height: var(--ant-lh-tight);
}

.name {
  font-size: var(--ant-fs-lg);
  font-weight: var(--ant-fw-bold);
  letter-spacing: 0.01em;
}

.is-lg .name {
  font-size: var(--ant-fs-display);
}

.tagline {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-xs);
}

.is-lg .tagline {
  font-size: var(--ant-fs-md);
}
</style>
