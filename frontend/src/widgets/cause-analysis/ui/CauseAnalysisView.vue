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
import { DataTable } from '@/shared/ui'

const props = withDefaults(defineProps<{ groups: NcGroup[]; density?: Density }>(), { density: 'compact' })
/** Выбранная группа (`group_key`). */
const selected = defineModel<string | null>('selected', { default: null })

const { t, d } = useI18n()
const rows = computed(() => sortGroups(props.groups))
const toggle = (key: string) => (selected.value = selected.value === key ? null : key)
</script>

<template>
  <div class="causes" :class="`density-${density}`" data-testid="cause-analysis">
    <DataTable class="table">
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
    </DataTable>
  </div>
</template>

<style scoped>
.causes {
  font-size: var(--ant-fs-body);
}

.density-large {
  font-size: var(--ant-fs-lg);
}

tbody tr {
  cursor: pointer;
}

tbody tr:hover,
tbody tr:focus-visible {
  background: var(--ant-surface-subtle);
  outline: none;
}

tbody tr.on {
  background: var(--ant-accent-soft);
  box-shadow: inset 3px 0 0 var(--ant-accent);
}

.num {
  font-family: var(--ant-font-mono);
  text-align: right;
}

.strong {
  font-weight: var(--ant-fw-bold);
}

.muted {
  color: var(--ant-text-3);
}
</style>
