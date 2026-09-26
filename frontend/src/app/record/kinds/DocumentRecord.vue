<script setup lang="ts">
/**
 * Окно документа (Д-70): `?open=document:‹id›` из реестра документов, паспорта
 * изделия, уведомлений. Содержимое — окно документа виджета documents-registry:
 * маршрут подписей, содержимое, версии и кнопки по правам.
 */
import { DocumentDrawer } from '@/widgets/documents-registry'

defineProps<{ id: string; show: boolean }>()
const emit = defineEmits<{ close: [] }>()
</script>

<template>
  <DocumentDrawer :id="id" :show="show" @close="emit('close')" />
</template>
