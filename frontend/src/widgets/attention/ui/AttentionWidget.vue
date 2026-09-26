<script setup lang="ts">
/**
 * Виджет «Требует вашего внимания» (FR-8, PRD §3a) — поверх живой карты:
 * просроченные решения с ценой задержки («просрочено на 37 мин — стоят 18
 * изделий, 2 операции»), меры без подтверждённой результативности, временные
 * меры с недостигнутым условием выхода. Клик — к объекту (FR-7).
 * Сроки и эскалации порождает модуль notifications (AD-40); здесь только показ.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAttention, type AttentionEntry } from '@/entities/notification'
import { useDrillDown } from '@/features/drill-down'
import { backendModeOf } from '@/shared/api/response'
import { statusPalette } from '@/shared/api/generated/statuses'
import type { WidgetProps } from '@/shared/config/widget'
import { formatMinutes } from '@/shared/lib/duration'
import { WidgetFrame } from '@/shared/ui'

const props = defineProps<WidgetProps>()
const { t } = useI18n()
const drill = useDrillDown()

const runId = computed(() => (typeof props.slice.run_id === 'string' && props.slice.run_id ? props.slice.run_id : undefined))
const query = useAttention(computed(() => (runId.value ? { run_id: runId.value } : {})))
const entries = computed(() => query.data.value?.data ?? null)

/** Текст строки — только ключами liveMap.attention (NFR-UI-3). */
function text(e: AttentionEntry): string {
  if (e.kind === 'overdue_decision') {
    return t('liveMap.attention.overdueDecision', {
      target: e.target,
      overdue: formatMinutes(t, e.overdue_minutes),
      items: t('plural.items', { n: e.items }, e.items),
      operations: t('plural.operations', { n: e.operations }, e.operations),
    })
  }
  return t(e.kind === 'unverified_measures' ? 'liveMap.attention.unverifiedMeasures' : 'liveMap.attention.temporaryMeasures', { n: e.n })
}

const rows = computed(() =>
  (entries.value ?? []).map((e) => ({
    e,
    text: text(e),
    color: e.kind === 'overdue_decision' ? statusPalette.danger : statusPalette.attention,
    link: e.ref && drill.canOpen(e.ref) ? e.ref : null,
  })),
)
/** Просроченное решение — признак проблемы на производстве, не «брак». */
const state = computed(() => (entries.value?.some((e) => e.kind === 'overdue_decision') ? 'defect_indication' : 'normal'))
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="backendModeOf(query.data.value)"
    :state="state"
    :loading="query.isPending.value && !entries"
    :error="entries ? undefined : query.error.value"
    :empty="!!entries && !entries.length"
    :data-widget="widgetId"
  >
    <ul class="attention">
      <li v-for="r in rows" :key="r.e.entry_id" :data-kind="r.e.kind" :style="{ borderColor: r.color }">
        <button v-if="r.link" type="button" class="link" @click="drill.open(r.link)">{{ r.text }}</button>
        <span v-else>{{ r.text }}</span>
      </li>
    </ul>
  </WidgetFrame>
</template>

<style scoped>
.attention {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.attention li {
  padding: 6px 10px;
  border-left: 3px solid;
  border-radius: 4px;
  background: #f9fafb;
  font-size: 14px;
}

.link {
  padding: 0;
  border: 0;
  background: none;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.link:hover {
  text-decoration: underline;
}
</style>
