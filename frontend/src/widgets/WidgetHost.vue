<script setup lang="ts">
/**
 * Хост виджета: по id из стола находит виджет в реестре, лениво грузит его и
 * передаёт WidgetProps. Неизвестный id (стол разошёлся с реестром, хотя make check
 * это ловит) — рамка в состоянии «ошибка входа», а не пустое место.
 * `frame` — как рамка встроена (UI-6, Д-70): без повторного заголовка на вкладке
 * стола, без обводки в окне записи.
 */
import { computed, defineAsyncComponent, provide, type Component } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Density } from '@/shared/config/widget'
import { WIDGET_FRAME_CONTEXT, WidgetFrame, type FrameContext } from '@/shared/ui'
import { isWidgetId, widgetRegistry, type WidgetId } from './registry'

const props = withDefaults(
  defineProps<{
    widget: string
    slotId: string
    slice?: Record<string, unknown>
    density?: Density
    frame?: FrameContext
  }>(),
  { slice: () => ({}), density: 'comfortable', frame: () => ({}) },
)

// Геттеры — чтобы рамка видела смену свойства без пересоздания виджета.
provide(WIDGET_FRAME_CONTEXT, {
  get hideTitle() {
    return props.frame.hideTitle
  },
  get plain() {
    return props.frame.plain
  },
})

const { t } = useI18n()

/** Один асинхронный компонент на виджет — не пересоздаём при перерисовке. */
const cache = new Map<WidgetId, Component>()
function componentOf(id: WidgetId): Component {
  let c = cache.get(id)
  if (!c) {
    c = defineAsyncComponent(widgetRegistry[id].load)
    cache.set(id, c)
  }
  return c
}

const resolved = computed(() => (isWidgetId(props.widget) ? { id: props.widget, def: widgetRegistry[props.widget] } : null))
</script>

<template>
  <component
    :is="componentOf(resolved.id)"
    v-if="resolved"
    :widget-id="resolved.id"
    :title-key="resolved.def.titleKey"
    :slot-id="slotId"
    :slice="slice"
    :density="density"
  />
  <WidgetFrame v-else title-key="shell.frame.inputError" state="input_error" :density="density" :data-widget="widget">
    {{ t('shell.desk.unknownWidget', { widget }) }}
  </WidgetFrame>
</template>
