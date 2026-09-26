<script setup lang="ts">
/**
 * Поиск изделия по номеру или коду маркировки (PRD §3a, UI-5): подпись
 * простыми словами, а что такое код маркировки (DataMatrix) — в подсказке у
 * значка. Сканер клавиатурного типа вводит строку и Enter сам. Найдено —
 * окно изделия справа (Д-70, поверх текущего стола); нет — сообщение.
 */
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NIcon, NInput, NTooltip, useMessage } from 'naive-ui'
import { InfoCircle, Search } from '@vicons/tabler'
import { useQueryClient } from '@tanstack/vue-query'
import { itemLookupQueryOptions } from '@/entities/item'
import { useDrillDown } from '@/features/drill-down'
import { statusOf } from '@/shared/api'
import { useProblemText } from '@/shared/i18n/problem'
import { useMomentStore } from '@/shared/model/moment'

const { t } = useI18n()
const drill = useDrillDown()
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
    drill.open({ entity: 'item', id: found.data.item_id })
    query.value = ''
  } catch (err) {
    message.warning(statusOf(err) === 404 ? t('empty.notFound', { query: q }) : problemText(err))
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <NInput
    v-model:value="query"
    clearable
    :placeholder="t('shell.header.searchLabel')"
    :aria-label="t('shell.header.searchLabel')"
    :loading="busy"
    data-testid="item-search"
    @keyup.enter="search"
  >
    <template #prefix><NIcon><Search /></NIcon></template>
    <template #suffix>
      <NTooltip :style="{ maxWidth: '360px' }">
        <template #trigger>
          <NIcon class="hint" data-testid="item-search-hint" :aria-label="t('shell.header.searchHint')" tabindex="0"><InfoCircle /></NIcon>
        </template>
        <span class="ant-wrap">{{ t('shell.header.searchHint') }}</span>
      </NTooltip>
    </template>
  </NInput>
</template>

<style scoped>
.hint {
  color: var(--ant-text-3);
  cursor: help;
}
</style>
