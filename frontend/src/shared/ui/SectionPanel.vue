<script setup lang="ts">
/**
 * Панель (карточка) секции: заголовок с местом под действия, тело, подвал.
 * Виды: `card` — белая карточка с рамкой; `subtle` — приглушённый блок внутри
 * карточки; `plain` — только заголовок и отступы.
 *
 * Когда брать: смысловой блок внутри виджета или страницы («Сигнал», «Решение»,
 * «Документы»). Сам виджет на столе — WidgetFrame, не SectionPanel.
 */
import type { PanelVariant } from './types'

withDefaults(
  defineProps<{
    /** Заголовок секции. */
    title?: string
    /** Пояснение под заголовком. */
    subtitle?: string
    variant?: PanelVariant
    /** Внутренние отступы тела. */
    padded?: boolean
  }>(),
  { title: undefined, subtitle: undefined, variant: 'card', padded: true },
)
</script>

<template>
  <section class="section-panel" :class="[`is-${variant}`, { 'is-padded': padded }]">
    <header v-if="title || $slots.extra" class="head">
      <div class="titles ant-box">
        <h3 v-if="title" class="title ant-ellipsis" :title="title">{{ title }}</h3>
        <p v-if="subtitle" class="subtitle ant-wrap">{{ subtitle }}</p>
      </div>
      <div v-if="$slots.extra" class="extra ant-box"><slot name="extra" /></div>
    </header>
    <div class="body ant-box"><slot /></div>
    <footer v-if="$slots.footer" class="foot ant-box"><slot name="footer" /></footer>
  </section>
</template>

<style scoped>
.section-panel {
  display: flex;
  flex-direction: column;
  min-width: 0;
  border-radius: var(--ant-radius-lg);
}

.is-card {
  border: 1px solid var(--ant-border);
  background: var(--ant-surface);
  box-shadow: var(--ant-shadow-sm);
}

.is-subtle {
  border: 1px solid var(--ant-border);
  background: var(--ant-surface-subtle);
}

.head {
  display: flex;
  gap: var(--ant-space-3);
  align-items: flex-start;
  justify-content: space-between;
  padding: var(--ant-pad-section) var(--ant-pad-section) 0;
}

.is-plain .head {
  padding: 0 0 var(--ant-space-2);
}

.titles {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  gap: 2px;
}

.title {
  font-size: var(--ant-fs-title);
}

.subtitle {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.extra {
  display: flex;
  flex: 0 1 auto;
  flex-wrap: wrap;
  gap: var(--ant-space-2);
  align-items: center;
  justify-content: flex-end;
}

.body {
  display: flex;
  flex-direction: column;
  gap: var(--ant-gap);
}

.is-padded .body {
  padding: var(--ant-pad-section);
}

.is-plain.is-padded .body {
  padding: 0;
}

.head + .body {
  padding-top: var(--ant-space-3);
}

.foot {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-2);
  padding: var(--ant-space-3) var(--ant-pad-section);
  border-top: 1px solid var(--ant-border);
}
</style>
