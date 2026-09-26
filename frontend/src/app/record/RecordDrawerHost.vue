<script setup lang="ts">
/**
 * Окно записи оболочки (Д-70): читает `?open=‹тип›:‹id›`, находит содержимое в
 * реестре и показывает его в правом окне поверх текущей страницы. Закрытие
 * (крестик, Esc, щелчок мимо) убирает параметр из адреса; «назад» в браузере
 * тоже закрывает окно. Неизвестный тип в адресе — окно не открывается.
 */
import { computed, defineAsyncComponent, ref, watch, type Component } from 'vue'
import { sameRecord, useRecordLink } from '@/shared/model/record'
import type { DrillRef } from '@/shared/model/drill'
import { RECORD_KINDS, recordKinds } from './registry'

// Типы окна объявляет оболочка (ShellLayout) через RECORD_DRAWER.
const link = useRecordLink()

const cache = new Map<string, Component>()
function componentOf(kind: string): Component {
  let c = cache.get(kind)
  if (!c) {
    c = defineAsyncComponent(recordKinds[kind]!.load)
    cache.set(kind, c)
  }
  return c
}

const wanted = computed<DrillRef | null>(() => {
  const ref = link.current.value
  return ref && RECORD_KINDS.has(ref.entity) ? ref : null
})
// Последняя открытая запись остаётся в окне, пока оно уезжает.
const shown = ref<DrillRef | null>(wanted.value)
watch(wanted, (ref) => {
  if (ref && !sameRecord(ref, shown.value)) shown.value = ref
})
</script>

<template>
  <component
    :is="componentOf(shown.entity)"
    v-if="shown"
    :id="shown.id"
    :key="`${shown.entity}:${shown.id}`"
    :show="!!wanted"
    @close="link.close()"
  />
</template>
