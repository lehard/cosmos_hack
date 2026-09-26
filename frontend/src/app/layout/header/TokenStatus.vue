<script setup lang="ts">
/**
 * Индикатор «Токен» в шапке (AD-14, Д-72): состояние ключа подписи на рабочем
 * месте по расширению «Главный — подпись» — готов (PIN введён), загружен
 * (PIN спросит окно подписи), не загружен, расширения нет. В подсказке —
 * владелец ключа и класс хранения (физический ключ / ключ в браузере).
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NIcon, NTooltip } from 'naive-ui'
import { Key } from '@vicons/tabler'
import { useTokenInfo, useTokenStatus } from '@/shared/lib/token-agent'

const { t, te } = useI18n()
const status = useTokenStatus()
const info = useTokenInfo()

const text = computed(() => {
  switch (status.value) {
    case 'inserted':
      return t('common.header.tokenInserted')
    case 'locked':
      return t('common.header.tokenLocked')
    case 'missing':
      return t('common.header.tokenMissing')
    default:
      return t('common.header.tokenAgentMissing')
  }
})

const who = computed(() => {
  const i = info.value
  if (!i?.person_id) return ''
  const key = `common.header.tokenStorage.${i.key_storage ?? ''}`
  return t('common.header.tokenWho', { person: i.person_id, storage: te(key) ? t(key) : (i.key_storage ?? '—') })
})
</script>

<template>
  <NTooltip>
    <template #trigger>
      <span class="token" :data-status="status" data-testid="token-status">
        <NIcon :depth="status === 'inserted' ? 1 : 3" :class="{ ready: status === 'inserted' }"><Key /></NIcon>
        {{ t('common.header.tokenStatus') }}
      </span>
    </template>
    {{ text }}
    <template v-if="who"><br />{{ who }}</template>
  </NTooltip>
</template>

<style scoped>
.token {
  display: inline-flex;
  gap: 4px;
  align-items: center;
  font-size: var(--ant-fs-sm);
  white-space: nowrap;
}
.ready {
  color: var(--ant-accent);
}
</style>
