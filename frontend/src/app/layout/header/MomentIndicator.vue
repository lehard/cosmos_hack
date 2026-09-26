<script setup lang="ts">
/**
 * Момент просмотра (AD-21): «Сейчас» или «Как было / Что мы знали — на момент …».
 * В воспроизведении — кнопка возврата к «сейчас».
 */
import { useI18n } from 'vue-i18n'
import { NButton, NTag } from 'naive-ui'
import { useMomentStore } from '@/shared/model/moment'

const { t, d } = useI18n()
const moment = useMomentStore()
</script>

<template>
  <span class="moment" data-testid="moment">
    <NTag v-if="!moment.asOf" size="small" :bordered="false">{{ t('common.modes.now') }}</NTag>
    <template v-else>
      <NTag size="small" type="warning">
        {{ t(moment.axis === 'recorded' ? 'common.modes.asOfRecorded' : 'common.modes.asOfOccurred') }} ·
        {{ t('common.modes.atMoment', { time: d(new Date(moment.asOf), 'dateTime') }) }}
      </NTag>
      <NButton size="tiny" quaternary @click="moment.goLive()">{{ t('shell.header.goLive') }}</NButton>
    </template>
  </span>
</template>

<style scoped>
.moment {
  display: inline-flex;
  gap: 4px;
  align-items: center;
  white-space: nowrap;
}
</style>
