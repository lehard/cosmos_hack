<script setup lang="ts">
/**
 * Общая рамка виджета (AD-21, NFR-UI-4, FR-150). Даёт каждому виджету одинаково:
 * - заголовок и метку режима данных `fixtures | live` (заголовок Ant-Backend);
 * - момент: «Сейчас» или «Как было / Что мы знали — на момент …» (axis, as_of);
 * - четыре состояния данных: норма / признак дефекта / оценка невозможна / ошибка
 *   входа — с цветом тона из словаря статусов (контракт, AD-30); плюс загрузка
 *   и «нет записей»;
 * - слот действий, который выключен в воспроизведении (FR-4).
 *
 * Виджет сам решает, в каком состоянии его данные, — рамка только показывает.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NAlert, NCard, NEmpty, NSpin, NTag } from 'naive-ui'
import type { BackendMode } from '@/shared/api/generated/model'
import { statusAxes, statusPalette, type StatusTone } from '@/shared/api/generated/statuses'
import { naiveSizeOf, type Density, type WidgetDataState } from '@/shared/config/widget'
import { useProblemText } from '@/shared/i18n/problem'
import { useMomentStore } from '@/shared/model/moment'

const props = withDefaults(
  defineProps<{
    /** Ключ текста заголовка. */
    titleKey: string
    density?: Density
    /** Режим, отдавший данные (backendModeOf(response)); null — не показывать. */
    mode?: BackendMode | null
    /** Состояние данных. */
    state?: WidgetDataState
    loading?: boolean
    /** Ошибка запроса — рамка переходит в «ошибку входа» с текстом по коду. */
    error?: unknown
    /** Данных нет. */
    empty?: boolean
    /** Текст пустого состояния. */
    emptyKey?: string
  }>(),
  { density: 'comfortable', mode: null, state: 'normal', loading: false, error: undefined, empty: false, emptyKey: 'empty.noRecords' },
)

const { t, d } = useI18n()
const problemText = useProblemText()
const moment = useMomentStore()

/** Состояние → текст и тон словаря статусов. */
const STATES: Record<WidgetDataState, { key: string; tone: StatusTone }> = {
  normal: { key: 'shell.frame.normal', tone: 'neutral' },
  defect_indication: { key: 'inspection.outcome.defectFound', tone: statusAxes.quality.values.signal.tone },
  unable_to_assess: { key: 'inspection.outcome.unableToAssess', tone: statusAxes.quality.values.unable_to_assess.tone },
  input_error: { key: 'shell.frame.inputError', tone: 'danger' },
}

const effectiveState = computed<WidgetDataState>(() => (props.error ? 'input_error' : props.state))
const stateInfo = computed(() => STATES[effectiveState.value])
const accent = computed(() => (effectiveState.value === 'normal' ? 'transparent' : statusPalette[stateInfo.value.tone]))

const momentLabel = computed(() => {
  if (!moment.asOf) return t('common.modes.now')
  const axis = t(moment.axis === 'recorded' ? 'common.modes.asOfRecorded' : 'common.modes.asOfOccurred')
  return `${axis} · ${t('common.modes.atMoment', { time: d(new Date(moment.asOf), 'dateTime') })}`
})

const modeLabel = computed(() =>
  props.mode === 'fixtures' ? t('common.modes.backendFixtures') : props.mode === 'live' ? t('common.modes.backendLive') : null,
)
</script>

<template>
  <NCard
    class="widget-frame"
    :class="`density-${density}`"
    :size="naiveSizeOf(density)"
    :style="{ '--accent': accent }"
    :data-state="effectiveState"
    :data-mode="mode ?? undefined"
    :title="t(titleKey)"
    :segmented="{ content: true, footer: 'soft' }"
  >
    <template #header-extra>
      <div class="meta">
        <NTag v-if="effectiveState !== 'normal'" size="small" :bordered="false" :color="{ color: 'transparent', textColor: accent }">
          {{ t(stateInfo.key) }}
        </NTag>
        <span class="moment" :data-replay="moment.isReplay || undefined">{{ momentLabel }}</span>
        <NTag v-if="modeLabel" size="small" :bordered="true" class="mode">{{ modeLabel }}</NTag>
      </div>
    </template>

    <NSpin v-if="loading" size="small" class="center" />
    <NAlert v-else-if="error" type="error" :bordered="false" :show-icon="false">{{ problemText(error) }}</NAlert>
    <NEmpty v-else-if="empty" :description="t(emptyKey)" class="center" />
    <slot v-else />

    <template v-if="$slots.actions" #action>
      <fieldset class="actions" :disabled="moment.isReplay" :data-disabled="moment.isReplay || undefined">
        <slot name="actions" />
      </fieldset>
      <p v-if="moment.isReplay" class="replay-note">{{ t('shell.frame.actionsDisabled') }}</p>
    </template>
  </NCard>
</template>

<style scoped>
.widget-frame {
  height: 100%;
  border-left: 3px solid var(--accent);
}

.meta {
  display: flex;
  gap: 8px;
  align-items: center;
  color: #6b7280;
  font-size: 12px;
}

.moment[data-replay] {
  color: #1f2937;
  font-weight: 700;
}

.center {
  display: flex;
  justify-content: center;
  padding: 16px 0;
}

.actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin: 0;
  padding: 0;
  border: 0;
}

.replay-note {
  margin: 8px 0 0;
  color: #6b7280;
  font-size: 12px;
}

.density-large {
  font-size: 16px;
}

.density-compact {
  font-size: 13px;
}
</style>
