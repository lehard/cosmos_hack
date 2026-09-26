<script setup lang="ts">
/**
 * Выбор периода показателей: смена / сутки / неделя / месяц (FR-3). Период общий
 * для всех виджетов аналитики — в фокусе аналитики (Pinia).
 */
import { useI18n } from 'vue-i18n'
import { NRadioButton, NRadioGroup } from 'naive-ui'
import { PERIODS, useMetricFocusStore } from '../model/focus'

defineProps<{ size?: 'small' | 'medium' | 'large' }>()
const { t } = useI18n()
const focus = useMetricFocusStore()
</script>

<template>
  <NRadioGroup v-model:value="focus.period" :size="size ?? 'small'" :aria-label="t('analytics.period')" data-testid="period">
    <NRadioButton v-for="p in PERIODS" :key="p" :value="p" :data-period="p">{{ t(`liveMap.period.${p}`) }}</NRadioButton>
  </NRadioGroup>
</template>
