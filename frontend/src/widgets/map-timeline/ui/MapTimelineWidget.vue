<script setup lang="ts">
/**
 * Виджет «Таймлайн» (FR-4, FR-155; AD-22, AD-37) — контейнер. Диапазон истории
 * и метки читает через entities/live-map; управляет моментом useMomentStore —
 * по нему перечитывают себя все виджеты стола. Прогон сценария (AD-38) —
 * срез `run_id` или адрес `?run=…`: таймлайн идёт в пределах прогона.
 * Действия на столе в воспроизведении выключены рамкой и правилом прав.
 */
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useProblemText } from '@/shared/i18n/problem'
import { useTimeline } from '@/entities/live-map'
import { backendModeOf } from '@/shared/api/response'
import type { WidgetProps } from '@/shared/config/widget'
import { useMomentStore } from '@/shared/model/moment'
import { WidgetFrame } from '@/shared/ui'
import { usePlayback, type PlaybackRange } from '../model/playback'
import TimelineBar from './TimelineBar.vue'

const props = defineProps<WidgetProps>()
const route = useRoute()
const moment = useMomentStore()
const problemText = useProblemText()

const runId = computed(() => {
  const q = route?.query.run
  const v = typeof q === 'string' && q ? q : props.slice.run_id
  return typeof v === 'string' && v ? v : undefined
})

const query = useTimeline(computed(() => (runId.value ? { run_id: runId.value } : {})))
const data = computed(() => query.data.value?.data ?? null)
const range = computed<PlaybackRange | null>(() => (data.value ? { from: Date.parse(data.value.from), to: Date.parse(data.value.to) } : null))
const playback = usePlayback(() => range.value)
</script>

<template>
  <!-- Ошибка входа не прячет управление: «Сейчас» и переход к моменту работают и без меток. -->
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="backendModeOf(query.data.value)"
    :state="query.error.value && !data ? 'input_error' : 'normal'"
    :data-widget="widgetId"
  >
    <p v-if="query.error.value && !data" class="problem" role="alert">{{ problemText(query.error.value) }}</p>
    <TimelineBar
      :range="range"
      :marks="data?.marks"
      :as-of="moment.asOf"
      :axis="moment.axis"
      :playing="playback.playing.value"
      :speed="playback.speed.value"
      :density="density"
      @play="playback.play()"
      @pause="playback.pause()"
      @live="playback.goLive()"
      @jump="(at) => playback.jump(at)"
      @speed="(s) => (playback.speed.value = s)"
      @axis="(a) => (moment.asOf ? moment.travel(moment.asOf, a) : (moment.axis = a))"
    />
  </WidgetFrame>
</template>

<style scoped>
.problem {
  margin: 0 0 8px;
  color: #d64545;
  font-size: 13px;
}
</style>
