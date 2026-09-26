<script setup lang="ts">
/**
 * Кнопка действия (UI-2): подпись никогда не выходит за границы кнопки —
 * либо многоточие с подсказкой полного текста, либо перенос строки.
 * Все прочие свойства и события NButton (type, size, block, loading, disabled,
 * data-*, @click) передаются как есть.
 *
 * Когда брать: любая кнопка с текстом, длина которого заранее неизвестна
 * (имена, названия операций, переводы). Кнопка-иконка — NButton с aria-label.
 */
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { NButton, NTooltip } from 'naive-ui'
import type { TextOverflow } from './types'

defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{
    /** Подпись кнопки. */
    label: string
    /** Многоточие (по умолчанию) или перенос строки. */
    overflow?: TextOverflow
    /** Подсказка; без неё подсказка — полный текст подписи, если он обрезан. */
    hint?: string
  }>(),
  { overflow: 'ellipsis', hint: undefined },
)

const labelEl = ref<HTMLElement | null>(null)
const truncated = ref(false)

/** Обрезана ли подпись сейчас. */
function measure(): void {
  const el = labelEl.value
  truncated.value = !!el && props.overflow === 'ellipsis' && el.scrollWidth > el.clientWidth + 1
}

let observer: ResizeObserver | null = null
onMounted(() => {
  measure()
  if (typeof ResizeObserver !== 'undefined' && labelEl.value) {
    observer = new ResizeObserver(measure)
    observer.observe(labelEl.value)
  }
})
onBeforeUnmount(() => observer?.disconnect())
watch(() => props.label, measure, { flush: 'post' })

const tip = computed(() => props.hint ?? props.label)
const showTip = computed(() => !!props.hint || truncated.value)
</script>

<template>
  <NTooltip :disabled="!showTip" :delay="250" :style="{ maxWidth: '360px' }">
    <template #trigger>
      <NButton v-bind="$attrs" class="action-button" :class="`is-${overflow}`" :data-truncated="truncated || undefined" @mouseenter="measure">
        <template v-if="$slots.icon" #icon><span class="icon ant-box"><slot name="icon" /></span></template>
        <span ref="labelEl" class="label" :class="overflow === 'wrap' ? 'ant-wrap' : 'ant-ellipsis'">{{ label }}</span>
      </NButton>
    </template>
    <span class="ant-wrap">{{ tip }}</span>
  </NTooltip>
</template>

<style scoped>
.action-button {
  max-width: 100%;
  min-width: 0;
}

.action-button :deep(.n-button__content) {
  min-width: 0;
  max-width: 100%;
  overflow: hidden;
}

.label {
  min-width: 0;
}

.icon {
  display: inline-flex;
}

.is-wrap {
  height: auto;
  min-height: var(--n-height);
  padding-top: 6px;
  padding-bottom: 6px;
  line-height: var(--ant-lh-tight);
}

.is-wrap :deep(.n-button__content) {
  white-space: normal;
  text-align: left;
}
</style>
