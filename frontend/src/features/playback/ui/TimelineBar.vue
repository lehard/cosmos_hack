<script setup lang="ts">
/**
 * Полоса времени (FR-4, FR-155): «Сейчас» ↔ воспроизведение, пауза,
 * скорость ×1…×1000, переход к моменту (ползунок или дата), метки значимых
 * событий — клик ведёт к моменту метки. Ось: «как было» (occurred) или
 * «что мы знали» (recorded, AD-37). Компонент только показывает и сообщает
 * о намерениях; моментом управляет контейнер.
 * `compact` — одна строка внизу живой карты (UI-19): кнопки, ползунок с
 * метками, момент и ось; без выбора даты и пояснения — высота постоянная.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NDatePicker, NRadioButton, NRadioGroup, NSlider } from 'naive-ui'
import type { TimelineMark, TimelineMarkKind } from '@/entities/live-map'
import type { Axis } from '@/shared/api/generated/model'
import { statusPalette } from '@/shared/api/generated/statuses'
import { naiveSizeOf, type Density } from '@/shared/config/widget'
import { SPEEDS, type PlaybackRange, type Speed } from '../model/playback'
import { ActionButton } from '@/shared/ui'

const props = withDefaults(
  defineProps<{
    range: PlaybackRange | null
    marks?: readonly TimelineMark[]
    /** Момент воспроизведения; null — «сейчас». */
    asOf: string | null
    axis: Axis
    playing: boolean
    speed: Speed
    density?: Density
    /** Одна строка постоянной высоты (внизу живой карты). */
    compact?: boolean
  }>(),
  { marks: () => [], density: 'comfortable', compact: false },
)
const emit = defineEmits<{
  play: []
  pause: []
  live: []
  jump: [at: number]
  speed: [speed: Speed]
  axis: [axis: Axis]
}>()
const { t, d } = useI18n()

const size = computed(() => (props.compact ? 'small' : naiveSizeOf(props.density)))
const replay = computed(() => props.asOf !== null)
const position = computed(() => (props.asOf ? Date.parse(props.asOf) : (props.range?.to ?? 0)))

/** Метка → цвет тона: остановка и эскалация — красные, всплеск — жёлтый, пересмотр — синий. */
const MARK_TONE: Record<TimelineMarkKind, string> = {
  escalation: statusPalette.danger,
  process_stop: statusPalette.critical,
  spike: statusPalette.attention,
  revision: statusPalette.info,
}
const MARK_TEXT: Record<TimelineMarkKind, string> = {
  escalation: 'liveMap.marks.escalation',
  process_stop: 'liveMap.marks.processStop',
  spike: 'liveMap.marks.spike',
  revision: 'liveMap.marks.revision',
}

const placed = computed(() => {
  const r = props.range
  if (!r || r.to <= r.from) return []
  return props.marks
    .map((m) => ({ m, at: Date.parse(m.at) }))
    .filter(({ at }) => at >= r.from && at <= r.to)
    .map(({ m, at }) => ({
      m,
      at,
      left: `${((at - r.from) / (r.to - r.from)) * 100}%`,
      color: MARK_TONE[m.kind],
      title: `${t(MARK_TEXT[m.kind])}${m.title ? `: ${m.title}` : ''} · ${d(new Date(at), 'dateTime')}`,
    }))
})

const formatTooltip = (v: number) => d(new Date(v), 'dateTime')
</script>

<template>
  <div class="timeline" :class="{ 'timeline--compact': compact }" :data-replay="replay || undefined" :data-playing="playing || undefined">
    <div class="controls">
      <ActionButton :size="size" :type="replay ? 'default' : 'primary'" :disabled="!replay" data-action="live" @click="emit('live')" :label="t('liveMap.playback.now')" />
      <ActionButton v-if="!playing" :size="size" :disabled="!range" data-action="play" @click="emit('play')" :label="`▶ ${replay ? t('liveMap.playback.play') : t('liveMap.playback.replay')}`" />
      <ActionButton v-else :size="size" data-action="pause" @click="emit('pause')" :label="`⏸ ${t('liveMap.playback.pause')}`" />
      <NRadioGroup :value="speed" :size="size" name="speed" @update:value="(v: Speed) => emit('speed', v)">
        <NRadioButton v-for="s in SPEEDS" :key="s" :value="s" :data-speed="s">{{ t('liveMap.playback.speed', { speed: s }) }}</NRadioButton>
      </NRadioGroup>
      <NDatePicker
        v-if="!compact"
        type="datetime"
        :size="size"
        :value="asOf ? Date.parse(asOf) : null"
        :placeholder="t('liveMap.playback.goTo')"
        :is-date-disabled="(ts: number) => !!range && (ts < range.from - 86_400_000 || ts > range.to)"
        data-testid="go-to"
        @update:value="(v: number | null) => (v === null ? emit('live') : emit('jump', v))"
      />
      <NRadioGroup v-if="!compact" :value="axis" :size="size" name="axis" @update:value="(v: Axis) => emit('axis', v)">
        <NRadioButton value="occurred" data-axis="occurred">{{ t('common.modes.asOfOccurred') }}</NRadioButton>
        <NRadioButton value="recorded" data-axis="recorded">{{ t('common.modes.asOfRecorded') }}</NRadioButton>
      </NRadioGroup>
    </div>

    <div v-if="range" class="track">
      <div class="marks">
        <button
          v-for="p in placed"
          :key="p.m.mark_id"
          type="button"
          class="mark"
          :style="{ left: p.left, background: p.color }"
          :title="p.title"
          :aria-label="p.title"
          :data-mark="p.m.mark_id"
          :data-kind="p.m.kind"
          @click="emit('jump', p.at)"
        />
      </div>
      <NSlider
        :min="range.from"
        :max="range.to"
        :step="1000"
        :value="position"
        :format-tooltip="formatTooltip"
        data-testid="scrub"
        @update:value="(v: number) => emit('jump', v)"
      />
      <div v-if="!compact" class="ends">
        <span>{{ d(new Date(range.from), 'dateTime') }}</span>
        <span v-if="replay" class="at" data-testid="at">{{ d(new Date(position), 'dateTime') }}</span>
        <span>{{ t('liveMap.playback.now') }}</span>
      </div>
    </div>

    <template v-if="compact">
      <span class="at at--compact ant-ellipsis" data-testid="at" :title="t('liveMap.playback.replayNote')">
        {{ replay ? d(new Date(position), 'dateTime') : t('liveMap.playback.now') }}
      </span>
      <NRadioGroup :value="axis" :size="size" name="axis" @update:value="(v: Axis) => emit('axis', v)">
        <NRadioButton value="occurred" data-axis="occurred">{{ t('common.modes.asOfOccurred') }}</NRadioButton>
        <NRadioButton value="recorded" data-axis="recorded">{{ t('common.modes.asOfRecorded') }}</NRadioButton>
      </NRadioGroup>
    </template>
    <p v-else-if="replay" class="note" data-testid="replay-note">{{ t('liveMap.playback.replayNote') }}</p>
  </div>
</template>

<style scoped>
.timeline {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.controls {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.track {
  position: relative;
  padding-top: 14px;
}

.marks {
  position: absolute;
  top: 0;
  right: 0;
  left: 0;
  height: 12px;
}

.mark {
  position: absolute;
  width: 8px;
  height: 12px;
  margin-left: -4px;
  padding: 0;
  border: 0;
  border-radius: 2px;
  cursor: pointer;
}

.ends {
  display: flex;
  justify-content: space-between;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.at {
  color: var(--ant-text);
  font-weight: var(--ant-fw-bold);
}

.note {
  margin: 0;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

/* Одна строка постоянной высоты: кнопки — ползунок с метками — момент — ось. */
.timeline--compact {
  flex-direction: row;
  flex-wrap: wrap;
  gap: var(--ant-space-3);
  align-items: center;
}

.timeline--compact .controls {
  flex: none;
  flex-wrap: nowrap;
}

.timeline--compact .track {
  flex: 1 1 240px;
  min-width: 0;
  padding-top: 12px;
}

.at--compact {
  flex: none;
  max-width: 18ch;
  font-size: var(--ant-fs-meta);
}
</style>
