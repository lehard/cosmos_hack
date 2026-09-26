<script setup lang="ts">
/**
 * Общая рамка виджета (AD-21, NFR-UI-4, FR-150). Даёт каждому виджету одинаково:
 * - заголовок; режим данных `fixtures | live` — только атрибутом `data-mode`
 *   (Д-70, UI-4, UI-6: метки режима в заголовке панели не показываются, режим
 *   заготовок пояснён в меню пользователя);
 * - момент — только при просмотре прошлого: «Как было / Что мы знали — на
 *   момент …» (axis, as_of); «Сейчас» не пишется — это обычное состояние;
 * - четыре состояния данных: норма / признак дефекта / оценка невозможна / ошибка
 *   входа — с цветом тона из словаря статусов (контракт, AD-30); плюс загрузка
 *   и «нет записей»;
 * - слот действий, который выключен в воспроизведении (FR-4).
 *
 * Виджет сам решает, в каком состоянии его данные, — рамка только показывает.
 *
 * Плотность стола (AD-21) рамка раздаёт содержимому: класс `ant-density-‹…›`
 * (CSS-переменные размеров) и вложенная тема Naive UI (высоты, шрифт).
 * Заголовок — одна строка с многоточием и подсказкой (UI-2). Если заголовок уже
 * показан выше (страница, вкладка, окно записи — WIDGET_FRAME_CONTEXT), рамка
 * его не повторяет (UI-6).
 */
import { computed, inject } from 'vue'
import { useI18n } from 'vue-i18n'
import { NAlert, NCard, NConfigProvider, NSpin } from 'naive-ui'
import type { BackendMode } from '@/shared/api/generated/model'
import { statusAxes, statusPalette, type StatusTone } from '@/shared/api/generated/statuses'
import { naiveSizeOf, type Density, type WidgetDataState } from '@/shared/config/widget'
import { useProblemText } from '@/shared/i18n/problem'
import { useMomentStore } from '@/shared/model/moment'
import EmptyState from './EmptyState.vue'
import { WIDGET_FRAME_CONTEXT } from './frame'
import { densityClass, densityOverrides } from './theme'

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

const context = inject(WIDGET_FRAME_CONTEXT, {})

/** Момент — только при просмотре прошлого (Д-70). */
const momentLabel = computed(() => {
  if (!moment.asOf) return null
  const axis = t(moment.axis === 'recorded' ? 'common.modes.asOfRecorded' : 'common.modes.asOfOccurred')
  return `${axis} · ${t('common.modes.atMoment', { time: d(new Date(moment.asOf), 'dateTime') })}`
})

const title = computed(() => t(props.titleKey))
const overrides = computed(() => densityOverrides(props.density))

/** Шапка рамки нужна, если есть заголовок или что сказать о состоянии и моменте. */
const showHeader = computed(() => !context.hideTitle || effectiveState.value !== 'normal' || !!momentLabel.value)
</script>

<template>
  <NCard
    class="widget-frame"
    :class="[densityClass(density), { 'widget-frame--plain': context.plain, 'widget-frame--headless': !showHeader }]"
    :size="naiveSizeOf(density)"
    :style="{ '--accent': accent }"
    :data-state="effectiveState"
    :data-mode="mode ?? undefined"
    :segmented="{ content: true, footer: 'soft' }"
  >
    <template v-if="showHeader" #header>
      <h3 v-if="!context.hideTitle" class="title ant-ellipsis" :title="title">{{ title }}</h3>
    </template>
    <template v-if="showHeader" #header-extra>
      <div class="meta ant-box">
        <span v-if="effectiveState !== 'normal'" class="state ant-ellipsis">{{ t(stateInfo.key) }}</span>
        <span v-if="momentLabel" class="moment ant-ellipsis" data-replay="true" :title="momentLabel">{{ momentLabel }}</span>
      </div>
    </template>

    <NConfigProvider abstract :theme-overrides="overrides">
      <div class="frame-body ant-box">
        <div v-if="loading" class="center ant-box"><NSpin size="small" /></div>
        <NAlert v-else-if="error" type="error" :bordered="false" :show-icon="false">
          <span class="ant-wrap">{{ problemText(error) }}</span>
        </NAlert>
        <EmptyState v-else-if="empty" :title="t(emptyKey)" compact />
        <slot v-else />
      </div>
    </NConfigProvider>

    <template v-if="$slots.actions" #action>
      <NConfigProvider abstract :theme-overrides="overrides">
        <fieldset class="actions ant-box" :disabled="moment.isReplay" :data-disabled="moment.isReplay || undefined">
          <slot name="actions" />
        </fieldset>
      </NConfigProvider>
      <p v-if="moment.isReplay" class="replay-note ant-wrap">{{ t('shell.frame.actionsDisabled') }}</p>
    </template>
  </NCard>
</template>

<style scoped>
.widget-frame {
  height: 100%;
  min-width: 0;
  box-shadow: var(--ant-shadow-sm), inset 3px 0 0 var(--accent);
  font-size: var(--ant-fs-body);
}

.widget-frame :deep(.n-card-header) {
  gap: var(--ant-space-3);
  min-width: 0;
}

.widget-frame :deep(.n-card-header__main) {
  min-width: 0;
}

.widget-frame :deep(.n-card-header__extra) {
  flex: 0 1 auto;
  min-width: 0;
}

.widget-frame :deep(.n-card__content) {
  min-width: 0;
}

.title {
  font-size: var(--ant-fs-title);
  line-height: var(--ant-lh-tight);
}

.meta {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 10px;
  align-items: center;
  justify-content: flex-end;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-xs);
}

.meta > span {
  max-width: 28ch;
}

.state {
  color: var(--accent);
  font-weight: var(--ant-fw-bold);
}

.moment[data-replay] {
  color: var(--ant-text);
  font-weight: var(--ant-fw-bold);
}

/* Внутри окна записи: без обводки, тени и внутренних полей — поля даёт окно. */
.widget-frame--plain {
  border: 0;
  background: transparent;
  box-shadow: none;
}

.widget-frame--plain :deep(.n-card-header),
.widget-frame--plain :deep(.n-card__content),
.widget-frame--plain :deep(.n-card__action) {
  padding-right: 0;
  padding-left: 0;
}

.widget-frame--plain :deep(.n-card__content) {
  padding-top: 0;
}

.frame-body {
  min-width: 0;
}

.center {
  display: flex;
  justify-content: center;
  padding: var(--ant-space-4) 0;
}

.actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-2);
  min-width: 0;
  margin: 0;
  padding: 0;
  border: 0;
}

.replay-note {
  margin: var(--ant-space-2) 0 0;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-xs);
}
</style>
