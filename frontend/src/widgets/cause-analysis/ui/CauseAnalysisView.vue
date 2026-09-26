<script setup lang="ts">
/**
 * «Разбор причин» — основная область стола технолога (PRD §3a): несоответствия,
 * сгруппированные по виду дефекта × операции × оборудованию. Выбор группы
 * открывает её общие факторы (FR-135) и разбор обстоятельств (FR-153).
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { sortGroups, type NcGroup } from '@/entities/incident'
import { codeToKey } from '@/shared/i18n'
import type { Density } from '@/shared/config/widget'

const props = withDefaults(defineProps<{ groups: NcGroup[]; density?: Density }>(), { density: 'compact' })
/** Выбранная группа (`group_key`). */
const selected = defineModel<string | null>('selected', { default: null })

const { t, d } = useI18n()
const rows = computed(() => sortGroups(props.groups))
const toggle = (key: string) => (selected.value = selected.value === key ? null : key)
</script>

<template>
  <div class="causes" :class="`density-${density}`" data-testid="cause-analysis">
    <table class="table">
      <thead>
        <tr>
          <th>{{ t('common.words.defectType') }}</th>
          <th>{{ t('common.words.operation') }}</th>
          <th>{{ t('common.words.equipment') }}</th>
          <th class="num">{{ t('widgets.analysis.causes.count') }}</th>
          <th>{{ t('widgets.analysis.causes.investigation') }}</th>
          <th>{{ t('widgets.analysis.causes.lastFound') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="g in rows"
          :key="g.group_key"
          :class="{ on: selected === g.group_key }"
          :data-group="g.group_key"
          :aria-selected="selected === g.group_key"
          tabindex="0"
          @click="toggle(g.group_key)"
          @keydown.enter.prevent="toggle(g.group_key)"
          @keydown.space.prevent="toggle(g.group_key)"
        >
          <td class="strong">{{ g.defect_type }}</td>
          <td>{{ g.operation }}</td>
          <td>{{ g.equipment }}</td>
          <td class="num">{{ g.nc_count }}</td>
          <td>{{ t(`statuses.ncInvestigation.${codeToKey(g.investigation)}`) }}</td>
          <td class="muted">{{ d(new Date(g.last_found_at), 'dateTime') }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<style scoped>
.causes {
  font-size: 13px;
}

.density-large {
  font-size: 16px;
}

.table {
  width: 100%;
  border-collapse: collapse;
}

.table th {
  padding: 4px 8px;
  color: #6b7280;
  font-weight: 400;
  text-align: left;
  border-bottom: 1px solid #e5e7eb;
}

.table td {
  padding: 5px 8px;
  border-bottom: 1px solid #f3f4f6;
}

tbody tr {
  cursor: pointer;
}

tbody tr:hover,
tbody tr:focus-visible {
  background: #f8fafc;
  outline: none;
}

tbody tr.on {
  background: #eff6ff;
  box-shadow: inset 3px 0 0 #2f6fdb;
}

.num {
  font-family: 'PT Mono', monospace;
  text-align: right;
}

.strong {
  font-weight: 700;
}

.muted {
  color: #6b7280;
}
</style>
