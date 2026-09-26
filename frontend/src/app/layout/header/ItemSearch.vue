<script setup lang="ts">
/**
 * Поиск по номеру детали или скану DataMatrix (PRD §3a): сканер клавиатурного
 * типа вводит строку и Enter. Найдено — паспорт изделия; нет — сообщение.
 */
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { NIcon, NInput, useMessage } from 'naive-ui'
import { Search } from '@vicons/tabler'
import { useQueryClient } from '@tanstack/vue-query'
import { itemLookupQueryOptions } from '@/entities/item'
import { statusOf } from '@/shared/api'
import { useProblemText } from '@/shared/i18n/problem'
import { useMomentStore } from '@/shared/model/moment'

const { t } = useI18n()
const router = useRouter()
const message = useMessage()
const queryClient = useQueryClient()
const moment = useMomentStore()
const problemText = useProblemText()

const query = ref('')
const busy = ref(false)

async function search() {
  const q = query.value.trim()
  if (!q || busy.value) return
  busy.value = true
  try {
    const found = await queryClient.fetchQuery(itemLookupQueryOptions(q, moment.params))
    await router.push({ name: 'item', params: { id: found.data.item_id } })
    query.value = ''
  } catch (err) {
    message.warning(statusOf(err) === 404 ? t('empty.notFound', { query: q }) : problemText(err))
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <NInput v-model:value="query" clearable :placeholder="t('common.header.searchPlaceholder')" :loading="busy" @keyup.enter="search">
    <template #prefix><NIcon><Search /></NIcon></template>
  </NInput>
</template>
